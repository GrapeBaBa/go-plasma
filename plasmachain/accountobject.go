// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.

package plasmachain

import (
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// accountObject represents a wallet address which is being modified.
type accountObject struct {
	address common.Address
	db      *StateDB
	acct    *Account
	deleted bool
}

//NewAccountObject creates an account object.
func NewAccountObject(db *StateDB, addr common.Address, a *Account) *accountObject {
	return &accountObject{
		db:      db,
		address: addr,
		acct:    a,
		deleted: false,
	}
}

// EncodeRLP implements rlp.Encoder.
func (self *accountObject) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, self.acct)
}

func (self *accountObject) AddToken(tokenID uint64, tokenBalace, tokenDenomination *big.Int) {
	self.db.journal.AddEntry(accountTokenChange{
		addr:                 self.address,
		prevTokens:           self.acct.Tokens,
		prevAcctBalance:      self.acct.Balance,
		prevAcctDenomination: self.acct.Denomination,
	})
	self.acct.Tokens = append(self.acct.Tokens, tokenID)
	//self.acct.Balance = self.acct.Balance + tokenBalace
	//self.acct.Denomination = self.acct.Denomination + tokenDenomination
	self.acct.Balance = new(big.Int).Add(self.Balance(), tokenBalace)
	self.acct.Denomination = new(big.Int).Add(self.Denomination(), tokenDenomination)
}

func (self *accountObject) RemoveToken(tokenID uint64, tokenBalace, tokenDenomination *big.Int) {
	self.db.journal.AddEntry(accountTokenChange{
		addr:                 self.address,
		prevTokens:           self.acct.Tokens,
		prevAcctBalance:      self.acct.Balance,
		prevAcctDenomination: self.acct.Denomination,
	})
	updatedTokenList := []uint64{}
	for _, tID := range self.acct.Tokens {
		if tID != tokenID {
			updatedTokenList = append(updatedTokenList, tID)
		}
	}
	self.acct.Tokens = updatedTokenList
	//self.acct.Balance = self.acct.Balance - tokenBalace
	//self.acct.Denomination = self.acct.Denomination - tokenDenomination
	self.acct.Balance = new(big.Int).Sub(self.Balance(), tokenBalace)
	self.acct.Denomination = new(big.Int).Sub(self.Denomination(), tokenDenomination)
}

func (self *accountObject) SetAccountBalance(balance *big.Int) {
	self.db.journal.AddEntry(accountBalanceChange{
		addr:            self.address,
		prevAcctBalance: self.acct.Balance,
	})
	self.acct.Balance = balance
}

func (self *accountObject) SetAccountDenomination(denomination *big.Int) {
	self.db.journal.AddEntry(accountDenominationChange{
		addr:                 self.address,
		prevAcctDenomination: self.acct.Denomination,
	})
	self.acct.Denomination = denomination
}

func (self *accountObject) deepCopy(db *StateDB) *accountObject {
	accountObject := NewAccountObject(db, self.address, self.acct)
	return accountObject
}

//
// Attribute accessors
//

func (self *accountObject) Address() common.Address {
	return self.address
}

func (self *accountObject) Balance() *big.Int {
	return self.acct.Balance
}

func (self *accountObject) Denomination() *big.Int {
	return self.acct.Denomination
}

func (self *accountObject) String() string {
	s := fmt.Sprintf("Obj: %v, acct: %v, MarkDel: %v", self.address, self.acct, self.deleted)
	return s
}
