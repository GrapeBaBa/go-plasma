// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package wolk

import (
	"bytes"
	"context"

	"encoding/base64"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	wolkcommon "github.com/wolkdb/cloudstore/common"
	"github.com/wolkdb/cloudstore/wolk/cloud"
)

type StorageLayer interface {
	GetChunk(k []byte) (v []byte, err error)
	GetChunkBatch([]*cloud.RawChunk) error
	SetChunk(k []byte, v []byte) (err error)
	SetChunkBatch([]*cloud.RawChunk) (err error)
	SetShare(chunk []byte) (mr []byte, chunkID []byte, err error)
	GetShare(chunkID []byte) (chunk []byte, err error)
}

type RemoteStorage struct {
	chainType    string
	blockchainId uint64
	endpointUrl  string
	tokenId      uint64
	rpcclient    *rpc.Client
	chunkQueue   []cloud.RawChunk
	batchMode    bool
}

func NewRemoteStorage(_blockchainId uint64, _endpointUrl string, _chainType string, _tokenId uint64) (a *RemoteStorage, err error) {
	rpcclient, err := rpc.Dial(_endpointUrl)
	log.Debug("NewRemoteStorage", "url", _endpointUrl)
	if err != nil {
		return a, err
	}

	return &RemoteStorage{
		blockchainId: _blockchainId,
		endpointUrl:  _endpointUrl,
		chainType:    _chainType,
		tokenId:      _tokenId,
		rpcclient:    rpcclient,
		batchMode:    true,
	}, nil
}

func (self *RemoteStorage) Sign(chunkID *common.Hash) (sig []byte) {
	// TODO
	return sig
}

// GetChunk(chunkID common.Hash, tokenId uint64, sig []byte) (o map[string]interface{}) {
func (self *RemoteStorage) GetChunk(chunkID []byte) (chunk []byte, err error) {
	var resp map[string]interface{}
	//log.Info(fmt.Sprintf("remoteStorage:GetChunk chunkID [%+v](%x)", chunkID, chunkID))
	chunkIDHash := common.BytesToHash(chunkID)
	sig := self.Sign(&chunkIDHash)
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	err = self.rpcclient.CallContext(ctx, &resp, "cloudstore_getChunk", self.blockchainId, chunkID, self.tokenId, sig)
	if err != nil {
		log.Info(fmt.Sprintf("[remotestorage:GetChunk] Error %+v", err))
		return chunk, err
	}

	if err, ok := resp["err"]; ok {
		return chunk, fmt.Errorf("GetChunk error %v", err)
	}

	if _, ok := resp["v"]; !ok {
		return chunk, fmt.Errorf("No value")
	}
	decodedChunk, _ := base64.StdEncoding.DecodeString(resp["v"].(string))
	fixedChunk := []byte(decodedChunk)
	return fixedChunk, nil
}

// SetChunk(chunkID common.Hash, chunk []byte, tokenId uint64, sig []byte) (o map[string]interface{}) {
func (self *RemoteStorage) SetChunk(chunkID []byte, chunk []byte) (err error) {
	var resp map[string]interface{}
	chunkIDHash := common.BytesToHash(chunkID)
	sig := self.Sign(&chunkIDHash)

	//log.Info(fmt.Sprintf("remoteStorage:SetChunk chunkID [%+v](%x) chunkbytes[%+v]", chunkID, chunkID, chunk))
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	err = self.rpcclient.CallContext(ctx, &resp, "cloudstore_setChunk", self.blockchainId, chunkID, chunk, self.tokenId, sig)
	if err != nil {
		log.Info(fmt.Sprintf("[remotestorage:SetChunk] Error %+v", err))
		return err
	}
	return nil
}

func (self *RemoteStorage) SetChunkBatch(chunks []*cloud.RawChunk) (err error) {
	var resp map[string]interface{}
	kAsHash := common.BytesToHash([]byte(""))
	sig := self.Sign(&kAsHash)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = self.rpcclient.CallContext(ctx, &resp, "cloudstore_setChunkBatch", self.blockchainId, chunks, self.tokenId, sig)
	if err != nil {
		log.Info("Error calling cloudstore_setChunkBatch: ", "error", err)
		return err
	}
	if resp["err"] != nil {
		log.Info("Error calling cloudstore_setChunkBatch: ", "error", resp["err"])
		return fmt.Errorf(resp["err"].(string))
	}
	return nil
}

