// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.
package plasmachain

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// tokenObject represents a Plasma token which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// token balance can be accessed and modified through the object.
// Finally, call Commit to write the modified token state into cloud storage database.
type tokenObject struct {
	tokenID uint64
	db      *StateDB
	token   *Token
	deleted bool
}

// newObject creates a state object.
func NewTokenObject(db *StateDB, tokenID uint64, t *Token) *tokenObject {
	return &tokenObject{
		db:      db,
		tokenID: tokenID,
		token:   t,
		deleted: false,
	}
}

// EncodeRLP implements rlp.Encoder.
func (self *tokenObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, self.token)
}

func (self *tokenObject) SetOwner(owner common.Address) {
	self.db.journal.AddEntry(ownerChange{
		tokenID:   self.tokenID,
		prevOwner: self.token.Owner,
	})
	self.token.Owner = owner
}

//equivalent of prevNounce
func (self *tokenObject) SetPrevBlock(prevBlocknum uint64) {
	self.db.journal.AddEntry(prevBlockChange{
		tokenID:   self.tokenID,
		prevBlock: self.token.PrevBlock,
	})
	self.token.PrevBlock = prevBlocknum
}

func (self *tokenObject) SetBalance(balance uint64, allowance uint64) {
	self.db.journal.AddEntry(balanceChange{
		tokenID:       self.tokenID,
		prevBalance:   self.token.Balance,
		prevAllowance: self.token.Allowance,
	})

	self.token.Balance = balance
	self.token.Allowance = allowance
}

func (self *tokenObject) SetWithdrawal(allowance uint64, spent uint64) {
	self.db.journal.AddEntry(withdrawChange{
		tokenID:       self.tokenID,
		prevAllowance: self.token.Allowance,
		prevSpent:     self.token.Spent,
	})

	self.token.Allowance = allowance
	self.token.Spent = spent

}

func (self *tokenObject) SetExit() {
	self.db.journal.AddEntry(exitChange{
		tokenID: self.tokenID,
	})
	self.deleted = true
}

func (self *tokenObject) deepCopy(db *StateDB) *tokenObject {
	tokenObject := NewTokenObject(db, self.tokenID, self.token)

	return tokenObject
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (self *tokenObject) TokenID() uint64 {
	return self.tokenID
}

func (self *tokenObject) PrevBlock() uint64 {
	return self.token.PrevBlock
}

func (self *tokenObject) Owner() common.Address {
	return self.token.Owner
}

func (self *tokenObject) Denomination() uint64 {
	return self.token.Denomination
}

func (self *tokenObject) Balance() uint64 {
	return self.token.Balance
}

func (self *tokenObject) Allowance() uint64 {
	return self.token.Allowance
}

func (self *tokenObject) Spent() uint64 {
	return self.token.Spent
}

func (self *tokenObject) ValidDemination() bool {
	if self.token.Denomination == self.token.Balance+self.token.Spent+self.token.Allowance {
		return true
	} else {
		return false
	}
}

func (self *tokenObject) String() string {
	s := fmt.Sprintf("Obj: %v, t: %v, MarkDel: %v", self.tokenID, self.token, self.deleted)
	return s
}
