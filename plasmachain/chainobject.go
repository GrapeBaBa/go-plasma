// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.
package plasmachain

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// accountObject represents a wallet address which is being modified.
type chainObject struct {
	blockchainID uint64
	db           *StateDB
	anb          *anchorBlock
	deleted      bool
}

func NewChainObject(db *StateDB, chainID uint64, anchorblock *anchorBlock) *chainObject {
	return &chainObject{
		blockchainID: chainID,
		db:           db,
		anb:          anchorblock,
		deleted:      false,
	}
}

// EncodeRLP implements rlp.Encoder.
func (self *chainObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, self.anb)
}

func (self *chainObject) AddOwner(addr common.Address) {
	self.db.journal.AddEntry(chainOwnerChange{
		chainID:    self.blockchainID,
		prevOwners: self.anb.Owners,
	})
	self.anb.Owners = append(self.anb.Owners, &addr)
}

func (self *chainObject) RemoveOwner(addr common.Address) {
	self.db.journal.AddEntry(chainOwnerChange{
		chainID:    self.blockchainID,
		prevOwners: self.anb.Owners,
	})
	self.anb.Owners = remove(self.anb.Owners, &addr)
}

func (self *chainObject) SetEndBlock(endBlock uint64) {
	self.db.journal.AddEntry(anchorBlockChange{
		chainID:   self.blockchainID,
		prevStart: self.anb.EndBlock,
		prevEnd:   self.anb.EndBlock,
	})
	self.anb.EndBlock = endBlock
}

func (self *chainObject) SetStartBlock(startBlock uint64) {
	self.db.journal.AddEntry(anchorBlockChange{
		chainID:   self.blockchainID,
		prevStart: self.anb.EndBlock,
		prevEnd:   self.anb.EndBlock,
	})
	self.anb.StartBlock = startBlock
}

func (self *chainObject) deepCopy(db *StateDB) *chainObject {
	chainObject := NewChainObject(db, self.BlockchainID(), self.anb)
	return chainObject
}

//
// Attribute accessors
//

func (self *chainObject) BlockchainID() uint64 {
	return self.blockchainID
}

func (self *chainObject) Existed(addr common.Address) bool {
	for _, a := range self.anb.Owners {
		if *a == addr {
			return true
		}
	}
	return false
}

func (self *chainObject) OwnerCount() int {
	return len(self.anb.Owners)
}

func (self *chainObject) StartBlock() uint64 {
	return self.anb.StartBlock
}

func (self *chainObject) EndBlock() uint64 {
	return self.anb.EndBlock
}

func (self *chainObject) String() string {
	s := fmt.Sprintf("Obj: %v, anchorBlock: %v, MarkDel: %v", self.blockchainID, self.anb, self.deleted)
	return s
}

func remove(addrList []*common.Address, addr *common.Address) []*common.Address {
	for i := len(addrList) - 1; i >= 0; i-- {
		if addrList[i] == addr {
			addrList = append(addrList[:i], addrList[i+1:]...)
		}
	}
	return addrList
}

func removeDuplicates(addrList []*common.Address) []*common.Address {
	existed := map[common.Address]bool{}
	for addr := range addrList {
		existed[*addrList[addr]] = true
	}

	addrs := []*common.Address{}
	for uniqueAddr, _ := range existed {
		addrs = append(addrs, &uniqueAddr)
	}
	return addrs
}
