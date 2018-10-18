// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk go-plasma library.

package plasmachain

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestBloom(t *testing.T) {
	rand.Seed(42)
	nkeys := 2
	l := make([]uint64, nkeys)
	for i := 0; i < nkeys; i++ {
		l[i] = rand.Uint64()
	}

	bl := NewBloom(l)
	//	func CheckBloom(b []byte, k uint64) (isMember bool) {
	for i, v := range l {
		r := CheckBloom(bl, v)
		fmt.Printf("l[%d]: %16x => %v\n", i, v, r)
		if r == false {
			t.Fatalf("failed Bloom")
		}
	}

	for i, v := range l {
		r := CheckBloom(bl, v-33)
		fmt.Printf("l[%d]-1: %16x => %v\n", i, v-1, r)
		if r {
			t.Fatalf("failed Bloom")
		}
	}
}
