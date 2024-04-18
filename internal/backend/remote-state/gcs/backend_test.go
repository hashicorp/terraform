// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package gcs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/storage"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

const (
	noPrefix        = ""
	noEncryptionKey = ""
	noKmsKeyName    = ""
)

// See https://cloud.google.com/storage/docs/using-encryption-keys#generating_your_own_encryption_key
const encryptionKey = "yRyCOikXi1ZDNE0xN3yiFsJjg7LGimoLrGFcLZgQoVk="

// KMS key ring name and key name are hardcoded here and re-used because key rings (and keys) cannot be deleted
// Test code asserts their presence and creates them if they're absent. They're not deleted at the end of tests.
// See: https://cloud.google.com/kms/docs/faq#cannot_delete
const (
	keyRingName = "tf-gcs-backend-acc-tests"
	keyName     = "tf-test-key-1"
	kmsRole     = "roles/cloudkms.cryptoKeyEncrypterDecrypter" // GCS service account needs this binding on the created key
)

var keyRingLocation = os.Getenv("GOOGLE_REGION")

func preCheckTestAcc(t *testing.T) {
	if v := os.Getenv("TF_ACC"); v == "" {
		t.Fatalf("TF_ACC must be set to run acceptance tests, as they provision real resources")
	}
}

func TestStateFile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prefix        string
		name          string
		wantStateFile string
		wantLockFile  string
	}{
		{"state", "default", "state/default.tfstate", "state/default.tflock"},
		{"state", "test", "state/test.tfstate", "state/test.tflock"},
		{"state", "test", "state/test.tfstate", "state/test.tflock"},
		{"state", "test", "state/test.tfstate", "state/test.tflock"},
	}
	for _, c := range cases {
		b := &Backend{
			prefix: c.prefix,
		}

		if got := b.stateFile(c.name); got != c.wantStateFile {
			t.Errorf("stateFile(%q) = %q, want %q", c.name, got, c.wantStateFile)
		}

		if got := b.lockFile(c.name); got != c.wantLockFile {
			t.Errorf("lockFile(%q) = %q, want %q", c.name, got, c.wantLockFile)
		}
	}
}

