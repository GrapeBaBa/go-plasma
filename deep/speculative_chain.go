// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package deep

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/fatih/set.v0"
	lane "gopkg.in/oleiade/lane.v1"
)

// The speculative chain represents blocks that we have minted which haven't been accepted into the chain yet, building
// on each other in a chain. It has three basic operations:
// * add new block to end
// * accept / remove oldest block
// * unwind / remove invalid blocks to the end
//
// Additionally:
// * clear state when we stop minting
// * set the parent when we're not minting (so it's always current)
type speculativeChain struct {
	head                       Block
	unappliedBlocks            *lane.Deque
	expectedInvalidBlockHashes *set.Set // This is thread-safe. This set is referred to as our "guard" below.
	proposedTxes               *set.Set // This is thread-safe.
}

func newSpeculativeChain() *speculativeChain {
	return &speculativeChain{
		head:                       nil,
		unappliedBlocks:            lane.NewDeque(),
		expectedInvalidBlockHashes: set.New(),
		proposedTxes:               set.New(),
	}
}

func (chain *speculativeChain) clear(block Block) {
	chain.head = block
	chain.unappliedBlocks = lane.NewDeque()
	chain.expectedInvalidBlockHashes.Clear()
	chain.proposedTxes.Clear()
}

// Append a new speculative block
func (chain *speculativeChain) extend(block Block) {
	chain.head = block
	chain.recordProposedTransactions(block.Transactions())
	chain.unappliedBlocks.Append(block)
}

// Set the parent of the speculative chain
//
// Note: This is only called when not minter
func (chain *speculativeChain) setHead(block Block) {
	chain.head = block
}

// Accept this block, removing it from the head of the speculative chain
func (chain *speculativeChain) accept(acceptedBlock Block) {
	earliestProposedI := chain.unappliedBlocks.Shift()
	var earliestProposed Block
	if nil != earliestProposedI {
		earliestProposed = earliestProposedI.(Block)
	}

	if expectedBlock := earliestProposed == nil || earliestProposed.Hash() == acceptedBlock.Hash(); expectedBlock {
		// Remove the txes in this accepted block from our blacklist.
		chain.removeProposedTxes(acceptedBlock)
	} else {
		log.Info("Another node minted; Clearing speculative state", "block", acceptedBlock.Hash())

		chain.clear(acceptedBlock)
	}
}

// Remove all blocks in the chain from the specified one until the end
func (chain *speculativeChain) unwindFrom(invalidHash common.Hash, headBlock Block) {

	// check our "guard" to see if this is a (descendant) block we're
	// expected to be ruled invalid. if we find it, remove from the guard
	if chain.expectedInvalidBlockHashes.Has(invalidHash) {
		log.Info("Removing expected-invalid block from guard.", "block", invalidHash)

		chain.expectedInvalidBlockHashes.Remove(invalidHash)

		return
	}

	// pop from the RHS repeatedly, updating minter.parent each time. if not
	// our block, add to guard. in all cases, call removeProposedTxes
	for {
		currBlockI := chain.unappliedBlocks.Pop()

		if nil == currBlockI {
			log.Info("(Popped all blocks from queue.)")

			break
		}

		currBlock := currBlockI.(Block)

		log.Info("Popped block from queue RHS.", "block", currBlock.Hash())

		// Maintain invariant: the parent always points the last speculative block or the head of the blockchain
		// if there are not speculative blocks.
		if speculativeParentI := chain.unappliedBlocks.Last(); nil != speculativeParentI {
			chain.head = speculativeParentI.(Block)
		} else {
			chain.head = headBlock
		}

		chain.removeProposedTxes(currBlock)

		if currBlock.Hash() != invalidHash {
			log.Info("Haven't yet found block; adding descendent to guard.\n", "invalid block", invalidHash, "descendant", currBlock.Hash())

			chain.expectedInvalidBlockHashes.Add(currBlock.Hash())
		} else {
			break
		}
	}
}

// We keep track of txes we've put in all newly-mined blocks since the last
// ChainHeadEvent, and filter them out so that we don't try to create blocks
// with the same transactions. This is necessary because the TX pool will keep
// supplying us these transactions until they are in the chain (after having
// flown through raft).
func (chain *speculativeChain) recordProposedTransactions(txes Transactions) {
	txHashIs := make([]interface{}, len(txes))
	for i, tx := range txes {
		txHashIs[i] = tx.Hash()
	}
	chain.proposedTxes.Add(txHashIs...)
}

// Removes txes in block from our "blacklist" of "proposed tx" hashes. When we
// create a new block and use txes from the tx pool, we ignore those that we
// have already used ("proposed"), but that haven't yet officially made it into
// the chain yet.
//
// It's important to remove hashes from this blacklist (once we know we don't
// need them in there anymore) so that it doesn't grow endlessly.
func (chain *speculativeChain) removeProposedTxes(block Block) {
	minedTxes := block.Transactions()
	minedTxInterfaces := make([]interface{}, len(minedTxes))
	for i, tx := range minedTxes {
		minedTxInterfaces[i] = tx.Hash()
	}

	// NOTE: we are using a thread-safe Set here, so it's fine if we access this
	// here and in mintNewBlock concurrently. using a finer-grained set-specific
	// lock here is preferable, because mintNewBlock holds its locks for a
	// nontrivial amount of time.
	chain.proposedTxes.Remove(minedTxInterfaces...)
}

func (chain *speculativeChain) withoutProposedTxes(addrTxes map[common.Address]Transactions) map[common.Address]Transactions {
	newMap := make(map[common.Address]Transactions)

	for addr, txes := range addrTxes {
		filteredTxes := make(Transactions, 0)
		for _, tx := range txes {
			if !chain.proposedTxes.Has(tx.Hash()) {
				filteredTxes = append(filteredTxes, tx)
			}
		}
		if len(filteredTxes) > 0 {
			newMap[addr] = filteredTxes
		}
	}

	return newMap
}
