package btcharness

import (
	"fmt"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/jfixby/coinharness"
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/commandline"
)

// networkFor resolves network argument for node and wallet console commands
func NetworkFor(net coinharness.Network) string {
	if net.Params() == &chaincfg.SimNetParams {
		return "simnet"
	}
	if net.Params() == &chaincfg.TestNet3Params {
		return "testnet"
	}
	if net.Params() == &chaincfg.RegressionNetParams {
		return "regtest"
	}
	if net.Params() == &chaincfg.MainNetParams {
		// no argument needed for the MainNet
		return commandline.NoArgument
	}

	// should never reach this line, report violation
	pin.ReportTestSetupMalfunction(fmt.Errorf("unknown network: %v ", net))
	return ""
}
