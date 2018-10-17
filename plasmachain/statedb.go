// Copyright 2018 Wolk Inc.
// This file is part of the Wolk Deep Blockchains library.
package plasmachain

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/wolkdb/go-plasma/deep"
	"github.com/wolkdb/go-plasma/smt"
)

type revision struct {
	id           int
	journalIndex int
}

type StateDB struct {
	ChunkStore deep.StorageLayer
	Storage    *Storage

	// TODO: why are we including these?
	thash   common.Hash
	bhash   common.Hash
	txIndex int

	hash          []byte
	DefaultHashes [smt.TreeDepth][]byte
	operatorKey   *ecdsa.PrivateKey

	// token ID objects
	tokenObjects map[uint64]*tokenObject
	tokenIsSet   map[uint64]bool

	// account objects
	accountObjects map[common.Address]*accountObject

	// layer3chain objects
	chainObjects map[uint64]*chainObject
	chainIsSet   map[uint64]bool

	// transactionStorage (SMT)
	thashList map[uint64]common.Hash

	// AnchorTransaction (SMT)
	atxList map[uint64]*anchorBlock

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        *journal
	validRevisions []revision
	nextRevisionId int

	lock sync.Mutex
}

func NewStateDB(cs deep.StorageLayer, headerHash common.Hash, operatorKey *ecdsa.PrivateKey) *StateDB {
	var self StateDB
	self.Storage = NewStorage(cs)
	self.Storage.InitTrie(headerHash)
	self.DefaultHashes = smt.ComputeDefaultHashes()
	self.ChunkStore = cs
	self.operatorKey = operatorKey
	self.tokenObjects = make(map[uint64]*tokenObject)
	self.tokenIsSet = make(map[uint64]bool)
	self.chainIsSet = make(map[uint64]bool)

	self.chainObjects = make(map[uint64]*chainObject)
	self.accountObjects = make(map[common.Address]*accountObject)

	self.thashList = make(map[uint64]common.Hash)
	self.atxList = make(map[uint64]*anchorBlock)
	self.journal = newJournal()
	return &self
}

func (self *StateDB) RevertToSnapshot(revid int) error {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= revid
	})
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != revid {
		return fmt.Errorf("[statedb:RevertToSnapshot] revision id %v cannot be reverted", revid)
	}
	snapshot := self.validRevisions[idx].journalIndex

	// Replay the journal to undo changes.
	for i := self.journal.Len() - 1; i >= snapshot; i-- {
		self.journal.entries[i].revert(self)
	}
	self.journal.entries = self.journal.entries[:snapshot]

	// Remove invalidated snapshots from the stack.
	self.validRevisions = self.validRevisions[:idx]
	return nil
}

func (self *StateDB) Append(tokenID uint64, sobj *tokenObject, txhash common.Hash) {
	self.tokenObjects[tokenID] = sobj
	self.tokenIsSet[tokenID] = true
	self.thashList[tokenID] = txhash
}

func (self *StateDB) AppendAnchor(chainID uint64, cobj *chainObject) {
	self.chainObjects[chainID] = cobj
	self.chainIsSet[chainID] = true
	log.Debug("AppendAnchor", "Anchor", cobj)
}

func (self *StateDB) Snapshot() (i int) {
	id := self.nextRevisionId
	self.nextRevisionId++
	self.validRevisions = append(self.validRevisions, revision{id, self.journal.Len()})
	return id
}

// Prepare sets the current transaction hash and index and block hash which is
// used when the EVM emits new state logs.
func (self *StateDB) Prepare(thash, bhash common.Hash, ti int) {
	//TODO: where is Prepare truly required?
	self.thash = thash
	self.bhash = bhash
	self.txIndex = ti
}

func (s *StateDB) clearJournal() {
	s.journal = newJournal()
	s.validRevisions = s.validRevisions[:0]
}

