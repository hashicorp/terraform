package tar

import (
	"archive/tar"
	"io"
	"os"
)

// Reader provides sequential access to the contents of a tar archive.
type Reader struct {
	tar.Reader
}

// NewReader creates a new Reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{Reader: *tar.NewReader(r)}
}

// NextFile advances to the next file in the tar archive.
func (r *Reader) NextFile() (name string, err error) {
	hdr, err := r.Next()
	if err != nil {
		return "", err
	}
	return hdr.Name, nil
}

// Writer provides sequential writing of a tar archive in POSIX.1 format.
type Writer struct {
	tar.Writer
	closers []io.Closer
}

// NewWriter creates a new Writer writing to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{Writer: *tar.NewWriter(w)}
}

// NewWriteMultiCloser creates a new Writer writing to w that also closes all
// closers in order on close.
func NewWriteMultiCloser(w io.WriteCloser, closers ...io.Closer) *Writer {
	return &Writer{
		Writer:  *tar.NewWriter(w),
		closers: closers,
	}
}

// NextFile computes and writes a header and prepares to accept the file's
// contents.
func (w *Writer) NextFile(name string, fi os.FileInfo) error {
	if name == "" {
		name = fi.Name()
	}
	hdr, err := tar.FileInfoHeader(fi, name)
	if err != nil {
		return err
	}
	hdr.Name = name
	return w.WriteHeader(hdr)
}

// Close closes the tar archive and all other closers, flushing any unwritten
// data to the underlying writer.
func (w *Writer) Close() error {
	err := w.Writer.Close()
	for _, c := range w.closers {
		if cerr := c.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}
