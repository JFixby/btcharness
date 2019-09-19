package walletcls

import (
	"github.com/jfixby/coinharness"
	"github.com/jfixby/coinharness/consolewallet"
	"github.com/jfixby/dcrharness"
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/commandline"
)

// ConsoleWalletFactory produces a new ConsoleWallet-instance upon request
type ConsoleWalletFactory struct {
	// WalletExecutablePathProvider returns path to the btcd executable
	WalletExecutablePathProvider commandline.ExecutablePathProvider
	ConsoleCommandCook           WalletConsoleCommandCook
	RPCClientFactory             dcrharness.DcrRPCClientFactory
}

// NewWallet creates and returns a fully initialized instance of the ConsoleWallet.
func (factory *ConsoleWalletFactory) NewWallet(config *coinharness.TestWalletConfig) coinharness.Wallet {
	pin.AssertNotNil("ActiveNet", config.ActiveNet)
	pin.AssertNotNil("WorkingDir", config.WorkingDir)
	pin.AssertNotEmpty("WorkingDir", config.WorkingDir)

	pin.AssertNotEmpty("NodeUser", config.NodeUser)
	pin.AssertNotEmpty("NodePassword", config.NodePassword)
	pin.AssertNotEmpty("WalletUser", config.WalletUser)
	pin.AssertNotEmpty("WalletPassword", config.WalletPassword)

	args := &consolewallet.NewConsoleWalletArgs{
		ClientFac:                    &factory.RPCClientFactory,
		ConsoleCommandCook:           &factory.ConsoleCommandCook,
		WalletExecutablePathProvider: factory.WalletExecutablePathProvider,
		WalletUser:                   config.WalletUser,
		WalletPass:                   config.WalletPassword,
		NodeUser:                     config.NodeUser,
		NodePass:                     config.NodePassword,
		AppDir:                       config.WorkingDir,
		NodeRPCHost:                  config.NodeRPCHost,
		NodeRPCPort:                  config.NodeRPCPort,
		WalletRPCHost:                config.WalletRPCHost,
		WalletRPCPort:                config.WalletRPCPort,
		ActiveNet:                    config.ActiveNet,
	}

	return consolewallet.NewConsoleWallet(args)
}

type WalletConsoleCommandCook struct {
}

// cookArguments prepares arguments for the command-line call
func (cook *WalletConsoleCommandCook) CookArguments(par *consolewallet.ConsoleCommandParams) map[string]interface{} {
	result := make(map[string]interface{})

	result["dcrdusername"] = par.NodeRpcUser
	result["dcrdpassword"] = par.NodeRpcPass
	result["username"] = par.WalletRpcUser
	result["password"] = par.WalletRpcPass
	result["rpcconnect"] = par.RpcConnect
	result["rpclisten"] = par.RpcListen
	result["appdata"] = par.AppDir
	result["debuglevel"] = par.DebugLevel
	result["cafile"] = par.NodeCertFile
	result["rpccert"] = par.CertFile
	result["rpckey"] = par.KeyFile
	result["nogrpc"] = commandline.NoArgumentValue

	result[dcrharness.NetworkFor(par.Network)] = commandline.NoArgumentValue

	commandline.ArgumentsCopyTo(par.ExtraArguments, result)
	return result
}
