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
	"unicode"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/naoina/toml"
	"github.com/wolkdb/go-plasma/cloudstore"
	"github.com/wolkdb/go-plasma/cmd/utils"
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

type cloudstoreConfig struct {
	Cloudstore cloudstore.Config
	Node       node.Config
}

func loadConfig(file string, cfg *cloudstoreConfig) error {
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
	cfg.IPCPath = "cloudstore.ipc"
	return cfg
}

func makeConfigNode(ctx *cli.Context) (*node.Node, cloudstoreConfig) {
	// Load defaults.
	cfg := cloudstoreConfig{
		Cloudstore: cloudstore.DefaultConfig,
		Node:       defaultNodeConfig(),
	}

	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	log.Debug(fmt.Sprintf("static node %v", cfg.Node.StaticNodes()))
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}

	utils.SetCloudstoreConfig(ctx, &cfg.Cloudstore)
	log.Info("utils.SetCloudstoreConfig", "cfg", fmt.Sprintf("%v", cfg.Cloudstore))

	return stack, cfg
}

func makeFullNode(ctx *cli.Context) *node.Node {
	//stack, cfg := makeConfigNode(ctx)
	stack, _ := makeConfigNode(ctx)
	log.Info("calling RegisterCloudstoreService")

	//cfg0 := &cfg.Cloudstore
	ch := make(chan *cloudstore.Cloudstore, 1)
	var err error
	err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		//deepchain, err := cloudstore.New(ctx, cfg0)
		//TODO: Add back config and make it so we're configuring RemoteStorage ---
		deepchain, err := cloudstore.New(ctx, true)
		log.Info("sending cloudstore channel")
		ch <- deepchain
		log.Info("sent cloudstore channel")
		return deepchain, err
	})
	if err != nil {
		os.Exit(0)
		log.Error("Failed to register the Cloudstore Chain")
	}
	log.Info(fmt.Sprintf("[flags:Cloudstore] Cloudstore SUCCESS %v", ch))
	/*
		err = stack.Register(func(ctx *node.ServiceContext) (n node.Service, e error) {
			//TODO: figure out what to do here
			deepchain := <-ch
			poaNode, err := deep.NewPOA(ctx, nil, deepchain.Chain())
			if err != nil {
				fmt.Printf("ERR3 %v\n", err)
			}
			return poaNode, err

			log.Info("registering cloudstore")
			return n, e
		})
		if err != nil {
			log.Error("Failed to register the NoSQL service: %v", err)
		}
	*/
	return stack
}

/*
func makeFullNodeRAFT(ctx *cli.Context) *node.Node {
	stack, cfg := makeConfigNode(ctx)
	log.Info("calling RegisterCloudstoreService")
	fCh := utils.RegisterCloudstoreService(stack, &cfg.Cloudstore)
	log.Info(fmt.Sprintf("calling RegisterRaftService %v", fCh))
	utils.RegisterRaftService(stack, ctx, &cfg.Node, fCh)
	log.Info("finish RegisterRaftService")
	return stack
}
*/

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
