// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package oss

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore"
	"github.com/aliyun/credentials-go/credentials/providers"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
)

// verify that we are doing ACC tests or the OSS tests specifically
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_OSS_TEST") == ""
	if skip {
		t.Log("oss backend tests require setting TF_ACC or TF_OSS_TEST")
		t.Skip()
	}
	if skip {
		t.Fatal("oss backend tests require setting ALICLOUD_ACCESS_KEY or ALICLOUD_ACCESS_KEY_ID")
	}
	if os.Getenv("ALICLOUD_REGION") == "" {
		os.Setenv("ALICLOUD_REGION", "cn-beijing")
	}
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	testACC(t)
	config := map[string]interface{}{
		"region":              "cn-beijing",
		"bucket":              "terraform-backend-oss-test",
		"prefix":              "mystate",
		"key":                 "first.tfstate",
		"tablestore_endpoint": "https://terraformstate.cn-beijing.ots.aliyuncs.com",
		"tablestore_table":    "TableStore",
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

	if !strings.HasPrefix(b.ossClient.Config.Endpoint, "https://oss-cn-beijing") {
		t.Fatalf("Incorrect region was provided")
	}
	if b.bucketName != "terraform-backend-oss-test" {
		t.Fatalf("Incorrect bucketName was provided")
	}
	if b.statePrefix != "mystate" {
		t.Fatalf("Incorrect state file path was provided")
	}
	if b.stateKey != "first.tfstate" {
		t.Fatalf("Incorrect keyName was provided")
	}

	if b.ossClient.Config.AccessKeyID == "" {
		t.Fatalf("No Access Key Id was provided")
	}
	if b.ossClient.Config.AccessKeySecret == "" {
		t.Fatalf("No Secret Access Key was provided")
	}
}

func TestBackendConfigWorkSpace(t *testing.T) {
	testACC(t)
	bucketName := fmt.Sprintf("terraform-backend-oss-test-%d", rand.Intn(1000))
	config := map[string]interface{}{
		"region":              "cn-beijing",
		"bucket":              bucketName,
		"prefix":              "mystate",
		"key":                 "first.tfstate",
		"tablestore_endpoint": "https://terraformstate.cn-beijing.ots.aliyuncs.com",
		"tablestore_table":    "TableStore",
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)
	createOSSBucket(t, b.ossClient, bucketName)
	defer deleteOSSBucket(t, b.ossClient, bucketName)
	if _, diags := b.Workspaces(); diags.HasErrors() {
		t.Fatal(diags.Err().Error())
	}
	if !strings.HasPrefix(b.ossClient.Config.Endpoint, "https://oss-cn-beijing") {
		t.Fatalf("Incorrect region was provided")
	}
	if b.bucketName != bucketName {
		t.Fatalf("Incorrect bucketName was provided")
	}
	if b.statePrefix != "mystate" {
		t.Fatalf("Incorrect state file path was provided")
	}
	if b.stateKey != "first.tfstate" {
		t.Fatalf("Incorrect keyName was provided")
	}

	if b.ossClient.Config.AccessKeyID == "" {
		t.Fatalf("No Access Key Id was provided")
	}
	if b.ossClient.Config.AccessKeySecret == "" {
		t.Fatalf("No Secret Access Key was provided")
	}
}

