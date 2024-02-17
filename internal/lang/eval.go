// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/ext/dynblock"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/blocktoattr"
	"github.com/hashicorp/terraform/internal/tfdiags"
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
	refs, diags := References(s.ParseRef, traversals)

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

	refs, diags := ReferencesInBlock(s.ParseRef, body, schema)

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
	diags = diags.Append(checkForUnknownFunctionDiags(evalDiags))

	return val, diags
}

// EvalSelfBlock evaluates the given body only within the scope of the provided
// object and instance key data. References to the object must use self, and the
// key data will only contain count.index or each.key. The static values for
// terraform and path will also be available in this context.
func (s *Scope) EvalSelfBlock(body hcl.Body, self cty.Value, schema *configschema.Block, keyData instances.RepetitionData) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	spec := schema.DecoderSpec()

	vals := make(map[string]cty.Value)
	vals["self"] = self

	if !keyData.CountIndex.IsNull() {
		vals["count"] = cty.ObjectVal(map[string]cty.Value{
			"index": keyData.CountIndex,
		})
	}
	if !keyData.EachKey.IsNull() {
		vals["each"] = cty.ObjectVal(map[string]cty.Value{
			"key": keyData.EachKey,
		})
	}

	refs, refDiags := References(s.ParseRef, hcldec.Variables(body, spec))
	diags = diags.Append(refDiags)

	terraformAttrs := map[string]cty.Value{}
	pathAttrs := map[string]cty.Value{}

	// We could always load the static values for Path and Terraform values,
	// but we want to parse the references so that we can get source ranges for
	// user diagnostics.
	for _, ref := range refs {
		// we already loaded the self value
		if ref.Subject == addrs.Self {
			continue
		}

		switch subj := ref.Subject.(type) {
		case addrs.PathAttr:
			val, valDiags := normalizeRefValue(s.Data.GetPathAttr(subj, ref.SourceRange))
			diags = diags.Append(valDiags)
			pathAttrs[subj.Name] = val

		case addrs.TerraformAttr:
			val, valDiags := normalizeRefValue(s.Data.GetTerraformAttr(subj, ref.SourceRange))
			diags = diags.Append(valDiags)
			terraformAttrs[subj.Name] = val

		case addrs.CountAttr, addrs.ForEachAttr:
			// each and count have already been handled.

		default:
			// This should have been caught in validation, but point the user
			// to the correct location in case something slipped through.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Invalid reference`,
				Detail:   fmt.Sprintf("The reference to %q is not valid in this context", ref.Subject),
				Subject:  ref.SourceRange.ToHCL().Ptr(),
			})
		}
	}

	vals["path"] = cty.ObjectVal(pathAttrs)
	vals["terraform"] = cty.ObjectVal(terraformAttrs)

	ctx := &hcl.EvalContext{
		Variables: vals,
		Functions: s.Functions(),
	}

	val, decDiags := hcldec.Decode(body, schema.DecoderSpec(), ctx)
	diags = diags.Append(checkForUnknownFunctionDiags(decDiags))
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
	refs, diags := ReferencesInExpr(s.ParseRef, expr)

	ctx, ctxDiags := s.EvalContext(refs)
	diags = diags.Append(ctxDiags)
	if diags.HasErrors() {
		// We'll stop early if we found problems in the references, because
		// it's likely evaluation will produce redundant copies of the same errors.
		return cty.UnknownVal(wantType), diags
	}

	val, evalDiags := expr.Value(ctx)
	diags = diags.Append(checkForUnknownFunctionDiags(evalDiags))

	if wantType != cty.DynamicPseudoType {
		var convErr error
		val, convErr = convert.Convert(val, wantType)
		if convErr != nil {
			val = cty.UnknownVal(wantType)
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Incorrect value type",
				Detail:      fmt.Sprintf("Invalid expression value: %s.", tfdiags.FormatError(convErr)),
				Subject:     expr.Range().Ptr(),
				Expression:  expr,
				EvalContext: ctx,
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
	staticDiags := s.Data.StaticValidateReferences(refs, selfAddr, s.SourceAddr)
	diags = diags.Append(staticDiags)
	if staticDiags.HasErrors() {
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
	outputValues := map[string]cty.Value{}
	pathAttrs := map[string]cty.Value{}
	terraformAttrs := map[string]cty.Value{}
	countAttrs := map[string]cty.Value{}
	forEachAttrs := map[string]cty.Value{}
	checkBlocks := map[string]cty.Value{}
	runBlocks := map[string]cty.Value{}
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
					Detail:  `The "self" object is not available in this context. This object can be used only in resource provisioner, connection, and postcondition blocks.`,
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
				diags = diags.Append(hclDiags)
			case addrs.StringKey:
				self, hclDiags = hcl.Index(val, cty.StringVal(string(k)), ref.SourceRange.ToHCL().Ptr())
				diags = diags.Append(hclDiags)
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
		case addrs.ModuleCallInstanceOutput:
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

		case addrs.OutputValue:
			val, valDiags := normalizeRefValue(s.Data.GetOutput(subj, rng))
			diags = diags.Append(valDiags)
			outputValues[subj.Name] = val

		case addrs.Check:
			val, valDiags := normalizeRefValue(s.Data.GetCheckBlock(subj, rng))
			diags = diags.Append(valDiags)
			checkBlocks[subj.Name] = val

		case addrs.Run:
			val, valDiags := normalizeRefValue(s.Data.GetRunBlock(subj, rng))
			diags = diags.Append(valDiags)
			runBlocks[subj.Name] = val

		default:
			// Should never happen
			panic(fmt.Errorf("Scope.buildEvalContext cannot handle address type %T", rawSubj))
		}
	}

	// Managed resources are exposed in two different locations. The primary
	// is at the top level where the resource type name is the root of the
	// traversal, but we also expose them under "resource" as an escaping
	// technique if we add a reserved name in a future language edition which
	// conflicts with someone's existing provider.
	for k, v := range buildResourceObjects(managedResources) {
		vals[k] = v
	}
	vals["resource"] = cty.ObjectVal(buildResourceObjects(managedResources))

	vals["data"] = cty.ObjectVal(buildResourceObjects(dataResources))
	vals["module"] = cty.ObjectVal(wholeModules)
	vals["var"] = cty.ObjectVal(inputVariables)
	vals["local"] = cty.ObjectVal(localValues)
	vals["path"] = cty.ObjectVal(pathAttrs)
	vals["terraform"] = cty.ObjectVal(terraformAttrs)
	vals["count"] = cty.ObjectVal(countAttrs)
	vals["each"] = cty.ObjectVal(forEachAttrs)

	// Checks, outputs, and run blocks are conditionally included in the
	// available scope, so we'll only write out their values if we actually have
	// something for them.
	if len(checkBlocks) > 0 {
		vals["check"] = cty.ObjectVal(checkBlocks)
	}

	if len(outputValues) > 0 {
		vals["output"] = cty.ObjectVal(outputValues)
	}

	if len(runBlocks) > 0 {
		vals["run"] = cty.ObjectVal(runBlocks)
	}

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

// checkForUnknownFunctionDiags inspects the diagnostics for errors from unknown
// function calls, and tailors the messages to better suit Terraform. We now
// have multiple namespaces where functions may be declared, and it's up to the
// user to have properly configured the module to populate the provider
// namespace. The generic unknown function diagnostic from hcl does not direct
// the user on how to remedy the situation in Terraform, and we can give more
// useful information in a few Terraform specific cases here.
func checkForUnknownFunctionDiags(diags hcl.Diagnostics) hcl.Diagnostics {
	for _, d := range diags {
		extra, ok := hcl.DiagnosticExtra[hclsyntax.FunctionCallUnknownDiagExtra](d)
		if !ok {
			continue
		}
		name := extra.CalledFunctionName()
		namespace := extra.CalledFunctionNamespace()
		namespaceParts := strings.Split(namespace, "::")
		if len(namespaceParts) < 2 {
			// no namespace (namespace includes ::, so will have at least 2
			// parts), but check if there is a matching name in a provider
			// namspace.
			if d.EvalContext == nil {
				continue
			}

			for funcName := range d.EvalContext.Functions {
				if strings.HasSuffix(funcName, "::"+name) {
					d.Detail = fmt.Sprintf("%s Did you mean %q?", d.Detail, funcName)
					break
				}
			}
			continue
		}

		// the diagnostic isn't really shared with anything, and copying would
		// still retain the internal pointers, so we're going to modify the
		// diagnostic in-place if we want to change the output. Log the original
		// diagnostic for debugging purposes in case we overwrite something
		// potentially useful in the future from hcl.
		log.Printf("[ERROR] UnknownFunctionCall: %s", d.Error())
		d.Summary = "Unknown provider function"

		if namespaceParts[0] != "provider" {
			// help if the user is skipping the provider:: prefix before the
			// provider name.
			d.Detail = fmt.Sprintf(`The function namespace %q is not valid. Provider function calls must use the "provider::" namespace prefix.`, namespaceParts[0])
			continue
		}

		if namespaceParts[1] == "" {
			// missing provider name entirely
			d.Detail = `The function call must include the provider name after the "provider::" prefix.`
			continue
		}

		if d.EvalContext == nil {
			// There's no eval context for some reason, so we can't inspect the
			// available functions.
			d.Detail = fmt.Sprintf(`There is no function named "%s%s".`, namespace, name)
			continue
		}

		otherProviderFuncs := false
		for funcName := range d.EvalContext.Functions {
			// there are other functions in this provider namespace, so it must
			// have been included in the configuration, and we can be clear that
			// this a function which the provider does not support.
			if strings.HasPrefix(funcName, namespace) {
				otherProviderFuncs = true
				break
			}
		}
		if otherProviderFuncs {
			d.Detail = fmt.Sprintf("The function %q is not available from the provider %q.", name, namespaceParts[1])
			continue
		}

		// no other functions exist for this provider, so hint that the user may
		// need to include it in the configuration.
		d.Detail = fmt.Sprintf(`There is no function named "%s%s". Ensure that provider name %q is declared in this module's required_providers block, and that this provider offers a function named %q.`, namespace, name, namespaceParts[1], name)
	}

	return diags
}
