// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.
package plasmachain

import "github.com/ethereum/go-ethereum/node"

type Config struct {
	NetworkId        uint64 // Network ID to use for selecting peers to connect to
	DataDir          string
	Node             node.Config
	PlasmaAddr       string
	PlasmaPort       uint64
	CloudstoreAddr   string
	CloudstorePort   uint64
	RemoteDisabled   bool
	UseLayer1        bool
	RootContractAddr string
	operatorKey      string //TODO: Replace this keystore dir
	L1rpcEndpointUrl string
	L1wsEndpointUrl  string
}

// DefaultConfig contains default settings for use on the Ethereum main net.
var DefaultConfig = Config{
	NetworkId:        66,
	DataDir:          "/tmp/plasmachain",
	PlasmaAddr:       "",
	PlasmaPort:       80,
	CloudstoreAddr:   "",
	CloudstorePort:   9900,
	RemoteDisabled:   true,
	UseLayer1:        false,
	RootContractAddr: "0xa611dD32Bb2cC893bC57693bFA423c52658367Ca",
	operatorKey:      "6545ddd10c1e0d6693ba62dec711a2d2973124ae0374d822f845d322fb251645",
	L1rpcEndpointUrl: "http://localhost:8545",
	L1wsEndpointUrl:  "ws://localhost:8545",
}

// LocalTestConfig contains settings to test integration in local enviroments
var LocalTestConfig = Config{
	NetworkId:        66,
	DataDir:          "/tmp/plasmachain",
	PlasmaAddr:       "",
	PlasmaPort:       80,
	CloudstoreAddr:   "",
	CloudstorePort:   9900,
	RemoteDisabled:   true,
	UseLayer1:        false,
	RootContractAddr: "0xa611dD32Bb2cC893bC57693bFA423c52658367Ca",
	operatorKey:      "6545ddd10c1e0d6693ba62dec711a2d2973124ae0374d822f845d322fb251645",
	L1rpcEndpointUrl: "http://localhost:8545",
	L1wsEndpointUrl:  "ws://localhost:8545",
}

func (c *Config) GetPlasmaAddr() string     { return c.PlasmaAddr }
func (c *Config) GetPlasmaPort() uint64     { return c.PlasmaPort }
func (c *Config) GetCloudstoreAddr() string { return c.CloudstoreAddr }
func (c *Config) GetCloudstorePort() uint64 { return c.CloudstorePort }
func (c *Config) GetDataDir() string        { return c.DataDir }
func (c *Config) IsLocalMode() bool         { return c.RemoteDisabled }

/* Network Options

Local (Ganache)
L1rpcEndpointUrl: "http://localhost:8545"
L1wsEndpointUrl:  "ws://localhost:8545"

Local (Geth)
L1rpcEndpointUrl: "http://localhost:8545"
L1wsEndpointUrl:  "ws://localhost:8546"

Remote (Rinkeby)
L1rpcEndpointUrl: "https://rinkeby.infura.io/metamask"
L1wsEndpointUrl:  "wss://rinkeby.infura.io/ws"

Remote (Ropsten)
L1rpcEndpointUrl: "https://ropsten.infura.io/metamask"
L1wsEndpointUrl:  "wss://ropsten.infura.io/ws"

Remote (MainNet)
L1rpcEndpointUrl: "https://mainnet.infura.io/metamask"
L1wsEndpointUrl:  "wss://mainnet.infura.io/ws"
*/
