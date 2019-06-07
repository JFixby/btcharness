// Copyright (c) 2018 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcharness

import (
	"fmt"
	"github.com/decred/btcd/rpcclient"
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

	return rpcclient.New(cfg, h)
}
