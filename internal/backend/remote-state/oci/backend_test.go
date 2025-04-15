package oci

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
)

func TestBackendBasic(t *testing.T) {
	testACC(t)

	ctx := context.Background()

	bucketName := fmt.Sprintf("terraform-remote-oci-test-%x", time.Now().Unix())
	keyName := "testState.json"
	namespace := getEnvSettingWithBlankDefault(NamespaceAttrName)
	compartmentId := getEnvSettingWithBlankDefault("compartment_id")

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":    bucketName,
		"key":       keyName,
		"namespace": namespace,
	})).(*Backend)

	response := createOCIBucket(ctx, t, b.client.objectStorageClient, bucketName, namespace, compartmentId)
	defer deleteOCIBucket(ctx, t, b.client.objectStorageClient, bucketName, *response.ETag, namespace)

	backend.TestBackendStates(t, b)
}
func TestBackendLocked_FolceUnclock(t *testing.T) {
	testACC(t)
	ctx := context.Background()
	bucketName := fmt.Sprintf("terraform-remote-oci-test-%x", time.Now().Unix())
	keyName := "testState.json"
	namespace := getEnvSettingWithBlankDefault(NamespaceAttrName)
	compartmentId := getEnvSettingWithBlankDefault("compartment_id")
	b1 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":    bucketName,
		"key":       keyName,
		"namespace": namespace,
	})).(*Backend)
	b2 := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":    bucketName,
		"key":       keyName,
		"namespace": namespace,
	})).(*Backend)
	response := createOCIBucket(ctx, t, b1.client.objectStorageClient, bucketName, namespace, compartmentId)
	defer deleteOCIBucket(ctx, t, b1.client.objectStorageClient, bucketName, *response.ETag, namespace)
	// Test state locking and force-unlock
	backend.TestBackendStateLocks(t, b1, b2)
	backend.TestBackendStateForceUnlock(t, b1, b2)
}
func TestBackendBasic_multipart_Upload(t *testing.T) {
	testACC(t)

	ctx := context.Background()
	DefaultFilePartSize = 100 //	100 Bytes
	bucketName := fmt.Sprintf("terraform-remote-oci-test-%x", time.Now().Unix())
	keyName := "testState.json"
	namespace := getEnvSettingWithBlankDefault(NamespaceAttrName)
	compartmentId := getEnvSettingWithBlankDefault("compartment_id")

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"bucket":    bucketName,
		"key":       keyName,
		"namespace": namespace,
	})).(*Backend)

	response := createOCIBucket(ctx, t, b.client.objectStorageClient, bucketName, namespace, compartmentId)
	defer deleteOCIBucket(ctx, t, b.client.objectStorageClient, bucketName, *response.ETag, namespace)

	backend.TestBackendStates(t, b)
}

// Helper functions to create and delete OCI bucket
func createOCIBucket(ctx context.Context, t *testing.T, client *objectstorage.ObjectStorageClient, bucketName, namespace, compartmentId string) objectstorage.CreateBucketResponse {
	req := objectstorage.CreateBucketRequest{
		NamespaceName: common.String(namespace),
		CreateBucketDetails: objectstorage.CreateBucketDetails{
			Name:          common.String(bucketName),
			CompartmentId: common.String(compartmentId),
			Versioning:    objectstorage.CreateBucketDetailsVersioningEnabled,
		},
	}

	response, err := client.CreateBucket(ctx, req)
	if err != nil {
		t.Fatalf("failed to create OCI bucket: %v", err)
	}
	return response
}

func deleteOCIBucket(ctx context.Context, t *testing.T, client *objectstorage.ObjectStorageClient, bucketName, etag, namespace string) {
	request := objectstorage.ListObjectVersionsRequest{
		BucketName:    common.String(bucketName),
		NamespaceName: common.String(namespace),
		Prefix:        common.String(""),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(),
		},
	}

	response, err := client.ListObjectVersions(context.Background(), request)
	if err != nil {
		t.Fatalf("failed to list(First page) OCI bucket objects: %v", err)
	}

	request.Page = response.OpcNextPage

	for request.Page != nil {
		request.RequestMetadata.RetryPolicy = getDefaultRetryPolicy()

		listResponse, err := client.ListObjectVersions(context.Background(), request)
		if err != nil {
			t.Fatalf("failed to list OCI bucket objects: %v", err)
		}
		response.Items = append(response.Items, listResponse.Items...)
		request.Page = listResponse.OpcNextPage
	}

	var diagErr tfdiags.Diagnostics

	for _, objectVersion := range response.Items {

		deleteObjectVersionRequest := objectstorage.DeleteObjectRequest{
			BucketName:    common.String(bucketName),
			NamespaceName: common.String(namespace),
			ObjectName:    objectVersion.Name,
			VersionId:     objectVersion.VersionId,
			RequestMetadata: common.RequestMetadata{
				RetryPolicy: getDefaultRetryPolicy(),
			},
		}

		_, err := client.DeleteObject(context.Background(), deleteObjectVersionRequest)
		if err != nil {
			diagErr = diagErr.Append(err)
		}
	}
	if diagErr != nil {
		t.Fatalf("error while deleting object from bucket: %v", diagErr.Err())
	}

	req := objectstorage.DeleteBucketRequest{
		NamespaceName: common.String(namespace),
		BucketName:    common.String(bucketName),
		IfMatch:       common.String(etag),
	}

	_, err = client.DeleteBucket(ctx, req)
	if err != nil {
		t.Fatalf("failed to delete OCI bucket: %v", err)
	}
}

// verify that we are doing ACC tests or the S3 tests specifically
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_OCI_BACKEND_TEST") == ""
	if skip {
		t.Log("oci backend tests require setting TF_ACC or TF_OCI_BACKEND_TEST")
		t.Skip()
	}
}
