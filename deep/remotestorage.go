// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.
package deep

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	//"github.com/wolkdb/cloudstore/wolk/cloud"
	//"github.com/wolkdb/cloudstore_orig/wolk/cloud"
)

const maxRetries = 3

type Item struct {
	Value      []byte
	DeleteTime int64
}

type Chunk struct {
	ChunkID []byte `json:"chunkID"`
	Value   []byte `json:"val"`
	OK      bool   `json:"ok"`
	Error   error  `json:"error"`
}

type RemoteStorage struct {
	chainType              string
	blockchainId           uint64
	plasma_endpointUrl     string
	cloudstore_endpointUrl string
	tokenId                uint64
	key                    *ecdsa.PrivateKey
	plasma_rpcclient       *rpc.Client
	cloudstore_rpcclient   *rpc.Client
	chunkQueue             []*Chunk
	cache                  map[string]Item
	cachemu                sync.RWMutex
	ldb                    *leveldb.DB
	localOnly              bool
}

func NewRemoteStorage(_blockchainId uint64, config WolkConfig, _chainType string, _key *ecdsa.PrivateKey /*, _tokenId uint64*/) (a *RemoteStorage, err error) {
	_plasma_endpointUrl := "http://" + config.GetPlasmaAddr() + ":" + fmt.Sprintf("%d", config.GetPlasmaPort())
	_cloudstore_endpointUrl := "http://" + config.GetCloudstoreAddr() + ":" + fmt.Sprintf("%d", config.GetCloudstorePort())

	plasma_rpcclient, perr := rpc.Dial(_plasma_endpointUrl)

	if perr != nil {
		log.Info("ERR: plasma_rpcclient: ", "err", perr)
		return a, perr
	} else {
		log.Info("Connected to plasma successfully", "plasma_endpointUrl", _plasma_endpointUrl)
	}

	cloudstore_rpcclient, cerr := rpc.Dial(_cloudstore_endpointUrl)
	if cerr != nil {
		log.Info("ERR: cloudstore_rpcclient: ", "err", cerr)
		return a, cerr
	} else {
		log.Info("Connected to cloudstore successfully", "chunkstore_endpointUrl", _cloudstore_endpointUrl)
	}

	//TODO: 	Place 'o' in config file
	o := &opt.Options{
		BlockCacheCapacity: 4 * opt.GiB,
	}
	ldb, err := leveldb.OpenFile(config.GetDataDir(), o)
	if err != nil {
		log.Info("Error connecting to ldb", "err", err, "path", config.GetDataDir())
		//return a, fmt.Errorf("[remoteStorage:NewRemoteStorage] OpenFile %s: %v\n", config.GetDataDir(), err)
	} else {
		log.Info("remoteStorage:NewRemoteStorage ldb->OpenFile", "path", config.GetDataDir())
	}

	log.Info("Setting up RemoteStorage", "blockchainId", _blockchainId, "plasma_endpointUrl", _plasma_endpointUrl, "chunkstore_endpointUrl", _cloudstore_endpointUrl, "ldb", config.GetDataDir(), "localMode", config.IsLocalMode())
	return &RemoteStorage{
		blockchainId:           _blockchainId,
		cloudstore_endpointUrl: _cloudstore_endpointUrl, //required or optional?
		chainType:              _chainType,
		tokenId:                uint64(76),
		key:                    _key,
		plasma_rpcclient:       plasma_rpcclient,
		cloudstore_rpcclient:   cloudstore_rpcclient,
		ldb:                    ldb,
		localOnly:              config.IsLocalMode(),
	}, nil
}

/* TODO: When we figure out key/sig stuff */
func (self *RemoteStorage) Sign(chunkID *common.Hash) (sig []byte) {
	//This is for test signing
	sig, err := crypto.Sign(chunkID.Bytes(), self.key)
	if err != nil {
		return nil
	}
	return sig
}

func (self *RemoteStorage) QueueContents() []*Chunk {
	return self.chunkQueue
}

