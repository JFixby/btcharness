// Copyright (c) 2018 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package nodecls

import (
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/jfixby/btcharness"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/coinharness/consolenode"
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/commandline"
	"path/filepath"
)

// ConsoleNodeFactory produces a new ConsoleNode-instance upon request
type ConsoleNodeFactory struct {
	// NodeExecutablePathProvider returns path to the btcd executable
	NodeExecutablePathProvider commandline.ExecutablePathProvider
	ConsoleCommandCook         BtcdConsoleCommandCook
	RPCClientFactory           btcharness.BtcRPCClientFactory
}

// NewNode creates and returns a fully initialized instance of the ConsoleNode.
func (factory *ConsoleNodeFactory) NewNode(config *coinharness.TestNodeConfig) coinharness.Node {
	pin.AssertNotNil("WorkingDir", config.WorkingDir)
	pin.AssertNotEmpty("WorkingDir", config.WorkingDir)

	args := &consolenode.NewConsoleNodeArgs{
		ClientFac:                  &factory.RPCClientFactory,
		ConsoleCommandCook:         &factory.ConsoleCommandCook,
		NodeExecutablePathProvider: factory.NodeExecutablePathProvider,
		RpcUser:                    "user",
		RpcPass:                    "pass",
		AppDir:                     config.WorkingDir,
		P2PHost:                    config.P2PHost,
		P2PPort:                    config.P2PPort,
		NodeRPCHost:                config.NodeRPCHost,
		NodeRPCPort:                config.NodeRPCPort,
		ActiveNet:                  config.ActiveNet,
	}

	return consolenode.NewConsoleNode(args)
}

type BtcdConsoleCommandCook struct {
}

// cookArguments prepares arguments for the command-line call
func (cook *BtcdConsoleCommandCook) CookArguments(par *consolenode.ConsoleCommandParams) map[string]interface{} {
	result := make(map[string]interface{})

	result["txindex"] = commandline.NoArgumentValue
	result["addrindex"] = commandline.NoArgumentValue
	result["rpcuser"] = par.RpcUser
	result["rpcpass"] = par.RpcPass
	result["rpcconnect"] = par.RpcConnect
	result["rpclisten"] = par.RpcListen
	result["listen"] = par.P2pAddress
	result["datadir"] = par.AppDir
	result["debuglevel"] = par.DebugLevel
	result["profile"] = par.Profile
	result["rpccert"] = par.CertFile
	result["rpckey"] = par.KeyFile
	if par.MiningAddress != nil {
		result["miningaddr"] = par.MiningAddress.String()
	}
	result[networkFor(par.Network)] = commandline.NoArgumentValue

	commandline.ArgumentsCopyTo(par.ExtraArguments, result)
	return result
}

// networkFor resolves network argument for node and wallet console commands
func networkFor(net coinharness.Network) string {
	if net == &chaincfg.SimNetParams {
		return "simnet"
	}
	if net == &chaincfg.TestNet3Params {
		return "testnet"
	}
	if net == &chaincfg.RegressionNetParams {
		return "regtest"
	}
	if net == &chaincfg.MainNetParams {
		// no argument needed for the MainNet
		return commandline.NoArgument
	}

	// should never reach this line, report violation
	pin.ReportTestSetupMalfunction(fmt.Errorf("unknown network: %v ", net))
	return ""
}
