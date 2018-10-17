// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package eventlog

import (
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var _ = (*challengeMarshaling)(nil)

// MarshalJSON marshals as JSON.
func (c ChallengeEvent) MarshalJSON() ([]byte, error) {
	type ChallengeEvent struct {
		Challenger common.Address `json:"challenger" gencodec:"required"`
		TokenID    hexutil.Uint64 `json:"tokenID"    gencodec:"required"`
		TS         hexutil.Uint64 `json:"timestamp"  gencodec:"required"`
	}
	var enc ChallengeEvent
	enc.Challenger = c.Challenger
	enc.TokenID = hexutil.Uint64(c.TokenID)
	enc.TS = hexutil.Uint64(c.TS)
	return json.Marshal(&enc)
}

// UnmarshalJSON unmarshals from JSON.
func (c *ChallengeEvent) UnmarshalJSON(input []byte) error {
	type ChallengeEvent struct {
		Challenger *common.Address `json:"challenger" gencodec:"required"`
		TokenID    *hexutil.Uint64 `json:"tokenID"    gencodec:"required"`
		TS         *hexutil.Uint64 `json:"timestamp"  gencodec:"required"`
	}
	var dec ChallengeEvent
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if dec.Challenger == nil {
		return errors.New("missing required field 'challenger' for ChallengeEvent")
	}
	c.Challenger = *dec.Challenger
	if dec.TokenID == nil {
		return errors.New("missing required field 'tokenID' for ChallengeEvent")
	}
	c.TokenID = uint64(*dec.TokenID)
	if dec.TS == nil {
		return errors.New("missing required field 'timestamp' for ChallengeEvent")
	}
	c.TS = uint64(*dec.TS)
	return nil
}