func (self *RemoteStorage) QueueChunk(chunkID []byte, chunk []byte) (err error) {
	dupeFound := false
	var newChunk Chunk
	newChunk.ChunkID = chunkID
	newChunk.Value = chunk
	for i, c := range self.chunkQueue {
		if bytes.Equal(c.ChunkID, newChunk.ChunkID) {
			dupeFound = true
			self.chunkQueue[i].Value = newChunk.Value
			// use SetChunkLocal mark that its written to leveldb but not yet to cloudstore
			// when we Flush then this is removed
			//self.SetChunkLocal(newChunk.ChunkID, newChunk.Value)
		}
	}
	if !dupeFound {
		self.chunkQueue = append(self.chunkQueue, &newChunk)
	}
	return nil
}

func (self *RemoteStorage) Flush() (err error) {
	log.Info("self *RemoteStorage|Flush")
	if len(self.chunkQueue) == 0 {
		log.Info("[remotestorage:Flush] returning because queue is empty")
		return nil
	}
	//log.Info("*****RPC-SetChunkBatch", "nchunks", len(self.chunkQueue))
	for i, ch := range self.chunkQueue {
		log.Info("  *****RPC-SetChunkBatch", "i", i, "ch.ChunkID", ch.ChunkID, "hexChunkID", fmt.Sprintf("%x", ch.ChunkID), "len(ch.Value)", len(ch.Value))
	}

	err = self.SetChunkBatch(self.chunkQueue)
	if err != nil {
		log.Info("Flush: Error setting Chunk Batch ", "error", err)
		return err
	}
	log.Debug(fmt.Sprintf("[RemoteStorage:Flush] Flushed Queue: [%+v]", self.chunkQueue))

	//Clear Queue
	self.chunkQueue = self.chunkQueue[:0] //make([]cloud.RawChunk, 1)
	return nil
}

func (self *RemoteStorage) FlushToLocal() (err error) {
	log.Info("self *RemoteStorage|FlushToLocal")

	//log.Info("*****RPC-SetChunkBatch", "nchunks", len(self.chunkQueue))
	for i, ch := range self.chunkQueue {
		log.Info("  *****RPC-SetChunkBatch", "i", i, "ch.ChunkID", ch.ChunkID, "len(ch.Value)", len(ch.Value))
	}

	err = self.SetChunkBatchLocal(self.chunkQueue)
	if err != nil {
		log.Info("Flush: Error setting Chunk Batch ", "error", err)
		return fmt.Errorf("[deep:remotestorage:FlushToLocal] %s", err)
	}
	log.Debug(fmt.Sprintf("[RemoteStorage:FlushToLocal] Flushed Queue: [%+v]", self.chunkQueue))
	return nil
}

// SetChunk(chunkID common.Hash, chunk []byte, tokenId uint64, sig []byte) (o map[string]interface{}) {
func (self *RemoteStorage) SetChunk(chunkID []byte, chunk []byte) (err error) {
	//log.Info("******* RPC-SetChunk", "chunkID", chunkID, "chunkidhex", fmt.Sprintf("%x", chunkID), "len(chunk)", len(chunk))
	//set to local cache (ldb)
	localset_err := self.SetChunkLocal(chunkID, chunk)
	if self.localOnly {
		if localset_err != nil {
			log.Info(fmt.Sprintf("[remotestorage:SetChunk] [LocalMode] Error storing to cache [%+v].  Continuing to attempt cloudstore setChunk", localset_err))
			return localset_err
		}
		return nil
	}

	if localset_err != nil {
		log.Info(fmt.Sprintf("[remotestorage:SetChunk] Error storing to cache [%+v].  Continuing to attempt cloudstore setChunk", localset_err))
	}

	// cloudstore_err := self.SetChunkCloudstore(chunkID, chunk)
	// if cloudstore_err != nil {
	// 	log.Info("[remotestorage:SetChunk] Error storing chunk to cloudstore", "error", cloudstore_err)
	// 	return cloudstore_err
	// }

	return nil
}

