// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.

package plasmachain

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/wolkdb/go-plasma/deep"
)

func TestPlasmaChainInternal(t *testing.T) {

	const (
		tokensPerBlock = 5
		numBlocks      = 10
	)

	const (
		submitBlocks   = false
		withdrawTokens = false
		checkProofs    = true
		showProofs     = true
		simulated      = true
		verbose        = false
		sleeptime      = 5000
	)

	testAcct := map[string]string{
		"0xA45b77a98E2B840617e2eC6ddfBf71403bdCb683": "6545ddd10c1e0d6693ba62dec711a2d2973124ae0374d822f845d322fb251645",
		"0x82Da88C31E874C678D529ad51E43De3A4BAF3914": "91dae949703266bcd67ef2dfa9a20d7b62b25520a8375c6a5fad16bd72ae9d4e",
		"0x3088666E05794d2498D9d98326c1b426c9950767": "221d9cd60bd75fe3790b6f192df30d4c4e432b39870603dd4f06d66cdd8118b7",
		"0xBef06CC63C8f81128c26efeDD461A9124298092b": "554d5b4de75613c7733f0fa96ed0fb7cfc6d01923c31dcaa0c1fef7964552ef9",
		"0x74f978A3E049688777E6120D293F24348BDe5fA6": "1284cfe632342f62401ec83d13f8b0fb2ea51ab41e83e16a3537c46f8aa4f09d",
		"0x59B66c66b9159b62DaFCB5fEde243384DFca076D": "1ab79ae59bb651d4a68e0efbf2d924ef28fc2bfe703b7a356ec2af61d1c20a11",
		"0xf1F75F61078736aaaC75136Ea049FfcC6C5B1450": "9704bb9baccda90c240064617553e713433b9a0a228ef3c5a9fb18ae1e090e4b",
	}

	testKeys, testAddrs := loadTestkeys(testAcct)

	if verbose {
		hs := log.StreamHandler(os.Stderr, log.TerminalFormat(true))
		loglevel := log.LvlTrace
		hf := log.LvlFilterHandler(loglevel, hs)
		h := log.CallerFileHandler(hf)
		log.Root().SetHandler(h)
	}

	ctx := &node.ServiceContext{}
	chain, err := New(ctx, &LocalTestConfig, simulated)
	if err != nil {
		t.Fatalf("Plasmachain err: %v", err)
	}

	_, err = deep.NewPOA(ctx, nil, chain, 0)

	//chain.initDeposit(maxTokens, simulated)
	var currentBlockNum uint64
	currentBlockNum = chain.CurrentBlock().Number() + 1
	fmt.Printf("\n\n========= Block #%d Start =========\n", currentBlockNum)

	pendingTx := chain.GetPlasmaTransactionPool()
	txList, _ := json.Marshal(pendingTx)

	fmt.Printf("\n\n********* Pending PlasmaTransactions ********* \n\n%v\n", string(txList))

	chain.plasmatxpool.MintTestBlock()
	time.Sleep(sleeptime * time.Millisecond)

	fmt.Printf("\n\n********* Mined Block #%d ********* \n\n", currentBlockNum)
	b := chain.GetPlasmaBlock(rpc.BlockNumber(currentBlockNum))

	fmt.Printf("RootChain Contract Call >>> submitBlock(0x%x, %d)\n\n", b.header.TransactionRoot, b.header.BlockNumber)
	if submitBlocks && !simulated {
		chain.publishBlock(b.header.TransactionRoot, b.header.BlockNumber, submitBlocks)
	}

	fmt.Printf("%s\n\n", b)

	for _, acct := range pendingTx {
		for _, tx := range acct {
			chain.generateInternalProof(tx.Hash(), true)
		}
	}

	fmt.Printf("\n========= Block #%d Done =========\n", currentBlockNum)

	var tokenIDList []uint64

	if simulated || true {
		for i := uint64(0); i < 5; i++ {
			var tokenID uint64
			switch i {
			case 0:
				tokenID, _ = TokenInit(0, 1000000000000000000, common.HexToAddress("0xA45b77a98E2B840617e2eC6ddfBf71403bdCb683"))
				//tokenID, _ = TokenInit(0, 18446744073709551614, common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914"))
			case 1:
				tokenID, _ = TokenInit(1, 1000000000000000000, common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914"))
				//tokenID, _ = TokenInit(1, 18446744073709551614, common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914"))
			case 2:
				tokenID, _ = TokenInit(2, 2000000000000000000, common.HexToAddress("0x3088666E05794d2498D9d98326c1b426c9950767"))
			case 3:
				tokenID, _ = TokenInit(3, 3000000000000000000, common.HexToAddress("0xBef06CC63C8f81128c26efeDD461A9124298092b"))
			case 4:
				tokenID, _ = TokenInit(4, 4000000000000000000, common.HexToAddress("0x74f978A3E049688777E6120D293F24348BDe5fA6"))
			}
			tokenIDList = append(tokenIDList, tokenID)
		}
	} else {
		// build tokenIDList from event
	}

	for chain.CurrentBlock().Number() < numBlocks {

		cnt := 0
		currentBlockNum = chain.CurrentBlock().Number() + 1
		var txHashList = make(map[uint64]*common.Hash)

		fmt.Printf("\n\n========= Block #%d Start =========\n", currentBlockNum)

		for _, tokenID := range tokenIDList {
			randomTransact := uint64(currentBlockNum+tokenID) % 3
			if randomTransact == 1 {
				txhash, err := chain.internalAutoSign(tokenID, testKeys, testAddrs)
				if err != nil {
					t.Fatalf("internalAutoSign err: %v\n", err)
				}
				cnt = cnt + 1
				txHashList[tokenID] = txhash
			}
		}

		if cnt == 0 {
			reserveToken := tokenIDList[currentBlockNum%uint64(len(tokenIDList))]
			txhash, err := chain.internalAutoSign(reserveToken, testKeys, testAddrs)
			if err != nil {
				t.Fatalf("internalAutoSign err: %v\n", err)
			}
			txHashList[reserveToken] = txhash
		}

		pendingTx := chain.GetPlasmaTransactionPool()
		txList, _ := json.Marshal(pendingTx)
		fmt.Printf("\n\n********* Pending PlasmaTransactions ********* \n\n%v\n", string(txList))

		chain.plasmatxpool.MintTestBlock()
		time.Sleep(sleeptime * time.Millisecond)

		fmt.Printf("\n\n********* Mined Block #%d ********* \n\n", currentBlockNum)

		b := chain.GetPlasmaBlock(rpc.BlockNumber(currentBlockNum))
		fmt.Printf("RootChain Contract Call >>> submitBlock(0x%x, %d)\n\n", b.header.TransactionRoot, b.header.BlockNumber)

		if submitBlocks && !simulated {
			chain.publishBlock(b.header.TransactionRoot, b.header.BlockNumber, submitBlocks)
		}

		fmt.Printf("%s\n\n", b)

		for tokenID, txHash := range txHashList {
			randomCheck := uint64(currentBlockNum+tokenID) % 2
			chain.generateInternalProof(*txHash, randomCheck == 0)
		}

		fmt.Printf("\n========= Block #%d Done =========\n", currentBlockNum)

		if currentBlockNum != chain.CurrentBlock().Number() {
			t.Fatalf("Short circuited - fail to mint block #%v\n", currentBlockNum)
		}
	}
}

func (self *PlasmaChain) generateInternalProof(txHash common.Hash, verbose bool) (err error) {
	tokenID, txbyte, currProof, currBlk, err := self.GetPlasmaTransactionProof(txHash)
	if err != nil {
		return fmt.Errorf("InternalProof err: %v\n", err)
	} else if currProof == nil {
		return fmt.Errorf("[%x] %x Not Found\n", txHash.Bytes(), tokenID)
	}

	var currTx *Transaction
	currTx, err = DecodeRLPTransaction(txbyte)
	if err != nil || currTx == nil {
		return fmt.Errorf("[%x] %x Not Found\n", txHash.Bytes(), tokenID)
	}

	currHeader, err := self.getBlockHeaderByNumber(currBlk)
	if err != nil {
		return fmt.Errorf("currHeader retrieval err: %v\n", err)
	}

	currR := currHeader.TransactionRoot
	if currTx.PrevBlock == 0 {
		//Deposit Exit
		fmt.Printf("Curr TX [txHash:%s] [Root[%d]: 0x%x] [Proof: 0x%x] [Txbytes: 0x%x]\n\n", currTx.Hash().Hex(), currBlk, currR, currProof.Bytes(), currTx.Bytes())
		fmt.Printf("\n\n***** [0x%x] Deposit Exit (Auto Generated for: %s) *****\n\ndepositExit(uint64 depositIndex 0x%x)\n",
			tokenID, currTx.Recipient.Hex(), currTx.DepositIndex)
		return nil
	}
	prevTxHash, prevProof, err := self.getHash(tokenID, currTx.PrevBlock, 0)
	if err != nil {
		return fmt.Errorf("InternalProof prevTx err: %v\n", err)
	}

	var prevTx *Transaction
	prevTx, prevBlk, _, err := self.GetPlasmaTransactionReceipt(common.BytesToHash(prevTxHash))
	if err != nil {
		return fmt.Errorf("prevTx retrieve err: %v\n", err)
	}

	prevHeader, err := self.getBlockHeaderByNumber(prevBlk)
	if err != nil {
		return fmt.Errorf("prevHeader retrieval err: %v\n", err)
	}
	prevR := prevHeader.TransactionRoot

	fmt.Printf("Prev TX [txHash:%s] [Root[%d]: 0x%x] [Proof: 0x%x] [Txbytes: 0x%x]\n\n", prevTx.Hash().Hex(), prevBlk, prevR, prevProof.Bytes(), prevTx.Bytes())
	fmt.Printf("Curr TX [txHash:%s] [Root[%d]: 0x%x] [Proof: 0x%x] [Txbytes: 0x%x]\n\n", currTx.Hash().Hex(), currBlk, currR, currProof.Bytes(), currTx.Bytes())

	if true {
		if !prevProof.Check(prevTx.Hash().Bytes(), prevR.Bytes(), self.DefaultHashes, true) {
			return fmt.Errorf("checkproof failure")
		}
		if !currProof.Check(currTx.Hash().Bytes(), currR.Bytes(), self.DefaultHashes, true) {
			return fmt.Errorf("checkproof failure")
		}
	}

	if verbose {
		fmt.Printf("\n\n***** [0x%x] Smart Exit (Auto Generated for: %s) *****\n\nStartExit(\nbytes  prevtxBytes 0x%x, \nbytes  prevProof   0x%x,\nuint64 prevBlock   %d,\n\nbytes  currtxBytes 0x%x,\nbytes  currProof   0x%x,\nuint64 currBlock   %d)\n\n\n",
			tokenID, currTx.Recipient.Hex(), prevTx.Bytes(), prevProof.Bytes(), prevBlk, currTx.Bytes(), currProof.Bytes(), currBlk)
	} else {
		fmt.Println("")
	}
	return nil
}

func (chain *PlasmaChain) internalAutoSign(tokenID uint64, testKeys map[common.Address]*ecdsa.PrivateKey, testAddrs []common.Address) (txHash *common.Hash, err error) {

	//Initiating non-deposit transaction
	fmt.Printf("\n********* Model Token %x *********\n", tokenID)

	tinfo, err := chain.getTokenInfo(tokenID)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve %x info", tokenID)
	}

	var token *Token
	token, _, err = chain.GetPlasmaToken(hexutil.Uint64(tokenID), rpc.BlockNumber(chain.CurrentBlock().Number()))
	if token == nil {
		return nil, fmt.Errorf("token %x not found", tokenID)
	}

	currentOwner := token.Owner
	nonce := uint64(chain.CurrentBlock().Number()+tokenID) % uint64(len(testAddrs))
	recipient := testAddrs[nonce]
	fmt.Printf("TOKEN %s CURRENT OWNER: %x => RECIPIENT: %x\n", token, currentOwner, recipient)

	tx := NewTransaction(token, tinfo, &recipient)
	tx.PrevBlock = token.PrevBlock
	if k, ok := testKeys[currentOwner]; ok {
		err := tx.SignTx(k)
		if err != nil {
			return nil, fmt.Errorf("SignTx err: %v\n", err)
		}
		fmt.Printf("txbyte: 0x%x\n", tx.Bytes())

		txhash, err := chain.SendPlasmaTransaction(tx)
		if err != nil {
			return nil, fmt.Errorf("SendPlasmaTransaction err: %v\n", err)
		} else {
			fmt.Printf("SendPlasmaTransaction: %x success!\n", txhash)
			return &txhash, nil
		}
	} else {
		return nil, fmt.Errorf("Key not found for owner: %x", currentOwner)
	}
}

