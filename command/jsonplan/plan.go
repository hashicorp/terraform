package jsonplan

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/jsonconfig"
	"github.com/hashicorp/terraform/command/jsonstate"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/version"
)

// FormatVersion represents the version of the json format and will be
// incremented for any change to this format that requires changes to a
// consuming parser.
const FormatVersion = "0.1"

// Plan is the top-level representation of the json format of a plan. It includes
// the complete config and current state.
type plan struct {
	FormatVersion    string      `json:"format_version,omitempty"`
	TerraformVersion string      `json:"terraform_version,omitempty"`
	Variables        variables   `json:"variables,omitempty"`
	PlannedValues    stateValues `json:"planned_values,omitempty"`
	// ResourceChanges are sorted in a user-friendly order that is undefined at
	// this time, but consistent.
	ResourceChanges []resourceChange  `json:"resource_changes,omitempty"`
	OutputChanges   map[string]change `json:"output_changes,omitempty"`
	PriorState      json.RawMessage   `json:"prior_state,omitempty"`
	Config          json.RawMessage   `json:"configuration,omitempty"`
}

func newPlan() *plan {
	return &plan{
		FormatVersion: FormatVersion,
	}
}

// Change is the representation of a proposed change for an object.
type change struct {
	// Actions are the actions that will be taken on the object selected by the
	// properties below. Valid actions values are:
	//    ["no-op"]
	//    ["create"]
	//    ["read"]
	//    ["update"]
	//    ["delete", "create"]
	//    ["create", "delete"]
	//    ["delete"]
	// The two "replace" actions are represented in this way to allow callers to
	// e.g. just scan the list for "delete" to recognize all three situations
	// where the object will be deleted, allowing for any new deletion
	// combinations that might be added in future.
	Actions []string `json:"actions,omitempty"`

	// Before and After are representations of the object value both before and
	// after the action. For ["create"] and ["delete"] actions, either "before"
	// or "after" is unset (respectively). For ["no-op"], the before and after
	// values are identical. The "after" value will be incomplete if there are
	// values within it that won't be known until after apply.
	Before       json.RawMessage `json:"before,omitempty"`
	After        json.RawMessage `json:"after,omitempty"`
	AfterUnknown json.RawMessage `json:"after_unknown,omitempty"`
}

type output struct {
	Sensitive bool            `json:"sensitive"`
	Value     json.RawMessage `json:"value,omitempty"`
}

// variables is the JSON representation of the variables provided to the current
// plan.
type variables map[string]*variable

type variable struct {
	Value json.RawMessage `json:"value,omitempty"`
}

// Marshal returns the json encoding of a terraform plan.
func Marshal(
	config *configs.Config,
	p *plans.Plan,
	sf *statefile.File,
	schemas *terraform.Schemas,
) ([]byte, error) {
	output := newPlan()
	output.TerraformVersion = version.String()

	err := output.marshalPlanVariables(p.VariableValues, schemas)
	if err != nil {
		return nil, fmt.Errorf("error in marshalPlanVariables: %s", err)
	}

	// output.PlannedValues
	err = output.marshalPlannedValues(p.Changes, schemas)
	if err != nil {
		return nil, fmt.Errorf("error in marshalPlannedValues: %s", err)
	}

	// output.ResourceChanges
	err = output.marshalResourceChanges(p.Changes, schemas)
	if err != nil {
		return nil, fmt.Errorf("error in marshalResourceChanges: %s", err)
	}

	// output.OutputChanges
	err = output.marshalOutputChanges(p.Changes)
	if err != nil {
		return nil, fmt.Errorf("error in marshaling output changes: %s", err)
	}

	// output.PriorState
	if sf != nil && !sf.State.Empty() {
		output.PriorState, err = jsonstate.Marshal(sf, schemas)
		if err != nil {
			return nil, fmt.Errorf("error marshaling prior state: %s", err)
		}
	}

	// output.Config
	output.Config, err = jsonconfig.Marshal(config, schemas)
	if err != nil {
		return nil, fmt.Errorf("error marshaling config: %s", err)
	}

	ret, err := json.Marshal(output)
	return ret, err
}

func (p *plan) marshalPlanVariables(vars map[string]plans.DynamicValue, schemas *terraform.Schemas) error {
	if len(vars) == 0 {
		return nil
	}

	p.Variables = make(variables, len(vars))

	for k, v := range vars {
		val, err := v.Decode(cty.DynamicPseudoType)
		if err != nil {
			return err
		}
		valJSON, err := ctyjson.Marshal(val, val.Type())
		if err != nil {
			return err
		}
		p.Variables[k] = &variable{
			Value: valJSON,
		}
	}
	return nil
}

