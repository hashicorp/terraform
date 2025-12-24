package oras

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/backend"
	orasAuth "oras.land/oras-go/v2/registry/remote/auth"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestORASRetryConfigFromConfig(t *testing.T) {
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"repository":     "example.com/myorg/terraform-state",
		"retry_max":      9,
		"retry_wait_min": 15,
		"retry_wait_max": 150,
	})).(*Backend)

	if b.retryCfg.MaxAttempts != 10 { // retry_max is number of retries
		t.Fatalf("expected MaxAttempts %d, got %d", 10, b.retryCfg.MaxAttempts)
	}
	if b.retryCfg.InitialBackoff != 15*time.Second {
		t.Fatalf("expected InitialBackoff %s, got %s", 15*time.Second, b.retryCfg.InitialBackoff)
	}
	if b.retryCfg.MaxBackoff != 150*time.Second {
		t.Fatalf("expected MaxBackoff %s, got %s", 150*time.Second, b.retryCfg.MaxBackoff)
	}
}

func TestORASRetryConfigFromEnv(t *testing.T) {
	t.Setenv(envVarRepository, "example.com/myorg/terraform-state")
	t.Setenv(envVarRetryMax, "9")
	t.Setenv(envVarRetryWaitMin, "15")
	t.Setenv(envVarRetryWaitMax, "150")

	b := backend.TestBackendConfig(t, New(), nil).(*Backend)

	if b.retryCfg.MaxAttempts != 10 {
		t.Fatalf("expected MaxAttempts %d, got %d", 10, b.retryCfg.MaxAttempts)
	}
	if b.retryCfg.InitialBackoff != 15*time.Second {
		t.Fatalf("expected InitialBackoff %s, got %s", 15*time.Second, b.retryCfg.InitialBackoff)
	}
	if b.retryCfg.MaxBackoff != 150*time.Second {
		t.Fatalf("expected MaxBackoff %s, got %s", 150*time.Second, b.retryCfg.MaxBackoff)
	}
}

func TestORASVersioningConfigFromConfig(t *testing.T) {
	src := []byte(`
repository = "example.com/myorg/terraform-state"

versioning {
  enabled      = true
  max_versions = 42
}
`)

	f, diags := hclsyntax.ParseConfig(src, "synth.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("parse config: %s", diags.Error())
	}

	b := backend.TestBackendConfig(t, New(), f.Body).(*Backend)

	if !b.versioningEnabled {
		t.Fatalf("expected versioningEnabled to be true")
	}
	if b.versioningMaxVersions != 42 {
		t.Fatalf("expected versioningMaxVersions %d, got %d", 42, b.versioningMaxVersions)
	}
}

func TestTerraformTokenCredentialFunc_UsesEnvToken(t *testing.T) {
	// Ensure we don't leak env to other tests
	envName := "TF_TOKEN_example.com"
	old, hadOld := os.LookupEnv(envName)
	if err := os.Setenv(envName, "test-token"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv(envName, old)
		} else {
			_ = os.Unsetenv(envName)
		}
	})

	fn := terraformTokenCredentialFunc()
	cred, err := fn(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cred == orasAuth.EmptyCredential {
		t.Fatalf("expected credential")
	}
	if cred.AccessToken != "test-token" {
		t.Fatalf("expected access token to be %q, got %q", "test-token", cred.AccessToken)
	}
}

func TestORASCompressionConfigFromConfig(t *testing.T) {
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"repository":  "example.com/myorg/terraform-state",
		"compression": "gzip",
	})).(*Backend)
	if b.compression != "gzip" {
		t.Fatalf("expected compression %q, got %q", "gzip", b.compression)
	}
}

func TestORASLockTTLConfigFromConfig(t *testing.T) {
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"repository": "example.com/myorg/terraform-state",
		"lock_ttl":   60,
	})).(*Backend)
	if b.lockTTL != 60*time.Second {
		t.Fatalf("expected lockTTL %s, got %s", 60*time.Second, b.lockTTL)
	}
}

func TestORASRateLimitConfigFromConfig(t *testing.T) {
	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(map[string]interface{}{
		"repository":       "example.com/myorg/terraform-state",
		"rate_limit":       10,
		"rate_limit_burst": 3,
		"retry_max":        0,
		"retry_wait_min":   1,
		"retry_wait_max":   1,
		"compression":      "none",
	})).(*Backend)
	if b.rateLimit != 10 {
		t.Fatalf("expected rateLimit %d, got %d", 10, b.rateLimit)
	}
	if b.rateBurst != 3 {
		t.Fatalf("expected rateBurst %d, got %d", 3, b.rateBurst)
	}
}

type blockingLimiter struct {
	ch <-chan struct{}
}

func (l blockingLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-l.ch:
		return nil
	}
}

type countingRoundTripper struct {
	mu    sync.Mutex
	calls int
}

func (rt *countingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.mu.Lock()
	rt.calls++
	rt.mu.Unlock()
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func (rt *countingRoundTripper) Calls() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return rt.calls
}

func TestRateLimitedRoundTripper_WaitsBeforeRequest(t *testing.T) {
	gate := make(chan struct{})
	inner := &countingRoundTripper{}
	rt := &rateLimitedRoundTripper{limiter: blockingLimiter{ch: gate}, inner: inner}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://example.com/", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	done := make(chan struct{})
	go func() {
		_, _ = rt.RoundTrip(req)
		close(done)
	}()

	if inner.Calls() != 0 {
		t.Fatalf("expected no calls before limiter release")
	}

	close(gate)
	<-done

	if inner.Calls() != 1 {
		t.Fatalf("expected exactly 1 call after limiter release, got %d", inner.Calls())
	}
}
