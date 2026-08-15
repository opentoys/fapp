package service

import (
	"crypto/sha256"
	"hash"
	"io"
)

// hashReader computes sha256 and counts bytes while reading.
type hashReader struct {
	r io.Reader
	h hash.Hash
	n int64
}

func (hr *hashReader) Read(p []byte) (int, error) {
	n, err := hr.r.Read(p)
	if n > 0 {
		hr.h.Write(p[:n])
		hr.n += int64(n)
	}
	return n, err
}

func newHashReader(r io.Reader) *hashReader {
	return &hashReader{r: r, h: sha256.New()}
}