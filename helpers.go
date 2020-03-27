package btcharness

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/jfixby/coin"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
	"github.com/picfight/pfcd/dcrutil"
	"github.com/picfight/pfcd/rpcclient"
	"math/big"
	"time"

	"github.com/picfight/pfcd/blockchain"
	"github.com/picfight/pfcd/chaincfg"
	"github.com/picfight/pfcd/chaincfg/chainhash"
	"github.com/picfight/pfcd/txscript"
	"github.com/picfight/pfcd/wire"
)

// GenerateBlockArgs bundles GenerateBlock() arguments to minimize diff
// in case a new argument for the function is added
type GenerateBlockArgs struct {
	Txns          []*dcrutil.Tx
	BlockVersion  int32
	BlockTime     time.Time
	MineTo        []wire.TxOut
	MiningAddress dcrutil.Address
	Network       *chaincfg.Params
}

// GenerateAndSubmitBlock creates a block whose contents include the passed
// transactions and submits it to the running simnet node. For generating
// blocks with only a coinbase tx, callers can simply pass nil instead of
// transactions to be mined. Additionally, a custom block version can be set by
// the caller. An uninitialized time.Time should be used for the
// blockTime parameter if one doesn't wish to set a custom time.
func GenerateAndSubmitBlock(client coinharness.RPCClient, args *GenerateBlockArgs) (*dcrutil.Block, error) {
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
func GenerateAndSubmitBlockWithCustomCoinbaseOutputs(client coinharness.RPCClient, args *GenerateBlockArgs) (*dcrutil.Block, error) {
	txns := args.Txns
	blockVersion := args.BlockVersion
	pin.AssertTrue(fmt.Sprintf("Incorrect blockVersion(%v)", blockVersion), blockVersion > 0)
	blockTime := args.BlockTime
	mineTo := args.MineTo
	miningAddress := args.MiningAddress
	network := args.Network

	pin.AssertTrue("blockVersion != -1", blockVersion != -1)

	prevBlockHash, prevBlockHeight, err := client.Internal().(*rpcclient.Client).GetBestBlock()
	if err != nil {
		return nil, err
	}
	mBlock, err := client.Internal().(*rpcclient.Client).GetBlock(prevBlockHash)
	if err != nil {
		return nil, err
	}
	prevBlock := dcrutil.NewBlock(mBlock)
	mBlock.Header.Height = uint32(prevBlockHeight)

	// Create a new block including the specified transactions
	newBlock, err := CreateBlock(prevBlock, txns, blockVersion,
		blockTime, miningAddress, mineTo, network)
	if err != nil {
		return nil, err
	}

	// Submit the block to the simnet node.
	if err := client.Internal().(*rpcclient.Client).SubmitBlock(newBlock, nil); err != nil {
		return nil, err
	}

	return newBlock, nil
}

// CreateBlock creates a new block building from the previous block with a
// specified blockversion and timestamp. If the timestamp passed is zero (not
// initialized), then the timestamp of the previous block will be used plus 1
// second is used. Passing nil for the previous block results in a block that
// builds off of the genesis block for the specified chain.
func CreateBlock(prevBlock *dcrutil.Block, inclusionTxs []*dcrutil.Tx,
	blockVersion int32, blockTime time.Time, miningAddr dcrutil.Address,
	mineTo []wire.TxOut, net *chaincfg.Params) (*dcrutil.Block, error) {

	var (
		prevHash      *chainhash.Hash
		blockHeight   int64
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
		blockHeight = (prevBlock.Height() + 1)
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
	blockTxns := []*dcrutil.Tx{coinbaseTx}
	if inclusionTxs != nil {
		blockTxns = append(blockTxns, inclusionTxs...)
	}
	merkles := blockchain.BuildMerkleTreeStore(blockTxns)
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

	utilBlock := dcrutil.NewBlock(&block)
	utilBlock.MsgBlock().Header.Height = uint32(blockHeight)
	return utilBlock, nil
}

// solveBlock attempts to find a nonce which makes the passed block header hash
// to a value less than the target difficulty. When a successful solution is
// found true is returned and the nonce field of the passed header is updated
// with the solution. False is returned if no solution exists.
func solveBlock(header *wire.BlockHeader, targetDifficulty *big.Int) bool {
	// Note that the entire extra nonce range is iterated and the offset is
	// added relying on the fact that overflow will wrap around 0 as
	// provided by the Go spec.
	for i := uint32(0); ; i++ {
		// Update the nonce and hash the block header.
		header.Nonce = i
		hash := header.BlockHash()
		// The block is solved when the new block hash is less
		// than the target difficulty.  Yay!
		blockHash := blockchain.HashToBig(&hash)
		if blockHash.Cmp(targetDifficulty) <= 0 {
			pin.D("       blockHash", blockHash)
			pin.D("targetDifficulty", targetDifficulty)
			return true
		}
	}
	return false
}

// standardCoinbaseScript returns a standard script suitable for use as the
// signature script of the coinbase transaction of a new block. In particular,
// it starts with the block height that is required by version 2 blocks.
func standardCoinbaseScript(nextBlockHeight int64, extraNonce uint64) ([]byte, error) {
	return txscript.NewScriptBuilder().AddInt64(int64(nextBlockHeight)).
		AddInt64(int64(extraNonce)).Script()
}

// TxTreeRegular is the value for a normal transaction tree for a
// transaction's location in a block.
const TxTreeRegular int8 = 0

// createCoinbaseTx returns a coinbase transaction paying an appropriate
// subsidy based on the passed block height to the provided address.
func createCoinbaseTx(coinbaseScript []byte, nextBlockHeight int64,
	addr dcrutil.Address, mineTo []wire.TxOut,
	params *chaincfg.Params) (*dcrutil.Tx, error) {

	tx := wire.NewMsgTx()
	tx.AddTxIn(&wire.TxIn{
		// Coinbase transactions have no inputs, so previous outpoint is
		// zero hash and max index.
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex, wire.TxTreeRegular),
		Sequence:        wire.MaxTxInSequenceNum,
		BlockHeight:     wire.NullBlockHeight,
		BlockIndex:      wire.NullBlockIndex,
		SignatureScript: coinbaseScript,
	})

	// Block one is a special block that might pay out tokens to a ledger.
	if nextBlockHeight == 1 && len(params.BlockOneLedger) != 0 {
		// Convert the addresses in the ledger into useable format.
		addrs := make([]dcrutil.Address, len(params.BlockOneLedger))
		for i, payout := range params.BlockOneLedger {
			addr, err := dcrutil.DecodeAddress(payout.Address)
			if err != nil {
				return nil, err
			}
			addrs[i] = addr
		}

		for i, payout := range params.BlockOneLedger {
			// Make payout to this address.
			pks, err := txscript.PayToAddrScript(addrs[i])
			if err != nil {
				return nil, err
			}
			tx.AddTxOut(&wire.TxOut{
				Value:    payout.Amount,
				PkScript: pks,
			})
		}

		tx.TxIn[0].ValueIn = params.BlockOneSubsidy()

		return dcrutil.NewTx(tx), nil
	}

	subsidyCache := blockchain.NewSubsidyCache(0, params)
	voters := params.TicketsPerBlock
	// Create a coinbase with correct block subsidy and extranonce.
	subsidy := blockchain.CalcBlockWorkSubsidy(subsidyCache,
		nextBlockHeight,
		voters,
		params)
	tax := blockchain.CalcBlockTaxSubsidy(subsidyCache,
		nextBlockHeight,
		voters,
		params)

	// Tax output.
	if params.BlockTaxProportion > 0 {
		tx.AddTxOut(&wire.TxOut{
			Value:    tax,
			PkScript: params.OrganizationPkScript,
		})
	} else {
		// Tax disabled.
		scriptBuilder := txscript.NewScriptBuilder()
		trueScript, err := scriptBuilder.AddOp(txscript.OP_TRUE).Script()
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(&wire.TxOut{
			Value:    tax,
			PkScript: trueScript,
		})
	}

	random, err := wire.RandomUint64()
	if err != nil {
		return nil, err
	}
	height := nextBlockHeight
	opReturnPkScript, err := standardCoinbaseOpReturn(height, random)

	// Extranonce.
	tx.AddTxOut(&wire.TxOut{
		Value:    0,
		PkScript: opReturnPkScript,
	})
	// ValueIn.
	tx.TxIn[0].ValueIn = subsidy + tax

	// Create the script to pay to the provided payment address if one was
	// specified.  Otherwise create a script that allows the coinbase to be
	// redeemable by anyone.
	var pksSubsidy []byte
	if addr != nil {
		var err error
		pksSubsidy, err = txscript.PayToAddrScript(addr)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		scriptBuilder := txscript.NewScriptBuilder()
		pksSubsidy, err = scriptBuilder.AddOp(txscript.OP_TRUE).Script()
		if err != nil {
			return nil, err
		}
	}
	// Subsidy paid to miner.
	tx.AddTxOut(&wire.TxOut{
		Value:    subsidy,
		PkScript: pksSubsidy,
	})

	return dcrutil.NewTx(tx), nil
}

// standardCoinbaseOpReturn creates a standard OP_RETURN output to insert into
// coinbase to use as extranonces. The OP_RETURN pushes 32 bytes.
func standardCoinbaseOpReturn(height int64, extraNonce uint64) ([]byte, error) {
	enData := make([]byte, 12)
	binary.LittleEndian.PutUint32(enData[0:4], uint32(height))
	binary.LittleEndian.PutUint64(enData[4:12], extraNonce)
	extraNonceScript, err := txscript.GenerateProvablyPruneableOut(enData)
	if err != nil {
		return nil, err
	}

	return extraNonceScript, nil
}

func TransactionTxToRaw(chTx *coinharness.MessageTx) *wire.MsgTx {
	wireTx := &wire.MsgTx{
		//CachedHash: chTx.CachedHash.(*chainhash.Hash),
		SerType:  wire.TxSerializeType(chTx.SerType),
		Version:  uint16(chTx.Version),
		LockTime: chTx.LockTime,
		Expiry:   chTx.Expiry,
	}
	for _, ti := range chTx.TxIn {
		wireTx.TxIn = append(wireTx.TxIn,
			&wire.TxIn{
				ValueIn:         ti.ValueIn.ToAtoms(),
				SignatureScript: ti.SignatureScript,
				BlockHeight:     ti.BlockHeight,
				BlockIndex:      ti.BlockIndex,
				PreviousOutPoint: wire.OutPoint{
					Hash:  ti.PreviousOutPoint.Hash.(chainhash.Hash),
					Index: ti.PreviousOutPoint.Index,
					Tree:  ti.PreviousOutPoint.Tree,
				},
			},
		)
	}
	for _, to := range chTx.TxOut {
		wireTx.TxOut = append(wireTx.TxOut,
			&wire.TxOut{
				Value:    to.Value.ToAtoms(),
				Version:  to.Version,
				PkScript: to.PkScript,
			},
		)
	}

	return wireTx
}

func TransactionRawToTx(wireTx *wire.MsgTx) *coinharness.MessageTx {
	wireTx = wireTx.Copy()
	chTx := &coinharness.MessageTx{
		//CachedHash: wireTx.CachedHash,
		SerType:  uint16(wireTx.SerType),
		Version:  int32(wireTx.Version),
		LockTime: wireTx.LockTime,
		Expiry:   wireTx.Expiry,
	}
	for _, ti := range wireTx.TxIn {
		chTx.TxIn = append(chTx.TxIn,
			&coinharness.TxIn{
				ValueIn:         coin.Amount{ti.ValueIn},
				SignatureScript: ti.SignatureScript,
				BlockHeight:     ti.BlockHeight,
				BlockIndex:      ti.BlockIndex,
				PreviousOutPoint: coinharness.OutPoint{
					Hash:  ti.PreviousOutPoint.Hash,
					Index: ti.PreviousOutPoint.Index,
					Tree:  ti.PreviousOutPoint.Tree,
				},
			},
		)
	}
	for _, to := range wireTx.TxOut {
		chTx.TxOut = append(chTx.TxOut,
			&coinharness.TxOut{
				Value:    coin.Amount{to.Value},
				Version:  to.Version,
				PkScript: to.PkScript,
			},
		)
	}

	chTx.TxHash = func() coinharness.Hash {
		return wireTx.TxHash()
	}

	return chTx
}

func PayToAddrScript(addr coinharness.Address) ([]byte, error) {
	return txscript.PayToAddrScript(addr.Internal().(dcrutil.Address))
}

func TxSerializeSize(msg *coinharness.MessageTx) int {
	raw := TransactionTxToRaw(msg)
	return raw.SerializeSize()
}
