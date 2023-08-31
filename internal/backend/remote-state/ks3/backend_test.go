package ks3

import (
	"crypto/md5"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
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

	prefix := "dev/test"
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

// TODO: it seems failed to encrypt data
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

// func TestAssumeRole(t *testing.T) {
// 	t.Parallel()
//
// 	prefix := "prefix/assume-role"
// 	bucket := bucketName(t)
//
// 	role := &Role{
// 		Krn:             "krn:ksc:iam::73403251:role/tf-backend-role",
// 		SessionName:     "backend-test-assume-role",
// 		SessionDuration: 3600,
// 	}
// 	be := setupBackend(t, bucket, prefix, defaultKey, false, role)
// 	defer teardownBackend(t, be)
//
// 	ss, err := be.StateMgr(backend.DefaultStateName)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %s", err)
// 	}
//
// 	rs, ok := ss.(*remote.State)
// 	if !ok {
// 		t.Fatalf("wrong state manager type\ngot:  %T\nwant: %T", ss, rs)
// 	}
//
// 	remote.TestClient(t, rs.Client)
// }

// TODO: completed dead lock release
func TestDeadLockRelease(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		ld           string
		waitTime     time.Duration
		requiredLock bool
	}{
		"ignore old lock with 3 minutes": {
			"0",
			3 * time.Minute,
			true,
		},
		"ignore old lock with 3 hours": {
			"0",
			3 * time.Hour,
			true,
		},
		"unlimited with 3 minutes": {
			"-1",
			3 * time.Minute,
			false,
		},
		"unlimited with 3 hours": {
			"-1",
			3 * time.Hour,
			false,
		},
		"duration 3 hours": {
			"3h",
			2 * time.Hour,
			false,
		},
		"duration 3 hours required": {
			"3h",
			4 * time.Hour,
			true,
		},
		"duration 3 minute not required": {
			"3m",
			2 * time.Minute,
			false,
		},
		"duration 3 minute required": {
			"3m",
			4 * time.Minute,
			true,
		},
		"duration 0 minute": {
			"0m",
			4 * time.Minute,
			true,
		},
	}

	bucket := bucketName(t)

	for name, tcase := range cases {
		t.Run(name, func(t *testing.T) {

			prefix := "deadlock/" + strings.ReplaceAll(name, " ", "_")
			be0 := setupBackendWithLockDuration(t, bucket, prefix, defaultKey, false, tcase.ld)
			be1 := setupBackendWithLockDuration(t, bucket, prefix, defaultKey, false, tcase.ld)
			defer teardownBackend(t, be0)

			// Get the default state for each
			b1StateMgr, err := be0.StateMgr(backend.DefaultStateName)
			if err != nil {
				t.Fatalf("error: %s", err)
			}
			if err := b1StateMgr.RefreshState(); err != nil {
				t.Fatalf("bad: %s", err)
			}

			// Fast exit if this doesn't support locking at all
			if _, ok := b1StateMgr.(statemgr.Locker); !ok {
				t.Logf("TestBackend: backend %T doesn't support state locking, not testing", be0)
				return
			}

			t.Logf("TestBackend: testing state deat lock releasing for %T", be0)

			b2StateMgr, err := be1.StateMgr(backend.DefaultStateName)
			if err != nil {
				t.Fatalf("error: %s", err)
			}
			if err := b2StateMgr.RefreshState(); err != nil {
				t.Fatalf("bad: %s", err)
			}
			lockerA := b1StateMgr.(statemgr.Locker)
			lockerB := b2StateMgr.(statemgr.Locker)

			// mock old lock
			infoA := statemgr.NewLockInfo()
			infoA.Who = "ClientA"
			infoA.Operation = "test"
			// simulating the dead-lock existence time
			infoA.Created = infoA.Created.Add(-tcase.waitTime)

			// mock the new lock
			infoB := statemgr.NewLockInfo()
			infoB.Who = "ClientB"
			infoB.Operation = "test"

			lockIdA, err := lockerA.Lock(infoA)
			if err != nil {
				t.Fatalf("unable to require initial lock, err: %v", err)
			}
			if lockIdA == "" {
				t.Fatalf("lock id cannot be empty string")
			}

			// wants to get the remote lock
			lockIdB, err := lockerB.Lock(infoB)
			if err != nil {
				unlockErr := lockerA.Unlock(lockIdA)
				if unlockErr != nil {
					t.Fatal(unlockErr)
				}
				if tcase.requiredLock {
					t.Fatalf("expect required lock, but unable to obtain lock, %v", err)
				}
				return
			} else {
				if !tcase.requiredLock {
					unlockErr := lockerB.Unlock(lockIdB)
					if unlockErr != nil {
						t.Fatal(unlockErr)
					}
					t.Fatalf("expect fail to get lock, but got the lock")
				}
			}
			if lockIdB == "" {
				t.Fatalf("lockerB id cannot be empty string")
			}

			if err := lockerA.Unlock(lockIdA); err == nil {
				t.Fatalf("old lock has been covered, but old lock is exist")
			}

			if err := lockerB.Unlock(lockIdB); err != nil {
				t.Fatal(err)
			}
		})
	}

}

func setupBackend(t *testing.T, bucket, prefix, key string, encrypt bool, roles ...*Role) backend.Backend {
	t.Helper()

	skip := os.Getenv(ENV_ACCESS_KEY) == ""
	if skip {
		t.Skip("This test require setting KSYUN_ACCESS_KEY and KSYUN_SECRET_KEY environment variables")
	}

	if os.Getenv(ENV_REGION) == "" {
		os.Setenv(ENV_REGION, "cn-beijing-6")
	}

	region := os.Getenv(ENV_REGION)

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

func setupBackendWithLockDuration(t *testing.T, bucket, prefix, key string, encrypt bool, ld string, roles ...*Role) backend.Backend {
	t.Helper()

	skip := os.Getenv(ENV_ACCESS_KEY) == ""
	if skip {
		t.Skip("This test require setting KSYUN_ACCESS_KEY and KSYUN_SECRET_KEY environment variables")
	}

	if os.Getenv(ENV_REGION) == "" {
		os.Setenv(ENV_REGION, "cn-beijing-6")
	}

	region := os.Getenv(ENV_REGION)

	config := map[string]interface{}{
		"region":               region,
		"bucket":               bucket,
		"workspace_key_prefix": prefix,
		"key":                  key,
		"endpoint":             endpoint,
		"encrypt":              encrypt,
		"lock_duration":        ld,
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
