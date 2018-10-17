// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package deep

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/blake2s"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	//"github.com/wolkdb/go-plasma/cloud"
)

const chunkDefaultSize = 4096
const (
	operatorPrivateKey = "6545ddd10c1e0d6693ba62dec711a2d2973124ae0374d822f845d322fb251645"
)

func Blakeb(b []byte) []byte {
	h, _ := blake2s.New256(nil)
	h.Write(b)
	return h.Sum(nil)
}

func generateRandomData(l int) (r io.Reader, slice []byte) {
	slice = make([]byte, l)
	rand.Seed(time.Now().Unix())
	if _, err := rand.Read(slice); err != nil {
		panic("rand error")
	}
	r = io.LimitReader(bytes.NewReader(slice), int64(l))
	return
}

/*
func TestAnchor(t *testing.T) {

	// set up Anchor to L2
	blockchaincnt := uint64(0)
	tokenID := uint64(1234)
	blockchainID := uint64(222) //TODO: use testConfig

	var pkey *ecdsa.PrivateKey
	pkey, err := crypto.HexToECDSA("afc522d1476e251535e53abb92bcfa7cfd7ef5c86442ef95e20714f053551574")
	if err != nil {
		t.Fatalf("HexToECDSA err %v", err)
	}
	a, err := deep.NewAnchorStorage(blockchainID, tokenID, pkey)
	if err != nil {
		t.Fatalf("HexToECDSA err %v", err)
	}

	blockHashes := make(map[uint64]*common.Hash)
	nblocks := uint64(10)
	for blockNumber := uint64(0); blockNumber <= nblocks; blockNumber++ {
		blockHash := common.BytesToHash(deep.Keccak256([]byte(fmt.Sprintf("test%d", blockNumber))))
		blockHashes[blockNumber] = &blockHash

		tx := a.CreateAnchorTransaction(blockNumber, &blockHash)
		tx.AddOwner(common.HexToAddress("0xba3e1663ada8f3e4e1574bab2494c727d7cb81aa"))
		tx.AddOwner(common.HexToAddress("0xba3e1663ada8f3e4e1574bab2494c727d7cb81aa"))
		tx.AddOwner(common.HexToAddress("0xba3e1663ada8f3e4e1574bab2494c727d7cb81ab"))
		tx.RemoveOwner(common.HexToAddress("0xba3e1663ada8f3e4e1574bab2494c727d7cb81ab"))
		err := a.SendAnchorTransaction(tx)
		if err != nil {
			fmt.Printf("SendAnchorTransaction ERROR: %v", err)
		}
	}
}

func TestGetAnchor(t *testing.T) {
}

func TestSetChunk(t *testing.T) {
}

func TestGetChunk(t *testing.T) {
}
*/

const (
	DefaultEndpointUrl = "http://c0.wolk.com:9900"
)

type Config struct {
	PlasmaAddr string
	PlasmaPort uint64

	CloudstoreAddr string
	CloudstorePort uint64

	DataDir        string
	RemoteDisabled bool
}

// DefaultConfig contains default settings for use on the Ethereum main net.
var DefaultConfig = &Config{
	PlasmaAddr:     "plasma.wolk.com",
	PlasmaPort:     80,
	CloudstoreAddr: "c0.wolk.com",
	CloudstorePort: 9900,
	DataDir:        "/tmp/remtest",
	RemoteDisabled: false,
}

// LocalTestConfig contains settings to test integration in local enviroments
var LocalTestConfig = &Config{
	PlasmaAddr:     "plasma.wolk.com",
	PlasmaPort:     80,
	CloudstoreAddr: "c0.wolk.com",
	CloudstorePort: 9900,
	DataDir:        "/tmp/remtest",
	RemoteDisabled: true,
}

func (c *Config) GetPlasmaAddr() string     { return c.PlasmaAddr }
func (c *Config) GetPlasmaPort() uint64     { return c.PlasmaPort }
func (c *Config) GetCloudstoreAddr() string { return c.CloudstoreAddr }
func (c *Config) GetCloudstorePort() uint64 { return c.CloudstorePort }
func (c *Config) GetDataDir() string        { return c.DataDir }
func (c *Config) IsLocalMode() bool         { return c.RemoteDisabled }

