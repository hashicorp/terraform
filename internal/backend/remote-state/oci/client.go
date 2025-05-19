// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
)

type RemoteClient struct {
	objectStorageClient  *objectstorage.ObjectStorageClient
	namespace            string
	bucketName           string
	path                 string
	lockFilePath         string
	kmsKeyID             string
	SSECustomerKey       string
	SSECustomerKeySHA256 string
	SSECustomerAlgorithm string
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	logger := logWithOperation("download-state-file").Named(c.path)
	logger.Info("Downloading remote state")
	ctx := context.WithValue(context.Background(), "logger", logger)
	payload, err := c.getObject(ctx)
	if err != nil || len(payload.Data) == 0 {
		return nil, err
	}
	// md5 hash of whole state
	sum := md5.Sum(payload.Data)
	payload.MD5 = sum[:]
	return payload, nil
}

func (c *RemoteClient) getObject(ctx context.Context) (*remote.Payload, error) {
	logger := ctx.Value("logger").(hclog.Logger)
	headRequest := objectstorage.HeadObjectRequest{
		NamespaceName: common.String(c.namespace),
		ObjectName:    common.String(c.path),
		BucketName:    common.String(c.bucketName),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(),
		},
	}
	if c.SSECustomerKey != "" && c.SSECustomerKeySHA256 != "" {
		headRequest.OpcSseCustomerKey = common.String(c.SSECustomerKey)
		headRequest.OpcSseCustomerKeySha256 = common.String(c.SSECustomerKeySHA256)
		headRequest.OpcSseCustomerAlgorithm = common.String(c.SSECustomerAlgorithm)
	}
	// Get object from OCI
	headResponse, headErr := c.objectStorageClient.HeadObject(ctx, headRequest)
	if headErr != nil {
		var ociHeadErr common.ServiceError
		if errors.As(headErr, &ociHeadErr) && ociHeadErr.GetHTTPStatusCode() == 404 {
			logger.Debug(" State file '%s' not found. Initializing Terraform state...", c.path)
			return &remote.Payload{}, nil
		} else {
			return nil, fmt.Errorf("failed to access object '%s' in bucket '%s': %w", c.path, c.bucketName, headErr)
		}
	}

	getRequest := objectstorage.GetObjectRequest{
		NamespaceName: common.String(c.namespace),
		ObjectName:    common.String(c.path),
		BucketName:    common.String(c.bucketName),
		IfMatch:       headResponse.ETag,
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(),
		},
	}
	if c.SSECustomerKey != "" && c.SSECustomerKeySHA256 != "" {
		getRequest.OpcSseCustomerKey = common.String(c.SSECustomerKey)
		getRequest.OpcSseCustomerKeySha256 = common.String(c.SSECustomerKeySHA256)
		getRequest.OpcSseCustomerAlgorithm = common.String(c.SSECustomerAlgorithm)
	}
	// Get object from OCI
	getResponse, err := c.objectStorageClient.GetObject(ctx, getRequest)
	if err != nil {
		var ociErr common.ServiceError
		if errors.As(err, &ociErr) {
			return nil, fmt.Errorf("failed to access object HttpStatusCode: %d\nOpcRequestId: %s\n message: %s\n ErrorCode: %s", ociErr.GetHTTPStatusCode(), ociErr.GetOpcRequestID(), ociErr.GetMessage(), ociErr.GetCode())

		}
		return nil, fmt.Errorf("failed to access object '%s' in bucket '%s': %w", c.path, c.bucketName, err)
	}
	defer getResponse.Content.Close()

	// Read object content
	contentArray, err := io.ReadAll(getResponse.Content)
	if err != nil {
		return nil, fmt.Errorf("unable to read 'content' from response: %w", err)
	}

	// Compute MD5 hash
	md5Hash := getResponse.ContentMd5
	if md5Hash == nil || len(*md5Hash) == 0 {
		md5Hash = getResponse.OpcMultipartMd5
	}
	// Construct payload
	payload := &remote.Payload{
		Data: contentArray,
		MD5:  []byte(*md5Hash),
	}

	// Return an error instead of `nil, nil` if the object is empty
	if len(payload.Data) == 0 {
		return nil, fmt.Errorf("object %q is empty", c.path)
	}

	return payload, nil
}

