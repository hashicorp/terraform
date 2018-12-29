package lang

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/addrs"

	"github.com/hashicorp/hcl2/ext/dynblock"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
)

// ExpandBlock expands any "dynamic" blocks present in the given body. The
// result is a body with those blocks expanded, ready to be evaluated with
// EvalBlock.
//
// If the returned diagnostics contains errors then the result may be
// incomplete or invalid.
func (s *Scope) ExpandBlock(body hcl.Body, schema *configschema.Block) (hcl.Body, tfdiags.Diagnostics) {
	spec := schema.DecoderSpec()

	traversals := dynblock.ForEachVariablesHCLDec(body, spec)
	refs, diags := References(traversals)

	ctx, ctxDiags := s.EvalContext(refs)
	diags = diags.Append(ctxDiags)

	return dynblock.Expand(body, ctx), diags
}

// EvalBlock evaluates the given body using the given block schema and returns
// a cty object value representing its contents. The type of the result conforms
// to the implied type of the given schema.
//
// This function does not automatically expand "dynamic" blocks within the
// body. If that is desired, first call the ExpandBlock method to obtain
// an expanded body to pass to this method.
//
// If the returned diagnostics contains errors then the result may be
// incomplete or invalid.
func (s *Scope) EvalBlock(body hcl.Body, schema *configschema.Block) (cty.Value, tfdiags.Diagnostics) {
	spec := schema.DecoderSpec()

	traversals := hcldec.Variables(body, spec)
	refs, diags := References(traversals)

	ctx, ctxDiags := s.EvalContext(refs)
	diags = diags.Append(ctxDiags)
	if diags.HasErrors() {
		// We'll stop early if we found problems in the references, because
		// it's likely evaluation will produce redundant copies of the same errors.
		return cty.UnknownVal(schema.ImpliedType()), diags
	}

	val, evalDiags := hcldec.Decode(body, spec, ctx)
	diags = diags.Append(evalDiags)

	return val, diags
}

// EvalExpr evaluates a single expression in the receiving context and returns
// the resulting value. The value will be converted to the given type before
// it is returned if possible, or else an error diagnostic will be produced
// describing the conversion error.
//
// Pass an expected type of cty.DynamicPseudoType to skip automatic conversion
// and just obtain the returned value directly.
//
// If the returned diagnostics contains errors then the result may be
// incomplete, but will always be of the requested type.
func (s *Scope) EvalExpr(expr hcl.Expression, wantType cty.Type) (cty.Value, tfdiags.Diagnostics) {
	refs, diags := ReferencesInExpr(expr)

	ctx, ctxDiags := s.EvalContext(refs)
	diags = diags.Append(ctxDiags)
	if diags.HasErrors() {
		// We'll stop early if we found problems in the references, because
		// it's likely evaluation will produce redundant copies of the same errors.
		return cty.UnknownVal(wantType), diags
	}

	val, evalDiags := expr.Value(ctx)
	diags = diags.Append(evalDiags)

	if wantType != cty.DynamicPseudoType {
		var convErr error
		val, convErr = convert.Convert(val, wantType)
		if convErr != nil {
			val = cty.UnknownVal(wantType)
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Incorrect value type",
				Detail:   fmt.Sprintf("Invalid expression value: %s.", tfdiags.FormatError(convErr)),
				Subject:  expr.Range().Ptr(),
			})
		}
	}

	return val, diags
}