func newRemoteStorage(config *Config) (rs *RemoteStorage, err error) {
	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))
	_blockchainId := uint64(42)
	//_endpointUrl := DefaultEndpointUrl
	_chainType := "sql"
	//	_tokenID := uint64(1234)
	operatorKey, _ := crypto.HexToECDSA(operatorPrivateKey)
	rs, err = NewRemoteStorage(_blockchainId, config, _chainType, operatorKey)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

func equalChunks(c1 *Chunk, c2 *Chunk) bool {
	if bytes.Compare(c1.Value, c2.Value) != 0 {
		return false
	}
	if bytes.Compare(c1.ChunkID, c2.ChunkID) != 0 {
		return false
	}
	if c1.OK != c2.OK {
		return false
	}
	if c1.Error != c2.Error {
		return false
	}
	return true
}

func TestCreateRemoteStorage(t *testing.T) {

}

func TestSign(t *testing.T) {
	rs, err := newRemoteStorage(DefaultConfig)
	defer rs.Close()
	if err != nil {
		t.Fatalf("[remotestorage_test:TestRemoteSingle] NewRemoteStorage %s", err)
	}
	testChunkIdBytes := []byte("testChunkId")
	testChunkId := common.BytesToHash(testChunkIdBytes)
	actualSignedHash := rs.Sign(&testChunkId)
	if actualSignedHash == nil {
		t.Fatal("NIL signature. BAD!")
	}
	expectedSignedHash := common.Hex2Bytes("df998cd528000d415c664994284e6bf8c1d370cefdd089336cd65f6c674c029356d252ccb60b87b952f92ec42b282976106e482e4e5936d47552e0b30acfffe300")
	if bytes.Compare(actualSignedHash, expectedSignedHash) != 0 {
		t.Fatalf("mismatched sig actual [%+v][%x] vs expected [%+v][%x]", actualSignedHash, actualSignedHash, expectedSignedHash, expectedSignedHash)
	}
}

func TestQueueChunk(t *testing.T) {
	chunkCount := 10
	rs, err := newRemoteStorage(DefaultConfig)
	defer rs.Close()
	if err != nil {
		t.Fatalf("[remotestorage_test:TestRemoteSingle] NewRemoteStorage %s", err)
	}
	var chunkArray []*Chunk

	for i := 0; i < chunkCount; i++ {
		c := &Chunk{
			ChunkID: []byte(fmt.Sprintf("testChunkKey%d_%d", rand.Intn(1000), i)),
			Value:   []byte(fmt.Sprintf("testChunValue%d_%d", rand.Intn(1000), i)),
		}
		log.Info(fmt.Sprintf("SET: Round: %d - ChunkID: %s vs Value: %s", i, c.ChunkID, c.Value))
		chunkArray = append(chunkArray, c)
		rs.QueueChunk(c.ChunkID, c.Value)
	}

	for i, ch := range rs.QueueContents() {
		//log.Info(fmt.Sprintf("GET Round: %d - ChunkArray: id = %+v | %+v vs Queue: %+v | %+v", i, chunkArray[i].ChunkID, chunkArray[i].Value, ch.ChunkID, ch.Value))
		if !equalChunks(chunkArray[i], ch) {
			t.Fatalf("Error: ChunkArray Value: %+v vs Queue Value: %+v", chunkArray[i].Value, ch.Value)
			//TODO create chunk print
		}
	}

	flusherr := rs.Flush()
	if flusherr != nil {
		t.Fatalf("Error flushing")
	}
	if len(rs.QueueContents()) > 0 {
		t.Fatalf("Queue not emptied after flushing")
	}
	//TODO: request from cache and/or cloudstore to confirm the items were stored

	rs.Close()
}

