package terraform

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// UpgradeResourceState will, if necessary, run the provider-defined upgrade
// logic against the given state object to make it compliant with the
// current schema version. This is a no-op if the given state object is
// already at the latest version.
//
// If any errors occur during upgrade, error diagnostics are returned. In that
// case it is not safe to proceed with using the original state object.
func UpgradeResourceState(addr addrs.AbsResourceInstance, provider providers.Interface, src *states.ResourceInstanceObjectSrc, currentSchema *configschema.Block, currentVersion uint64) (*states.ResourceInstanceObjectSrc, tfdiags.Diagnostics) {
	// Remove any attributes from state that are not present in the schema.
	// This was previously taken care of by the provider, but data sources do
	// not go through the UpgradeResourceState process.
	//
	// Legacy flatmap state is already taken care of during conversion.
	// If the schema version is be changed, then allow the provider to handle
	// removed attributes.
	if len(src.AttrsJSON) > 0 && src.SchemaVersion == currentVersion {
		src.AttrsJSON = stripRemovedStateAttributes(src.AttrsJSON, currentSchema.ImpliedType())
	}

	if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
		// We only do state upgrading for managed resources.
		return src, nil
	}

	stateIsFlatmap := len(src.AttrsJSON) == 0

	// TODO: This should eventually use a proper FQN.
	providerType := addr.Resource.Resource.ImpliedProvider()
	if src.SchemaVersion > currentVersion {
		log.Printf("[TRACE] UpgradeResourceState: can't downgrade state for %s from version %d to %d", addr, src.SchemaVersion, currentVersion)
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Resource instance managed by newer provider version",
			// This is not a very good error message, but we don't retain enough
			// information in state to give good feedback on what provider
			// version might be required here. :(
			fmt.Sprintf("The current state of %s was created by a newer provider version than is currently selected. Upgrade the %s provider to work with this state.", addr, providerType),
		))
		return nil, diags
	}

	// If we get down here then we need to upgrade the state, with the
	// provider's help.
	// If this state was originally created by a version of Terraform prior to
	// v0.12, this also includes translating from legacy flatmap to new-style
	// representation, since only the provider has enough information to
	// understand a flatmap built against an older schema.
	if src.SchemaVersion != currentVersion {
		log.Printf("[TRACE] UpgradeResourceState: upgrading state for %s from version %d to %d using provider %q", addr, src.SchemaVersion, currentVersion, providerType)
	} else {
		log.Printf("[TRACE] UpgradeResourceState: schema version of %s is still %d; calling provider %q for any other minor fixups", addr, currentVersion, providerType)
	}

	req := providers.UpgradeResourceStateRequest{
		TypeName: addr.Resource.Resource.Type,

		// TODO: The internal schema version representations are all using
		// uint64 instead of int64, but unsigned integers aren't friendly
		// to all protobuf target languages so in practice we use int64
		// on the wire. In future we will change all of our internal
		// representations to int64 too.
		Version: int64(src.SchemaVersion),
	}

	if stateIsFlatmap {
		req.RawStateFlatmap = src.AttrsFlat
	} else {
		req.RawStateJSON = src.AttrsJSON
	}

	resp := provider.UpgradeResourceState(req)
	diags := resp.Diagnostics
	if diags.HasErrors() {
		return nil, diags
	}

	// After upgrading, the new value must conform to the current schema. When
	// going over RPC this is actually already ensured by the
	// marshaling/unmarshaling of the new value, but we'll check it here
	// anyway for robustness, e.g. for in-process providers.
	newValue := resp.UpgradedState
	if errs := newValue.Type().TestConformance(currentSchema.ImpliedType()); len(errs) > 0 {
		for _, err := range errs {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid resource state upgrade",
				fmt.Sprintf("The %s provider upgraded the state for %s from a previous version, but produced an invalid result: %s.", providerType, addr, tfdiags.FormatError(err)),
			))
		}
		return nil, diags
	}

	new, err := src.CompleteUpgrade(newValue, currentSchema.ImpliedType(), uint64(currentVersion))
	if err != nil {
		// We already checked for type conformance above, so getting into this
		// codepath should be rare and is probably a bug somewhere under CompleteUpgrade.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to encode result of resource state upgrade",
			fmt.Sprintf("Failed to encode state for %s after resource schema upgrade: %s.", addr, tfdiags.FormatError(err)),
		))
	}
	return new, diags
}

// stripRemovedStateAttributes deletes any attributes no longer present in the
// schema, so that the json can be correctly decoded.
func stripRemovedStateAttributes(state []byte, ty cty.Type) []byte {
	jsonMap := map[string]interface{}{}
	err := json.Unmarshal(state, &jsonMap)
	if err != nil {
		// we just log any errors here, and let the normal decode process catch
		// invalid JSON.
		log.Printf("[ERROR] UpgradeResourceState: %s", err)
		return state
	}

	// if no changes were made, we return the original state to ensure nothing
	// was altered in the marshaling process.
	if !removeRemovedAttrs(jsonMap, ty) {
		return state
	}

	js, err := json.Marshal(jsonMap)
	if err != nil {
		// if the json map was somehow mangled enough to not marhsal, something
		// went horribly wrong
		panic(err)
	}

	return js
}

// strip out the actual missing attributes, and return a bool indicating if any
// changes were made.
func removeRemovedAttrs(v interface{}, ty cty.Type) bool {
	modified := false
	// we're only concerned with finding maps that correspond to object
	// attributes
	switch v := v.(type) {
	case []interface{}:
		switch {
		// If these aren't blocks the next call will be a noop
		case ty.IsListType() || ty.IsSetType():
			eTy := ty.ElementType()
			for _, eV := range v {
				modified = removeRemovedAttrs(eV, eTy) || modified
			}
		}
		return modified
	case map[string]interface{}:
		switch {
		case ty.IsMapType():
			// map blocks aren't yet supported, but handle this just in case
			eTy := ty.ElementType()
			for _, eV := range v {
				modified = removeRemovedAttrs(eV, eTy) || modified
			}
			return modified

		case ty == cty.DynamicPseudoType:
			log.Printf("[DEBUG] UpgradeResourceState: ignoring dynamic block: %#v\n", v)
			return false

		case ty.IsObjectType():
			attrTypes := ty.AttributeTypes()
			for attr, attrV := range v {
				attrTy, ok := attrTypes[attr]
				if !ok {
					log.Printf("[DEBUG] UpgradeResourceState: attribute %q no longer present in schema", attr)
					delete(v, attr)
					modified = true
					continue
				}

				modified = removeRemovedAttrs(attrV, attrTy) || modified
			}
			return modified
		default:
			// This shouldn't happen, and will fail to decode further on, so
			// there's no need to handle it here.
			log.Printf("[WARN] UpgradeResourceState: unexpected type %#v for map in json state", ty)
			return false
		}
	}
	return modified
}
