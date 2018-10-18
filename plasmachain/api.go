// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.

package plasmachain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wolkdb/go-plasma/deep"
	"github.com/wolkdb/go-plasma/plasmachain/eventlog"
	"github.com/wolkdb/go-plasma/smt"
)

// PublicPlasmaAPI is the collection of Plasma full node APIs
type PublicPlasmaAPI struct {
	b Backend
}

// NewPublicPlasmaAPI creates a new API definition for the full node-
// related public debug methods of the Plasma service.
func NewPublicPlasmaAPI(b Backend) *PublicPlasmaAPI {
	return &PublicPlasmaAPI{b: b}
}

// BlockNumber returns the block number of the chain head.
func (s *PublicPlasmaAPI) SendPlasmaTransaction(ctx context.Context, tx *Transaction) (common.Hash, error) {
	return s.b.SendPlasmaTransaction(tx)
}

func (s *PublicPlasmaAPI) SendRawPlasmaTransaction(ctx context.Context, rawTx hexutil.Bytes) (txhash common.Hash, err error) {
	var plasmaTx *Transaction
	err = rlp.Decode(bytes.NewReader(rawTx), &plasmaTx)
	if err != nil {
		return txhash, err
	}
	return s.b.SendPlasmaTransaction(plasmaTx)
}

func (s *PublicPlasmaAPI) GetPlasmaBalance(ctx context.Context, addr common.Address, blockNumber rpc.BlockNumber) OrderedMap {

	if blockNumber == rpc.LatestBlockNumber {
		rawBlockNumber, _ := s.b.LatestBlockNumber()
		//TODO: check for error
		blockNumber = rpc.BlockNumber(rawBlockNumber)
	}

	o := make(map[string]interface{})
	acct, err := s.b.GetPlasmaBalance(addr, blockNumber)
	if err != nil {
		o["err"] = fmt.Sprintf("%v", err)
	} else {
		o["account"] = acct
	}
	om := ConvertToOrderMap(o)
	return om
}

func (s *PublicPlasmaAPI) GetPlasmaBlock(ctx context.Context, blockNumber rpc.BlockNumber) (om OrderedMap) {

	if blockNumber == rpc.LatestBlockNumber {
		rawBlockNumber, _ := s.b.LatestBlockNumber()
		//TODO: check for error
		blockNumber = rpc.BlockNumber(rawBlockNumber)
	}

	o := make(map[string]interface{})
	b := s.b.GetPlasmaBlock(blockNumber)
	if b != nil {
		//o = b.OutputBlock(false)
		o["block"] = b.header
	}
	om = ConvertToOrderMap(o)
	return om
}

func (s *PublicPlasmaAPI) GetPlasmaToken(ctx context.Context, tokenID hexutil.Uint64, blockNumber rpc.BlockNumber) (om OrderedMap) {

	if blockNumber == rpc.LatestBlockNumber {
		rawBlockNumber, _ := s.b.LatestBlockNumber()
		//TODO: check for error
		blockNumber = rpc.BlockNumber(rawBlockNumber)
	}

	o := make(map[string]interface{})
	t, tinfo, err := s.b.GetPlasmaToken(tokenID, blockNumber)
	if err != nil {
		o["err"] = fmt.Sprintf("%v", err)
	} else {
		o["token"] = t
		o["tokenInfo"] = tinfo
	}

	om = ConvertToOrderMap(o)
	return om
}

func (s *PublicPlasmaAPI) GetPlasmaBloomFilter(ctx context.Context, hash common.Hash) (om OrderedMap) {
	o := make(map[string]interface{})
	bloomByte, err := s.b.GetPlasmaBloomFilter(hash)
	if err != nil {
		o["err"] = err
	} else if len(bloomByte) == 0 {
		o["err"] = fmt.Sprint("not found")
	} else {
		o["filter"] = fmt.Sprintf("%x", bloomByte)
	}
	om = ConvertToOrderMap(o)
	return om
}

func (s *PublicPlasmaAPI) GetPlasmaTransactionReceipt(ctx context.Context, hash common.Hash) (om OrderedMap) {
	tx, blockNumber, receipt, err := s.b.GetPlasmaTransactionReceipt(hash)
	o := make(map[string]interface{})
	if err != nil {
		if err.Error() == "Looking for AnchorTX?" {
			o["err"] = fmt.Sprintf("%v", err)
		}
	} else {
		if blockNumber != 0 {
			o["plasmatransaction"] = tx
			o["blockNumber"] = blockNumber
			o["receipt"] = receipt
		}
	}
	om = ConvertToOrderMap(o)
	return om
}

