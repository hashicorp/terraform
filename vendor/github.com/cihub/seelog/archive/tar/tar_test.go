package tar_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/cihub/seelog/archive/tar"
	"github.com/cihub/seelog/io/iotest"
)

type file struct {
	name     string
	contents []byte
}

var tarTests = map[string]struct{ want []file }{
	"one file": {
		want: []file{
			{
				name:     "file",
				contents: []byte("I am a log file"),
			},
		},
	},
	"multiple files": {
		want: []file{
			{
				name:     "file1",
				contents: []byte("I am log file 1"),
			},
			{
				name:     "file2",
				contents: []byte("I am log file 2"),
			},
		},
	},
}

func TestWriterAndReader(t *testing.T) {
	for tname, tt := range tarTests {
		f, cleanup := iotest.TempFile(t)
		defer cleanup()
		writeFiles(t, f, tname, tt.want)
		readFiles(t, f, tname, tt.want)
	}
}

// writeFiles iterates through the files we want and writes them as a tarred
// file.
func writeFiles(t *testing.T, f *os.File, tname string, want []file) {
	w := tar.NewWriter(f)
	defer w.Close()

	// Write zipped files
	for _, fwant := range want {
		fi := iotest.FileInfo(t, fwant.contents)

		// Write the file
		err := w.NextFile(fwant.name, fi)
		switch err {
		case io.EOF:
			break
		default:
			t.Fatalf("%s: write header for next file: %v", tname, err)
		case nil: // Proceed below
		}
		if _, err := io.Copy(w, bytes.NewReader(fwant.contents)); err != nil {
			t.Fatalf("%s: copy to writer: %v", tname, err)
		}
	}
}

// readFiles iterates through tarred files and ensures they are the same.
func readFiles(t *testing.T, f *os.File, tname string, want []file) {
	r := tar.NewReader(f)

	for _, fwant := range want {
		fname, err := r.NextFile()
		switch err {
		case io.EOF:
			return
		default:
			t.Fatalf("%s: read header for next file: %v", tname, err)
		case nil: // Proceed below
		}

		if fname != fwant.name {
			t.Fatalf("%s: incorrect file name: got %q but want %q", tname, fname, fwant.name)
			continue
		}

		gotContents, err := ioutil.ReadAll(r)
		if err != nil {
			t.Fatalf("%s: read file: %v", tname, err)
		}

		if !bytes.Equal(gotContents, fwant.contents) {
			t.Errorf("%s: %q = %q but want %q", tname, fname, gotContents, fwant.contents)
		}
	}
}
