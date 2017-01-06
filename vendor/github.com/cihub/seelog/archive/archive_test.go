package archive_test

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/cihub/seelog/archive"
	"github.com/cihub/seelog/archive/gzip"
	"github.com/cihub/seelog/archive/tar"
	"github.com/cihub/seelog/archive/zip"
	"github.com/cihub/seelog/io/iotest"
)

const (
	gzipType = "gzip"
	tarType  = "tar"
	zipType  = "zip"
)

var types = []string{gzipType, tarType, zipType}

type file struct {
	name     string
	contents []byte
}

var (
	oneFile = []file{
		{
			name:     "file1",
			contents: []byte("This is a single log."),
		},
	}
	twoFiles = []file{
		{
			name:     "file1",
			contents: []byte("This is a log."),
		},
		{
			name:     "file2",
			contents: []byte("This is another log."),
		},
	}
)

type testCase struct {
	srcType, dstType string
	in               []file
}

func copyTests() map[string]testCase {
	// types X types X files
	tests := make(map[string]testCase, len(types)*len(types)*2)
	for _, srct := range types {
		for _, dstt := range types {
			tests[fmt.Sprintf("%s to %s: one file", srct, dstt)] = testCase{
				srcType: srct,
				dstType: dstt,
				in:      oneFile,
			}
			// gzip does not handle more than one file
			if srct != gzipType && dstt != gzipType {
				tests[fmt.Sprintf("%s to %s: two files", srct, dstt)] = testCase{
					srcType: srct,
					dstType: dstt,
					in:      twoFiles,
				}
			}
		}
	}
	return tests
}

func TestCopy(t *testing.T) {
	srcb, dstb := new(bytes.Buffer), new(bytes.Buffer)
	for tname, tt := range copyTests() {
		// Reset buffers between tests
		srcb.Reset()
		dstb.Reset()

		// Last file name (needed for gzip.NewReader)
		var fname string

		// Seed the src
		srcw := writer(t, tname, srcb, tt.srcType)
		for _, f := range tt.in {
			srcw.NextFile(f.name, iotest.FileInfo(t, f.contents))
			mustCopy(t, tname, srcw, bytes.NewReader(f.contents))
			fname = f.name
		}
		mustClose(t, tname, srcw)

		// Perform the copy
		srcr := reader(t, tname, srcb, tt.srcType, fname)
		dstw := writer(t, tname, dstb, tt.dstType)
		if err := archive.Copy(dstw, srcr); err != nil {
			t.Fatalf("%s: %v", tname, err)
		}
		srcr.Close() // Read-only
		mustClose(t, tname, dstw)

		// Read back dst to confirm our expectations
		dstr := reader(t, tname, dstb, tt.dstType, fname)
		for _, want := range tt.in {
			buf := new(bytes.Buffer)
			name, err := dstr.NextFile()
			if err != nil {
				t.Fatalf("%s: %v", tname, err)
			}
			mustCopy(t, tname, buf, dstr)
			got := file{
				name:     name,
				contents: buf.Bytes(),
			}

			switch {
			case got.name != want.name:
				t.Errorf("%s: got file %q but want file %q",
					tname, got.name, want.name)

			case !bytes.Equal(got.contents, want.contents):
				t.Errorf("%s: mismatched contents in %q: got %q but want %q",
					tname, got.name, got.contents, want.contents)
			}
		}
		dstr.Close()
	}
}

func writer(t *testing.T, tname string, w io.Writer, atype string) archive.WriteCloser {
	switch atype {
	case gzipType:
		return gzip.NewWriter(w)
	case tarType:
		return tar.NewWriter(w)
	case zipType:
		return zip.NewWriter(w)
	}
	t.Fatalf("%s: unrecognized archive type: %s", tname, atype)
	panic("execution continued after (*testing.T).Fatalf")
}

func reader(t *testing.T, tname string, buf *bytes.Buffer, atype string, fname string) archive.ReadCloser {
	switch atype {
	case gzipType:
		gr, err := gzip.NewReader(buf, fname)
		if err != nil {
			t.Fatalf("%s: %v", tname, err)
		}
		return gr
	case tarType:
		return archive.NopCloser(tar.NewReader(buf))
	case zipType:
		zr, err := zip.NewReader(
			bytes.NewReader(buf.Bytes()),
			int64(buf.Len()))
		if err != nil {
			t.Fatalf("%s: new zip reader: %v", tname, err)
		}
		return archive.NopCloser(zr)
	}
	t.Fatalf("%s: unrecognized archive type: %s", tname, atype)
	panic("execution continued after (*testing.T).Fatalf")
}

func mustCopy(t *testing.T, tname string, dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatalf("%s: copy: %v", tname, err)
	}
}

func mustClose(t *testing.T, tname string, c io.Closer) {
	if err := c.Close(); err != nil {
		t.Fatalf("%s: close: %v", tname, err)
	}
}