func (s *PublicPlasmaAPI) GetPlasmaTransactionProof(ctx context.Context, hash common.Hash) map[string]interface{} {
	tokenID, txbyte, proof, blockNum, err := s.b.GetPlasmaTransactionProof(hash)
	o := make(map[string]interface{})
	if err != nil {
		o["err"] = fmt.Sprintf("%v", err)
	}

	if txbyte != nil && proof != nil {
		o, _ = RPCMarshalProof(tokenID, txbyte, proof, blockNum)
	}
	//om = ConvertToOrderMap(o)
	return o
}

func (s *PublicPlasmaAPI) GetPlasmaExitProof(ctx context.Context, tokenID hexutil.Uint64) OrderedMap {
	o := make(map[string]interface{})
	rawBlockNumber, _ := s.b.LatestBlockNumber()
	t, tinfo, err := s.b.GetPlasmaToken(tokenID, rpc.BlockNumber(rawBlockNumber))
	if err != nil {
		o["err"] = fmt.Sprintf("%v", err)
	}
	if t == nil || tinfo == nil {
		return ConvertToOrderMap(o)
	}

	tID1, txbyte, proof, blk, lastBlk, err := s.b.getProofbyTokenID(uint64(tokenID), t.PrevBlock)
	if lastBlk == 0 {
		o["depositExit"], _ = RPCMarshalDepositProof(txbyte, proof, blk, tinfo.DepositIndex)
		return ConvertToOrderMap(o)
	}

	tID2, prevtxbyte, prevproof, prevBlk, _, err2 := s.b.getProofbyTokenID(uint64(tokenID), lastBlk)

	if err != nil || err2 != nil {
		o["err"] = fmt.Sprintf("tokenID %x not found", uint64(tokenID))
	} else if tID1 != tID2 {
		o["err"] = fmt.Sprintf("tokenID %x mismatch", uint64(tokenID))
	}

	if prevtxbyte != nil && prevproof != nil {
		o["prev"], _ = RPCMarshalSMTProof(prevtxbyte, prevproof, prevBlk)
	}

	if txbyte != nil && proof != nil {
		o["curr"], _ = RPCMarshalSMTProof(txbyte, proof, blk)
	}

	om := ConvertToOrderMap(o)
	return om
}

func (s *PublicPlasmaAPI) GetPlasmaTransactionPool(ctx context.Context) (txs map[common.Address][]*Transaction) {
	return s.b.GetPlasmaTransactionPool()
}

func (s *PublicPlasmaAPI) GetBlockchainID(ctx context.Context, tokenID hexutil.Uint64, blockchainNonce hexutil.Uint64) hexutil.Uint64 {
	return hexutil.Uint64(deep.GetBlockchainID(uint64(tokenID), uint64(blockchainNonce)))
}

// Anchor Transactions
func (s *PublicPlasmaAPI) SendAnchorTransaction(ctx context.Context, tx *deep.AnchorTransaction) (common.Hash, error) {
	return s.b.SendAnchorTransaction(tx)
}

func (s *PublicPlasmaAPI) SendRawAnchorTransaction(ctx context.Context, rawTx hexutil.Bytes) (txhash common.Hash, err error) {
	var anchorTx *deep.AnchorTransaction
	err = rlp.Decode(bytes.NewReader(rawTx), &anchorTx)
	if err != nil {
		return txhash, err
	}
	return s.b.SendAnchorTransaction(anchorTx)
}

func (s *PublicPlasmaAPI) GetAnchorTransactionPool(ctx context.Context) (txs map[uint64][]*deep.AnchorTransaction) {
	return s.b.GetAnchorTransactionPool()
}

func (s *PublicPlasmaAPI) GetAnchor(chainID hexutil.Uint64, blockNumber rpc.BlockNumber, tokenID hexutil.Uint64, sig hexutil.Bytes) (o map[string]interface{}) {

	if blockNumber == rpc.LatestBlockNumber {
		rawBlockNumber, _ := s.b.LatestBlockNumber()
		//TODO: check for error
		blockNumber = rpc.BlockNumber(rawBlockNumber)
	}

	o = make(map[string]interface{})
	v, p, pending, err := s.b.GetAnchor(uint64(chainID), blockNumber, uint64(tokenID), sig)
	if err != nil {
		o["err"] = fmt.Sprintf("%v", err)
	} else {
		o["v"] = v
		if p != nil {
			o["proof"] = p.String()
		}
		o["pending"] = pending
	}
	return o
}

