// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oci

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/oracle/oci-go-sdk/v65/objectstorage"

	"github.com/oracle/oci-go-sdk/v65/common"
)

var DefaultFilePartSize int64 = 128 * 1024 * 1024     // 128MB
const MaxFilePartSize int64 = 50 * 1024 * 1024 * 1024 // 50GB
const defaultNumberOfGoroutines = 10
const MaxCount int64 = 10000

type MultipartUploadData struct {
	client          *RemoteClient
	Data            []byte
	RequestMetadata common.RequestMetadata
}

type objectStorageUploadPartResponse struct {
	response   objectstorage.UploadPartResponse
	partNumber *int
	error      error
}

type objectStorageMultiPartUploadContext struct {
	client                  *RemoteClient
	sourceBlocks            chan objectStorageSourceBlock
	osUploadPartResponses   chan objectStorageUploadPartResponse
	wg                      *sync.WaitGroup
	errChan                 chan error
	multipartUploadResponse objectstorage.CreateMultipartUploadResponse
	multipartUploadRequest  objectstorage.CreateMultipartUploadRequest
	logger                  hclog.Logger
}

type objectStorageSourceBlock struct {
	section     *io.SectionReader
	blockNumber *int
}

func (multipartUploadData MultipartUploadData) multiPartUploadImpl(ctx context.Context) error {
	logger := ctx.Value("logger").(hclog.Logger).Named("multiPartUpload")
	sourceBlocks, err := multipartUploadData.objectMultiPartSplit()
	if err != nil {
		return fmt.Errorf("error splitting source data: %s", err)
	}

	multipartUploadRequest := &objectstorage.CreateMultipartUploadRequest{
		NamespaceName:   common.String(multipartUploadData.client.namespace),
		BucketName:      common.String(multipartUploadData.client.bucketName),
		RequestMetadata: multipartUploadData.RequestMetadata,
		CreateMultipartUploadDetails: objectstorage.CreateMultipartUploadDetails{
			Object: common.String(multipartUploadData.client.path),
		},
	}
	if multipartUploadData.client.kmsKeyID != "" {
		multipartUploadRequest.OpcSseKmsKeyId = common.String(multipartUploadData.client.kmsKeyID)
	} else if multipartUploadData.client.SSECustomerKey != "" && multipartUploadData.client.SSECustomerKeySHA256 != "" {
		multipartUploadRequest.OpcSseCustomerKey = common.String(multipartUploadData.client.SSECustomerKey)
		multipartUploadRequest.OpcSseCustomerKeySha256 = common.String(multipartUploadData.client.SSECustomerKeySHA256)
		multipartUploadRequest.OpcSseCustomerAlgorithm = common.String(multipartUploadData.client.SSECustomerAlgorithm)
	}

	multipartUploadResponse, err := multipartUploadData.client.objectStorageClient.CreateMultipartUpload(context.Background(), *multipartUploadRequest)
	if err != nil {
		return fmt.Errorf("error creating multipart upload: %s", err)
	}

	workerCount := defaultNumberOfGoroutines
	osUploadPartResponses := make(chan objectStorageUploadPartResponse, len(sourceBlocks))
	sourceBlocksChan := make(chan objectStorageSourceBlock, len(sourceBlocks))

	wg := &sync.WaitGroup{}
	wg.Add(len(sourceBlocks))

	// Push all source blocks into the channel
	for _, sourceBlock := range sourceBlocks {
		sourceBlocksChan <- sourceBlock
	}
	close(sourceBlocksChan)
	errChan := make(chan error, workerCount)
	// Start workers
	for i := 0; i < workerCount; i++ {
		go func() {
			ctx := &objectStorageMultiPartUploadContext{
				client:                  multipartUploadData.client,
				wg:                      wg,
				errChan:                 errChan,
				multipartUploadResponse: multipartUploadResponse,
				multipartUploadRequest:  *multipartUploadRequest,
				sourceBlocks:            sourceBlocksChan,
				osUploadPartResponses:   osUploadPartResponses,
				logger:                  logger,
			}
			ctx.uploadPartsWorker()
		}()
	}

	wg.Wait()
	close(osUploadPartResponses)
	close(errChan)

	// Collect errors from workers
	for workerErr := range errChan {
		if workerErr != nil {
			return workerErr
		}
	}
	commitMultipartUploadPartDetails := make([]objectstorage.CommitMultipartUploadPartDetails, len(sourceBlocks))
	i := 0
	for response := range osUploadPartResponses {
		if response.error != nil || response.partNumber == nil || response.response.ETag == nil {
			return fmt.Errorf("failed to upload part: %s", response.error)
		}
		partNumber, etag := *response.partNumber, *response.response.ETag
		commitMultipartUploadPartDetails[i] = objectstorage.CommitMultipartUploadPartDetails{
			PartNum: common.Int(partNumber),
			Etag:    common.String(etag),
		}
		i++
	}

	if len(commitMultipartUploadPartDetails) != len(sourceBlocks) {
		abortReq := objectstorage.AbortMultipartUploadRequest{
			UploadId:      multipartUploadResponse.MultipartUpload.UploadId,
			NamespaceName: multipartUploadResponse.Namespace,
			BucketName:    multipartUploadResponse.Bucket,
			ObjectName:    multipartUploadResponse.Object,
		}
		_, abortErr := multipartUploadData.client.objectStorageClient.AbortMultipartUpload(context.Background(), abortReq)
		if abortErr != nil {
			logger.Error(fmt.Sprintf("Failed to abort multipart upload: %s", abortErr))
		}
		return fmt.Errorf("not all parts uploaded successfully, multipart upload aborted")
	}

	commitMultipartUploadRequest := objectstorage.CommitMultipartUploadRequest{
		UploadId:           multipartUploadResponse.MultipartUpload.UploadId,
		NamespaceName:      multipartUploadResponse.Namespace,
		BucketName:         multipartUploadResponse.Bucket,
		ObjectName:         multipartUploadResponse.Object,
		OpcClientRequestId: multipartUploadResponse.OpcClientRequestId,
		RequestMetadata:    multipartUploadRequest.RequestMetadata,
		CommitMultipartUploadDetails: objectstorage.CommitMultipartUploadDetails{
			PartsToCommit: commitMultipartUploadPartDetails,
		},
	}
	_, err = multipartUploadData.client.objectStorageClient.CommitMultipartUpload(context.Background(), commitMultipartUploadRequest)
	if err != nil {
		return fmt.Errorf("failed to commit multipart upload: %s", err)
	}

	return nil
}
func (m MultipartUploadData) objectMultiPartSplit() ([]objectStorageSourceBlock, error) {
	dataSize := int64(len(m.Data))
	offsets, partSize, err := SplitSizeToOffsetsAndLimits(dataSize)
	if err != nil {
		return nil, fmt.Errorf("error splitting data into parts: %s", err)
	}
	sourceBlocks := make([]objectStorageSourceBlock, len(offsets))
	for i := range offsets {
		start := offsets[i]
		end := start + partSize
		if end > dataSize {
			end = dataSize
		}
		sourceBlocks[i] = objectStorageSourceBlock{
			section:     io.NewSectionReader(bytes.NewReader(m.Data), start, end-start),
			blockNumber: common.Int(i + 1),
		}
	}
	return sourceBlocks, nil
}

