package btcharness

import (
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/jfixby/pin"
	"math"
	"math/big"
	"runtime"
	"time"

	"github.com/btcsuite/btcd/blockchain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

// GenerateBlockArgs bundles GenerateBlock() arguments to minimize diff
// in case a new argument for the function is added
type GenerateBlockArgs struct {
	Txns          []*btcutil.Tx
	BlockVersion  int32
	BlockTime     time.Time
	MineTo        []wire.TxOut
	MiningAddress btcutil.Address
	Network       *chaincfg.Params
}

// GenerateAndSubmitBlock creates a block whose contents include the passed
// transactions and submits it to the running simnet node. For generating
// blocks with only a coinbase tx, callers can simply pass nil instead of
// transactions to be mined. Additionally, a custom block version can be set by
// the caller. An uninitialized time.Time should be used for the
// blockTime parameter if one doesn't wish to set a custom time.
func GenerateAndSubmitBlock(client *rpcclient.Client, args *GenerateBlockArgs) (*btcutil.Block, error) {
	pin.AssertTrue("args.MineTo is empty", len(args.MineTo) == 0)
	return GenerateAndSubmitBlockWithCustomCoinbaseOutputs(client, args)
}

// GenerateAndSubmitBlockWithCustomCoinbaseOutputs creates a block whose
// contents include the passed coinbase outputs and transactions and submits
// it to the running simnet node. For generating blocks with only a coinbase tx,
// callers can simply pass nil instead of transactions to be mined.
// Additionally, a custom block version can be set by the caller. A blockVersion
// of -1 indicates that the current default block version should be used. An
// uninitialized time.Time should be used for the blockTime parameter if one
// doesn't wish to set a custom time. The mineTo list of outputs will be added
// to the coinbase; this is not checked for correctness until the block is
// submitted; thus, it is the caller's responsibility to ensure that the outputs
// are correct. If the list is empty, the coinbase reward goes to the wallet
// managed by the Harness.
func GenerateAndSubmitBlockWithCustomCoinbaseOutputs(client *rpcclient.Client, args *GenerateBlockArgs) (*btcutil.Block, error) {
	txns := args.Txns
	blockVersion := args.BlockVersion
	pin.AssertTrue(fmt.Sprintf("Incorrect blockVersion(%v)", blockVersion), blockVersion > 0)
	blockTime := args.BlockTime
	mineTo := args.MineTo
	miningAddress := args.MiningAddress
	network := args.Network

	pin.AssertTrue("blockVersion != -1", blockVersion != -1)

	prevBlockHash, prevBlockHeight, err := client.GetBestBlock()
	if err != nil {
		return nil, err
	}
	mBlock, err := client.GetBlock(prevBlockHash)
	if err != nil {
		return nil, err
	}
	prevBlock := btcutil.NewBlock(mBlock)
	prevBlock.SetHeight(prevBlockHeight)

	// Create a new block including the specified transactions
	newBlock, err := CreateBlock(prevBlock, txns, blockVersion,
		blockTime, miningAddress, mineTo, network)
	if err != nil {
		return nil, err
	}

	// Submit the block to the simnet node.
	if err := client.SubmitBlock(newBlock, nil); err != nil {
		return nil, err
	}

	return newBlock, nil
}


// CreateBlock creates a new block building from the previous block with a
// specified blockversion and timestamp. If the timestamp passed is zero (not
// initialized), then the timestamp of the previous block will be used plus 1
// second is used. Passing nil for the previous block results in a block that
// builds off of the genesis block for the specified chain.
func CreateBlock(prevBlock *btcutil.Block, inclusionTxs []*btcutil.Tx,
	blockVersion int32, blockTime time.Time, miningAddr btcutil.Address,
	mineTo []wire.TxOut, net *chaincfg.Params) (*btcutil.Block, error) {

	var (
		prevHash      *chainhash.Hash
		blockHeight   int32
		prevBlockTime time.Time
	)

	// If the previous block isn't specified, then we'll construct a block
	// that builds off of the genesis block for the chain.
	if prevBlock == nil {
		prevHash = net.GenesisHash
		blockHeight = 1
		prevBlockTime = net.GenesisBlock.Header.Timestamp.Add(time.Minute)
	} else {
		prevHash = prevBlock.Hash()
		blockHeight = prevBlock.Height() + 1
		prevBlockTime = prevBlock.MsgBlock().Header.Timestamp
	}

	// If a target block time was specified, then use that as the header's
	// timestamp. Otherwise, add one second to the previous block unless
	// it's the genesis block in which case use the current time.
	var ts time.Time
	switch {
	case !blockTime.IsZero():
		ts = blockTime
	default:
		ts = prevBlockTime.Add(time.Second)
	}

	extraNonce := uint64(0)
	coinbaseScript, err := standardCoinbaseScript(blockHeight, extraNonce)
	if err != nil {
		return nil, err
	}
	coinbaseTx, err := createCoinbaseTx(coinbaseScript, blockHeight,
		miningAddr, mineTo, net)
	if err != nil {
		return nil, err
	}

	// Create a new block ready to be solved.
	blockTxns := []*btcutil.Tx{coinbaseTx}
	if inclusionTxs != nil {
		blockTxns = append(blockTxns, inclusionTxs...)
	}
	merkles := blockchain.BuildMerkleTreeStore(blockTxns, false)
	var block wire.MsgBlock
	block.Header = wire.BlockHeader{
		Version:    blockVersion,
		PrevBlock:  *prevHash,
		MerkleRoot: *merkles[len(merkles)-1],
		Timestamp:  ts,
		Bits:       net.PowLimitBits,
	}
	for _, tx := range blockTxns {
		if err := block.AddTransaction(tx.MsgTx()); err != nil {
			return nil, err
		}
	}

	found := solveBlock(&block.Header, net.PowLimit)
	if !found {
		return nil, errors.New("unable to solve block")
	}

	utilBlock := btcutil.NewBlock(&block)
	utilBlock.SetHeight(blockHeight)
	return utilBlock, nil
}


// solveBlock attempts to find a nonce which makes the passed block header hash
// to a value less than the target difficulty. When a successful solution is
// found true is returned and the nonce field of the passed header is updated
// with the solution. False is returned if no solution exists.
func solveBlock(header *wire.BlockHeader, targetDifficulty *big.Int) bool {
	// sbResult is used by the solver goroutines to send results.
	type sbResult struct {
		found bool
		nonce uint32
	}

	// solver accepts a block header and a nonce range to test. It is
	// intended to be run as a goroutine.
	quit := make(chan bool)
	results := make(chan sbResult)
	solver := func(hdr wire.BlockHeader, startNonce, stopNonce uint32) {
		// We need to modify the nonce field of the header, so make sure
		// we work with a copy of the original header.
		for i := startNonce; i >= startNonce && i <= stopNonce; i++ {
			select {
			case <-quit:
				return
			default:
				hdr.Nonce = i
				hash := hdr.BlockHash()
				if blockchain.HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
					select {
					case results <- sbResult{true, i}:
						return
					case <-quit:
						return
					}
				}
			}
		}
		select {
		case results <- sbResult{false, 0}:
		case <-quit:
			return
		}
	}

	startNonce := uint32(0)
	stopNonce := uint32(math.MaxUint32)
	numCores := uint32(runtime.NumCPU())
	noncesPerCore := (stopNonce - startNonce) / numCores
	for i := uint32(0); i < numCores; i++ {
		rangeStart := startNonce + (noncesPerCore * i)
		rangeStop := startNonce + (noncesPerCore * (i + 1)) - 1
		if i == numCores-1 {
			rangeStop = stopNonce
		}
		go solver(*header, rangeStart, rangeStop)
	}
	for i := uint32(0); i < numCores; i++ {
		result := <-results
		if result.found {
			close(quit)
			header.Nonce = result.nonce
			return true
		}
	}

	return false
}

