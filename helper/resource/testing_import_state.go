package resource

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
)

// testStepImportState runs an imort state test step
func testStepImportState(
	opts terraform.ContextOpts,
	state *terraform.State,
	step TestStep) (*terraform.State, error) {

	// Determine the ID to import
	var importId string
	switch {
	case step.ImportStateIdFunc != nil:
		var err error
		importId, err = step.ImportStateIdFunc(state)
		if err != nil {
			return state, err
		}
	case step.ImportStateId != "":
		importId = step.ImportStateId
	default:
		resource, err := testResource(step, state)
		if err != nil {
			return state, err
		}
		importId = resource.Primary.ID
	}

	importPrefix := step.ImportStateIdPrefix
	if importPrefix != "" {
		importId = fmt.Sprintf("%s%s", importPrefix, importId)
	}

	// Setup the context. We initialize with an empty state. We use the
	// full config for provider configurations.
	cfg, err := testConfig(opts, step)
	if err != nil {
		return state, err
	}

	opts.Config = cfg

	// import tests start with empty state
	opts.State = states.NewState()

	ctx, stepDiags := terraform.NewContext(&opts)
	if stepDiags.HasErrors() {
		return state, stepDiags.Err()
	}

	// The test step provides the resource address as a string, so we need
	// to parse it to get an addrs.AbsResourceAddress to pass in to the
	// import method.
	traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(step.ResourceName), "", hcl.Pos{})
	if hclDiags.HasErrors() {
		return nil, hclDiags
	}
	importAddr, stepDiags := addrs.ParseAbsResourceInstance(traversal)
	if stepDiags.HasErrors() {
		return nil, stepDiags.Err()
	}

	// Do the import
	importedState, stepDiags := ctx.Import(&terraform.ImportOpts{
		// Set the module so that any provider config is loaded
		Config: cfg,

		Targets: []*terraform.ImportTarget{
			&terraform.ImportTarget{
				Addr: importAddr,
				ID:   importId,
			},
		},
	})
	if stepDiags.HasErrors() {
		log.Printf("[ERROR] Test: ImportState failure: %s", stepDiags.Err())
		return state, stepDiags.Err()
	}

	newState, err := shimNewState(importedState, step.providers)
	if err != nil {
		return nil, err
	}

	// Go through the new state and verify
	if step.ImportStateCheck != nil {
		var states []*terraform.InstanceState
		for _, r := range newState.RootModule().Resources {
			if r.Primary != nil {
				is := r.Primary.DeepCopy()
				is.Ephemeral.Type = r.Type // otherwise the check function cannot see the type
				states = append(states, is)
			}
		}
		if err := step.ImportStateCheck(states); err != nil {
			return state, err
		}
	}

	// Verify that all the states match
	if step.ImportStateVerify {
		new := newState.RootModule().Resources
		old := state.RootModule().Resources
		for _, r := range new {
			// Find the existing resource
			var oldR *terraform.ResourceState
			for _, r2 := range old {
				if r2.Primary != nil && r2.Primary.ID == r.Primary.ID && r2.Type == r.Type {
					oldR = r2
					break
				}
			}
			if oldR == nil {
				return state, fmt.Errorf(
					"Failed state verification, resource with ID %s not found",
					r.Primary.ID)
			}

			// We'll try our best to find the schema for this resource type
			// so we can ignore Removed fields during validation. If we fail
			// to find the schema then we won't ignore them and so the test
			// will need to rely on explicit ImportStateVerifyIgnore, though
			// this shouldn't happen in any reasonable case.
			var rsrcSchema *schema.Resource
			if providerAddr, diags := addrs.ParseAbsProviderConfigStr(r.Provider); !diags.HasErrors() {
				providerType := providerAddr.ProviderConfig.Type
				if provider, ok := step.providers[providerType]; ok {
					if provider, ok := provider.(*schema.Provider); ok {
						rsrcSchema = provider.ResourcesMap[r.Type]
					}
				}
			}

			// don't add empty flatmapped containers, so we can more easily
			// compare the attributes
			skipEmpty := func(k, v string) bool {
				if strings.HasSuffix(k, ".#") || strings.HasSuffix(k, ".%") {
					if v == "0" {
						return true
					}
				}
				return false
			}

			// Compare their attributes
			actual := make(map[string]string)
			for k, v := range r.Primary.Attributes {
				if skipEmpty(k, v) {
					continue
				}
				actual[k] = v
			}

			expected := make(map[string]string)
			for k, v := range oldR.Primary.Attributes {
				if skipEmpty(k, v) {
					continue
				}
				expected[k] = v
			}

			// Remove fields we're ignoring
			for _, v := range step.ImportStateVerifyIgnore {
				for k := range actual {
					if strings.HasPrefix(k, v) {
						delete(actual, k)
					}
				}
				for k := range expected {
					if strings.HasPrefix(k, v) {
						delete(expected, k)
					}
				}
			}

			// Also remove any attributes that are marked as "Removed" in the
			// schema, if we have a schema to check that against.
			if rsrcSchema != nil {
				for k := range actual {
					for _, schema := range rsrcSchema.SchemasForFlatmapPath(k) {
						if schema.Removed != "" {
							delete(actual, k)
							break
						}
					}
				}
				for k := range expected {
					for _, schema := range rsrcSchema.SchemasForFlatmapPath(k) {
						if schema.Removed != "" {
							delete(expected, k)
							break
						}
					}
				}
			}

			if !reflect.DeepEqual(actual, expected) {
				// Determine only the different attributes
				for k, v := range expected {
					if av, ok := actual[k]; ok && v == av {
						delete(expected, k)
						delete(actual, k)
					}
				}

				spewConf := spew.NewDefaultConfig()
				spewConf.SortKeys = true
				return state, fmt.Errorf(
					"ImportStateVerify attributes not equivalent. Difference is shown below. Top is actual, bottom is expected."+
						"\n\n%s\n\n%s",
					spewConf.Sdump(actual), spewConf.Sdump(expected))
			}
		}
	}

	// Return the old state (non-imported) so we don't change anything.
	return state, nil
}