func (self *StateDB) StorePlasmaBloomFilter(b []byte) (h common.Hash, err error) {
	key := deep.Keccak256(b)
	err = self.ChunkStore.SetChunk(key, b)
	if err != nil {
		log.Info("StoreBlock ERR", "err", err)
		return h, err
	}
	return common.BytesToHash(key), nil
}

// CommitTo writes the state to cloudstore via the SMT
func (self *StateDB) Commit(cs deep.StorageLayer /*blockchainID uint64, */, blockNumber uint64, parentHash common.Hash) (dh deep.Header, err error) {
	defer self.clearJournal()
	// Commit objects to the SMT, and build bloom filter
	var tokenIDlist []uint64
	for tokenID, tokenObject := range self.tokenObjects {
		self.updateTokenObject(tokenObject)
		tokenIDlist = append(tokenIDlist, tokenID)
		delete(self.tokenObjects, tokenID)
		delete(self.tokenIsSet, tokenID)
		delete(self.thashList, tokenID)
	}
	log.Debug("Length of bloom", "Len", len(tokenIDlist))

	var bloomID common.Hash
	if len(tokenIDlist) > 0 {
		bloomChunk := NewBloom(tokenIDlist)
		bloomID, err = self.StorePlasmaBloomFilter(bloomChunk)
		if err != nil {
			log.Info("Commit:Bloom", "ERROR", err)
			return nil, err
		}
	}

	for addr, accountObject := range self.accountObjects {
		self.updateAccountObject(accountObject)
		delete(self.accountObjects, addr)
	}

	for chainID, chainObject := range self.chainObjects {
		self.updateChainObject(chainObject)
		delete(self.chainObjects, chainID)
	}

	for blockChainID, anchorBlock := range self.atxList {
		//self.makeAnchors(anchorTxs)
		anchorBlock.Sort()
		encoded, _ := rlp.EncodeToBytes(anchorBlock)
		log.Debug("anchorBlock: EncodeToBytes", "ENCODED", common.Bytes2Hex(encoded))
		v := deep.Keccak256(encoded)
		self.ChunkStore.SetChunk(v, encoded)

		for _, anchor := range anchorBlock.Anchors {
			//store individual anchor tx
			log.Info("Multiple Anchors", "txhash", anchor.Hex())
		}

		self.Storage.anchorStorage.Insert(deep.UInt64ToByte(blockChainID), anchorBlock.Bytes(), 0, 0)
		delete(self.atxList, blockChainID)

		/*
			var testa anchorObject
			err = rlp.Decode(bytes.NewReader(encoded), &testa)
			if err != nil {
				log.Info("anchorBlock: Verify", "anchorBlock", testa, "v", v, "ERR", err)
				return nil, err
			} else {
				log.Info("anchorBlock: Verify", "anchorBlock", testa)
			}
		*/
	}

	self.Storage.Flush()
	h := self.Storage.MakeHeaderHash(blockNumber, parentHash, bloomID)
	return &h, err
}

func (self *StateDB) getTokenObject(tokenID uint64) (s *tokenObject, err error) {
	//v0, found, storageBytes, prevBlock, err
	v, found, _, _, _, err := self.Storage.tokenStorage.Get(deep.UInt64ToByte(tokenID))
	if err != nil {
		return nil, err
	}
	if found == false {
		return nil, fmt.Errorf("Not found")
	}
	encoded, ok, err := self.ChunkStore.GetChunk(v)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}
	var t Token
	err = rlp.Decode(bytes.NewReader(encoded), &t)
	if err != nil {
		log.Info("getTokenObject: Decode", "tokenID", tokenID, "v", v, "ERR", err)
		return nil, err
	}
	log.Debug("getTokenObject", "tokenID", tokenID, "t", t.String())
	return NewTokenObject(self, tokenID, &t), nil
}

