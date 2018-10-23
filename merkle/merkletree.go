package merkletree

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
)

type Proof struct {
	Index uint
	Root  []byte
	Proof []byte
}

func is_a_power_of_2(x int) bool {
	if x == 1 {
		return true
	}
	if x%2 == 1 {
		return false
	}
	return is_a_power_of_2(x / 2)
}

// wrapper for hash func
func Computehash(data ...[]byte) []byte {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

func Merkelize(L [][]byte) [][]byte {
	for is_a_power_of_2(len(L)) == false {
		L = append(L, []byte(""))
	}
	LH := make([][]byte, len(L))
	for i, v := range L {
		LH[i] = v
	}
	nodes := make([][]byte, len(L))
	nodes = append(nodes, LH...)
	for i := len(L) - 1; i >= 0; i-- {
		nodes[i] = Computehash(append(nodes[i*2], nodes[i*2+1]...))
	}
	return nodes
}

func MerkleRoot(tree [][]byte) (merkelroot []byte) {
	return tree[1]
}

func GenProof(tree [][]byte, ind uint) (merkelroot []byte, mkproof []byte, index uint, err error) {
	index = ind
	treelen := uint(len(tree) / 2)
	if ind > treelen {
		return merkelroot, mkproof, index, fmt.Errorf("Invalid idx")
	}
	ind += treelen
	mkproof = append(mkproof, tree[ind]...)
	for ind > 1 {
		mkproof = append(mkproof, tree[ind^1]...)
		ind = ind / 2
	}
	return tree[1], mkproof, index, nil
}

func CheckProof(expectedMerkleRoot []byte, mkproof []byte, index uint) (isValid bool, merkleroot []byte, err error) {
	if len(mkproof)%32 != 0 {
		return false, merkleroot, errors.New("Invalid mkproof length")
	}
	merkleroot = append(merkleroot, mkproof[0:32]...)
	merklepath := merkleroot
	for depth := 1; depth < len(mkproof)/32; depth++ {
		rhash := make([]byte, 32)
		copy(rhash, mkproof[depth*32:(depth+1)*32])
		if index%2 > 0 {
			merkleroot = Computehash(append(rhash, merkleroot...))
		} else {
			merkleroot = Computehash(append(merkleroot, rhash...))
		}
		index = index / 2
		merklepath = append(merklepath, merkleroot...)
	}
	if bytes.Compare(expectedMerkleRoot, merkleroot) != 0 {
		return false, merkleroot, nil
	} else {
		return true, merkleroot, nil
	}
}

func ToProof(roothash []byte, mkProof []byte, ind uint) (p *Proof) {
	var externalProof []byte
	externalProof = append(externalProof, mkProof[32:]...)
	p = &Proof{Index: ind, Root: roothash, Proof: externalProof}
	return p
}

func (self *Proof) String() string {
	out := fmt.Sprintf("{\"index\":\"%d\",\"root\":\"%x\",\"proof\":[", self.Index, self.Root)
	for prev := 0; prev < len(self.Proof); prev += 32 {
		if prev > 0 {
			out = out + ","
		}
		out = out + common.Bytes2Hex(self.Proof[prev:prev+32])
	}
	out = out + "]}"
	return out
}
