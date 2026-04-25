package utils

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
	b.bits[n/64] |= 1 << (n % 64)
}

func (b *BitSet64) Clear(n int) {
	b.bits[n/64] &^= 1 << (n % 64)
}

func (b *BitSet64) Get(n int) bool {
	return b.bits[n/64]&(1<<(n%64)) != 0
}

func (b *BitSet64) Toggle(n int) {
	b.bits[n/64] ^= 1 << (n % 64)
}

// 查找最小的false
func (b *BitSet64) FindFirstFalseFest() int {
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