func (self *RemoteStorage) SetChunkBatch(chunks []*Chunk) (err error) {
	log.Info("self *RemoteStorage|SetChunkBatch")
	log.Info("*****RPC-SetChunkBatch", "nchunks", len(chunks))
	for i, ch := range chunks {
		log.Info("  *****RPC-SetChunkBatch", "i", i, "ch.ChunkID", ch.ChunkID, "len(ch.Value)", len(ch.Value), "ok", ch.OK, "err", ch.Error)
	}
	//setChunkBatch Local
	//TODO: is SetChunkBatchLocal return error for the whole batch?
	localset_err := self.SetChunkBatchLocal(chunks)
	if self.localOnly {
		if localset_err != nil {
			log.Info("[LocalMode] Error storing to cache.")
			return localset_err
		}
		return nil
	}

	if localset_err != nil {
		log.Info("Error storing to cache. Continuing to attempt cloudstore setChunk")
		fmt.Printf("[deep:remotestorage:SetChunkBatch] Error storing to cache. Continuing to attempt cloudstore setChunk")
	}

	//Cloudstore setChunkBatch
	// cloudstore_err := self.SetChunkBatchCloudstore(chunks)
	// if cloudstore_err != nil {
	// 	log.Info(fmt.Sprintf("Error storing to cloudstore [%+v].  Continuing to attempt cloudstore setChunkBatch", cloudstore_err))
	// 	return cloudstore_err
	// }
	return nil
}

// GetChunk(chunkID common.Hash, tokenId uint64, sig []byte) (o map[string]interface{}) {
func (self *RemoteStorage) GetChunk(chunkID []byte) (chunk []byte, ok bool, err error) {

	//getchunkbatch local
	chunk, ok, err = self.GetChunkLocal(chunkID)
	if self.localOnly || ok {
		return chunk, ok, err
	}

	//Cloudstore GetChunk
	// chunk, ok, err = self.GetChunkCloudstore(chunkID)
	// if err != nil {
	// 	return chunk, ok, err
	// }

	//store to local cache
	if err == nil && ok {
		setchunklocal_err := self.SetChunkLocal(chunkID, chunk)
		if setchunklocal_err != nil {
			log.Info("Error setting cache after retrieval from cloudstore")
		}
	}

	return chunk, ok, err
}

func (self *RemoteStorage) GetChunkBatch(chunks []*Chunk) error {
	//start := time.Now()
	//getchunkbatch local
	getchunk_err := self.GetChunkBatchLocal(chunks)
	var uncachedChunks []*Chunk
	uncachedChunks = make([]*Chunk, 0)
	for _, c := range chunks {
		if !c.OK {
			uncachedChunks = append(uncachedChunks, c)
		}
	}

	if self.localOnly {
		if getchunk_err != nil {
			return fmt.Errorf("[LocalMode] Error encountered trying to retrieve from cache: %v", getchunk_err)
		}
		if len(uncachedChunks) != 0 {
			return fmt.Errorf("[LocalMode] Error encountered trying to retrieve uncached Chunks: %v", uncachedChunks)
		}
	}

	if getchunk_err == nil && len(uncachedChunks) == 0 {
		return nil
	} else {
		log.Info("Error encountered trying to retrieve from cache. Proceed to requesting from remote cloudstore")
	}

	//getchunkbatch cloudstore
	// err := self.GetChunkBatchCloudstore(uncachedChunks)
	// if err != nil {
	// 	//TODO:
	// 	log.Info(fmt.Sprintf("[remotestorage:GetChunkBatch] Error in GetChunkBatchCloudstore | %+v", err))
	// 	return fmt.Errorf("[remotestorage:GetChunkBatch] Error in GetChunkBatchCloudstore | %+v", err)
	// }
	// var chunksToCache []*Chunk
	// chunksToCache = make([]*Chunk, 0)
	// chunksRetrieved := uncachedChunks
	// for _, ch := range chunksRetrieved {
	// 	if ch.ChunkID != nil && ch.Value != nil {
	// 		for i, mch := range chunks {
	// 			if bytes.Compare(ch.ChunkID, mch.ChunkID) == 0 {
	// 				mch.Value = ch.Value
	// 				mch.OK = ch.OK
	// 				mch.Error = ch.Error
	// 				if ch.OK {
	// 					chunksToCache = append(chunksToCache, mch)
	// 					log.Info(" *chunksToCache* ", "i", i, "ch.ChunkID", fmt.Sprintf("%x", mch.ChunkID))
	// 				}
	// 			}
	// 		}
	// 	} else {
	// 		//QUESTION: if chunkid is nil should there be an error?
	// 	}
	// }
	//
	// setChunkBatchLocal_err := self.SetChunkBatchLocal(chunksToCache)
	// if setChunkBatchLocal_err != nil {
	// 	log.Info("Error setting cache")
	// }
	return nil
}

