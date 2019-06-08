package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/command/jsondiags"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/lang"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"
	ctyconvert "github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/gocty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// TestingEvalCommand is a Command implementation that evaluates configuration
// objects against static data for unit testing purposes.
type TestingEvalCommand struct {
	Meta
}

func (c *TestingEvalCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("testing eval")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 3 {
		c.Ui.Error(c.Help())
		return 1
	}

	val, diags := c.testingEval(args[0], args[1], args[2])

	type Result struct {
		Value       json.RawMessage `json:"value,omitempty"`
		Type        json.RawMessage `json:"type,omitempty"`
		Diagnostics json.RawMessage `json:"diagnostics,omitempty"`
	}
	var result Result
	if len(diags) > 0 {
		result.Diagnostics = json.RawMessage(jsondiags.Diagnostics(diags))
	}
	if val != cty.NilVal {
		val = cty.UnknownAsNull(val) // can't represent unknowns in JSON
		result.Type, _ = val.Type().MarshalJSON()
		result.Value, _ = ctyjson.Marshal(val, val.Type())
	}

	resultJSON, _ := json.MarshalIndent(&result, "", "  ")
	fmt.Printf("%s\n", resultJSON)

	if diags.HasErrors() {
		return 1
	}
	return 0
}

func (c *TestingEvalCommand) testingEval(modDir, refStr, dataFn string) (val cty.Value, diags tfdiags.Diagnostics) {
	modDir = c.normalizePath(modDir)

	loader, err := c.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return
	}

	p := loader.Parser()
	mod, cfgDiags := p.LoadConfigDir(modDir)
	diags = diags.Append(cfgDiags)
	if diags.HasErrors() {
		return
	}

	ref, moreDiags := addrs.ParseRefStr(refStr)
	diags = diags.Append(moreDiags)

	if len(ref.Remaining) != 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported test object",
			fmt.Sprintf("Can only evaluate whole objects. To evaluate the object containing this value, use %s.", ref.Subject),
		))
		return
	}

	mocks, moreDiags := c.loadMocks(dataFn)
	diags = diags.Append(moreDiags)

	// Although we don't use any real provider plugins to do the evaluation,
	// we will go fetch the schemas from the plugins so we can ensure the
	// mock data is correctly typed and, if the target is a resource, decode
	// its configuration.
	//
	// Creating a context requires a configuration to analyze, so we'll
	// construct a stub of one containing only our single module.
	cfg, cfgDiags := configs.BuildConfig(
		mod, configs.ModuleWalkerFunc(func(req *configs.ModuleRequest) (*configs.Module, *version.Version, hcl.Diagnostics) {
			emptyMod, diags := configs.NewModule(nil, nil)
			return emptyMod, nil, diags
		}),
	)
	opts := c.contextOpts()
	opts.Config = cfg
	tfCtx, moreDiags := terraform.NewContext(opts)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return
	}
	schemas := tfCtx.Schemas()

	data := &testingEvalCommandData{
		Module:  mod,
		Schemas: schemas,
		Mocks:   mocks,
	}
	scope := &lang.Scope{
		Data:    data,
		BaseDir: modDir,
	}
	countIndexVal := &data.CountIndex

	switch addr := ref.Subject.(type) {

	case addrs.ResourceInstance:
		if addr.Key != addrs.NoKey {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unsupported test object",
				fmt.Sprintf("Cannot evaluate %s; only whole resources can be evaluated, not resource instances. To evaluate this resource, use %s.", ref.Subject, addr.Resource),
			))
			return
		}

		rAddr := addr.Resource
		rc := mod.ResourceByAddr(rAddr)
		if rc == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Undefined resource",
				fmt.Sprintf("Cannot evaluate undefined resource %s.", rAddr),
			))
			return
		}

		providerAddr := rc.ProviderConfigAddr()
		schema, _ := schemas.ResourceTypeConfig(providerAddr.Type, rAddr.Mode, rAddr.Type)
		if schema == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unsupported resource type",
				fmt.Sprintf("Provider %s does not support the resource type for %s.", providerAddr.Type, rAddr),
			))
		}
		objTy := schema.ImpliedType()

		if rc.Count != nil {
			// Result will be a tuple
			countVal, moreDiags := scope.EvalExpr(rc.Count, cty.Number)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return
			}
			var count int
			if err := gocty.FromCtyValue(countVal, &count); err != nil || count < 0 {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid count value",
					Detail:   fmt.Sprintf("Invalid count value: %s.", err),
					Subject:  rc.Count.Range().Ptr(),
				})
				return
			}

			var results []cty.Value
			for i := 0; i < count; i++ {
				*countIndexVal = cty.NumberIntVal(int64(i))

				result, moreDiags := scope.EvalBlock(rc.Config, schema)
				diags = diags.Append(moreDiags)
				if result == cty.NilVal {
					result = cty.NullVal(objTy)
				}

				results = append(results, result)
			}

			val = cty.TupleVal(results)
			return
		} else {
			// Result will be a single object
			result, moreDiags := scope.EvalBlock(rc.Config, schema)
			diags = diags.Append(moreDiags)
			if result == cty.NilVal {
				result = cty.NullVal(objTy)
			}

			val = result
			return
		}

	case addrs.LocalValue:
		lc := mod.Locals[addr.Name]
		if lc == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Undefined local value",
				fmt.Sprintf("Cannot evaluate undefined local value %s.", addr),
			))
			return
		}

		result, moreDiags := scope.EvalExpr(lc.Expr, cty.DynamicPseudoType)
		diags = diags.Append(moreDiags)
		if result == cty.NilVal {
			result = cty.NullVal(cty.DynamicPseudoType)
		}

		val = result
		return

	default:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported test object",
			fmt.Sprintf("Cannot evaluate %s; only resource blocks, data blocks, and local values can be evaluated.", ref.Subject),
		))
	}

	return
}

