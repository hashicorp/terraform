// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"

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
func TestBackendLocked_ForceUnlock(t *testing.T) {
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
	backend.TestBackendStateLocksInWS(t, b1, b2, "testenv")
	backend.TestBackendStateForceUnlock(t, b1, b2)
	backend.TestBackendStateForceUnlockInWS(t, b1, b2, "testenv")
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

// verify that we are doing ACC tests or the oci backend tests specifically
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_OCI_BACKEND_TEST") == ""
	if skip {
		t.Log("oci backend tests require setting TF_ACC or TF_OCI_BACKEND_TEST")
		t.Skip()
	}
}

func TestOCIBackendConfig_PrepareConfigValidation(t *testing.T) {
	cases := map[string]struct {
		config        cty.Value
		expectedDiags tfdiags.Diagnostics
		mock          func()
	}{
		"null bucket": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:    cty.NullVal(cty.String),
				NamespaceAttrName: cty.StringVal("test-namespace"),
				KeyAttrName:       cty.StringVal("test-key"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				requiredAttributeErrDiag(cty.GetAttrPath(BucketAttrName)),
			},
		},
		"empty bucket": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:    cty.StringVal(""),
				NamespaceAttrName: cty.StringVal("test-namespace"),
				KeyAttrName:       cty.StringVal("test-key"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				requiredAttributeErrDiag(cty.GetAttrPath(BucketAttrName)),
			},
		},
		"null namespace": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:    cty.StringVal("test-bucket"),
				NamespaceAttrName: cty.NullVal(cty.String),
				KeyAttrName:       cty.StringVal("test-key"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				requiredAttributeErrDiag(cty.GetAttrPath("namespace")),
			},
		},
		"empty namespace": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:    cty.StringVal("test-bucket"),
				NamespaceAttrName: cty.StringVal(""),
				KeyAttrName:       cty.StringVal("test-key"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				requiredAttributeErrDiag(cty.GetAttrPath("namespace")),
			},
		},
		"key with leading slash": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:    cty.StringVal("test-bucket"),
				NamespaceAttrName: cty.StringVal("test-namespace"),
				KeyAttrName:       cty.StringVal("/leading-slash"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`The value must not start or end with "/" and also not contain consecutive "/"`,
					cty.GetAttrPath(KeyAttrName),
				),
			},
		},
		"key with trailing slash": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:    cty.StringVal("test-bucket"),
				NamespaceAttrName: cty.StringVal("test-namespace"),
				KeyAttrName:       cty.StringVal("trailing-slash/"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`The value must not start or end with "/" and also not contain consecutive "/"`,
					cty.GetAttrPath(KeyAttrName),
				),
			},
		},
		"key with double slash": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:    cty.StringVal("test-bucket"),
				NamespaceAttrName: cty.StringVal("test-namespace"),
				KeyAttrName:       cty.StringVal("test/with/double//slash"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`The value must not start or end with "/" and also not contain consecutive "/"`,
					cty.GetAttrPath(KeyAttrName),
				),
			},
		},
		"workspace_key_prefix with leading slash": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:             cty.StringVal("test-bucket"),
				NamespaceAttrName:          cty.StringVal("test-namespace"),
				KeyAttrName:                cty.StringVal("test-key"),
				WorkspaceKeyPrefixAttrName: cty.StringVal("/env"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Value",
					`The value must not start  with "/" and also not contain consecutive "/"`,
					cty.GetAttrPath(WorkspaceKeyPrefixAttrName),
				),
			},
		},
		"encryption key conflict": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:               cty.StringVal("test-bucket"),
				NamespaceAttrName:            cty.StringVal("test-namespace"),
				KeyAttrName:                  cty.StringVal("test-key"),
				KmsKeyIdAttrName:             cty.StringVal("ocid1.key.oc1..example"),
				SseCustomerKeyAttrName:       cty.StringVal("base64-encoded-key"),
				SseCustomerKeySHA256AttrName: cty.StringVal("base64-encoded-key md5"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Attribute Combination",
					`Only one of kms_key_id, sse_customer_key can be set.`,
					cty.GetAttrPath(KmsKeyIdAttrName),
				),
			},
		},
		"Invalid encryption key combination": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:         cty.StringVal("test-bucket"),
				NamespaceAttrName:      cty.StringVal("test-namespace"),
				KeyAttrName:            cty.StringVal("test-key"),
				SseCustomerKeyAttrName: cty.StringVal("base64-encoded-key"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Attribute Combination",
					`  sse_customer_key and its SHA both required.`,
					cty.GetAttrPath(SseCustomerKeySHA256AttrName)),
			},
		},
		"private_key and private_key_path conflict": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:         cty.StringVal("test-bucket"),
				NamespaceAttrName:      cty.StringVal("test-namespace"),
				KeyAttrName:            cty.StringVal("test-key"),
				PrivateKeyAttrName:     cty.StringVal("-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----"),
				PrivateKeyPathAttrName: cty.StringVal("/path/to/key.pem"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				attributeErrDiag(
					"Invalid Attribute Combination",
					`Only one of private_key, private_key_path can be set.`,
					cty.GetAttrPath(PrivateKeyPathAttrName),
				),
			},
		},
		"invalid auth method": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:    cty.StringVal("test-bucket"),
				NamespaceAttrName: cty.StringVal("test-namespace"),
				KeyAttrName:       cty.StringVal("test-key"),
				AuthAttrName:      cty.StringVal("invalid-auth"),
			}),
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.AttributeValue(tfdiags.Error,
					"Invalid authentication method",
					fmt.Sprintf("auth must be one of '%s' or '%s' or '%s' or '%s' or '%s' or '%s'", AuthAPIKeySetting, AuthInstancePrincipalSetting, AuthInstancePrincipalWithCertsSetting, AuthSecurityToken, ResourcePrincipal, AuthOKEWorkloadIdentity), cty.GetAttrPath(AuthAttrName),
				),
			},
		},
		"missing region for InstancePrinciple auth": {
			config: cty.ObjectVal(map[string]cty.Value{
				BucketAttrName:    cty.StringVal("test-bucket"),
				NamespaceAttrName: cty.StringVal("test-namespace"),
				KeyAttrName:       cty.StringVal("test-key"),
				AuthAttrName:      cty.StringVal(AuthInstancePrincipalSetting),
			}),
			expectedDiags: tfdiags.Diagnostics{
				tfdiags.AttributeValue(tfdiags.Error,
					"Missing region attribute required",
					fmt.Sprintf("The attribute %q is required by the backend for %s authentication.\n\n", RegionAttrName, AuthInstancePrincipalSetting), cty.GetAttrPath(RegionAttrName),
				),
			},
			mock: func() {
				os.Setenv("OCI_region", "")
				os.Setenv("region", "")
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {

			b := New()
			if tc.mock != nil {
				tc.mock()
			}
			_, valDiags := b.PrepareConfig(populateSchema(t, b.ConfigSchema(), tc.config))

			if diff := cmp.Diff(valDiags, tc.expectedDiags, tfdiags.DiagnosticComparer); diff != "" {
				t.Errorf("unexpected diagnostics difference: %s", diff)
			}
		})
	}

}