func (c *RemoteClient) Put(data []byte) error {
	logger := logWithOperation("upload-state-file").Named(c.path)
	ctx := context.WithValue(context.Background(), "logger", logger)
	dataSize := int64(len(data))
	sum := md5.Sum(data)
	var err error
	if dataSize > DefaultFilePartSize {
		logger.Info("Using Multipart Feature")
		var multipartUploadData = MultipartUploadData{
			client: c,
			Data:   data,
			RequestMetadata: common.RequestMetadata{
				RetryPolicy: getDefaultRetryPolicy(),
			},
		}
		err = multipartUploadData.multiPartUploadImpl(ctx)
		if err != nil && dataSize <= MaxFilePartSize {
			logger.Error(fmt.Sprintf("Multipart upload failed, falling back to single part upload: %v", err))
			err = c.uploadSinglePartObject(ctx, data, sum[:])
		}
	} else {
		err = c.uploadSinglePartObject(ctx, data, sum[:])
	}
	if err != nil {
		return err
	}

	return nil
}

func (c *RemoteClient) uploadSinglePartObject(ctx context.Context, data, sum []byte) error {
	logger := ctx.Value("logger").(hclog.Logger).Named("singlePartUpload")
	logger.Info("Uploading single part object")
	if len(data) == 0 {
		return fmt.Errorf("uploadSinglePartObject: data is empty")
	}

	contentType := "application/json"

	putRequest := objectstorage.PutObjectRequest{
		ContentType:   common.String(contentType),
		NamespaceName: common.String(c.namespace),
		ObjectName:    common.String(c.path),
		BucketName:    common.String(c.bucketName),
		PutObjectBody: io.NopCloser(bytes.NewReader(data)),
		ContentMD5:    common.String(base64.StdEncoding.EncodeToString(sum)),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(),
		},
	}

	// Handle encryption settings
	if c.kmsKeyID != "" {
		putRequest.OpcSseKmsKeyId = common.String(c.kmsKeyID)
	} else if c.SSECustomerKey != "" && c.SSECustomerKeySHA256 != "" {
		putRequest.OpcSseCustomerKey = common.String(c.SSECustomerKey)
		putRequest.OpcSseCustomerKeySha256 = common.String(c.SSECustomerKeySHA256)
		putRequest.OpcSseCustomerAlgorithm = common.String(c.SSECustomerAlgorithm)
	}

	logger.Info(fmt.Sprintf("Uploading remote state: %s", c.path))

	putResponse, err := c.objectStorageClient.PutObject(ctx, putRequest)
	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}

	logger.Info("Uploaded state file response: %+v\n", putResponse)
	return nil
}

func (c *RemoteClient) Delete() error {

	return c.DeleteAllObjectVersions()
}
func (c *RemoteClient) DeleteAllObjectVersions() error {
	request := objectstorage.ListObjectVersionsRequest{
		BucketName:    common.String(c.bucketName),
		NamespaceName: common.String(c.namespace),
		Prefix:        common.String(c.path),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(),
		},
	}

	response, err := c.objectStorageClient.ListObjectVersions(context.Background(), request)
	if err != nil {
		return err
	}

	request.Page = response.OpcNextPage

	for request.Page != nil {
		request.RequestMetadata.RetryPolicy = getDefaultRetryPolicy()

		listResponse, err := c.objectStorageClient.ListObjectVersions(context.Background(), request)
		if err != nil {
			return err
		}
		response.Items = append(response.Items, listResponse.Items...)
		request.Page = listResponse.OpcNextPage
	}

	var diagErr tfdiags.Diagnostics

	for _, objectVersion := range response.Items {

		deleteObjectVersionRequest := objectstorage.DeleteObjectRequest{
			BucketName:    common.String(c.bucketName),
			NamespaceName: common.String(c.namespace),
			ObjectName:    objectVersion.Name,
			VersionId:     objectVersion.VersionId,
			RequestMetadata: common.RequestMetadata{
				RetryPolicy: getDefaultRetryPolicy(),
			},
		}

		_, err := c.objectStorageClient.DeleteObject(context.Background(), deleteObjectVersionRequest)
		if err != nil {
			diagErr = diagErr.Append(err)
		}
	}
	if diagErr != nil {
		return diagErr.Err()
	}

	return nil
}

