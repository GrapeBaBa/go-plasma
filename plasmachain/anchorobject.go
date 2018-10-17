// Copyright 2018 Wolk Inc.
// This file is part of the Wolk Plasma library.
package plasmachain

import (
	"fmt"
	"io"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/wolkdb/go-plasma/deep"
)

// anchorBlock represents a list of pending anchor tx submitted to a block
type anchorBlock struct {
	//TODO: Anchors should be excluded from the chainobject
	Anchors    []*deep.AnchorTransaction `json:"anchors"`
	Owners     []*common.Address         `json:"owners"`
	StartBlock uint64                    `json:"startBlock"`
	EndBlock   uint64                    `json:"endBlock"`

	//AnchorsHash[] this should be the feedback info
	data encAnchorBlock
}

type encAnchorBlock struct {
	Anchors    []*deep.AnchorTransaction `json:"anchors"`
	Owners     []*common.Address         `json:"owners"`
	StartBlock uint64                    `json:"startBlock"`
	EndBlock   uint64                    `json:"endBlock"`
}

//
func NewAnchorBlock(db *StateDB, anchorTx *deep.AnchorTransaction) *anchorBlock {
	return &anchorBlock{
		Anchors:    []*deep.AnchorTransaction{anchorTx},
		StartBlock: anchorTx.BlockNumber,
		EndBlock:   anchorTx.BlockNumber,
	}
}

func (self *anchorBlock) AddAnchorTx(anchorTx *deep.AnchorTransaction) {
	self.Anchors = append(self.Anchors, anchorTx)
}

func (self *anchorBlock) Sort() {
	//TODO: sort the ancorTx within a block
	sort.Slice(self.Anchors[:], func(i, j int) bool {
		return self.Anchors[i].BlockNumber < self.Anchors[j].BlockNumber
	})
	self.StartBlock = self.Anchors[0].BlockNumber
	self.EndBlock = self.Anchors[len(self.Anchors)-1].BlockNumber
}

// EncodeRLP implements rlp.Encoder.
func (a *anchorBlock) EncodeRLP(w io.Writer) error {
	a.data.Anchors = a.Anchors
	a.data.StartBlock = a.StartBlock
	a.data.EndBlock = a.EndBlock
	a.data.Owners = a.Owners
	return rlp.Encode(w, a.data)
}

// EncodeRLP implements rlp.Encoder.
func (a *anchorBlock) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode(&a.data); err != nil {
		return err
	}
	//FATAL: Bad idea to include anchor here !!
	a.Anchors = a.data.Anchors
	a.StartBlock = a.data.StartBlock
	a.EndBlock = a.data.EndBlock
	a.Owners = a.data.Owners
	return nil
}

func (a *anchorBlock) Bytes() (enc []byte) {
	enc, _ = rlp.EncodeToBytes(&a)
	return enc
}

func (a *anchorBlock) Hex() string {
	return fmt.Sprintf("%x", a.Bytes())
}

func (a *anchorBlock) Hash() (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, []interface{}{
		a.Anchors,
		a.Owners,
		a.StartBlock,
		a.EndBlock,
	})
	hw.Sum(h[:0])
	return h
}

func (a *anchorBlock) String() string {
	s := fmt.Sprintf("Anchors: %v, Owners: %x, StartBlock: %v, EndBlock: %v", a.Anchors, a.Owners, a.StartBlock, a.EndBlock)
	return s
}