func TestBackendConfigProfile(t *testing.T) {
	testACC(t)
	config := map[string]interface{}{
		"region":              "cn-beijing",
		"bucket":              "terraform-backend-oss-test",
		"prefix":              "mystate",
		"key":                 "first.tfstate",
		"tablestore_endpoint": "https://terraformstate.cn-beijing.ots.aliyuncs.com",
		"tablestore_table":    "TableStore",
		"profile":             "default",
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

	if !strings.HasPrefix(b.ossClient.Config.Endpoint, "https://oss-cn-beijing") {
		t.Fatalf("Incorrect region was provided")
	}
	if b.bucketName != "terraform-backend-oss-test" {
		t.Fatalf("Incorrect bucketName was provided")
	}
	if b.statePrefix != "mystate" {
		t.Fatalf("Incorrect state file path was provided")
	}
	if b.stateKey != "first.tfstate" {
		t.Fatalf("Incorrect keyName was provided")
	}

	if b.ossClient.Config.AccessKeyID == "" {
		t.Fatalf("No Access Key Id was provided")
	}
	if b.ossClient.Config.AccessKeySecret == "" {
		t.Fatalf("No Secret Access Key was provided")
	}
}

func TestGetConfigFromProfile_ignoresCachedCredentials(t *testing.T) {
	for _, mode := range []string{"CloudSSO", "OAuth", "External", "ChainableRamRoleArn"} {
		t.Run(mode, func(t *testing.T) {
			path := writeTestProfiles(t, fmt.Sprintf(`{"profiles": [{
				"name": "test", "mode": %q, "region_id": "cn-hangzhou",
				"access_key_id": "STS.cached", "access_key_secret": "cached", "sts_token": "cached"
			}]}`, mode))
			resetProfileCache(t)
			d := schema.TestResourceDataRaw(t, New().(*Backend).Schema, map[string]interface{}{
				"bucket":                  "terraform-backend-oss-test",
				"profile":                 "test",
				"shared_credentials_file": path,
			})

			for _, key := range []string{"access_key_id", "access_key_secret", "sts_token", "ram_role_arn", "expired_seconds"} {
				if got, err := getConfigFromProfile(d, key); err != nil || got != nil {
					t.Errorf("%s: want nil, got %#v (err: %v)", key, got, err)
				}
			}
			if got, _ := getConfigFromProfile(d, "region_id"); got != "cn-hangzhou" {
				t.Errorf("region_id: want %q, got %#v", "cn-hangzhou", got)
			}
		})
	}
}

func TestBackendConfigProfile_cloudSSOExpiredToken(t *testing.T) {
	clearCredentialEnv(t)
	path := writeTestProfiles(t, `{"profiles": [{
		"name": "sso", "mode": "CloudSSO",
		"cloud_sso_sign_in_url": "https://signin.example.com/login",
		"cloud_sso_account_id": "1234567890", "cloud_sso_access_config": "ac-test",
		"access_token": "expired", "cloud_sso_access_token_expire": 1,
		"access_key_id": "STS.cached", "access_key_secret": "cached", "sts_token": "cached", "sts_expiration": 1
	}]}`)
	resetProfileCache(t)

	b := New()
	cfg, diags := b.PrepareConfig(hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{
		"bucket":                  "terraform-backend-oss-test",
		"region":                  "cn-hangzhou",
		"profile":                 "sso",
		"shared_credentials_file": path,
	}))
	if diags.HasErrors() {
		t.Fatal(diags.Err())
	}

	diags = b.Configure(cfg)
	if !diags.HasErrors() {
		t.Fatal("expected an error for an expired CloudSSO access token, the cached STS credentials must not be used")
	}
	if got := diags.Err().Error(); !strings.Contains(got, "re-login") {
		t.Fatalf("want an error asking to re-login, got: %s", got)
	}
}

func TestOSSCredentialsProvider(t *testing.T) {
	p := &ossCredentialsProvider{provider: &fakeCredentialsProvider{responses: []fakeResponse{
		{creds: &providers.Credentials{AccessKeyId: "ak1", AccessKeySecret: "sk1", SecurityToken: "token1"}},
		{err: errors.New("refresh failed")},
		{creds: &providers.Credentials{AccessKeyId: "ak2", AccessKeySecret: "sk2", SecurityToken: "token2"}},
	}}}

	for i, want := range []ossCredentials{
		{"ak1", "sk1", "token1"},
		{"ak1", "sk1", "token1"}, // the failed refresh falls back to the last credentials
		{"ak2", "sk2", "token2"},
	} {
		if got := p.GetCredentials(); got != want {
			t.Fatalf("call %d: want %+v, got %+v", i, want, got)
		}
	}
}

type fakeCredentialsProvider struct {
	responses []fakeResponse
}

type fakeResponse struct {
	creds *providers.Credentials
	err   error
}

