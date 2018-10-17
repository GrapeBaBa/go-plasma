// Copyright 2018 Wolk Inc.  All rights reserved.
// This file is part of the Wolk Deep Blockchains library.
package plasmachain

import (
	"math/bits"

	"github.com/wolkdb/go-plasma/smt"
)

/*
Each Plasma block has a bloom filter stored in a single SWARM chunk for all tokenIDs spent in the block.

This Bloom filter uses 8 15-bit numbers contained within a 64bit key (60 bits out of 64)
to set 8 bits in a 4K chunk (32,768 bits).  Checking for all 15 bits being set for a given
key enables fast set membership testing, but with false positives like this:

 1K keys  1 in 174K   https://hur.st/bloomfilter/?n=1024&p=&m=4KiB&k=8
 2K keys  1 in 1.7K   https://hur.st/bloomfilter/?n=2048&p=&m=4KiB&k=8

As the number of transactions per block goes from 1K to 2K, the false positive rate skyrockets.
To address this, we simply have 1 chunk per 1K keys.  100K keys, 100 chunks.

False positives cause additional proof bits sent between peers,
so this must be further optimized with optimal k and chunk size selection.

*/

const chunkSize = 4096

func NewBloom(l []uint64) []byte {
	b := make([]byte, chunkSize)
	for _, k := range l {
		key1 := smt.UIntToByte(k)
		key2 := smt.UIntToByte(bits.ReverseBytes64(k))
		for i := 0; i < 4; i++ {
			n1 := (uint(key1[i*2]) | uint(key1[i*2+1])<<8) & 0x7FFF // 15 bit number
			n2 := (uint(key2[i*2]) | uint(key2[i*2+1])<<8) & 0x7FFF // 15 bit number
			b[n1/8] |= uint8(1 << uint(n1%8))
			b[n2/8] |= uint8(1 << uint(n2%8))
		}
	}
	return b
}

func CheckBloom(b []byte, k uint64) (isMember bool) {
	if len(b) != chunkSize {
		return true
	}
	key1 := smt.UIntToByte(k)
	key2 := smt.UIntToByte(bits.ReverseBytes64(k))
	for i := 0; i < 4; i++ {
		n1 := (uint(key1[i*2]) | uint(key1[i*2+1])<<8) & 0x7FFF // 15 bit number
		n2 := (uint(key2[i*2]) | uint(key2[i*2+1])<<8) & 0x7FFF // 15 bit number
		if (b[n1/8] & (1 << uint(n1%8))) == 0 {
			return false
		} else if (b[n2/8] & (1 << uint(n2%8))) == 0 {
			return false
		}
	}
	return true
}
