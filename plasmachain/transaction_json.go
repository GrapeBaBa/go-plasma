// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package plasmachain

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var _ = (*transactionMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (t Transaction) MarshalJSON() ([]byte, error) {
	type Transaction struct {
		TokenID      hexutil.Uint64  `json:"tokenID"      gencodec:"required"`
		Denomination hexutil.Uint64  `json:"denomination" gencodec:"required"`
		DepositIndex hexutil.Uint64  `json:"depositIndex" gencodec:"required"`
		PrevBlock    hexutil.Uint64  `json:"prevBlock"    gencodec:"required"`
		PrevOwner    *common.Address `json:"prevOwner"    gencodec:"required"`
		Recipient    *common.Address `json:"recipient"    gencodec:"required"`
		Allowance    hexutil.Uint64  `json:"allowance"    gencodec:"required"`
		Spent        hexutil.Uint64  `json:"spent"        gencodec:"required"`
		Sig          hexutil.Bytes   `json:"sig"`
	}
	var enc Transaction
	enc.TokenID = hexutil.Uint64(t.TokenID)
	enc.Denomination = hexutil.Uint64(t.Denomination)
	enc.DepositIndex = hexutil.Uint64(t.DepositIndex)
	enc.PrevBlock = hexutil.Uint64(t.PrevBlock)
	enc.PrevOwner = t.PrevOwner
	enc.Recipient = t.Recipient
	enc.Allowance = hexutil.Uint64(t.Allowance)
	enc.Spent = hexutil.Uint64(t.Spent)
	enc.Sig = t.Sig
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *Transaction) UnmarshalJSON(input []byte) error {
	type Transaction struct {
		TokenID      *hexutil.Uint64 `json:"tokenID"      gencodec:"required"`
		Denomination *hexutil.Uint64 `json:"denomination" gencodec:"required"`
		DepositIndex *hexutil.Uint64 `json:"depositIndex" gencodec:"required"`
		PrevBlock    *hexutil.Uint64 `json:"prevBlock"    gencodec:"required"`
		PrevOwner    *common.Address `json:"prevOwner"    gencodec:"required"`
		Recipient    *common.Address `json:"recipient"    gencodec:"required"`
		Allowance    *hexutil.Uint64 `json:"allowance"    gencodec:"required"`
		Spent        *hexutil.Uint64 `json:"spent"        gencodec:"required"`
		Sig          *hexutil.Bytes  `json:"sig"`
	}
	var dec Transaction
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.TokenID == nil {
		return errors.New("missing required field 'tokenID' for Transaction")
	}
	t.TokenID = uint64(*dec.TokenID)
	if dec.Denomination == nil {
		return errors.New("missing required field 'denomination' for Transaction")
	}
	t.Denomination = uint64(*dec.Denomination)
	if dec.DepositIndex == nil {
		return errors.New("missing required field 'depositIndex' for Transaction")
	}
	t.DepositIndex = uint64(*dec.DepositIndex)
	if dec.PrevBlock == nil {
		return errors.New("missing required field 'prevBlock' for Transaction")
	}
	t.PrevBlock = uint64(*dec.PrevBlock)
	if dec.PrevOwner == nil {
		return errors.New("missing required field 'prevOwner' for Transaction")
	}
	t.PrevOwner = dec.PrevOwner
	if dec.Recipient == nil {
		return errors.New("missing required field 'recipient' for Transaction")
	}
	t.Recipient = dec.Recipient
	if dec.Allowance == nil {
		return errors.New("missing required field 'allowance' for Transaction")
	}
	t.Allowance = uint64(*dec.Allowance)
	if dec.Spent == nil {
		return errors.New("missing required field 'spent' for Transaction")
	}
	t.Spent = uint64(*dec.Spent)
	if dec.Sig != nil {
		t.Sig = *dec.Sig
	}
	return nil
}

var _ = (*tokenInfoMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (t TokenInfo) MarshalJSON() ([]byte, error) {
	type TokenInfo struct {
		DepositIndex hexutil.Uint64 `json:"depositIndex"  gencodec:"required"`
		Denomination hexutil.Uint64 `json:"denomination"  gencodec:"required"`
		Depositor    common.Address `json:"depositor"     gencodec:"required"`
		TokenID      hexutil.Uint64 `json:"tokenID"`
	}
	var enc TokenInfo
	enc.DepositIndex = hexutil.Uint64(t.DepositIndex)
	enc.Denomination = hexutil.Uint64(t.Denomination)
	enc.Depositor = t.Depositor
	enc.TokenID = hexutil.Uint64(t.TokenID())
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *TokenInfo) UnmarshalJSON(input []byte) error {
	type TokenInfo struct {
		DepositIndex *hexutil.Uint64 `json:"depositIndex"  gencodec:"required"`
		Denomination *hexutil.Uint64 `json:"denomination"  gencodec:"required"`
		Depositor    *common.Address `json:"depositor"     gencodec:"required"`
	}
	var dec TokenInfo
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.DepositIndex == nil {
		return errors.New("missing required field 'depositIndex' for TokenInfo")
	}
	t.DepositIndex = uint64(*dec.DepositIndex)
	if dec.Denomination == nil {
		return errors.New("missing required field 'denomination' for TokenInfo")
	}
	t.Denomination = uint64(*dec.Denomination)
	if dec.Depositor == nil {
		return errors.New("missing required field 'depositor' for TokenInfo")
	}
	t.Depositor = *dec.Depositor
	return nil
}
