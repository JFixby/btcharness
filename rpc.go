// Copyright (c) 2018 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcharness

import (
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
	"io/ioutil"
)

type RPCClientFactory struct {
}

func (f *RPCClientFactory) NewRPCConnection(config coinharness.RPCConnectionConfig, handlers coinharness.RPCClientNotificationHandlers) (coinharness.RPCClient, error) {
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

type RPCClient struct {
	rpc *rpcclient.Client
}

func (c *RPCClient) SubmitBlock(block coinharness.Block) (error) {
	return c.rpc.SubmitBlock(block.(*btcutil.Block), nil)
}

func (c *RPCClient) LoadTxFilter(reload bool, addr []coinharness.Address) (error) {
	addresses := []btcutil.Address{}
	for _, e := range addr {
		addresses = append(addresses, e.Internal().(btcutil.Address))
	}
	return c.rpc.LoadTxFilter(reload, addresses, nil)
}

func (c *RPCClient) AddNode(args *coinharness.AddNodeArguments) error {
	return c.rpc.AddNode(args.TargetAddr, args.Command.(rpcclient.AddNodeCommand))
}

func (c *RPCClient) Disconnect() {
	c.rpc.Disconnect()
}

func (c *RPCClient) Shutdown() {
	c.rpc.Shutdown()
}

func (c *RPCClient) GetBlock(hash coinharness.Hash) (coinharness.Block, error) {
	return c.rpc.GetBlock(hash.(*chainhash.Hash))
}

func (c *RPCClient) NotifyBlocks() error {
	return c.rpc.NotifyBlocks()
}

func (c *RPCClient) GetBlockCount() (int64, error) {
	return c.rpc.GetBlockCount()
}

func (c *RPCClient) Generate(blocks uint32) (result []coinharness.Hash, e error) {
	list, e := c.rpc.Generate(blocks)
	if e != nil {
		return nil, e
	}
	for _, el := range list {
		result = append(result, el)
	}
	return result, nil
}

func (c *RPCClient) Internal() interface{} {
	return c.rpc
}

func (c *RPCClient) GetRawMempool(_ interface{}) (result []coinharness.Hash, e error) {
	list, e := c.rpc.GetRawMempool()
	if e != nil {
		return nil, e
	}
	for _, el := range list {
		result = append(result, el)
	}
	return result, nil
}

func (c *RPCClient) SendRawTransaction(tx coinharness.CreatedTransactionTx, allowHighFees bool) (result coinharness.Hash, e error) {
	txx := TransactionTxToRaw(tx)
	r, e := c.rpc.SendRawTransaction(txx, allowHighFees)
	return r, e
}

func (c *RPCClient) GetPeerInfo() ([]coinharness.PeerInfo, error) {
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

	result := &RPCClient{rpc: legacy}
	return result, nil
}

func (c *RPCClient) GetNewAddress(account string) (coinharness.Address, error) {
	legacy, err := c.rpc.GetNewAddress(account)
	if err != nil {
		return nil, err
	}

	result := &Address{Address: legacy}
	return result, nil
}

func (c *RPCClient) ValidateAddress(address coinharness.Address) (*coinharness.ValidateAddressResult, error) {
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

func (c *RPCClient) GetBalance(account string) (*coinharness.GetBalanceResult, error) {
	legacy, err := c.rpc.GetBalance(account)
	// *btcjson.ValidateAddressWalletResult
	if err != nil {
		return nil, err
	}
	result := &coinharness.GetBalanceResult{
		TotalSpendable: legacy,
	}
	return result, nil
}

func (c *RPCClient) GetBestBlock() (coinharness.Hash, int64, error) {
	x, y, z := c.rpc.GetBestBlock()
	return x, int64(y), z
}

func (c *RPCClient) CreateNewAccount(account string) error {
	return c.rpc.CreateNewAccount(account)
}

func (c *RPCClient) WalletLock() error {
	return c.rpc.WalletLock()
}

func (c *RPCClient) WalletInfo() (*coinharness.WalletInfoResult, error) {
	//result := &coinharness.WalletInfoResult{
	//}
	return nil, fmt.Errorf("method is not supported WalletInfo()")
}

func (c *RPCClient) WalletUnlock(passphrase string, timeoutSecs int64) error {
	return c.rpc.WalletPassphrase(passphrase, timeoutSecs)
}

func (c *RPCClient) CreateTransaction(*coinharness.CreateTransactionArgs) (coinharness.CreatedTransactionTx, error) {
	panic("")
}

func (c *RPCClient) GetBuildVersion() (coinharness.BuildVersion, error) {
	//legacy, err := c.rpc.GetBuildVersion()
	//if err != nil {
	//	return nil, err
	//}
	//return legacy, nil
	return nil, fmt.Errorf("bitcoin does not support this feature (GetBuildVersion)")
}

type Address struct {
	Address btcutil.Address
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
