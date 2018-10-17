package wrlp

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestW(t *testing.T) {
	var w Block
	w.blockNumber = 322

	for i := 0; i < 3; i++ {
		var tx Transaction
		tx.TokenID = uint64(87650001 + i)
		tx.Denomination = uint64(10000000 * (i + 1))
		tx.DepositIndex = uint64(i)
		w.txs = append(w.txs, &tx)
	}

	//	var txlist Transaction
	//	var wtxlist []WolkTransaction
	for i := 0; i < 5; i++ {
		var wtx WolkTransaction
		wtx.Key = uint64(101 + i)
		h := common.BytesToHash([]byte(fmt.Sprintf("%d", i)))
		wtx.Val = &h
		w.wtxs = append(w.wtxs, &wtx)
	}

	encoded, err := rlp.EncodeToBytes(&w)
	if err != nil {
		t.Fatalf("%v", err)
	}
	var w2 Block
	err = rlp.Decode(bytes.NewReader(encoded), &w2)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Printf("%x => %v\n", encoded, w2)
	for i, tx := range w2.txs {
		fmt.Printf("Transaction %d : %s\n", i, tx.String())
	}
	for i, wtx := range w2.wtxs {
		fmt.Printf("WolkTransaction %d : %s\n", i, wtx.String())
	}

}
