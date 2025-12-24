// Package oras implements a Terraform backend using OCI registries via ORAS.
// See README.md for configuration options and wire format details.
package oras

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	oras "oras.land/oras-go/v2"
	"oras.land/oras-go/v2/errdef"
	orasRegistry "oras.land/oras-go/v2/registry"
	orasErrcode "oras.land/oras-go/v2/registry/remote/errcode"
)

const (
	mediaTypeStateLayer     = "application/vnd.terraform.statefile.v1"
	mediaTypeStateLayerGzip = "application/vnd.terraform.statefile.v1+gzip"
	artifactTypeState       = "application/vnd.terraform.state.v1"
	artifactTypeLock        = "application/vnd.terraform.lock.v1"

	annotationWorkspace = "org.terraform.workspace"
	// annotationUpdatedAt changes on every Put so registries can retain distinct version tags.
	annotationUpdatedAt = "org.terraform.state.updated_at"
	annotationLockID    = "org.terraform.lock.id"
	annotationLockInfo  = "org.terraform.lock.info"
)

type RetryConfig struct {
	MaxAttempts       int
	InitialBackoff    time.Duration
	MaxBackoff        time.Duration
	BackoffMultiplier float64
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

func withRetry[T any](ctx context.Context, cfg RetryConfig, operation func(ctx context.Context) (T, error)) (T, error) {
	var zero T
	var lastErr error

	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}

	backoff := cfg.InitialBackoff
	if backoff <= 0 {
		backoff = time.Second
	}

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		result, err := operation(ctx)
		if err == nil {
			return result, nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		if !isTransientError(err) {
			return zero, err
		}
		if attempt == cfg.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(backoff):
		}

		backoff = time.Duration(float64(backoff) * cfg.BackoffMultiplier)
		if cfg.MaxBackoff > 0 && backoff > cfg.MaxBackoff {
			backoff = cfg.MaxBackoff
		}
	}

	return zero, lastErr
}

func withRetryNoResult(ctx context.Context, cfg RetryConfig, operation func(ctx context.Context) error) error {
	_, err := withRetry(ctx, cfg, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, operation(ctx)
	})
	return err
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	var errResp *orasErrcode.ErrorResponse
	if errors.As(err, &errResp) {
		switch errResp.StatusCode {
		case http.StatusTooManyRequests,
			http.StatusRequestTimeout,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		}
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout() || netErr.Temporary()
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return dnsErr.Temporary()
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Temporary() || opErr.Timeout()
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	// Fallback to string matching for wrapped errors that lost their type info
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "eof")
}

const (
	stateTagPrefix           = "state-"
	lockTagPrefix            = "locked-"
	unlockedTagPrefix        = "unlocked-" // for registries that don't support DELETE (looking at you, GHCR)
	stateVersionTagSeparator = "-v"
)

type RemoteClient struct {
	repo             *orasRepositoryClient
	workspaceName    string
	stateTag         string
	lockTag          string
	unlockedTag      string
	retryConfig      RetryConfig
	stateCompression string
	lockTTL          time.Duration
	now              func() time.Time

	versioningEnabled     bool
	versioningMaxVersions int
}

var _ remote.Client = (*RemoteClient)(nil)
var _ remote.ClientLocker = (*RemoteClient)(nil)

func newRemoteClient(repo *orasRepositoryClient, workspaceName string) *RemoteClient {
	wsTag := workspaceTagFor(workspaceName)
	return &RemoteClient{
		repo:                  repo,
		workspaceName:         workspaceName,
		stateTag:              stateTagPrefix + wsTag,
		lockTag:               lockTagPrefix + wsTag,
		unlockedTag:           unlockedTagPrefix + wsTag,
		retryConfig:           DefaultRetryConfig(),
		stateCompression:      "none",
		lockTTL:               0,
		now:                   time.Now,
		versioningEnabled:     false,
		versioningMaxVersions: 0,
	}
}

