// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package plasmachain

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wolkdb/go-plasma/deep"
	"github.com/wolkdb/go-plasma/smt"
)

type PlasmaChain struct {
	connWS          *ethclient.Client
	session         *RootChainSession
	operatorKey     *ecdsa.PrivateKey
	plasmatxpool    *PlasmaTxPool
	currentBlock    *Block
	blockNumber     uint64
	DefaultHashes   [smt.TreeDepth][]byte
	chainType       string
	networkId       uint64
	blockchainID    uint64
	config          *Config
	ChainConfig     *params.ChainConfig
	ChunkStore      deep.StorageLayer
	Storage         *Storage
	shutdownChan    chan bool // Channel for shutting down plasma
	protocolManager *ProtocolManager
	eventMux        *event.TypeMux
	accountManager  *accounts.Manager
	ApiBackend      *PlasmaApiBackend
	scope           event.SubscriptionScope
	chainFeed       event.Feed
	chainSideFeed   event.Feed
	chainHeadFeed   event.Feed
	lock            sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

// New creates a new Plasma object (including the initialisation of the common Plasma object)
func New(ctx *node.ServiceContext, config *Config, simulated bool) (*PlasmaChain, error) {

	var self PlasmaChain

	self.config = config
	err := self.setRootContract(self.config.RootContractAddr, self.config.L1rpcEndpointUrl, self.config.L1wsEndpointUrl, self.config.operatorKey)
	if err != nil {
		if self.config.UseLayer1 {
			log.Error("setRootContract", "error", err, "Action", "Terminated")
			return nil, err
		} else {
			log.Info("setRootContract", "error", err, "Action", "Proceed")
		}
	}

	self.eventMux = new(event.TypeMux) // SHOULD BE: ctx.EventMux
	self.DefaultHashes = smt.ComputeDefaultHashes()

	self.shutdownChan = make(chan bool)
	self.networkId = config.NetworkId
	self.chainType = "plasma"
	self.blockchainID = 0

	self.ChunkStore, err = deep.NewRemoteStorage(self.blockchainID, config, self.chainType, self.operatorKey)
	self.Storage = NewStorage(self.ChunkStore)
	self.plasmatxpool = &PlasmaTxPool{}

	if self.protocolManager, err = NewProtocolManager(config, config.NetworkId, self.eventMux, &self); err != nil {
		return nil, fmt.Errorf("NewProtocolManager: %v", err)
	}
	log.Info("Initializing Plasma protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	self.ApiBackend = &PlasmaApiBackend{&self}

	//TODO: removing loadLastState from new
	if err := self.loadLastState(simulated); err != nil {
		return nil, err
	}
	log.Info("Loaded last state")

	return &self, nil
}

/* Node Service */
/* ========================================================================== */

func (s *PlasmaChain) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

func (s *PlasmaChain) Start(srvr *p2p.Server) error {
	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	return nil
}

func (s *PlasmaChain) Stop() error {
	log.Info("Blockchain manager stopped")

	s.protocolManager.Stop()
	//	close(s.shutdownChan)
	return nil
}

/* ########################################################################## */

/* Method Implemented for Deep Interface  */
/* ========================================================================== */

func (s *PlasmaChain) GetBlockChainID() uint64 {
	return s.blockchainID
}

func (self *PlasmaChain) StateAt(headerHash common.Hash) (stateDB deep.StateDB, err error) {
	return NewStateDB(self.ChunkStore, headerHash, self.operatorKey), nil
}

func (self *PlasmaChain) ApplyTransaction(stateDB deep.StateDB, rtx deep.Transaction) (err error) {
	log.Debug("Preparing Block", "Block Number", self.blockNumber+1)
	log.Debug("ApplyTransaction Reflect", "TYPE", reflect.TypeOf(rtx))
	s := stateDB.(*StateDB)

	switch rtx.(type) {
	case *deep.AnchorTransaction:
		err = self.ApplyAnchorTransaction(s, rtx.(*deep.AnchorTransaction))
	case *Transaction:
		err = self.ApplyPlasmaTransaction(s, rtx.(*Transaction))
	default:
	}

	if err != nil {
		return err
	}
	return nil
}

func (self *PlasmaChain) ValidateBody(b deep.Block) (err error) {
	var block Block
	block = b.(Block)
	_, err = block.ValidateBody()
	return err
}

func (self *PlasmaChain) CurrentBlock() (b deep.Block) {
	return self.currentBlock
}

func (self *PlasmaChain) GetBlock(headerHash common.Hash, number uint64) (b deep.Block, err error) {
	return self.getBlockByNumber(number)
}

func (self *PlasmaChain) HasBlock(h common.Hash, number uint64) bool {
	data, err := self.getBlockByNumber(number)
	if err != nil {
		log.Info("Error", "err", err)
		return false
	}
	if data == nil {
		return false
	}
	return true
}

func (self *PlasmaChain) GetBlockByHash(h common.Hash) (b deep.Block, err error) {
	key := h.Bytes()
	val, ok, err := self.ChunkStore.GetChunk(key)
	if err != nil {
		log.Info("Error", "err", err)
		return nil, err
	} else if !ok {
		log.Info("PlasmaChain:GetBlockByHash | chunk not found", "err", err)
		return nil, nil
	} else if len(val) == 0 {
		log.Info("PlasmaChain:GetBlockByHash | chunk found but empty", "err", err)
		return nil, nil
	}
	block := FromChunk(val)
	return block, nil
}

func (self *PlasmaChain) GetBlockByNumber(blockNumber rpc.BlockNumber) (b deep.Block, err error) {
	return self.getBlockByNumber(uint64(blockNumber))
}

func (self *PlasmaChain) InsertChain(chain deep.Blocks) (int, error) {
	n, events, err := self.insertChain(chain)
	self.PostChainEvents(events)
	return n, err
}

//may be able to be optimized
func (self *PlasmaChain) InsertChainForPOA(chain deep.Blocks) (int, error) {

	blocknumber := self.blockNumber
	var lastblock deep.Block
	for _, block := range chain {
		blocknumber = blocknumber + 1
		lastblock = block
	}

	//currentHeadBlockHash := GetCanonicalHash(self.Cloudstore, blocknumber, self.blockchainID)
	//log.Info(fmt.Sprintf("backend.go:insertChain | blockHead Canonical Hash is: %+v", currentHeadBlockHash))

	//WriteHeadBlockHash(self.Cloudstore, currentHeadBlockHash, self.blockchainID)
	self.currentBlock = lastblock.(*Block)
	//self.blockNumber = blocknumber
	self.blockNumber = self.currentBlock.Number()
	log.Info("🔨 block to mine", "self.currentBlock number", self.currentBlock.Number(), "self.blockNumber", self.blockNumber)
	if self.config.UseLayer1 {
		self.publishBlock(self.currentBlock.header.TransactionRoot, self.currentBlock.Number(), true)
		log.Info("Rootchain >> publishBlock", "block#", self.currentBlock.Number(), "TransactionRoot", self.currentBlock.header.TransactionRoot)
	}
	return 0, nil
}

func (self PlasmaChain) SetHead(hash common.Hash) (err error) {
	// TODO
	return nil
}

func (self PlasmaChain) SetHeadBlock(headBlock deep.Block) (err error) {
	// TODO
	return nil
}

func (self *PlasmaChain) SubscribeChainHeadEvent(ch chan<- deep.ChainHeadEvent) (sub event.Subscription) {
	return self.scope.Track(self.chainHeadFeed.Subscribe(ch))
}

func (self *PlasmaChain) MakeBlock(parentHash common.Hash, dhdr deep.Header, bn uint64, inputtxs []deep.Transaction) (bl deep.Block, err error) {
	hdr := dhdr.(*Header)

	var plasmaTxs []*Transaction
	var anchorTxs []*deep.AnchorTransaction

	for _, t := range inputtxs {
		switch t.(type) {
		case *Transaction:
			plasmaTxs = append(plasmaTxs, t.(*Transaction))
		case *deep.AnchorTransaction:
			anchorTxs = append(anchorTxs, t.(*deep.AnchorTransaction))
		default:
		}
	}

	body := &Body{
		Layer2Transactions: plasmaTxs,
		AnchorTransactions: anchorTxs,
	}
	hdr.SignHeader(self.operatorKey)

	block := &Block{
		header: hdr,
		body:   body,
	}
	block.SignBlock(self.operatorKey)
	return block, nil
}

func (self *PlasmaChain) WriteBlock(block deep.Block) (err error) {
	return WriteBlock(self.ChunkStore, block)
}

func (self *PlasmaChain) ChainType() string {
	return self.chainType
}

func (self *PlasmaChain) GetStorageLayer() deep.StorageLayer {
	return self.ChunkStore
}

func (self *PlasmaChain) LatestBlockNumber() (uint64, error) {
	if self.currentBlock == nil {
		return 0, fmt.Errorf("No current Block!")
	}
	return self.currentBlock.Number(), nil
}

/* ########################################################################## */

/* Backend and Blockchain Interface */
/* ========================================================================== */
func (self *PlasmaChain) TxPool() deep.TxPool {
	return self.plasmatxpool
}

func (self *PlasmaChain) BlockChain() deep.BlockChain {
	return self
}

/* ########################################################################## */

/* TxPool  */
/* ========================================================================== */

func (self *PlasmaChain) validateTx(rtx deep.Transaction) (err error) {
	switch rtx.(type) {
	case *deep.AnchorTransaction:
		//Validate ExtraData only (does not verify blockchain ownership)
		err = rtx.(*deep.AnchorTransaction).ValidateAnchor()
	case *Transaction:
		//Validate Signiture only (does not verify token state ownership)
		err = rtx.(*Transaction).ValidateSig()
	default:

	}

	if err != nil {
		return err
	}
	return nil
}

func (self *PlasmaChain) validateAnchorTx(anchorTx *deep.AnchorTransaction) (err error) {
	return nil
}

func (self *PlasmaChain) addTransactionToPool(tx *Transaction) (err error) {
	return self.plasmatxpool.addTransactionToPool(tx)
}

func (self *PlasmaChain) addAnchorTransactionToPool(tx *deep.AnchorTransaction) (err error) {
	return self.plasmatxpool.addAnchorTransactionToPool(tx)
}

/* ########################################################################## */

/* ApplyTransaction for State  */
/* ========================================================================== */

func (self *PlasmaChain) ApplyAnchorTransaction(s *StateDB, anchorTx *deep.AnchorTransaction) (err error) {

	var chainID = anchorTx.BlockChainID
	var dirtied = false //useful for debuging
	var chainObj *chainObject
	log.Debug("Anchor TX: apply", "TX", anchorTx)
	log.Debug("currentChainObject", "memory", s.chainObjects)
	chainObj, err = s.getOrCreateChain(chainID)
	if err != nil {
		return err
	}

	//TODO: validate signer from ownerlist. Prevent Anchor0 spam

	if anchorTx.BlockNumber == 0 {
		//Register Layer3 Block
		log.Debug("Anchor 0", "ChainID", chainID, "Txhash", anchorTx.Hash(), "TX", anchorTx.String())
		if chainObj.OwnerCount() > 0 || chainObj.EndBlock() > 0 {
			return fmt.Errorf("Invalid Anchor 0")
		}
		signer, err := anchorTx.GetSigner()
		if err != nil {
			return err
		}

		dirtied = true
		chainObj.AddOwner(signer)
		log.Debug("Anchor 0: Add default owner", "ChainID", chainID, "Owner", signer.Hex())

		Onwership := anchorTx.Extra

		if len(Onwership.RemovedOwners) > 0 {
			return fmt.Errorf("Can't remove user with Anchor 0 ")
		}

		for _, newOwner := range Onwership.AddedOwners {
			if !chainObj.Existed(newOwner) {
				dirtied = true
				chainObj.AddOwner(newOwner)
				log.Info("Add New Owner", "ChainID", chainID, "Seq#", chainObj.OwnerCount(), "Addr", signer.Hex())
			} else {
				log.Info("Owner Already Exist", "ChainID", chainID, "Addr", signer.Hex())
			}
		}

	} else {
		// Normal Anchor Txn
		log.Debug("Anchor", "ChainID", chainID, "BN", anchorTx.BlockNumber, "Txhash", anchorTx.Hash(), "TX", anchorTx.String())
		if chainObj.OwnerCount() == 0 {
			return fmt.Errorf("Unregistered Chain")
		}

		if chainObj.EndBlock() == anchorTx.BlockNumber {
			return fmt.Errorf("Attempted ReAnchoring [#%v]", anchorTx.BlockNumber)
		}

		if chainObj.EndBlock()+1 != anchorTx.BlockNumber {
			return fmt.Errorf("Jumped Sequence: Last Block [#%v] >>> Anchor [#%v]", chainObj.EndBlock(), anchorTx.BlockNumber)
		}

		signer, _ := anchorTx.GetSigner()
		if !chainObj.Existed(signer) {
			return fmt.Errorf("Invalid Signer: %x", signer)
		}

		Onwership := anchorTx.Extra
		for _, newOwner := range Onwership.AddedOwners {
			if !chainObj.Existed(newOwner) {
				dirtied = true
				chainObj.AddOwner(newOwner)
				log.Info("Add New Owner", "ChainID", chainID, "Seq#", chainObj.OwnerCount(), "Addr", newOwner.Hex())
			} else {
				log.Info("[AddOwner] Owner Already Exist", "ChainID", chainID, "Addr", newOwner.Hex())
			}
		}

		for _, addr := range Onwership.RemovedOwners {
			if chainObj.Existed(addr) {
				if chainObj.OwnerCount() == 1 {
					log.Info("Trying to Remove Last Owner", "ChainID", chainID, "Total Owner", chainObj.OwnerCount(), "Addr", addr.Hex())
					return fmt.Errorf("Can not remove only owner")
				}
				dirtied = true
				chainObj.RemoveOwner(addr)
				log.Info("Remove Owner", "ChainID", chainID, "Total Owner", chainObj.OwnerCount(), "Addr", addr.Hex())
			} else {
				log.Info("[RemoveOwner] Owner Doesn't Exist", "ChainID", chainID, "Addr", addr.Hex())
			}
		}
	}

	log.Debug("Anchor TX: apply", "dirtied?", dirtied)

	//Anchor Range

	log.Debug("Set Prevblock", "bn", self.blockNumber+1)

	if _, isSet := s.chainIsSet[anchorTx.BlockChainID]; !isSet {
		//Flagin the startBlock per each chain
		log.Debug("Anchor TX: apply", "Status", "First AnchorTx within this block")
		chainObj.SetStartBlock(anchorTx.BlockNumber)
		chainObj.SetEndBlock(anchorTx.BlockNumber)
	} else {
		log.Debug("Anchor TX: apply", "Status", "Additional AnchorTx within this block")
		chainObj.SetEndBlock(anchorTx.BlockNumber)
	}
	s.AppendAnchor(chainID, chainObj)
	log.Info("chainObjects", "chain", chainObj)
	// WARNING: AnchorTx is not loaded to chainStorage

	// quesionable logic..
	var aobj *anchorBlock

	if _, isSet := s.atxList[chainID]; isSet {
		aobj = s.atxList[chainID]
		aobj.AddAnchorTx(anchorTx)
	} else {
		aobj = NewAnchorBlock(s, anchorTx)
	}
	s.atxList[chainID] = aobj
	err = self.StoreAnchorTransaction(anchorTx, self.blockNumber+1)
	if err != nil {
		log.Info("Anchor TX: apply", "Status", "StoreAnchorTransaction failed!!")
		return err
	}
	return nil
}

func (self *PlasmaChain) StoreAnchorTransaction(tx *deep.AnchorTransaction, blockNumber uint64) (err error) {

	key := tx.Hash().Bytes()
	txbytes := tx.Bytes()

	/*
		var tx2 *deep.AnchorTransaction
		_ = rlp.Decode(bytes.NewReader(txbytes), &tx2)
		log.Info("PlasmaChunkstore:StoreAnchorTransaction", "Decode", tx2)
	*/

	//_, tx2 := deep.BytesToAnchorTransaction(txbytes)
	//enc1, _ := rlp.EncodeToBytes(&tx.Extra)
	//log.Info("PlasmaChunkstore:StoreAnchorTransaction", "Encode", enc1)

	enc, err := rlp.EncodeToBytes(&tx)
	if err != nil {
		log.Info("PlasmaChunkstore:StoreAnchorTransaction", "Encode", common.Bytes2Hex(enc), "ERR", err)
	} else {
		log.Info("PlasmaChunkstore:StoreAnchorTransaction", "Encode", common.Bytes2Hex(enc))
	}

	err = self.ChunkStore.SetChunk(key, txbytes)
	log.Info("PlasmaChunkstore:StoreAnchorTransaction", "key", common.Bytes2Hex(key), "txbytes", common.Bytes2Hex(txbytes))
	if err != nil {
		return err
	}

	// Write blockNumber
	key2 := append([]byte("bn"), tx.Hash().Bytes()...)
	err = self.ChunkStore.SetChunk(key2, deep.UInt64ToByte(blockNumber))
	if err != nil {
		return err
	}
	return nil
}

func (self *PlasmaChain) ApplyPlasmaTransaction(s *StateDB, tx *Transaction) (err error) {

	var sobj *tokenObject
	var senderobj *accountObject
	var recipientobj *accountObject

	var dirtied = false
	var senderdirtied = false
	log.Debug("tx to apply", "txhash", tx.Hash(), "txn", tx.String(), "txbyte", common.Bytes2Hex(tx.Bytes()))

	senderobj, err = s.getOrCreateAccount(*tx.PrevOwner)
	recipientobj, err = s.getOrCreateAccount(*tx.Recipient)
	if err != nil {
		return err
	}

	if tx.PrevBlock == 0 {
		// token deposit
		log.Debug("Deposit", "tokenID", tx.TokenID, "Denomination", tx.Denomination, "Recipient", *tx.Recipient, "txh", tx.Hash())
		sobj, err = s.createDeposit(tx.TokenID, tx.Denomination, *tx.Recipient)
		if err != nil {
			return err
		}
		// Add token to acct tokenList and update balance
		recipientobj.AddToken(tx.TokenID, new(big.Int).SetUint64(sobj.Balance()), new(big.Int).SetUint64(sobj.Denomination()))
		dirtied = true

	} else {
		//token transfer
		sobj, err = s.getTokenObject(tx.TokenID)
		if err != nil {
			log.Info("Token Transfer Problem 2", "tokenID", tx.TokenID, "err", err)
			return err
		}

		if sobj.PrevBlock() != tx.PrevBlock {
			//Invalid PrevBlock
			if sobj.PrevBlock() == 0 {
				// extremely rare: this is the true genesis
				if sobj.Denomination() > 0 && !sobj.deleted {
					//special case for first transfer, allow it bypass prevblock check if token does exists
				} else {
					log.Info("Token Transfer Problem 2a", "sobj prevBlock", sobj.PrevBlock(), "tx PrevBlock", tx.PrevBlock)
					return fmt.Errorf("Invalid fisrt transfer")
				}
			} else {
				log.Info("Token Transfer Problem 2b", "sobj prevBlock", sobj.PrevBlock(), "tx PrevBlock", tx.PrevBlock)
				return fmt.Errorf("Invalid PrevBlock")
			}
		}

		if sobj.Owner() != *tx.PrevOwner {
			//Invalid Owner
			log.Info("Token Transfer Problem 3", "sobj owner", sobj.Owner(), "tx prevowner", *tx.PrevOwner)
			return fmt.Errorf("Unauthorized token transfer")
		}

		if !sobj.ValidDemination() || (sobj.Denomination() != tx.Denomination) || (tx.Denomination != tx.Spent+tx.Allowance+tx.balance) {
			//Invalid token amount
			log.Info("Token Transfer Problem 4", "sobj", sobj.ValidDemination(), "denom", sobj.Denomination() == tx.Denomination, "tx", tx.Denomination == tx.Spent+tx.Allowance+tx.balance)
			return fmt.Errorf("Invalid token amount")
		}

		log.Info("ApplyTransaction STATE OBJECT", "sobj", sobj)
		if sobj.Owner() != *tx.Recipient {
			//Ownership transfer
			log.Debug("Change STATE OBJECT", "owner", tx.Recipient)
			sobj.SetOwner(*tx.Recipient)
			recipientobj.AddToken(tx.TokenID, new(big.Int).SetUint64(sobj.Balance()), new(big.Int).SetUint64(sobj.Denomination()))
			senderobj.RemoveToken(tx.TokenID, new(big.Int).SetUint64(sobj.Balance()), new(big.Int).SetUint64(sobj.Denomination()))
			dirtied = true
			senderdirtied = true
		}

		if tx.Spent > sobj.Spent() {
			//operator Withdrawal, which is
			log.Debug("Change STATE OBJECT", "Spent", tx.Spent)
			if tx.Spent+tx.Allowance != sobj.Spent()+sobj.Allowance() {
				log.Info("Token Transfer Problem 5", "Spent", tx.Spent)
				//attempt to modify {allowance, spent, balance} at the same time
				//TODO: retrun proper error msg
				//return err
			}
			sobj.SetWithdrawal(tx.Allowance, tx.Spent)
			//withdraw is independent of acct trie
			dirtied = true
		}

		if tx.balance != sobj.Balance() {
			//bandwidth consumption
			consumption := sobj.Balance() - tx.balance
			log.Debug("Change STATE OBJECT", "balanceupdate", tx.balance)
			if tx.balance+tx.Allowance != sobj.Balance()+sobj.Allowance() {
				log.Info("Token Transfer Problem 6", "balanceupdate", tx.Spent)
				//attempt to modify {allowance, spent, balance} at the same time
				//TODO: retrun proper error msg
				//return err
			}
			sobj.SetBalance(tx.balance, tx.Allowance)

			//recipientBal := recipientobj.Balance() - consumption
			recipientBal := new(big.Int).Sub(recipientobj.Balance(), new(big.Int).SetUint64(consumption))
			recipientobj.SetAccountBalance(recipientBal)
			dirtied = true
		}
	}

	//verify each token can only have 1 transaction per SMT block
	if _, isSet := s.tokenIsSet[tx.TokenID]; isSet {
		if s.tokenIsSet[tx.TokenID] {
			return fmt.Errorf("tokenID %x has more than 1 transaction per SMT block", tx.TokenID)
		}
	}

	//modify tokenstorage state if dirtied
	if dirtied {
		log.Debug("Set Prevblock", "bn", self.blockNumber+1)
		sobj.SetPrevBlock(self.blockNumber + 1)
		s.Append(tx.TokenID, sobj, tx.Hash())
		s.accountObjects[*tx.Recipient] = recipientobj
		if senderdirtied {
			s.accountObjects[*tx.PrevOwner] = senderobj
		}

		if err != nil {
			return err
		}
		//modify transactionStorage Trie
	} else {
		//non-dirty transaction is self-anchoring.
		//which is NOT allowed for now.
		return fmt.Errorf("Self-anchoring not allowed for now")
	}

	//modify Transactions trie
	err = self.StoreTransaction(tx, self.blockNumber+1)
	if err != nil {
		return err
	} else {
		log.Debug("StoreTransaction", "txh", tx.Hash(), "tx", tx)
	}
	return nil
}

func (self *PlasmaChain) StoreTransaction(tx *Transaction, blockNumber uint64) (err error) {
	key := tx.Hash().Bytes()
	txbytes := tx.Bytes()
	err = self.ChunkStore.SetChunk(key, txbytes)
	log.Debug("PlasmaChain:StoreTransaction", "key", common.Bytes2Hex(key), "txbytes", common.Bytes2Hex(txbytes))
	if err != nil {
		return err
	}

	// Write blockNumber
	key2 := append([]byte("bn"), tx.Hash().Bytes()...)
	err = self.ChunkStore.SetChunk(key2, deep.UInt64ToByte(blockNumber))
	log.Debug("plasmachain:StoreTransaction", "key2", common.Bytes2Hex(key2), "blockNumber", deep.UInt64ToByte(blockNumber))
	if err != nil {
		return err
	}

	//write token deposit blockNumber
	if tx.PrevBlock == 0 {
		key3 := append([]byte("plasmaDeposit"), deep.UInt64ToByte(tx.TokenID)...)
		err = self.ChunkStore.SetChunk(key3, deep.UInt64ToByte(blockNumber))
		log.Debug("plasmachain:StoreTransaction", "key3", common.Bytes2Hex(key3), "blockNumber", deep.UInt64ToByte(blockNumber))
		if err != nil {
			return err
		}

		tokenID, tinfo := TokenInit(tx.DepositIndex, tx.Denomination, *tx.Recipient)
		if tokenID != tx.TokenID {
			return fmt.Errorf("Invalid tokenID key: %x\n", tokenID)
		}

		infoByte, err := rlp.EncodeToBytes(&tinfo)
		key4 := deep.Keccak256([]byte(fmt.Sprintf("plasmaTokenInfo%x", tokenID)))
		log.Debug("PlasmaChain:StoreTransaction", "key4", common.Bytes2Hex(key4), "infoByte", common.Bytes2Hex(infoByte))
		err = self.ChunkStore.SetChunk(key4, infoByte)
		if err != nil {
			return err
		}
	}
	return nil
}

/* ########################################################################## */

/* Methods Implemented for Minter interface  */
/* ========================================================================== */

func WriteBlock(db deep.StorageLayer, block deep.Block) error {
	log.Info(fmt.Sprintf("Attempting to Write Block: %+v", block))
	header := block.Header()

	WriteHeader(db, header)
	//body := block.Body()
	//WriteBody(db, block.Hash(), block.Number(), body)

	BlockKey := deep.Keccak256([]byte(fmt.Sprintf("plasma%d", block.Number())))
	data, err := block.Encode()
	if err != nil {
		log.Info("WriteBlock", "ERROR", err)
	}

	var bdecoded *Block
	_ = rlp.Decode(bytes.NewReader(data), &bdecoded)
	//log.Debug("backend :WriteBlock", "Decode", bdecoded)

	//log.Debug(fmt.Sprintf("WriteBlock with key of %x", BlockKey), "Data", common.Bytes2Hex(data))
	err = db.SetChunk(BlockKey, data)
	if err != nil {
		return err
	} else {
		log.Debug(fmt.Sprintf("WriteBlock key: %x block: %+v", BlockKey, block))
	}
	return nil
}

func WriteHeader(db deep.StorageLayer, header deep.Header) error {
	headerKey := deep.Keccak256([]byte(fmt.Sprintf("plasmaHeader%d", header.Number())))
	headerhash := header.Hash().Bytes()
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		log.Info(fmt.Sprintf("Error Encoding Bytes %+v", err))
	}

	err = db.SetChunk(headerKey, headerhash)
	if err != nil {
		log.Info("Error Storing Chunk", "error", err)
		return err
	}
	err = db.SetChunk(headerhash, data)
	if err != nil {
		log.Info("Error Storing Chunk", "error", err)
		return err
	}
	log.Debug("WriteHeader", "headerKey", common.Bytes2Hex(headerKey), "headerhash", common.Bytes2Hex(headerhash), "RLP", common.Bytes2Hex(data))
	//WriteCanonicalHash(db, header.Hash(), header.Number())
	return nil
}

/* Write the whole block instead
func (self *PlasmaChain) WriteBody(hash common.Hash, number uint64, body deep.Body) {
	if value, ok := body.(*Block); ok {
		data, err := rlp.EncodeToBytes(value)
		bodyhash := crypto.Keccak256(data[:])
		if err != nil {
		}
		self.RemoteStorage.StoreChunk(bodyhash, data)
	}
}

func WriteBody(db deep.RemoteStorage, hash common.Hash, number uint64, body deep.Body) {
	data, err := rlp.EncodeToBytes(body)
	bodyhash := crypto.Keccak256(data[:])
	db.SetChunk(bodyhash, data)
	if err != nil {
		log.Info("Storing Body failed", "data", data)
	}
	log.Info("Storing body", "body", body, "hash", hash, "number", number, "hashbytes", hash.Bytes())
}

*/

func (bc *PlasmaChain) PostChainEvents(events []interface{}) {
	log.Debug(fmt.Sprintf("PostChainEvents %v", events))
	/*
		// post event logs for further processing
		if logs != nil {
			bc.logsFeed.Send(logs)
		}
	*/
	for _, event := range events {
		switch ev := event.(type) {
		case deep.ChainEvent:
			bc.chainFeed.Send(ev)

		case deep.ChainHeadEvent:
			bc.chainHeadFeed.Send(ev)

			//case deep.ChainSideEvent:
			//	bc.chainSideFeed.Send(ev)
		}
	}
}

func (self *PlasmaChain) ProcessPlasmaBlock(block *Block) (err error) {
	log.Debug("[(self *PlasmaChain) ProcessPlasmaBlock]", "blockNumber", block.header.BlockNumber)
	self.blockNumber = block.header.BlockNumber
	// TODO: validate sig

	// remove txs from plasmaTxPool
	covered := make(map[common.Hash]bool)
	for _, tx := range block.body.Layer2Transactions {
		covered[tx.Hash()] = true
	}
	for _, tx := range block.body.AnchorTransactions {
		covered[tx.Hash()] = true
	}
	newtxpool := make([]*Transaction, 0)
	for _, tx := range self.plasmatxpool.txpool {
		if covered[tx.Hash()] {

		} else {
			newtxpool = append(newtxpool, tx)
		}
	}
	self.plasmatxpool.txpool = newtxpool
	newwtxpool := make([]*deep.AnchorTransaction, 0)
	for _, tx := range self.plasmatxpool.wtxpool {
		if covered[tx.Hash()] {

		} else {
			newwtxpool = append(newwtxpool, tx)
		}
	}
	self.plasmatxpool.wtxpool = newwtxpool
	return err
}

func (self *PlasmaChain) insertChain(chain deep.Blocks) (int, []interface{}, error) {
	//currentHeadHash := GetHeadBlockHash(self.Chunkstore)
	//currentHead, err := self.GetBlockByHash(currentHeadHash)
	//TODO
	//currentHead := self.currentBlock
	headers := make([]*Header, len(chain))
	seals := make([]bool, len(chain))
	events := make([]interface{}, 0, len(chain))
	var lastCanon *Block

	for i, b := range chain {
		block := CopyBlock(b.(*Block))
		if h, ok := block.Header().(*Header); ok {
			headers[i] = h
			seals[i] = true

			var parent *Block
			log.Debug(fmt.Sprintf("InsertChain %d %v ", block.NumberU64(), block))
			if i == 0 {
				bl, _ := self.GetBlock(block.ParentHash(), block.NumberU64()-1)
				if b, ok := bl.(*Block); ok {
					parent = b
				}
			} else {
				//parent = chain[i-1]
				parent = lastCanon
			}
			log.Debug(fmt.Sprintf("parent = %v", parent))

			// alias state.New because we introduce a variable named state on the next line
			//stateNew := state.New
			/*
				operatorKey, _ := crypto.HexToECDSA(operatorPrivateKey)
				state := NewStateDB(self.RemoteStorage, parent.header.Hash(), operatorKey)
				state.Init(parent.Root())
			*/
			//TODO: validate block More using the state retrieved here

			//state.Init(parent.NumberU64())
			//Doing this in minter.go - mintNewBlock Already : smhash, err := state.Commit(self.Chunkstore, h.BlockNumber, h.ParentHash)
			// do something for smhash
			//log.Debug("smhash = ", smhash)
			self.WriteBlock(block)
			//                        currentHead = block

			// store smhash somewhere
			events = append(events, deep.ChainHeadEvent{block})
		}
		lastCanon = block
	}
	log.Debug(fmt.Sprintf("lastCanon = %v", lastCanon))
	/*
	   if lastCanon != nil && self.LastBlockHash() == lastCanon.Hash() {
	           events = append(events, deep.ChainHeadEvent{lastCanon})
	   }
	*/
	//WriteHeadBlockHash(self.Chunkstore, currentHead.Hash())
	self.currentBlock = lastCanon
	//	self.PostChainEvents(events)
	// if self.config.UseLayer1 {
	// 	self.publishBlock(self.currentBlock.TransactionRoot, self.currentBlock.BlockNumber, true)
	// 	log.Info("Rootchain >> publishBlock", "block#", self.currentBlock.Number(), "TransactionRoot", self.currentBlock.TransactionRoot)
	// }
	return 0, events, nil
}

/*
func GetCanonicalHash(db cloud.Cloudstore, number uint64, cid uint64) (headerHash common.Hash) {
	chainname := append(namePrefix, ChainName...)
	chainid := append(chainPrefix, encodeBlockNumber(cid)...)
	key := append(append(append(append(chainname, chainid...), headerPrefix...), encodeBlockNumber(number)...), numSuffix...)
	log.Info("Getting Canonical Hash", "headerPrefix", headerPrefix, "number", number, "encBN", encodeBlockNumber(number), "numsuffix", numSuffix, "key", key)
	headerData, err := db.GetChunk(key)
	if len(headerData) == 0 {
		log.Info("Retrieved empty header chunk")
		return common.Hash{}
	}
	if err != nil {
		log.Info("Error Retrieving Header Chunk", "err", err)
		return common.Hash{}
	}
	log.Info(fmt.Sprintf("Resulting Header Hash: canonheaderhash (%x) [%+v]", common.BytesToHash(headerData), headerData))
	return common.BytesToHash(headerData)
}

func WriteHeadBlockHash(db cloud.Cloudstore, hash common.Hash, cid uint64) error {
	chainid := append(chainPrefix, encodeBlockNumber(cid)...)
	key := append(headBlockKey, chainid...)
	log.Info(fmt.Sprintf("Writing HeadBlockHash of with key [%s] [%+v] and got hash [%+v] [%x]", key, key, hash, hash))
	if err := db.SetChunk(key, hash.Bytes()); err != nil {
		//log.Crit("Failed to store last block's hash", "err", err)
		log.Info("Failed to store last block's hash", "err", err)
		return fmt.Errorf("Failed to store last block's hash | Error %+v", err)
	}
	return nil
}
*/

/* ########################################################################## */

/* Plasma Internal Method Do not expose */
/* ========================================================================== */

func (self *PlasmaChain) getBlockByNumber(blockNumber uint64) (b deep.Block, err error) {
	k := deep.Keccak256([]byte(fmt.Sprintf("plasma%d", blockNumber)))
	v, ok, err := self.ChunkStore.GetChunk(k)
	if err != nil {
		log.Info(fmt.Sprintf("Error retrieve block %d with key [%x] | {%+v}", blockNumber, k, err))
		return b, err
	} else if !ok {
		log.Info(fmt.Sprintf("Attempt to retrieve block %d with key [%x] and key not found", blockNumber, k))
		return b, nil
	} else if len(v) == 0 {
		return b, fmt.Errorf("Attempt to retrieve block %d with key [%x] and got nothing back", blockNumber, k)
	}
	log.Info(fmt.Sprintf("Successfully retrieved block %d with key [%x]", blockNumber, k))
	bl := FromChunk(v)
	log.Info("getBlockByNumber", "bn", blockNumber, "bl", bl.String())
	return *bl, nil
}

func (self *PlasmaChain) getBlockHeaderByNumber(blockNumber uint64) (h Header, err error) {
	headerKey := deep.Keccak256([]byte(fmt.Sprintf("plasmaHeader%d", blockNumber)))
	headerHash, ok, err := self.ChunkStore.GetChunk(headerKey)
	if err != nil {
		log.Debug("Error retrieving chunk", "error", err)
		return h, err
	} else if !ok {
		log.Debug("headerKey chunk not found")
		return h, nil
	} else if len(headerHash) == 0 {
		return h, fmt.Errorf("try to retrieve [block #%d] headerKey [%x] and got nothing back", blockNumber, headerKey)
	}
	data, ok, err := self.ChunkStore.GetChunk(headerHash)
	if err != nil {
		log.Debug("Error retrieving chunk", "error", err)
		return h, err
	} else if !ok {
		log.Debug("headerHash chunk not found")
		return h, fmt.Errorf("[block #%d] headerHash [%x] chunk not found", blockNumber, headerHash)
	} else if len(data) == 0 {
		return h, fmt.Errorf("try to retrieve [block #%d] headerHash [%x] and got nothing back", blockNumber, headerHash)
	}
	header := FromHeader(data)
	return *header, nil
}

func (self *PlasmaChain) getPlasmaToken(tokenID, blockNumber uint64) (t *Token, err error) {
	//tokenID8 := deep.BytesToUint64(tokenID.Bytes())
	h, err := self.getBlockHeaderByNumber(blockNumber)
	if err != nil {
		return t, err
	}
	log.Debug("PLASMAAPI: getPlasmaToken", "Header", h)
	s := NewStateDB(self.ChunkStore, h.Hash(), self.operatorKey)
	t, err = s.GetToken(tokenID)
	if err != nil {
		return t, err
	}
	log.Debug("GetPlasmaToken", "tokenID", tokenID)
	return t, nil
}

func (self *PlasmaChain) getPlasmaBalance(address common.Address, blockNumber uint64) (acct *Account, err error) {
	log.Debug("PLASMAAPI: getPlasmaBalance", "address", address)
	h, err := self.getBlockHeaderByNumber(blockNumber)
	if err != nil {
		return acct, err
	}
	log.Debug("PLASMAAPI: getPlasmaBalance", "Header", h)
	s := NewStateDB(self.ChunkStore, h.Hash(), self.operatorKey)

	acct, err = s.GetAccount(address)
	if err != nil {
		return acct, err
	}
	return acct, nil
}

func (self *PlasmaChain) getProofbyTokenID(tokenIndex uint64, bn uint64) (tokenID uint64, txbyte []byte, proof *smt.Proof, Blk uint64, prevBlk uint64, err error) {
	txHash, txProof, err := self.getHash(tokenIndex, bn, 0)
	if err != nil {
		return tokenID, txbyte, proof, Blk, prevBlk, err
	}

	var tx *Transaction
	tx, blk, receipt, err := self.GetPlasmaTransactionReceipt(common.BytesToHash(txHash))
	if err != nil || receipt == 0 {
		return tokenID, txbyte, proof, Blk, prevBlk, err
	}
	return tokenIndex, tx.Bytes(), txProof, blk, tx.PrevBlock, nil
}

func (self *PlasmaChain) getHash(index, bn uint64, treeType int) (hashKey []byte, proof *smt.Proof, err error) {
	log.Debug("Internal getSHash", "index", index, "BN", bn, "type", treeType)

	h, err := self.getBlockHeaderByNumber(bn)
	if err != nil {
		return hashKey, proof, err
	}

	s := NewStorage(self.ChunkStore)
	var tree *smt.SparseMerkleTree

	switch treeType {

	case 1:
		s.accountStorage.InitWithRoot(h.AccountRoot)
		tree = s.accountStorage
	case 2:
		s.tokenStorage.InitWithRoot(h.TokenRoot)
		tree = s.tokenStorage
	case 3:
		s.anchorStorage.InitWithRoot(h.AnchorRoot)
		tree = s.anchorStorage
	case 4:
		s.chainStorage.InitWithRoot(h.L3ChainRoot)
		tree = s.chainStorage
	default:
		s.transactionStorage.InitWithRoot(h.TransactionRoot)
		tree = s.transactionStorage
	}

	v0, found, pr, _, _, err := tree.Get(smt.UIntToByte(index))
	//log.Info("Get SMT All", "proof", p)
	if pr == nil || !found {
		return hashKey, pr, fmt.Errorf("Not found")
	} else {
		return v0, pr, nil
	}
}

func (self *PlasmaChain) getTokenInfo(tokenID uint64) (tInfo *TokenInfo, err error) {
	key := deep.Keccak256([]byte(fmt.Sprintf("plasmaTokenInfo%x", tokenID)))
	infoByte, ok, err := self.ChunkStore.GetChunk(key)
	if err != nil {
		log.Debug("[PlasmaChain:GetTokenInfo] GetChunk Error", "error", err)
		return tInfo, err
	} else if !ok {
		log.Debug("[PlasmaChain:GetTokenInfo] GetChunk - Chunk not found")
		return tInfo, nil
	} else if len(infoByte) == 0 {
		log.Debug("[PlasmaChain:GetTokenInfo] GetChunk - Chunk found but empty")
		return tInfo, fmt.Errorf("tokenID %x missing", tokenID)
	}

	err = rlp.Decode(bytes.NewReader(infoByte), &tInfo)
	if err != nil || tInfo == nil {
		return tInfo, err
	}
	return tInfo, nil
}

func (self *PlasmaChain) getTransaction(txhash common.Hash) (tx *Transaction, blockNumber uint64, err error) {
	key := txhash.Bytes()
	val, ok, err := self.ChunkStore.GetChunk(key)
	if err != nil {
		log.Info("PlasmaChain:getTransaction - Error", "error", err)
		return nil, blockNumber, err
	} else if !ok {
		log.Info("PlasmaChain:getTransaction - Chunk not found")
		return nil, blockNumber, err
	} else if len(val) == 0 {
		return nil, blockNumber, fmt.Errorf("tx %x can't be empty ", key)
	}

	tx, err = DecodeRLPTransaction(val)
	if err != nil {
		//return helpful msg if there's API mismatch
		var anchorTx *deep.AnchorTransaction
		err2 := rlp.Decode(bytes.NewReader(val), &anchorTx)
		if err2 == nil && anchorTx != nil {
			return nil, blockNumber, errors.New("Looking for AnchorTX?")
		}
		return nil, blockNumber, err
	}
	// Read blockNumber
	key2 := append([]byte("bn"), txhash.Bytes()...)
	val2, ok, err := self.ChunkStore.GetChunk(key2)
	if err != nil {
		return nil, blockNumber, nil
	} else if !ok {
		return nil, blockNumber, nil
	} else if len(val2) == 0 {
		return nil, blockNumber, fmt.Errorf("tx key2 %x can't be empty ", key2)
	}
	blockNumber = deep.BytesToUint64(val2)
	return tx, blockNumber, nil
}

func (self *PlasmaChain) getBlockTokenTransaction(blockNumber uint64, tokenID uint64) (tx *Transaction, p *smt.Proof, transactionRoot common.Hash, err error) {
	b, err := self.getBlockByNumber(blockNumber)
	if err != nil {
		return tx, p, transactionRoot, err
	}
	if b == nil {
		return tx, p, transactionRoot, fmt.Errorf("no block %d", blockNumber)
	}
	s := NewStorage(self.ChunkStore)
	s.InitWithBlock(b.(*Block))
	chunkID, ok, _, _, _, err := s.transactionStorage.Get(smt.UIntToByte(tokenID))
	if err != nil {
		return tx, p, transactionRoot, err
	} else if !ok {
		return tx, p, transactionRoot, fmt.Errorf("could not find tokenID transaction in block")
	}
	tx, _, err = self.getTransaction(common.BytesToHash(chunkID))
	if err != nil {
		return tx, p, transactionRoot, err
	} else if tx == nil {
		return tx, p, transactionRoot, fmt.Errorf("FAILED to get chunkID %x", chunkID)
	}
	txhash := tx.Hash().Bytes()
	p = s.transactionStorage.GenerateProof(smt.UIntToByte(tokenID), txhash)
	return tx, p, b.(*Block).header.TransactionRoot, nil
}

func (self *PlasmaChain) getTokenHistory(tokenID uint64) (txs []*Transaction) {
	//TODO
	return txs
}

/* Plasma Function for external API  */
/* ========================================================================== */

func (self *PlasmaChain) GetTokenInfo(tokenID uint64) (*TokenInfo, error) {
	tinfo, _ := self.getTokenInfo(tokenID)
	if tinfo == nil {
		return nil, fmt.Errorf("Not found")
	}
	return tinfo, nil
}

func (self *PlasmaChain) validateDatabaseOwner(dbname string, address common.Address) (err error) {
	// TODO : read dbname on MainNet, match against sig
	return nil
}

/* ########################################################################## */

/* Plasma APIs  */
/* ========================================================================== */
// APIs returns the collection of RPC services the ethereum package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *PlasmaChain) APIs() (apis []rpc.API) {
	return []rpc.API{
		{
			Namespace: "plasma",
			Version:   "1.0",
			Service:   NewPublicPlasmaAPI(s),
			Public:    true,
		}}
}

