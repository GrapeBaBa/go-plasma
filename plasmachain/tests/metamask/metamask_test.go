package metamask

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	expected_signer = common.HexToAddress("0xeDafCC405196A51AE462442E870d0Ea9d5D395A1")
)

func deep.Keccak256(data ...[]byte) []byte {
	hasher := sha3.NewKeccak256()
	for _, b := range data {
		hasher.Write(b)
	}
	return hasher.Sum(nil)
}

func TestMetamask(t *testing.T) {
	// generate encoded data and signature
	var encoded []byte
	var msg []byte
	var sig []byte
	if true {
		// Test case: App Transaction [TS + Data]
		var tx AppTransaction
		// TODO: Bruce to provide test case for RLP hash of 2 items (int + bytes)
		tx.ts = uint64(1529000648)
		tx.data = []byte(common.FromHex("abcdef123456789")) // this is SQL that is encrypted with Symmetric Key
		fmt.Printf("AppTx: %s\n", tx.String())

		// encode AppTransaction structure to Bytes
		var err error
		encoded, err = rlp.EncodeToBytes(&tx)
		if err != nil {
			t.Fatalf("%v", err)
		}
		fmt.Printf("Encoded bytes: %x\n", encoded)
		/*
			[sourabh@www6001 ~]$ rlp decode 0xce845b22b2c8880abcdef123456789
			[ '5b22b2c8', '0abcdef123456789' ]
		*/

		// decode the above encoded data to AppTransaction
		var decodedTx AppTransaction
		err = rlp.Decode(bytes.NewReader(encoded), &decodedTx)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		if decodedTx.ts != tx.ts {
			fmt.Printf("Error: ts mismatch %d != %d", decodedTx.ts, tx.ts)
		}
		if bytes.Compare(decodedTx.data, tx.data) != 0 {
			fmt.Printf("Error: data mismatch %x != %x", decodedTx.data, tx.data)
		}
		fmt.Printf("Decoded Tx: %s\n", decodedTx.String())
		// TODO: Bruce to provide test signature for above test case instead of below temporary
		sig = common.FromHex("4f70d2a2ee5c9797618a03a495847135fbb2d580134d1ebb008f8dd12262125155326d4891d878ab8246dd39869239e9932bc4c194a0cc1e87d151305e7fd7ce1c")
	} else {
		// Simple string
		encoded = []byte("asdf")
		sig = common.FromHex("87eb152bea5d3ca68d5127a988ac3a0ba053e418b60679c6d84a6ff96746ebe91ef691e3dfc7a903b128f4f365a5debac9282943d516f96fb84219029b462c9e1b")
	}

	// Ethereum's famous signing mechanism
	msg = append([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(encoded))), encoded...)
	msgHash := deep.Keccak256(msg)
	fmt.Printf("msgHash: %x\nsig: %x\n", msgHash, sig)

	// recover Ethereum address from signature
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	recoveredPub, err := crypto.Ecrecover(msgHash, sig)
	if err != nil {
		t.Fatalf("Ecrecover: %v\n", err)
	}
	if len(recoveredPub) == 0 || recoveredPub[0] != 4 {
		t.Fatalf("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(recoveredPub[1:])[12:])
	if bytes.Compare(addr.Bytes(), expected_signer.Bytes()) != 0 {
		t.Fatalf("Invalid signer - recovered: %x but expected %x", addr.Bytes(), expected_signer.Bytes())
	}
	fmt.Printf("Addr: %x is correct!\n", addr.Bytes())
}
