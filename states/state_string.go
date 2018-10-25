package states

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/hcl2shim"
)

// String returns a rather-odd string representation of the entire state.
//
// This is intended to match the behavior of the older terraform.State.String
// method that is used in lots of existing tests. It should not be used in
// new tests: instead, use "cmp" to directly compare the state data structures
// and print out a diff if they do not match.
//
// This method should never be used in non-test code, whether directly by call
// or indirectly via a %s or %q verb in package fmt.
func (s *State) String() string {
	if s == nil {
		return "<nil>"
	}

	// sort the modules by name for consistent output
	modules := make([]string, 0, len(s.Modules))
	for m := range s.Modules {
		modules = append(modules, m)
	}
	sort.Strings(modules)

	var buf bytes.Buffer
	for _, name := range modules {
		m := s.Modules[name]
		mStr := m.testString()

		// If we're the root module, we just write the output directly.
		if m.Addr.IsRoot() {
			buf.WriteString(mStr + "\n")
			continue
		}

		// We need to build out a string that resembles the not-quite-standard
		// format that terraform.State.String used to use, where there's a
		// "module." prefix but then just a chain of all of the module names
		// without any further "module." portions.
		buf.WriteString("module")
		for _, step := range m.Addr {
			buf.WriteByte('.')
			buf.WriteString(step.Name)
			if step.InstanceKey != addrs.NoKey {
				buf.WriteByte('[')
				buf.WriteString(step.InstanceKey.String())
				buf.WriteByte(']')
			}
		}
		buf.WriteString(":\n")

		s := bufio.NewScanner(strings.NewReader(mStr))
		for s.Scan() {
			text := s.Text()
			if text != "" {
				text = "  " + text
			}

			buf.WriteString(fmt.Sprintf("%s\n", text))
		}
	}

	return strings.TrimSpace(buf.String())
}