// Anchor APIs
/* ========================================================================== */

//TODO: deadcode. complete rewrite required
func (self *PlasmaChain) GetAnchor(blockchainID uint64, blockNumber rpc.BlockNumber, tokenID uint64, sig []byte) (v common.Hash, p *smt.Proof, pending uint64, err error) {
	return v, p, pending, err
}

// Transaction APIs
/* ========================================================================== */

func (self *PlasmaChain) SendPlasmaTransaction(tx *Transaction) (txhash common.Hash, err error) {
	if tx.balance == 0 {
		tx.balance = tx.Denomination - tx.Allowance - tx.Spent
	}
	err = self.validateTx(tx)
	if err != nil {
		log.Debug("SendPlasmaTransaction ERR:", "err", err)
		return txhash, err
	}

	log.Debug("PLASMAAPI: SendPlasmaTransaction", "tx", tx)
	err = self.addTransactionToPool(tx)
	if err != nil {
		return txhash, err
	}
	//self.protocolManager.plasma_txCh <- PlasmaTxPreEvent{Tx: tx}
	return tx.Hash(), nil
}

func (self *PlasmaChain) SendAnchorTransaction(tx *deep.AnchorTransaction) (txhash common.Hash, err error) {

	err = self.validateTx(tx)
	if err != nil {
		log.Debug("SendAnchorTransaction ERR:", "err", err)
		return txhash, err
	}

	log.Debug("SendAnchorTransaction", "anchor", tx)
	err = self.addAnchorTransactionToPool(tx)
	if err != nil {
		return txhash, err
	}
	self.protocolManager.BroadcastAnchorTx(txhash, tx)
	return tx.Hash(), nil
}

