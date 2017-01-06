// Package gzip implements reading and writing of gzip format compressed files.
// See the compress/gzip package for more details.
package gzip

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

// Reader is an io.Reader that can be read to retrieve uncompressed data from a
// gzip-format compressed file.
type Reader struct {
	gzip.Reader
	name  string
	isEOF bool
}

// NewReader creates a new Reader reading the given reader.
func NewReader(r io.Reader, name string) (*Reader, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &Reader{
		Reader: *gr,
		name:   name,
	}, nil
}

// NextFile returns the file name. Calls subsequent to the first call will
// return EOF.
func (r *Reader) NextFile() (name string, err error) {
	if r.isEOF {
		return "", io.EOF
	}

	r.isEOF = true
	return r.name, nil
}

// Writer is an io.WriteCloser. Writes to a Writer are compressed and written to w.
type Writer struct {
	gzip.Writer
	name        string
	noMoreFiles bool
}

// NextFile never returns a next file, and should not be called more than once.
func (w *Writer) NextFile(name string, _ os.FileInfo) error {
	if w.noMoreFiles {
		return fmt.Errorf("gzip: only accepts one file: already received %q and now %q", w.name, name)
	}
	w.noMoreFiles = true
	w.name = name
	return nil
}

// NewWriter returns a new Writer. Writes to the returned writer are compressed
// and written to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{Writer: *gzip.NewWriter(w)}
}