func (c *RemoteClient) packStateManifest(ctx context.Context, layers []ocispec.Descriptor) (ocispec.Descriptor, error) {
	return oras.PackManifest(ctx, c.repo.inner, oras.PackManifestVersion1_1, artifactTypeState, oras.PackManifestOptions{
		Layers: layers,
		ManifestAnnotations: map[string]string{
			annotationWorkspace: c.workspaceName,
			annotationUpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		},
	})
}

func (c *RemoteClient) packLockManifest(ctx context.Context, id, infoJSON string) (ocispec.Descriptor, error) {
	return oras.PackManifest(ctx, c.repo.inner, oras.PackManifestVersion1_1, artifactTypeLock, oras.PackManifestOptions{
		ManifestAnnotations: map[string]string{
			annotationWorkspace: c.workspaceName,
			annotationLockID:    id,
			annotationLockInfo:  infoJSON,
		},
	})
}

func (c *RemoteClient) Get() (*remote.Payload, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ctx := context.Background()
	ret, err := withRetry(ctx, c.retryConfig, func(ctx context.Context) (*remote.Payload, error) {
		return c.get(ctx)
	})
	if err != nil {
		return nil, diags.Append(err)
	}
	return ret, diags
}

func (c *RemoteClient) get(ctx context.Context) (*remote.Payload, error) {
	m, err := c.fetchManifest(ctx, c.stateTag)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if m.ArtifactType != artifactTypeState {
		return nil, fmt.Errorf("unexpected state manifest artifactType %q for %q", m.ArtifactType, c.stateTag)
	}
	if len(m.Layers) == 0 {
		return nil, nil
	}

	layer := m.Layers[0]
	rc, err := c.repo.inner.Fetch(ctx, layer)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	var r io.Reader = rc
	switch layer.MediaType {
	case mediaTypeStateLayer:
		// uncompressed, nothing to do
	case mediaTypeStateLayerGzip:
		gz, err := gzip.NewReader(rc)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		r = gz
	default:
		return nil, fmt.Errorf("unsupported state layer media type %q", layer.MediaType)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	md5sum := md5.Sum(data)
	return &remote.Payload{MD5: md5sum[:], Data: data}, nil
}

func (c *RemoteClient) Put(state []byte) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	ctx := context.Background()
	err := withRetryNoResult(ctx, c.retryConfig, func(ctx context.Context) error {
		return c.put(ctx, state)
	})
	return diags.Append(err)
}

func (c *RemoteClient) put(ctx context.Context, state []byte) error {
	stateToPush := state
	layerMediaType := mediaTypeStateLayer

	if c.stateCompression == "gzip" {
		compressed, err := compressGzip(state)
		if err != nil {
			return fmt.Errorf("compressing state: %w", err)
		}
		stateToPush = compressed
		layerMediaType = mediaTypeStateLayerGzip
	}

	layerDesc, err := oras.PushBytes(ctx, c.repo.inner, layerMediaType, stateToPush)
	if err != nil {
		return err
	}

	manifestDesc, err := c.packStateManifest(ctx, []ocispec.Descriptor{layerDesc})
	if err != nil {
		return err
	}

	if err := c.repo.inner.Tag(ctx, manifestDesc, c.stateTag); err != nil {
		return err
	}

	if !c.versioningEnabled {
		return nil
	}

	nextVersion, existing, err := c.nextStateVersion(ctx)
	if err != nil {
		return err
	}

	newVersionTag := c.versionTagFor(nextVersion)
	if err := c.repo.inner.Tag(ctx, manifestDesc, newVersionTag); err != nil {
		return err
	}

	if c.versioningMaxVersions > 0 {
		existing = append(existing, nextVersion)
		if err := c.enforceVersionRetention(ctx, manifestDesc, existing); err != nil {
			return err
		}
	}

	return nil
}

