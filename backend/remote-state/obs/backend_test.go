package obs

import (
	"crypto/md5"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
	"github.com/huaweicloud/golangsdk/openstack/obs"
)

const (
	defaultPrefix = ""
	defaultKey    = "terraform.tfstate"
)

// verify that we are doing ACC tests or the OBS tests specifically
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_OBS_TEST") == ""
	if skip {
		t.Skip("obs backend tests require setting TF_ACC or TF_OBS_TEST")
	}

	skip = os.Getenv("OS_ACCESS_KEY") == "" || os.Getenv("OS_SECRET_KEY") == ""
	if skip {
		t.Skip("obs backend tests require setting OS_ACCESS_KEY and OS_SECRET_KEY")
	}

	if os.Getenv("OS_REGION_NAME") == "" {
		os.Setenv("OS_REGION_NAME", "cn-north-1")
	}
}

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
		{"", "", "terraform.tfstate", "terraform.tfstate", "terraform.tfstate.tflock"},
		{"", "default", "test.tfstate", "test.tfstate", "test.tfstate.tflock"},
		{"", "dev", "test.tfstate", "dev/test.tfstate", "dev/test.tfstate.tflock"},
		{"terraform", "default", "terraform.tfstate", "terraform/terraform.tfstate", "terraform/terraform.tfstate.tflock"},
		{"terraform/test", "default", "test.tfstate", "terraform/test/test.tfstate", "terraform/test/test.tfstate.tflock"},
		{"terraform/test", "dev", "test.tfstate", "terraform/test/dev/test.tfstate", "terraform/test/dev/test.tfstate.tflock"},
	}

	for i, c := range cases {
		b := &Backend{
			prefix:  c.prefix,
			keyName: c.key,
		}

		if got := b.statePath(c.stateName); got != c.wantStateFile {
			t.Errorf("case %d: stateFile(%q) = %q, want %q", i, c.stateName, got, c.wantStateFile)
		}

		if got := b.lockPath(c.stateName); got != c.wantLockFile {
			t.Errorf("case %d: lockFile(%q) = %q, want %q", i, c.stateName, got, c.wantLockFile)
		}
	}
}

func TestRemoteClient(t *testing.T) {
	testACC(t)
	t.Parallel()

	bucket := bucketName(t)
	be := setupBackend(t, bucket, defaultPrefix, defaultKey, false)

	createOBSBucket(t, be, bucket)
	defer deleteOBSBucket(t, be, bucket)

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

func TestRemoteClientWithPrefix(t *testing.T) {
	testACC(t)
	t.Parallel()

	prefix := "prefix/test"
	bucket := bucketName(t)
	be := setupBackend(t, bucket, prefix, defaultKey, false)

	createOBSBucket(t, be, bucket)
	defer deleteOBSBucket(t, be, bucket)

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
	testACC(t)
	t.Parallel()

	bucket := bucketName(t)
	be := setupBackend(t, bucket, defaultPrefix, defaultKey, true)

	createOBSBucket(t, be, bucket)
	defer deleteOBSBucket(t, be, bucket)

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
	testACC(t)
	t.Parallel()

	bucket := bucketName(t)
	be := setupBackend(t, bucket, defaultPrefix, defaultKey, false)

	createOBSBucket(t, be, bucket)
	defer deleteOBSBucket(t, be, bucket)

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
	testACC(t)
	t.Parallel()

	bucket := bucketName(t)
	be0 := setupBackend(t, bucket, defaultPrefix, defaultKey, false)
	be1 := setupBackend(t, bucket, defaultPrefix, defaultKey, false)

	createOBSBucket(t, be0, bucket)
	defer deleteOBSBucket(t, be0, bucket)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
	backend.TestBackendStateForceUnlock(t, be0, be1)
}

func TestBackendWithPrefix(t *testing.T) {
	testACC(t)
	t.Parallel()

	prefix := "prefix/test"
	bucket := bucketName(t)
	be0 := setupBackend(t, bucket, prefix, defaultKey, false)
	be1 := setupBackend(t, bucket, prefix, defaultKey, false)

	createOBSBucket(t, be0, bucket)
	defer deleteOBSBucket(t, be0, bucket)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
}

func TestBackendWithEncryption(t *testing.T) {
	testACC(t)
	t.Parallel()

	bucket := bucketName(t)
	be0 := setupBackend(t, bucket, defaultPrefix, defaultKey, true)
	be1 := setupBackend(t, bucket, defaultPrefix, defaultKey, true)

	createOBSBucket(t, be0, bucket)
	defer deleteOBSBucket(t, be0, bucket)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
}

func bucketName(t *testing.T) string {
	unique := fmt.Sprintf("%s-%x", t.Name(), time.Now().UnixNano())
	return fmt.Sprintf("terraform-test-%s", fmt.Sprintf("%x", md5.Sum([]byte(unique)))[:10])
}

func setupBackend(t *testing.T, bucket, prefix, key string, encrypt bool) backend.Backend {
	region := os.Getenv("OS_REGION_NAME")
	config := map[string]interface{}{
		"region":  region,
		"bucket":  bucket,
		"prefix":  prefix,
		"key":     key,
		"encrypt": encrypt,
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config))

	return b
}

func createOBSBucket(t *testing.T, be backend.Backend, bucketName string) {
	backend, ok := be.(*Backend)
	if !ok {
		t.Fatalf("be is a %T, want a *obs.Backend", be)
	}
	client := backend.obsClient

	// Be clear about what we're doing in case the user needs to clean this up later.
	region := os.Getenv("OS_REGION_NAME")

	opts := &obs.CreateBucketInput{}
	opts.Bucket = bucketName
	opts.Location = region

	if _, err := client.CreateBucket(opts); err != nil {
		t.Fatal("failed to create test OBS bucket:", err)
	}
}

func deleteOBSBucket(t *testing.T, be backend.Backend, bucketName string) {
	backend, ok := be.(*Backend)
	if !ok {
		t.Fatalf("be is a %T, want a *obs.Backend", be)
	}
	client := backend.obsClient

	warning := "WARNING: Failed to delete the test OBS bucket. " +
		"It may have been left in your Cloud account and may incur storage charges. (error was %s)"

	// first we have to delete objects in the bucket
	listOpts := &obs.ListObjectsInput{
		Bucket: bucketName,
	}
	resp, err := client.ListObjects(listOpts)
	if err != nil {
		t.Logf(warning, err)
		return
	}

	if len(resp.Contents) > 0 {
		objects := make([]obs.ObjectToDelete, len(resp.Contents))
		for i, content := range resp.Contents {
			objects[i].Key = content.Key
		}

		deleteOpts := &obs.DeleteObjectsInput{
			Bucket:  bucketName,
			Objects: objects,
		}

		output, err := client.DeleteObjects(deleteOpts)
		if err != nil {
			t.Logf("failed to delete objects of OBS bucket %s, %s", bucketName, err)
		} else {
			if len(output.Errors) > 0 {
				t.Logf("some objects are still residual in bucket %s: %v", bucketName, output.Errors)
			}
		}
	}

	// Delete the bucket itself
	if _, err = client.DeleteBucket(bucketName); err != nil {
		t.Errorf(warning, err)
	}
}
