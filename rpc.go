// Copyright (c) 2018 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcharness

import (
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
	"io/ioutil"
)

type BtcRPCClientFactory struct {
}

func (f *BtcRPCClientFactory) NewRPCConnection(config coinharness.RPCConnectionConfig, handlers coinharness.RPCClientNotificationHandlers) (coinharness.RPCClient, error) {
	var h *rpcclient.NotificationHandlers
	if handlers != nil {
		h = handlers.
		(*rpcclient.NotificationHandlers)
	}

	file := config.CertificateFile
	fmt.Println("reading: " + file)
	cert, err := ioutil.ReadFile(file)
	pin.CheckTestSetupMalfunction(err)

	cfg := &rpcclient.ConnConfig{
		Host:                 config.Host,
		Endpoint:             config.Endpoint,
		User:                 config.User,
		Pass:                 config.Pass,
		Certificates:         cert,
		DisableAutoReconnect: true,
		HTTPPostMode:         false,
	}

	return NewRPCClient(cfg, h)
}

type BTCPCClient struct {
	rpc *rpcclient.Client
}

func (c *BTCPCClient) AddNode(args *coinharness.AddNodeArguments) error {
	return c.rpc.AddNode(args.TargetAddr, args.Command.(rpcclient.AddNodeCommand))
}

func (c *BTCPCClient) Disconnect() {
	c.rpc.Disconnect()
}

func (c *BTCPCClient) Shutdown() {
	c.rpc.Shutdown()
}

func (c *BTCPCClient) NotifyBlocks() error {
	return c.rpc.NotifyBlocks()
}

func (c *BTCPCClient) GetBlockCount() (int64, error) {
	return c.rpc.GetBlockCount()
}

func (c *BTCPCClient) Generate(blocks uint32) (result []coinharness.Hash, e error) {
	list, e := c.rpc.Generate(blocks)
	if e != nil {
		return nil, e
	}
	for _, el := range list {
		result = append(result, el)
	}
	return result, nil
}

func (c *BTCPCClient) Internal() interface{} {
	return c.rpc
}

func (c *BTCPCClient) GetRawMempool(_ interface{}) (result []coinharness.Hash, e error) {
	list, e := c.rpc.GetRawMempool()
	if e != nil {
		return nil, e
	}
	for _, el := range list {
		result = append(result, el)
	}
	return result, nil
}

func (c *BTCPCClient) SendRawTransaction(tx coinharness.CreatedTransactionTx, allowHighFees bool) (result coinharness.Hash, e error) {
	txx := TransactionTxToRaw(tx)
	r, e := c.rpc.SendRawTransaction(txx, allowHighFees)
	return r, e
}

func (c *BTCPCClient) GetPeerInfo() ([]coinharness.PeerInfo, error) {
	pif, err := c.rpc.GetPeerInfo()
	if err != nil {
		return nil, err
	}

	l := []coinharness.PeerInfo{}
	for _, i := range pif {
		inf := coinharness.PeerInfo{}
		inf.Addr = i.Addr
		l = append(l, inf)

	}
	return l, nil
}

func NewRPCClient(config *rpcclient.ConnConfig, handlers *rpcclient.NotificationHandlers) (coinharness.RPCClient, error) {
	legacy, err := rpcclient.New(config, handlers)
	if err != nil {
		return nil, err
	}

	result := &BTCPCClient{rpc: legacy}
	return result, nil
}

func (c *BTCPCClient) GetNewAddress(account string) (coinharness.Address, error) {
	legacy, err := c.rpc.GetNewAddress(account)
	if err != nil {
		return nil, err
	}

	result := &BTCAddress{Address: legacy}
	return result, nil
}

func (c *BTCPCClient) ValidateAddress(address coinharness.Address) (*coinharness.ValidateAddressResult, error) {
	legacy, err := c.rpc.ValidateAddress(address.Internal().(btcutil.Address))
	// *btcjson.ValidateAddressWalletResult
	if err != nil {
		return nil, err
	}
	result := &coinharness.ValidateAddressResult{
		Address:      legacy.Address,
		Account:      legacy.Account,
		IsValid:      legacy.IsValid,
		IsMine:       legacy.IsMine,
		IsCompressed: legacy.IsCompressed,
	}
	return result, nil
}

func (c *BTCPCClient) GetBalance(account string) (*coinharness.GetBalanceResult, error) {
	legacy, err := c.rpc.GetBalance(account)
	// *btcjson.ValidateAddressWalletResult
	if err != nil {
		return nil, err
	}
	result := &coinharness.GetBalanceResult{
		BlockHash:      legacy.BlockHash,
		TotalSpendable: legacy.TotalSpendable,
	}
	return result, nil
}

func (c *BTCPCClient) GetBestBlock() (coinharness.Hash, int64, error) {
	return c.rpc.GetBestBlock()
}

func (c *BTCPCClient) CreateNewAccount(account string) error {
	return c.rpc.CreateNewAccount(account)
}

func (c *BTCPCClient) WalletLock() error {
	return c.rpc.WalletLock()
}

func (c *BTCPCClient) WalletInfo() (*coinharness.WalletInfoResult, error) {
	r, err := c.rpc.WalletInfo()
	if err != nil {
		return nil, err
	}
	result := &coinharness.WalletInfoResult{
		Unlocked:        r.Unlocked,
		DaemonConnected: r.DaemonConnected,
		Voting:          r.DaemonConnected,
	}
	return result, nil
}

func (c *BTCPCClient) WalletUnlock(passphrase string, timeoutSecs int64) error {
	return c.rpc.WalletPassphrase(passphrase, timeoutSecs)
}

func (c *BTCPCClient) CreateTransaction(*coinharness.CreateTransactionArgs) (coinharness.CreatedTransactionTx, error) {
	panic("")
}

func (c *BTCPCClient) GetBuildVersion() (coinharness.BuildVersion, error) {
	//legacy, err := c.rpc.GetBuildVersion()
	//if err != nil {
	//	return nil, err
	//}
	//return legacy, nil
	return nil, fmt.Errorf("bitcoin does not support this feature (GetBuildVersion)")
}

type BTCAddress struct {
	Address btcutil.Address
}

func (c *BTCAddress) String() string {
	return c.Address.String()
}

func (c *BTCAddress) Internal() interface{} {
	return c.Address
}

func (c *BTCAddress) IsForNet(net coinharness.Network) bool {
	return c.Address.IsForNet(net.(*chaincfg.Params))
}