func (self *StateDB) createDeposit(tokenID uint64, denomination uint64, owner common.Address) (s *tokenObject, err error) {
	_, isFound, _, _, _, err := self.Storage.tokenStorage.Get(deep.UInt64ToByte(tokenID))
	if err != nil {
		log.Info("createDeposit", "tokenID", tokenID, "status", "ERROR")
		return s, err
	} else if isFound {
		log.Info("createDeposit", "tokenID", tokenID, "status", "ERROR")
		return s, fmt.Errorf("tokenID collation")
	}

	log.Debug("createDeposit", "tokenID", tokenID, "status", "canDeposit")
	t := Token{
		Denomination: denomination,
		PrevBlock:    0,
		Owner:        owner,
		Balance:      denomination,
		Allowance:    0,
		Spent:        0,
	}
	return NewTokenObject(self, tokenID, &t), nil
}

//TODO: handle future tokenID colation after delete
func (self *StateDB) updateTokenObject(s *tokenObject) (err error) {
	log.Debug("updateTokenObject", "s", s)
	encoded, _ := rlp.EncodeToBytes(s)
	log.Debug("updateTokenObject: EncodeToBytes", "ENCODED", common.Bytes2Hex(encoded))
	v := deep.Keccak256(encoded)
	err = self.ChunkStore.SetChunk(v, encoded)
	if err != nil {
		log.Info("updateTokenObject: StoreChunk", "v", v, "ERROR", err)
		return err
	} else {
		log.Debug("updateTokenObject: StoreChunk", "v", common.Bytes2Hex(v), "encoded", common.Bytes2Hex(encoded))
	}

	var t Token
	err = rlp.Decode(bytes.NewReader(encoded), &t)
	log.Debug("updateTokenObject check", "tokenID", deep.UInt64ToByte(s.tokenID), "t", t.String())

	if s.deleted {
		//token marked for deletion due to finalized exit
		err = self.Storage.tokenStorage.Delete(deep.UInt64ToByte(s.tokenID))
	} else {
		err = self.Storage.tokenStorage.Insert(deep.UInt64ToByte(s.tokenID), v, 0, 0)
	}

	if err == nil {
		txhash := self.thashList[s.tokenID]
		err = self.Storage.transactionStorage.Insert(deep.UInt64ToByte(s.tokenID), txhash.Bytes(), 0, 0)
	}

	if err != nil {
		log.Info("updateTokenObject", "s", s, "ERROR", err)
		return err
	}
	return nil
}

// using last 8bytes of the address for now
func (self *StateDB) getAccountObject(addr common.Address) (a *accountObject, err error) {
	//v0, found, storageBytes, prevBlock, err
	//TODO: SMT need to support uint160
	shortAddr := addr[12:20]
	v, found, _, _, _, err := self.Storage.accountStorage.Get(shortAddr)
	if err != nil {
		return nil, err
	}
	if found == false {
		return nil, fmt.Errorf("Not found")
	}
	encoded, ok, err := self.ChunkStore.GetChunk(v)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}
	var acct Account
	err = rlp.Decode(bytes.NewReader(encoded), &acct)
	if err != nil {
		log.Info("getAccountObject: Decode", "account", addr, "shortAddr", shortAddr, "v", v, "ERR", err)
		return nil, err
	}
	log.Debug("getAccountObject", "account", addr, "shortAddr", shortAddr, "acct", acct.String())
	return NewAccountObject(self, addr, &acct), nil
}

// deprecated
func (self *StateDB) createAccount(addr common.Address) (a *accountObject, err error) {
	shortAddr := addr[12:20]
	_, isFound, _, _, _, err := self.Storage.accountStorage.Get(shortAddr)
	if err != nil {
		log.Info("createAccount", "acct", addr, "shortAddr", shortAddr, "status", "ERROR")
		return a, err
	} else if isFound {
		log.Debug("createAccount", "acct", addr, "shortAddr", shortAddr, "status", "shortAddr collation")
		return a, fmt.Errorf("shortAddr already exist")
	}

	log.Debug("createAccount", "acct", addr, "status", "canOpen")
	acct := Account{
		Tokens:       []uint64{},
		Denomination: big.NewInt(0),
		Balance:      big.NewInt(0),
	}
	return NewAccountObject(self, addr, &acct), nil
}

