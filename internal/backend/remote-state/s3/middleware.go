// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package s3

import (
	"context"
	"net/http"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// This will not be needed once https://github.com/aws/aws-sdk-go-v2/issues/2282
// is addressed
func addS3WrongRegionErrorMiddleware(stack *middleware.Stack) error {
	return stack.Deserialize.Insert(
		&s3WrongRegionErrorMiddleware{},
		"ResponseErrorWrapper",
		middleware.After,
	)
}

var _ middleware.DeserializeMiddleware = &s3WrongRegionErrorMiddleware{}

type s3WrongRegionErrorMiddleware struct{}

func (m *s3WrongRegionErrorMiddleware) ID() string {
	return "tf_S3WrongRegionErrorMiddleware"
}

func (m *s3WrongRegionErrorMiddleware) HandleDeserialize(ctx context.Context, in middleware.DeserializeInput, next middleware.DeserializeHandler) (
	out middleware.DeserializeOutput, metadata middleware.Metadata, err error,
) {
	out, metadata, err = next.HandleDeserialize(ctx, in)
	if err == nil || !IsA[*smithy.GenericAPIError](err) {
		return out, metadata, err
	}

	resp, ok := out.RawResponse.(*smithyhttp.Response)
	if !ok || resp.StatusCode != http.StatusMovedPermanently {
		return out, metadata, err
	}

	reqRegion := awsmiddleware.GetRegion(ctx)

	bucketRegion := resp.Header.Get("X-Amz-Bucket-Region")

	err = newBucketRegionError(reqRegion, bucketRegion)

	return out, metadata, err
}
