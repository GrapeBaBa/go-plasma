// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package deep

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

type minterPOA struct {
	head            uint64
	state           StateDB
	config          *params.ChainConfig
	mu              sync.Mutex
	mux             *event.TypeMux
	deepchain       Backend
	chain           BlockChain
	remoteStorage   StorageLayer
	minting         int32 // Atomic status counter
	mineflag        int32 // Atomic status counter
	blockTime       time.Duration
	txPreChan       chan TxPreEvent
	txPreSub        event.Subscription
	mintRequestChan chan MintRequestEvent
	mintRequestSub  event.Subscription
	mintTime        time.Duration
}

func NewMinterPOAStruct(config *params.ChainConfig, deepchain *CliqueService, mintTime time.Duration) *minterPOA {
	return &minterPOA{
		config:          config,
		deepchain:       deepchain,
		mux:             deepchain.EventMux(),
		chain:           deepchain.BlockChain(),
		blockTime:       1 * time.Second,
		head:            0,
		txPreChan:       make(chan TxPreEvent, 4096),
		mintRequestChan: make(chan MintRequestEvent, 1),
		mintTime:        mintTime,
	}
}

func newMinterPOA(config *params.ChainConfig, deepchain *CliqueService, mintTime time.Duration) *minterPOA {
	minter := NewMinterPOAStruct(config, deepchain, mintTime)
	minter.txPreSub = deepchain.TxPool().SubscribeTxPreEvent(minter.txPreChan)
	minter.mintRequestSub = deepchain.TxPool().SubscribeMintRequestEvent(minter.mintRequestChan)
	//minter.cloudstore = minter.chain.GetCloudstore()
	go minter.mintingLoop()
	return minter
}

func (minter *minterPOA) start() {
	atomic.StoreInt32(&minter.minting, 1)
	atomic.StoreInt32(&minter.mineflag, 0)
}

func (minter *minterPOA) stop() {
	minter.mu.Lock()
	defer minter.mu.Unlock()

	atomic.StoreInt32(&minter.minting, 0)
}

func (minter *minterPOA) mintingLoop() {
	var ticker *time.Ticker
	if minter.mintTime == 0 {
		ticker = time.NewTicker(2 * time.Second)
		ticker.Stop()
	} else {
		ticker = time.NewTicker(minter.mintTime)
	}
	defer ticker.Stop()
	defer minter.txPreSub.Unsubscribe()
	for {
		select {
		case <-ticker.C:
			if atomic.LoadInt32(&minter.mineflag) == 0 {
				minter.mintNewBlock()
			}

		case <-minter.mintRequestChan:
			if atomic.LoadInt32(&minter.mineflag) == 0 {
				minter.mintNewBlock()
			}

		case <-minter.txPreSub.Err():
			return
		}
	}
}

func (minter *minterPOA) makeBlock(parentHash common.Hash, header Header, n uint64, txs Transactions) (b Block, err error) {
	return minter.deepchain.BlockChain().MakeBlock(parentHash, header, n, txs)
}

func (minter *minterPOA) getTransactions() (tr Transactions) {
	alltxns := minter.deepchain.TxPool().GetTransactions()
	return alltxns

}

func (minter *minterPOA) GetStorageLayer() StorageLayer {
	return minter.remoteStorage
}

func (minter *minterPOA) MintNewBlock() error {
	return minter.mintNewBlock()
}

func (minter *minterPOA) mintNewBlock() error {
	//log.Debug(fmt.Sprintf("minter.go:mintNewBlock - minter.chain is [%+v]", minter.chain))
	//TODO: Add Do Not Disturb concept from minter
	atomic.StoreInt32(&minter.mineflag, 1)
	defer atomic.StoreInt32(&minter.mineflag, 0)

	minter.mu.Lock()
	defer minter.mu.Unlock()

	allAddrTxes := minter.getTransactions()
	if len(allAddrTxes) == 0 {
		//log.Info("Not minting a new block since there are no pending transactions")
		return nil
	}

	//log.Info("Minter details", "minter", minter)
	parent := minter.chain.CurrentBlock()
	log.Info("Parent is", "parent", parent)
	parentHash := parent.Hash()
	root := parent.Root()
	minter.head = minter.chain.CurrentBlock().Number()
	//log.Info(fmt.Sprintf("Got Block for Number (minter.head)=[%d]", minter.head))
	log.Info("HEAD vals ", "head", minter.head, "parentHash", parentHash, "root", root)

	log.Info("StateAt", "root", root)
	state, err := minter.chain.StateAt(root)
	if err != nil {
		//panic(fmt.Sprint("[deep:poa:mintNewBlock] failed to get parent state: ", err))
		return fmt.Errorf("[deep:poa:mintNewBlock] %s", err)
	}

	log.Info("setting state of minter to", "state", state)
	minter.state = state
	committedTxes := minter.commitTransactions(allAddrTxes, minter.chain)
	committedTxesCount := len(committedTxes)

	if committedTxesCount != len(allAddrTxes) {
		log.Info("[deep:poa:mintNewBlock] not all txns submitted were committed")
		//No error returned, just log as fyi
	}

	if committedTxesCount == 0 {
		//log.Info("[deep:poa:mintNewBlock] Not minting a new block since there are no pending transactions")
		return nil //No Error, just return b/c there are no
	}
	if minter.remoteStorage == nil {
		minter.remoteStorage = minter.chain.GetStorageLayer()
	}
	header, err := state.Commit(minter.remoteStorage, minter.head+1, parentHash)
	if err != nil {
		return fmt.Errorf("[deep:poa:mintNewBlock]state.Commit %s", err)
	}

	block, err := minter.makeBlock(parentHash, header, minter.head+1, committedTxes)
	if err != nil {
		return fmt.Errorf("[deep:poa:mintNewBlock] minter.makeBlock %s", err)
	}
	//	log.Info("Generated makeBlock", "header", header, "block num", block.Number(), "num txes", txCount)

	log.Info("Flushing Chunks ")
	err = minter.remoteStorage.Flush()
	if err != nil {
		log.Error("[deep:poa:mintNewBlock] Error flushing chunks associated with attempted block", "error", err)
		return fmt.Errorf("[deep:poa:mintNewBlock] Error flushing chunks associated with attempted block %x, %s", block.Hash(), err)
	} else {
		fmt.Printf("[deep:poa:mintNewBlock] flushed chunks %x\n", block.Hash())
	}

	err = minter.chain.WriteBlock(block)
	if err != nil {
		log.Info("[deep:poa:mintNewBlock] Error Writing Block")
		return fmt.Errorf("[deep:poa:mintNewBlock] %s", err)
	}

	//TODO: review moving this out and have it triggered by an event
	_, err = minter.chain.InsertChainForPOA(Blocks{block})
	if err != nil {
		return fmt.Errorf("[deep:poa:mintNewBlock] %s", err)
	}

	minter.head = minter.chain.CurrentBlock().Number()
	for _, tx := range committedTxes {
		err := minter.deepchain.TxPool().RemoveTransaction(tx.Hash())
		if err != nil {
			log.Info(fmt.Sprintf("[deep:poa:mintNewBlock] RemoveTransaction failed at mintNewBlock on hash: %x", tx.Hash()))
			return fmt.Errorf("[deep:poa:mintNewBlock] %s", err)
		}
	}

	if block != nil {
		log.Info("🔨  [deep:poa:mintNewBlock] Mined block", "number", block.Number(), "hash", fmt.Sprintf("%x", block.Hash().Bytes()[:4]), "head", minter.head)
	} else {
		fmt.Printf("[deep:poa:mintNewBlock] did NOT mine block %d\n", block.Number())
	}

	host, _ := os.Hostname()
	log.Info(fmt.Sprintf("[minter.go:mintNewBlock] HOST: %s ChainType %s ChainID: %d", host, minter.chain.ChainType(), minter.chain.GetBlockChainID()))

	return nil

}

