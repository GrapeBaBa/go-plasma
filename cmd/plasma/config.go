// Copyright 2017 Wolk Inc.
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"time"
	"unicode"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/naoina/toml"
	// "github.com/wolkdb/go-plasma/cloudstore"
	"github.com/wolkdb/go-plasma/cmd/utils"
	"github.com/wolkdb/go-plasma/deep"
	"github.com/wolkdb/go-plasma/plasmachain"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      utils.MigrateFlags(dumpConfig),
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(nodeFlags, rpcFlags...),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type plasmaConfig struct {
	Plasma plasmachain.Config
	Node   node.Config
}

func loadConfig(file string, cfg *plasmaConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit)
	cfg.HTTPModules = append(cfg.HTTPModules, "eth", "shh")
	cfg.WSModules = append(cfg.WSModules, "eth", "shh")
	cfg.IPCPath = "plasmachain.ipc"
	return cfg
}

func makeConfigNode(ctx *cli.Context) (*node.Node, plasmaConfig) {
	// Load defaults.
	cfg := plasmaConfig{
		Plasma: plasmachain.DefaultConfig,
		Node:   defaultNodeConfig(),
	}

	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}

	utils.SetPlasmaConfig(ctx, &cfg.Plasma)
	log.Info("utils.SetPlasmaConfig", "cfg", fmt.Sprintf("%v", cfg.Plasma))

	return stack, cfg
}

func makeFullNode(ctx *cli.Context) *node.Node {
	stack, cfg := makeConfigNode(ctx)
	ch_p := make(chan *plasmachain.PlasmaChain, 1)
	
	var err error
	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		deepchain, err := plasmachain.New(ctx, &(cfg.Plasma), true)
		if err != nil {
			fmt.Printf("ERR1 %v\n", err)
		}
		ch_p <- deepchain
		return deepchain, err
	})
	if err != nil {
		log.Info("plasmachain FAILURE")
		os.Exit(0)
	}
	log.Info("plasmachain SUCCESS")

        mintTimeMillis := ctx.GlobalInt(utils.MintTimeFlag.Name)
        mintTime := time.Duration(mintTimeMillis) * time.Millisecond

	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		deepchain := <-ch_p
		poaNode, err := deep.NewPOA(ctx, nil, deepchain, mintTime)
		if err != nil {
			fmt.Printf("ERR3 %v\n", err)
		}
		return poaNode, err
	})
	if err != nil {
		log.Error("Failed to register the POAService service: %v", err)
	}
	log.Info("POAService SUCCESS")

	return stack
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := ""

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}
	io.WriteString(os.Stdout, comment)
	os.Stdout.Write(out)
	return nil
}
