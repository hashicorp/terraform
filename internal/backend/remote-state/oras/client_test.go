package oras

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/internal/states/statemgr"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	oraslib "oras.land/oras-go/v2"
	orasErrcode "oras.land/oras-go/v2/registry/remote/errcode"
)

func retryConfigForTests() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}
}

func TestRemoteClient_LockContentionAndUnlockMismatch(t *testing.T) {
	ctx := context.Background()
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: fake}

	client1 := newRemoteClient(repo, "default")
	client2 := newRemoteClient(repo, "default")

	info := &statemgr.LockInfo{ID: "lock-1", Operation: "test", Info: "hello"}
	id, err := client1.Lock(info)
	if err != nil {
		t.Fatalf("expected first lock to succeed, got error: %v", err)
	}
	if id != "lock-1" {
		t.Fatalf("expected lock id to be %q, got %q", "lock-1", id)
	}

	info2 := &statemgr.LockInfo{ID: "lock-2", Operation: "test", Info: "hello"}
	_, err = client2.Lock(info2)
	if err == nil {
		t.Fatalf("expected second lock to fail")
	}
	if _, ok := err.(*statemgr.LockError); !ok {
		t.Fatalf("expected LockError, got %T: %v", err, err)
	}

	if err := client1.Unlock("wrong"); err == nil {
		t.Fatalf("expected unlock mismatch error")
	}

	if err := client1.Unlock("lock-1"); err != nil {
		t.Fatalf("expected unlock success, got: %v", err)
	}

	// Unlock deletes the lock manifest, so the lock tag must no longer resolve.
	if _, err := fake.Resolve(ctx, client1.lockTag); err == nil {
		t.Fatalf("expected lock tag to be gone after unlock")
	}

	// After unlock, it should be possible to lock again.
	_, err = client2.Lock(&statemgr.LockInfo{ID: "lock-3", Operation: "test"})
	if err != nil {
		t.Fatalf("expected lock after unlock to succeed, got: %v", err)
	}
}

