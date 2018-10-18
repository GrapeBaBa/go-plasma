// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.

// adapted from
// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package deep

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eapache/channels"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type InvalidRaftOrdering struct {
	// Current head of the chain
	headBlock Block

	// New block that should point to the head, but doesn't
	invalidBlock Block
}

type minter struct {
	head          uint64
	state         StateDB
	config        *params.ChainConfig
	mu            sync.Mutex
	mux           *event.TypeMux
	deepchain     Backend
	chain         BlockChain
	remoteStorage StorageLayer
	coinbase      common.Address
	minting       int32 // Atomic status counter
	mineflag      int32
	shouldMine    *channels.RingChannel
	blockTime     time.Duration
	mintTime      time.Duration
	//speculativeChain *speculativeChain

	invalidRaftOrderingChan chan InvalidRaftOrdering
	chainHeadChan           chan ChainHeadEvent // was: core.ChainHeadEvent
	chainHeadSub            event.Subscription
	txPreChan               chan TxPreEvent // was: core.TxPreEvent
	txPreSub                event.Subscription
	mintRequestChan         chan MintRequestEvent
	mintReuqestSub          event.Subscription
}

//func newMinter(config *params.ChainConfig, deepchain *RaftService, blockTime time.Duration, chainHeadChan chan interface{}, txPreChan chan interface{}) *minter {
func newMinter(config *params.ChainConfig, deepchain *RaftService, blockTime time.Duration, mintTime time.Duration) *minter {
	minter := &minter{
		config:     config,
		deepchain:  deepchain,
		mux:        deepchain.EventMux(),
		chain:      deepchain.BlockChain(),
		shouldMine: channels.NewRingChannel(1),
		blockTime:  blockTime,
		mintTime:   mintTime,
		//speculativeChain: newSpeculativeChain(),
		invalidRaftOrderingChan: make(chan InvalidRaftOrdering, 1),
		chainHeadChan:           make(chan ChainHeadEvent, 1),
		txPreChan:               make(chan TxPreEvent, 4096),
		mintRequestChan:         make(chan MintRequestEvent, 1),
		head:                    0,
	}

	minter.remoteStorage = minter.chain.GetStorageLayer()
	log.Info(fmt.Sprintf("minter.go:newMinter | minter assigned chain of [%+v]", minter.chain))
	minter.chainHeadSub = deepchain.BlockChain().SubscribeChainHeadEvent(minter.chainHeadChan)
	log.Info(fmt.Sprintf("SubscribeTxPreEvent %v", minter.txPreChan))
	minter.txPreSub = deepchain.TxPool().SubscribeTxPreEvent(minter.txPreChan)
	minter.mintReuqestSub = deepchain.TxPool().SubscribeMintRequestEvent(minter.mintRequestChan)

	//minter.speculativeChain.clear(minter.chain.CurrentBlock())
	go minter.eventLoop()
	go minter.mintingLoop()
	go minter.triggerLoop()

	return minter
}

func (minter *minter) start() {
	atomic.StoreInt32(&minter.minting, 1)
	atomic.StoreInt32(&minter.mineflag, 0)
	minter.requestMinting()
}

func (minter *minter) stop() {
	minter.mu.Lock()
	defer minter.mu.Unlock()

	//minter.speculativeChain.clear(minter.chain.CurrentBlock())
	atomic.StoreInt32(&minter.minting, 0)
}

// Notify the minting loop that minting should occur, if it's not already been
// requested. Due to the use of a RingChannel, this function is idempotent if
// called multiple times before the minting occurs.
func (minter *minter) requestMinting() {
	minter.shouldMine.In() <- struct{}{}
}

func (minter *minter) triggerLoop() {
	log.Debug("triggerLoop")
	if minter.mintTime == 0 {
		return
	}
	trigger := time.NewTicker(minter.mintTime)
	for {
		select {
		case <-trigger.C:
			log.Info("Mineflag", "value", atomic.LoadInt32(&minter.mineflag))
			if atomic.LoadInt32(&minter.mineflag) == 0 {
				log.Debug("triggerLoop minting Request")
				minter.requestMinting()
			}
		}
	}
}

/*
func (minter *minter) updateSpeculativeChainPerNewHead(newHeadBlock Block) {
	minter.mu.Lock()
	defer minter.mu.Unlock()

	minter.speculativeChain.accept(newHeadBlock)
}

func (minter *minter) updateSpeculativeChainPerInvalidOrdering(headBlock Block, invalidBlock Block) {
	invalidHash := invalidBlock.Hash()

	log.Info("Handling InvalidRaftOrdering", "invalid block", invalidHash, "current head", headBlock.Hash())

	minter.mu.Lock()
	defer minter.mu.Unlock()

	// 1. if the block is not in our db, exit. someone else mined this.
	if !minter.chain.HasBlock(invalidHash, invalidBlock.NumberU64()) {
		log.Info("Someone else mined invalid block; ignoring", "block", invalidHash)

		return
	}

	minter.speculativeChain.unwindFrom(invalidHash, headBlock)
}
*/