func TestRemoteSingle(t *testing.T) {
	rs, err := newRemoteStorage(DefaultConfig)
	defer rs.Close()
	if err != nil {
		t.Fatalf("[remotestorage_test:TestRemoteSingle] NewRemoteStorage %s", err)
	}

	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 524288, 524288 + 1, 524288 + 4097, 1000000, 7 * 524288, 7*524288 + 1, 7*524288 + 4097}
	sizes = sizes[:5]

	start := time.Now()
	successes := 0
	for _, sz := range sizes {
		_, chunk := generateRandomData(sz)
		chunkID := Blakeb(chunk)
		st := time.Now()
		err := rs.SetChunk(chunkID, chunk)
		if err != nil {
			log.Error("[remotestorage_test:TestRemoteSingle] SetChunk", "sz", sz, "err", err)
		} else {
			log.Info("[remotestorage_test:TestRemoteSingle] SetChunk", "sz", sz, "w", time.Since(st))
		}

		st = time.Now()

		//Check that chunk stored locally/cached successfully
		chunkFromCache, ok, err := rs.GetChunkLocal(chunkID)
		if err != nil {
			log.Error("[remotestorage_test:TestRemoteSingle] GetChunkLocal", "sz", sz, "err", err)
		} else if !ok {
			log.Error("[remotestorage_test:TestRemoteSingle] GetChunkLocal NOT OK", "sz", sz)
		} else if bytes.Compare(chunk, chunkFromCache) != 0 {
			log.Error("[remotestorage_test:TestRemoteSigle] GetChunkLocal compare failure", "sz", sz)
		} else {
			log.Info("[remotestorage_test:TestRemoteSingle] GetChunkLocal", "sz", sz, "r", time.Since(st))
			successes++
		}

		//Check General GetChunk -- should just go to cache and not cloudstore
		chunkR, ok, err := rs.GetChunk(chunkID)
		if err != nil {
			log.Error("[remotestorage_test:TestRemoteSingle] GetChunk", "sz", sz, "err", err)
		} else if !ok {
			log.Error("[remotestorage_test:TestRemoteSingle] GetChunk NOT OK", "sz", sz)
		} else if bytes.Compare(chunk, chunkR) != 0 {
			log.Error("[remotestorage_test:TestRemoteSigle] GetChunk compare failure", "sz", sz)
		} else {
			log.Info("[remotestorage_test:TestRemoteSingle] GetChunk", "sz", sz, "r", time.Since(st))
			successes++
		}
	}
	log.Info("[remotestorage_test:TestRemoteSingle]", "successes", successes, "rw", time.Since(start))
	rs.Close()
}

func TestRemoteBatch(t *testing.T) {
	rs, err := newRemoteStorage(DefaultConfig)
	defer rs.Close()
	if err != nil {
		t.Fatalf("[remotestorage_test:TestRemoteBatch] NewRemoteStorage %s", err)
	}

	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 524288, 524288 + 1}
	sizes = sizes[:15]

	// SetChunkBatch
	chunks := make([]*Chunk, 0)
	for _, sz := range sizes {
		_, v := generateRandomData(sz)
		k := Blakeb(v)
		chunks = append(chunks, &Chunk{ChunkID: k, Value: v})
	}
	start := time.Now()
	log.Info(" === START SetChunkBatch === ")
	err = rs.SetChunkBatch(chunks)
	if err != nil {
		log.Error("[remotestorage_test:TestRemoteBatch] SetChunkBatch", "err", err)
		t.Fatalf("Error setting ChunkBatch %+v", err)
	} else {
		log.Info("[remotestorage_test:TestRemoteBatch] SetChunkBatch", "w", time.Since(start))
	}
	log.Info(" === END SetChunkBatch === ")

	st := time.Now()
	chunksR := make([]*Chunk, 0)
	for i, _ := range sizes {
		chunksR = append(chunksR, &Chunk{ChunkID: chunks[i].ChunkID})
	}

	chunksRLocal := make([]*Chunk, 0)
	for i, _ := range sizes {
		chunksRLocal = append(chunksRLocal, &Chunk{ChunkID: chunks[i].ChunkID})
	}

	//GetChunkBatchLocal
	log.Info(" === START GetChunkBatchLocal === ")
	err = rs.GetChunkBatchLocal(chunksRLocal)
	if err != nil {
		log.Error("[remotestorage_test:TestRemoteBatch] GetChunkBatchLocal", "err", err)
		t.Fatalf("Error getting ChunkBatch %+v", err)
	} else {
		successes := 0
		for i, _ := range sizes {
			if bytes.Compare(chunks[i].Value, chunksRLocal[i].Value) != 0 {
				log.Error("[remotestorage_test:TestRemoteBatch] GetChunkBatchLocal Compare mismatch", "w bytes", len(chunksRLocal[i].Value), "r bytes", len(chunksRLocal[i].Value))
				t.Fatalf("Error GetChunkBatchLocal Compare mismatch set [%+v] vs get [%+v]", chunks[i].Value, chunksRLocal[i].Value)
			} else if chunksRLocal[i].Error != nil {
				log.Error("[remotestorage_test:TestRemoteBatch] GetChunkBatchLocal Error", "err", chunksRLocal[i].Error)
				t.Fatalf("Error Getting BatchChunks expecting Chunk[%d] has an error (%+v)", i, chunksRLocal[i].Error)
			} else if !chunksRLocal[i].OK {
				log.Error("[remotestorage_test:TestRemoteBatch] GetChunkBatchLocal Chunk not found", "chunkId", chunksRLocal[i].ChunkID, "index", i, "isok", chunksRLocal[i].OK, "val", chunksRLocal[i].Value)
				t.Fatalf("Error Getting BatchChunks from cache Chunk[%d] not found ID (%+v) or is not found", i, chunksRLocal[i].ChunkID)
			} else {
				log.Info("[remotestorage_test:TestRemoteBatch] GetChunkBatchLocal SUCCESS")
				successes++
			}
		}
		log.Info("[remotestorage_test:TestRemoteBatch] GetChunkBatch", "r", time.Since(st), "rw", time.Since(start))
	}

	// GetChunkBatch
	err = rs.GetChunkBatch(chunksR)
	if err != nil {
		log.Error("[remotestorage_test:TestRemoteBatch] GetChunkBatch", "err", err)
		t.Fatalf("Error getting ChunkBatch %+v", err)
	} else {
		successes := 0
		for i, _ := range sizes {
			if bytes.Compare(chunks[i].Value, chunksR[i].Value) != 0 {
				log.Error("[remotestorage_test:TestRemoteBatch] GetChunkBatch Compare mismatch", "w bytes", len(chunks[i].Value), "r bytes", len(chunksR[i].Value))
			} else {
				successes++
			}
		}
		log.Info("[remotestorage_test:TestRemoteBatch] GetChunkBatch", "r", time.Since(st), "rw", time.Since(start))
	}
	rs.Close()
}

