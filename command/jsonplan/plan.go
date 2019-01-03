package jsonplan

import (
	"encoding/json"
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/jsonconfig"
	"github.com/hashicorp/terraform/command/jsonstate"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"

	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// FormatVersion represents the version of the json format and will be
// incremented for any change to this format that requires changes to a
// consuming parser.
const FormatVersion = "0.1"

// Plan is the top-level representation of the json format of a plan. It includes
// the complete config and current state.
type plan struct {
	FormatVersion   string            `json:"format_version,omitempty"`
	PlannedValues   stateValues       `json:"planned_values,omitempty"`
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
	Sensitive bool            `json:"sensitive,omitempty"`
	Value     json.RawMessage `json:"value,omitempty"`
}

// Marshal returns the json encoding of a terraform plan.
func Marshal(
	config *configs.Config,
	p *plans.Plan,
	s *states.State,
	schemas *terraform.Schemas,
) ([]byte, error) {

	output := newPlan()

	// output.PlannedValues
	err := output.marshalPlannedValues(p.Changes, schemas)
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
	output.PriorState, err = jsonstate.Marshal(s, schemas)
	if err != nil {
		return nil, fmt.Errorf("error marshaling prior state: %s", err)
	}

	// output.Config
	output.Config, err = jsonconfig.Marshal(config, schemas)
	if err != nil {
		return nil, fmt.Errorf("error marshaling config: %s", err)
	}

	// add some polish
	ret, err := json.MarshalIndent(output, "", "  ")
	return ret, err
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

		schema, _ := schemas.ResourceTypeConfig(rc.ProviderAddr.ProviderConfig.StringCompact(), addr.Resource.Resource.Mode, addr.Resource.Resource.Type)
		if schema == nil {
			return fmt.Errorf("no schema found for %s", r.Address)
		}

		changeV, err := rc.Decode(schema.ImpliedType())
		if err != nil {
			return err
		}

		var before, after []byte
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
			}
		}

		afterUnknown, _ := cty.Transform(changeV.After, func(path cty.Path, val cty.Value) (cty.Value, error) {
			if val.IsNull() {
				return cty.False, nil
			}

			if !val.Type().IsPrimitiveType() {
				return val, nil // just pass through non-primitives; they already contain our transform results
			}

			if val.IsKnown() {
				// null rather than false here so that known values
				// don't appear at all in JSON serialization of our result
				return cty.False, nil
			}

			return cty.True, nil
		})

		a, _ := ctyjson.Marshal(afterUnknown, afterUnknown.Type())

		r.Change = change{
			Actions:      []string{rc.Action.String()},
			Before:       json.RawMessage(before),
			After:        json.RawMessage(after),
			AfterUnknown: a,
		}

		r.Deposed = rc.DeposedKey == states.NotDeposed

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

		p.ResourceChanges = append(p.ResourceChanges, r)

	}

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
			Actions:      []string{oc.Action.String()},
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
