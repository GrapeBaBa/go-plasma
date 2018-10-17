// Copyright 2018 Wolk Inc.
// This file is part of the Wolk Plasma library.
package plasmachain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/wolkdb/go-plasma/deep"
)

type Account struct {
	Tokens       []uint64 `json:"tokens"        gencodec:"required"`
	Denomination *big.Int `json:"denomination"  gencodec:"required"`
	Balance      *big.Int `json:"balance"       gencodec:"required"`

	data encAccount
}

type encAccount struct {
	Tokens       []uint64 `json:"tokens"`
	Denomination *big.Int `json:"denomination"`
	Balance      *big.Int `json:"balance"`
}

//# go:generate gencodec -type Account -field-override accountMarshaling -out account_json.go
type accountMarshaling struct {
	Tokens       []hexutil.Uint64
	Denomination *hexutil.Big
	Balance      *hexutil.Big
}

func NewAccount() *Account {
	return &Account{
		Tokens:       []uint64{},
		Denomination: new(big.Int),
		Balance:      new(big.Int),
	}
}

//shortAddr = last 8bytes of addr
func accountKey(addr common.Address) uint64 {
	shortAddr := addr[12:20]
	return deep.BytesToUint64(shortAddr)
}

func (a *Account) EncodeRLP(w io.Writer) (err error) {
	a.data.Tokens = a.Tokens
	a.data.Balance = a.Balance
	a.data.Denomination = a.Denomination
	return rlp.Encode(w, a.data)
}

func (a *Account) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode(&a.data); err != nil {
		return err
	}
	a.Tokens = a.data.Tokens
	a.Balance = a.data.Balance
	a.Denomination = a.data.Denomination
	return nil
}

func (a *Account) String() string {
	tokenList := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(a.Tokens)), ","), "[]")
	return fmt.Sprintf("{\"Tokens\":\"%s\", \"Denomination\":\"%x\", \"Balance\":\"%x\"}",
		tokenList, a.Denomination, a.Balance)
}

var _ = (*accountMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (a Account) MarshalJSON() ([]byte, error) {
	type Account struct {
		Tokens       []hexutil.Uint64 `json:"tokens"        gencodec:"required"`
		Denomination *hexutil.Big     `json:"denomination"  gencodec:"required"`
		Balance      *hexutil.Big     `json:"balance"       gencodec:"required"`
	}
	var enc Account
	if a.Tokens != nil {
		enc.Tokens = make([]hexutil.Uint64, len(a.Tokens))
		for k, v := range a.Tokens {
			enc.Tokens[k] = hexutil.Uint64(v)
		}
	}
	enc.Denomination = (*hexutil.Big)(a.Denomination)
	enc.Balance = (*hexutil.Big)(a.Balance)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (a *Account) UnmarshalJSON(input []byte) error {
	type Account struct {
		Tokens       []hexutil.Uint64 `json:"tokens"        gencodec:"required"`
		Denomination *hexutil.Big     `json:"denomination"  gencodec:"required"`
		Balance      *hexutil.Big     `json:"balance"       gencodec:"required"`
	}
	var dec Account
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Tokens == nil {
		return errors.New("missing required field 'tokens' for Account")
	}
	a.Tokens = make([]uint64, len(dec.Tokens))
	for k, v := range dec.Tokens {
		a.Tokens[k] = uint64(v)
	}
	if dec.Denomination == nil {
		return errors.New("missing required field 'denomination' for Account")
	}
	a.Denomination = (*big.Int)(dec.Denomination)
	if dec.Balance == nil {
		return errors.New("missing required field 'balance' for Account")
	}
	a.Balance = (*big.Int)(dec.Balance)
	return nil
}
