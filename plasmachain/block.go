// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.

package plasmachain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/wolkdb/go-plasma/deep"
	"github.com/wolkdb/go-plasma/smt"
)

// NewGeneratedPlasmaBlockEvent is posted when a block has been imported.
type NewGeneratedPlasmaBlockEvent struct{ Block *Block }

type Block struct {
	header *Header `json:"header"`
	body   *Body   `json:"body"`
	// used for encoding
	data encBlock
}

type encBlock struct {
	Header *Header
	Body   *Body
}

type Body struct {
	Layer2Transactions []*Transaction            `json:"layer2Transactions"`
	AnchorTransactions []*deep.AnchorTransaction `json:"AnchorTransactions"`

	data encBody
}

type encBody struct {
	Layer2Transactions []*Transaction            `json:"layer2Transactions"`
	AnchorTransactions []*deep.AnchorTransaction `json:"AnchorTransactions"`
}

type Header struct {
	ParentHash      common.Hash `json:"parentHash"     gencodec:"required"`
	BlockNumber     uint64      `json:"blockNumber"    gencodec:"required`
	Time            uint64      `json:"time"           gencodec:"required`
	BloomID         common.Hash `json:"bloomID"         gencodec:"required`
	TransactionRoot common.Hash `json:"transactionRoot" gencodec:"required`
	TokenRoot       common.Hash `json:"tokenRoot"       gencodec:"required`
	AccountRoot     common.Hash `json:"accountRoot"     gencodec:"required`
	L3ChainRoot     common.Hash `json:"l3ChainRoot"     gencodec:"required`
	AnchorRoot      common.Hash `json:"anchorRoot"      gencodec:"required`
	Sig             []byte      `json:"sig"             gencodec:"required`
	data            encHeader
}

type encHeader struct {
	ParentHash      common.Hash `json:"parentHash"`
	BlockNumber     uint64      `json:"blockNumber"`
	Time            uint64      `json:"time"`
	BloomID         common.Hash `json:"bloomID"`         // incremental plasma transactions -- build at each block
	TransactionRoot common.Hash `json:"transactionRoot"` // incremental plasma transactions -- used in L1
	TokenRoot       common.Hash `json:"tokenRoot"`       // all token storage
	AccountRoot     common.Hash `json:"accountRoot"`     // all account storage
	L3ChainRoot     common.Hash `json:"l3ChainRoot"`     // l3 blockcahins storage
	AnchorRoot      common.Hash `json:"anchorRoot"`      // incremental Anchor transactions
	Sig             []byte      `json:"sig"`
}

//# go:generate gencodec -type Header -field-override headetMarshaling -out header_json.go
type headetMarshaling struct {
	BlockNumber hexutil.Uint64
	Time        hexutil.Uint64
	Sig         hexutil.Bytes
	HeaderHash  common.Hash `json:"headerHash"`
}

func (b *Block) EncodeRLP(w io.Writer) (err error) {
	b.data.Header = b.header
	b.data.Body = b.body
	return rlp.Encode(w, b.data)
}

func (b *Block) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode(&b.data); err != nil {
		return err
	}
	b.header = b.data.Header
	b.body = b.data.Body
	return nil
}

func (bo *Body) EncodeRLP(w io.Writer) (err error) {
	bo.data.Layer2Transactions = bo.Layer2Transactions
	bo.data.AnchorTransactions = bo.AnchorTransactions
	return rlp.Encode(w, bo.data)
}

func (bo *Body) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode(&bo.data); err != nil {
		return err
	}
	bo.Layer2Transactions = bo.data.Layer2Transactions
	bo.AnchorTransactions = bo.data.AnchorTransactions
	return nil
}

// EncodeRLP writes x as RLP list [a, b] that omits the Name field.
func (hdr *Header) EncodeRLP(w io.Writer) (err error) {
	hdr.data.ParentHash = hdr.ParentHash
	hdr.data.BlockNumber = hdr.BlockNumber
	hdr.data.Time = hdr.Time
	hdr.data.BloomID = hdr.BloomID
	hdr.data.TransactionRoot = hdr.TransactionRoot
	hdr.data.TokenRoot = hdr.TokenRoot
	hdr.data.AccountRoot = hdr.AccountRoot
	hdr.data.L3ChainRoot = hdr.L3ChainRoot
	hdr.data.AnchorRoot = hdr.AnchorRoot
	hdr.data.Sig = hdr.Sig
	return rlp.Encode(w, hdr.data)
}

