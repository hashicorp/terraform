package remote

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

func TestRefreshState_Init(t *testing.T) {
	defer fixDir(testDir(t))
	remote, srv := testRemote(t, nil)
	defer srv.Close()

	sc, err := RefreshState(remote)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if sc != StateChangeInit {
		t.Fatalf("bad: %s", sc)
	}

	local := testReadLocal(t)
	if !local.Remote.Equals(remote) {
		t.Fatalf("Bad: %#v", local)
	}
	if local.Serial != 1 {
		t.Fatalf("Bad: %#v", local)
	}
}

func TestRefreshState_Noop(t *testing.T) {
	// TODO
}

func TestRefreshState_UpdateLocal(t *testing.T) {
	// TODO
}

func TestRefreshState_LocalNewer(t *testing.T) {
	// TODO
}

func TestRefreshState_Conflict(t *testing.T) {
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
	s, err := terraform.ReadState(bytes.NewReader(r))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !remote.Equals(s.Remote) {
		t.Fatalf("remote mismatch")
	}
}

func TestPersist(t *testing.T) {
	tmp, cwd := testDir(t)
	defer fixDir(tmp, cwd)

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
	if err := Persist(bytes.NewReader(blank)); err != nil {
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

// testRemote is used to make a test HTTP server to
// return a given state file
func testRemote(t *testing.T, s *terraform.State) (*terraform.RemoteState, *httptest.Server) {
	var b64md5 string
	buf := bytes.NewBuffer(nil)

	if s != nil {
		terraform.WriteState(s, buf)
		md5 := md5.Sum(buf.Bytes())
		b64md5 = base64.StdEncoding.EncodeToString(md5[:16])
	}

	cb := func(resp http.ResponseWriter, req *http.Request) {
		if s == nil {
			resp.WriteHeader(404)
			return
		}
		resp.Header().Set("Content-MD5", b64md5)
		resp.Write(buf.Bytes())
	}
	srv := httptest.NewServer(http.HandlerFunc(cb))
	remote := &terraform.RemoteState{
		Name:   "foo",
		Server: srv.URL,
	}
	return remote, srv
}

// testDir is used to change the current working directory
// into a test directory that should be remoted after
func testDir(t *testing.T) (string, string) {
	tmp, err := ioutil.TempDir("", "remote")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	os.Chdir(tmp)
	if err := EnsureDirectory(); err != nil {
		t.Fatalf("err: %v", err)
	}
	return tmp, cwd
}

// fixDir is used to as a defer to testDir
func fixDir(tmp, cwd string) {
	os.Chdir(cwd)
	os.RemoveAll(tmp)
}

// testReadLocal is used to just get the local state
func testReadLocal(t *testing.T) *terraform.State {
	path, err := HiddenStatePath()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	raw, err := ioutil.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("err: %v", err)
	}
	if raw == nil {
		return nil
	}
	s, err := terraform.ReadState(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return s
}
