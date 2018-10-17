// Copyright 2018 Wolk Inc.
// This file is part of the Wolk Plasma library.
package deep

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestAnchorRegister(t *testing.T) {

	blockchaincnt := uint64(0)
	tokenID := uint64(0x37b01bd3adfc4ef3)
	blockNumber := uint64(0)
	blockHash := common.BytesToHash(Keccak256([]byte(fmt.Sprintf("test%d", blockNumber))))
	blockchainId := GetBlockchainID(tokenID, blockchaincnt)

	var pkey *ecdsa.PrivateKey
	pkey, err := crypto.HexToECDSA("91dae949703266bcd67ef2dfa9a20d7b62b25520a8375c6a5fad16bd72ae9d4e")
	signer := common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914")
	if err != nil {
		t.Fatalf("HexToECDSA err %v", err)
	}

	tx := NewAnchorTransaction(blockchainId, blockNumber, &blockHash)
	err = tx.SignTx(pkey)
	if err != nil {
		t.Fatalf("SignTx err %v", err)
	} else {
		fmt.Printf("txByte: 0x%v \n", common.Bytes2Hex(tx.Bytes()))
	}
	fmt.Printf("Anchor TX: %v\n", tx)
	addr, err := tx.GetSigner()
	if err != nil {
		t.Fatalf("GetSigner err %v", err)
	} else if bytes.Compare(signer.Bytes(), addr.Bytes()) != 0 {
		t.Fatalf("Invalid Signer ERR [Expected:%x] [Derived: %x] ", signer.Bytes(), addr.Bytes())
	} else {
		fmt.Printf("Signer: %v txHash: %v shortHash: %v signedHash: 0x%v\n", addr.Hex(), tx.Hash().Hex(), tx.ShortHash().Hex(), common.Bytes2Hex(signHash(tx.ShortHash().Bytes())))
	}

	encoded := tx.Bytes()
	var tx2 *AnchorTransaction
	err = rlp.Decode(bytes.NewReader(encoded), &tx2)

	if err != nil {
		t.Fatalf("BytesToAnchorTransaction  err %v", err)
	} else {
		fmt.Printf("Anchor TX Decode: %v\n", tx2)
	}
}

func TestAnchorDemoUpdate(t *testing.T) {

	blockchaincnt := uint64(0)
	tokenID := uint64(0x37b01bd3adfc4ef3)
	blockNumber := uint64(1)
	blockHash := common.BytesToHash(Keccak256([]byte(fmt.Sprintf("test%d", blockNumber))))
	blockchainId := GetBlockchainID(tokenID, blockchaincnt)

	var pkey *ecdsa.PrivateKey
	pkey, err := crypto.HexToECDSA("91dae949703266bcd67ef2dfa9a20d7b62b25520a8375c6a5fad16bd72ae9d4e")
	signer := common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914")
	if err != nil {
		t.Fatalf("HexToECDSA err %v", err)
	}

	tx := NewAnchorTransaction(blockchainId, blockNumber, &blockHash)

	extra := Ownership{
		AddedOwners:   []common.Address{common.HexToAddress("0xba3e1663ada8f3e4e1574bab2494c727d7cb81aa"), common.HexToAddress("0xba3e1663ada8f3e4e1574bab2494c727d7cb81bb")},
		RemovedOwners: []common.Address{common.HexToAddress("0xba3e1663ada8f3e4e1574bab2494c727d7cb81aa"), common.HexToAddress("0xba3e1663ada8f3e4e1574bab2494c727d7cb81cc"), common.HexToAddress("0xba3e1663ada8f3e4e1574bab2494c727d7cb81dd")},
	}
	tx.Extra = extra
	err = tx.SignTx(pkey)
	if err == nil {
		t.Fatalf("ValidateExtraData ERR %v", tx)
	} else {
		fmt.Print("ValidateExtraData Check OK\n")
	}

	tx.Extra = Ownership{}

	tx.AddOwner(common.HexToAddress("0x3088666E05794d2498D9d98326c1b426c9950767"))
	tx.AddOwner(common.HexToAddress("0x3088666E05794d2498D9d98326c1b426c9950767"))
	tx.AddOwner(common.HexToAddress("0xBef06CC63C8f81128c26efeDD461A9124298092b"))
	tx.RemoveOwner(common.HexToAddress("0xBef06CC63C8f81128c26efeDD461A9124298092b"))

	err = tx.SignTx(pkey)
	if err != nil {
		t.Fatalf("SignTx err %v", err)
	} else {
		fmt.Printf("txByte: 0x%v \n", common.Bytes2Hex(tx.Bytes()))
	}
	fmt.Printf("Anchor TX: %v\n", tx)
	addr, err := tx.GetSigner()
	if err != nil {
		t.Fatalf("GetSigner err %v", err)
	} else if bytes.Compare(signer.Bytes(), addr.Bytes()) != 0 {
		t.Fatalf("Invalid Signer ERR [Expected:%x] [Derived: %x] ", signer.Bytes(), addr.Bytes())
	} else {
		fmt.Printf("Signer: %v txHash: %v shortHash: %v signedHash: 0x%v\n", addr.Hex(), tx.Hash().Hex(), tx.ShortHash().Hex(), common.Bytes2Hex(signHash(tx.ShortHash().Bytes())))
	}

	encoded := tx.Bytes()
	var tx2 *AnchorTransaction
	err = rlp.Decode(bytes.NewReader(encoded), &tx2)

	if err != nil {
		t.Fatalf("BytesToAnchorTransaction  err %v", err)
	} else {
		fmt.Printf("Anchor TX Decode: %v\n", tx2)
	}
}
