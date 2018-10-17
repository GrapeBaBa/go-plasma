// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package plasmachain

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/wolkdb/go-plasma/deep"
)

func TestBlock(t *testing.T) {
	var u *Block
	var h Header
	h = Header{
		BlockNumber:     2,
		Time:            987654321,
		TransactionRoot: common.BytesToHash(deep.Keccak256([]byte("0"))),
		BloomID:         common.BytesToHash(deep.Keccak256([]byte("2"))),
		TokenRoot:       common.BytesToHash(deep.Keccak256([]byte("3"))),
		AccountRoot:     common.BytesToHash(deep.Keccak256([]byte("5"))),
		AnchorRoot:      common.BytesToHash(deep.Keccak256([]byte("7"))),
	}
	var b Body

	u = &Block{
		header: &h,
		body:   &b,
	}

	for i := 0; i < 3; i++ {
		denomination := uint64(10000000 * (i + 1))
		depositIndex := uint64(i + 1)
		depositor := common.HexToAddress("0x069984A8D9eBb6B15EB144e0CB01A81B9c7379B0")
		_, tokenInfo := TokenInit(depositIndex, denomination, depositor)
		token := NewToken(depositIndex, denomination, depositor)
		token.PrevBlock = 1
		token.Owner = depositor
		recipient := common.HexToAddress("0xF91A593154C170017FbEd4b0348e259827642bD8")
		tx := NewTransaction(token, tokenInfo, &recipient)
		u.body.Layer2Transactions = append(u.body.Layer2Transactions, tx)
	}
	operatorPrivateKey := "6545ddd10c1e0d6693ba62dec711a2d2973124ae0374d822f845d322fb251645"
	pkey, err := crypto.HexToECDSA(operatorPrivateKey)
	if err != nil {
		t.Fatalf("No privatekey")
	}
	u.SignBlock(pkey)

	/*
	   BlockChainID common.Hash  `json:"blockchainId"` //Hash(TokenID, Number)
	   BlockNumber  uint64       `json:"blocknumber"`
	   BlockHash    *common.Hash `json:"blockhash"` //= Hash(BlockchainID)
	   Sig          []byte       `json:"sig"`       //- Signing of BlockHash as MsgHash using private key of owner of TokenID
	*/

	for i := 0; i < 5; i++ {
		var wtx deep.AnchorTransaction
		wtx.BlockNumber = uint64(101 + i)
		//blockchainName := []byte("testBlockchain")
		//wtx.BlockChainID = common.BytesToHash(deep.Keccak256(blockchainName))
		tokenID := uint64(1234)
		blockchainID := deep.GetBlockchainID(tokenID, 0)
		wtx.BlockChainID = blockchainID
		h := common.BytesToHash(deep.Keccak256(deep.UInt64ToByte(wtx.BlockChainID)))
		wtx.BlockHash = &h
		u.body.AnchorTransactions = append(u.body.AnchorTransactions, &wtx)
	}

	encoded, _ := rlp.EncodeToBytes(u)
	var s Block // []interface{}
	s.header = new(Header)
	s.body = new(Body)
	err = rlp.Decode(bytes.NewReader(encoded), &s)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	out := u.BlockToChunk()
	s2 := FromChunk(out)

	// header checks: u == s && u == s2?
	if u.header.BlockNumber != s.header.BlockNumber || u.header.BlockNumber != s2.header.BlockNumber {
		t.Fatalf("BlockNumber failure %d != %d, %d", u.header.BlockNumber, s.header.BlockNumber, s2.header.BlockNumber)
	}
	if bytes.Compare(u.header.TransactionRoot.Bytes(), s.header.TransactionRoot.Bytes()) != 0 || bytes.Compare(u.header.TransactionRoot.Bytes(), s2.header.TransactionRoot.Bytes()) != 0 {
		t.Fatalf("TransactionRoot failure %x != %x, %x", u.header.TransactionRoot.Bytes(), s.header.TransactionRoot.Bytes(), s2.header.TransactionRoot.Bytes())
	}

	if bytes.Compare(u.header.BloomID.Bytes(), s.header.BloomID.Bytes()) != 0 || bytes.Compare(u.header.BloomID.Bytes(), s2.header.BloomID.Bytes()) != 0 {
		t.Fatalf("BloomID failure %x != %x, %x", u.header.BloomID.Bytes(), s.header.BloomID.Bytes(), s2.header.BloomID.Bytes())
	}
	if bytes.Compare(u.header.TokenRoot.Bytes(), s.header.TokenRoot.Bytes()) != 0 || bytes.Compare(u.header.TokenRoot.Bytes(), s2.header.TokenRoot.Bytes()) != 0 {
		t.Fatalf("TokenRoot failure %x != %x, %x", u.header.TokenRoot.Bytes(), s.header.TokenRoot.Bytes(), s2.header.TokenRoot.Bytes())
	}

	if bytes.Compare(u.header.AccountRoot.Bytes(), s.header.AccountRoot.Bytes()) != 0 || bytes.Compare(u.header.AccountRoot.Bytes(), s2.header.AccountRoot.Bytes()) != 0 {
		t.Fatalf("AccountRoot failure %x != %x, %x", u.header.AccountRoot.Bytes(), s.header.AccountRoot.Bytes(), s2.header.AccountRoot.Bytes())
	}

	if bytes.Compare(u.header.AnchorRoot.Bytes(), s.header.AnchorRoot.Bytes()) != 0 || bytes.Compare(u.header.AnchorRoot.Bytes(), s2.header.AnchorRoot.Bytes()) != 0 {
		t.Fatalf("AnchorRoot failure %x != %x, %x", u.header.AnchorRoot.Bytes(), s.header.AnchorRoot.Bytes(), s2.header.AnchorRoot.Bytes())
	}

	if bytes.Compare(u.header.Sig, s.header.Sig) != 0 || bytes.Compare(u.header.Sig, s2.header.Sig) != 0 {
		t.Fatalf("Sig failure %x != %x, %x", u.header.Sig, s.header.Sig, s2.header.Sig)
	}
	// transactions check: u == s && u == s2?
	if len(u.body.Layer2Transactions) != len(s.body.Layer2Transactions) || len(u.body.Layer2Transactions) != len(s2.body.Layer2Transactions) {
		t.Fatalf("txs failure %d != %d, %d", len(u.body.Layer2Transactions), len(s.body.Layer2Transactions), len(s2.body.Layer2Transactions))
	}
	if len(u.body.AnchorTransactions) != len(s.body.AnchorTransactions) || len(u.body.AnchorTransactions) != len(s2.body.AnchorTransactions) {
		t.Fatalf("AnchorTransactions failure %d != %d, %d", len(u.body.AnchorTransactions), len(s.body.AnchorTransactions), len(s2.body.AnchorTransactions))
	}
	if strings.Compare(s2.String(), s.String()) != 0 {
		t.Fatalf("Mismatched s serialization")
		fmt.Printf("s : %s\n", s.String())
		fmt.Printf("s2: %s\n", s2.String())
	}
	/* TODO
	validated, err := s.ValidateBlock()
	if err != nil || validated == false {
		t.Fatalf("ValidateBlock %v\n", err)
	}
	*/
}
