// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package deep

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/log"
	//	"github.com/ethereum/go-ethereum/p2p/discover"
)

const (
	protocolName           = "raft"
	protocolVersion uint64 = 0x01

	raftMsg = 0x00

	minterRole   = 1 // TODO WRONG: etcdRaft.LEADER
	verifierRole = 2 // TODO WRONG: etcdRaft.NOT_LEADER

	// Raft's ticker interval
	tickerMS = 100

	// We use a bounded channel of constant size buffering incoming messages
	msgChanSize = 1000

	// Snapshot after this many raft messages
	//
	// TODO: measure and get this as low as possible without affecting performance
	//
	snapshotPeriod = 250

	peerUrlKeyPrefix = "peerUrl-"

	chainExtensionMessage = "Successfully extended chain"
)

var (
	appliedDbKey = []byte("applied")
)

// TODO: this is just copied over from cmd/utils/cmd.go. dedupe

// Fatalf formats a message to standard error and exits the program.
// The message is also printed to standard output if standard error
// is redirected to a different file.
func fatalf(format string, args ...interface{}) {
	w := io.MultiWriter(os.Stdout, os.Stderr)
	if runtime.GOOS == "windows" {
		// The SameFile check below doesn't work on Windows.
		// stdout is unlikely to get redirected though, so just print there.
		w = os.Stdout
	} else {
		outf, _ := os.Stdout.Stat()
		errf, _ := os.Stderr.Stat()
		if outf != nil && errf != nil && os.SameFile(outf, errf) {
			w = os.Stderr
		}
	}
	fmt.Fprintf(w, "Fatal: "+format+"\n", args...)
	os.Exit(1)
}

var DoEmitCheckpoints = true

const (
	TxCreated          = "TX-CREATED"
	TxAccepted         = "TX-ACCEPTED"
	BecameMinter       = "BECAME-MINTER"
	BecameVerifier     = "BECAME-VERIFIER"
	BlockCreated       = "BLOCK-CREATED"
	BlockVotingStarted = "BLOCK-VOTING-STARTED"
)

func EmitCheckpoint(checkpointName string, logValues ...interface{}) {
	args := []interface{}{"name", checkpointName}
	args = append(args, logValues...)
	if DoEmitCheckpoints {
		log.Info("CHECKPOINT", args...)
	}
}

func Keccak256(data ...[]byte) []byte {
	hasher := sha3.NewKeccak256()
	for _, b := range data {
		hasher.Write(b)
	}
	return hasher.Sum(nil)
}

func UInt64ToByte(i uint64) (k []byte) {
	k = make([]byte, 8)
	binary.BigEndian.PutUint64(k, uint64(i))
	return k
}

func BytesToUint64(inp []byte) uint64 {
	return binary.BigEndian.Uint64(inp)
}

/*
func nodeHasRaftPort(n *discover.Node) bool {
	// TODO
	return false
}
func getNodeRaftPort(n *discover.Node) uint16 {
	// TODO
	return 999
}
*/
