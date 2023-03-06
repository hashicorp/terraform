package cos

import (
	"crypto/md5"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/remote"
)

const (
	defaultPrefix = ""
	defaultKey    = "terraform.tfstate"
)

// Testing Thanks to GCS

func TestStateFile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prefix        string
		stateName     string
		key           string
		wantStateFile string
		wantLockFile  string
	}{
		{"", "default", "default.tfstate", "default.tfstate", "default.tfstate.tflock"},
		{"", "default", "test.tfstate", "test.tfstate", "test.tfstate.tflock"},
		{"", "dev", "test.tfstate", "dev/test.tfstate", "dev/test.tfstate.tflock"},
		{"terraform/test", "default", "default.tfstate", "terraform/test/default.tfstate", "terraform/test/default.tfstate.tflock"},
		{"terraform/test", "default", "test.tfstate", "terraform/test/test.tfstate", "terraform/test/test.tfstate.tflock"},
		{"terraform/test", "dev", "test.tfstate", "terraform/test/dev/test.tfstate", "terraform/test/dev/test.tfstate.tflock"},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%s %s %s", c.prefix, c.key, c.stateName), func(t *testing.T) {
			b := &Backend{
				prefix: c.prefix,
				key:    c.key,
			}
			if got, want := b.stateFile(c.stateName), c.wantStateFile; got != want {
				t.Errorf("wrong state file name\ngot:  %s\nwant: %s", got, want)
			}
			if got, want := b.lockFile(c.stateName), c.wantLockFile; got != want {
				t.Errorf("wrong lock file name\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

func TestRemoteClient(t *testing.T) {
	t.Parallel()

	bucket := bucketName(t)

	be := setupBackend(t, bucket, defaultPrefix, defaultKey, false)
	defer teardownBackend(t, be)

	ss, err := be.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	rs, ok := ss.(*remote.State)
	if !ok {
		t.Fatalf("wrong state manager type\ngot:  %T\nwant: %T", ss, rs)
	}

	remote.TestClient(t, rs.Client)
}

func TestRemoteClientWithPrefix(t *testing.T) {
	t.Parallel()

	prefix := "prefix/test"
	bucket := bucketName(t)

	be := setupBackend(t, bucket, prefix, defaultKey, false)
	defer teardownBackend(t, be)

	ss, err := be.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	rs, ok := ss.(*remote.State)
	if !ok {
		t.Fatalf("wrong state manager type\ngot:  %T\nwant: %T", ss, rs)
	}

	remote.TestClient(t, rs.Client)
}

func TestRemoteClientWithEncryption(t *testing.T) {
	t.Parallel()

	bucket := bucketName(t)

	be := setupBackend(t, bucket, defaultPrefix, defaultKey, true)
	defer teardownBackend(t, be)

	ss, err := be.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	rs, ok := ss.(*remote.State)
	if !ok {
		t.Fatalf("wrong state manager type\ngot:  %T\nwant: %T", ss, rs)
	}

	remote.TestClient(t, rs.Client)
}

func TestRemoteLocks(t *testing.T) {
	t.Parallel()

	bucket := bucketName(t)

	be := setupBackend(t, bucket, defaultPrefix, defaultKey, false)
	defer teardownBackend(t, be)

	remoteClient := func() (remote.Client, error) {
		ss, err := be.StateMgr(backend.DefaultStateName)
		if err != nil {
			return nil, err
		}

		rs, ok := ss.(*remote.State)
		if !ok {
			return nil, fmt.Errorf("be.StateMgr(): got a %T, want a *remote.State", ss)
		}

		return rs.Client, nil
	}

	c0, err := remoteClient()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	c1, err := remoteClient()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	remote.TestRemoteLocks(t, c0, c1)
}

func TestBackend(t *testing.T) {
	t.Parallel()

	bucket := bucketName(t)

	be0 := setupBackend(t, bucket, defaultPrefix, defaultKey, false)
	defer teardownBackend(t, be0)

	be1 := setupBackend(t, bucket, defaultPrefix, defaultKey, false)
	defer teardownBackend(t, be1)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
	backend.TestBackendStateForceUnlock(t, be0, be1)
}

func TestBackendWithPrefix(t *testing.T) {
	t.Parallel()

	prefix := "prefix/test"
	bucket := bucketName(t)

	be0 := setupBackend(t, bucket, prefix, defaultKey, false)
	defer teardownBackend(t, be0)

	be1 := setupBackend(t, bucket, prefix+"/", defaultKey, false)
	defer teardownBackend(t, be1)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
}

func TestBackendWithEncryption(t *testing.T) {
	t.Parallel()

	bucket := bucketName(t)

	be0 := setupBackend(t, bucket, defaultPrefix, defaultKey, true)
	defer teardownBackend(t, be0)

	be1 := setupBackend(t, bucket, defaultPrefix, defaultKey, true)
	defer teardownBackend(t, be1)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
}

func setupBackend(t *testing.T, bucket, prefix, key string, encrypt bool) backend.Backend {
	t.Helper()

	skip := os.Getenv("TF_COS_APPID") == ""
	if skip {
		t.Skip("This test require setting TF_COS_APPID environment variables")
	}

	if os.Getenv(PROVIDER_REGION) == "" {
		os.Setenv(PROVIDER_REGION, "ap-guangzhou")
	}

	appId := os.Getenv("TF_COS_APPID")
	region := os.Getenv(PROVIDER_REGION)

	config := map[string]interface{}{
		"region": region,
		"bucket": bucket + appId,
		"prefix": prefix,
		"key":    key,
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config))
	be := b.(*Backend)

	c, err := be.client("tencentcloud")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	err = c.putBucket()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	return b
}

func teardownBackend(t *testing.T, b backend.Backend) {
	t.Helper()

	c, err := b.(*Backend).client("tencentcloud")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	err = c.deleteBucket(true)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}

func bucketName(t *testing.T) string {
	unique := fmt.Sprintf("%s-%x", t.Name(), time.Now().UnixNano())
	return fmt.Sprintf("terraform-test-%s-%s", fmt.Sprintf("%x", md5.Sum([]byte(unique)))[:10], "")
}
