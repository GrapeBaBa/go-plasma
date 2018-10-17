// Copyright 2018 Wolk Inc.
// This file is part of the Wolk Deep Blockchains library.
package smt

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/wolkdb/go-plasma/deep"
)

type SparseMerkleTree struct {
	ChunkStore    deep.StorageLayer
	root          *Node
	DefaultHashes [TreeDepth][]byte
}

func NewSparseMerkleTree(cs deep.StorageLayer) *SparseMerkleTree {
	var self SparseMerkleTree
	self.root = NewNode(TreeDepth-1, nil)
	self.ChunkStore = cs
	self.DefaultHashes = ComputeDefaultHashes()
	return &self
}

func (self *SparseMerkleTree) Init(hash common.Hash) {
	self.root.SetHash(hash.Bytes())
}

//merkleroot <=> chunkHash mapping is only stored at the highest SMT level
func (self *SparseMerkleTree) InitWithRoot(merkleroot common.Hash) bool {
	chunkHash, ok, err := self.ChunkStore.GetChunk(merkleroot.Bytes())
	if err != nil {
		return false
	} else if !ok || len(chunkHash) == 0 { //TODO: How are we determining empty val?
		return false
	}
	self.root.SetHash(chunkHash)
	return true
}

func (self *SparseMerkleTree) Copy() (t *SparseMerkleTree) {
	// SMTTODO
	return t
}

func (self *SparseMerkleTree) Flush() common.Hash {
	self.root.computeMerkleRoot(self.ChunkStore, self.DefaultHashes)
	self.root.flush(self.ChunkStore)
	self.root.flushRoot(self.ChunkStore)
	return common.BytesToHash(self.root.chunkHash)
}

func (self *SparseMerkleTree) Delete(k []byte) error {
	self.root.delete(k, 0)
	return nil
}

func (self *SparseMerkleTree) GenerateProof(k []byte, v []byte) (p *Proof) {
	var pr Proof
	pr.key = k
	self.root.generateProof(self.ChunkStore, k, v, 0, self.DefaultHashes, &pr)
	return &pr
}

func (self *SparseMerkleTree) TryGet(key []byte) (b []byte, err error) {
	v0, _, _, _, _, err := self.Get(key)
	if err != nil {
		return b, err
	}
	return v0, nil
}

func (self *SparseMerkleTree) TryUpdate(key, value []byte) error {
	return self.Insert(key, value, 0, 0)
}
func (self *SparseMerkleTree) TryDelete(key []byte) error {
	return self.Delete(key)
}
func (self *SparseMerkleTree) GetKey(key []byte) []byte {
	v0, _, _, _, _, _ := self.Get(key)
	return v0
}

func (self *SparseMerkleTree) Hash() common.Hash {
	return self.ChunkHash()
}

func (self *SparseMerkleTree) Get(k []byte) (v0 []byte, found bool, p *Proof, storageBytes uint64, prevBlock uint64, err error) {
	v0, found, storageBytes, prevBlock, err = self.root.get(self.ChunkStore, k, 0)
	if found {
		var pr Proof
		pr.key = k
		ok := self.root.generateProof(self.ChunkStore, k, v0, 0, self.DefaultHashes, &pr)
		if !ok {
			return v0, found, &pr, storageBytes, prevBlock, fmt.Errorf("NO proof")
		}
		return v0, found, &pr, storageBytes, prevBlock, nil
	}
	return v0, found, nil, storageBytes, prevBlock, nil
}

func (self *SparseMerkleTree) ChunkHash() common.Hash {
	return common.BytesToHash(self.root.chunkHash)
}

func (self *SparseMerkleTree) MerkleRoot() common.Hash {
	return common.BytesToHash(self.root.merkleRoot)
}

func (self *SparseMerkleTree) Insert(k []byte, v []byte, storageBytesNew uint64, blockNum uint64) error {
	self.root.insert(self.ChunkStore, k, v, 0, storageBytesNew, blockNum)
	return nil
}

func (self *SparseMerkleTree) Dump() {
	self.root.dump(nil)
}