type testingEvalCommandMocks struct {
	Resources      map[string]json.RawMessage `json:"resources"`
	LocalValues    map[string]json.RawMessage `json:"locals"`
	InputVariables map[string]json.RawMessage `json:"variables"`
	ModuleCalls    map[string]json.RawMessage `json:"modules"`
	PathAttrs      map[string]json.RawMessage `json:"paths"`
	TerraformAttrs map[string]json.RawMessage `json:"terraform"`
}

func (c *TestingEvalCommand) loadMocks(fn string) (*testingEvalCommandMocks, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var ret testingEvalCommandMocks

	var src []byte
	var err error
	switch fn {
	case "-":
		src, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read mock data",
				fmt.Sprintf("Could not read mock data from stdin: %s.", err),
			))
			return &ret, diags
		}
	default:
		src, err = ioutil.ReadFile(fn)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read mock data",
				fmt.Sprintf("Could not read mock data from %s: %s.", fn, err),
			))
			return &ret, diags
		}
	}

	err = json.Unmarshal(src, &ret)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to read mock data",
			fmt.Sprintf("Invalid mock data JSON: %s.", err),
		))
	}
	return &ret, diags
}

type testingEvalCommandData struct {
	Module     *configs.Module
	Schemas    *terraform.Schemas
	Mocks      *testingEvalCommandMocks
	CountIndex cty.Value
}

var _ lang.Data = (*testingEvalCommandData)(nil)

func (d *testingEvalCommandData) StaticValidateReferences(refs []*addrs.Reference, self addrs.Referenceable) tfdiags.Diagnostics {
	// TODO: Validate against the schema
	return nil
}

func (d *testingEvalCommandData) GetCountAttr(addr addrs.CountAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	switch addr.Name {
	case "index":
		if d.CountIndex == cty.NilVal {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid count reference",
				Detail:   "The \"count\" object cannot be used in this context.",
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.UnknownVal(cty.Number), diags
		}
		return d.CountIndex, diags
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid count attribute",
			Detail:   "The count object only has one attribute: \"index\".",
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.UnknownVal(cty.Number), diags
	}
}

