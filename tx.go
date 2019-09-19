package dcrharness

import (
	"github.com/decred/dcrd/wire"
	"github.com/jfixby/coinharness"
)

//------------------------------------------
type OutputTx struct {
	Parent *wire.TxOut
}

func (o *OutputTx) PkScript() []byte {
	return o.Parent.PkScript
}

func (o *OutputTx) Value() int64 {
	return o.Parent.Value
}

//------------------------------------------
type InputTx struct {
	Parent *wire.TxIn
}

func (o *InputTx) PreviousOutPoint() coinharness.OutPoint {
	return o.Parent.PreviousOutPoint
}

//------------------------------------------
type CreatedTransactionTx struct {
	Parent *wire.MsgTx
}

func (o *CreatedTransactionTx) LockTime() uint32 {
	return o.Parent.LockTime
}

func (o *CreatedTransactionTx) Version() int32 {
	return int32(o.Parent.Version)
}

func (o *CreatedTransactionTx) TxHash() coinharness.Hash {
	return o.Parent.TxHash()
}

func (o *CreatedTransactionTx) TxIn() (result []coinharness.InputTx) {
	result = []coinharness.InputTx{}
	for _, ti := range o.Parent.TxIn {
		result = append(result, &InputTx{ti})
	}
	return
}

func (o *CreatedTransactionTx) TxOut() (result []coinharness.OutputTx) {
	result = []coinharness.OutputTx{}
	for _, ti := range o.Parent.TxOut {
		result = append(result, &OutputTx{ti})
	}
	return
}
