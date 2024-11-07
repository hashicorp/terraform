// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package convert

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/providers"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
)

func TestProtoDeferred(t *testing.T) {
	testCases := []struct {
		reason   proto.Deferred_Reason
		expected providers.DeferredReason
	}{
		{
			reason:   proto.Deferred_UNKNOWN,
			expected: providers.DeferredReasonInvalid,
		},
		{
			reason:   proto.Deferred_RESOURCE_CONFIG_UNKNOWN,
			expected: providers.DeferredReasonResourceConfigUnknown,
		},
		{
			reason:   proto.Deferred_PROVIDER_CONFIG_UNKNOWN,
			expected: providers.DeferredReasonProviderConfigUnknown,
		},
		{
			reason:   proto.Deferred_ABSENT_PREREQ,
			expected: providers.DeferredReasonAbsentPrereq,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("deferred reason %q", tc.reason.String()), func(t *testing.T) {
			d := &proto.Deferred{
				Reason: tc.reason,
			}

			deferred := ProtoToDeferred(d)
			if deferred.Reason != providers.DeferredReason(tc.expected) {
				t.Fatalf("expected %q, got %q", tc.expected, deferred.Reason)
			}
		})
	}
}

func TestProtoDeferred_Nil(t *testing.T) {
	deferred := ProtoToDeferred(nil)
	if deferred != nil {
		t.Fatalf("expected nil, got %v", deferred)
	}
}