func TestAccRemoteClient(t *testing.T) {
	preCheckTestAcc(t)
	t.Parallel()

	bucket := bucketName(t)
	be := setupBackend(t, bucket, noPrefix, noEncryptionKey, noKmsKeyName)
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

func TestAccRemoteClientWithEncryption(t *testing.T) {
	preCheckTestAcc(t)
	t.Parallel()

	bucket := bucketName(t)
	be := setupBackend(t, bucket, noPrefix, encryptionKey, noKmsKeyName)
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

func TestAccRemoteLocks(t *testing.T) {
	preCheckTestAcc(t)
	t.Parallel()

	bucket := bucketName(t)
	be := setupBackend(t, bucket, noPrefix, noEncryptionKey, noKmsKeyName)
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

func TestAccBackend(t *testing.T) {
	preCheckTestAcc(t)
	t.Parallel()

	bucket := bucketName(t)

	be0 := setupBackend(t, bucket, noPrefix, noEncryptionKey, noKmsKeyName)
	defer teardownBackend(t, be0, noPrefix)

	be1 := setupBackend(t, bucket, noPrefix, noEncryptionKey, noKmsKeyName)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
	backend.TestBackendStateForceUnlock(t, be0, be1)
}

func TestAccBackendWithPrefix(t *testing.T) {
	preCheckTestAcc(t)
	t.Parallel()

	prefix := "test/prefix"
	bucket := bucketName(t)

	be0 := setupBackend(t, bucket, prefix, noEncryptionKey, noKmsKeyName)
	defer teardownBackend(t, be0, prefix)

	be1 := setupBackend(t, bucket, prefix+"/", noEncryptionKey, noKmsKeyName)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
}

func TestAccBackendWithCustomerSuppliedEncryption(t *testing.T) {
	preCheckTestAcc(t)
	t.Parallel()

	bucket := bucketName(t)

	be0 := setupBackend(t, bucket, noPrefix, encryptionKey, noKmsKeyName)
	defer teardownBackend(t, be0, noPrefix)

	be1 := setupBackend(t, bucket, noPrefix, encryptionKey, noKmsKeyName)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
}

func TestAccBackendWithCustomerManagedKMSEncryption(t *testing.T) {
	preCheckTestAcc(t)
	t.Parallel()

	projectID := os.Getenv("GOOGLE_PROJECT")
	bucket := bucketName(t)

	// Taken from global variables in test file
	kmsDetails := map[string]string{
		"project":  projectID,
		"location": keyRingLocation,
		"ringName": keyRingName,
		"keyName":  keyName,
	}

	kmsName := setupKmsKey(t, kmsDetails)

	be0 := setupBackend(t, bucket, noPrefix, noEncryptionKey, kmsName)
	defer teardownBackend(t, be0, noPrefix)

	be1 := setupBackend(t, bucket, noPrefix, noEncryptionKey, kmsName)

	backend.TestBackendStates(t, be0)
	backend.TestBackendStateLocks(t, be0, be1)
}

func TestBackendEncryptionKeyEmptyConflict(t *testing.T) {
	// This test is for the edge case where encryption_key and
	// kms_encryption_key are both set in the configuration but set to empty
	// strings. The "SDK-like" helpers treat unset as empty string, so
	// we need an extra rule to catch them both being set to empty string
	// directly inside the configuration, and this test covers that
	// special case.
	//
	// The following assumes that the validation check we're testing will, if
	// failing, always block attempts to reach any real GCP services, and so
	// this test should be fine to run without an acceptance testing opt-in.

	// This test is for situations where these environment variables are not set.
	t.Setenv("GOOGLE_ENCRYPTION_KEY", "")
	t.Setenv("GOOGLE_KMS_ENCRYPTION_KEY", "")

	backend := New()
	schema := backend.ConfigSchema()
	rawVal := cty.ObjectVal(map[string]cty.Value{
		"bucket": cty.StringVal("fake-placeholder"),

		// These are both empty strings but should still be considered as
		// set when we enforce teh rule that they can't both be set at once.
		"encryption_key":     cty.StringVal(""),
		"kms_encryption_key": cty.StringVal(""),
	})
	// The following mimicks how the terraform_remote_state data source
	// treats its "config" argument, which is a realistic situation where
	// we take an arbitrary object and try to force it to conform to the
	// backend's schema.
	configVal, err := schema.CoerceValue(rawVal)
	if err != nil {
		t.Fatalf("unexpected coersion error: %s", err)
	}
	configVal, diags := backend.PrepareConfig(configVal)
	if diags.HasErrors() {
		t.Fatalf("unexpected PrepareConfig error: %s", diags.Err().Error())
	}

	configDiags := backend.Configure(configVal)
	if !configDiags.HasErrors() {
		t.Fatalf("unexpected success; want error")
	}
	gotErr := configDiags.Err().Error()
	wantErr := `can't set both encryption_key and kms_encryption_key`
	if !strings.Contains(gotErr, wantErr) {
		t.Errorf("wrong error\ngot: %s\nwant substring: %s", gotErr, wantErr)
	}
}

// setupBackend returns a new GCS backend.
func setupBackend(t *testing.T, bucket, prefix, key, kmsName string) backend.Backend {
	t.Helper()
	ctx := context.Background()

	projectID := os.Getenv("GOOGLE_PROJECT")
	if projectID == "" || os.Getenv("TF_ACC") == "" {
		t.Skip("This test creates a bucket in GCS and populates it. " +
			"Since this may incur costs, it will only run if " +
			"the TF_ACC and GOOGLE_PROJECT environment variables are set.")
	}

	config := map[string]interface{}{
		"bucket": bucket,
		"prefix": prefix,
	}
	// Only add encryption keys to config if non-zero value set
	// If not set here, default values are supplied in `TestBackendConfig` by `PrepareConfig` function call
	if len(key) > 0 {
		config["encryption_key"] = key
	}
	if len(kmsName) > 0 {
		config["kms_encryption_key"] = kmsName
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config))
	be := b.(*Backend)

	// create the bucket if it doesn't exist
	bkt := be.storageClient.Bucket(bucket)
	_, err := bkt.Attrs(ctx)
	if err != nil {
		if err != storage.ErrBucketNotExist {
			t.Fatal(err)
		}

		attrs := &storage.BucketAttrs{
			Location: os.Getenv("GOOGLE_REGION"),
		}
		err := bkt.Create(ctx, projectID, attrs)
		if err != nil {
			t.Fatal(err)
		}
	}

	return b
}

// setupKmsKey asserts that a KMS key chain and key exist and necessary IAM bindings are in place
// If the key ring or key do not exist they are created and permissions are given to the GCS Service account
func setupKmsKey(t *testing.T, keyDetails map[string]string) string {
	t.Helper()

	projectID := os.Getenv("GOOGLE_PROJECT")
	if projectID == "" || os.Getenv("TF_ACC") == "" {
		t.Skip("This test creates a KMS key ring and key in Cloud KMS. " +
			"Since this may incur costs, it will only run if " +
			"the TF_ACC and GOOGLE_PROJECT environment variables are set.")
	}

	// KMS Client
	ctx := context.Background()
	opts, err := testGetClientOptions(t)
	if err != nil {
		e := fmt.Errorf("testGetClientOptions() failed: %s", err)
		t.Fatal(e)
	}
	c, err := kms.NewKeyManagementClient(ctx, opts...)
	if err != nil {
		e := fmt.Errorf("kms.NewKeyManagementClient() failed: %v", err)
		t.Fatal(e)
	}
	defer c.Close()

	// Get KMS key ring, create if doesn't exist
	reqGetKeyRing := &kmspb.GetKeyRingRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", keyDetails["project"], keyDetails["location"], keyDetails["ringName"]),
	}
	var keyRing *kmspb.KeyRing
	keyRing, err = c.GetKeyRing(ctx, reqGetKeyRing)
	if err != nil {
		if !strings.Contains(err.Error(), "NotFound") {
			// Handle unexpected error that isn't related to the key ring not being made yet
			t.Fatal(err)
		}
		// Create key ring that doesn't exist
		t.Logf("Cloud KMS key ring `%s` not found: creating key ring",
			fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", keyDetails["project"], keyDetails["location"], keyDetails["ringName"]),
		)
		reqCreateKeyRing := &kmspb.CreateKeyRingRequest{
			Parent:    fmt.Sprintf("projects/%s/locations/%s", keyDetails["project"], keyDetails["location"]),
			KeyRingId: keyDetails["ringName"],
		}
		keyRing, err = c.CreateKeyRing(ctx, reqCreateKeyRing)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Cloud KMS key ring `%s` created successfully", keyRing.Name)
	}

	// Get KMS key, create if doesn't exist (and give GCS service account permission to use)
	reqGetKey := &kmspb.GetCryptoKeyRequest{
		Name: fmt.Sprintf("%s/cryptoKeys/%s", keyRing.Name, keyDetails["keyName"]),
	}
	var key *kmspb.CryptoKey
	key, err = c.GetCryptoKey(ctx, reqGetKey)
	if err != nil {
		if !strings.Contains(err.Error(), "NotFound") {
			// Handle unexpected error that isn't related to the key not being made yet
			t.Fatal(err)
		}
		// Create key that doesn't exist
		t.Logf("Cloud KMS key `%s` not found: creating key",
			fmt.Sprintf("%s/cryptoKeys/%s", keyRing.Name, keyDetails["keyName"]),
		)
		reqCreateKey := &kmspb.CreateCryptoKeyRequest{
			Parent:      keyRing.Name,
			CryptoKeyId: keyDetails["keyName"],
			CryptoKey: &kmspb.CryptoKey{
				Purpose: kmspb.CryptoKey_ENCRYPT_DECRYPT,
			},
		}
		key, err = c.CreateCryptoKey(ctx, reqCreateKey)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("Cloud KMS key `%s` created successfully", key.Name)
	}

	// Get GCS Service account email, check has necessary permission on key
	// Note: we cannot reuse the backend's storage client (like in the setupBackend function)
	// because the KMS key needs to exist before the backend buckets are made in the test.
	sc, err := storage.NewClient(ctx, opts...) //reuse opts from KMS client
	if err != nil {
		e := fmt.Errorf("storage.NewClient() failed: %v", err)
		t.Fatal(e)
	}
	defer sc.Close()
	gcsServiceAccount, err := sc.ServiceAccount(ctx, keyDetails["project"])
	if err != nil {
		t.Fatal(err)
	}

	// Assert Cloud Storage service account has permission to use this key.
	member := fmt.Sprintf("serviceAccount:%s", gcsServiceAccount)
	iamHandle := c.ResourceIAM(key.Name)
	policy, err := iamHandle.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ok := policy.HasRole(member, kmsRole); !ok {
		// Add the missing permissions
		t.Logf("Granting GCS service account %s %s role on key %s", gcsServiceAccount, kmsRole, key.Name)
		policy.Add(member, kmsRole)
		err = iamHandle.SetPolicy(ctx, policy)
		if err != nil {
			t.Fatal(err)
		}
	}
	return key.Name
}

// teardownBackend deletes all states from be except the default state.
func teardownBackend(t *testing.T, be backend.Backend, prefix string) {
	t.Helper()
	gcsBE, ok := be.(*Backend)
	if !ok {
		t.Fatalf("be is a %T, want a *gcsBackend", be)
	}
	ctx := context.Background()

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

// getClientOptions returns the []option.ClientOption needed to configure Google API clients
// that are required in acceptance tests but are not part of the gcs backend itself
func testGetClientOptions(t *testing.T) ([]option.ClientOption, error) {
	t.Helper()

	var creds string
	if v := os.Getenv("GOOGLE_BACKEND_CREDENTIALS"); v != "" {
		creds = v
	} else {
		creds = os.Getenv("GOOGLE_CREDENTIALS")
	}
	if creds == "" {
		t.Skip("This test required credentials to be supplied via" +
			"the GOOGLE_CREDENTIALS or GOOGLE_BACKEND_CREDENTIALS environment variables.")
	}

	var opts []option.ClientOption
	var credOptions []option.ClientOption

	contents, err := readPathOrContents(creds)
	if err != nil {
		return nil, fmt.Errorf("error loading credentials: %s", err)
	}
	if !json.Valid([]byte(contents)) {
		return nil, fmt.Errorf("the string provided in credentials is neither valid json nor a valid file path")
	}
	credOptions = append(credOptions, option.WithCredentialsJSON([]byte(contents)))
	opts = append(opts, credOptions...)
	opts = append(opts, option.WithUserAgent(httpclient.UserAgentString()))

	return opts, nil
}