func (hdr *Header) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode(&hdr.data); err != nil {
		return err
	}
	hdr.ParentHash = hdr.data.ParentHash
	hdr.BlockNumber = hdr.data.BlockNumber
	hdr.Time = hdr.data.Time
	hdr.BloomID = hdr.data.BloomID
	hdr.TransactionRoot = hdr.data.TransactionRoot
	hdr.TokenRoot = hdr.data.TokenRoot
	hdr.AccountRoot = hdr.data.AccountRoot
	hdr.L3ChainRoot = hdr.data.L3ChainRoot
	hdr.AnchorRoot = hdr.data.AnchorRoot
	hdr.Sig = hdr.data.Sig
	return nil
}

func CopyBlock(block *Block) *Block {
	header := CopyHeader(block.Header().(*Header))
	body := CopyBody(block.Body().(*Body))
	return &Block{
		header: header,
		body:   body,
	}
}

func CopyHeader(hdr *Header) *Header {
	return &Header{
		ParentHash:      hdr.ParentHash,
		BlockNumber:     hdr.BlockNumber,
		Time:            hdr.Time,
		BloomID:         hdr.BloomID,
		TransactionRoot: hdr.TransactionRoot,
		TokenRoot:       hdr.TokenRoot,
		AccountRoot:     hdr.AccountRoot,
		L3ChainRoot:     hdr.L3ChainRoot,
		AnchorRoot:      hdr.AnchorRoot,
		Sig:             hdr.Sig,
	}
}

func CopyBody(body *Body) *Body {
	var txs []*Transaction
	for _, tx := range body.Layer2Transactions {
		txs = append(txs, tx)
	}
	var atxs []*deep.AnchorTransaction
	for _, atx := range body.AnchorTransactions {
		atxs = append(atxs, atx)
	}
	return &Body{Layer2Transactions: txs, AnchorTransactions: atxs}
}

func NewBlock() *Block {
	var b Block
	b.body = new(Body)
	b.header = new(Header)
	return &b
}

func (block Block) Hash() (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, []interface{}{
		block.header,
		block.body.Layer2Transactions,
		block.body.AnchorTransactions,
	})
	hw.Sum(h[:0])
	return h
}

// WARNING: state/ownership is not checked by signTX
func (block *Block) SignBlock(priv *ecdsa.PrivateKey) (err error) {
	return block.header.SignHeader(priv)
}

func (block *Block) GetSigner() (signer common.Address, err error) {
	return block.header.GetSigner()
}

func (block *Block) ValidateBlock() (validated bool, err error) {
	signer, err := block.GetSigner()
	if err != nil {
		log.Info("ValidateBlock ERR", "signer", signer)
		return false, err
	}
	//TODO: block requires minter addr to validate
	// if bytes.Compare(common.HexToAddress(operatorAddress).Bytes(), signer.Bytes()) != 0 {
	// 	return false, fmt.Errorf("Invalid block signer %x %x", signer.Bytes(), common.HexToAddress(operatorAddress).Bytes())
	// }

	return true, nil
}

func (block Block) ValidateBody() (valid bool, err error) {
	return true, nil
}

func (block Block) Header() deep.Header {
	return block.header
}

func (block Block) Body() deep.Body {
	return block.body
}

func (block Block) Root() (p common.Hash) {
	return block.header.Hash()
}

func (block Block) ParentHash() (p common.Hash) {
	return block.header.ParentHash
}

func (block Block) Number() (n uint64) {
	return block.header.BlockNumber
}

func (block Block) NumberU64() (n uint64) {
	return block.header.BlockNumber
}

func (block Block) Time() (tm *big.Int) {
	i := new(big.Int).SetUint64(block.header.Time)
	return i
}

func (block Block) Transactions() (txs []deep.Transaction) {
	for _, tx := range block.body.Layer2Transactions {
		txs = append(txs, tx)
	}
	return txs
}