// standardCoinbaseScript returns a standard script suitable for use as the
// signature script of the coinbase transaction of a new block. In particular,
// it starts with the block height that is required by version 2 blocks.
func standardCoinbaseScript(nextBlockHeight int32, extraNonce uint64) ([]byte, error) {
	return txscript.NewScriptBuilder().AddInt64(int64(nextBlockHeight)).
		AddInt64(int64(extraNonce)).Script()
}

// createCoinbaseTx returns a coinbase transaction paying an appropriate
// subsidy based on the passed block height to the provided address.
func createCoinbaseTx(coinbaseScript []byte, nextBlockHeight int32,
	addr btcutil.Address, mineTo []wire.TxOut,
	net *chaincfg.Params) (*btcutil.Tx, error) {

	// Create the script to pay to the provided payment address.
	pkScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, err
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxIn(&wire.TxIn{
		// Coinbase transactions have no inputs, so previous outpoint is
		// zero hash and max index.
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex),
		SignatureScript: coinbaseScript,
		Sequence:        wire.MaxTxInSequenceNum,
	})
	if len(mineTo) == 0 {
		tx.AddTxOut(&wire.TxOut{
			Value:    blockchain.CalcBlockSubsidy(nextBlockHeight, net),
			PkScript: pkScript,
		})
	} else {
		for i := range mineTo {
			tx.AddTxOut(&mineTo[i])
		}
	}
	return btcutil.NewTx(tx), nil
}

