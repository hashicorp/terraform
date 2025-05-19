// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oss

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/aliyun/aliyun-tablestore-go-sdk/tablestore"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/hcl2shim"
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
	if _, err := b.Workspaces(); err != nil {
		t.Fatal(err.Error())
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