func (d *testingEvalCommandData) GetResourceInstance(addr addrs.ResourceInstance, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	rc := d.Module.ResourceByAddr(addr.Resource)
	if rc == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undeclared resource",
			Detail:   fmt.Sprintf("This module contains no declaration for %s.", addr),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	schema, _ := d.Schemas.ResourceTypeConfig(rc.ProviderConfigAddr().Type, addr.Resource.Mode, addr.Resource.Type)
	if schema == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported resource type",
			Detail:   fmt.Sprintf("No schema information is available for %s.", addr.Resource),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	rawData, ok := d.Mocks.Resources[addr.Resource.String()]
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Resource mock data unavailable",
			Detail:   fmt.Sprintf("The mock \"resources\" object contains no property for %s.", addr.Resource),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	var valuesRaw []json.RawMessage
	if rc.Count != nil {
		err := json.Unmarshal(rawData, &valuesRaw)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid mock value for resource",
				Detail:   fmt.Sprintf("The resource %s has \"count\" set, so its mock value must be a JSON array.", addr.Resource),
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.DynamicVal, diags
		}
	} else {
		valuesRaw = []json.RawMessage{rawData}
	}

	values := make([]cty.Value, len(valuesRaw))
	ty := schema.ImpliedType()
	for i, valueRaw := range valuesRaw {
		val, err := ctyjson.Unmarshal(valueRaw, ty)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid mock value for resource instance",
				Detail:   fmt.Sprintf("Unsuitable mock value for %s: %s.", addr.Resource.Instance(addrs.IntKey(i)), err),
				Subject:  rng.ToHCL().Ptr(),
			})
			continue
		}

		val, err = ctyconvert.Convert(val, ty)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid mock value for resource instance",
				Detail:   fmt.Sprintf("Unsuitable mock value for %s: %s.", addr.Resource.Instance(addrs.IntKey(i)), err),
				Subject:  rng.ToHCL().Ptr(),
			})
			continue
		}

		values[i] = val
	}

	if diags.HasErrors() {
		return cty.DynamicVal, diags
	}

	switch {
	case addr.Key == addrs.NoKey:
		// Could be reference to either individual instance or whole sequence
		// of resources, depending on whether count is set.
		switch {
		case rc.Count != nil:
			return cty.TupleVal(values), diags
		default:
			return values[0], diags
		}
	default:
		idx, ok := addr.Key.(addrs.IntKey)
		if !ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Unsupported resource instance address",
				Detail:   fmt.Sprintf("Testing evaluator can't produce a mock value for %s: for_each is not yet supported.", addr.Resource),
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.DynamicVal, diags
		}
		if idx < 0 || int(idx) >= len(values) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Reference to resource instance out of range",
				Detail:   fmt.Sprintf("Mock instance data is only available for indices 0 through %d.", len(values)-1),
				Subject:  rng.ToHCL().Ptr(),
			})
			return cty.DynamicVal, diags
		}
		return values[int(idx)], diags
	}
}

func (d *testingEvalCommandData) GetLocalValue(addr addrs.LocalValue, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	lc := d.Module.Locals[addr.Name]
	if lc == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undeclared local value",
			Detail:   fmt.Sprintf("This module contains no declaration for %s.", addr),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	rawValue, ok := d.Mocks.LocalValues[addr.Name]
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Local value mock data unavailable",
			Detail:   fmt.Sprintf("The mock \"locals\" object contains no property for %s.", addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	gotTy, err := ctyjson.ImpliedType(rawValue)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid mock value for local value",
			Detail:   fmt.Sprintf("Unsuitable mock value for %s: %s.", addr, err),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	val, err := ctyjson.Unmarshal(rawValue, gotTy)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid mock value for local value",
			Detail:   fmt.Sprintf("Unsuitable mock value for %s: %s.", addr, err),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	return val, diags
}

func (d *testingEvalCommandData) GetModuleInstance(addr addrs.ModuleCallInstance, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// FIXME: This assumes that "count" and "for_each" are not supported for
	// modules, which is true at the time of writing but will change in future.

	mc := d.Module.ModuleCalls[addr.Call.Name]
	if mc == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undeclared module call",
			Detail:   fmt.Sprintf("This module contains no declaration for %s.", addr),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	rawValue, ok := d.Mocks.ModuleCalls[addr.Call.Name]
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Local value mock data unavailable",
			Detail:   fmt.Sprintf("The mock \"modules\" object contains no property for %s.", addr.Call.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	gotTy, err := ctyjson.ImpliedType(rawValue)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid mock value for module call",
			Detail:   fmt.Sprintf("Unsuitable mock value for %s: %s.", addr, err),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	if !gotTy.IsObjectType() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid mock value for module call",
			Detail:   fmt.Sprintf("Unsuitable mock value for %s: must be an object with a property for each output value.", addr),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	val, err := ctyjson.Unmarshal(rawValue, gotTy)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid mock value for module call",
			Detail:   fmt.Sprintf("Unsuitable mock value for %s: %s.", addr, err),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	return val, diags
}

func (d *testingEvalCommandData) GetModuleInstanceOutput(addr addrs.ModuleCallOutput, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	modVal, diags := d.GetModuleInstance(addr.Call, rng)
	if diags.HasErrors() {
		return cty.DynamicVal, diags
	}
	if !modVal.IsKnown() {
		return cty.DynamicVal, diags
	}
	if !(modVal.Type().IsObjectType() && modVal.Type().HasAttribute(addr.Name)) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undefined module output",
			Detail:   fmt.Sprintf("The mock object for %s has no property %q.", addr.Call, addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}
	return modVal.GetAttr(addr.Name), diags
}

func (d *testingEvalCommandData) GetPathAttr(addr addrs.PathAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	val := cty.DynamicVal

	switch addr.Name {
	case "cwd", "root", "module":
		rawValue, ok := d.Mocks.PathAttrs[addr.Name]
		if !ok {
			val = cty.StringVal(".")
			break
		}

		value, err := ctyjson.Unmarshal(rawValue, cty.String)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid path attribute mock",
				Detail:   fmt.Sprintf("The given mock value for %s is unsuitable: %s.", addr, err),
				Subject:  rng.ToHCL().Ptr(),
			})
			break
		}

		val = value

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid path attribute",
			Detail:   fmt.Sprintf("The path object has no attribute %q.", addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
	}

	return val, diags
}

