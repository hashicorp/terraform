package ks3

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
	endpoint      = "ks3-cn-beijing.ksyuncs.com"
	testBucket    = "tf-backend"
)

type Role struct {
	Krn             string
	SessionName     string
	SessionDuration int
}

func TestBackendStateFile(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		prefix          string
		workspace       string
		key             string
		expectStateFile string
		expectLockFile  string
	}{
		"default": {
			"", "default", "terraform.tfstate", "terraform.tfstate", "terraform.tfstate.tflock",
		},
		"ws-development": {
			"", "development", "ksyun.tfstate", "development/ksyun.tfstate", "development/ksyun.tfstate.tflock",
		},
		"prefix-dev": {
			"dev/role1", "default", "ksyun.tfstate", "dev/role1/ksyun.tfstate", "dev/role1/ksyun.tfstate.tflock",
		},
	}

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {
			b := &Backend{
				workspaceKeyPrefix: tcase.prefix,
				key:                tcase.key,
			}

			if got, expect := b.stateFile(tcase.workspace), tcase.expectStateFile; got != expect {
				t.Errorf("state file expect %s, but got %s", expect, got)
			}
			if got, expect := b.lockFile(tcase.workspace), tcase.expectLockFile; got != expect {
				t.Errorf("lock file expect %s, but got %s", expect, got)
			}

		})
	}
}

func TestRemoteClient(t *testing.T) {
	t.Parallel()

	bucket := "tf-backend"

	be := setupBackend(t, bucket, defaultPrefix, defaultKey, false)

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

	prefix := "dev/test"
	bucket := "tf-backend"

	be := setupBackend(t, bucket, prefix, defaultKey, false)

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

// TODO: it seems failed to encrypt data
func TestRemoteClientWithEncryption(t *testing.T) {
	t.Parallel()

	bucket := "tf-backend"

	be := setupBackend(t, bucket, defaultPrefix, defaultKey, true)

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

	bucket := testBucket

	be := setupBackend(t, bucket, defaultPrefix, defaultKey, false)

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
	bucket := testBucket

	be0 := setupBackend(t, bucket, prefix, defaultKey, false)

	be1 := setupBackend(t, bucket, prefix+"/", defaultKey, false)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
}

func TestAssumeRole(t *testing.T) {
	t.Parallel()

	prefix := "prefix/assume-role"
	bucket := testBucket

	role := &Role{
		Krn:             "krn:ksc:iam::73403251:role/tf-backend-role",
		SessionName:     "backend-test-assume-role",
		SessionDuration: 3600,
	}
	be := setupBackend(t, bucket, prefix, defaultKey, false, role)

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

func setupBackend(t *testing.T, bucket, prefix, key string, encrypt bool, roles ...*Role) backend.Backend {
	t.Helper()

	if os.Getenv(PROVIDER_REGION) == "" {
		os.Setenv(PROVIDER_REGION, "cn-beijing-6")
	}

	region := os.Getenv(PROVIDER_REGION)

	config := map[string]interface{}{
		"region":               region,
		"bucket":               bucket,
		"workspace_key_prefix": prefix,
		"key":                  key,
		"endpoint":             endpoint,
		"encrypt":              encrypt,
	}

	if roles != nil && len(roles) > 0 {
		for _, role := range roles {
			config["role_krn"] = role.Krn
			config["session_name"] = role.SessionName
			config["assume_role_duration_seconds"] = role.SessionDuration
		}

	}
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config))
	be := b.(*Backend)

	c, err := be.client("ksyun")
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

	c, err := b.(*Backend).client("ksyun")
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
	return fmt.Sprintf("tf-backend-test-%s", fmt.Sprintf("%x", md5.Sum([]byte(unique)))[:10])
}
