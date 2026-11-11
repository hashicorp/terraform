// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

// valueTable maps the value IDs that the client hands out to their values,
// each encoded once, at first use, so that an ID always stands for the same
// bytes. It lives as long as the client, and its zero value is ready to use.
type valueTable struct {
	// ids is whether the plugin resolves value IDs (the resource_value_ids
	// server capability).
	ids        atomic.Bool
	generation atomic.Uint64
	entries    sync.Map // ID -> *valueEntry
}

type valueEntry struct {
	once  sync.Once
	attrs *proto.ResourceAttributes // with raw, redacted_paths and id
	err   error
}

var _ callback.Values = (*valueTable)(nil)

// attributes builds a ResourceAttributes from a value and its source, which
// is nil if the value has none. Every ResourceAttributes the client sends is
// built here. Without the resource_value_ids capability, a value is inline
// with no ID, as before value IDs; with it, a value with a source carries its
// ID, and only the ID if byRef. encode encodes the value; it's called once
// per source. encoded is false for a memo hit.
func (t *valueTable) attributes(src *callback.ValueSource, byRef bool, encode func() (*proto.ResourceAttributes, error)) (attrs *proto.ResourceAttributes, encoded bool, err error) {
	if src == nil {
		attrs, err = encode()
		return attrs, true, err
	}
	id := src.ID()
	v, _ := t.entries.LoadOrStore(id, &valueEntry{})
	e := v.(*valueEntry)
	e.once.Do(func() {
		encoded = true
		e.attrs, e.err = encode()
		if e.attrs == nil && e.err == nil {
			e.err = fmt.Errorf("no value")
		}
		if e.err == nil {
			e.attrs.Id = id
		} else {
			e.attrs = nil
		}
	})
	switch {
	case e.err != nil:
		return nil, encoded, e.err
	case !t.ids.Load():
		return &proto.ResourceAttributes{Raw: e.attrs.Raw, RedactedPaths: e.attrs.RedactedPaths}, encoded, nil
	case byRef:
		return &proto.ResourceAttributes{Id: id}, encoded, nil
	default:
		return e.attrs, encoded, nil
	}
}

// NextGeneration starts a policy phase, and returns its generation for the
// value sources of the phase.
func (t *valueTable) NextGeneration() uint64 {
	return t.generation.Add(1)
}

// ResourceAttributes encodes the value of a related member.
func (t *valueTable) ResourceAttributes(value cty.Value, src callback.ValueSource) (*proto.ResourceAttributes, error) {
	attrs, _, err := t.valueAttributes(value, &src)
	return attrs, err
}

// valueAttributes is attributes for a cty.Value, by reference.
func (t *valueTable) valueAttributes(value cty.Value, src *callback.ValueSource) (*proto.ResourceAttributes, bool, error) {
	return t.attributes(src, true, func() (*proto.ResourceAttributes, error) {
		return EncodeResourceAttributes(value)
	})
}

// GetResourcesResponse implements callback.Values: without the capability,
// the results as before value IDs; with it, the values by reference.
func (t *valueTable) GetResourcesResponse(ctx context.Context, resources []callback.Resource, partial bool) (*proto.GetResourcesResponse, error) {
	resp := &proto.GetResourcesResponse{Partial: partial}
	var encodes int
	for _, r := range resources {
		attrs, encoded, err := t.valueAttributes(r.Value, &r.Source)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize resource: %w", err)
		}
		if encoded {
			encodes++
		}
		if attrs.Id != "" {
			resp.Values = append(resp.Values, attrs)
		} else {
			resp.Results = append(resp.Results, attrs.Raw)
		}
	}
	trace.SpanFromContext(ctx).SetAttributes(
		attribute.Int("policy.values.encodes", encodes),
		attribute.Int("policy.values.memo_hits", len(resources)-encodes),
	)
	return resp, nil
}

// GetResourceValues implements callback.Values: the values of the given IDs,
// in order. An ID that the client didn't hand out is NOT_FOUND.
func (t *valueTable) GetResourceValues(ctx context.Context, ids []string) ([]*proto.ResourceAttributes, error) {
	trace.SpanFromContext(ctx).SetAttributes(attribute.Int("policy.values.requested", len(ids)))
	ret := make([]*proto.ResourceAttributes, len(ids))
	for i, id := range ids {
		v, ok := t.entries.Load(id)
		if !ok {
			return nil, status.Errorf(codes.NotFound, "no resource value with ID %q", id)
		}
		e := v.(*valueEntry)
		// An ID is handed out only after its entry is encoded, so this only
		// waits for an encoding in progress.
		e.once.Do(func() {})
		if e.attrs == nil {
			return nil, status.Errorf(codes.NotFound, "no resource value with ID %q", id)
		}
		ret[i] = e.attrs
	}
	return ret, nil
}