func (self *RemoteStorage) GetChunkLocal(chunkID []byte) (chunk []byte, ok bool, err error) {
	if self.ldb == nil {
		return []byte(""), false, fmt.Errorf("cache unavailable")
	}
	v, err := self.ldb.Get(chunkID, nil)
	if err == leveldb.ErrNotFound {
		return v, false, nil
	} else if err != nil {
		return v, false, err
	}
	return v, true, nil
}

// func (self *RemoteStorage) GetChunkCloudstore(chunkID []byte) (chunk []byte, ok bool, err error) {
// 	var resp map[string]interface{}
// 	chunkIDHash := common.BytesToHash(chunkID)
// 	sig := self.Sign(&chunkIDHash)
// 	currentRetries := 0
//
// 	for currentRetries < maxRetries {
// 		currentRetries++
// 		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
// 		defer cancel()
// 		err = self.cloudstore_rpcclient.CallContext(ctx, &resp, "cloudstore_getChunk", self.blockchainId, chunkID, self.tokenId, sig)
// 		if err != nil {
// 			if currentRetries >= maxRetries || !strings.Contains(err.Error(), "context deadline exceeded") {
// 				log.Info(fmt.Sprintf("[remotestorage:GetChunkCloudstore] Error %+v", err))
// 				return chunk, false, err
// 			} else {
// 				log.Info(fmt.Sprintf("[remotestorage:GetChunkCloudstore] Retrying %d time | Error: %+v", currentRetries, err))
// 			}
// 		} else {
// 			currentRetries = maxRetries
// 		}
// 	}
//
// 	if respErr, errFound := resp["err"]; errFound {
// 		log.Info(fmt.Sprintf("[remotestorage:GetChunk] Error %+v", respErr))
// 		return chunk, false, fmt.Errorf("Error in GetChunk: %+v", respErr.(string))
// 	}
//
// 	var respVal interface{}
// 	var respOk bool
// 	if respVal, respOk = resp["v"]; !respOk {
// 		log.Info("*****RPC-GetChunk NOT FOUND", "chunkID", chunkID, "chunkIDHex", fmt.Sprintf("%x", chunkID))
// 		// treat not found case (3)
// 		return chunk, false, nil
// 	}
// 	// old cloudstore is going down this route, to be depcreated
// 	if respVal == nil { //TODO: why is this nil vs empty?
// 		log.Info("*****RPC-GetChunk NIL CASE", "chunkID", chunkID, "chunkIDHex", fmt.Sprintf("%x", chunkID))
// 		return chunk, false, nil
// 	}
//
// 	decodedChunk, _ := base64.StdEncoding.DecodeString(respVal.(string))
// 	fixedChunk := []byte(decodedChunk)
// 	log.Info("*****RPC-GetChunk FOUND", "chunkID", chunkID, "chunkIDHex", fmt.Sprintf("%x", chunkID), "len(chunk)", len(fixedChunk))
//
// 	return fixedChunk, true, nil
// }

func (self *RemoteStorage) GetChunkBatchLocal(chunks []*Chunk) (err error) {
	if self.ldb == nil {
		return fmt.Errorf("cache unavailable")
	}
	for i, chunk := range chunks {
		v, err := self.ldb.Get(chunk.ChunkID, nil)
		if err == leveldb.ErrNotFound {
			chunks[i] = &Chunk{Value: v, OK: false, Error: nil}
		} else if err != nil {
			chunks[i] = &Chunk{Value: v, OK: false, Error: err}
		}
		chunks[i] = &Chunk{Value: v, OK: true, Error: nil}
		log.Info(fmt.Sprintf("Chunk found index[%d], len(chunk) [%d] chunk [%+v] err (%+v)", i, len(chunks[i].Value), chunks[i].OK, chunks[i].Error))
	}
	return nil
}

