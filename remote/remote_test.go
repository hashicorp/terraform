package remote

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
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
	conf := &terraform.RemoteState{
		Type:   "",
		Config: map[string]string{},
	}
	if err := ValidConfig(conf); err == nil {
		t.Fatalf("blank should be not be valid: %v", err)
	}
	conf.Config["name"] = "hashicorp/test-remote-state"
	conf.Config["access_token"] = "abcd"
	if err := ValidConfig(conf); err != nil {
		t.Fatalf("should be valid")
	}
	if conf.Type != "atlas" {
		t.Fatalf("should default to atlas")
	}
}

func TestRefreshState_Init(t *testing.T) {
	defer testFixCwd(testDir(t))
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

func TestRefreshState_NewVersion(t *testing.T) {
	defer testFixCwd(testDir(t))

	rs := terraform.NewState()
	rs.Serial = 100
	rs.Version = terraform.StateVersion + 1
	remote, srv := testRemote(t, rs)
	defer srv.Close()

	local := terraform.NewState()
	local.Serial = 99
	testWriteLocal(t, local)

	_, err := RefreshState(remote)
	if err == nil {
		t.Fatalf("New version should fail!")
	}
}

func TestRefreshState_Noop(t *testing.T) {
	defer testFixCwd(testDir(t))

	rs := terraform.NewState()
	rs.Serial = 100
	remote, srv := testRemote(t, rs)
	defer srv.Close()

	local := terraform.NewState()
	local.Serial = 100
	testWriteLocal(t, local)

	sc, err := RefreshState(remote)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if sc != StateChangeNoop {
		t.Fatalf("bad: %s", sc)
	}
}

func TestRefreshState_UpdateLocal(t *testing.T) {
	defer testFixCwd(testDir(t))

	rs := terraform.NewState()
	rs.Serial = 100
	remote, srv := testRemote(t, rs)
	defer srv.Close()

	local := terraform.NewState()
	local.Serial = 99
	testWriteLocal(t, local)

	sc, err := RefreshState(remote)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if sc != StateChangeUpdateLocal {
		t.Fatalf("bad: %s", sc)
	}

	// Should update
	local2 := testReadLocal(t)
	if local2.Serial != 100 {
		t.Fatalf("Bad: %#v", local2)
	}
}

func TestRefreshState_LocalNewer(t *testing.T) {
	defer testFixCwd(testDir(t))

	rs := terraform.NewState()
	rs.Serial = 99
	remote, srv := testRemote(t, rs)
	defer srv.Close()

	local := terraform.NewState()
	local.Serial = 100
	testWriteLocal(t, local)

	sc, err := RefreshState(remote)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if sc != StateChangeLocalNewer {
		t.Fatalf("bad: %s", sc)
	}
}

func TestRefreshState_Conflict(t *testing.T) {
	defer testFixCwd(testDir(t))

	rs := terraform.NewState()
	rs.Serial = 50
	rs.RootModule().Outputs["foo"] = "bar"
	remote, srv := testRemote(t, rs)
	defer srv.Close()

	local := terraform.NewState()
	local.Serial = 50
	local.RootModule().Outputs["foo"] = "baz"
	testWriteLocal(t, local)

	sc, err := RefreshState(remote)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if sc != StateChangeConflict {
		t.Fatalf("bad: %s", sc)
	}
}

func TestPushState_NoState(t *testing.T) {
	defer testFixCwd(testDir(t))

	remote, srv := testRemotePush(t, 200)
	defer srv.Close()

	sc, err := PushState(remote, false)
	if err.Error() != "No local state to push" {
		t.Fatalf("err: %v", err)
	}
	if sc != StateChangeNoop {
		t.Fatalf("Bad: %v", sc)
	}
}

func TestPushState_Update(t *testing.T) {
	defer testFixCwd(testDir(t))

	remote, srv := testRemotePush(t, 200)
	defer srv.Close()

	local := terraform.NewState()
	testWriteLocal(t, local)

	sc, err := PushState(remote, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if sc != StateChangeUpdateRemote {
		t.Fatalf("Bad: %v", sc)
	}
}

func TestPushState_RemoteNewer(t *testing.T) {
	defer testFixCwd(testDir(t))

	remote, srv := testRemotePush(t, 412)
	defer srv.Close()

	local := terraform.NewState()
	testWriteLocal(t, local)

	sc, err := PushState(remote, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if sc != StateChangeRemoteNewer {
		t.Fatalf("Bad: %v", sc)
	}
}

func TestPushState_Conflict(t *testing.T) {
	defer testFixCwd(testDir(t))

	remote, srv := testRemotePush(t, 409)
	defer srv.Close()

	local := terraform.NewState()
	testWriteLocal(t, local)

	sc, err := PushState(remote, false)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if sc != StateChangeConflict {
		t.Fatalf("Bad: %v", sc)
	}
}

func TestPushState_Error(t *testing.T) {
	defer testFixCwd(testDir(t))

	remote, srv := testRemotePush(t, 500)
	defer srv.Close()

	local := terraform.NewState()
	testWriteLocal(t, local)

	sc, err := PushState(remote, false)
	if err != ErrRemoteInternal {
		t.Fatalf("err: %v", err)
	}
	if sc != StateChangeNoop {
		t.Fatalf("Bad: %v", sc)
	}
}

func TestDeleteState(t *testing.T) {
	defer testFixCwd(testDir(t))

	remote, srv := testRemotePush(t, 200)
	defer srv.Close()

	local := terraform.NewState()
	testWriteLocal(t, local)

	err := DeleteState(remote)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestBlankState(t *testing.T) {
	remote := &terraform.RemoteState{
		Type: "http",
		Config: map[string]string{
			"address": "http://foo.com/",
		},
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
	defer testFixCwd(tmp, cwd)

	EnsureDirectory()

	// Place old state file, should backup
	old := filepath.Join(tmp, LocalDirectory, HiddenStateFile)
	ioutil.WriteFile(old, []byte("test"), 0777)

	remote := &terraform.RemoteState{
		Type: "http",
		Config: map[string]string{
			"address": "http://foo.com/",
		},
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
		enc := json.NewEncoder(buf)
		if err := enc.Encode(s); err != nil {
			t.Fatalf("err: %v", err)
		}
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
		Type: "http",
		Config: map[string]string{
			"address": srv.URL,
		},
	}
	return remote, srv
}

// testRemotePush is used to make a test HTTP server to
// return a given status code on push
func testRemotePush(t *testing.T, c int) (*terraform.RemoteState, *httptest.Server) {
	cb := func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(c)
	}
	srv := httptest.NewServer(http.HandlerFunc(cb))
	remote := &terraform.RemoteState{
		Type: "http",
		Config: map[string]string{
			"address": srv.URL,
		},
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

// testFixCwd is used to as a defer to testDir
func testFixCwd(tmp, cwd string) {
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

// testWriteLocal is used to write the local state
func testWriteLocal(t *testing.T, s *terraform.State) {
	path, err := HiddenStatePath()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(s); err != nil {
		t.Fatalf("err: %v", err)
	}
	err = ioutil.WriteFile(path, buf.Bytes(), 0777)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}
