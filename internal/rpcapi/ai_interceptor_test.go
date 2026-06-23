package rpcapi

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func TestAIGatekeeperInterceptor(t *testing.T) {
	interceptor := AIGatekeeperInterceptor()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/terraform1.Plugin/ApplyResource",
	}

	// Test 1: No metadata, fallback allowed
	ctx := context.Background()
	resp, err := interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("expected no error without metadata, got %v", err)
	}
	if resp != "success" {
		t.Fatalf("expected success, got %v", resp)
	}

	// Test 2: Expired token
	md := metadata.Pairs("ai_authorization_jwt", "expired")
	ctx = metadata.NewIncomingContext(context.Background(), md)
	_, err = interceptor(ctx, nil, info, handler)
	if err == nil {
		t.Fatalf("expected error for expired token, got nil")
	}

	// Test 3: Rejected token
	md = metadata.Pairs("ai_authorization_jwt", "reject_policy")
	ctx = metadata.NewIncomingContext(context.Background(), md)
	_, err = interceptor(ctx, nil, info, handler)
	if err == nil {
		t.Fatalf("expected error for rejected token, got nil")
	}

	// Test 4: Valid token
	md = metadata.Pairs("ai_authorization_jwt", "valid_token")
	ctx = metadata.NewIncomingContext(context.Background(), md)
	resp, err = interceptor(ctx, nil, info, handler)
	if err != nil {
		t.Fatalf("expected no error for valid token, got %v", err)
	}
	if resp != "success" {
		t.Fatalf("expected success, got %v", resp)
	}
}
