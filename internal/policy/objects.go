// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/zclconf/go-cty/cty"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

// objectTable maps the identifiers that the client hands out to their
// resource objects, each encoded once, at first use, so that an identifier
// always stands for the same values (contract §3.4). It lives as long as the
// client, and its zero value is ready to use.
type objectTable struct {
	// identifiers is whether the plugin accepts identifiers for the matches
	// of getresources (the getresources_identifiers server capability).
	identifiers atomic.Bool
	generation  atomic.Uint64
	entries     sync.Map // identifier -> *objectEntry
}

type objectEntry struct {
	once     sync.Once
	resource *proto.GetResourcesByIdentifierResponse_Resource
	err      error
}

var _ callback.Objects = (*objectTable)(nil)

// NextGeneration starts a policy phase, and returns its generation for the
// objects of the phase.
func (t *objectTable) NextGeneration() uint64 {
	return t.generation.Add(1)
}

// RegisterObject returns the identifier of an object, and encodes the object
// unless an object with the same identifier is already registered, which
// stands for the same values. It fails if the encoding fails.
func (t *objectTable) RegisterObject(obj *callback.Object) (string, error) {
	_, id, err := t.register(obj)
	return id, err
}

// register is RegisterObject, and also returns the encoding of the object.
// encoded is false for a memo hit.
func (t *objectTable) register(obj *callback.Object) (e *objectEntry, id string, err error) {
	id = obj.Identifier()
	v, _ := t.entries.LoadOrStore(id, &objectEntry{})
	e = v.(*objectEntry)
	e.once.Do(func() {
		e.resource, e.err = encodeObject(id, obj)
	})
	return e, id, e.err
}

// encodeObject encodes the values of an object, with the paths its schema
// declares sensitive redacted.
func encodeObject(id string, obj *callback.Object) (*proto.GetResourcesByIdentifierResponse_Resource, error) {
	ret := &proto.GetResourcesByIdentifierResponse_Resource{
		Identifier: id,
		Type:       obj.Type,
		Metadata:   obj.Metadata,
	}
	var err error
	if obj.Attrs != cty.NilVal {
		if ret.Attrs, err = resourceAttributesToProto(markSchemaSensitive(CtyToPolicyValue(obj.Attrs), obj.Schema)); err != nil {
			return nil, err
		}
	}
	if obj.PriorAttrs != cty.NilVal {
		if ret.PriorAttrs, err = resourceAttributesToProto(markSchemaSensitive(CtyToPolicyValue(obj.PriorAttrs), obj.Schema)); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

// markSchemaSensitive adds the paths that the schema declares sensitive to
// the redacted paths of v, in the order CtyToPolicyValue gives them.
func markSchemaSensitive(v PolicyValue, block *configschema.Block) PolicyValue {
	paths := v.RedactedPaths
	if block != nil {
		paths = slices.Concat(paths, block.SensitivePaths(v.Raw, nil))
	}
	var pvms []cty.PathValueMarks
	for _, path := range paths {
		pvms = append(pvms, cty.PathValueMarks{Path: path, Marks: cty.NewValueMarks(marks.Sensitive)})
	}
	return CtyToPolicyValue(v.Raw.MarkWithPaths(pvms))
}

// GetResourcesResponse implements callback.Objects: the matches by
// identifier if the plugin accepts identifiers, and their values otherwise.
func (t *objectTable) GetResourcesResponse(ctx context.Context, resources []callback.Resource, partial bool) (*proto.GetResourcesResponse, error) {
	resp := &proto.GetResourcesResponse{Partial: partial}
	if !t.identifiers.Load() {
		for _, r := range resources {
			attrs, err := resourceAttributesToProto(CtyToPolicyValue(r.Value))
			if err != nil {
				return nil, fmt.Errorf("failed to serialize resource: %w", err)
			}
			resp.Results = append(resp.Results, attrs.Raw)
		}
		return resp, nil
	}
	for _, r := range resources {
		if r.Object == nil {
			return nil, fmt.Errorf("no object for a match of type %q", r.Value.Type().FriendlyName())
		}
		id, err := t.RegisterObject(r.Object)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize resource %s: %w", r.Object.Address, err)
		}
		resp.Identifiers = append(resp.Identifiers, id)
	}
	return resp, nil
}

// GetResourcesByIdentifier implements callback.Objects: the objects of the
// given identifiers, in order. An identifier that the client didn't hand out
// in this session is an error diagnostic, and then the response has no
// resources.
func (t *objectTable) GetResourcesByIdentifier(ctx context.Context, identifiers []string) *proto.GetResourcesByIdentifierResponse {
	trace.SpanFromContext(ctx).SetAttributes(attribute.Int("policy.objects.requested", len(identifiers)))
	resp := &proto.GetResourcesByIdentifierResponse{}
	for _, id := range identifiers {
		v, ok := t.entries.Load(id)
		if ok {
			e := v.(*objectEntry)
			// An identifier is handed out only after its entry is encoded,
			// so this only waits for an encoding in progress.
			e.once.Do(func() {})
			if e.resource != nil {
				resp.Resources = append(resp.Resources, e.resource)
				continue
			}
		}
		resp.Diagnostics = append(resp.Diagnostics, &proto.Diagnostic{
			Severity: proto.Severity_ERROR,
			Summary:  "Unknown resource identifier",
			Detail:   fmt.Sprintf("Terraform has no resource with the identifier %q in this policy session.", id),
		})
	}
	if len(resp.Diagnostics) > 0 {
		resp.Resources = nil
	}
	return resp
}