func (self *RemoteStorage) GetChunkBatch(reqChunks []*cloud.RawChunk) (err error) {
	var resp map[string]interface{}
	kAsHash := common.BytesToHash([]byte(""))
	sig := self.Sign(&kAsHash)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = self.rpcclient.CallContext(ctx, &resp, "cloudstore_getChunkBatch", self.blockchainId, reqChunks, self.tokenId, sig)
	if err != nil {
		log.Info("Error calling cloudstore_getChunkBatch: ", "error", err)
		return err
	}
	if resp["err"] != nil {
		log.Info("Error calling cloudstore_getChunkBatch: ", "error", resp["err"])
		return fmt.Errorf(resp["err"].(string))
	}
	if resp["chunks"] != nil {
		chunks := resp["chunks"].([]interface{})
		for i, ch := range chunks {
			chunk := ch.(map[string]interface{})
			if chunk["chunkID"] != nil && chunk["val"] != nil {
				chunkID, _ := base64.StdEncoding.DecodeString(chunk["chunkID"].(string))
				val, _ := base64.StdEncoding.DecodeString(chunk["val"].(string))
				found := false
				for _, mch := range reqChunks {
					if bytes.Compare(chunkID, mch.ChunkID) == 0 {
						mch.Value = val
						found = true
					}
				}
				if !found {
					fmt.Printf("[RemoteStorage:GetChunkBatch] chunk %d chunkID %x len(val) %d NOT MATCHED\n", i, chunkID, len(val))
				}
			}
		}
	} else {
		fmt.Printf("RemoteStorage:GetChunkBatch] NOCHUNKS\n")
	}
	return nil
}

func (self *RemoteStorage) GetShare(chunkID []byte) (chunk []byte, err error) {
	var resp map[string]interface{}
	//log.Info(fmt.Sprintf("remoteStorage:GetChunk chunkID [%+v](%x)", chunkID, chunkID))
	chunkIDHash := common.BytesToHash(chunkID)
	sig := self.Sign(&chunkIDHash)
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	err = self.rpcclient.CallContext(ctx, &resp, "cloudstore_getShare", self.blockchainId, chunkID, self.tokenId, sig)
	if err != nil {
		log.Info(fmt.Sprintf("[remotestorage:GetShare] Error %+v", err))
		return chunk, err
	}
	if _, ok := resp["v"]; !ok {
		return nil, fmt.Errorf("%s", resp["err"])
	}
	decodedChunk, _ := base64.StdEncoding.DecodeString(resp["v"].(string))
	return []byte(decodedChunk), nil
}

func (self *RemoteStorage) SetShare(chunk []byte) (merkleRoot []byte, chunkKey []byte, err error) {
	var resp map[string]interface{}
	//log.Info(fmt.Sprintf("remoteStorage:GetChunk chunkID [%+v](%x)", chunkID, chunkID))
	chunkID := wolkcommon.Keccak256(chunk)
	chunkIDHash := common.BytesToHash(chunkID)
	sig := self.Sign(&chunkIDHash)

	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	log.Trace("setShare-chunk", "len(chunk)", len(chunk))
	err = self.rpcclient.CallContext(ctx, &resp, "cloudstore_setShare", self.blockchainId, chunk, self.tokenId, sig)
	if err != nil {
		log.Info(fmt.Sprintf("[remotestorage:GetShare] Error %+v", err))
		return merkleRoot, chunkKey, err
	}

	if _, ok := resp["err"]; ok {
		return merkleRoot, chunkKey, err
	}
	if _, ok := resp["mr"]; ok {
		if _, ok := resp["h"]; ok {
			mr0, _ := base64.StdEncoding.DecodeString(resp["mr"].(string))
			h0, _ := base64.StdEncoding.DecodeString(resp["h"].(string))
			return []byte(mr0), []byte(h0), nil
		} else {
			return merkleRoot, chunkKey, fmt.Errorf("No hash")
		}
	}
	return merkleRoot, chunkKey, fmt.Errorf("No merkle root")
}
