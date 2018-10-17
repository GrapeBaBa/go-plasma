package wolk

import (
	"bytes"
	"crypto/rand"
	"io"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/wolkdb/cloudstore/wolk/cloud"
	"golang.org/x/crypto/blake2s"
)

const (
	DefaultEndpointUrl = "http://localhost:9900"
)

func newRemoteStorage(t *testing.T) *RemoteStorage {
	log.Root().SetHandler(log.CallerFileHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true)))))
	_blockchainId := uint64(42)
	_endpointUrl := DefaultEndpointUrl
	_chainType := "sql"
	_tokenID := uint64(1234)

	rs, err := NewRemoteStorage(_blockchainId, _endpointUrl, _chainType, _tokenID)
	if err != nil {
		t.Fatalf("[remotestorage_test:TestRemoteSingle] NewRemoteStorage %s", err)
	}
	return rs
}

const chunkDefaultSize = 4096

func Blakeb(b []byte) []byte {
	h, _ := blake2s.New256(nil)
	h.Write(b)
	return h.Sum(nil)
}

func generateRandomData(l int) (r io.Reader, slice []byte) {
	slice = make([]byte, l)
	if _, err := rand.Read(slice); err != nil {
		panic("rand error")
	}
	r = io.LimitReader(bytes.NewReader(slice), int64(l))
	return
}

func TestRemoteShare(t *testing.T) {
	start := time.Now()
	var rs StorageLayer
	rs = newRemoteStorage(t)
	NCHUNKS := 100
	successes := 0
	for num := 0; num < NCHUNKS; num++ {
		chunk := []byte("abcdefghijklmnopqrstuvwxyz")
		_, chunk = generateRandomData(chunkDefaultSize)

		// SetShare
		_, key, err := rs.SetShare(chunk)
		if err != nil {
			log.Error("[remotestorage_test:TestRemoteShare] SetShare", "err", err)
		}
		// GetShare
		_, err2 := rs.GetShare(key)
		if err2 != nil {
			log.Error("[remotestorage_test:TestRemoteShare] SetShare", "err", err)
		} else {
			successes++
		}
	}
	log.Info("[remotestorage_test:TestRemoteShare]", "successes", successes, "rw", time.Since(start))
}

func TestRemoteSingle(t *testing.T) {
	rs := newRemoteStorage(t)
	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 524288, 524288 + 1, 524288 + 4097, 1000000, 7 * 524288, 7*524288 + 1, 7*524288 + 4097}
	sizes = sizes[:15]

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
		chunkR, err := rs.GetChunk(chunkID)
		if err != nil {
			log.Error("[remotestorage_test:TestRemoteSingle] GetChunk", "sz", sz, "err", err)
		}
		if bytes.Compare(chunk, chunkR) != 0 {
			log.Error("[remotestorage_test:TestRemoteSigle] GetChunk compare failure", "sz", sz)
		} else {
			log.Info("[remotestorage_test:TestRemoteSingle] GetChunk", "sz", sz, "r", time.Since(st))
			successes++
		}
	}
	log.Info("[remotestorage_test:TestRemoteSingle]", "successes", successes, "rw", time.Since(start))
}

func TestRemoteBatch(t *testing.T) {
	rs := newRemoteStorage(t)
	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 524288, 524288 + 1}
	sizes = sizes[:15]

	// SetChunkBatch
	chunks := make([]*cloud.RawChunk, 0)
	for _, sz := range sizes {
		_, v := generateRandomData(sz)
		k := Blakeb(v)
		chunks = append(chunks, &cloud.RawChunk{ChunkID: k, Value: v})
	}
	start := time.Now()
	err := rs.SetChunkBatch(chunks)
	if err != nil {
		log.Error("[remotestorage_test:TestRemoteBatch] SetChunkBatch", "err", err)
	} else {
		log.Info("[remotestorage_test:TestRemoteBatch] SetChunkBatch", "w", time.Since(start))
	}

	// GetChunkBatch
	st := time.Now()
	chunksR := make([]*cloud.RawChunk, 0)
	for i, _ := range sizes {
		chunksR = append(chunksR, &cloud.RawChunk{ChunkID: chunks[i].ChunkID})
	}
	err = rs.GetChunkBatch(chunksR)
	if err != nil {
		log.Error("[remotestorage_test:TestRemoteBatch] GetChunkBatch", "err", err)
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
}
