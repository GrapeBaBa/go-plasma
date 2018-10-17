// Copyright 2018 Wolk Inc.
// This file is part of the Wolk Plasma library.
package plasmachain

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestDeposit(t *testing.T) {
	denomination := uint64(1000000000000000000)
	depositIndex := uint64(1)
	depositor := common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914")
	operator := common.HexToAddress("0xA45b77a98E2B840617e2eC6ddfBf71403bdCb683")

	tokenID, tinfo := TokenInit(depositIndex, denomination, depositor)
	if tokenID != uint64(0x37b01bd3adfc4ef3) {
		t.Fatal("tokenID err")
	}

	token := NewToken(depositIndex, denomination, depositor)
	token.PrevBlock = 0
	token.Owner = operator
	recipient := depositor

	tx := NewTransaction(token, tinfo, &recipient)
	var pkey *ecdsa.PrivateKey
	pkey, _ = crypto.HexToECDSA("91dae949703266bcd67ef2dfa9a20d7b62b25520a8375c6a5fad16bd72ae9d4e")
	err := tx.SignTx(pkey)
	if err != nil {
		t.Fatalf("SignTx err %v", err)
	}

	err = tx.ValidateSig()
	if err == nil {
		t.Fatal("ValidateTx err")
	}

	pkey, _ = crypto.HexToECDSA("6545ddd10c1e0d6693ba62dec711a2d2973124ae0374d822f845d322fb251645")
	err = tx.SignTx(pkey)
	if err != nil {
		t.Fatalf("SignTx err %v", err)
	} else {
		base64Sig := base64.StdEncoding.EncodeToString(tx.Sig)
		fmt.Printf("Deposit: %s\ntxhash : 0x%x\nmsghash: 0x%x\nb64Sig : %s\n", tx.String(), tx.Hash(), tx.ShortHash(), base64Sig)
	}

	err = tx.ValidateSig()
	if err != nil {
		t.Fatalf("ValidateTx err %v", err)
	}

	txbytes := tx.Bytes()
	ok, tx2 := BytesToTransaction(txbytes)
	if !ok {
		t.Fatalf("BytesToTransaction  err %v", err)
	} else {
		fmt.Printf("txBytes: %s \n", common.ToHex(txbytes))
	}

	if strings.Compare(tx.String(), tx2.String()) != 0 {
		t.Fatalf("RLP serialization mismatch")
	}

}

func TestTransaction(t *testing.T) {
	denomination := uint64(1000000000000000000)
	depositIndex := uint64(1)
	depositor := common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914")
	_, tinfo := TokenInit(depositIndex, denomination, depositor)

	token := NewToken(depositIndex, denomination, depositor)
	token.Owner = depositor
	recipient := common.HexToAddress("0x3088666E05794d2498D9d98326c1b426c9950767")

	tx := NewTransaction(token, tinfo, &recipient)
	tx.Allowance = 100000000000000001
	tx.Spent = 200000000000000002
	tx.PrevBlock = 1
	tx.balance = tx.balance - tx.Allowance - tx.Spent

	var pkey *ecdsa.PrivateKey
	pkey, err := crypto.HexToECDSA("afc522d1476e251535e53abb92bcfa7cfd7ef5c86442ef95e20714f053551574")
	if err != nil {
		t.Fatalf("HexToECDSA err %v", err)
	}

	err = tx.SignTx(pkey)
	if err != nil {
		t.Fatalf("SignTx err %v", err)
	}

	err = tx.ValidateSig()
	if err == nil {
		t.Fatal("ValidateTx err")
	}

	pkey, _ = crypto.HexToECDSA("91dae949703266bcd67ef2dfa9a20d7b62b25520a8375c6a5fad16bd72ae9d4e")
	err = tx.SignTx(pkey)
	if err != nil {
		t.Fatalf("SignTx err %v", err)
	} else {
		fmt.Printf("Transfer: %s\ntxhash : 0x%x\nmsghash: 0x%x\n", tx.String(), tx.Hash(), tx.ShortHash())
	}

	err = tx.ValidateSig()
	if err != nil {
		t.Fatalf("ValidateTx err %v", err)
	}

	txbytes := tx.Bytes()
	ok, tx2 := BytesToTransaction(txbytes)
	if !ok {
		t.Fatalf("BytesToTransaction  err %v", err)
	} else {
		fmt.Printf("txBytes: %s\n", common.ToHex(txbytes))
	}

	if strings.Compare(tx.String(), tx2.String()) != 0 {
		t.Fatalf("RLP serialization mismatch")
	}

	tx_jsonencode, err := json.Marshal(tx2)
	if err != nil {
		t.Fatalf("json Marshall: err %v", err)
	} else {
		fmt.Printf("\njson Marshall: %s\n", tx_jsonencode)
	}

	tx3 := Transaction{}
	err = json.Unmarshal(tx_jsonencode, &tx3)
	if err != nil {
		t.Fatalf("json Unmarshall: err %v", err)
	} else {
		fmt.Printf("json Unmarshall: %s\n\n", tx3.String())
	}

	str := `{"tokenID":11378577392995273713,"denomination":50000000000000000,"depositIndex":12,"prevBlock":3,"prevOwner":"0x069984a8d9ebb6b15eb144e0cb01a81b9c7379b0","recipient":"0xf91a593154c170017fbed4b0348e259827642bd8","allowance":30000000000000000,"spent":10000000000000001,"sig": "21KVTf5c8oL1X9lreV50IVYOW0L+Gm+3bHY6IhUqVy0lMupN0FYgUHa7s5DqdksoqtsMwzr4k9RVYoU5a3xjoAE=" }`
	//tx4 := Transaction{}
	var tx4 Transaction
	json.Unmarshal([]byte(str), &tx4)
	fmt.Printf("Transfer: %s\ntxhash : 0x%x\nmsghash: 0x%x\n", tx4.String(), tx.Hash(), tx.ShortHash())
}
