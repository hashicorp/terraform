package zip

import (
	"archive/zip"
	"io"
	"os"
)

// Reader provides sequential access to the contents of a zip archive.
type Reader struct {
	zip.Reader
	unread []*zip.File
	rc     io.ReadCloser
}

// NewReader returns a new Reader reading from r, which is assumed to have the
// given size in bytes.
func NewReader(r io.ReaderAt, size int64) (*Reader, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, err
	}
	return &Reader{Reader: *zr}, nil
}

// NextFile advances to the next file in the zip archive.
func (r *Reader) NextFile() (name string, err error) {
	// Initialize unread
	if r.unread == nil {
		r.unread = r.Files()[:]
	}

	// Close previous file
	if r.rc != nil {
		r.rc.Close() // Read-only
	}

	if len(r.unread) == 0 {
		return "", io.EOF
	}

	// Open and return next unread
	f := r.unread[0]
	name, r.unread = f.Name, r.unread[1:]
	r.rc, err = f.Open()
	if err != nil {
		return "", err
	}
	return name, nil
}

func (r *Reader) Read(p []byte) (n int, err error) {
	return r.rc.Read(p)
}

// Files returns the full list of files in the zip archive.
func (r *Reader) Files() []*zip.File {
	return r.File
}

// Writer provides sequential writing of a zip archive.1 format.
type Writer struct {
	zip.Writer
	w io.Writer
}

// NewWriter returns a new Writer writing to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{Writer: *zip.NewWriter(w)}
}

// NextFile computes and writes a header and prepares to accept the file's
// contents.
func (w *Writer) NextFile(name string, fi os.FileInfo) error {
	if name == "" {
		name = fi.Name()
	}
	hdr, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	hdr.Name = name
	w.w, err = w.CreateHeader(hdr)
	return err
}

func (w *Writer) Write(p []byte) (n int, err error) {
	return w.w.Write(p)
}
