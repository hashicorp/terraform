package getter

import (
	"runtime"
	"testing"
)

type fileTest struct {
	in, pwd, out string
	err          bool
}

var fileTests = []fileTest{
	{"./foo", "/pwd", "file:///pwd/foo", false},
	{"./foo?foo=bar", "/pwd", "file:///pwd/foo?foo=bar", false},
	{"foo", "/pwd", "file:///pwd/foo", false},
}

var unixFileTests = []fileTest{
	{"/foo", "/pwd", "file:///foo", false},
	{"/foo?bar=baz", "/pwd", "file:///foo?bar=baz", false},
}

var winFileTests = []fileTest{
	{"/foo", "/pwd", "file:///pwd/foo", false},
	{`C:\`, `/pwd`, `file://C:/`, false},
	{`C:\?bar=baz`, `/pwd`, `file://C:/?bar=baz`, false},
}

func TestFileDetector(t *testing.T) {
	if runtime.GOOS == "windows" {
		fileTests = append(fileTests, winFileTests...)
	} else {
		fileTests = append(fileTests, unixFileTests...)
	}

	f := new(FileDetector)
	for i, tc := range fileTests {
		out, ok, err := f.Detect(tc.in, tc.pwd)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		if !ok {
			t.Fatal("not ok")
		}

		if out != tc.out {
			t.Fatalf("%d: bad: %#v", i, out)
		}
	}
}

var noPwdFileTests = []fileTest{
	{in: "./foo", pwd: "", out: "", err: true},
	{in: "foo", pwd: "", out: "", err: true},
}

var noPwdUnixFileTests = []fileTest{
	{in: "/foo", pwd: "", out: "file:///foo", err: false},
}

var noPwdWinFileTests = []fileTest{
	{in: "/foo", pwd: "", out: "", err: true},
	{in: `C:\`, pwd: ``, out: `file://C:/`, err: false},
}

func TestFileDetector_noPwd(t *testing.T) {
	if runtime.GOOS == "windows" {
		noPwdFileTests = append(noPwdFileTests, noPwdWinFileTests...)
	} else {
		noPwdFileTests = append(noPwdFileTests, noPwdUnixFileTests...)
	}

	f := new(FileDetector)
	for i, tc := range noPwdFileTests {
		out, ok, err := f.Detect(tc.in, tc.pwd)
		if err != nil != tc.err {
			t.Fatalf("%d: err: %s", i, err)
		}
		if !ok {
			t.Fatal("not ok")
		}

		if out != tc.out {
			t.Fatalf("%d: bad: %#v", i, out)
		}
	}
}
