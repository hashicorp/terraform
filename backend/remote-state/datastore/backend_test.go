package datastore

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/hashicorp/terraform/backend"
)

const (
	defaultTestNamespace = "tf_acc_test"

	envTestAcc         = "TF_ACC"
	envTestDS          = "TF_DATASTORE_TEST"
	envTestDSProject   = "TF_DATASTORE_TEST_PROJECT"
	envTestDSNamespace = "TF_DATASTORE_TEST_NAMESPACE"
	envTestDSCredsFile = "TF_DATASTORE_TEST_CREDS_FILE"
)

const creds = `{
  "type": "service_account",
  "project_id": "REDACTED",
  "private_key_id": "6cb82c286bcc30f24fc5ff71e12bfd4f75270d52",
  "private_key": "REDACTED",
  "client_email": "REDACTED@REDACTED.iam.gserviceaccount.com",
  "client_id": "106666355455433859711",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://accounts.google.com/o/oauth2/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/REDACTED%40REDACTED.iam.gserviceaccount.com"
}`

// Verify that we are doing either acceptance tests or the Datastore tests.
func testACC(t *testing.T) {
	t.Helper()

	skip := os.Getenv(envTestAcc) == "" && os.Getenv(envTestDS) == ""
	if skip {
		t.Logf("Datastore backend tests require setting %v or %v", envTestAcc, envTestDS)
		t.Skip()
	}
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = &Backend{}
}

func credsFile() (string, error) {
	f, err := ioutil.TempFile("", "creds")
	if err != nil {
		return "", err
	}
	if _, err := f.Write([]byte(creds)); err != nil {
		return "", err
	}
	f.Close()
	return f.Name(), nil
}

func TestBackendConfig(t *testing.T) {
	// This test just instantiates the client. Shouldn't make any actual
	// requests nor incur any costs.

	creds, err := credsFile()
	if err != nil {
		t.Fatalf("cannot create temporary credentials file: %v", err)
	}
	defer func() {
		if err := os.Remove(creds); err != nil {
			t.Fatalf("cannot remove temporary credentials file: %v", err)
		}

	}()

	cases := []struct {
		name   string
		config map[string]interface{}
	}{
		// Note all test cases must specify a credentials_file so that we can
		// instantiate a Datastore client in environments (like Travis CI) that
		// don't have Application Default Credentials setup.
		{
			name: "ProjectOnly",
			config: map[string]interface{}{
				"project":          "tfproject",
				"credentials_file": creds,
			},
		},
		{
			name: "ProjectAndNamespace",
			config: map[string]interface{}{
				"project":          "tfproject",
				"namespace":        "tfnamespace",
				"credentials_file": creds,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := backend.TestBackendConfig(t, New(), tc.config).(*Backend)

			if _, ok := tc.config["namespace"]; !ok {
				return
			}
			want := tc.config["namespace"].(string)
			if b.ns != want {
				t.Fatalf("b.ns: got %v, want %v", b.ns, want)
			}
		})
	}
}

func configFromEnv(t *testing.T) map[string]interface{} {
	t.Helper()
	p := os.Getenv(envTestDSProject)
	if p == "" {
		t.Fatalf("Datastore backend tests require setting %v", envTestDSProject)
	}
	c := map[string]interface{}{
		"project":   p,
		"namespace": defaultTestNamespace,
	}
	if ns, ok := os.LookupEnv(envTestDSNamespace); ok {
		c["namespace"] = ns
	}
	if cf, ok := os.LookupEnv(envTestDSCredsFile); ok {
		c["credentials_file"] = cf
	}
	return c
}

func cleanupTestNamespace(t *testing.T, cfg map[string]interface{}) {
	t.Helper()

	ctx := context.Background()
	o := []option.ClientOption{}
	if f, ok := cfg["credentials_file"]; ok {
		o = []option.ClientOption{option.WithCredentialsFile(f.(string))}
	}
	p := cfg["project"].(string)
	ns := cfg["namespace"].(string)

	if ns == "" {
		// Be paranoid and avoid cleaning the default namespace in case
		// we delete entites that weren't created by our test harness.
		t.Logf("refusing to cleanup default namespace of project %v - please cleanup manually", p)
		return
	}

	ds, err := datastore.NewClient(ctx, p, o...)
	if err != nil {
		t.Fatalf("cannot initialise Google Cloud Datastore client to cleanup namespace %v in project %v: %v", ns, p, err)
	}

	cleanup := func(kind string, entity interface{}) {
		for i := ds.Run(ctx, datastore.NewQuery(kind).Namespace(ns).KeysOnly()); ; {
			k, err := i.Next(entity)
			if err == iterator.Done {
				return
			}
			if err != nil {
				t.Fatalf("cannot query entities to delete from Google Datastore: %v", err)
			}
			if err := ds.Delete(ctx, k); err != nil {
				t.Fatalf("cannot delete entity %v from Google Datastore: %v", k, err)
			}
		}
	}

	cleanup(kindTerraformStateLock, &entityLock{})
	cleanup(kindTerraformState, &entityState{})
}

func TestBackend(t *testing.T) {
	testACC(t)
	config := configFromEnv(t)
	defer cleanupTestNamespace(t, config)

	b1 := backend.TestBackendConfig(t, New(), config).(*Backend)
	b2 := backend.TestBackendConfig(t, New(), config).(*Backend)
	backend.TestBackend(t, b1, b2)
}
