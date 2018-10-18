// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.
package plasmachain

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/wolkdb/go-plasma/deep"
)

// representation for token SMT trie, sturct does not include tokenID
type Token struct {
	Denomination uint64         `json:"denomination" gencodec:"required"`
	PrevBlock    uint64         `json:"prevBlock"  	 gencodec:"required"`
	Owner        common.Address `json:"owner"        gencodec:"required"`
	Balance      uint64         `json:"balance"      gencodec:"required"`
	Allowance    uint64         `json:"allowance"    gencodec:"required"`
	Spent        uint64         `json:"spent"        gencodec:"required"`
	data         encToken
}

type encToken struct {
	Denomination uint64
	PrevBlock    uint64
	Owner        common.Address
	Balance      uint64
	Allowance    uint64
	Spent        uint64
}

type TokenInfo struct {
	DepositIndex uint64         `json:"depositIndex"  gencodec:"required"`
	Denomination uint64         `json:"denomination"  gencodec:"required"`
	Depositor    common.Address `json:"depositor"     gencodec:"required"`
}

//# go:generate gencodec -type Token -field-override tokenMarshaling -out token_json.go
type tokenMarshaling struct {
	Denomination hexutil.Uint64
	PrevBlock    hexutil.Uint64
	Balance      hexutil.Uint64
	Allowance    hexutil.Uint64
	Spent        hexutil.Uint64
}

//# go:generate gencodec -type TokenInfo -field-override tokenInfoMarshaling -out tinfo_json.go
type tokenInfoMarshaling struct {
	DepositIndex hexutil.Uint64
	Denomination hexutil.Uint64
	TokenID      hexutil.Uint64 `json:"tokenID"`
}

func (t *Token) EncodeRLP(w io.Writer) (err error) {
	t.data.Denomination = t.Denomination
	t.data.PrevBlock = t.PrevBlock
	t.data.Owner = t.Owner
	t.data.Balance = t.Balance
	t.data.Allowance = t.Allowance
	t.data.Spent = t.Spent
	return rlp.Encode(w, t.data)
}

func (t *Token) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode(&t.data); err != nil {
		return err
	}
	t.Denomination = t.data.Denomination
	t.PrevBlock = t.data.PrevBlock
	t.Owner = t.data.Owner
	t.Balance = t.data.Balance
	t.Allowance = t.data.Allowance
	t.Spent = t.data.Spent
	return nil
}

//tokenID = last 8bytes of deep.Keccak256(msg.sender, depositIndex, denomination)
func tokenKey(depositIndex uint64, denomination uint64, depositor common.Address) uint64 {
	b0 := depositor.Bytes()
	b1 := deep.UInt64ToByte(depositIndex)
	b2 := deep.UInt64ToByte(denomination)
	b := deep.Keccak256(append(append(b0, b1...), b2...))
	tokenID := b[24:32]
	return deep.BytesToUint64(tokenID)
}

func TokenInit(depositIndex uint64, denomination uint64, depositor common.Address) (tokenID uint64, tinfo *TokenInfo) {
	tokenID = tokenKey(depositIndex, denomination, depositor)
	tinfo = &TokenInfo{
		DepositIndex: depositIndex,
		Denomination: denomination,
		Depositor:    depositor,
	}
	return tokenID, tinfo
}

func (tinfo *TokenInfo) TokenID() uint64 {
	return tokenKey(tinfo.DepositIndex, tinfo.Denomination, tinfo.Depositor)
}

func NewToken(depositIndex uint64, denomination uint64, depositor common.Address) *Token {
	//tokenID := tokenKey(depositIndex, denomination, depositor)
	return &Token{
		Denomination: denomination,
		PrevBlock:    0,
		Owner:        depositor,
		Balance:      denomination,
		Allowance:    0,
		Spent:        0,
	}
}

func (t *Token) String() string {
	return fmt.Sprintf("{\"Denomination\":\"%d\", \"PrevBlock\":\"%d\", \"Owner\":\"%x\", \"Balance\":\"%d\", \"Allowance\":\"%d\", \"Spent\":\"%d\"}",
		t.Denomination, t.PrevBlock, t.Owner, t.Balance, t.Allowance, t.Spent)
}