func (c *RemoteClient) versionTagFor(version int) string {
	return fmt.Sprintf("%s%s%d", c.stateTag, stateVersionTagSeparator, version)
}

func (c *RemoteClient) nextStateVersion(ctx context.Context) (next int, existing []int, err error) {
	var tags []string
	if err := c.repo.inner.Tags(ctx, "", func(page []string) error {
		tags = append(tags, page...)
		return nil
	}); err != nil {
		return 0, nil, err
	}

	max := 0
	for _, t := range tags {
		base, v, ok := splitStateVersionTag(t)
		if !ok || base != c.stateTag {
			continue
		}
		existing = append(existing, v)
		if v > max {
			max = v
		}
	}

	return max + 1, existing, nil
}

// enforceVersionRetention prunes old versions, handling the edge case where
// multiple version tags may point to the same digest (content-addressable storage...).
func (c *RemoteClient) enforceVersionRetention(ctx context.Context, current ocispec.Descriptor, versions []int) error {
	if c.versioningMaxVersions <= 0 || len(versions) <= c.versioningMaxVersions {
		return nil
	}

	sort.Ints(versions)
	toDeleteCount := len(versions) - c.versioningMaxVersions
	deleteVersions := versions[:toDeleteCount]
	keepVersions := versions[toDeleteCount:]

	// Build tag sets for quick lookup
	deleteTagSet := make(map[string]struct{}, len(deleteVersions))
	keepTagSet := make(map[string]struct{}, len(keepVersions))
	for _, v := range deleteVersions {
		deleteTagSet[c.versionTagFor(v)] = struct{}{}
	}
	for _, v := range keepVersions {
		keepTagSet[c.versionTagFor(v)] = struct{}{}
	}

	// Group tags by digest to handle multiple tags per manifest
	groups := c.groupVersionsByDigest(ctx, versions, current.Digest)
	if len(groups) == 0 {
		return nil
	}

	logger := logging.HCLogger().Named("backend.oras")

	for _, g := range groups {
		tagsToDelete, tagsToKeep := classifyTags(g.tags, deleteTagSet, keepTagSet)
		if len(tagsToDelete) == 0 {
			continue
		}

		// Handle mixed keep/delete tags on same digest
		if len(tagsToKeep) > 0 {
			if err := c.retagToNewManifest(ctx, tagsToKeep, logger); err != nil {
				return err
			}
		}

		// Delete the digest (now safe since keep tags are moved)
		if err := c.deleteDigestWithFallback(ctx, g.desc, tagsToDelete[0]); err != nil {
			return err
		}
	}

	return nil
}

type digestGroup struct {
	desc ocispec.Descriptor
	tags []string
}

func (c *RemoteClient) groupVersionsByDigest(ctx context.Context, versions []int, currentDigest digest.Digest) map[string]*digestGroup {
	groups := make(map[string]*digestGroup)
	for _, v := range versions {
		tag := c.versionTagFor(v)
		desc, err := c.repo.inner.Resolve(ctx, tag)
		if err != nil || desc.Digest == currentDigest {
			continue
		}
		key := desc.Digest.String()
		if g, ok := groups[key]; ok {
			g.tags = append(g.tags, tag)
		} else {
			groups[key] = &digestGroup{desc: desc, tags: []string{tag}}
		}
	}
	return groups
}

func classifyTags(tags []string, deleteSet, keepSet map[string]struct{}) (toDelete, toKeep []string) {
	for _, tag := range tags {
		if _, ok := deleteSet[tag]; ok {
			toDelete = append(toDelete, tag)
		} else if _, ok := keepSet[tag]; ok {
			toKeep = append(toKeep, tag)
		}
	}
	return
}

