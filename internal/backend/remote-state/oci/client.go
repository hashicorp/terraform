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
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"
	"io"
)

type RemoteClient struct {
	objectStorageClient  *objectstorage.ObjectStorageClient
	namespace            string
	bucketName           string
	path                 string
	lockFilePath         string
	kmsKeyID             string
	etag                 string
	SSECustomerKey       string
	SSECustomerKeySHA256 string
	SSECustomerAlgorithm string
}

func (c *RemoteClient) Get() (*remote.Payload, error) {
	ctx := context.TODO()

	logger.Info("Downloading remote state")

	payload, err := c.getObject(ctx)
	if err != nil {
		return nil, err
	}
	if md5Hash, err := c.getMd5(ctx); err != nil {
		logger.Warn("Failed to download MD5 hash of remote state")
	} else if !bytes.Equal(md5Hash, payload.MD5) {
		logger.Error("state md5 mismatch expected: %s, actual: %s", string(md5Hash), string(payload.MD5))
		return payload, fmt.Errorf("state md5 mismatch expected: %s, actual: %s", string(md5Hash), string(payload.MD5))
	}
	return payload, nil
}

func (c *RemoteClient) getObject(ctx context.Context) (*remote.Payload, error) {
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
			return nil, nil
		} else {
			return nil, fmt.Errorf("failed to access object '%s' in bucket '%s': %w", c.path, c.bucketName, headErr)
		}
	}

	c.etag = *headResponse.ETag

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
	sum := md5.Sum(contentArray)
	md5Hash := base64.StdEncoding.EncodeToString(sum[:])

	// Construct payload
	payload := &remote.Payload{
		Data: contentArray,
		MD5:  []byte(md5Hash),
	}

	// Return an error instead of `nil, nil` if the object is empty
	if len(payload.Data) == 0 {
		return nil, fmt.Errorf("object %q is empty", c.path)
	}

	return payload, nil
}

func (c *RemoteClient) getMd5(ctx context.Context) ([]byte, error) {
	getRequest := objectstorage.GetObjectRequest{
		NamespaceName: common.String(c.namespace),
		ObjectName:    common.String(fmt.Sprintf("%s.md5", c.path)),
		BucketName:    common.String(c.bucketName),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(),
		},
	}
	getResponse, err := c.objectStorageClient.GetObject(ctx, getRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to download md5 hash of statefile: %w", err)
	}
	var data []byte
	_, err = getResponse.Content.Read(data)
	if err != nil {
		return nil, err
	}
	// Read object content
	contentArray, err := io.ReadAll(getResponse.Content)
	if err != nil {
		return nil, fmt.Errorf("[MD5]unable to read 'content' from response: %w", err)
	}

	return contentArray, nil
}
func (c *RemoteClient) Put(data []byte) error {
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
		err = multipartUploadData.multiPartUploadImpl()
		if err != nil && dataSize <= MaxFilePartSize {
			logger.Error(fmt.Sprintf("Multipart upload failed, falling back to single part upload: %v", err))
			err = c.uploadSinglePartObject(data, sum[:])
		}
	} else {
		err = c.uploadSinglePartObject(data, sum[:])
	}
	if err != nil {
		return err
	}

	return c.putMd5([]byte(base64.StdEncoding.EncodeToString(sum[:])))
}

func (c *RemoteClient) uploadSinglePartObject(data, sum []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("uploadSinglePartObject: data is empty")
	}

	ctx := context.Background()
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
	if c.etag != "" {
		putRequest.IfMatch = common.String(c.etag)
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

func (c *RemoteClient) putMd5(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("uploadSinglePartObject: data is empty")
	}

	ctx := context.Background()
	sum := md5.Sum(data)

	putRequest := objectstorage.PutObjectRequest{
		NamespaceName: common.String(c.namespace),
		ObjectName:    common.String(fmt.Sprintf("%s.md5", c.path)),
		BucketName:    common.String(c.bucketName),
		PutObjectBody: io.NopCloser(bytes.NewReader(data)),
		ContentMD5:    common.String(base64.StdEncoding.EncodeToString(sum[:])),
		RequestMetadata: common.RequestMetadata{
			RetryPolicy: getDefaultRetryPolicy(),
		},
	}
	_, err := c.objectStorageClient.PutObject(ctx, putRequest)
	if err != nil {
		return fmt.Errorf("failed to upload md5Hash: %w", err)
	}
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
		lockInfo, err := c.getLockInfo(ctx)
		if err != nil {
			putErr = errors.Join(putErr, err)
		}
		return "", &statemgr.LockError{
			Err:  putErr,
			Info: lockInfo,
		}
	}
	logger.Debug("state lock response code: %+d\n", putResponse.String())
	return info.ID, nil

}

// getLockInfoWithFile retrieves and parses a lock file from an S3 bucket.
func (c *RemoteClient) getLockInfo(ctx context.Context) (*statemgr.LockInfo, error) {
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
		return nil, fmt.Errorf("failed to get existing lock file: %w", err)
	}
	lockByteData, err := io.ReadAll(getResponse.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read existing lock file content: %w", err)
	}
	lockInfo := &statemgr.LockInfo{}
	if err := json.Unmarshal(lockByteData, lockInfo); err != nil {
		return lockInfo, fmt.Errorf("failed to unmarshal JSON data into LockInfo struct: %w", err)
	}
	return lockInfo, nil
}
func (c *RemoteClient) Unlock(id string) error {
	ctx := context.TODO()
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
		return err
	}
	lockByteData, err := io.ReadAll(getResponse.Content)
	if err != nil {
		return err
	}
	lockInfo := &statemgr.LockInfo{}
	if err := json.Unmarshal(lockByteData, lockInfo); err != nil {
		return fmt.Errorf("failed to unmarshal JSON data into LockInfo struct: %w", err)
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
		IfMatch:       getResponse.ETag,
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
	logger.Debug("Unlock response: %v\n", deleteResponse.String())
	return nil
}