func (c *RemoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	logger := logWithOperation("lock-state-file").Named(c.lockFilePath)
	logger.Info("Locking remote state")
	ctx := context.TODO()
	info.Path = c.path
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	putObjReq := objectstorage.PutObjectRequest{
		BucketName:    common.String(c.bucketName),
		NamespaceName: common.String(c.namespace),
		ObjectName:    common.String(c.lockFilePath),
		IfNoneMatch:   common.String("*"),
		PutObjectBody: io.NopCloser(bytes.NewReader(infoBytes)),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(),
		},
	}

	putResponse, putErr := c.objectStorageClient.PutObject(ctx, putObjReq)
	if putErr != nil {
		lockInfo, _, err := c.getLockInfo(ctx)
		if err != nil {
			putErr = errors.Join(putErr, err)
		}
		return "", &statemgr.LockError{
			Err:  putErr,
			Info: lockInfo,
		}
	}
	logger.Info("state lock response code: %+d\n", putResponse.RawResponse.StatusCode)
	return info.ID, nil

}

// getLockInfo retrieves and parses a lock file from an oci bucket.
func (c *RemoteClient) getLockInfo(ctx context.Context) (*statemgr.LockInfo, string, error) {
	// Attempt to retrieve the lock file from
	getRequest := objectstorage.GetObjectRequest{
		NamespaceName: common.String(c.namespace),
		ObjectName:    common.String(c.lockFilePath),
		BucketName:    common.String(c.bucketName),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(),
		},
	}

	getResponse, err := c.objectStorageClient.GetObject(ctx, getRequest)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get existing lock file: %w", err)
	}
	lockByteData, err := io.ReadAll(getResponse.Content)
	if err != nil {
		return nil, *getResponse.ETag, fmt.Errorf("failed to read existing lock file content: %w", err)
	}
	lockInfo := &statemgr.LockInfo{}
	if err := json.Unmarshal(lockByteData, lockInfo); err != nil {
		return lockInfo, "", fmt.Errorf("failed to unmarshal JSON data into LockInfo struct: %w", err)
	}
	return lockInfo, *getResponse.ETag, nil
}
func (c *RemoteClient) Unlock(id string) error {
	ctx := context.TODO()
	logger := logWithOperation("unlock-state-file").Named(c.lockFilePath)
	logger.Info("unlocking remote state")
	lockInfo, etag, err := c.getLockInfo(ctx)

	if err != nil {
		return fmt.Errorf("Failed to retrieve lock information from OCI Object Storage: %w", err)
	}
	// Verify that the provided lock ID matches the lock ID of the retrieved lock file.
	if lockInfo.ID != id {
		return &statemgr.LockError{
			Info: lockInfo,
			Err:  fmt.Errorf("lock ID '%s' does not match the existing lock ID '%s'", id, lockInfo.ID),
		}
	}

	deleteRequest := objectstorage.DeleteObjectRequest{
		NamespaceName: common.String(c.namespace),
		ObjectName:    common.String(c.lockFilePath),
		BucketName:    common.String(c.bucketName),
		IfMatch:       common.String(etag),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(),
		},
	}
	deleteResponse, err := c.objectStorageClient.DeleteObject(ctx, deleteRequest)
	if err != nil {
		return &statemgr.LockError{
			Info: lockInfo,
			Err:  err,
		}
	}
	logger.Info("Unlock response: %v\n", deleteResponse.RawResponse.StatusCode)
	return nil
}