func (s *PublicPlasmaAPI) EventHandler(event interface{}) (err error) {
	eventMap := event.(map[string]interface{})
	eventType, ok := eventMap["event"].(string)
	if !ok {
		return fmt.Errorf("Invalid Event Type")
	}

	if eventMap["log"] == nil {
		return fmt.Errorf("Invalid Event log: log is empty")
	}
	eventLog := eventMap["log"]

	switch eventType {
	case "Deposit":
		//{"depositor":"0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","depositIndex":"0x19","denomination":"0xde0b6b3a7640000","tokenID":"0xe6a1a18a43b04212"}

		var depositEvent *eventlog.DepositEvent
		resbyte, _ := json.Marshal(eventLog)
		err = json.Unmarshal(resbyte, &depositEvent)
		if err != nil {
			return fmt.Errorf("[%v] Invalid log %v\n", eventType, eventLog)
		}
		log.Info("EventHandler", "Event", eventType, "log", string(resbyte))
		tokenInfo := &TokenInfo{
			Depositor:    depositEvent.Depositor,
			Denomination: depositEvent.Denomination,
			DepositIndex: depositEvent.DepositIndex,
		}
		err = s.b.processDeposit(deep.UInt64ToByte(depositEvent.TokenID), tokenInfo)
		if err != nil {
			log.Info("Error encountered processing Deposit: ", "error", err)
		}

	case "StartExit":
		//{"exiter":"0xa45b77a98e2b840617e2ec6ddfbf71403bdcb683","depositIndex":"0x0","denomination":"0xde0b6b3a7640000","tokenID":"0xb437230feb2d24db","timestamp":"0x5bb54fc1"}

		var startExitEvent *eventlog.StartExitEvent
		resbyte, _ := json.Marshal(eventLog)
		err = json.Unmarshal(resbyte, &startExitEvent)
		if err != nil {
			return fmt.Errorf("[%v] Invalid log %v\n", eventType, eventLog)
		}
		log.Info("EventHandler", "Event", eventType, "log", string(resbyte))
		//fmt.Printf("Event %v %+v", eventType, startExitEvent)
		err = s.b.startExit(startExitEvent.Exiter, startExitEvent.Denomination, startExitEvent.DepositIndex, startExitEvent.TokenID, startExitEvent.TS)
		if err != nil {
			log.Info("Error encountered startExit: ", "error", err)
		}

	case "PublishedBlock":
		//{"rootHash":"0x82da88c31e874c678d529ad51e43de3a4baf3914","currentDepositIndex":"0x15","blkNum":"0x15"}
		var publishBlockEvent *eventlog.PublishedBlockEvent
		resbyte, _ := json.Marshal(eventLog)
		err = json.Unmarshal(resbyte, &publishBlockEvent)
		if err != nil {
			return fmt.Errorf("[%v] Invalid log %v\n", eventType, eventLog)
		}
		log.Info("EventHandler", "Event", eventType, "log", string(resbyte))
		//fmt.Printf("Event %v %+v", eventType, publishBlockEvent)
		err = s.b.publishedBlock(publishBlockEvent.RootHash, publishBlockEvent.CurrentDepositIndex, publishBlockEvent.Blocknumber)
		if err != nil {
			log.Info("Error encountered publishedBlock: ", "error", err)
		}

	case "FinalizedExit":
		//{"exiter":"0x74f978a3e049688777e6120d293f24348bde5fa6","depositIndex":"0x4","denomination":"0x3782dace9d900000","tokenID":"0x7c00dfa72e8832ed","timestamp":"0x5bb54ffe"}
		var finalizedExitEvent *eventlog.FinalizedExitEvent
		resbyte, _ := json.Marshal(eventLog)
		err = json.Unmarshal(resbyte, &finalizedExitEvent)
		if err != nil {
			return fmt.Errorf("[%v] Invalid log %v\n", eventType, eventLog)
		}
		log.Info("EventHandler", "Event", eventType, "log", string(resbyte))
		//fmt.Printf("Event %v %+v", eventType, finalizedExitEvent)
		err = s.b.finalizedExit(finalizedExitEvent.Exiter, finalizedExitEvent.Denomination, finalizedExitEvent.DepositIndex, finalizedExitEvent.TokenID, finalizedExitEvent.TS)
		if err != nil {
			log.Info("Error encountered finalizedExit: ", "error", err)
		}

	case "Challenge":
		//{"challenger":"0xbef06cc63c8f81128c26efedd461a9124298092b","tokenID":"0x9af84bc1208918b","timestamp":"0x5bb54fc2"}
		var challengeEvent *eventlog.ChallengeEvent
		resbyte, _ := json.Marshal(eventLog)
		err = json.Unmarshal(resbyte, &challengeEvent)
		if err != nil {
			return fmt.Errorf("[%v] Invalid log %v\n", eventType, eventLog)
		}
		log.Info("EventHandler", "Event", eventType, "log", string(resbyte))
		//fmt.Printf("Event %v %+v", eventType, challengeEvent)
		err = s.b.challenge(challengeEvent.Challenger, challengeEvent.TokenID, challengeEvent.TS)
		if err != nil {
			log.Info("Error encountered challenge: ", "error", err)
		}

	case "ExitStarted":
	case "ExitCompleted":
	case "CurrtentExit":
	case "ExitTime":
		log.Info("EventHandler", "Test Event Type", eventType, "log", eventLog)
	default:
		log.Info("EventHandler", "Unrecognized Event Type", eventType, "log", eventLog)
	}
	return nil
}

