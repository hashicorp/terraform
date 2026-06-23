package aigatekeeper

import (
	"context"
	"fmt"
	"time"
)

// Gatekeeper defines the interface for the AI-mediated zero-trust service.
type Gatekeeper interface {
	// RequestAuthorization queries the AI inference engine for permission to execute
	// a planned DAG of resources. It returns a signed JWT or an error.
	RequestAuthorization(ctx context.Context, req AuthorizationRequest) (string, error)
}

type AuthorizationRequest struct {
	ProtocolVersion int32
	ClientIdentity  string
	DAGPreview      []byte // Serialized structured DAG
	Nonce           string
	Timestamp       time.Time
}

// localGatekeeper provides a lightweight local sidecar or in-process fallback for the AI Gatekeeper.
type localGatekeeper struct {
	FallbackAllowed bool
}

func NewLocalGatekeeper(fallbackAllowed bool) Gatekeeper {
	return &localGatekeeper{FallbackAllowed: fallbackAllowed}
}

func (g *localGatekeeper) RequestAuthorization(ctx context.Context, req AuthorizationRequest) (string, error) {
	// In a complete implementation, this would:
	// 1. Send the DAG preview to an external AI service or local ML model.
	// 2. Wait for the inference diagnostics (anomaly score, allowed capabilities).
	// 3. Issue and sign a JWT containing the permitted capabilities.
	
	if g.FallbackAllowed {
		// Return a mock fallback JWT token
		return "mock.jwt.token", nil
	}
	
	return "", fmt.Errorf("AI inference service unavailable and fallback is disabled")
}
