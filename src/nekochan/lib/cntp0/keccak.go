package cntp0

import (
	"hash"
	"io"
)

// XXX hack, prone to break
type keccakstate interface {
	hash.Hash
	io.Reader
}

type sha3wrap struct {
	// our usage of it doesn't require to accept writes after pulling out digest
	keccakstate
}

func (s sha3wrap) Sum(b []byte) []byte {
	// we always supply buffer with proper capacity, so don't do alloc+append
	w := b[:len(b)+s.Size()]
	s.Read(w[len(b):])
	return w
}

type shakewrap struct {
	// we need to override size
	keccakstate
	size uint32
}

func (s shakewrap) Sum(b []byte) []byte {
	// we always supply buffer with proper capacity, so don't do alloc+append
	w := b[:len(b)+int(s.size)]
	s.Read(w[len(b):])
	return w
}

func (s shakewrap) Size() int {
	return int(s.size)
}
