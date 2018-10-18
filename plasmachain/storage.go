// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.
package plasmachain

import (
	"bytes"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"

	"github.com/ethereum/go-ethereum/log"
	"github.com/wolkdb/go-plasma/deep"
	"github.com/wolkdb/go-plasma/smt"
)

type Storage struct {
	ChunkStore         deep.StorageLayer
	transactionStorage *smt.SparseMerkleTree
	accountStorage     *smt.SparseMerkleTree
	tokenStorage       *smt.SparseMerkleTree
	anchorStorage      *smt.SparseMerkleTree
	chainStorage       *smt.SparseMerkleTree
	//	tokenInfoStorage       *smt.SparseMerkleTree
}

func NewStorage(cs deep.StorageLayer) *Storage {
	var self Storage
	self.transactionStorage = smt.NewSparseMerkleTree(cs)
	self.accountStorage = smt.NewSparseMerkleTree(cs)
	self.tokenStorage = smt.NewSparseMerkleTree(cs)
	self.anchorStorage = smt.NewSparseMerkleTree(cs)
	self.chainStorage = smt.NewSparseMerkleTree(cs)
	self.ChunkStore = cs
	return &self
}

func (self *Storage) InitTrie(headerHash common.Hash) {
	encodedHeader, ok, hErr := self.ChunkStore.GetChunk(headerHash.Bytes())
	if hErr != nil {
		//TODO:
	} else if !ok {
		//TODO:
	}
	header := new(Header)
	if err := rlp.Decode(bytes.NewReader(encodedHeader), header); err != nil {
		log.Error("Invalid block header RLP", "v", headerHash.Hex(), "RLP", encodedHeader, "err", err)
	}

	self.tokenStorage.InitWithRoot(header.TokenRoot)
	self.accountStorage.InitWithRoot(header.AccountRoot)
	self.chainStorage.InitWithRoot(header.L3ChainRoot)
}

// InitWithBlock is called for past lookup. DO NOT apply state changes to it
func (self *Storage) InitWithBlock(b *Block) {
	self.transactionStorage.InitWithRoot(b.header.TransactionRoot)
	self.accountStorage.InitWithRoot(b.header.AccountRoot)
	self.tokenStorage.InitWithRoot(b.header.TokenRoot)
	self.anchorStorage.InitWithRoot(b.header.AnchorRoot)
	self.chainStorage.InitWithRoot(b.header.L3ChainRoot)
}

// InitWithHeader is called for past lookup. DO NOT apply state changes to it
func (self *Storage) InitWithHeader(h *Header) {
	self.transactionStorage.InitWithRoot(h.TransactionRoot)
	self.accountStorage.InitWithRoot(h.AccountRoot)
	self.tokenStorage.InitWithRoot(h.TokenRoot)
	self.anchorStorage.InitWithRoot(h.AnchorRoot)
	self.chainStorage.InitWithRoot(h.L3ChainRoot)
}

func (self *Storage) Flush() {
	self.transactionStorage.Flush()
	self.accountStorage.Flush()
	self.tokenStorage.Flush()
	self.anchorStorage.Flush()
	self.chainStorage.Flush()
}

// Called by commit
func (self *Storage) MakeHeaderHash(blockNumber uint64, parentHash, bloomID common.Hash) Header {
	var h Header
	h.BlockNumber = blockNumber
	h.ParentHash = parentHash
	h.Time = uint64(time.Now().Unix())
	h.TokenRoot = self.tokenStorage.MerkleRoot()
	h.TransactionRoot = self.transactionStorage.MerkleRoot()
	h.BloomID = bloomID
	h.AccountRoot = self.accountStorage.MerkleRoot()
	h.L3ChainRoot = self.chainStorage.MerkleRoot()
	h.AnchorRoot = self.anchorStorage.MerkleRoot()
	return h
}

//TODO: re-write deposit code
func (self *Storage) Deposit(tx *Transaction, denomination uint64, blockNumber uint64) (err error) {
	return nil
}

//TODO: re-write deposit code
func (self *Storage) TokenTransfer(tx *Transaction, newBlockNumber uint64) (err error) {
	return nil
}
