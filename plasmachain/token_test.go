// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.
package plasmachain

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestTokenInfo(t *testing.T) {
	tokenID, tokenInfo := TokenInit(1, 1000000000000000000, common.HexToAddress("0x82Da88C31E874C678D529ad51E43De3A4BAF3914"))
	encoded, err := rlp.EncodeToBytes(tokenInfo)
	if err != nil || len(encoded) == 0 {
		fmt.Printf("Error: %v\n", err)
	}
	var tinfo *TokenInfo
	err = rlp.Decode(bytes.NewReader(encoded), &tinfo)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	if tokenID != tokenKey(tinfo.DepositIndex, tinfo.Denomination, tinfo.Depositor) {
		t.Fatalf("TokenInfo Hash err %x\n", tokenID)
	}
}

func TestToken(t *testing.T) {
	expectedTokenTD := uint64(0xed93d73d0bda35d1)
	denomination := uint64(50000000000000000)
	depositIndex := uint64(0)
	depositor := common.HexToAddress("0xf91a593154c170017fbed4b0348e259827642bd8")
	tokenID := tokenKey(depositIndex, denomination, depositor)
	if tokenID != expectedTokenTD {
		t.Fatalf("tokenKey mismatch: expected[%v] found [%v]", expectedTokenTD, tokenID)
	}

	u := NewToken(depositIndex, denomination, depositor)
	out := `{"Denomination":"50000000000000000", "PrevBlock":"42", "Owner":"aaaa593154c170017fbed4b0348e259827642bbb", "Balance":"10000000000000000", "Allowance":"25000000000000000", "Spent":"15000000000000000"}`
	u.PrevBlock = 42
	u.Allowance = 25000000000000000
	u.Spent = 15000000000000000
	u.Balance = 10000000000000000
	u.Owner = common.HexToAddress("0xaaaa593154c170017fbed4b0348e259827642bbb")
	fmt.Printf("Token: %v\n", u)
	if strings.Compare(out, u.String()) != 0 {
		t.Fatalf("token.String() mismatch expected[%s] found [%s]", out, u.String())
	}
	encoded, _ := rlp.EncodeToBytes(u)
	fmt.Printf("RLP[u]: %x\n", encoded)
	var s Token // []interface{}
	err := rlp.Decode(bytes.NewReader(encoded), &s)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Printf("decode s: %v\n", s.String())

	if u.Denomination != s.Denomination {
		t.Fatalf("Denomination failure %d != %d", u.Denomination, s.Denomination)
	}

	if u.PrevBlock != s.PrevBlock {
		t.Fatalf("PrevBlock failure %d != %d", u.PrevBlock, s.PrevBlock)
	}

	if bytes.Compare(u.Owner.Bytes(), s.Owner.Bytes()) != 0 {
		t.Fatalf("Owner failure %x != %x", u.Owner.Bytes(), s.Owner.Bytes())
	}
	if strings.Compare(u.String(), s.String()) != 0 {
		fmt.Printf("u: %s\n", u.String())
		fmt.Printf("s: %s\n", s.String())
		t.Fatalf("Mismatched string serialization")
	}
}
