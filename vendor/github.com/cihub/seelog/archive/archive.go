package archive

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/cihub/seelog/archive/gzip"
)

// Reader is the interface for reading files from an archive.
type Reader interface {
	NextFile() (name string, err error)
	io.Reader
}

// ReadCloser is the interface that groups Reader with the Close method.
type ReadCloser interface {
	Reader
	io.Closer
}

// Writer is the interface for writing files to an archived format.
type Writer interface {
	NextFile(name string, fi os.FileInfo) error
	io.Writer
}

// WriteCloser is the interface that groups Writer with the Close method.
type WriteCloser interface {
	Writer
	io.Closer
}

type nopCloser struct{ Reader }

func (nopCloser) Close() error { return nil }

// NopCloser returns a ReadCloser with a no-op Close method wrapping the
// provided Reader r.
func NopCloser(r Reader) ReadCloser {
	return nopCloser{r}
}

// Copy copies from src to dest until either EOF is reached on src or an error
// occurs.
//
// When the archive format of src matches that of dst, Copy streams the files
// directly into dst. Otherwise, copy buffers the contents to disk to compute
// headers before writing to dst.
func Copy(dst Writer, src Reader) error {
	switch src := src.(type) {
	case tarReader:
		if dst, ok := dst.(tarWriter); ok {
			return copyTar(dst, src)
		}
	case zipReader:
		if dst, ok := dst.(zipWriter); ok {
			return copyZip(dst, src)
		}
	// Switch on concrete type because gzip has no special methods
	case *gzip.Reader:
		if dst, ok := dst.(*gzip.Writer); ok {
			_, err := io.Copy(dst, src)
			return err
		}
	}

	return copyBuffer(dst, src)
}

func copyBuffer(dst Writer, src Reader) (err error) {
	const defaultFileMode = 0666

	buf, err := ioutil.TempFile("", "archive_copy_buffer")
	if err != nil {
		return err
	}
	defer os.Remove(buf.Name()) // Do not care about failure removing temp
	defer buf.Close()           // Do not care about failure closing temp
	for {
		// Handle the next file
		name, err := src.NextFile()
		switch err {
		case io.EOF: // Done copying
			return nil
		default: // Failed to write: bail out
			return err
		case nil: // Proceed below
		}

		// Buffer the file
		if _, err := io.Copy(buf, src); err != nil {
			return fmt.Errorf("buffer to disk: %v", err)
		}

		// Seek to the start of the file for full file copy
		if _, err := buf.Seek(0, os.SEEK_SET); err != nil {
			return err
		}

		// Set desired file permissions
		if err := os.Chmod(buf.Name(), defaultFileMode); err != nil {
			return err
		}
		fi, err := buf.Stat()
		if err != nil {
			return err
		}

		// Write the buffered file
		if err := dst.NextFile(name, fi); err != nil {
			return err
		}
		if _, err := io.Copy(dst, buf); err != nil {
			return fmt.Errorf("copy to dst: %v", err)
		}
		if err := buf.Truncate(0); err != nil {
			return err
		}
		if _, err := buf.Seek(0, os.SEEK_SET); err != nil {
			return err
		}
	}
}

type tarReader interface {
	Next() (*tar.Header, error)
	io.Reader
}

type tarWriter interface {
	WriteHeader(hdr *tar.Header) error
	io.Writer
}

type zipReader interface {
	Files() []*zip.File
}

type zipWriter interface {
	CreateHeader(fh *zip.FileHeader) (io.Writer, error)
}

func copyTar(w tarWriter, r tarReader) error {
	for {
		hdr, err := r.Next()
		switch err {
		case io.EOF:
			return nil
		default: // Handle error
			return err
		case nil: // Proceed below
		}

		info := hdr.FileInfo()
		// Skip directories
		if info.IsDir() {
			continue
		}
		if err := w.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := io.Copy(w, r); err != nil {
			return err
		}
	}
}

func copyZip(zw zipWriter, r zipReader) error {
	for _, f := range r.Files() {
		if err := copyZipFile(zw, f); err != nil {
			return err
		}
	}
	return nil
}

func copyZipFile(zw zipWriter, f *zip.File) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close() // Read-only

	hdr := f.FileHeader
	hdr.SetModTime(time.Now())
	w, err := zw.CreateHeader(&hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, rc)
	return err
}