// retagToNewManifest moves tags to a fresh manifest so we can delete the old digest.
func (c *RemoteClient) retagToNewManifest(ctx context.Context, tags []string, logger interface{ Debug(string, ...interface{}) }) error {
	if len(tags) == 0 {
		return nil
	}
	logger.Debug("retention: detaching keep tags from digest", "tags", tags)

	m, err := c.fetchManifest(ctx, tags[0])
	if err != nil {
		return err
	}
	if len(m.Layers) == 0 {
		return nil // No content to retain
	}

	newDesc, err := c.packStateManifest(ctx, m.Layers)
	if err != nil {
		return err
	}
	for _, tag := range tags {
		if err := c.repo.inner.Tag(ctx, newDesc, tag); err != nil {
			return err
		}
	}
	return nil
}

func (c *RemoteClient) deleteDigestWithFallback(ctx context.Context, desc ocispec.Descriptor, fallbackTag string) error {
	err := c.repo.inner.Delete(ctx, desc)
	if err == nil || isNotFound(err) {
		return nil
	}
	if !isDeleteUnsupported(err) {
		return err
	}
	// GHCR fallback
	if ghErr := tryDeleteGHCRTag(ctx, c.repo, fallbackTag); ghErr != nil {
		return fmt.Errorf("oras backend retention: registry does not support OCI manifest deletion and GHCR API deletion failed for %q: %w", fallbackTag, ghErr)
	}
	return nil
}

func (c *RemoteClient) Delete() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	ctx := context.Background()
	err := withRetryNoResult(ctx, c.retryConfig, func(ctx context.Context) error {
		return c.delete(ctx)
	})
	return diags.Append(err)
}

func (c *RemoteClient) delete(ctx context.Context) error {
	desc, err := c.repo.inner.Resolve(ctx, c.stateTag)
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return err
	}
	return c.repo.inner.Delete(ctx, desc)
}

func (c *RemoteClient) Lock(info *statemgr.LockInfo) (string, error) {
	// Lock contention errors are not retried.
	return c.lock(context.Background(), info)
}

func (c *RemoteClient) lock(ctx context.Context, info *statemgr.LockInfo) (string, error) {
	if info == nil {
		return "", fmt.Errorf("lock info is required")
	}

	// Check for existing lock (with retry for transient network errors)
	existingDesc, err := withRetry(ctx, c.retryConfig, func(ctx context.Context) (ocispec.Descriptor, error) {
		return c.repo.inner.Resolve(ctx, c.lockTag)
	})
	if err == nil {
		existing, err := c.getLockInfo(ctx)
		if err != nil {
			return "", err
		}
		if existing != nil && existing.ID != "" {
			if c.isLockStale(existing) {
				if err := c.clearLock(ctx, existingDesc); err != nil {
					return "", err
				}
			} else {
				return "", &statemgr.LockError{Info: existing, Err: fmt.Errorf("state is locked")}
			}
		}
	} else if !isNotFound(err) {
		return "", err
	}

	info.Path = c.stateTag
	infoBytes, err := json.Marshal(info)
	if err != nil {
		return "", err
	}

	manifestDesc, err := c.packLockManifest(ctx, info.ID, string(infoBytes))
	if err != nil {
		return "", err
	}

	// Tag with retry for transient network errors
	err = withRetryNoResult(ctx, c.retryConfig, func(ctx context.Context) error {
		return c.repo.inner.Tag(ctx, manifestDesc, c.lockTag)
	})
	if err != nil {
		if _, resolveErr := c.repo.inner.Resolve(ctx, c.lockTag); resolveErr == nil {
			existing, _ := c.getLockInfo(ctx)
			return "", &statemgr.LockError{Info: existing, Err: fmt.Errorf("state is locked")}
		}
		return "", err
	}

	return info.ID, nil
}

func (c *RemoteClient) isLockStale(info *statemgr.LockInfo) bool {
	if c.lockTTL <= 0 || info == nil || info.Created.IsZero() {
		return false
	}
	now := time.Now
	if c.now != nil {
		now = c.now
	}
	age := now().UTC().Sub(info.Created)
	if age < 0 {
		return false
	}
	return age > c.lockTTL
}