func (block Block) Encode() ([]byte, error) {
	return rlp.EncodeToBytes(&block)
}

func (block *Block) BlockToChunk() (out []byte) {
	out, _ = rlp.EncodeToBytes(block)
	return out
}

func (block *Block) OutputBlock(fullTx bool) (out map[string]interface{}) {
	out = make(map[string]interface{})
	out["parentHash"] = block.header.ParentHash.Hex()
	out["blockNumber"] = fmt.Sprintf("%x", smt.UIntToByte(block.header.BlockNumber))
	out["bloomID"] = block.header.BloomID.Hex()
	out["transactionRoot"] = block.header.TransactionRoot.Hex()
	out["tokenRoot"] = block.header.TokenRoot.Hex()
	out["accountRoot"] = block.header.AccountRoot.Hex()
	out["l3ChainRoot"] = block.header.L3ChainRoot.Hex()
	out["anchorRoot"] = block.header.AnchorRoot.Hex()
	out["sig"] = fmt.Sprintf("%x", block.header.Sig)
	if fullTx {
		out["layer2Transactions"] = fmt.Sprintf("%v", block.body.Layer2Transactions)
		out["AnchorTransactions"] = fmt.Sprintf("%v", block.body.AnchorTransactions)
	}
	return out
}
func (block *Block) String() string {
	sh, err := json.Marshal(block.header)
	if err != nil {
		return "error_marshalling_block"
	}
	sb, err := json.Marshal(block.body)
	if err != nil {
		return "error_marshalling_block"
	}
	return fmt.Sprintf("{\"header\":\"%s\", \"body\":\"%s\"}", string(sh), string(sb))
}

func FromChunk(in []byte) (b *Block) {
	var ob Block // []interface{}
	ob.header = new(Header)
	ob.body = new(Body)
	err := rlp.Decode(bytes.NewReader(in), &ob)
	if err != nil {
		return nil
	}
	return &ob
}

//TODO: why is sig not part of headerhash
func (hdr *Header) Hash() (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, []interface{}{
		hdr.ParentHash,
		hdr.BlockNumber,
		hdr.Time,
		hdr.BloomID,
		hdr.TransactionRoot,
		hdr.TokenRoot,
		hdr.AccountRoot,
		hdr.L3ChainRoot,
		hdr.AnchorRoot,
	})
	hw.Sum(h[:0])
	return h
}

func (hdr *Header) HeaderHash() (h common.Hash) {
	return hdr.Hash()
}

func (hdr *Header) UnsignedHash() common.Hash {
	unsignedhdr := &Header{
		ParentHash:      hdr.ParentHash,
		BlockNumber:     hdr.BlockNumber,
		Time:            hdr.Time,
		BloomID:         hdr.BloomID,
		TransactionRoot: hdr.TransactionRoot,
		TokenRoot:       hdr.TokenRoot,
		AccountRoot:     hdr.AccountRoot,
		L3ChainRoot:     hdr.L3ChainRoot,
		AnchorRoot:      hdr.AnchorRoot,
	}
	return unsignedhdr.Hash()
}

func (hdr *Header) GetSigner() (signer common.Address, err error) {
	recoveredPub, err := crypto.Ecrecover(hdr.UnsignedHash().Bytes(), hdr.Sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(recoveredPub) == 0 || recoveredPub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(recoveredPub[1:])[12:])
	return addr, nil
}

// WARNING: state/ownership is not checked by signTX
func (hdr *Header) SignHeader(priv *ecdsa.PrivateKey) (err error) {
	hash := hdr.UnsignedHash()
	sig, err := crypto.Sign(hash.Bytes(), priv)
	if err != nil {
		return err
	}
	hdr.Sig = make([]byte, 65)
	copy(hdr.Sig, sig)
	return nil
}

func (header *Header) Number() (n uint64) {
	return header.BlockNumber
}

func (header *Header) NumberU64() (n uint64) {
	return header.BlockNumber
}

func FromHeader(in []byte) (b *Header) {
	var h Header // []interface{}
	err := rlp.Decode(bytes.NewReader(in), &h)
	if err != nil {
		return nil
	}
	return &h
}
