package wolklog

import (
	"log/syslog"
	"runtime"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/ethereum/go-ethereum/log"
)

const DEBUG = "wolk-debug"
const CLOUD = "wolk-cloud"

type WolkCloudLog struct {
	Chain         string      `json:"chain"`
	ChainId       uint64      `json:"cid"`
	SetKey        int         `json:"sk,omitempty"`
	GetKey        int         `json:"gk,omitempty"`
	SqlRead       int         `json:"sr,omitempty"`
	SqlWrite      int         `json:"sw,omitempty"`
	SetChunkBatch int         `json:"scb,omitempty"`
	SetChunk      int         `json:"sc,omitempty"`
	GetChunk      int         `json:"gc,omitempty"`
	PlasmaTx      int         `json:"tp,omitempty"`
	AnchorTx      int         `json:"ta,omitempty"`
	MintBlock     int         `json:"mb,omitempty"`
	TxHash        common.Hash `json:"txhash,omitempty"`
	Label         string      `json:"label,omitempty"`
	Host          string      `json:"host,omitempty"`
	Duration      uint64      `json:"duration,omitempty"`
	BlockHash     common.Hash `json:"blockhash,omitempty"`
	ParentHash    common.Hash `json:"parenthash,omitempty"`
	BlockNumber   uint64      `json:"bn,omitempty"`
	ChunkId       string      `json:"chunkid,omitempty"`
	Provider      string      `json:"provider,omitempty"`
	Bytes         uint64      `json:"bytes,omitempty"`
	Operator      int         `json:"operatorid,omitempty"`
	Timestamp     uint64      `json:"timestamp,omitempty"`
	Extra         string      `json:"extra,omitempty"`
}

/* Example Usage:

logLineObject := &WolkCloudLog{
       Chain:   "sql",
       ChainId: 77,
       SetChunk: 1,
       Label: "cloudster.SetChunk",
       Host: 127.0.0.1,
       ChunkId: "91dae949703266bcd67ef2dfa9a20d7b62b25520a8375c6a5fad16bd72ae9d4e",
       Bytes: 4096,
       Timestamp: uint64(time.Now().UnixNano()),
}

logLineString, _ := json.Marshal(logLineObject)
wolklog.SysLog(string(logLineString), wolklog.CLOUD)

*/

func SysLog(info string, path string) error {

	//path = wolk-cloud for timing lines
	//path = wolk-debug for debug lines

	thisOS := runtime.GOOS
	if thisOS != "linux" {
		return nil
	}
	logger, err := syslog.Dial("tcp", "127.0.0.1:5000", syslog.LOG_ERR, path) // connection to a log daemon

	if err != nil {
		log.Debug("[handler.go SysLog] syslog.Dial", "Error", err)
		return err
	}
	defer logger.Close()
	syslogerr := logger.Info(info)
	if syslogerr != nil {
		log.Debug("[handler.go SysLog] logger.Info", "Error", syslogerr)
	}

	return nil
}

/*
{
	“chain”:”sql”,
"sc”: //SetChunk
"gc”: //GetChunk
“tp”: //plasma tx
“ta”: //anchor tx
“mb”: //mint block
"blocknumber":111,
  	"cid":1, //blockchainid
  	"txhash": "0x11...",
  	"label":"SQLMutate",
  	"host":127.0.0.1,
  	"extra":"extra stuff",
	“ms”: 33 //Blocking calls have execution timings measured
      "blockhash": "0b11...", //hash not always there, so when applicable
      "parenthash": "0a11...", //maybe useful given inconsistency of blockhash
      "label":"minter.GetTransactions",
      "host":127.0.0.1,
      "chunkid": "0c11..."
      "provider":"amazon",
      "bytes":4096,
      "Operatorid":1, //plasma relevant
  	"timestamp": 111111 //floating point
}

*/