func (minter *minter) eventLoop() {
	defer minter.chainHeadSub.Unsubscribe()
	defer minter.txPreSub.Unsubscribe()
	defer minter.mintReuqestSub.Unsubscribe()

	for {
		select {
		case ev := <-minter.chainHeadChan:
			//TODO:
			log.Info(fmt.Sprintf("minter eventLoop %d", minter.chain.CurrentBlock().Number()))

			log.Info("minter.eventLoop", "HeadChan", ev)
			/*
				newHeadBlock := ev.Block
			*/

			if atomic.LoadInt32(&minter.minting) == 1 {
				/*
					minter.updateSpeculativeChainPerNewHead(newHeadBlock)
				*/

				//
				// TODO(bts): not sure if this is the place, but we're going to
				// want to put an upper limit on our speculative mining chain
				// length.
				//

				//				minter.requestMinting()
			} else {
				minter.mu.Lock()
				/*
					minter.speculativeChain.setHead(newHeadBlock)
				*/
				minter.mu.Unlock()
			}

		//case <-minter.txPreChan:
		case <-minter.mintRequestChan:
			log.Info("Received MintRequest", "minter.minting", atomic.LoadInt32(&minter.minting), "minter.mineflag", atomic.LoadInt32(&minter.mineflag))
			if atomic.LoadInt32(&minter.minting) == 1 && atomic.LoadInt32(&minter.mineflag) == 0 {
				minter.requestMinting()
			}

		case ev := <-minter.invalidRaftOrderingChan:
			// TODO
			headBlock := ev.headBlock
			invalidBlock := ev.invalidBlock
			log.Debug(fmt.Sprintf("invalidRaftOrderingChan %v %v", headBlock, invalidBlock))

			//minter.updateSpeculativeChainPerInvalidOrdering(headBlock, invalidBlock)

		// system stopped
		case <-minter.chainHeadSub.Err():
			return
		case <-minter.txPreSub.Err():
			return
		}
	}
}

// Returns a wrapper around no-arg func `f` which can be called without limit
// and returns immediately: this will call the underlying func `f` at most once
// every `rate`. If this function is called more than once before the underlying
// `f` is invoked (per this rate limiting), `f` will only be called *once*.
//
// TODO(joel): this has a small bug in that you can't call it *immediately* when
// first allocated.
func throttle(rate time.Duration, f func()) func() {
	request := channels.NewRingChannel(1)

	// every tick, block waiting for another request. then serve it immediately
	go func() {
		ticker := time.NewTicker(rate)
		defer ticker.Stop()

		for range ticker.C {
			<-request.Out()
			go f()
		}
	}()

	return func() {
		request.In() <- struct{}{}
	}
}

// This function spins continuously, blocking until a block should be created
// (via requestMinting()). This is throttled by `minter.blockTime`:
//
//   1. A block is guaranteed to be minted within `blockTime` of being
//      requested.
//   2. We never mint a block more frequently than `blockTime`.
func (minter *minter) mintingLoop() {
	throttledMintNewBlock := throttle(minter.blockTime, func() {
		if atomic.LoadInt32(&minter.minting) == 1 {
			minter.mintNewBlock()
		}
	})

	for range minter.shouldMine.Out() {
		log.Debug("shouldMine.Out")
		throttledMintNewBlock()
	}
}

func generateNanoTimestamp(parent Block) (tstamp int64) {
	parentTime := parent.Time().Int64()
	tstamp = time.Now().UnixNano()

	if parentTime >= tstamp {
		// Each successive block needs to be after its predecessor.
		tstamp = parentTime + 1
	}

	return
}

func (minter *minter) makeBlock(p common.Hash, h Header, n uint64, txs Transactions) (b Block, err error) {
	return minter.deepchain.BlockChain().MakeBlock(p, h, n, txs)
}

func (minter *minter) getTransactions() (tr Transactions) {

	alltxns := minter.deepchain.TxPool().GetTransactions()

	//	addrTxes := minter.speculativeChain.withoutProposedTxes(allAddrTxes)

	// TODO: signer := MakeSigner(minter.chain.Config(), minter.chain.CurrentBlock().Number())

	return alltxns
}

// Sends-off events asynchronously.
func (minter *minter) firePendingBlockEvents() {
	go func() {
		minter.mux.Post(PendingStateEvent{})
	}()
}