// func (self *RemoteStorage) GetChunkBatchCloudstore(chunks []*Chunk) (err error) {
// 	var resp map[string]interface{}
// 	chunksHash := common.BytesToHash([]byte(""))
// 	sig := self.Sign(&chunksHash)
// 	log.Info("*****RPC-GetChunkBatch", "bcid", self.blockchainId, "tokenID", self.tokenId, "sig", sig)
//
// 	//TODO: check for "context deadline exceeded" and retry before erroring
// 	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
// 	defer cancel()
// 	err = self.cloudstore_rpcclient.CallContext(ctx, &resp, "cloudstore_getChunkBatch", self.blockchainId, chunks, self.tokenId, sig)
// 	if err != nil {
// 		log.Info(fmt.Sprintf("[remotestorage:GetChunkBatch] Error %+v", err))
// 		return err
// 	}
//
// 	if respErr, errFound := resp["err"]; errFound {
// 		// treat  TRUE ERR case (4)
// 		log.Info(fmt.Sprintf("[remotestorage:GetChunkBatch] Error %+v", respErr))
// 		return fmt.Errorf("Error in GetChunkBatch: %+v", respErr.(string))
// 	}
//
// 	if resp["chunks"] != nil {
// 		chunksRetrieved := resp["chunks"].([]interface{}) //QUESTION: Add assert?
// 		for _, ch := range chunksRetrieved {
// 			retrievedChunk := ch.(map[string]interface{}) //QUESTION: Add assert?
// 			if retrievedChunk["chunkID"] != nil && retrievedChunk["val"] != nil {
// 				retrievedChunkID, _ := base64.StdEncoding.DecodeString(retrievedChunk["chunkID"].(string))
// 				retrievedVal, _ := base64.StdEncoding.DecodeString(retrievedChunk["val"].(string))
// 				found := false
// 				for _, origChunk := range chunks {
// 					if bytes.Compare(retrievedChunkID, origChunk.ChunkID) == 0 {
// 						origChunk.Value = retrievedVal
// 						origChunk.OK = retrievedChunk["OK"].(bool)
// 						origChunk.Error = retrievedChunk["Error"].(error)
// 						found = true
// 					}
// 				}
// 				if !found {
// 					fmt.Printf("[RemoteStorage:GetChunkBatch] chunkID %x len(val) %d NOT MATCHED\n", retrievedChunkID, len(retrievedVal))
// 					//TODO: add error
// 				}
// 			} else {
// 				//QUESTION: if chunkid is nil should there be an error?
// 			}
// 		}
// 	}
// 	return nil
// }

func (self *RemoteStorage) SetChunkLocal(chunkID []byte, chunk []byte) (err error) {
	/*
		chunkIDHash := common.BytesToHash(chunkID)
		sig := self.Sign(&chunkIDHash)
	*/
	if self.ldb == nil {
		return fmt.Errorf("cache unavailable")
	}
	err = self.ldb.Put(chunkID, chunk, nil)
	if err != nil {
		return err
	}
	return nil
}

// func (self *RemoteStorage) SetChunkCloudstore(chunkID []byte, chunk []byte) (err error) {
// 	var resp map[string]interface{}
// 	chunkIDHash := common.BytesToHash(chunkID)
// 	sig := self.Sign(&chunkIDHash)
//
// 	//TODO: check for "context deadline exceeded" and retry before erroring
// 	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
// 	defer cancel()
// 	//log.Info(fmt.Sprintf("remoteStorage:StoreChunk chunkID [%+v](%x) chunkbytes[%+v]", chunkID, chunkID, chunk))
// 	err = self.cloudstore_rpcclient.CallContext(ctx, &resp, "cloudstore_setChunk", self.blockchainId, chunkID, chunk, self.tokenId, sig)
// 	if err != nil {
// 		log.Info(fmt.Sprintf("[remotestorage:SetChunk] Error %+v", err))
// 		return err
// 	}
// 	return nil
// }

func (self *RemoteStorage) SetChunkBatchLocal(chunks []*Chunk) (err error) {
	log.Info("Starting SetChunkBatchLocal")
	if self.ldb == nil {
		log.Info("ldb is nil ... Problem")
		return fmt.Errorf("[deep:remotestorage:SetChunkBatchLocal] cache unavailable")
	}
	batch := new(leveldb.Batch)
	for _, ch := range chunks {
		batch.Put(ch.ChunkID, ch.Value)
	}

	err = self.ldb.Write(batch, nil)
	if err != nil {
		log.Debug("Error writing batch", "err", err)
		return fmt.Errorf("[deep:remotestorage:SetChunkBatchLocal] error writing batch %s", err)
	}
	log.Info("batch written locally")
	return err
}