func loadTestkeys(testAcct map[string]string) (map[common.Address]*ecdsa.PrivateKey, []common.Address) {

	testAddrs := []common.Address{}
	testKeys := make(map[common.Address]*ecdsa.PrivateKey)
	for _, tkey := range testAcct {
		pkey, _ := crypto.HexToECDSA(tkey)
		address := crypto.PubkeyToAddress(pkey.PublicKey)
		testKeys[address] = pkey
		testAddrs = append(testAddrs, address)
	}

	return testKeys, testAddrs
}

func hexArr(b []byte) string {
	s := common.Bytes2Hex(b)
	var hexArray []string
	if s[0:2] == "0x" {
		s = s[2:]
	}
	if len(s) == 0 {
		return "[\"\"]"
	} else if len(s)%2 != 0 {
		return "Invalid Hex"
	}

	for i := 0; i < len(s); i += 2 {
		hexArray = append(hexArray, "\"0x"+string(s[i:i+2])+"\"")
	}
	return "[" + strings.Join(hexArray, ",") + "]"
}

func sendPlasmaTransaction(chainrpc *rpc.Client, tx *Transaction) (interface{}, error) {
	log.Info("Backend_test.go:SendAnchorTransaction", "tx", tx)
	var resp interface{}
	err := chainrpc.Call(&resp, "plasma_sendPlasmaTransaction", tx)
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func getPlasmaBlock(chainrpc *rpc.Client, blockNumber int) (*Header, error) {
	var resp = make(map[string]interface{})
	log.Info("Backend_test.go:getPlasmaBlock", "bn", blockNumber)
	err := chainrpc.Call(&resp, "plasma_getPlasmaBlock", hexutil.EncodeUint64(uint64(blockNumber)))
	if err != nil {
		return nil, err
	}
	var header *Header
	if resp["block"] != nil {
		resbyte, _ := json.Marshal(resp["block"])
		_ = json.Unmarshal(resbyte, &header)
	}
	return header, nil
}

func getPlasmaToken(chainrpc *rpc.Client, tokenID uint64, blockNumber int) (*Token, error) {
	var resp = make(map[string]interface{})
	log.Info("Backend_test.go:getPlasmaToken", "bn", blockNumber)
	err := chainrpc.Call(&resp, "plasma_getPlasmaToken", hexutil.EncodeUint64(uint64(tokenID)), hexutil.EncodeUint64(uint64(blockNumber)))
	if err != nil {
		return nil, err
	}
	log.Info("Backend_test.go:getPlasmaToken", "resp", resp)
	var token *Token
	if resp["token"] != nil {
		resbyte, _ := json.Marshal(resp["token"])
		_ = json.Unmarshal(resbyte, &token)
	}
	return token, nil
}
