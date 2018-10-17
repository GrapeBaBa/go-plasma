// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package plasmachain

import "github.com/ethereum/go-ethereum/node"

type Config struct {
	NetworkId      uint64 // Network ID to use for selecting peers to connect to
	DataDir        string
	Node           node.Config
	PlasmaAddr     string
	PlasmaPort     uint64
	CloudstoreAddr string
	CloudstorePort uint64
	RemoteDisabled bool
}

// DefaultConfig contains default settings for use on the Ethereum main net.
var DefaultConfig = Config{
	NetworkId:      66,
	DataDir:        "/tmp/plasmachain",
	PlasmaAddr:     "",
	PlasmaPort:     80,
	CloudstoreAddr: "",
	CloudstorePort: 9900,
	RemoteDisabled: true,
}

// LocalTestConfig contains settings to test integration in local enviroments
var LocalTestConfig = Config{
	NetworkId:      66,
	DataDir:        "/tmp/plasmachain",
	PlasmaAddr:     "",
	PlasmaPort:     80,
	CloudstoreAddr: "",
	CloudstorePort: 9900,
	RemoteDisabled: true,
}

func (c *Config) GetPlasmaAddr() string     { return c.PlasmaAddr }
func (c *Config) GetPlasmaPort() uint64     { return c.PlasmaPort }
func (c *Config) GetCloudstoreAddr() string { return c.CloudstoreAddr }
func (c *Config) GetCloudstorePort() uint64 { return c.CloudstorePort }
func (c *Config) GetDataDir() string        { return c.DataDir }
func (c *Config) IsLocalMode() bool         { return c.RemoteDisabled }
