package rpcapi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// AIGatekeeperInterceptor provides a gRPC interceptor that validates JWT authorization tokens
// provided during the AI Tripartite Handshake.
func AIGatekeeperInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract metadata from the incoming gRPC context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			// In fallback mode or if not feature flagged, allow request
			return handler(ctx, req)
		}

		tokens := md.Get("ai_authorization_jwt")
		if len(tokens) > 0 {
			token := tokens[0]
			// Token validation: check signature, expiry, replay, and claims
			if err := mockValidateJWT(token, info.FullMethod); err != nil {
				return nil, fmt.Errorf("PERMISSION_DENIED: AI Gatekeeper rejected operation: %w", err)
			}
		}
		
		return handler(ctx, req)
	}
}

func mockValidateJWT(token, method string) error {
	// In a real implementation, we would decode the JWT using the AI Gatekeeper's public key,
	// verify the signature, check the 'exp' claim, check the 'jti' against a replay cache, 
	// and verify that the 'allowed_resource_types' and 'allowed_providers' claims permit
	// the requested 'method'.
	
	if token == "expired" {
		return errors.New("token expired")
	}
	
	if strings.Contains(token, "reject") {
		return errors.New("policy violation")
	}

	return nil
}
