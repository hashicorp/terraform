// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/zclconf/go-cty/cty"
)

type PolicyCallbackManager struct {
	WalkOperation walkOperation
	Schema        providers.GetProviderSchemaResponse
	Config        *configs.Config
}

func NewPolicyCallbackManager(walkOperation walkOperation, schema providers.GetProviderSchemaResponse, config *configs.Config) *PolicyCallbackManager {
	return &PolicyCallbackManager{
		WalkOperation: walkOperation,
		Schema:        schema,
		Config:        config,
	}
}

// GetResourcesByIdentifierCallback fetches only the subject and resources in its
// emitted relationship tree, from the immutable snapshot of the completed walk.
func (cb *PolicyCallbackManager) GetResourcesByIdentifierCallback(ctx EvalContext, subject addrs.AbsResourceInstance, relations []policy.Relation) func(context.Context, []string) ([]*proto.GetResourcesByIdentifierResponse_Resource, error) {
	graph := ctx.PolicyState()
	allowed := make(map[string]*policyResource)
	bind := func(identifier string) {
		// Keep an unavailable entry distinct from an identifier never emitted.
		allowed[identifier] = nil
		if graph != nil {
			if resource := graph.resourceMap[identifier]; resource != nil {
				snapshot := *resource
				allowed[identifier] = &snapshot
			}
		}
	}
	bind(subject.String())
	var addRelations func([]policy.Relation)
	addRelations = func(relations []policy.Relation) {
		for _, relation := range relations {
			for identifier, nested := range relation.Resources {
				bind(identifier)
				addRelations(nested)
			}
		}
	}
	addRelations(relations)

	return func(callbackCtx context.Context, identifiers []string) ([]*proto.GetResourcesByIdentifierResponse_Resource, error) {
		if err := callbackCtx.Err(); err != nil {
			return nil, err
		}
		resources := make([]*proto.GetResourcesByIdentifierResponse_Resource, 0, len(identifiers))
		for _, identifier := range identifiers {
			if err := callbackCtx.Err(); err != nil {
				return nil, err
			}
			resource, emitted := allowed[identifier]
			if !emitted {
				return nil, fmt.Errorf("policy resource identifier %q was not sent for %s", identifier, subject)
			}
			if resource == nil {
				return nil, fmt.Errorf("policy resource identifier %q is unavailable", identifier)
			}
			addr := resource.Addr

			action := resource.Action
			// Related replacements refer to the object left after the run.
			if action.IsReplace() {
				action = plans.Create
			}
			operation, ok := policyOperationForAction(action)
			if !ok {
				return nil, fmt.Errorf("unsupported policy resource action %q for %s", action, identifier)
			}
			attrs, err := policy.ResourceAttributesToProto(resource.Value)
			if err != nil {
				return nil, fmt.Errorf("failed to serialize attributes for %s: %w", identifier, err)
			}
			prior := resource.PriorValue
			if prior.Raw == cty.NilVal {
				// Match EvaluateResource's normalization of absent prior attributes.
				prior = policy.CtyToPolicyValue(cty.EmptyObjectVal)
			}
			priorAttrs, err := policy.ResourceAttributesToProto(prior)
			if err != nil {
				return nil, fmt.Errorf("failed to serialize prior attributes for %s: %w", identifier, err)
			}
			resources = append(resources, &proto.GetResourcesByIdentifierResponse_Resource{
				Identifier: identifier,
				Type:       addr.Resource.Resource.Type,
				Metadata: &proto.ResourceMetadata{
					ProviderType: resource.ProviderAddr.Provider.Type,
					Operation:    operation,
					ModulePath:   addr.Module.String(),
					Address:      identifier,
				},
				Attrs:      attrs,
				PriorAttrs: priorAttrs,
			})
		}
		return resources, nil
	}
}