// current implementation drops an uncommitted transaction and goes on trying to commit the next ones in the txes pool.
func (env *minterPOA) commitTransactions(txes Transactions, bc BlockChain) Transactions {
	var committedTxes Transactions

	numtxns := len(txes)
	for i := 0; i < numtxns; i++ {
		tx := txes[i]
		err := env.commitTransaction(tx, bc)
		switch {
		case err != nil:
			log.Info("TX failed, mark for removal", "hash", tx.Hash(), "err", err)
			fmt.Printf("[deep:poa:commitTransactions] err when committing this txn: %s\n", err)
			env.deepchain.TxPool().RemoveTransaction(tx.Hash())
			i--       //re-do index i, b/c previous index i was removed.
			numtxns-- //update number of loops
			//QUESTION: Should we skip txes for the the rest of the pool?
		default:
			committedTxes = append(committedTxes, tx)
		}
	}
	return committedTxes
}

func (env *minterPOA) commitTransaction(tx Transaction, bc BlockChain) error {
	snapshot := env.state.Snapshot()

	err := bc.ApplyTransaction(env.state, tx)
	if err != nil { //transaction failed
		log.Info("Error Applying Transaction: ", "error", err)
		reverterr := env.state.RevertToSnapshot(snapshot) //this is in the wrong place, probably.
		if reverterr != nil {                             //revert failed
			return fmt.Errorf("[deep:poa:commitTransaction] %s, %s", err, reverterr)
		}
		return fmt.Errorf("[deep:poa:commitTransaction] %s", err)
	}
	return nil
}

type CliqueService struct {
	blockchain    BlockChain
	remoteStorage StorageLayer
	txMu          sync.Mutex
	eventMux      *event.TypeMux
	minter        *minterPOA
}

func NewCliqueService(_eventMux *event.TypeMux, _blockchain BlockChain) *CliqueService {
	return &CliqueService{
		eventMux:   _eventMux,
		blockchain: _blockchain,
	}
}

func NewPOA(ctx *node.ServiceContext, chainConfig *params.ChainConfig, blockChain BlockChain, mintTime time.Duration) (*CliqueService, error) {
	service := &CliqueService{
		eventMux: ctx.EventMux,
		//remoteStorage: nil,
		blockchain: blockChain,
	}

	service.minter = newMinterPOA(chainConfig, service, mintTime)
	return service, nil
}

// Backend interface methods:
func (service *CliqueService) BlockChain() BlockChain     { return service.blockchain }
func (service *CliqueService) StorageLayer() StorageLayer { return service.remoteStorage }
func (service *CliqueService) EventMux() *event.TypeMux   { return service.eventMux }
func (service *CliqueService) TxPool() TxPool             { return service.blockchain.TxPool() }
func (service *CliqueService) Protocols() []p2p.Protocol  { return []p2p.Protocol{} }
func (service *CliqueService) APIs() []rpc.API            { return []rpc.API{} }

// Start implements node.Service, starting the background data propagation thread
// of the protocol.
func (service *CliqueService) Start(p2pServer *p2p.Server) error {
	log.Info("Clique started")
	return nil
}

// Stop implements node.Service, stopping the background data propagation thread
// of the protocol.
func (service *CliqueService) Stop() error {
	service.blockchain.Stop()
	service.minter.stop()
	if service.eventMux != nil {
		service.eventMux.Stop()
	}
	//service.cloudstore.Close()

	log.Info("Clique stopped")
	return nil
}