/*
SplitSizeToOffsetsAndLimits splits a file size into chunks based on DefaultFilePartSize.
Returns the byte offsets and byte limits for each chunk.
Returns an error if the size exceeds MaxCount parts.
*/
func SplitSizeToOffsetsAndLimits(size int64) ([]int64, int64, error) {
	partSize := DefaultFilePartSize
	totalParts := (size + partSize - 1) / partSize
	if totalParts > MaxCount {
		return nil, 0, fmt.Errorf("file exceeds maximum part count")
	}
	offsets := make([]int64, totalParts)
	for i := range offsets {
		offsets[i] = int64(i) * partSize
	}
	return offsets, partSize, nil
}

func (ctx *objectStorageMultiPartUploadContext) uploadPartsWorker() {
	for block := range ctx.sourceBlocks {
		buffer := make([]byte, block.section.Size())
		_, err := block.section.Read(buffer)
		if err != nil {
			ctx.errChan <- fmt.Errorf("error reading source block %d: %w", block.blockNumber, err)
			return
		}
		tmpLength := int64(len(buffer))
		sum := md5.Sum(buffer)
		uploadPartRequest := &objectstorage.UploadPartRequest{
			UploadId:       ctx.multipartUploadResponse.UploadId,
			ObjectName:     ctx.multipartUploadResponse.Object,
			NamespaceName:  ctx.multipartUploadResponse.Namespace,
			BucketName:     ctx.multipartUploadResponse.Bucket,
			ContentLength:  &tmpLength,
			UploadPartBody: io.NopCloser(bytes.NewReader(buffer)),
			UploadPartNum:  block.blockNumber,
			ContentMD5:     common.String(base64.StdEncoding.EncodeToString(sum[:])),
			RequestMetadata: common.RequestMetadata{
				RetryPolicy: getDefaultRetryPolicy(),
			},
		}

		if ctx.client.kmsKeyID != "" {
			uploadPartRequest.OpcSseKmsKeyId = common.String(ctx.client.kmsKeyID)
		} else if ctx.client.SSECustomerKey != "" && ctx.client.SSECustomerKeySHA256 != "" {
			uploadPartRequest.OpcSseCustomerKey = common.String(ctx.client.SSECustomerKey)
			uploadPartRequest.OpcSseCustomerKeySha256 = common.String(ctx.client.SSECustomerKeySHA256)
			uploadPartRequest.OpcSseCustomerAlgorithm = common.String(ctx.client.SSECustomerAlgorithm)
		}

		response, err := ctx.client.objectStorageClient.UploadPart(context.Background(), *uploadPartRequest)
		if err != nil {
			ctx.errChan <- fmt.Errorf("failed to upload part %d: %w", *block.blockNumber, err)
			return
		}
		ctx.osUploadPartResponses <- objectStorageUploadPartResponse{
			response:   response,
			error:      nil,
			partNumber: block.blockNumber,
		}
		ctx.wg.Done()

	}
}