func TestLocalSingle(t *testing.T) {
	rs, err := newRemoteStorage(LocalTestConfig)
	defer rs.Close()
	if err != nil {
		t.Fatalf("[remotestorage_test:TestLocalSingle] NewRemoteStorage %s", err)
	}

	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 524288, 524288 + 1, 524288 + 4097, 1000000, 7 * 524288, 7*524288 + 1, 7*524288 + 4097}
	sizes = sizes[:5]

	start := time.Now()
	successes := 0
	for _, sz := range sizes {
		_, chunk := generateRandomData(sz)
		chunkID := Blakeb(chunk)
		st := time.Now()
		err := rs.SetChunk(chunkID, chunk)
		if err != nil {
			log.Error("[remotestorage_test:TestLocalSingle] SetChunk", "sz", sz, "err", err)
		} else {
			log.Info("[remotestorage_test:TestLocalSingle] SetChunk", "sz", sz, "w", time.Since(st))
		}

		st = time.Now()

		//Check that chunk stored locally/cached successfully
		chunkFromCache, ok, err := rs.GetChunkLocal(chunkID)
		if err != nil {
			log.Error("[remotestorage_test:TestLocalSingle] GetChunkLocal", "sz", sz, "err", err)
		} else if !ok {
			log.Error("[remotestorage_test:TestLocalSingle] GetChunkLocal NOT OK", "sz", sz)
		} else if bytes.Compare(chunk, chunkFromCache) != 0 {
			log.Error("[remotestorage_test:TestLocalSingle] GetChunkLocal compare failure", "sz", sz)
		} else {
			log.Info("[remotestorage_test:TestLocalSingle] GetChunkLocal", "sz", sz, "r", time.Since(st))
			successes++
		}

		//Check General GetChunk -- should just go to cache and not cloudstore
		chunkR, ok, err := rs.GetChunk(chunkID)
		if err != nil {
			log.Error("[remotestorage_test:TestLocalSingle] GetChunk", "sz", sz, "err", err)
		} else if !ok {
			log.Error("[remotestorage_test:TestLocalSingle] GetChunk NOT OK", "sz", sz)
		} else if bytes.Compare(chunk, chunkR) != 0 {
			log.Error("[remotestorage_test:TestLocalSingle] GetChunk compare failure", "sz", sz)
		} else {
			log.Info("[remotestorage_test:TestLocalSingle] GetChunk", "sz", sz, "r", time.Since(st))
			successes++
		}
	}
	log.Info("[remotestorage_test:TestLocalSingle]", "successes", successes, "rw", time.Since(start))
	rs.Close()
}