// A 3-tier lookup procedure
func (self *StateDB) getOrCreateAccount(addr common.Address) (a *accountObject, err error) {

	//Tier1 : Lookup from current state
	if currentAccounObject, isSet := self.accountObjects[addr]; isSet {
		return currentAccounObject, nil
	}

	shortAddr := addr[12:20]
	v, isFound, _, _, _, err := self.Storage.accountStorage.Get(shortAddr)
	if err != nil {
		log.Info("getOrCreateAccount", "acct", addr, "shortAddr", shortAddr, "status", "ERROR")
		return a, err
	}

	//Tier2 : Lookup from rs
	if isFound {
		encoded, ok, err := self.ChunkStore.GetChunk(v)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, nil
		}
		var acct Account
		err = rlp.Decode(bytes.NewReader(encoded), &acct)
		if err != nil {
			return nil, err
		}
		log.Debug("AccountObject", "account", addr, "shortAddr", shortAddr, "acct", acct.String())
		return NewAccountObject(self, addr, &acct), nil
	} else {
		//Tier 3: createAccount if still not found
		log.Debug("createAccount", "acct", addr, "status", "canOpen")
		acct := Account{
			Tokens:       []uint64{},
			Denomination: big.NewInt(0),
			Balance:      big.NewInt(0),
		}
		return NewAccountObject(self, addr, &acct), nil
	}
}

func (self *StateDB) updateAccountObject(a *accountObject) (err error) {
	log.Debug("updateAccountObject", "a", a)
	encoded, _ := rlp.EncodeToBytes(a)
	log.Debug("updateAccountObject: EncodeToBytes", "ENCODED", common.Bytes2Hex(encoded))
	v := deep.Keccak256(encoded)
	err = self.ChunkStore.SetChunk(v, encoded)

	if err != nil {
		log.Info("updateAccountObject: StoreChunk", "v", v, "ERROR", err)
		return err
	} else {
		log.Debug("updateAccountObject: StoreChunk", "v", v, "encoded", encoded)
	}
	shortAddr := a.address[12:20]
	if a.deleted {
		//token marked for deletion due to finalized exit
		err = self.Storage.accountStorage.Delete(shortAddr)
	} else {
		err = self.Storage.accountStorage.Insert(shortAddr, v, 0, 0)
	}

	if err != nil {
		log.Info("updateAccountObject", "a", a, "ERROR", err)
		return err
	}
	return nil
}

// modeled after acct trie
func (self *StateDB) getChainObject(chainID uint64) (c *chainObject, err error) {
	v, found, _, _, _, err := self.Storage.chainStorage.Get(deep.UInt64ToByte(chainID))
	if err != nil {
		return nil, err
	}
	if found == false {
		return nil, fmt.Errorf("Not found")
	}
	encoded, ok, err := self.ChunkStore.GetChunk(v)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}
	var anb anchorBlock
	err = rlp.Decode(bytes.NewReader(encoded), &anb)
	if err != nil {
		log.Info("getChainObject: Decode", "ChainID", chainID, "v", v, "ERR", err)
		return nil, err
	}
	log.Debug("getChainObject", "ChainID", chainID, "AnchorBlock", anb.String())
	return NewChainObject(self, chainID, &anb), nil
}

// deprecated
func (self *StateDB) createChain(chainID uint64) (c *chainObject, err error) {
	_, isFound, _, _, _, err := self.Storage.chainStorage.Get(deep.UInt64ToByte(chainID))
	if err != nil {
		log.Info("createChain", "ChainID", chainID, "status", "ERROR")
		return c, err
	} else if isFound {
		log.Debug("createChain", "ChainID", chainID, "status", "Already Exist")
		return c, fmt.Errorf("shortAddr already exist")
	}

	log.Debug("createChain", "ChainID", chainID, "status", "canRegister")
	anb := anchorBlock{
		Owners:     []*common.Address{},
		StartBlock: 0,
		EndBlock:   0,
	}
	return NewChainObject(self, chainID, &anb), nil
}

