package gcs

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/state/remote"
)

const (
	noPrefix        = ""
	noEncryptionKey = ""
)

// See https://cloud.google.com/storage/docs/using-encryption-keys#generating_your_own_encryption_key
var encryptionKey = "yRyCOikXi1ZDNE0xN3yiFsJjg7LGimoLrGFcLZgQoVk="

func TestStateFile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prefix           string
		defaultStateFile string
		name             string
		wantStateFile    string
		wantLockFile     string
	}{
		{"state", "", "default", "state/default.tfstate", "state/default.tflock"},
		{"state", "", "test", "state/test.tfstate", "state/test.tflock"},
		{"state", "legacy.tfstate", "default", "legacy.tfstate", "legacy.tflock"},
		{"state", "legacy.tfstate", "test", "state/test.tfstate", "state/test.tflock"},
		{"state", "legacy.state", "default", "legacy.state", "legacy.state.tflock"},
		{"state", "legacy.state", "test", "state/test.tfstate", "state/test.tflock"},
	}
	for _, c := range cases {
		b := &Backend{
			prefix:           c.prefix,
			defaultStateFile: c.defaultStateFile,
		}

		if got := b.stateFile(c.name); got != c.wantStateFile {
			t.Errorf("stateFile(%q) = %q, want %q", c.name, got, c.wantStateFile)
		}

		if got := b.lockFile(c.name); got != c.wantLockFile {
			t.Errorf("lockFile(%q) = %q, want %q", c.name, got, c.wantLockFile)
		}
	}
}

func TestRemoteClient(t *testing.T) {
	t.Parallel()

	bucket := bucketName(t)
	be := setupBackend(t, bucket, noPrefix, noEncryptionKey)
	defer teardownBackend(t, be, noPrefix)

	ss, err := be.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("be.StateMgr(%q) = %v", backend.DefaultStateName, err)
	}

	rs, ok := ss.(*remote.State)
	if !ok {
		t.Fatalf("be.StateMgr(): got a %T, want a *remote.State", ss)
	}

	remote.TestClient(t, rs.Client)
}
func TestRemoteClientWithEncryption(t *testing.T) {
	t.Parallel()

	bucket := bucketName(t)
	be := setupBackend(t, bucket, noPrefix, encryptionKey)
	defer teardownBackend(t, be, noPrefix)

	ss, err := be.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("be.StateMgr(%q) = %v", backend.DefaultStateName, err)
	}

	rs, ok := ss.(*remote.State)
	if !ok {
		t.Fatalf("be.StateMgr(): got a %T, want a *remote.State", ss)
	}

	remote.TestClient(t, rs.Client)
}

func TestRemoteLocks(t *testing.T) {
	t.Parallel()

	bucket := bucketName(t)
	be := setupBackend(t, bucket, noPrefix, noEncryptionKey)
	defer teardownBackend(t, be, noPrefix)

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
		t.Fatalf("remoteClient(0) = %v", err)
	}
	c1, err := remoteClient()
	if err != nil {
		t.Fatalf("remoteClient(1) = %v", err)
	}

	remote.TestRemoteLocks(t, c0, c1)
}

func TestBackend(t *testing.T) {
	t.Parallel()

	bucket := bucketName(t)

	be0 := setupBackend(t, bucket, noPrefix, noEncryptionKey)
	defer teardownBackend(t, be0, noPrefix)

	be1 := setupBackend(t, bucket, noPrefix, noEncryptionKey)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
	backend.TestBackendStateForceUnlock(t, be0, be1)
	testStaleLocks(t, be0, be1)
}

func TestBackendWithPrefix(t *testing.T) {
	t.Parallel()

	prefix := "test/prefix"
	bucket := bucketName(t)

	be0 := setupBackend(t, bucket, prefix, noEncryptionKey)
	defer teardownBackend(t, be0, prefix)

	be1 := setupBackend(t, bucket, prefix+"/", noEncryptionKey)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
	testStaleLocks(t, be0, be1)
}

func TestBackendWithEncryption(t *testing.T) {
	t.Parallel()

	bucket := bucketName(t)

	be0 := setupBackend(t, bucket, noPrefix, encryptionKey)
	defer teardownBackend(t, be0, noPrefix)

	be1 := setupBackend(t, bucket, noPrefix, encryptionKey)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
	testStaleLocks(t, be0, be1)
}