func (c *RemoteClient) clearLock(ctx context.Context, desc ocispec.Descriptor) error {
	// Delete with retry for transient network errors
	err := withRetryNoResult(ctx, c.retryConfig, func(ctx context.Context) error {
		return c.repo.inner.Delete(ctx, desc)
	})
	if err == nil || isNotFound(err) {
		return nil
	}
	if !isDeleteUnsupported(err) {
		return err
	}

	// GHCR fallback: retag to unlocked manifest
	return c.retagToUnlocked(ctx)
}

func (c *RemoteClient) Unlock(id string) error {
	// Lock ID mismatch errors are not retried.
	return c.unlock(context.Background(), id)
}

func (c *RemoteClient) unlock(ctx context.Context, id string) error {
	// Resolve with retry for transient network errors
	desc, err := withRetry(ctx, c.retryConfig, func(ctx context.Context) (ocispec.Descriptor, error) {
		return c.repo.inner.Resolve(ctx, c.lockTag)
	})
	if err != nil {
		if isNotFound(err) {
			return nil
		}
		return err
	}

	existing, err := c.getLockInfo(ctx)
	if err != nil {
		return err
	}
	if existing == nil || existing.ID == "" {
		return nil
	}
	if id != "" && existing.ID != id {
		return fmt.Errorf("lock ID mismatch: held by %q", existing.ID)
	}

	// Delete with retry for transient network errors
	err = withRetryNoResult(ctx, c.retryConfig, func(ctx context.Context) error {
		return c.repo.inner.Delete(ctx, desc)
	})
	if err == nil {
		return nil
	}
	if !isDeleteUnsupported(err) {
		return err
	}

	// GHCR fallback: retag to unlocked manifest
	return c.retagToUnlocked(ctx)
}

func (c *RemoteClient) retagToUnlocked(ctx context.Context) error {
	// Resolve with retry
	desc, err := withRetry(ctx, c.retryConfig, func(ctx context.Context) (ocispec.Descriptor, error) {
		return c.repo.inner.Resolve(ctx, c.unlockedTag)
	})
	if isNotFound(err) {
		desc, err = oras.PackManifest(ctx, c.repo.inner, oras.PackManifestVersion1_1, artifactTypeLock, oras.PackManifestOptions{})
		if err != nil {
			return err
		}
		// Tag with retry
		if err := withRetryNoResult(ctx, c.retryConfig, func(ctx context.Context) error {
			return c.repo.inner.Tag(ctx, desc, c.unlockedTag)
		}); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	// Final tag with retry
	return withRetryNoResult(ctx, c.retryConfig, func(ctx context.Context) error {
		return c.repo.inner.Tag(ctx, desc, c.lockTag)
	})
}

func (c *RemoteClient) getLockInfo(ctx context.Context) (*statemgr.LockInfo, error) {
	m, err := c.fetchManifest(ctx, c.lockTag)
	if err != nil {
		return nil, err
	}
	if m.ArtifactType != artifactTypeLock {
		return nil, fmt.Errorf("unexpected lock manifest artifactType %q for %q", m.ArtifactType, c.lockTag)
	}

	if raw, ok := m.Annotations[annotationLockInfo]; ok && raw != "" {
		var info statemgr.LockInfo
		if err := json.Unmarshal([]byte(raw), &info); err != nil {
			return nil, fmt.Errorf("decoding lock info: %w", err)
		}
		if info.ID == "" {
			info.ID = m.Annotations[annotationLockID]
		}
		if info.Path == "" {
			info.Path = c.stateTag
		}
		return &info, nil
	}

	id := m.Annotations[annotationLockID]
	if id == "" {
		return &statemgr.LockInfo{}, nil
	}
	return &statemgr.LockInfo{ID: id, Path: c.stateTag}, nil
}

type manifest struct {
	ArtifactType string               `json:"artifactType"`
	MediaType    string               `json:"mediaType"`
	Annotations  map[string]string    `json:"annotations"`
	Layers       []ocispec.Descriptor `json:"layers"`
}

func (c *RemoteClient) fetchManifest(ctx context.Context, reference string) (*manifest, error) {
	return withRetry(ctx, c.retryConfig, func(ctx context.Context) (*manifest, error) {
		return c.fetchManifestInternal(ctx, reference)
	})
}

func (c *RemoteClient) fetchManifestInternal(ctx context.Context, reference string) (*manifest, error) {
	data, err := fetchReferenceBytes(ctx, c.repo.inner, reference)
	if err != nil {
		return nil, err
	}

	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("decoding manifest %q: %w", reference, err)
	}
	if m.Annotations == nil {
		m.Annotations = map[string]string{}
	}
	return &m, nil
}