func (d *testingEvalCommandData) GetTerraformAttr(addr addrs.TerraformAttr, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	val := cty.DynamicVal

	switch addr.Name {
	case "workspace":
		rawValue, ok := d.Mocks.TerraformAttrs[addr.Name]
		if !ok {
			val = cty.StringVal("default")
			break
		}

		value, err := ctyjson.Unmarshal(rawValue, cty.String)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid terraform attribute mock",
				Detail:   fmt.Sprintf("The given mock value for %s is unsuitable: %s.", addr, err),
				Subject:  rng.ToHCL().Ptr(),
			})
			break
		}

		val = value

	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid terraform attribute",
			Detail:   fmt.Sprintf("The terraform object has no attribute %q.", addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
	}

	return val, diags
}

func (d *testingEvalCommandData) GetInputVariable(addr addrs.InputVariable, rng tfdiags.SourceRange) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	vc := d.Module.Variables[addr.Name]
	if vc == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Reference to undeclared variable",
			Detail:   fmt.Sprintf("This module contains no declaration for %s.", addr),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	rawValue, ok := d.Mocks.InputVariables[addr.Name]
	if !ok {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Input variable mock data unavailable",
			Detail:   fmt.Sprintf("The mock \"variables\" object contains no property for %s.", addr.Name),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	wantTy := vc.Type
	if wantTy == cty.NilType {
		wantTy = cty.DynamicPseudoType
	}

	gotTy, err := ctyjson.ImpliedType(rawValue)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid mock value for input variable",
			Detail:   fmt.Sprintf("Unsuitable mock value for %s: %s.", addr, err),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	val, err := ctyjson.Unmarshal(rawValue, gotTy)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid mock value for input variable",
			Detail:   fmt.Sprintf("Unsuitable mock value for %s: %s.", addr, err),
			Subject:  rng.ToHCL().Ptr(),
		})
		return cty.DynamicVal, diags
	}

	val, err = ctyconvert.Convert(val, wantTy)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid mock value for input variable",
			Detail:   fmt.Sprintf("Unsuitable mock value for %s: %s.", addr, err),
			Subject:  rng.ToHCL().Ptr(),
		})
	}

	return val, diags
}

func (c *TestingEvalCommand) Help() string {
	helpText := `
Usage: terraform testing eval MODULE-DIR REF-ADDR DATA-FILE

  A plumbing command that evaluates a single object identified by
  REF-ADDR from the module in MODULE-DIR using values from
  DATA-FILE as a mock dataset for expression evaluation.

  The result is printed in JSON format on stdout. If the data
  on stdout is not valid JSON, stderr may contain a human-
  readable description of a general initialization error.

  `
	return strings.TrimSpace(helpText)
}

func (c *TestingEvalCommand) Synopsis() string {
	return "Plumbing command for testing configuration objects"
}