// PlasmaCash APIs
/* ========================================================================== */

func (self *PlasmaChain) GetPlasmaToken(tokenID hexutil.Uint64, blockNumber rpc.BlockNumber) (t *Token, tinfo *TokenInfo, err error) {

	t, err = self.getPlasmaToken(uint64(tokenID), uint64(blockNumber))
	if err != nil {
		return t, tinfo, err
	}
	tinfo, err = self.getTokenInfo(uint64(tokenID))
	return t, tinfo, err
}

func (self *PlasmaChain) GetPlasmaBalance(address common.Address, blockNumber rpc.BlockNumber) (acct *Account, err error) {
	return self.getPlasmaBalance(address, uint64(blockNumber))
}

func (self *PlasmaChain) GetPlasmaBlock(blockNumber rpc.BlockNumber) (block *Block) {
	// TODO: treat -1, -2 cases
	log.Info("PLASMAAPI: GetPlasmaBlock", "blockNumber", blockNumber)
	b, err := self.getBlockByNumber(uint64(blockNumber))
	if err != nil {
		log.Info("PLASMAAPI: GetPlasmaBlock", "bn", blockNumber, "ERROR", err)
		return nil
	}
	if b == nil {
		log.Info("PLASMAAPI: GetPlasmaBlock", "bn", blockNumber, "ERROR", "Not Found", "Verbose Error", err)
		return nil
	}
	b0 := b.(Block)
	return &b0
}