// testString is used to produce part of the output of State.String. It should
// never be used directly.
func (m *Module) testString() string {
	var buf bytes.Buffer

	if len(m.Resources) == 0 {
		buf.WriteString("<no state>")
	}

	// We use AbsResourceInstance here, even though everything belongs to
	// the same module, just because we have a sorting behavior defined
	// for those but not for just ResourceInstance.
	addrsOrder := make([]addrs.AbsResourceInstance, 0, len(m.Resources))
	for _, rs := range m.Resources {
		for ik := range rs.Instances {
			addrsOrder = append(addrsOrder, rs.Addr.Instance(ik).Absolute(addrs.RootModuleInstance))
		}
	}

	sort.Slice(addrsOrder, func(i, j int) bool {
		return addrsOrder[i].Less(addrsOrder[j])
	})

	for _, fakeAbsAddr := range addrsOrder {
		addr := fakeAbsAddr.Resource
		rs := m.Resource(addr.ContainingResource())
		is := m.ResourceInstance(addr)

		// Here we need to fake up a legacy-style address as the old state
		// types would've used, since that's what our tests against those
		// old types expect. The significant difference is that instancekey
		// is dot-separated rather than using index brackets.
		k := addr.ContainingResource().String()
		if addr.Key != addrs.NoKey {
			switch tk := addr.Key.(type) {
			case addrs.IntKey:
				k = fmt.Sprintf("%s.%d", k, tk)
			default:
				// No other key types existed for the legacy types, so we
				// can do whatever we want here. We'll just use our standard
				// syntax for these.
				k = k + tk.String()
			}
		}

		id := LegacyInstanceObjectID(is.Current)

		taintStr := ""
		if is.Current != nil && is.Current.Status == ObjectTainted {
			taintStr = " (tainted)"
		}

		deposedStr := ""
		if len(is.Deposed) > 0 {
			deposedStr = fmt.Sprintf(" (%d deposed)", len(is.Deposed))
		}

		buf.WriteString(fmt.Sprintf("%s:%s%s\n", k, taintStr, deposedStr))
		buf.WriteString(fmt.Sprintf("  ID = %s\n", id))
		buf.WriteString(fmt.Sprintf("  provider = %s\n", rs.ProviderConfig.String()))

		// Attributes were a flatmap before, but are not anymore. To preserve
		// our old output as closely as possible we need to do a conversion
		// to flatmap. Normally we'd want to do this with schema for
		// accuracy, but for our purposes here it only needs to be approximate.
		// This should produce an identical result for most cases, though
		// in particular will differ in a few cases:
		//  - The keys used for elements in a set will be different
		//  - Values for attributes of type cty.DynamicPseudoType will be
		//    misinterpreted (but these weren't possible in old world anyway)
		var attributes map[string]string
		if obj := is.Current; obj != nil {
			switch {
			case obj.AttrsFlat != nil:
				// Easy (but increasingly unlikely) case: the state hasn't
				// actually been upgraded to the new form yet.
				attributes = obj.AttrsFlat
			case obj.AttrsJSON != nil:
				ty, err := ctyjson.ImpliedType(obj.AttrsJSON)
				if err == nil {
					val, err := ctyjson.Unmarshal(obj.AttrsJSON, ty)
					if err == nil {
						attributes = hcl2shim.FlatmapValueFromHCL2(val)
					}
				}
			}
		}
		attrKeys := make([]string, 0, len(attributes))
		for ak, _ := range attributes {
			if ak == "id" {
				continue
			}

			attrKeys = append(attrKeys, ak)
		}

		sort.Strings(attrKeys)

		for _, ak := range attrKeys {
			av := attributes[ak]
			buf.WriteString(fmt.Sprintf("  %s = %s\n", ak, av))
		}

		// CAUTION: Since deposed keys are now random strings instead of
		// incrementing integers, this result will not be deterministic
		// if there is more than one deposed object.
		i := 1
		for _, t := range is.Deposed {
			id := LegacyInstanceObjectID(t)
			taintStr := ""
			if t.Status == ObjectTainted {
				taintStr = " (tainted)"
			}
			buf.WriteString(fmt.Sprintf("  Deposed ID %d = %s%s\n", i, id, taintStr))
			i++
		}

		if obj := is.Current; obj != nil && len(obj.Dependencies) > 0 {
			buf.WriteString(fmt.Sprintf("\n  Dependencies:\n"))
			for _, dep := range obj.Dependencies {
				buf.WriteString(fmt.Sprintf("    %s\n", dep.String()))
			}
		}
	}

	if len(m.OutputValues) > 0 {
		buf.WriteString("\nOutputs:\n\n")

		ks := make([]string, 0, len(m.OutputValues))
		for k := range m.OutputValues {
			ks = append(ks, k)
		}
		sort.Strings(ks)

		for _, k := range ks {
			v := m.OutputValues[k]
			lv := hcl2shim.ConfigValueFromHCL2(v.Value)
			switch vTyped := lv.(type) {
			case string:
				buf.WriteString(fmt.Sprintf("%s = %s\n", k, vTyped))
			case []interface{}:
				buf.WriteString(fmt.Sprintf("%s = %s\n", k, vTyped))
			case map[string]interface{}:
				var mapKeys []string
				for key := range vTyped {
					mapKeys = append(mapKeys, key)
				}
				sort.Strings(mapKeys)

				var mapBuf bytes.Buffer
				mapBuf.WriteString("{")
				for _, key := range mapKeys {
					mapBuf.WriteString(fmt.Sprintf("%s:%s ", key, vTyped[key]))
				}
				mapBuf.WriteString("}")

				buf.WriteString(fmt.Sprintf("%s = %s\n", k, mapBuf.String()))
			default:
				buf.WriteString(fmt.Sprintf("%s = %#v\n", k, lv))
			}
		}
	}

	return buf.String()
}

// LegacyInstanceObjectID is a helper for extracting an object id value from
// an instance object in a way that approximates how we used to do this
// for the old state types. ID is no longer first-class, so this is preserved
// only for compatibility with old tests that include the id as part of their
// expected value.
func LegacyInstanceObjectID(obj *ResourceInstanceObjectSrc) string {
	if obj == nil {
		return "<not created>"
	}

	if obj.AttrsJSON != nil {
		type WithID struct {
			ID string `json:"id"`
		}
		var withID WithID
		err := json.Unmarshal(obj.AttrsJSON, &withID)
		if err == nil {
			return withID.ID
		}
	} else if obj.AttrsFlat != nil {
		if flatID, exists := obj.AttrsFlat["id"]; exists {
			return flatID
		}
	}

	// For resource types created after we removed id as special there may
	// not actually be one at all. This is okay because older tests won't
	// encounter this, and new tests shouldn't be using ids.
	return "<none>"
}