// EvalReference evaluates the given reference in the receiving scope and
// returns the resulting value. The value will be converted to the given type before
// it is returned if possible, or else an error diagnostic will be produced
// describing the conversion error.
//
// Pass an expected type of cty.DynamicPseudoType to skip automatic conversion
// and just obtain the returned value directly.
//
// If the returned diagnostics contains errors then the result may be
// incomplete, but will always be of the requested type.
func (s *Scope) EvalReference(ref *addrs.Reference, wantType cty.Type) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// We cheat a bit here and just build an EvalContext for our requested
	// reference with the "self" address overridden, and then pull the "self"
	// result out of it to return.
	ctx, ctxDiags := s.evalContext([]*addrs.Reference{ref}, ref.Subject)
	diags = diags.Append(ctxDiags)
	val := ctx.Variables["self"]
	if val == cty.NilVal {
		val = cty.DynamicVal
	}

	var convErr error
	val, convErr = convert.Convert(val, wantType)
	if convErr != nil {
		val = cty.UnknownVal(wantType)
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incorrect value type",
			Detail:   fmt.Sprintf("Invalid expression value: %s.", tfdiags.FormatError(convErr)),
			Subject:  ref.SourceRange.ToHCL().Ptr(),
		})
	}

	return val, diags
}

// EvalContext constructs a HCL expression evaluation context whose variable
// scope contains sufficient values to satisfy the given set of references.
//
// Most callers should prefer to use the evaluation helper methods that
// this type offers, but this is here for less common situations where the
// caller will handle the evaluation calls itself.
func (s *Scope) EvalContext(refs []*addrs.Reference) (*hcl.EvalContext, tfdiags.Diagnostics) {
	return s.evalContext(refs, s.SelfAddr)
}

