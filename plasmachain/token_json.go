// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package plasmachain

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var _ = (*tokenMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (t Token) MarshalJSON() ([]byte, error) {
	type Token struct {
		Denomination hexutil.Uint64 `json:"denomination" gencodec:"required"`
		PrevBlock    hexutil.Uint64 `json:"prevBlock"  	 gencodec:"required"`
		Owner        common.Address `json:"owner"        gencodec:"required"`
		Balance      hexutil.Uint64 `json:"balance"   gencodec:"required"`
		Allowance    hexutil.Uint64 `json:"allowance" gencodec:"required"`
		Spent        hexutil.Uint64 `json:"spent"     gencodec:"required"`
	}
	var enc Token
	enc.Denomination = hexutil.Uint64(t.Denomination)
	enc.PrevBlock = hexutil.Uint64(t.PrevBlock)
	enc.Owner = t.Owner
	enc.Balance = hexutil.Uint64(t.Balance)
	enc.Allowance = hexutil.Uint64(t.Allowance)
	enc.Spent = hexutil.Uint64(t.Spent)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (t *Token) UnmarshalJSON(input []byte) error {
	type Token struct {
		Denomination *hexutil.Uint64 `json:"denomination" gencodec:"required"`
		PrevBlock    *hexutil.Uint64 `json:"prevBlock"  	 gencodec:"required"`
		Owner        *common.Address `json:"owner"        gencodec:"required"`
		Balance      *hexutil.Uint64 `json:"balance"   gencodec:"required"`
		Allowance    *hexutil.Uint64 `json:"allowance" gencodec:"required"`
		Spent        *hexutil.Uint64 `json:"spent"     gencodec:"required"`
	}
	var dec Token
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Denomination == nil {
		return errors.New("missing required field 'denomination' for Token")
	}
	t.Denomination = uint64(*dec.Denomination)
	if dec.PrevBlock != nil {
		t.PrevBlock = uint64(*dec.PrevBlock)
	}
	if dec.Owner == nil {
		return errors.New("missing required field 'owner' for Token")
	}
	t.Owner = *dec.Owner
	if dec.Balance == nil {
		return errors.New("missing required field 'balance' for Token")
	}
	t.Balance = uint64(*dec.Balance)
	if dec.Allowance == nil {
		return errors.New("missing required field 'allowance' for Token")
	}
	t.Allowance = uint64(*dec.Allowance)
	if dec.Spent == nil {
		return errors.New("missing required field 'spent' for Token")
	}
	t.Spent = uint64(*dec.Spent)
	return nil
}
