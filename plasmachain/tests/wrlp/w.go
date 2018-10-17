package wrlp

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type Block struct {
	blockNumber uint64
	txs         []*Transaction
	wtxs        []*WolkTransaction
	data        bdata
}

type Transaction struct {
	TokenID      uint64
	Denomination uint64
	DepositIndex uint64
}

type WolkTransaction struct {
	Key uint64
	Val *common.Hash
}

type bdata struct {
	BlockNumber uint64
	Txs         []*Transaction
	Wtxs        []*WolkTransaction
}

func (b *Block) EncodeRLP(w io.Writer) (err error) {
	b.data.BlockNumber = b.blockNumber
	b.data.Txs = b.txs
	b.data.Wtxs = b.wtxs
	return rlp.Encode(w, b.data)
}

func (b *Block) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode(&b.data); err != nil {
		return err
	}
	b.blockNumber = b.data.BlockNumber
	b.txs = b.data.Txs
	b.wtxs = b.data.Wtxs
	return nil
}

func (tx *Transaction) String() string {
	return fmt.Sprintf("{\"TokenID\":\"%x\", \"Denomination\":\"%v\", \"DepositIndex\":\"%v\"}", tx.TokenID, tx.Denomination, tx.DepositIndex)
}

func (tx *WolkTransaction) String() string {
	return fmt.Sprintf("{\"Key\":\"%x\", \"Val\":\"%x\"}", tx.Key, tx.Val)
}