func (p *plan) marshalResourceChanges(changes *plans.Changes, schemas *terraform.Schemas) error {
	if changes == nil {
		// Nothing to do!
		return nil
	}
	for _, rc := range changes.Resources {
		var r resourceChange
		addr := rc.Addr
		r.Address = addr.String()

		dataSource := addr.Resource.Resource.Mode == addrs.DataResourceMode
		// We create "delete" actions for data resources so we can clean up
		// their entries in state, but this is an implementation detail that
		// users shouldn't see.
		if dataSource && rc.Action == plans.Delete {
			continue
		}

		schema, _ := schemas.ResourceTypeConfig(
			rc.ProviderAddr.ProviderConfig.Type.LegacyString(),
			addr.Resource.Resource.Mode,
			addr.Resource.Resource.Type,
		)
		if schema == nil {
			return fmt.Errorf("no schema found for %s", r.Address)
		}

		changeV, err := rc.Decode(schema.ImpliedType())
		if err != nil {
			return err
		}

		var before, after []byte
		var afterUnknown cty.Value
		if changeV.Before != cty.NilVal {
			before, err = ctyjson.Marshal(changeV.Before, changeV.Before.Type())
			if err != nil {
				return err
			}
		}
		if changeV.After != cty.NilVal {
			if changeV.After.IsWhollyKnown() {
				after, err = ctyjson.Marshal(changeV.After, changeV.After.Type())
				if err != nil {
					return err
				}
				afterUnknown = cty.EmptyObjectVal
			} else {
				filteredAfter := omitUnknowns(changeV.After)
				if filteredAfter.IsNull() {
					after = nil
				} else {
					after, err = ctyjson.Marshal(filteredAfter, filteredAfter.Type())
					if err != nil {
						return err
					}
				}
				afterUnknown = unknownAsBool(changeV.After)
			}
		}

		a, err := ctyjson.Marshal(afterUnknown, afterUnknown.Type())
		if err != nil {
			return err
		}

		r.Change = change{
			Actions:      actionString(rc.Action.String()),
			Before:       json.RawMessage(before),
			After:        json.RawMessage(after),
			AfterUnknown: a,
		}

		if rc.DeposedKey != states.NotDeposed {
			r.Deposed = rc.DeposedKey.String()
		}

		key := addr.Resource.Key
		if key != nil {
			r.Index = key
		}

		switch addr.Resource.Resource.Mode {
		case addrs.ManagedResourceMode:
			r.Mode = "managed"
		case addrs.DataResourceMode:
			r.Mode = "data"
		default:
			return fmt.Errorf("resource %s has an unsupported mode %s", r.Address, addr.Resource.Resource.Mode.String())
		}
		r.ModuleAddress = addr.Module.String()
		r.Name = addr.Resource.Resource.Name
		r.Type = addr.Resource.Resource.Type
		r.ProviderName = rc.ProviderAddr.ProviderConfig.StringCompact()

		p.ResourceChanges = append(p.ResourceChanges, r)

	}

	sort.Slice(p.ResourceChanges, func(i, j int) bool {
		return p.ResourceChanges[i].Address < p.ResourceChanges[j].Address
	})

	return nil
}

func (p *plan) marshalOutputChanges(changes *plans.Changes) error {
	if changes == nil {
		// Nothing to do!
		return nil
	}

	p.OutputChanges = make(map[string]change, len(changes.Outputs))
	for _, oc := range changes.Outputs {
		changeV, err := oc.Decode()
		if err != nil {
			return err
		}

		var before, after []byte
		afterUnknown := cty.False
		if changeV.Before != cty.NilVal {
			before, err = ctyjson.Marshal(changeV.Before, changeV.Before.Type())
			if err != nil {
				return err
			}
		}
		if changeV.After != cty.NilVal {
			if changeV.After.IsWhollyKnown() {
				after, err = ctyjson.Marshal(changeV.After, changeV.After.Type())
				if err != nil {
					return err
				}
			} else {
				afterUnknown = cty.True
			}
		}

		a, _ := ctyjson.Marshal(afterUnknown, afterUnknown.Type())

		c := change{
			Actions:      actionString(oc.Action.String()),
			Before:       json.RawMessage(before),
			After:        json.RawMessage(after),
			AfterUnknown: a,
		}

		p.OutputChanges[oc.Addr.OutputValue.Name] = c
	}

	return nil
}

func (p *plan) marshalPlannedValues(changes *plans.Changes, schemas *terraform.Schemas) error {
	// marshal the planned changes into a module
	plan, err := marshalPlannedValues(changes, schemas)
	if err != nil {
		return err
	}
	p.PlannedValues.RootModule = plan

	// marshalPlannedOutputs
	outputs, err := marshalPlannedOutputs(changes)
	if err != nil {
		return err
	}
	p.PlannedValues.Outputs = outputs

	return nil
}

