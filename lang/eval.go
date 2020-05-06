package lang

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/dynblock"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/lang/blocktoattr"
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

	traversals := dynblock.ExpandVariablesHCLDec(body, spec)
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

	refs, diags := ReferencesInBlock(body, schema)

	ctx, ctxDiags := s.EvalContext(refs)
	diags = diags.Append(ctxDiags)
	if diags.HasErrors() {
		// We'll stop early if we found problems in the references, because
		// it's likely evaluation will produce redundant copies of the same errors.
		return cty.UnknownVal(schema.ImpliedType()), diags
	}

	// HACK: In order to remain compatible with some assumptions made in
	// Terraform v0.11 and earlier about the approximate equivalence of
	// attribute vs. block syntax, we do a just-in-time fixup here to allow
	// any attribute in the schema that has a list-of-objects or set-of-objects
	// kind to potentially be populated instead by one or more nested blocks
	// whose type is the attribute name.
	body = blocktoattr.FixUpBlockAttrs(body, schema)

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
	dataResources := map[string]map[string]cty.Value{}
	managedResources := map[string]map[string]cty.Value{}
	wholeModules := map[string]cty.Value{}
	inputVariables := map[string]cty.Value{}
	localValues := map[string]cty.Value{}
	pathAttrs := map[string]cty.Value{}
	terraformAttrs := map[string]cty.Value{}
	countAttrs := map[string]cty.Value{}
	forEachAttrs := map[string]cty.Value{}
	var self cty.Value

	for _, ref := range refs {
		rng := ref.SourceRange

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

			if selfAddr == addrs.Self {
				// Programming error: the self address cannot alias itself.
				panic("scope SelfAddr attempting to alias itself")
			}

			// self can only be used within a resource instance
			subj := selfAddr.(addrs.ResourceInstance)

			val, valDiags := normalizeRefValue(s.Data.GetResource(subj.ContainingResource(), rng))

			diags = diags.Append(valDiags)

			// Self is an exception in that it must always resolve to a
			// particular instance. We will still insert the full resource into
			// the context below.
			var hclDiags hcl.Diagnostics
			// We should always have a valid self index by this point, but in
			// the case of an error, self may end up as a cty.DynamicValue.
			switch k := subj.Key.(type) {
			case addrs.IntKey:
				self, hclDiags = hcl.Index(val, cty.NumberIntVal(int64(k)), ref.SourceRange.ToHCL().Ptr())
				diags.Append(hclDiags)
			case addrs.StringKey:
				self, hclDiags = hcl.Index(val, cty.StringVal(string(k)), ref.SourceRange.ToHCL().Ptr())
				diags.Append(hclDiags)
			default:
				self = val
			}
			continue
		}

		// This type switch must cover all of the "Referenceable" implementations
		// in package addrs, however we are removing the possibility of
		// Instances beforehand.
		switch addr := rawSubj.(type) {
		case addrs.ResourceInstance:
			rawSubj = addr.ContainingResource()
		case addrs.ModuleCallInstance:
			rawSubj = addr.Call
		case addrs.AbsModuleCallOutput:
			rawSubj = addr.Call.Call
		}

		switch subj := rawSubj.(type) {
		case addrs.Resource:
			var into map[string]map[string]cty.Value
			switch subj.Mode {
			case addrs.ManagedResourceMode:
				into = managedResources
			case addrs.DataResourceMode:
				into = dataResources
			default:
				panic(fmt.Errorf("unsupported ResourceMode %s", subj.Mode))
			}

			val, valDiags := normalizeRefValue(s.Data.GetResource(subj, rng))
			diags = diags.Append(valDiags)

			r := subj
			if into[r.Type] == nil {
				into[r.Type] = make(map[string]cty.Value)
			}
			into[r.Type][r.Name] = val

		case addrs.ModuleCall:
			val, valDiags := normalizeRefValue(s.Data.GetModule(subj, rng))
			diags = diags.Append(valDiags)
			wholeModules[subj.Name] = val

		case addrs.InputVariable:
			val, valDiags := normalizeRefValue(s.Data.GetInputVariable(subj, rng))
			diags = diags.Append(valDiags)
			inputVariables[subj.Name] = val

		case addrs.LocalValue:
			val, valDiags := normalizeRefValue(s.Data.GetLocalValue(subj, rng))
			diags = diags.Append(valDiags)
			localValues[subj.Name] = val

		case addrs.PathAttr:
			val, valDiags := normalizeRefValue(s.Data.GetPathAttr(subj, rng))
			diags = diags.Append(valDiags)
			pathAttrs[subj.Name] = val

		case addrs.TerraformAttr:
			val, valDiags := normalizeRefValue(s.Data.GetTerraformAttr(subj, rng))
			diags = diags.Append(valDiags)
			terraformAttrs[subj.Name] = val

		case addrs.CountAttr:
			val, valDiags := normalizeRefValue(s.Data.GetCountAttr(subj, rng))
			diags = diags.Append(valDiags)
			countAttrs[subj.Name] = val

		case addrs.ForEachAttr:
			val, valDiags := normalizeRefValue(s.Data.GetForEachAttr(subj, rng))
			diags = diags.Append(valDiags)
			forEachAttrs[subj.Name] = val

		default:
			// Should never happen
			panic(fmt.Errorf("Scope.buildEvalContext cannot handle address type %T", rawSubj))
		}
	}

	for k, v := range buildResourceObjects(managedResources) {
		vals[k] = v
	}
	vals["data"] = cty.ObjectVal(buildResourceObjects(dataResources))
	vals["module"] = cty.ObjectVal(wholeModules)
	vals["var"] = cty.ObjectVal(inputVariables)
	vals["local"] = cty.ObjectVal(localValues)
	vals["path"] = cty.ObjectVal(pathAttrs)
	vals["terraform"] = cty.ObjectVal(terraformAttrs)
	vals["count"] = cty.ObjectVal(countAttrs)
	vals["each"] = cty.ObjectVal(forEachAttrs)
	if self != cty.NilVal {
		vals["self"] = self
	}

	return ctx, diags
}

func buildResourceObjects(resources map[string]map[string]cty.Value) map[string]cty.Value {
	vals := make(map[string]cty.Value)
	for typeName, nameVals := range resources {
		vals[typeName] = cty.ObjectVal(nameVals)
	}
	return vals
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
