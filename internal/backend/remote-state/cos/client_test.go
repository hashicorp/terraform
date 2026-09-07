// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cos

import (
	"errors"
	"fmt"
	"testing"

	sdkErrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
)

// TestIsTagNotExistError verifies that isTagNotExistError correctly recognizes
// the "tag does not exist" error returned by the TencentCloud tag service, even
// when the SDK error is wrapped by DeleteTag.
func TestIsTagNotExistError(t *testing.T) {
	tagNotExist := sdkErrors.NewTencentCloudSDKError(ignoreDelTagErrorCode, "tag does not exist", "req-1")
	otherCode := sdkErrors.NewTencentCloudSDKError("ResourceNotFound.Other", "some other error", "req-2")

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "raw TagNonExist SDK error",
			err:  tagNotExist,
			want: true,
		},
		{
			name: "TagNonExist wrapped with %w (DeleteTag behavior)",
			err:  fmt.Errorf("failed to delete tag: %w", tagNotExist),
			want: true,
		},
		{
			name: "TagNonExist wrapped with %s (old behavior, loses type)",
			err:  fmt.Errorf("failed to delete tag: %s", tagNotExist),
			want: false,
		},
		{
			name: "other SDK error code wrapped with %w",
			err:  fmt.Errorf("failed to delete tag: %w", otherCode),
			want: false,
		},
		{
			name: "plain non-SDK error",
			err:  errors.New("boom"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isTagNotExistError(tc.err); got != tc.want {
				t.Fatalf("isTagNotExistError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
