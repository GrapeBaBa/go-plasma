// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.

package plasmachain

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wolkdb/go-plasma/deep"
	"github.com/wolkdb/go-plasma/smt"
)

// Backend interface provides the common API services (that are provided by
// both full and light clients) with access to necessary functions.
type Backend interface {
	// Plasma Chain
	GetPlasmaBlock(blockNumber rpc.BlockNumber) (block *Block)
	GetPlasmaBalance(addr common.Address, blockNumber rpc.BlockNumber) (a *Account, err error)
	GetPlasmaToken(tokenID hexutil.Uint64, blockNumber rpc.BlockNumber) (t *Token, tinfo *TokenInfo, err error)
	LatestBlockNumber() (uint64, error)

	// Plasma Transactions
	GetPlasmaTransactionPool() (txs map[common.Address][]*Transaction)
	SendPlasmaTransaction(tx *Transaction) (common.Hash, error)
	GetPlasmaTransactionReceipt(hash common.Hash) (tx *Transaction, blockNumber uint64, receipt uint8, err error)
	GetPlasmaTransactionProof(hash common.Hash) (tokenID uint64, txbyte []byte, proof *smt.Proof, blockNumber uint64, err error)
	GetPlasmaBloomFilter(hash common.Hash) ([]byte, error)
	getProofbyTokenID(tokenIndex uint64, bn uint64) (tokenID uint64, txbyte []byte, proof *smt.Proof, blk uint64, prevBlk uint64, err error)

	// Anchor Transactions
	GetAnchorTransactionPool() (txs map[uint64][]*deep.AnchorTransaction)
	SendAnchorTransaction(tx *deep.AnchorTransaction) (common.Hash, error)

	processDeposit(tokenId []byte, t *TokenInfo) error //replaced by eventhandler
	startExit(exiter common.Address, denomination uint64, depositIndex uint64, tokenId uint64, ts uint64) error
	publishedBlock(_rootHash common.Hash, _currentDepositIndex uint64, _blockNumber uint64) error
	finalizedExit(exiter common.Address, denomination uint64, depositIndex uint64, tokenId uint64, ts uint64) error
	challenge(challenger common.Address, tokenId uint64, ts uint64) error
}

type PlasmaApiBackend struct {
	plasma *PlasmaChain
}

func (b *PlasmaApiBackend) SendPlasmaTransaction(tx *Transaction) (h common.Hash, err error) {
	return b.plasma.SendPlasmaTransaction(tx)
}

func (b *PlasmaApiBackend) GetPlasmaBalance(addr common.Address, blockNumber rpc.BlockNumber) (a *Account, err error) {
	return b.plasma.GetPlasmaBalance(addr, blockNumber)
}

func (b *PlasmaApiBackend) GetPlasmaToken(tokenID hexutil.Uint64, blockNumber rpc.BlockNumber) (t *Token, tinfo *TokenInfo, err error) {
	return b.plasma.GetPlasmaToken(tokenID, blockNumber)
}

func (b *PlasmaApiBackend) GetPlasmaBlock(blockNumber rpc.BlockNumber) (block *Block) {
	return b.plasma.GetPlasmaBlock(blockNumber)
}

func (b *PlasmaApiBackend) GetPlasmaBloomFilter(hash common.Hash) ([]byte, error) {
	return b.plasma.GetPlasmaBloomFilter(hash)
}

func (b *PlasmaApiBackend) GetPlasmaTransactionReceipt(hash common.Hash) (tx *Transaction, blockNumber uint64, receipt uint8, err error) {
	return b.plasma.GetPlasmaTransactionReceipt(hash)
}

func (b *PlasmaApiBackend) GetPlasmaTransactionProof(hash common.Hash) (tokenID uint64, txbyte []byte, proof *smt.Proof, blockNumber uint64, err error) {
	return b.plasma.GetPlasmaTransactionProof(hash)
}

func (b *PlasmaApiBackend) GetPlasmaTransactionPool() (txs map[common.Address][]*Transaction) {
	return b.plasma.GetPlasmaTransactionPool()
}

func (b *PlasmaApiBackend) SubscribePlasmaTxPreEvent(ch chan<- PlasmaTxPreEvent) event.Subscription {
	return b.plasma.protocolManager.SubscribePlasmaTxPreEvent(ch)
}

func (b *PlasmaApiBackend) SendAnchorTransaction(tx *deep.AnchorTransaction) (txhash common.Hash, err error) {
	return b.plasma.SendAnchorTransaction(tx)
}

func (b *PlasmaApiBackend) GetAnchorTransactionPool() (txs map[uint64][]*deep.AnchorTransaction) {
	return b.plasma.GetAnchorTransactionPool()
}

func (b *PlasmaApiBackend) LatestBlockNumber() (blockNumber uint64, err error) {
	return b.plasma.LatestBlockNumber()
}
