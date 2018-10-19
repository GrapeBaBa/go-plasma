package merkletree

import (
	"fmt"
	"testing"
)

func TestMerkleTree(t *testing.T) {
	nitems := int64(236)
	o := make([][]byte, nitems)
	for x := int64(0); x < nitems; x++ {
		o[x] = Computehash([]byte(fmt.Sprintf("Val%d", x)))
		fmt.Printf("[%d] v(%s) keccak %x\n", x, []byte(fmt.Sprintf("Val%d", x)), o[x])

	}

	index := uint(37)
	fmt.Printf("value: %x\n", o[index])

	// build merkle tree
	mtree := Merkelize(o)
	root := mtree[1]
	fmt.Printf("root: %x\n", root)

	roothash, mkproof, ind, err := GenProof(mtree, index)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	} else {
		fmt.Printf("[GenProof] root %x proof %x, ind %d\n", roothash, mkproof, ind)
	}

	isValid, merkleroot, err := CheckProof(roothash, mkproof, ind)
	if err != nil {
		t.Fatalf("err: %v\n", err)
	} else {
		fmt.Printf("[CheckProof] isValid: %v merkleroot: %x\n", isValid, merkleroot)
	}
}