// Workspace tag helpers

func workspaceTagFor(workspace string) string {
	ref := orasRegistry.Reference{Reference: workspace}
	if err := ref.ValidateReferenceAsTag(); err == nil {
		return workspace
	}
	h := sha256.Sum256([]byte(workspace))
	return "ws-" + hex.EncodeToString(h[:8])
}

func listWorkspacesFromTags(repo *orasRepositoryClient) ([]string, error) {
	ctx := context.Background()
	var tags []string
	if err := repo.inner.Tags(ctx, "", func(page []string) error {
		tags = append(tags, page...)
		return nil
	}); err != nil {
		return nil, err
	}

	tagSet := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		tagSet[t] = struct{}{}
	}

	seen := map[string]bool{}
	var out []string
	for _, tag := range tags {
		if !strings.HasPrefix(tag, stateTagPrefix) {
			continue
		}
		if base, _, ok := splitStateVersionTag(tag); ok {
			if _, ok := tagSet[base]; ok {
				continue
			}
		}
		name, err := workspaceNameFromTag(ctx, repo, tag)
		if err != nil {
			return nil, err
		}
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}

func splitStateVersionTag(tag string) (base string, version int, ok bool) {
	idx := strings.LastIndex(tag, stateVersionTagSeparator)
	if idx < 0 {
		return "", 0, false
	}
	base = tag[:idx]
	if base == "" {
		return "", 0, false
	}
	s := tag[idx+len(stateVersionTagSeparator):]
	if s == "" {
		return "", 0, false
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return "", 0, false
	}
	return base, v, true
}

func workspaceNameFromTag(ctx context.Context, repo *orasRepositoryClient, stateTag string) (string, error) {
	wsTag := strings.TrimPrefix(stateTag, stateTagPrefix)
	if !strings.HasPrefix(wsTag, "ws-") {
		return wsTag, nil
	}
	// Hash fallback - need to read annotation
	data, err := fetchReferenceBytes(ctx, repo.inner, stateTag)
	if err != nil {
		return "", err
	}

	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return wsTag, nil
	}
	if name := m.Annotations[annotationWorkspace]; name != "" {
		return name, nil
	}
	return wsTag, nil
}

func fetchReferenceBytes(ctx context.Context, repo orasRepository, reference string) ([]byte, error) {
	desc, err := repo.Resolve(ctx, reference)
	if err != nil {
		return nil, err
	}
	rc, err := repo.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	return io.ReadAll(rc)
}

func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return nil, err
	}
	if _, err := gz.Write(data); err != nil {
		gz.Close() // best effort cleanup
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func isNotFound(err error) bool {
	if errors.Is(err, errdef.ErrNotFound) {
		return true
	}
	var resp *orasErrcode.ErrorResponse
	if errors.As(err, &resp) {
		return resp.StatusCode == 404
	}
	return false
}

func isDeleteUnsupported(err error) bool {
	var resp *orasErrcode.ErrorResponse
	if errors.As(err, &resp) {
		return resp.StatusCode == 405
	}
	return false
}
