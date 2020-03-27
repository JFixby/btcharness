package btcharness

import (
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
	"github.com/picfight/pfcd/blockchain"
	"github.com/picfight/pfcd/chaincfg"
	"github.com/picfight/pfcd/dcrec/secp256k1"
	"github.com/picfight/pfcd/dcrutil"
	"github.com/picfight/pfcd/hdkeychain"
	"github.com/picfight/pfcd/wire"
)

// InMemoryWalletFactory produces a new InMemoryWallet-instance upon request
type InMemoryWalletFactory struct {
}

// NewWallet creates and returns a fully initialized instance of the InMemoryWallet.
func (f *InMemoryWalletFactory) NewWallet(cfg *coinharness.TestWalletConfig) coinharness.Wallet {
	pin.AssertNotNil("ActiveNet", cfg.ActiveNet)
	//w, e := newMemWallet(, cfg.Seed)

	net := cfg.ActiveNet
	harnessHDSeed := cfg.Seed.([]byte)[:]
	hdRoot, err := hdkeychain.NewMaster(harnessHDSeed, net.Params().(*chaincfg.Params))
	//hdRoot, err := cfg.NewMasterKeyFromSeed(harnessHDSeed, net)
	pin.CheckTestSetupMalfunction(err)

	var ekey coinharness.ExtendedKey = &ExtendedKey{hdRoot}
	// The first child key from the hd root is reserved as the coinbase
	// generation address.
	coinbaseChild, err := hdRoot.Child(0)
	pin.CheckTestSetupMalfunction(err)
	coinbaseKey, err := coinbaseChild.ECPrivKey()
	pin.CheckTestSetupMalfunction(err)
	coinbaseAddr, err := PrivateKeyKeyToAddr(&PrivateKey{coinbaseKey}, cfg.ActiveNet)
	pin.CheckTestSetupMalfunction(err)

	// Track the coinbase generation address to ensure we properly track
	// newly generated coins we can spend.
	addrs := make(map[uint32]coinharness.Address)
	addrs[0] = coinbaseAddr

	clientFac := &RPCClientFactory{}
	//clientFac := cfg.RPCClientFactory
	return &coinharness.InMemoryWallet{
		Net:                 net,
		CoinbaseKey:         coinbaseKey,
		CoinbaseAddr:        coinbaseAddr,
		HdIndex:             1,
		HdRoot:              ekey,
		Addrs:               addrs,
		Utxos:               make(map[coinharness.OutPoint]*coinharness.Utxo),
		ChainUpdateSignal:   make(chan string),
		ReorgJournal:        make(map[int64]*coinharness.UndoEntry),
		RPCClientFactory:    clientFac,
		PrivateKeyKeyToAddr: PrivateKeyKeyToAddr,
		ReadBlockHeader:     ReadBlockHeader,
		NewTxFromBytes:      NewTxFromBytes,
		IsCoinBaseTx:        IsCoinBaseTx,
	}
	//NewTxFromBytes      func(txBytes []byte) (*Tx, error) //dcrutil.NewTxFromBytes(txBytes)
	//IsCoinBaseTx        func(*MessageTx) bool             //blockchain.IsCoinBaseTx(mtx)
}

func IsCoinBaseTx(tx *coinharness.MessageTx) bool {
	mtx := TransactionTxToRaw(tx)
	return blockchain.IsCoinBaseTx(mtx)
}

func NewTxFromBytes(txBytes []byte) (*coinharness.Tx, error) {
	dcrTx, err := dcrutil.NewTxFromBytes(txBytes)
	if err != nil {
		return nil, err
	}
	chTx := &coinharness.Tx{
		Hash:   dcrTx.Hash(),
		TxTree: dcrTx.Tree(),
		Index:  dcrTx.Index(),
		MsgTx:  TransactionRawToTx(dcrTx.MsgTx()),
	}
	return chTx, nil
}

// PrivateKeyKeyToAddr maps the passed private to corresponding p2pkh address.
func PrivateKeyKeyToAddr(key coinharness.PrivateKey, net coinharness.Network) (coinharness.Address, error) {
	k := key.(*PrivateKey).legacy
	addr, err := keyToAddr(k, net.Params().(*chaincfg.Params))
	if err != nil {
		return nil, err
	}
	return &Address{Address: addr}, nil
}

// keyToAddr maps the passed private to corresponding p2pkh address.
func keyToAddr(key *secp256k1.PrivateKey, net *chaincfg.Params) (dcrutil.Address, error) {
	pubKey := (*secp256k1.PublicKey)(&key.PublicKey)
	serializedKey := pubKey.SerializeCompressed()
	pubKeyAddr, err := dcrutil.NewAddressSecpPubKey(serializedKey, net)
	if err != nil {
		return nil, err
	}
	return pubKeyAddr.AddressPubKeyHash(), nil
}

func ReadBlockHeader(header []byte) coinharness.BlockHeader {
	var hdr wire.BlockHeader
	if err := hdr.FromBytes(header); err != nil {
		panic(err)
	}
	return &BlockHeader{legacy: hdr}
}
