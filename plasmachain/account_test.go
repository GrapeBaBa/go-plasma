// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.

package plasmachain

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestAccount(t *testing.T) {
	expectedTokenTD := uint64(0xed93d73d0bda35d1)
	denomination := uint64(50000000000000000)
	depositIndex := uint64(0)
	depositor := common.HexToAddress("0xf91a593154c170017fbed4b0348e259827642bd8")
	balance := uint64(16294579238595022365)
	tokenID := tokenKey(depositIndex, denomination, depositor)
	if tokenID != expectedTokenTD {
		t.Fatalf("tokenKey mismatch: expected[%v] found [%v]", expectedTokenTD, tokenID)
	}

	testAcct := NewAccount()

	testAcct.Tokens = []uint64{uint64(0xed93d73d0bda35d1), uint64(0x37b01bd3adfc4ef3), uint64(0x9af84bc1208918b)}
	testAcct.Denomination = new(big.Int).SetUint64(denomination)
	testAcct.Balance = new(big.Int).Mul(new(big.Int).SetUint64(balance), testAcct.Denomination)

	fmt.Printf("testAcct: %v\n", testAcct)

	encoded, _ := rlp.EncodeToBytes(testAcct)
	fmt.Printf("RLP[u]: 0x%x\n", encoded)
	var a Account // []interface{}
	err := rlp.Decode(bytes.NewReader(encoded), &a)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Printf("decode s: %v\n", a.String())

}
