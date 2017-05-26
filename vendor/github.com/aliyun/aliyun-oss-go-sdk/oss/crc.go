package oss

import (
	"hash"
	"hash/crc64"
)

// digest represents the partial evaluation of a checksum.
type digest struct {
	crc uint64
	tab *crc64.Table
}

// NewCRC creates a new hash.Hash64 computing the CRC-64 checksum
// using the polynomial represented by the Table.
func NewCRC(tab *crc64.Table, init uint64) hash.Hash64 { return &digest{init, tab} }

// Size returns the number of bytes Sum will return.
func (d *digest) Size() int { return crc64.Size }

// BlockSize returns the hash's underlying block size.
// The Write method must be able to accept any amount
// of data, but it may operate more efficiently if all writes
// are a multiple of the block size.
func (d *digest) BlockSize() int { return 1 }

// Reset resets the Hash to its initial state.
func (d *digest) Reset() { d.crc = 0 }

// Write (via the embedded io.Writer interface) adds more data to the running hash.
// It never returns an error.
func (d *digest) Write(p []byte) (n int, err error) {
	d.crc = crc64.Update(d.crc, d.tab, p)
	return len(p), nil
}

// Sum64 returns crc64 value.
func (d *digest) Sum64() uint64 { return d.crc }

// Sum returns hash value.
func (d *digest) Sum(in []byte) []byte {
	s := d.Sum64()
	return append(in, byte(s>>56), byte(s>>48), byte(s>>40), byte(s>>32), byte(s>>24), byte(s>>16), byte(s>>8), byte(s))
}