func TestLocalBatch(t *testing.T) {
	rs, err := newRemoteStorage(LocalTestConfig)
	defer rs.Close()
	if err != nil {
		t.Fatalf("[remotestorage_test:TestLocalBatch] NewRemoteStorage %s", err)
	}

	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 524288, 524288 + 1}
	sizes = sizes[:15]

	// SetChunkBatch
	chunks := make([]*Chunk, 0)
	for _, sz := range sizes {
		_, v := generateRandomData(sz)
		k := Blakeb(v)
		chunks = append(chunks, &Chunk{ChunkID: k, Value: v})
	}
	start := time.Now()
	log.Info(" === START SetChunkBatch === ")
	err = rs.SetChunkBatch(chunks)
	if err != nil {
		log.Error("[remotestorage_test:TestLocalBatch] SetChunkBatch", "err", err)
		t.Fatalf("Error setting ChunkBatch %+v", err)
	} else {
		log.Info("[remotestorage_test:TestLocalBatch] SetChunkBatch", "w", time.Since(start))
	}
	log.Info(" === END SetChunkBatch === ")

	st := time.Now()
	chunksR := make([]*Chunk, 0)
	for i, _ := range sizes {
		chunksR = append(chunksR, &Chunk{ChunkID: chunks[i].ChunkID})
	}

	chunksRLocal := make([]*Chunk, 0)
	for i, _ := range sizes {
		chunksRLocal = append(chunksRLocal, &Chunk{ChunkID: chunks[i].ChunkID})
	}

	//GetChunkBatchLocal
	log.Info(" === START GetChunkBatchLocal === ")
	err = rs.GetChunkBatchLocal(chunksRLocal)
	if err != nil {
		log.Error("[remotestorage_test:TestLocalBatch] GetChunkBatchLocal", "err", err)
		t.Fatalf("Error getting ChunkBatch %+v", err)
	} else {
		successes := 0
		for i, _ := range sizes {
			if bytes.Compare(chunks[i].Value, chunksRLocal[i].Value) != 0 {
				log.Error("[remotestorage_test:TestLocalBatch] GetChunkBatchLocal Compare mismatch", "w bytes", len(chunksRLocal[i].Value), "r bytes", len(chunksRLocal[i].Value))
				t.Fatalf("Error GetChunkBatchLocal Compare mismatch set [%+v] vs get [%+v]", chunks[i].Value, chunksRLocal[i].Value)
			} else if chunksRLocal[i].Error != nil {
				log.Error("[remotestorage_test:TestLocalBatch] GetChunkBatchLocal Error", "err", chunksRLocal[i].Error)
				t.Fatalf("Error Getting BatchChunks expecting Chunk[%d] has an error (%+v)", i, chunksRLocal[i].Error)
			} else if !chunksRLocal[i].OK {
				log.Error("[remotestorage_test:TestLocalBatch] GetChunkBatchLocal Chunk not found", "chunkId", chunksRLocal[i].ChunkID, "index", i, "isok", chunksRLocal[i].OK, "val", chunksRLocal[i].Value)
				t.Fatalf("Error Getting BatchChunks from cache Chunk[%d] not found ID (%+v) or is not found", i, chunksRLocal[i].ChunkID)
			} else {
				log.Info("[remotestorage_test:TestLocalBatch] GetChunkBatchLocal SUCCESS")
				successes++
			}
		}
		log.Info("[remotestorage_test:TestLocalBatch] GetChunkBatch", "r", time.Since(st), "rw", time.Since(start))
	}

	// GetChunkBatch
	err = rs.GetChunkBatch(chunksR)
	if err != nil {
		log.Error("[remotestorage_test:TestLocalBatch] GetChunkBatch", "err", err)
		t.Fatalf("Error getting ChunkBatch %+v", err)
	} else {
		successes := 0
		for i, _ := range sizes {
			if bytes.Compare(chunks[i].Value, chunksR[i].Value) != 0 {
				log.Error("[remotestorage_test:TestLocalBatch] GetChunkBatch Compare mismatch", "w bytes", len(chunks[i].Value), "r bytes", len(chunksR[i].Value))
			} else {
				successes++
			}
		}
		log.Info("[remotestorage_test:TestLocalBatch] GetChunkBatch", "r", time.Since(st), "rw", time.Since(start))
	}
	rs.Close()
}

