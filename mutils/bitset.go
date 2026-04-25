package mutils

import (
	"math/bits"
)

type BitSet64 struct {
	bits []uint64
	size int
}

func NewBitSet64(size int) *BitSet64 {
	wordSize := (size + 63) / 64
	return &BitSet64{
		bits: make([]uint64, wordSize),
		size: size,
	}
}

func (b *BitSet64) Set(n int) {
	if n < 0 || n >= b.size {
		panic("bitset: index out of range")
	}
	b.bits[n/64] |= 1 << (n % 64)
}

func (b *BitSet64) Clear(n int) {
	if n < 0 || n >= b.size {
		panic("bitset: index out of range")
	}
	b.bits[n/64] &^= 1 << (n % 64)
}

func (b *BitSet64) Get(n int) bool {
	if n < 0 || n >= b.size {
		panic("bitset: index out of range")
	}
	return b.bits[n/64]&(1<<(n%64)) != 0
}

func (b *BitSet64) Toggle(n int) {
	if n < 0 || n >= b.size {
		panic("bitset: index out of range")
	}
	b.bits[n/64] ^= 1 << (n % 64)
}

// 查找最小的false
func (b *BitSet64) FindFirstFalseFast() int {
	for i := 0; i < len(b.bits); i++ {
		inverted := ^b.bits[i]
		if inverted != 0 {
			firstOne := bits.TrailingZeros64(inverted)
			result := i*64 + firstOne
			if result < b.size {
				return result
			}
		}
	}
	return -1
}