/* WARNING:
getOrCreateChain behave differently than the getOrCreateAccount!!
New chainObject is initiated without owners. Any in-memory chainObject with
endBlock > 0 && leng(owners) > 0 must evaluated to False */

func (self *StateDB) getOrCreateChain(chainID uint64) (c *chainObject, err error) {

	//Tier1 : Lookup from current state
	if currentChainObject, isSet := self.chainObjects[chainID]; isSet {
		log.Debug("getOrCreateChain", "must go in here!!!", chainID, self.chainObjects[chainID])
		return currentChainObject, nil
	}

	v, isFound, _, _, _, err := self.Storage.chainStorage.Get(deep.UInt64ToByte(chainID))
	if err != nil {
		log.Info("getOrCreateChain", "chainID", chainID, "status", "ERROR")
		return c, err
	}

	//Tier2 : Lookup from rs
	if isFound {
		encoded, ok, err := self.ChunkStore.GetChunk(v)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, nil
		}
		var anb anchorBlock
		log.Debug("ChainObject Found ", "chainID", chainID, "encoded", common.Bytes2Hex(encoded))
		err = rlp.Decode(bytes.NewReader(encoded), &anb)
		if err != nil {
			return nil, err
		}
		log.Debug("ChainObject", "chainID", chainID, "anchorBlock", anb.String())
		return NewChainObject(self, chainID, &anb), nil
	} else {
		//Tier 3: registerChain if still not found
		log.Debug("createChain", "chainID", chainID, "status", "canRegister")
		anb := anchorBlock{
			Owners:     []*common.Address{},
			StartBlock: 0,
			EndBlock:   0,
		}
		return NewChainObject(self, chainID, &anb), nil
	}
}

func (self *StateDB) updateChainObject(c *chainObject) (err error) {
	log.Debug("updateChainObject", "c", c)
	encoded, _ := rlp.EncodeToBytes(c)
	log.Debug("updateChainObject: EncodeToBytes", "ENCODED", common.Bytes2Hex(encoded))
	v := deep.Keccak256(encoded)
	err = self.ChunkStore.SetChunk(v, encoded)

	if err != nil {
		log.Info("updateChainObject: StoreChunk", "v", v, "ERROR", err)
		return err
	} else {
		log.Debug("updateChainObject: StoreChunk", "v", v, "encoded", encoded)
	}

	chainID := deep.UInt64ToByte(c.BlockchainID())
	if c.deleted {
		//TODO: remove closed layer3chain
		err = self.Storage.chainStorage.Delete(chainID)
	} else {
		log.Debug("updateChainObject: Insert", "chainID", chainID, "v", v)
		err = self.Storage.chainStorage.Insert(chainID, v, 0, 0)
	}

	if err != nil {
		log.Info("updateChainObject", "c", c, "ERROR", err)
		return err
	}
	return nil
}

// Object accessors

func (self *StateDB) GetToken(tokenID uint64) (t *Token, err error) {
	tokenObject, err := self.getTokenObject(tokenID)
	if err != nil {
		return t, err
	}
	return tokenObject.token, nil
}

func (self *StateDB) GetAccount(addr common.Address) (a *Account, err error) {
	accountObject, err := self.getAccountObject(addr)
	if err != nil {
		return a, err
	}
	return accountObject.acct, nil
}

func (self *StateDB) GetChain(chainID uint64) (a *anchorBlock, err error) {
	chainObject, err := self.getChainObject(chainID)
	if err != nil {
		return a, err
	}
	return chainObject.anb, nil
}
