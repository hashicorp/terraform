package aws

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

var testAccProviders map[string]terraform.ResourceProvider
var testAccProvider *schema.Provider

func init() {
	testAccProvider = Provider().(*schema.Provider)
	testAccProviders = map[string]terraform.ResourceProvider{
		"aws": testAccProvider,
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().(*schema.Provider).InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ terraform.ResourceProvider = Provider()
}

func prepareFakeCredentialFile(access_key_id string, secret_key string) (*os.File, error) {
	credentialFile, err := ioutil.TempFile("", "aws_credential_test")

	if err != nil {
		return nil, err
	}

	contents := fmt.Sprintf(`[default]
aws_access_key_id = %s
aws_secret_access_key = %s
`, access_key_id, secret_key)

	credentialFile.Write([]byte(fmt.Sprintf(contents)))
	credentialFile.Close()

	return credentialFile, nil

}

func cleanAwsEnvConfig() func() {
	oldAccessKey := os.Getenv("AWS_ACCESS_KEY")
	oldSecretKey := os.Getenv("AWS_SECRET_KEY")
	oldAccessKeyId := os.Getenv("AWS_ACCESS_KEY_ID")
	oldSecretAccessKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	os.Setenv("AWS_ACCESS_KEY", "")
	os.Setenv("AWS_SECRET_KEY", "")
	os.Setenv("AWS_ACCESS_KEY_ID", "")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "")

	return func() {
		os.Setenv("AWS_ACCESS_KEY", oldAccessKey)
		os.Setenv("AWS_SECRET_KEY", oldSecretKey)
		os.Setenv("AWS_ACCESS_KEY_ID", oldAccessKeyId)
		os.Setenv("AWS_SECRET_ACCESS_KEY", oldSecretAccessKey)
	}
}

func TestEnvVarsOverrideCredentialsFile(t *testing.T) {
	resetEnvVars := cleanAwsEnvConfig()
	defer resetEnvVars()

	os.Setenv("AWS_ACCESS_KEY_ID", "access_key_id_from_aws_cli_env_var")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "secret_access_key_from_aws_cli_env_var")

	credentialFile, err := prepareFakeCredentialFile("access_key_id_from_config", "secret_access_key_from_config")

	if err != nil {
		t.Fatalf("Could not create temporary AWS config file: '%s'", err)
	}

	os.Setenv("AWS_CREDENTIAL_FILE", credentialFile.Name())
	defer os.Remove(credentialFile.Name())

	auth := awsAuthSource{}

	access_key, _ := auth.accessKeyResolver()()

	if access_key != "access_key_id_from_aws_cli_env_var" {
		t.Errorf("expected: %s, got: %#v", "access_key_id_from_aws_cli_env_var", access_key)
	}

	secret_key, _ := auth.secretKeyResolver()()

	if secret_key != "secret_access_key_from_aws_cli_env_var" {
		t.Errorf("expected: %#v, got: %#v", "secret_access_key_from_aws_cli_env_var", secret_key)
	}
}

// See:
// - https://github.com/hashicorp/terraform/pull/851
// - https://github.com/hashicorp/terraform/issues/866
//
// We should not change default behaviour in a minor release, if end-user has both variations
// of the env key set in their environment then we should give preference to the legacy one.
func TestDeprecatedEnvVarsOverrideOfficialOnes(t *testing.T) {
	resetEnvVars := cleanAwsEnvConfig()
	defer resetEnvVars()

	os.Setenv("AWS_ACCESS_KEY", "access_key_id_from_legacy_env_var")
	os.Setenv("AWS_SECRET_KEY", "secret_access_key_from_legacy_env_var")
	os.Setenv("AWS_ACCESS_KEY_ID", "access_key_id_from_aws_cli_env_var")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "secret_access_key_from_aws_cli_env_var")

	auth := awsAuthSource{}

	access_key, _ := auth.accessKeyResolver()()

	if access_key != "access_key_id_from_legacy_env_var" {
		t.Errorf("expected: %s, got: %#v", "access_key_id_from_legacy_env_var", access_key)
	}

	secret_key, _ := auth.secretKeyResolver()()

	if secret_key != "secret_access_key_from_legacy_env_var" {
		t.Errorf("expected: %s, got: %#v", "secret_access_key_from_legacy_env_var", secret_key)
	}
}

func TestLoadCredentialsFromFileWhenNoConfigInEnv(t *testing.T) {
	resetEnvVars := cleanAwsEnvConfig()
	defer resetEnvVars()

	credentialFile, err := prepareFakeCredentialFile("access_key_id_from_config", "secret_access_key_from_config")

	if err != nil {
		t.Fatalf("Could not create temporary AWS config file: '%s'", err)
	}

	os.Setenv("AWS_CREDENTIAL_FILE", credentialFile.Name())
	defer os.Remove(credentialFile.Name())

	auth := awsAuthSource{}

	access_key, _ := auth.accessKeyResolver()()

	if access_key != "access_key_id_from_config" {
		t.Errorf("expected: %s, got: %#v", "access_key_id_from_config", access_key)
	}

	secret_key, _ := auth.secretKeyResolver()()

	if secret_key != "secret_access_key_from_config" {
		t.Errorf("expected: %s, bad: %#v", "secret_access_key_from_config", secret_key)
	}
}

func TestAuthSourcerReturnsNilWhenDefaultsCannotBeFound(t *testing.T) {
	resetEnvVars := cleanAwsEnvConfig()
	defer resetEnvVars()

	auth := awsAuthSource{}

	access_key, _ := auth.accessKeyResolver()()

	if access_key != nil {
		t.Errorf("expected: nil, got: %#v", access_key)
	}

	secret_key, _ := auth.secretKeyResolver()()

	if secret_key != nil {
		t.Errorf("expected: nil, got: %#v", secret_key)
	}
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("AWS_ACCESS_KEY_ID"); v == "" {
		t.Fatal("AWS_ACCESS_KEY_ID must be set for acceptance tests")
	}
	if v := os.Getenv("AWS_SECRET_ACCESS_KEY"); v == "" {
		t.Fatal("AWS_SECRET_ACCESS_KEY must be set for acceptance tests")
	}
	if v := os.Getenv("AWS_DEFAULT_REGION"); v == "" {
		log.Println("[INFO] Test: Using us-west-2 as test region")
		os.Setenv("AWS_DEFAULT_REGION", "us-west-2")
	}
}
