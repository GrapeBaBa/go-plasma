// Copyright 2018 Wolk Inc.
// This file is part of the Wolk go-plasma library.
package smt_test

import (
	"bytes"
	"fmt"
	"testing"
	"time"
	
	"github.com/ethereum/go-ethereum/common"
	"github.com/wolkdb/go-plasma/plasmachain"
	"github.com/wolkdb/go-plasma/smt"
)

func TestSequence(t *testing.T) {

	// setup plasma store
	pcs, err := plasmachain.NewPlasmaChunkstore(plasmachain.DefaultChunkstorePath)
	if err != nil {
		t.Fatalf("[smt_test:NewCloudstore]%v", err)
	}
	defer pcs.Close()

	smt0 := smt.NewSparseMerkleTree(pcs)
	nkeys := uint64(2)
	kv := make(map[uint64]common.Hash)
	for i := uint64(0); i < nkeys; i++ {

		k := smt.UIntToByte(i)
		v := smt.Keccak256([]byte(fmt.Sprintf("value%d", i)))
		kv[i] = common.BytesToHash(v)
		err = smt0.Insert(k, v, 0, 0)
		if err != nil {
			t.Fatalf("SetKey: %v\n", err)
		}
	}
	smt0.Flush()
	smt0.Dump()
	chunkHash := smt0.ChunkHash()
	merkleRoot := smt0.MerkleRoot()
	fmt.Printf("Generated:  Hash: %x Merkle Root: %x\n", chunkHash, merkleRoot)
	passes := 0
	smt0 = smt.NewSparseMerkleTree(pcs)
	smt0.Init(chunkHash)
	for i := uint64(0); i < nkeys; i++ {
		k := i
		v1, found, proof, storageBytes, prevBlock, err := smt0.Get(smt.UIntToByte(k))
		smt0.Flush()
		// smt0.Dump()
		if err != nil {
			fmt.Printf("err not found %x %v \n", k, err)
		} else if found {
			if bytes.Compare(kv[k].Bytes(), v1) == 0 {
				checkproof := proof.Check(v1, merkleRoot.Bytes(), smt0.DefaultHashes, false)
				if checkproof {
					passes++
				} else {
					fmt.Printf("k:%x v:%x storageBytes:%d prevBlock: %d ", k, v1, storageBytes, prevBlock)
					t.Fatalf("CHECK PROOF ==> FAILURE\n")
				}
			} else {
				t.Fatalf("k:%x v:%x sb:%d kv[k]:%x INCORRECT\n", k, v1, storageBytes, kv[k])
			}
		} else {
			fmt.Printf("k:%x not found \n", k)
		}
	}
	fmt.Printf("%d/%d keys PASSED\n", passes, nkeys)

}

func TestSMT(t *testing.T) {

	// setup plasma store
	pcs, err := plasmachain.NewPlasmaChunkstore(plasmachain.DefaultChunkstorePath)
	if err != nil {
		t.Fatalf("[smt_test:NewCloudstore]%v", err)
	}
	defer pcs.Close()

	smt0 := smt.NewSparseMerkleTree(pcs)
	nkeys := uint64(500)
	nversions := uint64(3)
	chunkHash := make(map[uint64]common.Hash)
	merkleRoot := make(map[uint64]common.Hash)
	kv := make(map[uint64]map[uint64]common.Hash)
	for ver := uint64(0); ver < nversions; ver++ {
		kv[ver] = make(map[uint64]common.Hash)
		for i := uint64(0); i < nkeys; i++ {
			storageBytesNew := uint64(3)
			k := smt.Bytes32ToUint64(smt.Keccak256(smt.Uint64ToBytes32(i % 10000)))
			v := smt.Keccak256([]byte(fmt.Sprintf("%d%d", i, ver)))
			// fmt.Printf("%x version %d == %x\n", k, ver, v)
			kv[ver][k] = common.BytesToHash(v)
			prevBlock := ver
			err = smt0.Insert(smt.UIntToByte(k), v, storageBytesNew, prevBlock)
			if err != nil {
				t.Fatalf("SetKey: %v\n", err)
			}
		}
		smt0.Flush()
		// smt0.Dump()
		chunkHash[ver] = smt0.ChunkHash()
		merkleRoot[ver] = smt0.MerkleRoot()
		fmt.Printf("Generated: Version %d Hash: %x Merkle Root: %x\n", ver, chunkHash[ver], merkleRoot[ver])
	}

	fmt.Println("----------------------------------------------------------------------------------------sleep 20 Second")
	time.Sleep(20 * time.Second)

	for ver := uint64(0); ver < nversions; ver++ {
		smt0 = smt.NewSparseMerkleTree(pcs)
		smt0.Init(chunkHash[ver])
		passes := 0
		for i := uint64(0); i < nkeys; i++ {
			k := smt.Bytes32ToUint64(smt.Keccak256(smt.Uint64ToBytes32(i % 10000)))
			v1, found, proof, storageBytes, prevBlock, err := smt0.Get(smt.UIntToByte(k))
			// smt0.Flush()
			// smt0.Dump()
			if err != nil {
				fmt.Printf("err not found %x %v \n", k, err)
			} else if found {
				if bytes.Compare(kv[ver][k].Bytes(), v1) == 0 {
					checkproof := proof.Check(v1, merkleRoot[ver].Bytes(), smt0.DefaultHashes, false)
					if checkproof {
						passes++
					} else {
						fmt.Printf("k:%x v:%x storageBytes:%d prevBlock: %d ver %d -- ", k, v1, storageBytes, prevBlock, ver)
						t.Fatalf("CHECK PROOF ==> FAILURE\n")
					}

				} else {
					t.Fatalf("k:%x v:%x sb:%d kv[k]:%x INCORRECT\n", k, v1, storageBytes, kv[k])
				}
			} else {
				fmt.Printf("k:%x not found \n", k)
			}
		}
		fmt.Printf("Version %d  -- %d/%d keys PASSED\n", ver, passes, nkeys)
	}
}