func testStaleLocks(t *testing.T, b1, b2 backend.Backend) {
	t.Helper()

	// Get the default state for each
	b1StateMgr, err := b1.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := b1StateMgr.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	b2StateMgr, err := b2.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	if err := b2StateMgr.RefreshState(); err != nil {
		t.Fatalf("bad: %s", err)
	}

	// Reassign so its obvious whats happening
	lockerA := b1StateMgr.(state.Locker)
	lockerB := b2StateMgr.(state.Locker)

	infoA := state.NewLockInfo()
	infoA.Operation = "test"
	infoA.Who = "clientA"

	infoB := state.NewLockInfo()
	infoB.Operation = "test"
	infoB.Who = "clientB"

	// For faster tests, reduce the duration until the lock is considered stale.
	heartbeatInterval = 5 * time.Second
	minHeartbeatAgeUntilStale = 20 * time.Second

	lockIDA, err := lockerA.Lock(infoA)
	if err != nil {
		t.Fatal("unable to get initial lock:", err)
	}

	// Stop heartbeating on the lock file. It will be considered stale after minHeartbeatAgeUntilStale.
	lockerA.(*remote.State).Client.(*remoteClient).stopHeartbeatCh <- true

	// Lock is still held by A after 10 seconds.
	time.Sleep(10 * time.Second)
	_, err = lockerB.Lock(infoB)
	if err == nil {
		lockerA.Unlock(lockIDA)
		t.Fatal("client B obtained lock while held by client A")
	}

	// Wait a bit longer, and the lock will become stale.
	time.Sleep(20 * time.Second)
	lockIDB, err := lockerB.Lock(infoB)
	if err != nil {
		t.Fatal("client B failed to obtain lock that was previously held by client A but that went stale")
	}

	if err := lockerB.Unlock(lockIDB); err != nil {
		t.Fatal("error unlocking client B", err)
	}
}

// setupBackend returns a new GCS backend.
func setupBackend(t *testing.T, bucket, prefix, key string) backend.Backend {
	t.Helper()

	projectID := os.Getenv("GOOGLE_PROJECT")
	if projectID == "" || os.Getenv("TF_ACC") == "" {
		t.Skip("This test creates a bucket in GCS and populates it. " +
			"Since this may incur costs, it will only run if " +
			"the TF_ACC and GOOGLE_PROJECT environment variables are set.")
	}

	config := map[string]interface{}{
		"project":        projectID,
		"bucket":         bucket,
		"prefix":         prefix,
		"encryption_key": key,
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config))
	be := b.(*Backend)

	// create the bucket if it doesn't exist
	bkt := be.storageClient.Bucket(bucket)
	_, err := bkt.Attrs(be.storageContext)
	if err != nil {
		if err != storage.ErrBucketNotExist {
			t.Fatal(err)
		}

		attrs := &storage.BucketAttrs{
			Location: be.region,
		}
		err := bkt.Create(be.storageContext, be.projectID, attrs)
		if err != nil {
			t.Fatal(err)
		}
	}

	return b
}

// teardownBackend deletes all states from be except the default state.
func teardownBackend(t *testing.T, be backend.Backend, prefix string) {
	t.Helper()
	gcsBE, ok := be.(*Backend)
	if !ok {
		t.Fatalf("be is a %T, want a *gcsBackend", be)
	}
	ctx := gcsBE.storageContext

	bucket := gcsBE.storageClient.Bucket(gcsBE.bucketName)
	objs := bucket.Objects(ctx, nil)

	for o, err := objs.Next(); err == nil; o, err = objs.Next() {
		if err := bucket.Object(o.Name).Delete(ctx); err != nil {
			log.Printf("Error trying to delete object: %s %s\n\n", o.Name, err)
		} else {
			log.Printf("Object deleted: %s", o.Name)
		}
	}

	// Delete the bucket itself.
	if err := bucket.Delete(ctx); err != nil {
		t.Errorf("deleting bucket %q failed, manual cleanup may be required: %v", gcsBE.bucketName, err)
	}
}

// bucketName returns a valid bucket name for this test.
func bucketName(t *testing.T) string {
	name := fmt.Sprintf("tf-%x-%s", time.Now().UnixNano(), t.Name())

	// Bucket names must contain 3 to 63 characters.
	if len(name) > 63 {
		name = name[:63]
	}

	return strings.ToLower(name)
}