func (self *PlasmaChain) GetPlasmaBloomFilter(hash common.Hash) (b []byte, err error) {
	log.Debug("PLASMAAPI: GetPlasmaBloomFilter", "hash", hash)
	key := hash.Bytes()
	val, ok, err := self.ChunkStore.GetChunk(key)
	if err != nil {
		return b, err
	} else if !ok {
		return b, nil
	} else if len(val) == 0 {
		return nil, fmt.Errorf("bloomID %x can't be empty ", key)
	}
	return val, nil
}

//Doing a quick, non-authoritative check to avoid excessive chunk call
func (self *PlasmaChain) GetPlasmaTransactionReceipt(hash common.Hash) (tx *Transaction, blockNumber uint64, receipt uint8, err error) {
	log.Debug("PLASMAAPI: GetPlasmaTransactionReceipt", "hash", hash)
	txn, bn, err := self.getTransaction(hash)
	log.Debug("GetPlasmaTransactionReceipt", "bn", bn, "tx", txn.String())
	if err != nil {
		return tx, blockNumber, receipt, err
	} else if txn == nil || bn == 0 {
		return tx, blockNumber, receipt, nil
	} else if self.blockNumber < bn {
		return txn, bn, 0, nil
	}
	return txn, bn, 1, nil
}

func (self *PlasmaChain) GetPlasmaTransactionProof(hash common.Hash) (tokenID uint64, txbyte []byte, proof *smt.Proof, blockNumber uint64, err error) {
	log.Debug("GetPlasmaTransactionProof", "hash", hash)
	tx, bn, err := self.getTransaction(hash)
	if err != nil {
		return tokenID, txbyte, proof, blockNumber, err
	} else if len(tx.Bytes()) == 0 || tx == nil || bn == 0 {
		return tokenID, txbyte, proof, blockNumber, err
	}

	h, err := self.getBlockHeaderByNumber(bn)
	if err != nil {
		return tokenID, txbyte, proof, blockNumber, err
	}

	s := NewStorage(self.ChunkStore)
	s.transactionStorage.InitWithRoot(h.TransactionRoot)
	p := s.transactionStorage.GenerateProof(smt.UIntToByte(tx.TokenID), hash.Bytes())
	log.Debug("GetPlasmaTransactionProof", "proof", p)
	if p == nil {
		return tokenID, txbyte, proof, blockNumber, fmt.Errorf("unable to generate proof")
	}
	return tx.TokenID, tx.Bytes(), p, bn, nil
}