func (minter *minter) mintNewBlock() error {
	log.Info("MineFlag", "value", atomic.LoadInt32(&minter.mineflag))
	//states that minting is going on do not disturb
	if atomic.LoadInt32(&minter.mineflag) == 1 {
		log.Debug(fmt.Sprintf("mintNewBlock atomic is 1"))
		// need to wait next minting...
		// may need to modify not to wait next minting comes
		return nil
	}
	atomic.StoreInt32(&minter.mineflag, 1)
	minter.mu.Lock() //Locks the minter object (as a WHOLE)
	defer minter.mu.Unlock()

	//log.Debug(fmt.Sprintf("minter.go:mintNewBlock - minter.chain is [%+v]", minter.chain))
	minter.head = minter.chain.CurrentBlock().Number()
	parent := minter.chain.CurrentBlock()
	parentNumber := parent.Number()
	state, err := minter.chain.StateAt(parent.Root())
	if err != nil {
		log.Debug("Error getting state", "error", err, "root", parent.Root(), "state", state)
		panic(fmt.Sprint("failed to get parent state: ", err))
	}
	//log.Debug("State Object retrieved and root", "root", parent.Root(), "state", state)
	minter.state = state
	transactions := minter.getTransactions()
	committedTxes := minter.commitTransactions(transactions, minter.chain)
	txCount := len(committedTxes)
	if txCount == 0 {
		log.Info("Not minting a new block since there are no pending transactions")
		atomic.StoreInt32(&minter.mineflag, 0)
		return nil
	}

	minter.firePendingBlockEvents()
	var hdr Header

	if hdr, err = state.Commit(minter.remoteStorage, parentNumber+1, parent.Hash()); err != nil {
		panic(fmt.Sprint("error committing public state: ", err))
	}
	block, err := minter.makeBlock(parent.Hash(), hdr, parentNumber+1, committedTxes)
	log.Info("Generated next block", "block num", block.Number(), "num txes", txCount)
	//minter.speculativeChain.extend(block)

	log.Info("Flushing Chunks ")
	err = minter.remoteStorage.Flush()
	if err != nil {
		log.Error("[deep:minter:mintNewBlock] Error flushing chunks associated with attempted block", "error", err)
		return fmt.Errorf("[deep:mint:mintNewBlock] Error flushing chunks associated with attempted block %x, %s", block.Hash(), err)
	} else {
		fmt.Printf("[deep:minter:mintNewBlock] flushed chunks %x\n", block.Hash())
	}

	minter.mux.Post(NewMinedBlockEvent{Block: block})

	// elapsed := time.Since(time.Unix(0, header.Time.Int64()))
	log.Info("🔨  Mined block", "number", block.Number(), "hash", fmt.Sprintf("%x", block.Hash().Bytes()[:4]))

	host, _ := os.Hostname()
	log.Info(fmt.Sprintf("[minter.go:mintNewBlock] HOST: %s ChainType %s ChainId: %d", host, minter.chain.ChainType(), minter.chain.GetBlockChainID()))

	return nil
}

func (env *minter) commitTransactions(txes Transactions, bc BlockChain) Transactions {
	log.Info("Starting to commit transactions", "count", len(txes))
	var committedTxes Transactions

	numtxns := len(txes)
	for i := 0; i < numtxns; i++ {
		tx := txes[i]
		// state.Prepare(tx.Hash(), common.Hash{}, txCount)
		err := env.commitTransaction(tx, bc)
		switch {
		case err != nil:
			log.Info("TX failed, will be removed", "hash", tx.Hash(), "err", err)
			// TODO: skip rest of txes from this account?
			env.deepchain.TxPool().RemoveTransaction(tx.Hash())
			i--       //re-do index i, b/c previous index i was removed
			numtxns-- //update number of loops
		default:
			committedTxes = append(committedTxes, tx)
		}
	}

	return committedTxes
}

func (env *minter) commitTransaction(tx Transaction, bc BlockChain) error {
	log.Info("Starting to commit transaction", "txhash", fmt.Sprintf("%x", tx.Hash()))
	snapshot := env.state.Snapshot()
	err := bc.ApplyTransaction(env.state, tx)
	log.Debug(fmt.Sprintf("commitTransaction %v %v", snapshot, err))
	if err != nil { //transaction failed
		reverterr := env.state.RevertToSnapshot(snapshot)
		if reverterr != nil { //revert failed
			return fmt.Errorf("[deep:minter:commitTransaction] %s, %s", err, reverterr)
		}
		log.Info("ERROR: Unable to Apply TX", "tx.hash", tx.Hash())
		return fmt.Errorf("[deep:minter:commitTransaction] %s", err)
	}

	return nil
}

func (minter *minter) ResetMiningFlag() {
	atomic.StoreInt32(&minter.mineflag, 0)
}
