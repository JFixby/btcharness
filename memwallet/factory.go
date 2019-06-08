// Copyright (c) 2018 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package memwallet

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/jfixby/btcharness"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
)

// WalletFactory produces a new InMemoryWallet-instance upon request
type WalletFactory struct {
}

// NewWallet creates and returns a fully initialized instance of the InMemoryWallet.
func (f *WalletFactory) NewWallet(cfg *coinharness.TestWalletConfig) coinharness.Wallet {
	pin.AssertNotNil("ActiveNet", cfg.ActiveNet)
	w, e := newMemWallet(cfg.ActiveNet.(*chaincfg.Params), cfg.Seed.([chainhash.HashSize + 4]byte))
	pin.CheckTestSetupMalfunction(e)
	return w
}

func newMemWallet(net *chaincfg.Params, harnessHDSeed [chainhash.HashSize + 4]byte) (*InMemoryWallet, error) {
	hdRoot, err := hdkeychain.NewMaster(harnessHDSeed[:], net)
	if err != nil {
		return nil, nil
	}

	// The first child key from the hd root is reserved as the coinbase
	// generation address.
	coinbaseChild, err := hdRoot.Child(0)
	if err != nil {
		return nil, err
	}
	coinbaseKey, err := coinbaseChild.ECPrivKey()
	if err != nil {
		return nil, err
	}
	coinbaseAddr, err := keyToAddr(coinbaseKey, net)
	if err != nil {
		return nil, err
	}

	// Track the coinbase generation address to ensure we properly track
	// newly generated coins we can spend.
	addrs := make(map[uint32]btcutil.Address)
	addrs[0] = coinbaseAddr

	clientFac := &btcharness.BtcRPCClientFactory{}

	return &InMemoryWallet{
		net:               net,
		coinbaseKey:       coinbaseKey,
		coinbaseAddr:      coinbaseAddr,
		hdIndex:           1,
		hdRoot:            hdRoot,
		addrs:             addrs,
		utxos:             make(map[wire.OutPoint]*utxo),
		chainUpdateSignal: make(chan string),
		reorgJournal:      make(map[int32]*undoEntry),
		RPCClientFactory:  clientFac,
	}, nil
}