func (f *fakeCredentialsProvider) GetCredentials() (*providers.Credentials, error) {
	r := f.responses[0]
	f.responses = f.responses[1:]
	return r.creds, r.err
}

func (f *fakeCredentialsProvider) GetProviderName() string { return "fake" }

func writeTestProfiles(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// resetProfileCache clears the profile that getConfigFromProfile caches per process.
func resetProfileCache(t *testing.T) {
	t.Helper()
	providerConfig = nil
	t.Cleanup(func() { providerConfig = nil })
}

func clearCredentialEnv(t *testing.T) {
	t.Helper()
	for _, name := range []string{
		"ALICLOUD_ACCESS_KEY", "ALIBABA_CLOUD_ACCESS_KEY_ID", "ALICLOUD_ACCESS_KEY_ID",
		"ALICLOUD_SECRET_KEY", "ALIBABA_CLOUD_ACCESS_KEY_SECRET", "ALICLOUD_ACCESS_KEY_SECRET",
		"ALICLOUD_SECURITY_TOKEN", "ALIBABA_CLOUD_SECURITY_TOKEN",
		"ALICLOUD_ASSUME_ROLE_ARN", "ALIBABA_CLOUD_ROLE_ARN",
		"ALIBABA_CLOUD_CLI_PROFILE_DISABLED",
	} {
		t.Setenv(name, "")
	}
}

func TestBackendConfig_invalidKey(t *testing.T) {
	testACC(t)
	cfg := hcl2shim.HCL2ValueFromConfigValue(map[string]interface{}{
		"region":              "cn-beijing",
		"bucket":              "terraform-backend-oss-test",
		"prefix":              "/leading-slash",
		"name":                "/test.tfstate",
		"tablestore_endpoint": "https://terraformstate.cn-beijing.ots.aliyuncs.com",
		"tablestore_table":    "TableStore",
	})

	_, results := New().PrepareConfig(cfg)
	if !results.HasErrors() {
		t.Fatal("expected config validation error")
	}
}

func TestBackend(t *testing.T) {
	testACC(t)

	bucketName := fmt.Sprintf("terraform-remote-oss-test-%x", time.Now().Unix())
	statePrefix := "multi/level/path/"

	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket": bucketName,
		"prefix": statePrefix,
	})).(*Backend)

	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket": bucketName,
		"prefix": statePrefix,
	})).(*Backend)

	createOSSBucket(t, b1.ossClient, bucketName)
	defer deleteOSSBucket(t, b1.ossClient, bucketName)

	backend.TestBackendStates(t, b1)
	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)
}

func createOSSBucket(t *testing.T, ossClient *oss.Client, bucketName string) {
	// Be clear about what we're doing in case the user needs to clean this up later.
	if err := ossClient.CreateBucket(bucketName); err != nil {
		t.Fatal("failed to create test OSS bucket:", err)
	}
}

func deleteOSSBucket(t *testing.T, ossClient *oss.Client, bucketName string) {
	warning := "WARNING: Failed to delete the test OSS bucket. It may have been left in your Alibaba Cloud account and may incur storage charges. (error was %s)"

	// first we have to get rid of the env objects, or we can't delete the bucket
	bucket, err := ossClient.Bucket(bucketName)
	if err != nil {
		t.Fatal("Error getting bucket:", err)
		return
	}
	objects, err := bucket.ListObjects()
	if err != nil {
		t.Logf(warning, err)
		return
	}
	for _, obj := range objects.Objects {
		if err := bucket.DeleteObject(obj.Key); err != nil {
			// this will need cleanup no matter what, so just warn and exit
			t.Logf(warning, err)
			return
		}
	}

	if err := ossClient.DeleteBucket(bucketName); err != nil {
		t.Logf(warning, err)
	}
}