func TestCreateAnchorTransaction(t *testing.T) {
	testBytes := []byte("testBytesCreateAnchorTX")
	testHash := common.BytesToHash(testBytes)
	bn := uint64(10)

	rs, err := newRemoteStorage(DefaultConfig)
	defer rs.Close()
	if err != nil {
		t.Fatalf("[TestCreateAnchorTransaction] Error creating remote storage")
	}
	tx := rs.CreateAnchorTransaction(bn, &testHash)
	tx.SignTx(rs.key)
	expectedSig, _ := crypto.Sign(testHash.Bytes(), rs.key)
	if tx.BlockNumber != bn {
		t.Fatalf("[TestCreateAnchorTransaction] Error : Issue with Anchor TX BlockNumber")
	} else if tx.BlockChainID != rs.blockchainId {
		t.Fatalf("[TestCreateAnchorTransaction] Error : Issue with Anchor TX BlockChainID")
	} else if tx.BlockHash != &testHash {
		t.Fatalf("[TestCreateAnchorTransaction] Error : Issue with Anchor TX Hash")
	} else if bytes.Compare(expectedSig, tx.Sig) != 0 {
		//TODO: correctly verify sig: t.Fatalf("[TestCreateAnchorTransaction] Error : Issue with Anchor TX Signature e v[%x] a[%x]", expectedSig, tx.Sig)
	}

	/*
		type AnchorTransaction struct {
			BlockChainID uint64       `json:"blockchainID"    gencodec:"required"`
			BlockNumber  uint64       `json:"blocknumber"     gencodec:"required"`
			BlockHash    *common.Hash `json:"blockhash"       gencodec:"required"`
			Extra        Ownership    `json:"extra"`
			Sig          []byte       `json:"sig"             gencodec:"required"`
		}
	*/
}

func TestGetChunkCloudstore(t *testing.T) {
	rs, err := newRemoteStorage(DefaultConfig)
	defer rs.Close()
	if err != nil {
		t.Fatalf("[TestGetChunkCloudstore] Error creating remote storage")
	}

	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 524288, 524288 + 1, 524288 + 4097, 1000000, 7 * 524288, 7*524288 + 1, 7*524288 + 4097}
	sizes = sizes[:5]

	start := time.Now()
	successes := 0
	for _, sz := range sizes {
		_, chunk := generateRandomData(sz)
		chunkID := Blakeb(chunk)
		st := time.Now()
		// _, ok, err := rs.GetChunkCloudstore(chunkID)
		// if err != nil {
		// 	t.Fatalf("[remotestorage_test:TestGetChunkCloudstore] Error retrieving from Cloudstore: %+v", err)
		// } else if !ok {
		// 	//Expected
		// } else {
		// 	t.Fatalf("[remotestorage_test:TestGetChunkCloudstore] ChunkID not expected to be found : chunkid [%+v](%x)", chunkID, chunkID)
		// }

		err = rs.SetChunk(chunkID, chunk)
		if err != nil {
			log.Error("[remotestorage_test:TestGetChunkCloudstore] SetChunk", "sz", sz, "err", err)
			t.Fatalf("[remotestorage_test:TestGetChunkCloudstore] Error setting to Cloudstore")
		} else {
			log.Info("[remotestorage_test:TestGetChunkCloudstore] SetChunk", "sz", sz, "w", time.Since(st))
		}

		st = time.Now()
		//Check General GetChunk -- should just go to cache and not cloudstore
		// chR, ok, err := rs.GetChunkCloudstore(chunkID)
		// if err != nil {
		// 	t.Fatalf("[remotestorage_test:TestGetChunkCloudstore] Error retrieving from Cloudstore")
		// } else if !ok {
		// 	t.Fatalf("[remotestorage_test:TestGetChunkCloudstore] ChunkID not expected to be found : chunkid [%+v](%x)", chunkID, chunkID)
		// }
		// if bytes.Compare(chR, chunk) != 0 {
		// 	t.Fatalf("[remotestorage_test:TestGetChunkCloudstore] Mismatch data expected (%+v) actual (%+v)", chunk, chR)
		// }
	}
	log.Info("[remotestorage_test:TestGetChunkCloudstore]", "successes", successes, "rw", time.Since(start))
}