func (self *PlasmaChain) GetPlasmaTransactionFromPool(hash common.Hash) (tx *Transaction) {
	log.Debug("PLASMAAPI: GetPlasmaTransactionFromPool", "hash", hash)
	for _, tx := range self.plasmatxpool.txpool {
		if bytes.Compare(tx.Hash().Bytes(), hash.Bytes()) == 0 {
			return tx
		}
	}
	return nil
}

func (self *PlasmaChain) GetPlasmaTransactionPool() (txs map[common.Address][]*Transaction) {
	sort.Sort(PlasmabyTokenIDandPrevBlock(self.plasmatxpool.txpool))
	txs = make(map[common.Address][]*Transaction)
	for _, tx := range self.plasmatxpool.txpool {
		signer, err := tx.GetSigner()
		if err == nil {
			txs[signer] = append(txs[signer], tx)
		}
	}
	for _, addr := range txs {
		sort.Sort(PlasmabyTokenIDandPrevBlock(addr))
	}
	return txs
}

func (self *PlasmaChain) GetAnchorTransactionPool() (txs map[uint64][]*deep.AnchorTransaction) {
	sort.Sort(AnchorByChainIDandBlockNum(self.plasmatxpool.wtxpool))
	txs = make(map[uint64][]*deep.AnchorTransaction)
	for _, tx := range self.plasmatxpool.wtxpool {
		txs[tx.BlockChainID] = append(txs[tx.BlockChainID], tx)
	}
	for _, chainID := range txs {
		sort.Sort(AnchorByChainIDandBlockNum(chainID))
	}
	return txs
}

