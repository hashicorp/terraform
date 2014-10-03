package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGet_badSchema(t *testing.T) {
	dst := tempDir(t)
	u := testModule("basic")
	u = strings.Replace(u, "file", "nope", -1)

	if err := Get(dst, u); err == nil {
		t.Fatal("should error")
	}
}

func TestGet_file(t *testing.T) {
	dst := tempDir(t)
	u := testModule("basic")

	if err := Get(dst, u); err != nil {
		t.Fatalf("err: %s", err)
	}

	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestGet_fileForced(t *testing.T) {
	dst := tempDir(t)
	u := testModule("basic")
	u = "file::" + u

	if err := Get(dst, u); err != nil {
		t.Fatalf("err: %s", err)
	}

	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestGet_fileSubdir(t *testing.T) {
	dst := tempDir(t)
	u := testModule("basic//subdir")

	if err := Get(dst, u); err != nil {
		t.Fatalf("err: %s", err)
	}

	mainPath := filepath.Join(dst, "sub.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestGetCopy_dot(t *testing.T) {
	dst := tempDir(t)
	u := testModule("basic-dot")

	if err := GetCopy(dst, u); err != nil {
		t.Fatalf("err: %s", err)
	}

	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}

	mainPath = filepath.Join(dst, "foo.tf")
	if _, err := os.Stat(mainPath); err == nil {
		t.Fatal("should not have foo.tf")
	}
}

func TestGetCopy_file(t *testing.T) {
	dst := tempDir(t)
	u := testModule("basic")

	if err := GetCopy(dst, u); err != nil {
		t.Fatalf("err: %s", err)
	}

	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestGetDirSubdir(t *testing.T) {
	cases := []struct {
		Input    string
		Dir, Sub string
	}{
		{
			"hashicorp.com",
			"hashicorp.com", "",
		},
		{
			"hashicorp.com//foo",
			"hashicorp.com", "foo",
		},
		{
			"hashicorp.com//foo?bar=baz",
			"hashicorp.com?bar=baz", "foo",
		},
		{
			"file://foo//bar",
			"file://foo", "bar",
		},
	}

	for i, tc := range cases {
		adir, asub := getDirSubdir(tc.Input)
		if adir != tc.Dir {
			t.Fatalf("%d: bad dir: %#v", i, adir)
		}
		if asub != tc.Sub {
			t.Fatalf("%d: bad sub: %#v", i, asub)
		}
	}
}
