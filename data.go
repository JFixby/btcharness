package btcharness

import (
	"crypto/ecdsa"
	"github.com/jfixby/coinharness"
	"github.com/picfight/pfcd/chaincfg"
	"github.com/picfight/pfcd/dcrec/secp256k1"
	"github.com/picfight/pfcd/dcrutil"
	"github.com/picfight/pfcd/hdkeychain"
	"github.com/picfight/pfcd/wire"
)

type Address struct {
	Address dcrutil.Address
}

func (c *Address) ScriptAddress() []byte {
	return c.Address.ScriptAddress()
}

func (c *Address) String() string {
	return c.Address.String()
}

func (c *Address) Internal() interface{} {
	return c.Address
}

func (c *Address) IsForNet(net coinharness.Network) bool {
	return c.Address.IsForNet(net.Params().(*chaincfg.Params))
}

type BlockHeader struct {
	legacy wire.BlockHeader
}

func (h *BlockHeader) Height() int64 {
	return int64(h.legacy.Height)
}

type PrivateKey struct {
	legacy *secp256k1.PrivateKey
}

func (k *PrivateKey) PublicKey() coinharness.PublicKey {
	return PublicKey{legacy: k.legacy.PublicKey}
}

type PublicKey struct {
	legacy ecdsa.PublicKey
}

type ExtendedKey struct {
	legacy *hdkeychain.ExtendedKey
}

func (k *ExtendedKey) PrivateKey() (coinharness.PrivateKey, error) {
	ck, err := k.legacy.ECPrivKey()
	if err != nil {
		return nil, err
	}
	return &PrivateKey{ck}, nil
}

func (k *ExtendedKey) Child(u uint32) (coinharness.ExtendedKey, error) {
	ck, err := k.legacy.Child(u)
	if err != nil {
		return nil, err
	}
	return &ExtendedKey{ck}, nil
}
