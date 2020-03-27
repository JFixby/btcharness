package btcharness

import (
	"fmt"
	"github.com/jfixby/coin"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
	"github.com/picfight/pfcd/chaincfg/chainhash"
	"github.com/picfight/pfcd/dcrjson"
	"github.com/picfight/pfcd/dcrutil"
	"github.com/picfight/pfcd/rpcclient"
	"io/ioutil"
)

type RPCClientFactory struct {
}

func (f *RPCClientFactory) NewRPCConnection(config coinharness.RPCConnectionConfig, handlers *coinharness.NotificationHandlers) (coinharness.RPCClient, error) {
	h := ConvertHandlers(handlers)

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

func ConvertHandlers(handlers *coinharness.NotificationHandlers) *rpcclient.NotificationHandlers {
	if handlers == nil {
		return nil
	}
	return &rpcclient.NotificationHandlers{
		//
		OnClientConnected: handlers.OnClientConnected,
		//
		OnBlockConnected: handlers.OnBlockConnected,
		//
		OnBlockDisconnected: handlers.OnBlockDisconnected,
		//
		OnRelevantTxAccepted: handlers.OnRelevantTxAccepted,
		//
		OnWinningTickets: func(
			blockHash *chainhash.Hash,
			blockHeight int64,
			tickets []*chainhash.Hash,
		) {
			ts := []coinharness.Hash{}
			for _, e := range tickets {
				ts = append(ts, e)
			}
			handlers.OnWinningTickets(
				blockHash,
				blockHeight,
				ts,
			)
		},
		//
		OnSpentAndMissedTickets: func(
			hash *chainhash.Hash,
			height int64,
			stakeDiff int64,
			tickets map[chainhash.Hash]bool,
		) {
			ts := make(map[coinharness.Hash]bool)
			for k, v := range tickets {
				ts[k] = v
			}
			handlers.OnSpentAndMissedTickets(
				hash,
				height,
				stakeDiff,
				ts,
			)
		},
		//
		OnNewTickets: func(
			hash *chainhash.Hash,
			height int64,
			stakeDiff int64,
			tickets []*chainhash.Hash,
		) {
			ts := []coinharness.Hash{}
			for _, e := range tickets {
				ts = append(ts, e)
			}
			handlers.OnNewTickets(
				hash,
				height,
				stakeDiff,
				ts,
			)

		},
		//
		OnStakeDifficulty: func(
			hash *chainhash.Hash,
			height int64,
			stakeDiff int64,
		) {
			handlers.OnStakeDifficulty(
				hash,
				height,
				stakeDiff,
			)
		},
		//
		OnTxAccepted: func(
			hash *chainhash.Hash,
			amount dcrutil.Amount,
		) {
			handlers.OnTxAccepted(
				hash,
				coin.Amount{int64(amount)},
			)
		},
		//
		//OnTxAcceptedVerbose:     handlers.OnTxAcceptedVerbose,
		OnDcrdConnected: handlers.OnNodeConnected,
		//
		OnAccountBalance: func(
			account string,
			balance dcrutil.Amount,
			confirmed bool,
		) {
			handlers.OnAccountBalance(
				account,
				coin.Amount{int64(balance)},
				confirmed,
			)
		},
		//
		OnWalletLockState: handlers.OnWalletLockState,
		//
		OnTicketsPurchased: func(
			TxHash *chainhash.Hash,
			amount dcrutil.Amount,
		) {
			handlers.OnTicketsPurchased(
				TxHash,
				coin.Amount{int64(amount)},
			)
		},
		//
		OnVotesCreated: func(
			txHash *chainhash.Hash,
			blockHash *chainhash.Hash,
			height int32,
			sstxIn *chainhash.Hash,
			voteBits uint16,
		) {
			handlers.OnVotesCreated(
				txHash,
				blockHash,
				height,
				sstxIn,
				voteBits,
			)
		},
		//
		OnRevocationsCreated: func(
			txHash *chainhash.Hash,
			sstxIn *chainhash.Hash,
		) {
			handlers.OnRevocationsCreated(
				txHash,
				sstxIn,
			)
		},
		//
		OnUnknownNotification: handlers.OnUnknownNotification,
	}
}

func NewRPCClient(config *rpcclient.ConnConfig, handlers *rpcclient.NotificationHandlers) (coinharness.RPCClient, error) {
	legacy, err := rpcclient.New(config, handlers)
	if err != nil {
		return nil, err
	}

	result := &RPCClient{rpc: legacy}
	return result, nil
}

type RPCClient struct {
	rpc *rpcclient.Client
}

func (c *RPCClient) ListUnspent() ([]*coinharness.Unspent, error) {
	result, err := c.rpc.ListUnspent()
	if err != nil {
		return nil, err
	}
	var r []*coinharness.Unspent
	for _, e := range result {
		x := &coinharness.Unspent{}

		x.TxID = e.TxID
		x.Vout = e.Vout
		x.Tree = e.Tree
		x.TxType = e.TxType
		x.Address = e.Address
		x.Account = e.Account
		x.ScriptPubKey = e.ScriptPubKey
		x.RedeemScript = e.RedeemScript
		x.Amount = coin.FromFloat(e.Amount)
		x.Confirmations = e.Confirmations
		x.Spendable = e.Spendable

		r = append(r, x)
	}

	return r, nil
}

func (c *RPCClient) AddNode(args *coinharness.AddNodeArguments) error {
	return c.rpc.AddNode(args.TargetAddr, args.Command.(rpcclient.AddNodeCommand))
}

func (c *RPCClient) LoadTxFilter(reload bool, addr []coinharness.Address) error {
	addresses := []dcrutil.Address{}
	for _, e := range addr {
		addresses = append(addresses, e.Internal().(dcrutil.Address))
	}
	return c.rpc.LoadTxFilter(reload, addresses, nil)
}

func (c *RPCClient) SubmitBlock(block coinharness.Block) error {
	return c.rpc.SubmitBlock(block.(*dcrutil.Block), nil)
}

func (c *RPCClient) Disconnect() {
	c.rpc.Disconnect()
}

func (c *RPCClient) Shutdown() {
	c.rpc.Shutdown()
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

func (c *RPCClient) GetRawMempool(command interface{}) (result []coinharness.Hash, e error) {
	list, e := c.rpc.GetRawMempool(command.(dcrjson.GetRawMempoolTxTypeCmd))
	if e != nil {
		return nil, e
	}
	for _, el := range list {
		result = append(result, el)
	}
	return result, nil
}

func (c *RPCClient) SendRawTransaction(tx *coinharness.MessageTx, allowHighFees bool) (result coinharness.Hash, e error) {
	txx := TransactionTxToRaw(tx)
	r, e := c.rpc.SendRawTransaction(txx, allowHighFees)
	return r, e
}

func (c *RPCClient) GetBlock(hash coinharness.Hash) (*coinharness.MsgBlock, error) {
	block, err := c.rpc.GetBlock(hash.(*chainhash.Hash)) //*wire.MsgBlock
	if err != nil {
		return nil, err
	}

	b := &coinharness.MsgBlock{}

	ttx := []*coinharness.MessageTx{}
	for _, ti := range block.Transactions {
		ttx = append(ttx,
			TransactionRawToTx(ti),
		)
	}

	b.Transactions = ttx

	return b, nil
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

func (c *RPCClient) GetNewAddress(account string) (coinharness.Address, error) {
	legacy, err := c.rpc.GetNewAddress(account)
	if err != nil {
		return nil, err
	}

	result := &Address{Address: legacy}
	return result, nil
}

func (c *RPCClient) ValidateAddress(address coinharness.Address) (*coinharness.ValidateAddressResult, error) {
	legacy, err := c.rpc.ValidateAddress(address.Internal().(dcrutil.Address))
	// *dcrjson.ValidateAddressWalletResult
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

func (c *RPCClient) GetBalance() (*coinharness.GetBalanceResult, error) {
	legacy, err := c.rpc.GetBalance("*")
	// *dcrjson.ValidateAddressWalletResult
	if err != nil {
		return nil, err
	}
	result := &coinharness.GetBalanceResult{
		BlockHash: legacy.BlockHash,
		//TotalSpendable:   coin.FromFloat(legacy.TotalSpendable),
		//TotalUnconfirmed: coin.FromFloat(legacy.TotalUnconfirmed),
		//
		//CumulativeTotal:              coin.FromFloat(legacy.CumulativeTotal),
		//TotalVotingAuthority:         coin.FromFloat(legacy.TotalVotingAuthority),
		//TotalLockedByTickets:         coin.FromFloat(legacy.TotalLockedByTickets),
		//TotalImmatureStakeGeneration: coin.FromFloat(legacy.TotalImmatureStakeGeneration),
		//TotalImmatureCoinbaseRewards: coin.FromFloat(legacy.TotalImmatureCoinbaseRewards),
	}
	result.Balances = make(map[string]coinharness.GetAccountBalanceResult)
	for _, v := range legacy.Balances {
		x := coinharness.GetAccountBalanceResult{
			AccountName:             v.AccountName,
			Total:                   coin.FromFloat(v.Total),
			Spendable:               coin.FromFloat(v.Spendable),
			Unconfirmed:             coin.FromFloat(v.Unconfirmed),
			LockedByTickets:         coin.FromFloat(v.LockedByTickets),
			VotingAuthority:         coin.FromFloat(v.VotingAuthority),
			ImmatureCoinbaseRewards: coin.FromFloat(v.ImmatureCoinbaseRewards),
		}
		result.Balances[v.AccountName] = x
	}

	return result, nil
}

func (c *RPCClient) GetBestBlock() (coinharness.Hash, int64, error) {
	return c.rpc.GetBestBlock()
}

func (c *RPCClient) ListAccounts() (map[string]coin.Amount, error) {
	l, err := c.rpc.ListAccounts()
	if err != nil {
		return nil, err
	}

	r := make(map[string]coin.Amount)
	for k, v := range l {
		r[k] = coin.Amount{int64(v)}
	}
	return r, nil
}

func (c *RPCClient) CreateNewAccount(account string) error {
	return c.rpc.CreateNewAccount(account)
}

func (c *RPCClient) WalletLock() error {
	return c.rpc.WalletLock()
}

func (c *RPCClient) WalletInfo() (*coinharness.WalletInfoResult, error) {
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

func (c *RPCClient) WalletUnlock(passphrase string, timeoutSecs int64) error {
	return c.rpc.WalletPassphrase(passphrase, timeoutSecs)
}

func (c *RPCClient) GetBuildVersion() (coinharness.BuildVersion, error) {
	//legacy, err := c.rpc.GetBuildVersion()
	//if err != nil {
	//	return nil, err
	//}
	//return legacy, nil
	return nil, fmt.Errorf("decred does not support this feature (GetBuildVersion)")
}
