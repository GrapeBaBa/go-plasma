package metamask

import (
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
)

type AppTransaction struct {
	ts   uint64
	data []byte

	// used to RLP encode the data
	encAppData encAppTransaction
}

type encAppTransaction struct {
	TS   uint64
	Data []byte
}

func (tx *AppTransaction) EncodeRLP(w io.Writer) (err error) {
	tx.encAppData.TS = tx.ts
	tx.encAppData.Data = tx.data
	return rlp.Encode(w, tx.encAppData)
}

func (tx *AppTransaction) DecodeRLP(s *rlp.Stream) error {
	if err := s.Decode(&tx.encAppData); err != nil {
		return err
	}
	tx.ts = tx.encAppData.TS
	tx.data = tx.encAppData.Data
	return nil
}

func (tx *AppTransaction) String() string {
	return fmt.Sprintf("{\"ts\":\"%d\", \"data\":\"%x\"}", tx.ts, tx.data)
}
