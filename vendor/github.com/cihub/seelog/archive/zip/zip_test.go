package zip_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/cihub/seelog/archive/zip"
	"github.com/cihub/seelog/io/iotest"
)

var zipTests = map[string]struct{ want map[string][]byte }{
	"one file": {
		want: map[string][]byte{
			"file": []byte("I am a log file"),
		},
	},
	"multiple files": {
		want: map[string][]byte{
			"file1": []byte("I am log file 1"),
			"file2": []byte("I am log file 2"),
		},
	},
}

func TestWriterAndReader(t *testing.T) {
	for tname, tt := range zipTests {
		f, cleanup := iotest.TempFile(t)
		defer cleanup()
		writeFiles(t, f, tname, tt.want)
		readFiles(t, f, tname, tt.want)
	}
}

// writeFiles iterates through the files we want and writes them as a zipped
// file.
func writeFiles(t *testing.T, f *os.File, tname string, want map[string][]byte) {
	w := zip.NewWriter(f)
	defer w.Close()

	// Write zipped files
	for fname, fbytes := range want {
		fi := iotest.FileInfo(t, fbytes)

		// Write the file
		err := w.NextFile(fname, fi)
		switch err {
		case io.EOF:
			break
		default:
			t.Fatalf("%s: write header for next file: %v", tname, err)
		case nil: // Proceed below
		}
		if _, err := io.Copy(w, bytes.NewReader(fbytes)); err != nil {
			t.Fatalf("%s: copy to writer: %v", tname, err)
		}
	}
}

// readFiles iterates through zipped files and ensures they are the same.
func readFiles(t *testing.T, f *os.File, tname string, want map[string][]byte) {
	// Get zip Reader
	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("%s: stat zipped file: %v", tname, err)
	}
	r, err := zip.NewReader(f, fi.Size())
	if err != nil {
		t.Fatalf("%s: %v", tname, err)
	}

	for {
		fname, err := r.NextFile()
		switch err {
		case io.EOF:
			return
		default:
			t.Fatalf("%s: read header for next file: %v", tname, err)
		case nil: // Proceed below
		}

		wantBytes, ok := want[fname]
		if !ok {
			t.Errorf("%s: read unwanted file: %v", tname, fname)
			continue
		}

		gotBytes, err := ioutil.ReadAll(r)
		if err != nil {
			t.Fatalf("%s: read file: %v", tname, err)
		}

		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("%s: %q = %q but want %q", tname, fname, gotBytes, wantBytes)
		}
	}
}