// create the tablestore table, and wait until we can query it.
func createTablestoreTable(t *testing.T, otsClient *tablestore.TableStoreClient, tableName string) {
	tableMeta := new(tablestore.TableMeta)
	tableMeta.TableName = tableName
	tableMeta.AddPrimaryKeyColumn(pkName, tablestore.PrimaryKeyType_STRING)

	tableOption := new(tablestore.TableOption)
	tableOption.TimeToAlive = -1
	tableOption.MaxVersion = 1

	reservedThroughput := new(tablestore.ReservedThroughput)

	_, err := otsClient.CreateTable(&tablestore.CreateTableRequest{
		TableMeta:          tableMeta,
		TableOption:        tableOption,
		ReservedThroughput: reservedThroughput,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func deleteTablestoreTable(t *testing.T, otsClient *tablestore.TableStoreClient, tableName string) {
	params := &tablestore.DeleteTableRequest{
		TableName: tableName,
	}
	_, err := otsClient.DeleteTable(params)
	if err != nil {
		t.Logf("WARNING: Failed to delete the test TableStore table %q. It has been left in your Alibaba Cloud account and may incur charges. (error was %s)", tableName, err)
	}
}

func TestGetHttpProxyUrl(t *testing.T) {
	tests := []struct {
		name             string
		rawUrl           string
		httpProxy        string
		httpsProxy       string
		noProxy          string
		expectedProxyURL string
	}{
		{
			name:             "should set proxy using http_proxy environment variable",
			rawUrl:           "http://example.com",
			httpProxy:        "http://foo.bar:3128",
			httpsProxy:       "https://secure.example.com",
			noProxy:          "",
			expectedProxyURL: "http://foo.bar:3128",
		},
		{
			name:             "should set proxy using http_proxy environment variable",
			rawUrl:           "http://example.com",
			httpProxy:        "http://foo.barr",
			httpsProxy:       "https://secure.example.com",
			noProxy:          "",
			expectedProxyURL: "http://foo.barr",
		},
		{
			name:             "should set proxy using https_proxy environment variable",
			rawUrl:           "https://secure.example.com",
			httpProxy:        "http://foo.bar",
			httpsProxy:       "https://foo.bar.com:3128",
			noProxy:          "",
			expectedProxyURL: "https://foo.bar.com:3128",
		},
		{
			name:             "should set proxy using https_proxy environment variable",
			rawUrl:           "https://secure.example.com",
			httpProxy:        "",
			httpsProxy:       "http://foo.baz",
			noProxy:          "",
			expectedProxyURL: "http://foo.baz",
		},
		{
			name:             "should not set http proxy if NO_PROXY contains the host",
			rawUrl:           "http://example.internal",
			httpProxy:        "http://foo.bar:3128",
			httpsProxy:       "",
			noProxy:          "example.internal",
			expectedProxyURL: "",
		},
		{
			name:             "should not set HTTP proxy when NO_PROXY matches the domain with suffix",
			rawUrl:           "http://qqu.example.internal",
			httpProxy:        "http://foo.bar:3128",
			httpsProxy:       "",
			noProxy:          ".example.internal",
			expectedProxyURL: "",
		},
		{
			name:             "should not set https proxy if NO_PROXY contains the host",
			rawUrl:           "https://secure.internal",
			httpProxy:        "",
			httpsProxy:       "https://foo.baz:3128",
			noProxy:          "secure.internal",
			expectedProxyURL: "",
		},
		{
			name:             "should not set https proxy if NO_PROXY matches the domain with suffix",
			rawUrl:           "https://ss.qcsc.secure.internal",
			httpProxy:        "",
			httpsProxy:       "https://foo.baz:3128",
			noProxy:          ".secure.internal",
			expectedProxyURL: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			t.Setenv("HTTP_PROXY", tt.httpProxy)
			t.Setenv("HTTPS_PROXY", tt.httpsProxy)
			t.Setenv("NO_PROXY", tt.noProxy)

			proxyUrl, err := getHttpProxyUrl(tt.rawUrl)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if tt.expectedProxyURL == "" {
				if proxyUrl != nil {
					t.Fatalf("unexpected proxy  URL, want nil, got: %s", proxyUrl)
				}
			} else {
				if tt.expectedProxyURL != proxyUrl.String() {
					t.Fatalf("unexpected proxy URL, want: %s, got: %s", tt.expectedProxyURL, proxyUrl.String())
				}
			}
		})
	}
}