func TestRemoteClient_WorkspacesFromTags_TagSafeAndHashed(t *testing.T) {
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: fake}

	// Tag-safe workspace
	c1 := newRemoteClient(repo, "dev")
	c1.versioningEnabled = true
	c1.versioningMaxVersions = 10
	if diags := c1.Put([]byte("state-dev")); diags.HasErrors() {
		t.Fatalf("put dev: %v", diags.Err())
	}
	if diags := c1.Put([]byte("state-dev-2")); diags.HasErrors() {
		t.Fatalf("put dev second: %v", diags.Err())
	}

	// Tag-unsafe workspace (space)
	c2 := newRemoteClient(repo, "my workspace")
	c2.versioningEnabled = true
	c2.versioningMaxVersions = 10
	if diags := c2.Put([]byte("state-unsafe")); diags.HasErrors() {
		t.Fatalf("put unsafe: %v", diags.Err())
	}
	if diags := c2.Put([]byte("state-unsafe-2")); diags.HasErrors() {
		t.Fatalf("put unsafe second: %v", diags.Err())
	}

	got, err := listWorkspacesFromTags(repo)
	if err != nil {
		t.Fatalf("workspaces: %v", err)
	}

	want := map[string]struct{}{"dev": {}, "my workspace": {}}
	for _, w := range got {
		delete(want, w)
	}
	if len(want) != 0 {
		t.Fatalf("missing workspaces: %v; got %v", want, got)
	}
}

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{name: "nil error", err: nil, expected: false},
		{name: "regular error", err: errors.New("something went wrong"), expected: false},
		{name: "429 Too Many Requests", err: &orasErrcode.ErrorResponse{StatusCode: http.StatusTooManyRequests}, expected: true},
		{name: "502 Bad Gateway", err: &orasErrcode.ErrorResponse{StatusCode: http.StatusBadGateway}, expected: true},
		{name: "503 Service Unavailable", err: &orasErrcode.ErrorResponse{StatusCode: http.StatusServiceUnavailable}, expected: true},
		{name: "504 Gateway Timeout", err: &orasErrcode.ErrorResponse{StatusCode: http.StatusGatewayTimeout}, expected: true},
		{name: "408 Request Timeout", err: &orasErrcode.ErrorResponse{StatusCode: http.StatusRequestTimeout}, expected: true},
		{name: "404 Not Found (not transient)", err: &orasErrcode.ErrorResponse{StatusCode: http.StatusNotFound}, expected: false},
		{name: "401 Unauthorized (not transient)", err: &orasErrcode.ErrorResponse{StatusCode: http.StatusUnauthorized}, expected: false},
		{name: "error with connection reset in message", err: errors.New("read tcp: connection reset by peer"), expected: true},
		{name: "error with connection refused in message", err: errors.New("dial tcp: connection refused"), expected: true},
		{name: "error with timeout in message", err: errors.New("connection timeout occurred"), expected: true},
		{name: "error with EOF in message", err: errors.New("unexpected EOF"), expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTransientError(tt.err)
			if result != tt.expected {
				t.Errorf("isTransientError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestWithRetry_Success(t *testing.T) {
	ctx := context.Background()
	cfg := retryConfigForTests()

	attempts := 0
	result, err := withRetry(ctx, cfg, func(ctx context.Context) (string, error) {
		attempts++
		return "success", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}

func TestWithRetry_TransientFailureThenSuccess(t *testing.T) {
	ctx := context.Background()
	cfg := retryConfigForTests()

	attempts := 0
	result, err := withRetry(ctx, cfg, func(ctx context.Context) (string, error) {
		attempts++
		if attempts < 3 {
			return "", &orasErrcode.ErrorResponse{StatusCode: http.StatusServiceUnavailable}
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestWithRetry_NonTransientFailure(t *testing.T) {
	ctx := context.Background()
	cfg := retryConfigForTests()

	attempts := 0
	_, err := withRetry(ctx, cfg, func(ctx context.Context) (string, error) {
		attempts++
		return "", &orasErrcode.ErrorResponse{StatusCode: http.StatusUnauthorized}
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry for non-transient), got %d", attempts)
	}
}

func TestWithRetry_MaxAttemptsExhausted(t *testing.T) {
	ctx := context.Background()
	cfg := retryConfigForTests()

	attempts := 0
	_, err := withRetry(ctx, cfg, func(ctx context.Context) (string, error) {
		attempts++
		return "", &orasErrcode.ErrorResponse{StatusCode: http.StatusServiceUnavailable}
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestWithRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := RetryConfig{MaxAttempts: 5, InitialBackoff: 100 * time.Millisecond, MaxBackoff: 1 * time.Second, BackoffMultiplier: 2.0}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := withRetry(ctx, cfg, func(ctx context.Context) (string, error) {
		return "", &orasErrcode.ErrorResponse{StatusCode: http.StatusServiceUnavailable}
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestWithRetryNoResult(t *testing.T) {
	ctx := context.Background()
	cfg := retryConfigForTests()

	attempts := 0
	err := withRetryNoResult(ctx, cfg, func(ctx context.Context) error {
		attempts++
		if attempts < 2 {
			return &orasErrcode.ErrorResponse{StatusCode: http.StatusServiceUnavailable}
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRemoteClient_Put_VersioningTagsAndRetention(t *testing.T) {
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: fake}

	c := newRemoteClient(repo, "default")
	c.versioningEnabled = true
	c.versioningMaxVersions = 2

	if diags := c.Put([]byte("s1")); diags.HasErrors() {
		t.Fatalf("put s1: %v", diags.Err())
	}
	if diags := c.Put([]byte("s2")); diags.HasErrors() {
		t.Fatalf("put s2: %v", diags.Err())
	}
	if diags := c.Put([]byte("s3")); diags.HasErrors() {
		t.Fatalf("put s3: %v", diags.Err())
	}

	p, getDiags := c.Get()
	if getDiags.HasErrors() {
		t.Fatalf("get latest: %v", getDiags.Err())
	}
	if p == nil || string(p.Data) != "s3" {
		got := "<nil>"
		if p != nil {
			got = string(p.Data)
		}
		t.Fatalf("expected latest state %q, got %q", "s3", got)
	}

	ctx := context.Background()

	if _, err := fake.Resolve(ctx, c.versionTagFor(1)); err == nil {
		t.Fatalf("expected v1 to be deleted due to retention")
	}
	if _, err := fake.Resolve(ctx, c.versionTagFor(2)); err != nil {
		t.Fatalf("expected v2 to exist, got: %v", err)
	}
	if _, err := fake.Resolve(ctx, c.versionTagFor(3)); err != nil {
		t.Fatalf("expected v3 to exist, got: %v", err)
	}
}

func TestRemoteClient_Put_VersionRetention_WhenStateUnchanged(t *testing.T) {
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: fake}

	c := newRemoteClient(repo, "default")
	c.versioningEnabled = true
	c.versioningMaxVersions = 2

	stateBytes := []byte("same-state")
	if diags := c.Put(stateBytes); diags.HasErrors() {
		t.Fatalf("put 1: %v", diags.Err())
	}
	if diags := c.Put(stateBytes); diags.HasErrors() {
		t.Fatalf("put 2: %v", diags.Err())
	}
	if diags := c.Put(stateBytes); diags.HasErrors() {
		t.Fatalf("put 3: %v", diags.Err())
	}

	ctx := context.Background()
	if _, err := fake.Resolve(ctx, c.versionTagFor(1)); err == nil {
		t.Fatalf("expected v1 to be deleted due to retention")
	}
	if _, err := fake.Resolve(ctx, c.versionTagFor(2)); err != nil {
		t.Fatalf("expected v2 to exist, got: %v", err)
	}
	if _, err := fake.Resolve(ctx, c.versionTagFor(3)); err != nil {
		t.Fatalf("expected v3 to exist, got: %v", err)
	}
}

func TestWorkspaceTagFor_HashesInvalidWorkspaceNames(t *testing.T) {
	// Valid tag remains unchanged.
	if got := workspaceTagFor("default"); got != "default" {
		t.Fatalf("expected tag-safe workspace to remain unchanged, got %q", got)
	}

	// Invalid tag is hashed.
	got := workspaceTagFor("my workspace")
	if got == "my workspace" {
		t.Fatalf("expected invalid workspace name to be hashed")
	}
	if len(got) < 3 || got[:3] != "ws-" {
		t.Fatalf("expected hashed workspaceTag to start with ws-, got %q", got)
	}
}

type deleteUnsupportedRepo struct {
	inner *fakeORASRepo
}

func (r deleteUnsupportedRepo) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	return r.inner.Push(ctx, expected, content)
}
func (r deleteUnsupportedRepo) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	return r.inner.Fetch(ctx, target)
}
func (r deleteUnsupportedRepo) Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error) {
	return r.inner.Resolve(ctx, reference)
}
func (r deleteUnsupportedRepo) Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error {
	return r.inner.Tag(ctx, desc, reference)
}
func (r deleteUnsupportedRepo) Delete(ctx context.Context, target ocispec.Descriptor) error {
	_ = ctx
	_ = target
	return &orasErrcode.ErrorResponse{StatusCode: 405}
}
func (r deleteUnsupportedRepo) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	return r.inner.Exists(ctx, target)
}
func (r deleteUnsupportedRepo) Tags(ctx context.Context, last string, fn func(tags []string) error) error {
	return r.inner.Tags(ctx, last, fn)
}

func TestRemoteClient_UnlockFallbackWhenDeleteUnsupported(t *testing.T) {
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: deleteUnsupportedRepo{inner: fake}}

	client := newRemoteClient(repo, "default")

	// Take a lock.
	_, err := client.Lock(&statemgr.LockInfo{ID: "lock-1", Operation: "test"})
	if err != nil {
		t.Fatalf("expected lock to succeed, got: %v", err)
	}

	// Unlock should fallback to retagging rather than failing.
	if err := client.Unlock("lock-1"); err != nil {
		t.Fatalf("expected unlock to succeed via fallback, got: %v", err)
	}

	// After unlock, it should be possible to lock again.
	_, err = client.Lock(&statemgr.LockInfo{ID: "lock-2", Operation: "test"})
	if err != nil {
		t.Fatalf("expected lock after fallback unlock to succeed, got: %v", err)
	}
}

func TestRemoteClient_LockTTL_ClearsStaleLock(t *testing.T) {
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: fake}

	client1 := newRemoteClient(repo, "default")
	client2 := newRemoteClient(repo, "default")
	client2.lockTTL = time.Hour
	client2.now = func() time.Time { return time.Unix(10_000, 0).UTC() }

	staleCreated := time.Unix(1_000, 0).UTC()
	_, err := client1.Lock(&statemgr.LockInfo{ID: "lock-stale", Operation: "test", Created: staleCreated})
	if err != nil {
		t.Fatalf("expected first lock to succeed, got: %v", err)
	}

	_, err = client2.Lock(&statemgr.LockInfo{ID: "lock-new", Operation: "test"})
	if err != nil {
		t.Fatalf("expected lock to succeed after clearing stale lock, got: %v", err)
	}
}

func TestRemoteClient_LockTTL_ClearsStaleLock_DeleteUnsupportedFallback(t *testing.T) {
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: deleteUnsupportedRepo{inner: fake}}

	client1 := newRemoteClient(repo, "default")
	client2 := newRemoteClient(repo, "default")
	client2.lockTTL = time.Hour
	client2.now = func() time.Time { return time.Unix(10_000, 0).UTC() }

	staleCreated := time.Unix(1_000, 0).UTC()
	_, err := client1.Lock(&statemgr.LockInfo{ID: "lock-stale", Operation: "test", Created: staleCreated})
	if err != nil {
		t.Fatalf("expected first lock to succeed, got: %v", err)
	}

	_, err = client2.Lock(&statemgr.LockInfo{ID: "lock-new", Operation: "test"})
	if err != nil {
		t.Fatalf("expected lock to succeed after clearing stale lock via fallback, got: %v", err)
	}
}

func TestRemoteClient_StateCompression_GzipRoundTrip(t *testing.T) {
	ctx := context.Background()
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: fake}

	c := newRemoteClient(repo, "default")
	c.stateCompression = "gzip"

	original := []byte(strings.Repeat("hello-", 2000))

	if diags := c.Put(original); diags.HasErrors() {
		t.Fatalf("put: %v", diags.Err())
	}

	p, getDiags := c.Get()
	if getDiags.HasErrors() {
		t.Fatalf("get: %v", getDiags.Err())
	}
	if p == nil {
		t.Fatalf("expected payload")
	}
	if !bytes.Equal(p.Data, original) {
		t.Fatalf("expected roundtrip to match")
	}

	m, err := c.fetchManifest(ctx, c.stateTag)
	if err != nil {
		t.Fatalf("fetch manifest: %v", err)
	}
	if len(m.Layers) != 1 {
		t.Fatalf("expected 1 layer, got %d", len(m.Layers))
	}
	if m.Layers[0].MediaType != mediaTypeStateLayerGzip {
		t.Fatalf("expected gzip mediaType, got %q", m.Layers[0].MediaType)
	}
}

func TestRemoteClient_StateCompression_AutoDetectOnRead(t *testing.T) {
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: fake}

	writer := newRemoteClient(repo, "default")
	writer.stateCompression = "gzip"

	original := []byte(strings.Repeat("abc", 4096))
	if diags := writer.Put(original); diags.HasErrors() {
		t.Fatalf("put: %v", diags.Err())
	}

	reader := newRemoteClient(repo, "default")
	// reader.stateCompression defaults to "none", but it should still read
	// the compressed state via mediaType autodetection.
	if got, getDiags := reader.Get(); getDiags.HasErrors() {
		t.Fatalf("get: %v", getDiags.Err())
	} else if got == nil {
		t.Fatalf("expected payload")
	} else if !bytes.Equal(got.Data, original) {
		t.Fatalf("expected payload to match original, got len=%d want len=%d", len(got.Data), len(original))
	}
}

func TestRemoteClient_Get_StrictRejectsUnknownLayerMediaType(t *testing.T) {
	ctx := context.Background()
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: fake}

	c := newRemoteClient(repo, "default")

	// Create a state manifest that points to a layer with an unknown media type.
	layerDesc, err := oraslib.PushBytes(ctx, repo.inner, "application/vnd.terraform.statefile.v1+weird", []byte("junk"))
	if err != nil {
		t.Fatalf("push layer: %v", err)
	}

	manifestDesc, err := oraslib.PackManifest(ctx, repo.inner, oraslib.PackManifestVersion1_1, artifactTypeState, oraslib.PackManifestOptions{
		Layers: []ocispec.Descriptor{layerDesc},
		ManifestAnnotations: map[string]string{
			annotationWorkspace: "default",
			annotationUpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		},
	})
	if err != nil {
		t.Fatalf("pack manifest: %v", err)
	}
	if err := repo.inner.Tag(ctx, manifestDesc, c.stateTag); err != nil {
		t.Fatalf("tag manifest: %v", err)
	}

	if _, diags := c.Get(); !diags.HasErrors() {
		t.Fatalf("expected Get to fail for unknown layer media type")
	}
}

func TestRemoteClient_Get_StrictRejectsWrongArtifactType(t *testing.T) {
	ctx := context.Background()
	fake := newFakeORASRepo()
	repo := &orasRepositoryClient{inner: fake}

	c := newRemoteClient(repo, "default")

	// Create a manifest under the state tag but with the lock artifact type.
	layerDesc, err := oraslib.PushBytes(ctx, repo.inner, mediaTypeStateLayer, []byte("s1"))
	if err != nil {
		t.Fatalf("push layer: %v", err)
	}

	manifestDesc, err := oraslib.PackManifest(ctx, repo.inner, oraslib.PackManifestVersion1_1, artifactTypeLock, oraslib.PackManifestOptions{
		Layers: []ocispec.Descriptor{layerDesc},
		ManifestAnnotations: map[string]string{
			annotationWorkspace: "default",
			annotationUpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		},
	})
	if err != nil {
		t.Fatalf("pack manifest: %v", err)
	}
	if err := repo.inner.Tag(ctx, manifestDesc, c.stateTag); err != nil {
		t.Fatalf("tag manifest: %v", err)
	}

	if _, diags := c.Get(); !diags.HasErrors() {
		t.Fatalf("expected Get to fail for wrong artifact type")
	}
}
