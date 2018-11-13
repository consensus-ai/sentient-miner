package sentient

import (
	"hash"

	"golang.org/x/crypto/blake2b"
)

// SenHash is an abstraction to make blake-512 hasher return a 256-bit prefix.
// This is really a pretty lazy solution driven by lack of proper time and resources
// to fully switch the entire codebase to using 512-bit hashes and entropy.
type SenHash struct {
	blake512hasher hash.Hash
}

// Write adds data to the underlying hasher.
// Part of the io.Writer interface
func (h SenHash) Write(p []byte) (n int, err error) {
	return h.blake512hasher.Write(p)
}

// Sum appends the current hash to b and returns the resulting slice.
// It does not change the underlying hash state.
func (h SenHash) Sum(b []byte) []byte {
	b = append(b, h.blake512hasher.Sum(nil)[:32]...)
	return b
}

// Reset resets the Hash to its initial state.
func (h SenHash) Reset() {
	h.blake512hasher.Reset()
}

// Size returns the number of bytes Sum will return.
func (h SenHash) Size() int {
	return 32
}

// BlockSize returns the hash's underlying block size.
// The Write method must be able to accept any amount
// of data, but it may operate more efficiently if all writes
// are a multiple of the block size.
func (h SenHash) BlockSize() int {
	return h.BlockSize()
}

// NewSenHash creates a new instance of SenHash with blake512
// as the underlying hasher.
func NewSenHash() hash.Hash {
	b2b, _ := blake2b.New512(nil)
	return SenHash{
		blake512hasher: b2b,
	}
}