type KeyVal struct {
	Key string
	Val interface{}
}

//TODO: Define an ordered map
type OrderedMap []KeyVal

func (omap OrderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("{")
	for i, kv := range omap {
		if i != 0 {
			buf.WriteString(",")
		}
		// marshal key
		key, err := json.Marshal(kv.Key)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteString(":")
		// marshal value
		val, err := json.Marshal(kv.Val)
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteString("}")
	return buf.Bytes(), nil
}

func ConvertToOrderMap(unordered map[string]interface{}) OrderedMap {
	var om OrderedMap
	for k, v := range unordered {
		kv := KeyVal{k, v}
		om = append(om, kv)
	}
	return om
}

func RPCMarshalProof(tokenID uint64, txbyte []byte, proof *smt.Proof, blockNumber uint64) (map[string]interface{}, error) {
	proofByte := proof.Bytes() // copies the header once
	fields := map[string]interface{}{
		"tokenID":     hexutil.Uint64(tokenID),
		"blockNumber": hexutil.Uint64(blockNumber),
		"txbyte":      hexutil.Bytes(txbyte),
		"proofByte":   hexutil.Bytes(proofByte),
	}
	return fields, nil
}

func RPCMarshalSMTProof(txbyte []byte, proof *smt.Proof, blockNumber uint64) (map[string]interface{}, error) {
	proofByte := proof.Bytes() // copies the header once
	fields := map[string]interface{}{
		"txbyte":      hexutil.Bytes(txbyte),
		"proofByte":   hexutil.Bytes(proofByte),
		"blockNumber": hexutil.Uint64(blockNumber),
	}
	return fields, nil
}

func RPCMarshalDepositProof(txbyte []byte, proof *smt.Proof, blockNumber uint64, depositIndex uint64) (map[string]interface{}, error) {
	proofByte := proof.Bytes() // copies the header once
	fields := map[string]interface{}{
		"txbyte":       hexutil.Bytes(txbyte),
		"proofByte":    hexutil.Bytes(proofByte),
		"blockNumber":  hexutil.Uint64(blockNumber),
		"depositIndex": hexutil.Uint64(depositIndex),
	}
	return fields, nil
}

/*
func (s *PublicPlasmaAPI) rpcOutputBlock(b *types.Block, inclTx bool, fullTx bool) (map[string]interface{}, error) {
	fields, err := RPCMarshalProof(b, inclTx, fullTx)
	if err != nil {
		return nil, err
	}
	fields["totalDifficulty"] = (*hexutil.Big)(s.b.GetTd(b.Hash()))
	return fields, err
}
*/

// var header *Header
// if resp["block"] != nil {
// 	resbyte, _ := json.Marshal(resp["block"])
// 	_ = json.Unmarshal(resbyte, &header)
// }