func populateSchema(t *testing.T, schema *configschema.Block, value cty.Value) cty.Value {
	ty := schema.ImpliedType()
	var path cty.Path
	val, err := unmarshal(value, ty, path)
	if err != nil {
		t.Fatalf("populating schema: %s", err)
	}
	return val
}

func unmarshal(value cty.Value, ty cty.Type, path cty.Path) (cty.Value, error) {
	switch {
	case ty.IsPrimitiveType():
		return value, nil
	// case ty.IsListType():
	// 	return unmarshalList(value, ty.ElementType(), path)
	case ty.IsSetType():
		return unmarshalSet(value, ty.ElementType(), path)
	case ty.IsMapType():
		return unmarshalMap(value, ty.ElementType(), path)
	// case ty.IsTupleType():
	// 	return unmarshalTuple(value, ty.TupleElementTypes(), path)
	case ty.IsObjectType():
		return unmarshalObject(value, ty.AttributeTypes(), path)
	default:
		return cty.NilVal, path.NewErrorf("unsupported type %s", ty.FriendlyName())
	}
}

func unmarshalSet(dec cty.Value, ety cty.Type, path cty.Path) (cty.Value, error) {
	if dec.IsNull() {
		return dec, nil
	}

	length := dec.LengthInt()

	if length == 0 {
		return cty.SetValEmpty(ety), nil
	}

	vals := make([]cty.Value, 0, length)
	dec.ForEachElement(func(key, val cty.Value) (stop bool) {
		vals = append(vals, val)
		return
	})

	return cty.SetVal(vals), nil
}
func unmarshalMap(dec cty.Value, ety cty.Type, path cty.Path) (cty.Value, error) {
	if dec.IsNull() {
		return dec, nil
	}

	length := dec.LengthInt()

	if length == 0 {
		return cty.MapValEmpty(ety), nil
	}

	vals := make(map[string]cty.Value, length)
	dec.ForEachElement(func(key, val cty.Value) (stop bool) {
		vals[key.AsString()] = val
		return
	})

	return cty.MapVal(vals), nil
}

func unmarshalObject(dec cty.Value, atys map[string]cty.Type, path cty.Path) (cty.Value, error) {
	if dec.IsNull() {
		return dec, nil
	}
	valueTy := dec.Type()

	vals := make(map[string]cty.Value, len(atys))
	path = append(path, nil)
	for key, aty := range atys {
		path[len(path)-1] = cty.IndexStep{
			Key: cty.StringVal(key),
		}

		if !valueTy.HasAttribute(key) {
			vals[key] = cty.NullVal(aty)
		} else {
			val, err := unmarshal(dec.GetAttr(key), aty, path)
			if err != nil {
				return cty.DynamicVal, err
			}
			vals[key] = val
		}
	}

	return cty.ObjectVal(vals), nil
}