// func (self *RemoteStorage) SetChunkBatchCloudstore(chunks []*Chunk) (err error) {
// 	var resp map[string]interface{}
// 	dummyBytes := []byte("dummy")
// 	dummyMsgHash := Keccak256(dummyBytes)
// 	dummySignHash := common.BytesToHash(dummyMsgHash)
// 	sig := self.Sign(&dummySignHash)
//
// 	//TODO: check for "context deadline exceeded" and retry before erroring
// 	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
// 	defer cancel()
// 	err = self.cloudstore_rpcclient.CallContext(ctx, &resp, "cloudstore_setChunkBatch", self.blockchainId, chunks, self.tokenId, sig)
// 	if err != nil {
// 		log.Info("Error calling cloudstore_setChunkBatch: ", "error", err)
// 		self.chunkQueue = self.chunkQueue[:0]
// 		return err
// 	}
// 	if resp["err"] != nil {
// 		log.Info("Error calling cloudstore_setChunkBatch: ", "error", resp["err"])
// 		self.chunkQueue = self.chunkQueue[:0]
// 		return fmt.Errorf(resp["err"].(string))
// 	}
// 	log.Info(fmt.Sprintf("[remotestorage:SetChunkBatch] %+v", resp))
// 	return nil
// }

func (self *RemoteStorage) CreateAnchorTransaction(blockNumber uint64, blockHash *common.Hash) *AnchorTransaction {
	tx := NewAnchorTransaction(self.blockchainId, blockNumber, blockHash)
	return tx
}

func (self *RemoteStorage) SendAnchorTransaction(tx *AnchorTransaction) (err error) {
	tx.SignTx(self.key)
	log.Info("Anchor.go:SendAnchorTransaction", "tx", tx)
	//var resp map[string]interface{}
	var resp interface{}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	err = self.plasma_rpcclient.CallContext(ctx, &resp, "plasma_sendAnchorTransaction", tx)
	if err != nil {
		log.Info("[remotestorage:SendAnchorTransaction] Error sending anchortransaction from RemoteStorage", "error", err)
		return err
	}

	return nil
}

func (self *RemoteStorage) GetAnchor(tokenId uint64, blockNumber rpc.BlockNumber) (blockHash *common.Hash, err error) {
	sig := self.Sign(blockHash)
	var resp map[string]interface{}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	err = self.plasma_rpcclient.CallContext(ctx, &resp, "plasma_getAnchor", tokenId, blockNumber, sig)
	if err != nil {
		return blockHash, err
	}
	return blockHash, nil
}

func GetBlockchainID(tokenID uint64, blockIndex uint64) uint64 {
	b1 := UInt64ToByte(tokenID)
	b2 := UInt64ToByte(blockIndex)
	b := Keccak256(append(b1, b2...))
	blockChainID := b[24:32]
	return BytesToUint64(blockChainID)
}

func (self *RemoteStorage) Close() error {
	if self.ldb == nil {
		return nil
	}
	//self.cloudstore_rpcclient.Close()
	//self.plasma_rpcclient.Close()
	return self.ldb.Close()
}

func (self *RemoteStorage) Has(k []byte) (bool, error) {
	//TODO:
	return true, nil
}

func (self *RemoteStorage) Delete(k []byte) error {
	//TODO:
	return nil
}

/*
func (self *RemoteStorage) CacheStore(k, v []byte) error {
	MAXCACHESIZE := 10000
	strkey := string(k)
	item := Item{
		Value:      v,
		DeleteTime: time.Now().Add(1 * time.Hour).UnixNano(),
	}
	self.cachemu.Lock()
	self.cache[strkey] = item
	cachesize := len(self.cache)
	self.cachemu.Unlock()
	if cachesize > MAXCACHESIZE {
		go self.CleanCache(0)
	}
	return nil
}

func (self *RemoteStorage) CacheRetrieve(k []byte) ([]byte, error) {
	strkey := string(k)
	self.cachemu.RLock()
	if _, ok := self.cache[strkey]; ok {
		return self.cache[strkey].Value, nil
	}
	self.cachemu.RUnlock()
	// may need to change item.DeleteTime to make it alive longer
	return nil, fmt.Errorf("%s", "No Data Found")
}

// CleanCache will be run at backend.
func (self *RemoteStorage) CleanCache(size uint64) {
	// size will be used for the capacity
	tmpcache := make(map[string]Item)
	current := time.Now().UnixNano()
	self.cachemu.RLock()
	for key, item := range self.cache {
		if item.DeleteTime > current {
			tmpcache[key] = item
		}
	}
	self.cachemu.RUnlock()
	self.cachemu.Lock()
	self.cache = tmpcache
	self.cachemu.Unlock()
}
*/
