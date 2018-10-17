// Copyright 2018 Wolk Inc.
// This file is part of the Wolk Deep Blockchains library.
package plasmachain

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/wolkdb/go-plasma/deep"
)

type PlasmaTxPool struct {
	txpool   []*Transaction
	wtxpool  []*deep.AnchorTransaction
	txFeed   event.Feed
	mintFeed event.Feed
	scope    event.SubscriptionScope
}

type PlasmaTxs []*Transaction
type AnchorTxs []*deep.AnchorTransaction
type AnchorByChainIDandBlockNum AnchorTxs

func (s AnchorByChainIDandBlockNum) Len() int      { return len(s) }
func (s AnchorByChainIDandBlockNum) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s AnchorByChainIDandBlockNum) Less(i, j int) bool {
	if s[i].BlockChainID < s[j].BlockChainID {
		return true
	}
	if s[i].BlockChainID > s[j].BlockChainID {
		return false
	}
	return s[i].BlockNumber < s[j].BlockNumber
}

type PlasmabyTokenIDandPrevBlock PlasmaTxs

func (s PlasmabyTokenIDandPrevBlock) Len() int      { return len(s) }
func (s PlasmabyTokenIDandPrevBlock) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s PlasmabyTokenIDandPrevBlock) Less(i, j int) bool {
	if s[i].TokenID < s[j].TokenID {
		return true
	}
	if s[i].TokenID > s[j].TokenID {
		return false
	}
	return s[i].PrevBlock < s[j].PrevBlock
}

func (self *PlasmaTxPool) GetTransactions() (txns deep.Transactions) {
	sort.Sort(PlasmabyTokenIDandPrevBlock(self.txpool))
	for _, plasmaTx := range self.txpool {
		txns = append(txns, plasmaTx)
	}
	sort.Sort(AnchorByChainIDandBlockNum(self.wtxpool))
	for _, anchorTx := range self.wtxpool {
		txns = append(txns, anchorTx)
	}
	return txns
}

// SubscribeTxPreEvent should return an event subscription of TxPreEvent and send events to the given channel.
// plasma-raft-new: func (self PlasmaTxPool) SubscribeTxPreEvent(ch chan<- deep.TxPreEvent) (sub event.Subscription) {
func (self *PlasmaTxPool) SubscribeTxPreEvent(ch chan<- deep.TxPreEvent) (sub event.Subscription) {
	return self.scope.Track(self.txFeed.Subscribe(ch))
}

// SubscribeTxPreEvent should return an event subscription of TxPreEvent and send events to the given channel.
func (self *PlasmaTxPool) Stop() {

}

func (self *PlasmaTxPool) Len() int {
	return len(self.txpool) + len(self.wtxpool)
}

func appendUnique(txpool []*deep.Transaction, rtx *deep.Transaction) []*deep.Transaction {
	txhash := (*rtx).Hash().Bytes()
	for _, tx := range txpool {
		if bytes.Compare((*tx).Hash().Bytes(), txhash) == 0 {
			return txpool
		}
	}
	return append(txpool, rtx)
}

func (self *PlasmaTxPool) addTransactionToPool(tx *Transaction) (err error) {
	txhash := tx.Hash().Bytes()
	for _, plasmaTx := range self.txpool {
		if bytes.Compare(plasmaTx.Hash().Bytes(), txhash) == 0 {
			log.Info("PLASMAAPI: SendPlasmaTransaction", "status", "alredy existed in txpool", "txhash", tx.Hash().Hex())
			return nil
		}
	}
	self.txpool = append(self.txpool, tx)
	return nil
}

func (self *PlasmaTxPool) RemoveTransaction(txhash common.Hash) (err error) {
	for i, PlasmaTx := range self.txpool {
		if bytes.Compare(PlasmaTx.Hash().Bytes(), txhash.Bytes()) == 0 {
			log.Info("RemoveTransaction", "txhash", txhash)
			self.txpool = append(self.txpool[:i], self.txpool[i+1:]...)
			return nil
		}
	}

	for k, AnchorTx := range self.wtxpool {
		if bytes.Compare(AnchorTx.Hash().Bytes(), txhash.Bytes()) == 0 {
			log.Info("RemoveTransaction: Anchor", "txhash", txhash)
			self.wtxpool = append(self.wtxpool[:k], self.wtxpool[k+1:]...)
			return nil
		}
	}
	return fmt.Errorf("tx not found")
}

func (self *PlasmaTxPool) addAnchorTransactionToPool(tx *deep.AnchorTransaction) (err error) {
	txhash := tx.Hash().Bytes()
	for _, anchorTX := range self.wtxpool {
		if bytes.Compare(anchorTX.Hash().Bytes(), txhash) == 0 {
			log.Info("PLASMAAPI: SendAnchorTransaction", "status", "alredy existed in txpool", "txhash", tx.Hash().Hex())
			//return fmt.Errorf("txn %s already exist", tx.Hash().Hex())
			return nil
		}
	}
	self.wtxpool = append(self.wtxpool, tx)
	return nil
}

func (self *PlasmaTxPool) SubscribeMintRequestEvent(ch chan<- deep.MintRequestEvent) (sub event.Subscription) {
	return self.scope.Track(self.mintFeed.Subscribe(ch))
}

func (self *PlasmaTxPool) MintTestBlock() {
	self.mintFeed.Send(deep.MintRequestEvent{})
}
