package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
)

type SmokeTest struct {
	Name string

	DataResources  []*Resource
	Preconditions  []*CheckRule
	Postconditions []*CheckRule

	DeclRange hcl.Range
}

func (st *SmokeTest) Addr() addrs.SmokeTest {
	return addrs.SmokeTest{Name: st.Name}
}

func decodeSmokeTestBlock(block *hcl.Block, override bool) (*SmokeTest, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	ret := &SmokeTest{
		Name:      block.Labels[0],
		DeclRange: block.DefRange,
	}
	if override {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Cannot declare smoke tests in override files",
			Detail:   "Smoke tests are valid only in primary configuration files, and not in override files.",
			Subject:  block.DefRange.Ptr(),
		})
	}

	content, moreDiags := block.Body.Content(smokeTestBlockSchema)
	diags = append(diags, moreDiags...)

	if !hclsyntax.ValidIdentifier(ret.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid smoke test name",
			Detail:   badIdentifierDetail,
			Subject:  &block.LabelRanges[0],
		})
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "data":
			dataResource, moreDiags := decodeDataBlock(block, false)
			diags = append(diags, moreDiags...)
			if !moreDiags.HasErrors() {
				// NOTE: We catch duplicate names later when we merge
				// separate configuration files into a single module.
				ret.DataResources = append(ret.DataResources, dataResource)
			}
			dataResource.SmokeTest = ret

			// Smoke-test-specific data resources must not have their own
			// conditions, because the smoke test as a whole is responsible
			// for checking its input and results.
			for _, check := range dataResource.Preconditions {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Data resource in smoke test must not have precondition",
					Detail:   "To declare preconditions for this smoke test, use precondition blocks nested directly inside the smoke_test block instead.",
					Subject:  check.DeclRange.Ptr(),
				})
			}
			for _, check := range dataResource.Postconditions {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Data resource in smoke test must not have postcondition",
					Detail:   "To declare postconditions for this smoke test, use postcondition blocks nested directly inside the smoke_test block instead.",
					Subject:  check.DeclRange.Ptr(),
				})
			}
		case "precondition":
			check, moreDiags := decodeCheckRuleBlock(block, false)
			diags = append(diags, moreDiags...)
			if !moreDiags.HasErrors() {
				ret.Preconditions = append(ret.Preconditions, check)
			}
		case "postcondition":
			check, moreDiags := decodeCheckRuleBlock(block, false)
			diags = append(diags, moreDiags...)
			if !moreDiags.HasErrors() {
				ret.Postconditions = append(ret.Postconditions, check)
			}
		default:
			// Should not get here because the above cases should be exhaustive
			// for all of the block types in smokeTestBlockSchema.
			panic(fmt.Sprintf("unhandled smoke_test nested block type %q", block.Type))
		}
	}

	if len(ret.DataResources) == 0 && len(ret.Preconditions) == 0 && len(ret.Postconditions) == 0 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Smoke test checks nothing",
			Detail:   "Each smoke_test block must have at least one precondition, data, or postcondition block.",
			Subject:  block.DefRange.Ptr(),
		})
	}

	return ret, diags
}

var smokeTestBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "data", LabelNames: []string{"type", "name"}},
		{Type: "precondition"},
		{Type: "postcondition"},
	},
}
