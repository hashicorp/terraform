// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
)

// grpcClientForTesting creates an in-memory-only gRPC server, offers the
// caller a chance to register services with it, and then returns a
// client connected to that fake server, with which the caller can construct
// service-specific client objects.
//
// When finished with the returned client, call the close callback given as
// the second return value or else you will leak some goroutines handling the
// server end of this fake connection.
func grpcClientForTesting(ctx context.Context, t *testing.T, registerServices func(srv *grpc.Server)) (conn grpc.ClientConnInterface, close func()) {
	fakeListener := bufconn.Listen(1024 /* buffer size */)
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)

	// Caller gets an opportunity to register specific services before
	// we actually start "serving".
	registerServices(srv)

	go func() {
		if err := srv.Serve(fakeListener); err != nil {
			// We can't actually return an error here, but this should
			// not arise with our fake listener anyway so we'll just panic.
			panic(err)
		}
	}()

	fakeDialer := func(ctx context.Context, fakeAddr string) (net.Conn, error) {
		return fakeListener.DialContext(ctx)
	}
	realConn, err := grpc.DialContext(
		ctx, "testfake",
		grpc.WithContextDialer(fakeDialer),
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
	)
	if err != nil {
		t.Fatalf("failed to connect to the fake server: %s", err)
	}

	return realConn, func() {
		realConn.Close()
		srv.Stop()
		fakeListener.Close()
	}
}

func appliedChangeToRawState(t *testing.T, changes []stackstate.AppliedChange) map[string]*anypb.Any {
	ret := make(map[string]*anypb.Any)
	for _, change := range changes {
		raw, err := change.AppliedChangeProto()
		if err != nil {
			t.Fatalf("failed to marshal change to proto: %s", err)
		}
		for _, raw := range raw.Raw {
			ret[raw.Key] = raw.Value
		}
	}
	return ret
}

func mustDefaultRootProvider(provider string) addrs.AbsProviderConfig {
	return addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.NewDefaultProvider(provider),
	}
}

func mustAbsComponentInstance(t *testing.T, addr string) stackaddrs.AbsComponentInstance {
	ret, diags := stackaddrs.ParseAbsComponentInstanceStr(addr)
	if len(diags) > 0 {
		t.Fatalf("failed to parse component instance address %q: %s", addr, diags)
	}
	return ret
}

func mustAbsComponent(t *testing.T, addr string) stackaddrs.AbsComponent {
	ret, diags := stackaddrs.ParseAbsComponentInstanceStr(addr)
	if len(diags) > 0 {
		t.Fatalf("failed to parse component instance address %q: %s", addr, diags)
	}
	if ret.Item.Key != addrs.NoKey {
		t.Fatalf("expected component address %q to have no key, but got %q", addr, ret.Item.Key)
	}
	return stackaddrs.AbsComponent{
		Stack: ret.Stack,
		Item:  ret.Item.Component,
	}
}

func mustAbsResourceInstanceObject(t *testing.T, addr string) stackaddrs.AbsResourceInstanceObject {
	ret, diags := stackaddrs.ParseAbsResourceInstanceObjectStr(addr)
	if len(diags) > 0 {
		t.Fatalf("failed to parse resource instance object address %q: %s", addr, diags)
	}
	return ret
}

func mustMarshalAnyPb(msg proto.Message) *anypb.Any {
	var ret anypb.Any
	err := anypb.MarshalFrom(&ret, msg, proto.MarshalOptions{})
	if err != nil {
		panic(err)
	}
	return &ret
}

func mustMarshalJSONAttrs(attrs map[string]interface{}) []byte {
	jsonAttrs, err := json.Marshal(attrs)
	if err != nil {
		panic(err)
	}
	return jsonAttrs
}