// omitUnknowns recursively walks the src cty.Value and returns a new cty.Value,
// omitting any unknowns.
//
// The result also normalizes some types: all sequence types are turned into
// tuple types and all mapping types are converted to object types, since we
// assume the result of this is just going to be serialized as JSON (and thus
// lose those distinctions) anyway.
func omitUnknowns(val cty.Value) cty.Value {
	ty := val.Type()
	switch {
	case val.IsNull():
		return val
	case !val.IsKnown():
		return cty.NilVal
	case ty.IsPrimitiveType():
		return val
	case ty.IsListType() || ty.IsTupleType() || ty.IsSetType():
		var vals []cty.Value
		it := val.ElementIterator()
		for it.Next() {
			_, v := it.Element()
			newVal := omitUnknowns(v)
			if newVal != cty.NilVal {
				vals = append(vals, newVal)
			} else if newVal == cty.NilVal && ty.IsListType() {
				// list length may be significant, so we will turn unknowns into nulls
				vals = append(vals, cty.NullVal(v.Type()))
			}
		}
		// We use tuple types always here, because the work we did above
		// may have caused the individual elements to have different types,
		// and we're doing this work to produce JSON anyway and JSON marshalling
		// represents all of these sequence types as an array.
		return cty.TupleVal(vals)
	case ty.IsMapType() || ty.IsObjectType():
		vals := make(map[string]cty.Value)
		it := val.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			newVal := omitUnknowns(v)
			if newVal != cty.NilVal {
				vals[k.AsString()] = newVal
			}
		}
		// We use object types always here, because the work we did above
		// may have caused the individual elements to have different types,
		// and we're doing this work to produce JSON anyway and JSON marshalling
		// represents both of these mapping types as an object.
		return cty.ObjectVal(vals)
	default:
		// Should never happen, since the above should cover all types
		panic(fmt.Sprintf("omitUnknowns cannot handle %#v", val))
	}
}

// recursively iterate through a cty.Value, replacing unknown values (including
// null) with cty.True and known values with cty.False.
//
// The result also normalizes some types: all sequence types are turned into
// tuple types and all mapping types are converted to object types, since we
// assume the result of this is just going to be serialized as JSON (and thus
// lose those distinctions) anyway.
func unknownAsBool(val cty.Value) cty.Value {
	ty := val.Type()
	switch {
	case val.IsNull():
		return cty.False
	case !val.IsKnown():
		if ty.IsPrimitiveType() || ty.Equals(cty.DynamicPseudoType) {
			return cty.True
		}
		fallthrough
	case ty.IsPrimitiveType():
		return cty.BoolVal(!val.IsKnown())
	case ty.IsListType() || ty.IsTupleType() || ty.IsSetType():
		length := val.LengthInt()
		if length == 0 {
			// If there are no elements then we can't have unknowns
			return cty.EmptyTupleVal
		}
		vals := make([]cty.Value, 0, length)
		it := val.ElementIterator()
		for it.Next() {
			_, v := it.Element()
			vals = append(vals, unknownAsBool(v))
		}
		// The above transform may have changed the types of some of the
		// elements, so we'll always use a tuple here in case we've now made
		// different elements have different types. Our ultimate goal is to
		// marshal to JSON anyway, and all of these sequence types are
		// indistinguishable in JSON.
		return cty.TupleVal(vals)
	case ty.IsMapType() || ty.IsObjectType():
		var length int
		switch {
		case ty.IsMapType():
			length = val.LengthInt()
		default:
			length = len(val.Type().AttributeTypes())
		}
		if length == 0 {
			// If there are no elements then we can't have unknowns
			return cty.EmptyObjectVal
		}
		vals := make(map[string]cty.Value)
		it := val.ElementIterator()
		for it.Next() {
			k, v := it.Element()
			vAsBool := unknownAsBool(v)
			if !vAsBool.RawEquals(cty.False) { // all of the "false"s for known values for more compact serialization
				vals[k.AsString()] = unknownAsBool(v)
			}
		}
		// The above transform may have changed the types of some of the
		// elements, so we'll always use an object here in case we've now made
		// different elements have different types. Our ultimate goal is to
		// marshal to JSON anyway, and all of these mapping types are
		// indistinguishable in JSON.
		return cty.ObjectVal(vals)
	default:
		// Should never happen, since the above should cover all types
		panic(fmt.Sprintf("unknownAsBool cannot handle %#v", val))
	}
}

func actionString(action string) []string {
	switch {
	case action == "NoOp":
		return []string{"no-op"}
	case action == "Create":
		return []string{"create"}
	case action == "Delete":
		return []string{"delete"}
	case action == "Update":
		return []string{"update"}
	case action == "CreateThenDelete":
		return []string{"create", "delete"}
	case action == "Read":
		return []string{"read"}
	case action == "DeleteThenCreate":
		return []string{"delete", "create"}
	default:
		return []string{action}
	}
}
