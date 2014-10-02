package remote

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/terraform"
)

func TestEnsureDirectory(t *testing.T) {
	err := EnsureDirectory()
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	cwd, _ := os.Getwd()
	path := filepath.Join(cwd, LocalDirectory)

	_, err = os.Stat(path)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestHiddenStatePath(t *testing.T) {
	path, err := HiddenStatePath()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	cwd, _ := os.Getwd()
	expect := filepath.Join(cwd, LocalDirectory, HiddenStateFile)

	if path != expect {
		t.Fatalf("bad: %v", path)
	}
}

func TestValidConfig(t *testing.T) {
	conf := &terraform.RemoteState{}
	if err := validConfig(conf); err != nil {
		t.Fatalf("blank should be valid: %v", err)
	}
	conf.Server = "http://foo.com"
	if err := validConfig(conf); err == nil {
		t.Fatalf("server without name")
	}
	conf.Server = ""
	conf.AuthToken = "foo"
	if err := validConfig(conf); err == nil {
		t.Fatalf("auth without name")
	}
	conf.Name = "test"
	conf.Server = ""
	conf.AuthToken = ""
	if err := validConfig(conf); err != nil {
		t.Fatalf("should be valid")
	}
	if conf.Server != DefaultServer {
		t.Fatalf("should default server")
	}
}

func TestValidateConfig(t *testing.T) {
	// TODO:
}

func TestReadState(t *testing.T) {
	// TODO
}

func TestBlankState(t *testing.T) {
	remote := &terraform.RemoteState{
		Name:      "foo",
		Server:    "http://foo.com/",
		AuthToken: "foobar",
	}
	r, err := blankState(remote)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	s, err := terraform.ReadState(r)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !remote.Equals(s.Remote) {
		t.Fatalf("remote mismatch")
	}
}

func TestPersist(t *testing.T) {
	tmp, err := ioutil.TempDir("", "remote")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer os.RemoveAll(tmp)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	os.Chdir(tmp)
	defer os.Chdir(cwd)

	EnsureDirectory()

	// Place old state file, should backup
	old := filepath.Join(tmp, LocalDirectory, HiddenStateFile)
	ioutil.WriteFile(old, []byte("test"), 0777)

	remote := &terraform.RemoteState{
		Name:      "foo",
		Server:    "http://foo.com/",
		AuthToken: "foobar",
	}
	blank, _ := blankState(remote)
	if err := Persist(blank); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Check for backup
	backup := filepath.Join(tmp, LocalDirectory, BackupHiddenStateFile)
	out, err := ioutil.ReadFile(backup)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	if string(out) != "test" {
		t.Fatalf("bad: %v", out)
	}

	// Read the state
	out, err = ioutil.ReadFile(old)
	if err != nil {
		t.Fatalf("Err: %v", err)
	}
	s, err := terraform.ReadState(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("Err: %v", err)
	}

	// Check the remote
	if !remote.Equals(s.Remote) {
		t.Fatalf("remote mismatch")
	}
}