/* ########################################################################## */

/* Deposit related */
/* ========================================================================== */

func (self *PlasmaChain) setDeposits(deposits []*RootChainDeposit) map[uint64]*TokenInfo {
	var tokens = make(map[uint64]*TokenInfo)
	for _, deposit := range deposits {
		tokenID, tinfo := TokenInit(deposit.DepositIndex, deposit.Denomination, deposit.Depositor)
		log.Debug("setDeposits", "tokenID", tokenID, "tokenInfo", tinfo)
		tokens[tokenID] = tinfo
	}
	return tokens
}

func (self *PlasmaChain) processDeposits(tokens map[uint64]*TokenInfo) (err error) {
	for tokenID, tinfo := range tokens {
		err = self.processDeposit(deep.UInt64ToByte(tokenID), tinfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *PlasmaChain) processDeposit(tokenID []byte, t *TokenInfo) (err error) {
	tokenID8 := tokenID
	token := NewToken(t.DepositIndex, t.Denomination, t.Depositor)
	tx := NewTransaction(token, t, &(t.Depositor))
	a := crypto.PubkeyToAddress(self.operatorKey.PublicKey)
	tx.PrevOwner = &a
	err = tx.SignTx(self.operatorKey)
	if err != nil {
		return err
	}
	fmt.Printf("Deposit:  TokenID %x | Denomination %d | DepositIndex %d  (Depositor: %v, TxHash: 0x%s)\n", tokenID8, t.Denomination, t.DepositIndex, t.Depositor.Hex(), common.Bytes2Hex(tx.Hash().Bytes()))
	self.addTransactionToPool(tx)
	return nil
}

func (self *PlasmaChain) startExit(exiter common.Address, denomination uint64, depositIndex uint64, tokenID uint64, ts uint64) (err error) {
	return nil
}
func (self *PlasmaChain) publishedBlock(_rootHash common.Hash, _currentDepositIndex uint64, _blockNumber uint64) (err error) {
	return nil
}
func (self *PlasmaChain) finalizedExit(exiter common.Address, denomination uint64, depositIndex uint64, tokenID uint64, ts uint64) (err error) {
	return nil
}
func (self *PlasmaChain) challenge(challenger common.Address, tokenID uint64, ts uint64) (err error) {
	return nil
}

/* ########################################################################## */

/* RootChain Contract Related */
/* ========================================================================== */
func (self *PlasmaChain) setRootContract(contractAddr, rpc_endpointUrl, ws_endpointUrl, operatorPrivateKey string) (err error) {

	if !common.IsHexAddress(contractAddr) {
		return fmt.Errorf("Invalid contract Address %v", contractAddr)
	}

	rootContract := common.HexToAddress(contractAddr)
	if self.operatorKey, err = crypto.HexToECDSA(operatorPrivateKey); err != nil {
		return err
	}
	if self.connWS, err = setConnection(ws_endpointUrl); err != nil {
		return err
	}
	if self.session, err = setSession(rootContract, rpc_endpointUrl, self.operatorKey); err != nil {
		return err
	}
	return nil
}

func (self *PlasmaChain) loadLastState(simulated bool) error {
	// figure out what to do here in test vs prod
	if simulated {
		b := NewBlock()
		self.blockNumber = 0
		self.currentBlock = b
		//TODO: simulated should set to false in defaultsetting
		if !self.config.UseLayer1 {
			self.initDeposit(10, simulated)
		}
	} else {
		b := NewBlock()
		//self.blockNumber = 1 // the one we are working on right now, not the most recent block
		self.blockNumber = 0
		self.currentBlock = b
		self.initDeposit(10, simulated)
	}
	return nil
}

/* ########################################################################## */

/* Prtotype-use only. Should be Removed */
/* ========================================================================== */

func (self *PlasmaChain) initDeposit(maxTokens uint64, simulated bool) (err error) {

	if simulated {
		//fmt.Printf("Simulation : %v, preparing test deposits\n", simulated)
		for i := uint64(0); i < 5; i++ {
			var tinfo *TokenInfo
			var tokenID uint64
			switch i {
			case 0:
				tokenID, tinfo = TokenInit(0, 1000000000000000000, common.HexToAddress("0xA45b77a98E2B840617e2eC6ddfBf71403bdCb683"))
				//tokenID, tinfo = TokenInit(0, 18446744073709551614, common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914"))
				break
			case 1:
				tokenID, tinfo = TokenInit(1, 1000000000000000000, common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914"))
				//tokenID, tinfo = TokenInit(1, 18446744073709551614, common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914"))
				break
			case 2:
				tokenID, tinfo = TokenInit(2, 2000000000000000000, common.HexToAddress("0x3088666E05794d2498D9d98326c1b426c9950767"))
				break
			case 3:
				tokenID, tinfo = TokenInit(3, 3000000000000000000, common.HexToAddress("0xBef06CC63C8f81128c26efeDD461A9124298092b"))
				break
			case 4:
				tokenID, tinfo = TokenInit(4, 4000000000000000000, common.HexToAddress("0x74f978A3E049688777E6120D293F24348BDe5fA6"))
				break
			case 5:
				tokenID, tinfo = TokenInit(5, 5000000000000000000, common.HexToAddress("0x59B66c66b9159b62DaFCB5fEde243384DFca076D"))
				break
			}
			errD := self.processDeposit(deep.UInt64ToByte(tokenID), tinfo)
			if errD != nil {
				// TODO: do something!
			}
		}
	} else {
		fmt.Printf("Simulation : %v, getting actual deposit\n", simulated)
		currentDepositIndex, _ := self.session.CurrentDepositIndex()
		fmt.Printf("currentDepositIndex : %v\n", currentDepositIndex-1)
		// Create Filterer from session
		rootcontractFilterer := &self.session.Contract.RootChainFilterer
		pbs, _ := rootcontractFilterer.getPublishedBlock()
		prevdep, currdep := uint64(0), uint64(0)
		for _, pb := range pbs {
			currdep = pb.CurrentDepositIndex - 1
			deposits, _ := rootcontractFilterer.getPastDeposit(prevdep, currdep)
			if len(deposits) > 0 {
				fmt.Printf("PublishedBlock: #%v | Hash %x | DepIndex %v\n", pb.Blknum, pb.RootHash, pb.CurrentDepositIndex)
				tokens := self.setDeposits(deposits)
				errD := self.processDeposits(tokens)
				if errD != nil {
					// TODO: do something!
				}
			}
			prevdep = pb.CurrentDepositIndex
		}
		//recreating last block
		currdep = currentDepositIndex
		lastDeposits, _ := rootcontractFilterer.getPastDeposit(prevdep, currdep)
		fmt.Printf("Recreating latest block [prevdep %v, currdep %v]\n", prevdep, currdep-1)
		tokens := self.setDeposits(lastDeposits)
		errD := self.processDeposits(tokens)
		if errD != nil {
			// TODO: do something!
		}
	}
	return nil
}

/* ########################################################################## */
