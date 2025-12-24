package oras

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sort"
	"sync"

	digest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/errdef"
)

// fakeORASRepo is an in-memory OCI repository used by unit tests.
//
// It implements the minimal subset of the ORAS repository interface we use in
// this package (see orasRepository).
type fakeORASRepo struct {
	mu sync.Mutex

	blobs map[digest.Digest][]byte
	tags  map[string]ocispec.Descriptor
}

func newFakeORASRepo() *fakeORASRepo {
	return &fakeORASRepo{
		blobs: make(map[digest.Digest][]byte),
		tags:  make(map[string]ocispec.Descriptor),
	}
}

func (r *fakeORASRepo) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	_ = ctx

	b, err := io.ReadAll(content)
	if err != nil {
		return err
	}

	got := digest.FromBytes(b)
	if expected.Digest != "" && expected.Digest != got {
		return fmt.Errorf("digest mismatch: expected %s, got %s", expected.Digest, got)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	key := got
	if expected.Digest != "" {
		key = expected.Digest
	}
	r.blobs[key] = b
	return nil
}

func (r *fakeORASRepo) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	_ = ctx

	r.mu.Lock()
	b, ok := r.blobs[target.Digest]
	r.mu.Unlock()

	if !ok {
		return nil, errdef.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (r *fakeORASRepo) Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error) {
	_ = ctx

	r.mu.Lock()
	if d, ok := r.tags[reference]; ok {
		r.mu.Unlock()
		return d, nil
	}
	r.mu.Unlock()

	// Support resolving by digest in case a test needs it.
	if dgst, err := digest.Parse(reference); err == nil {
		r.mu.Lock()
		b, ok := r.blobs[dgst]
		r.mu.Unlock()
		if !ok {
			return ocispec.Descriptor{}, errdef.ErrNotFound
		}
		return ocispec.Descriptor{Digest: dgst, Size: int64(len(b))}, nil
	}

	return ocispec.Descriptor{}, errdef.ErrNotFound
}

func (r *fakeORASRepo) Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error {
	_ = ctx
	if reference == "" {
		return fmt.Errorf("tag must not be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.tags[reference] = desc
	return nil
}

func (r *fakeORASRepo) Delete(ctx context.Context, target ocispec.Descriptor) error {
	_ = ctx

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.blobs, target.Digest)
	for tag, desc := range r.tags {
		if desc.Digest == target.Digest {
			delete(r.tags, tag)
		}
	}
	return nil
}

func (r *fakeORASRepo) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	_ = ctx

	r.mu.Lock()
	_, ok := r.blobs[target.Digest]
	r.mu.Unlock()
	return ok, nil
}

func (r *fakeORASRepo) Tags(ctx context.Context, last string, fn func(tags []string) error) error {
	_ = ctx

	r.mu.Lock()
	tags := make([]string, 0, len(r.tags))
	for tag := range r.tags {
		tags = append(tags, tag)
	}
	r.mu.Unlock()

	sort.Strings(tags)

	start := 0
	if last != "" {
		// Start after the last tag.
		for i, t := range tags {
			if t == last {
				start = i + 1
				break
			}
		}
	}

	return fn(tags[start:])
}