func (s *Scope) evalContext(refs []*addrs.Reference, selfAddr addrs.Referenceable) (*hcl.EvalContext, tfdiags.Diagnostics) {
	if s == nil {
		panic("attempt to construct EvalContext for nil Scope")
	}

	var diags tfdiags.Diagnostics
	vals := make(map[string]cty.Value)
	funcs := s.Functions()
	ctx := &hcl.EvalContext{
		Variables: vals,
		Functions: funcs,
	}

	if len(refs) == 0 {
		// Easy path for common case where there are no references at all.
		return ctx, diags
	}

	// First we'll do static validation of the references. This catches things
	// early that might otherwise not get caught due to unknown values being
	// present in the scope during planning.
	if staticDiags := s.Data.StaticValidateReferences(refs, selfAddr); staticDiags.HasErrors() {
		diags = diags.Append(staticDiags)
		return ctx, diags
	}

	// The reference set we are given has not been de-duped, and so there can
	// be redundant requests in it for two reasons:
	//  - The same item is referenced multiple times
	//  - Both an item and that item's container are separately referenced.
	// We will still visit every reference here and ask our data source for
	// it, since that allows us to gather a full set of any errors and
	// warnings, but once we've gathered all the data we'll then skip anything
	// that's redundant in the process of populating our values map.
	dataResources := map[string]map[string]map[addrs.InstanceKey]cty.Value{}
	managedResources := map[string]map[string]map[addrs.InstanceKey]cty.Value{}
	wholeModules := map[string]map[addrs.InstanceKey]cty.Value{}
	moduleOutputs := map[string]map[addrs.InstanceKey]map[string]cty.Value{}
	inputVariables := map[string]cty.Value{}
	localValues := map[string]cty.Value{}
	pathAttrs := map[string]cty.Value{}
	terraformAttrs := map[string]cty.Value{}
	countAttrs := map[string]cty.Value{}
	var self cty.Value

	for _, ref := range refs {
		rng := ref.SourceRange
		isSelf := false

		rawSubj := ref.Subject
		if rawSubj == addrs.Self {
			if selfAddr == nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Invalid "self" reference`,
					// This detail message mentions some current practice that
					// this codepath doesn't really "know about". If the "self"
					// object starts being supported in more contexts later then
					// we'll need to adjust this message.
					Detail:  `The "self" object is not available in this context. This object can be used only in resource provisioner and connection blocks.`,
					Subject: ref.SourceRange.ToHCL().Ptr(),
				})
				continue
			}

			// Treat "self" as an alias for the configured self address.
			rawSubj = selfAddr
			isSelf = true

			if rawSubj == addrs.Self {
				// Programming error: the self address cannot alias itself.
				panic("scope SelfAddr attempting to alias itself")
			}
		}

		// This type switch must cover all of the "Referenceable" implementations
		// in package addrs.
		switch subj := rawSubj.(type) {

		case addrs.ResourceInstance:
			var into map[string]map[string]map[addrs.InstanceKey]cty.Value
			switch subj.Resource.Mode {
			case addrs.ManagedResourceMode:
				into = managedResources
			case addrs.DataResourceMode:
				into = dataResources
			default:
				panic(fmt.Errorf("unsupported ResourceMode %s", subj.Resource.Mode))
			}

			val, valDiags := normalizeRefValue(s.Data.GetResourceInstance(subj, rng))
			diags = diags.Append(valDiags)

			r := subj.Resource
			if into[r.Type] == nil {
				into[r.Type] = make(map[string]map[addrs.InstanceKey]cty.Value)
			}
			if into[r.Type][r.Name] == nil {
				into[r.Type][r.Name] = make(map[addrs.InstanceKey]cty.Value)
			}
			into[r.Type][r.Name][subj.Key] = val
			if isSelf {
				self = val
			}

		case addrs.ModuleCallInstance:
			val, valDiags := normalizeRefValue(s.Data.GetModuleInstance(subj, rng))
			diags = diags.Append(valDiags)

			if wholeModules[subj.Call.Name] == nil {
				wholeModules[subj.Call.Name] = make(map[addrs.InstanceKey]cty.Value)
			}
			wholeModules[subj.Call.Name][subj.Key] = val
			if isSelf {
				self = val
			}

		case addrs.ModuleCallOutput:
			val, valDiags := normalizeRefValue(s.Data.GetModuleInstanceOutput(subj, rng))
			diags = diags.Append(valDiags)

			callName := subj.Call.Call.Name
			callKey := subj.Call.Key
			if moduleOutputs[callName] == nil {
				moduleOutputs[callName] = make(map[addrs.InstanceKey]map[string]cty.Value)
			}
			if moduleOutputs[callName][callKey] == nil {
				moduleOutputs[callName][callKey] = make(map[string]cty.Value)
			}
			moduleOutputs[callName][callKey][subj.Name] = val
			if isSelf {
				self = val
			}

		case addrs.InputVariable:
			val, valDiags := normalizeRefValue(s.Data.GetInputVariable(subj, rng))
			diags = diags.Append(valDiags)
			inputVariables[subj.Name] = val
			if isSelf {
				self = val
			}

		case addrs.LocalValue:
			val, valDiags := normalizeRefValue(s.Data.GetLocalValue(subj, rng))
			diags = diags.Append(valDiags)
			localValues[subj.Name] = val
			if isSelf {
				self = val
			}

		case addrs.PathAttr:
			val, valDiags := normalizeRefValue(s.Data.GetPathAttr(subj, rng))
			diags = diags.Append(valDiags)
			pathAttrs[subj.Name] = val
			if isSelf {
				self = val
			}

		case addrs.TerraformAttr:
			val, valDiags := normalizeRefValue(s.Data.GetTerraformAttr(subj, rng))
			diags = diags.Append(valDiags)
			terraformAttrs[subj.Name] = val
			if isSelf {
				self = val
			}

		case addrs.CountAttr:
			val, valDiags := normalizeRefValue(s.Data.GetCountAttr(subj, rng))
			diags = diags.Append(valDiags)
			countAttrs[subj.Name] = val
			if isSelf {
				self = val
			}

		default:
			// Should never happen
			panic(fmt.Errorf("Scope.buildEvalContext cannot handle address type %T", rawSubj))
		}
	}

	for k, v := range buildResourceObjects(managedResources) {
		vals[k] = v
	}
	vals["data"] = cty.ObjectVal(buildResourceObjects(dataResources))
	vals["module"] = cty.ObjectVal(buildModuleObjects(wholeModules, moduleOutputs))
	vals["var"] = cty.ObjectVal(inputVariables)
	vals["local"] = cty.ObjectVal(localValues)
	vals["path"] = cty.ObjectVal(pathAttrs)
	vals["terraform"] = cty.ObjectVal(terraformAttrs)
	vals["count"] = cty.ObjectVal(countAttrs)
	if self != cty.NilVal {
		vals["self"] = self
	}

	return ctx, diags
}

func buildResourceObjects(resources map[string]map[string]map[addrs.InstanceKey]cty.Value) map[string]cty.Value {
	vals := make(map[string]cty.Value)
	for typeName, names := range resources {
		nameVals := make(map[string]cty.Value)
		for name, keys := range names {
			nameVals[name] = buildInstanceObjects(keys)
		}
		vals[typeName] = cty.ObjectVal(nameVals)
	}
	return vals
}

func buildModuleObjects(wholeModules map[string]map[addrs.InstanceKey]cty.Value, moduleOutputs map[string]map[addrs.InstanceKey]map[string]cty.Value) map[string]cty.Value {
	vals := make(map[string]cty.Value)

	for name, keys := range wholeModules {
		vals[name] = buildInstanceObjects(keys)
	}

	for name, keys := range moduleOutputs {
		if _, exists := wholeModules[name]; exists {
			// If we also have a whole module value for this name then we'll
			// skip this since the individual outputs are embedded in that result.
			continue
		}

		// The shape of this collection isn't compatible with buildInstanceObjects,
		// but rather than replicating most of the buildInstanceObjects logic
		// here we'll instead first transform the structure to be what that
		// function expects and then use it. This is a little wasteful, but
		// we do not expect this these maps to be large and so the extra work
		// here should not hurt too much.
		flattened := make(map[addrs.InstanceKey]cty.Value, len(keys))
		for k, vals := range keys {
			flattened[k] = cty.ObjectVal(vals)
		}
		vals[name] = buildInstanceObjects(flattened)
	}

	return vals
}

func buildInstanceObjects(keys map[addrs.InstanceKey]cty.Value) cty.Value {
	if val, exists := keys[addrs.NoKey]; exists {
		// If present, a "no key" value supersedes all other values,
		// since they should be embedded inside it.
		return val
	}

	// If we only have individual values then we need to construct
	// either a list or a map, depending on what sort of keys we
	// have.
	haveInt := false
	haveString := false
	maxInt := 0

	for k := range keys {
		switch tk := k.(type) {
		case addrs.IntKey:
			haveInt = true
			if int(tk) > maxInt {
				maxInt = int(tk)
			}
		case addrs.StringKey:
			haveString = true
		}
	}

	// We should either have ints or strings and not both, but
	// if we have both then we'll prefer strings and let the
	// language interpreter try to convert the int keys into
	// strings in a map.
	switch {
	case haveString:
		vals := make(map[string]cty.Value)
		for k, v := range keys {
			switch tk := k.(type) {
			case addrs.StringKey:
				vals[string(tk)] = v
			case addrs.IntKey:
				sk := strconv.Itoa(int(tk))
				vals[sk] = v
			}
		}
		return cty.ObjectVal(vals)
	case haveInt:
		// We'll make a tuple that is long enough for our maximum
		// index value. It doesn't matter if we end up shorter than
		// the number of instances because if length(...) were
		// being evaluated we would've got a NoKey reference and
		// thus not ended up in this codepath at all.
		vals := make([]cty.Value, maxInt+1)
		for i := range vals {
			if v, exists := keys[addrs.IntKey(i)]; exists {
				vals[i] = v
			} else {
				// Just a placeholder, since nothing will access this anyway
				vals[i] = cty.DynamicVal
			}
		}
		return cty.TupleVal(vals)
	default:
		// Should never happen because there are no other key types.
		log.Printf("[ERROR] strange makeInstanceObjects call with no supported key types")
		return cty.EmptyObjectVal
	}
}

func normalizeRefValue(val cty.Value, diags tfdiags.Diagnostics) (cty.Value, tfdiags.Diagnostics) {
	if diags.HasErrors() {
		// If there are errors then we will force an unknown result so that
		// we can still evaluate and catch type errors but we'll avoid
		// producing redundant re-statements of the same errors we've already
		// dealt with here.
		return cty.UnknownVal(val.Type()), diags
	}
	return val, diags
}
