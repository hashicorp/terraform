// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ValidatePolicyRelationships validates the policy plugin's relationship
// definitions against the provider schemas of an operation (contract §3.3),
// and returns the relationships without the definitions of providers that
// the operation doesn't use. version returns the version of a provider, or ""
// if it isn't known.
func ValidatePolicyRelationships(rels policy.Relationships, schemas map[addrs.Provider]providers.ProviderSchema, version func(addrs.Provider) string) (policy.Relationships, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	sources := make(map[string]addrs.Provider, len(schemas))
	for provider := range schemas {
		sources[provider.String()] = provider
	}

	ret := policy.Relationships{Usages: rels.Usages}
	for _, def := range rels.Definitions {
		provider, ok := sources[def.ProviderSource]
		if !ok {
			continue
		}
		v := &relationshipValidator{
			def:      def,
			schema:   schemas[provider],
			provider: def.ProviderSource,
		}
		if version := version(provider); version != "" {
			v.provider += " v" + version
		} else {
			v.provider += " (version unknown)"
		}
		defDiags := v.validate()
		diags = diags.Append(defDiags)
		if !defDiags.HasErrors() {
			ret.Definitions = append(ret.Definitions, def)
		}
	}
	return ret, diags
}

type relationshipValidator struct {
	def    *proto.RelationshipDefinition
	schema providers.ProviderSchema
	// provider names the provider and its version.
	provider string
}

func (v *relationshipValidator) validate() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	keyOK := v.checkType(&diags, v.def.KeyResource)
	valueOK := v.checkType(&diags, v.def.ValueResource)
	if !keyOK || !valueOK {
		return diags
	}
	for _, alt := range v.def.Match {
		for _, key := range alt.Keys {
			keyPath, valuePath := policyKeyPathNames(key.KeyPath), policyKeyPathNames(key.ValuePath)
			keyType, keyOK := v.leafType(&diags, v.def.KeyResource, keyPath)
			valueType, valueOK := v.leafType(&diags, v.def.ValueResource, valuePath)
			if keyOK && valueOK && !keyType.Equals(valueType) {
				v.error(&diags, "`%s` of %s is %s, but `%s` of %s is %s, in the schema of %s. Both paths of a key must read the same primitive type.",
					strings.Join(keyPath, "."), v.def.KeyResource, keyType.FriendlyName(),
					strings.Join(valuePath, "."), v.def.ValueResource, valueType.FriendlyName(), v.provider)
			}
		}
	}
	return diags
}

func (v *relationshipValidator) error(diags *tfdiags.Diagnostics, format string, args ...any) {
	var subject *hcl.Range
	if v.def.DeclRange != nil {
		subject = v.def.DeclRange.ToHclRange().Ptr()
	}
	*diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagError,
		Summary:  fmt.Sprintf("Invalid relationship %q", v.def.Name),
		Detail:   fmt.Sprintf(format, args...),
		Subject:  subject,
	})
}

// checkType checks that typ is a managed resource type of the provider.
func (v *relationshipValidator) checkType(diags *tfdiags.Diagnostics, typ string) bool {
	if _, ok := v.schema.ResourceTypes[typ]; ok {
		return true
	}
	switch {
	case v.schema.DataSources[typ].Body != nil:
		v.error(diags, "%s is a data source type of %s, not a managed resource type. Relationships relate managed resources only.", typ, v.provider)
	case v.schema.EphemeralResourceTypes[typ].Body != nil:
		v.error(diags, "%s is an ephemeral resource type of %s, not a managed resource type. Relationships relate managed resources only.", typ, v.provider)
	default:
		v.error(diags, "%s has no resource type %s.", v.provider, typ)
	}
	return false
}

// leafType returns the primitive type that a path of a resource type reads,
// through nested blocks, objects and lists and sets of them (proposal
// §2.2).
func (v *relationshipValidator) leafType(diags *tfdiags.Diagnostics, typ string, path []string) (cty.Type, bool) {
	schema := v.schema.ResourceTypes[typ]
	where := fmt.Sprintf("`%s` of %s", strings.Join(path, "."), typ)
	if schema.Body == nil || len(path) == 0 {
		v.error(diags, "%s doesn't exist in the schema of %s.", where, v.provider)
		return cty.NilType, false
	}
	ty := schema.Body.ImpliedType()
	var ctyPath cty.Path
	for i, name := range path {
		ty = policyElementType(ty)
		switch {
		case ty.IsMapType():
			v.error(diags, "%s steps through `%s`, of type %s, in the schema of %s. A key can't read a map.", where, strings.Join(path[:i], "."), ty.FriendlyName(), v.provider)
			return cty.NilType, false
		case ty == cty.DynamicPseudoType:
			v.error(diags, "%s steps through `%s`, which is dynamically typed in the schema of %s. A key must read a typed attribute.", where, strings.Join(path[:i], "."), v.provider)
			return cty.NilType, false
		case ty.IsObjectType() && ty.HasAttribute(name):
			ty = ty.AttributeType(name)
		default:
			v.error(diags, "%s doesn't exist in the schema of %s.", where, v.provider)
			return cty.NilType, false
		}
		ctyPath = ctyPath.GetAttr(name)
		if attr := schema.Body.AttributeByPath(ctyPath); attr != nil && attr.WriteOnly {
			v.error(diags, "%s is write-only, of type %s, in the schema of %s, so it's always null in the plan and the state.", where, ty.FriendlyName(), v.provider)
			return cty.NilType, false
		}
	}

	leaf := policyElementType(ty)
	switch {
	case leaf.IsMapType():
		v.error(diags, "%s has type %s in the schema of %s. A key can't read a map.", where, ty.FriendlyName(), v.provider)
	case leaf == cty.DynamicPseudoType:
		v.error(diags, "%s is dynamically typed in the schema of %s. A key must read a typed attribute.", where, v.provider)
	case leaf.IsPrimitiveType():
		return leaf, true
	default:
		v.error(diags, "%s has type %s in the schema of %s. A key must read a primitive or a collection of primitives.", where, ty.FriendlyName(), v.provider)
	}
	return cty.NilType, false
}

// policyElementType returns the type of the elements of lists and sets, at
// any depth.
func policyElementType(ty cty.Type) cty.Type {
	for ty.IsListType() || ty.IsSetType() {
		ty = ty.ElementType()
	}
	return ty
}
