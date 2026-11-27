// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package callback

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/internal/policy/proto"
)

func TestServerGetResourcesByIdentifier(t *testing.T) {
	registry := NewRegistry()
	server := &Server{Registry: registry}
	identifiers := []string{"test_resource.b", "test_resource.a", "test_resource.b"}
	resources := []*proto.GetResourcesByIdentifierResponse_Resource{
		{Identifier: identifiers[0]}, {Identifier: identifiers[1]}, {Identifier: identifiers[2]},
	}
	registry.Register(7, Functions{
		GetResourcesByIdentifier: func(ctx context.Context, got []string) ([]*proto.GetResourcesByIdentifierResponse_Resource, error) {
			if !reflect.DeepEqual(got, identifiers) {
				t.Errorf("identifiers = %v, want %v", got, identifiers)
			}
			return resources, nil
		},
	})
	response, err := server.GetResourcesByIdentifier(context.Background(), &proto.GetResourcesByIdentifierRequest{
		EvaluationRequestId: 7, Identifiers: identifiers,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(response.Resources, resources) {
		t.Errorf("resources = %v, want %v", response.Resources, resources)
	}

	registry.Unregister(7)
	if _, err := server.GetResourcesByIdentifier(context.Background(), &proto.GetResourcesByIdentifierRequest{EvaluationRequestId: 7}); err == nil {
		t.Fatal("expected an error after callback unregistration")
	}
}

func TestServerGetResourcesByIdentifierErrors(t *testing.T) {
	callbackErr := errors.New("resource unavailable")
	tests := map[string]struct {
		register bool
		callback func(context.Context, []string) ([]*proto.GetResourcesByIdentifierResponse_Resource, error)
		want     string
	}{
		"unknown evaluation":   {want: "no callback registered for ID 7 (request type: GetResourcesByIdentifier)"},
		"unavailable callback": {register: true, want: "GetResourcesByIdentifier callback unavailable for ID 7"},
		"callback error": {
			register: true,
			callback: func(context.Context, []string) ([]*proto.GetResourcesByIdentifierResponse_Resource, error) {
				return nil, callbackErr
			},
			want: callbackErr.Error(),
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			registry := NewRegistry()
			if tc.register {
				registry.Register(7, Functions{GetResourcesByIdentifier: tc.callback})
			}
			server := &Server{Registry: registry}
			response, err := server.GetResourcesByIdentifier(context.Background(), &proto.GetResourcesByIdentifierRequest{EvaluationRequestId: 7})
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if response != nil {
				t.Error("expected no partial response on error")
			}
		})
	}
}
