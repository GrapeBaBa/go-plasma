// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package plasmachain

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type journalEntry interface {
	revert(*StateDB) error
}

type journal struct {
	entries []journalEntry // Current changes tracked by the journal
}

//reverting tokenstate journal change. blockNumber is rolled-back
type (

	//full state revert
	resetObjectChange struct {
		prev *tokenObject
	}

	// Changes to token trie
	// Deposit(Createobject)
	depositChange struct {
		tokenID uint64
	}

	// owner exit (suicideChange)
	exitChange struct {
		tokenID uint64
	}

	// Changes to individual token.
	// owner consumes service
	balanceChange struct {
		tokenID       uint64
		prevBalance   uint64
		prevAllowance uint64
	}

	// operator withdraw allowance
	withdrawChange struct {
		tokenID       uint64
		prevAllowance uint64
		prevSpent     uint64
	}

	// ownership transfer affects Sender and Recipient
	ownerChange struct {
		tokenID   uint64
		prevOwner common.Address
	}

	// any operation will affect prevBlock
	prevBlockChange struct {
		tokenID   uint64
		prevBlock uint64
	}
)

//reverting account journal change.
type (
	resetAcctObjectChange struct {
		prev *accountObject
	}

	accountTokenChange struct {
		addr                 common.Address
		prevTokens           []uint64
		prevAcctBalance      *big.Int
		prevAcctDenomination *big.Int
	}

	openAccountChange struct {
		addr common.Address
	}

	accountBalanceChange struct {
		addr            common.Address
		prevAcctBalance *big.Int
	}

	accountDenominationChange struct {
		addr                 common.Address
		prevAcctDenomination *big.Int
	}
)

//reverting layer3 Chain journal change.
type (
	resetChainObjectChange struct {
		prev *chainObject
	}

	chainOwnerChange struct {
		chainID    uint64
		prevOwners []*common.Address
		prevBlock  uint64
	}

	openChainChange struct {
		chainID uint64
	}

	// any operation will affect EndBlock
	anchorBlockChange struct {
		chainID   uint64
		prevStart uint64
		prevEnd   uint64
	}
)

// newJournal create a new initialized journal.
func newJournal() *journal {
	return &journal{}
}

func (j *journal) Len() int {
	return len(j.entries)
}

func (j *journal) AddEntry(e journalEntry) {
	j.entries = append(j.entries, e)
}

func (ch depositChange) revert(s *StateDB) error {
	delete(s.tokenObjects, ch.tokenID)
	return nil
}

func (ch exitChange) revert(s *StateDB) error {
	tobj, err := s.getTokenObject(ch.tokenID)
	if err != nil {
		return err
	}
	tobj.deleted = false
	return nil
}

func (ch balanceChange) revert(s *StateDB) error {
	tobj, err := s.getTokenObject(ch.tokenID)
	if err != nil {
		return err
	}
	tobj.SetBalance(ch.prevBalance, ch.prevAllowance)
	return nil
}

func (ch withdrawChange) revert(s *StateDB) error {
	tobj, err := s.getTokenObject(ch.tokenID)
	if err != nil {
		return err
	}
	tobj.SetWithdrawal(ch.prevAllowance, ch.prevSpent)
	return nil
}

func (ch ownerChange) revert(s *StateDB) error {
	tobj, err := s.getTokenObject(ch.tokenID)
	if err != nil {
		return err
	}
	tobj.SetOwner(ch.prevOwner)
	return nil
}

// uncertain where should this be called?
func (ch prevBlockChange) revert(s *StateDB) error {
	tobj, err := s.getTokenObject(ch.tokenID)
	if err != nil {
		return err
	}
	tobj.SetPrevBlock(ch.prevBlock)
	return nil
}

// account state revert
func (ch accountBalanceChange) revert(s *StateDB) error {
	ao, err := s.getOrCreateAccount(ch.addr)
	if err != nil {
		return err
	}
	ao.acct.Balance = ch.prevAcctBalance
	return nil
}

func (ch accountDenominationChange) revert(s *StateDB) error {
	ao, err := s.getOrCreateAccount(ch.addr)
	if err != nil {
		return err
	}
	ao.acct.Denomination = ch.prevAcctDenomination
	return nil
}

func (ch accountTokenChange) revert(s *StateDB) error {
	ao, err := s.getOrCreateAccount(ch.addr)
	if err != nil {
		return err
	}
	ao.acct.Tokens = ch.prevTokens
	ao.acct.Balance = ch.prevAcctBalance
	ao.acct.Denomination = ch.prevAcctDenomination
	return nil
}

func (ch chainOwnerChange) revert(s *StateDB) error {
	c, err := s.getOrCreateChain(ch.chainID)
	if err != nil {
		return err
	}
	c.anb.Owners = ch.prevOwners
	return nil
}

func (ch anchorBlockChange) revert(s *StateDB) error {
	c, err := s.getOrCreateChain(ch.chainID)
	if err != nil {
		return err
	}
	c.anb.StartBlock = ch.prevStart
	c.anb.EndBlock = ch.prevEnd
	return nil
}
