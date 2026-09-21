// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/lang/simplerefs"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type PolicyCallbackManager struct {
	WalkOperation walkOperation
	Schema        providers.GetProviderSchemaResponse
	Config        *configs.Config

	// resources is a map of resource addresses to their policy resources.
	resources addrs.Map[addrs.AbsResourceInstance, *PolicyResource]

	// closed-by-default (M11) accounting, populated while the reference callback
	// runs. mu guards them because a policy's relationships may be resolved
	// concurrently within a single subject evaluation.
	mu           sync.Mutex
	sawPartial   bool
	undecidables []undecidableRef
}

func NewPolicyCallbackManager(walkOperation walkOperation, schema providers.GetProviderSchemaResponse, config *configs.Config) *PolicyCallbackManager {
	return &PolicyCallbackManager{
		WalkOperation: walkOperation,
		Schema:        schema,
		Config:        config,
	}
}

// PolicyResource co-locates the data required for the relationship analysis for a single resource
type PolicyResource struct {
	Addr       addrs.AbsResourceInstance
	ConfigBody hcl.Body
	Schema     *configschema.Block
	Value      cty.Value
}

func (cb *PolicyCallbackManager) RelatedResourcesCallback(ctx EvalContext, subjectAddr addrs.AbsResourceInstance, val cty.Value) func(context.Context, *callback.RelationshipBlock) (callback.RelatedResource, error) {
	return func(_ context.Context, blk *callback.RelationshipBlock) (callback.RelatedResource, error) {
		// Closed-by-default (M11): a require_reference relationship must be fully
		// decidable at plan time so the policy's enforce blocks evaluate on
		// complete information. We collect the source location of every connector
		// expression that could not be statically resolved. If the relationship
		// comes back partial, we record it so node_policy_resource can halt the
		// plan (with those exact locations) rather than silently deferring.
		var undecidable []undecidableRef
		related, err := cb.getRelatedResources(ctx, subjectAddr, blk, val, &undecidable)
		if err != nil {
			return related, err
		}
		if related.Partial {
			cb.mu.Lock()
			cb.sawPartial = true
			cb.undecidables = append(cb.undecidables, undecidable...)
			cb.mu.Unlock()
		}
		return related, nil
	}
}

// UndecidableResult reports whether a require_reference relationship could not be
// fully resolved at plan time during this manager's evaluation, along with the
// located undecidable connector expressions (deduplicated). node_policy_resource
// uses this to enforce the closed-by-default rule: if true, planning halts.
func (cb *PolicyCallbackManager) UndecidableResult() (bool, []undecidableRef) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.sawPartial, cb.undecidables
}

// GetRelatedResources returns the related resources for the given target
// resource type and connection. It does not enforce the closed-by-default rule
// (that lives in the RelatedResourcesCallback wrapper); callers that need the
// undecidable-expression locations use getRelatedResources directly.
func (cb *PolicyCallbackManager) GetRelatedResources(ctx EvalContext, subjectAddr addrs.AbsResourceInstance, blk *callback.RelationshipBlock, val cty.Value) (callback.RelatedResource, error) {
	return cb.getRelatedResources(ctx, subjectAddr, blk, val, nil)
}

// getRelatedResources is the resolution core. When undecidable is non-nil, the
// source range of every connector expression that resolves to a structurally
// undecidable outcome is appended to it, so the caller can report the exact
// lines responsible for an unknown relationship.
func (cb *PolicyCallbackManager) getRelatedResources(ctx EvalContext, subjectAddr addrs.AbsResourceInstance, blk *callback.RelationshipBlock, val cty.Value, undecidable *[]undecidableRef) (callback.RelatedResource, error) {
	found := make([]callback.RelatedResource, 0)
	partial := false
	var err error
	policyGraph := ctx.PolicyGraph()
	subjectResource := policyGraph.GetResource(subjectAddr)

	// Consider an example where the terraform config is:
	// resource "aws_s3_bucket" "example" {
	//   bucket = "my-bucket"
	// }
	// resource "aws_s3_bucket_acl" "example" {
	//   bucket = aws_s3_bucket.example.id
	// }
	// and the relationship block pair is
	// { sourceAttribute: "id", relatedAttribute: "bucket" }

	for addr, related := range policyGraph.resourceMap.Iter() {
		relatedAddr := addr.ConfigResource()
		if relatedAddr.Resource.Type != blk.RelatedType {
			continue
		}

		// Skip the subject resource, i.e aws_s3_bucket.example
		if relatedAddr.Equal(subjectAddr.ConfigResource()) {
			continue
		}

		// Deferred candidates make the overall answer incomplete.
		if ctx.Deferrals().DependenciesDeferred([]addrs.ConfigResource{relatedAddr}) {
			partial = true
			continue
		}

		// If the current iteration is for aws_s3_bucket_acl.example, we will
		// check for the given related attribute pair to match aws_s3_bucket.example.
		// We do that by checking if the related attribute (e.g. bucket) is a literal value
		// or a simple traversal.
		// If it is a literal value, we check if it matches relationship.QueryAttributes.
		// If it is a traversal, we check if the traversal points to aws_s3_bucket.example.id.
		resourceValue := related.Value
		matched := cb.matchWithAcc(ctx, subjectResource, related, blk, undecidable)
		if matched.IsWhollyKnown() && matched.True() {
			resourceValue, _ = related.Value.UnmarkDeep()

			// If the resource matched and the relationship has a nested block,
			// this is a multi-hop relationship: recurse with the matched
			// candidate as the new subject and collapse the recursion to the
			// terminal targets of the deepest hop. An outer candidate that
			// reaches no terminal (a broken chain) therefore contributes
			// nothing to related.<name>, which is the correct multi-hop
			// existence semantic. When there is no nested block, the matched
			// candidate itself is the terminal target.
			if blk.Nested != nil {
				nestedRes, nestedErr := cb.getRelatedResources(ctx, related.Addr, blk.Nested, resourceValue, undecidable)
				if nestedErr != nil {
					err = nestedErr
					continue
				}
				found = append(found, nestedRes.Related...)
				partial = partial || nestedRes.Partial
			} else {
				found = append(found, callback.RelatedResource{Value: resourceValue})
			}
		}
		partial = partial || !matched.IsWhollyKnown()
	}

	return callback.RelatedResource{
		Related: found,
		Partial: partial,
		Value:   val,
	}, err
}

// matchOutcome is the result of checking one reference direction of a
// relationship connector. It is a four-valued logic so the two directions can
// be combined without a "not evaluable" direction (a missing config body)
// spuriously forcing the overall result to unknown.
type matchOutcome int

const (
	// matchNA means this direction could not be evaluated at all (the config
	// body it needs is unavailable). It is ignored when combining directions.
	matchNA matchOutcome = iota
	matchTrue
	matchFalse
	matchUnknown
	// matchUndecidable means the connector expression is *structurally* too
	// complex to resolve statically (an opaque function, a value that may
	// reference more than one resource, or a reference mixed with a constant).
	// Unlike matchUnknown (a value not yet known), this is a config-complexity
	// problem the author can fix. It is treated like matchUnknown when combining
	// directions, but the closed-by-default check (M11) turns it into an error
	// citing the exact source lines rather than silently deferring.
	matchUndecidable
)

// undecidableRef locates a connector expression that could not be statically
// resolved, so the closed-by-default check can report the exact lines.
type undecidableRef struct {
	resource addrs.AbsResourceInstance
	attr     string
	reason   string
	rng      hcl.Range
}

// Match reports whether the related candidate is genuinely connected to the
// subject by a config reference on the connector's attribute pairs.
//
// A connector is an *undirected* reference edge: the reference may live on
// either resource. We therefore check both directions and OR the outcomes:
//   - Inbound: the target (related) config references the subject
//     (e.g. aws_s3_bucket_acl.bucket = aws_s3_bucket.this.id).
//   - Outbound: the subject config references the target (related)
//     (e.g. aws_instance.subnet_id = aws_subnet.this.id).
//
// This makes provenance hold regardless of which side holds the reference, and
// lets multi-hop chains verify each hop structurally in either direction
// (Terraform Core's GetRelatedResources recursion calls Match per hop).
func (c *PolicyCallbackManager) Match(ctx EvalContext, subject, related *PolicyResource, conn *callback.RelationshipBlock) cty.Value {
	return c.matchWithAcc(ctx, subject, related, conn, nil)
}

// matchWithAcc is Match with an optional accumulator for the source locations of
// connector expressions that resolve to a structurally undecidable outcome.
func (c *PolicyCallbackManager) matchWithAcc(ctx EvalContext, subject, related *PolicyResource, conn *callback.RelationshipBlock, undecidable *[]undecidableRef) cty.Value {
	// Value fallback (direction-agnostic): a non-null QueryAttributes filter
	// matches the candidate by value, bypassing the reference comparison.
	if !conn.QueryAttributes.IsNull() {
		filterMap := conn.QueryAttributes.AsValueMap()
		if matches, _ := resourceMatchesFilter(related.Addr.ConfigResource(), related.Schema, filterMap, related.Value); matches {
			return cty.True
		}
	}

	inbound := c.checkDirection(ctx, subject, related, conn, false, undecidable)
	if inbound == matchTrue {
		return cty.True
	}
	outbound := c.checkDirection(ctx, subject, related, conn, true, undecidable)
	return combineMatch(inbound, outbound)
}

// checkDirection verifies a single reference direction of the connector. When
// outbound is false (inbound) it parses the related (target) config and expects
// each connector's related attribute to reference the subject's subject
// attribute. When outbound is true it parses the subject config and expects the
// subject attribute to reference the related (target) attribute. All attribute
// pairs must match (logical AND).
func (c *PolicyCallbackManager) checkDirection(ctx EvalContext, subject, related *PolicyResource, conn *callback.RelationshipBlock, outbound bool, undecidable *[]undecidableRef) matchOutcome {
	bodyResource := related
	if outbound {
		bodyResource = subject
	}
	if bodyResource.ConfigBody == nil {
		return matchNA
	}

	bodySimple, diags := hclsyntax.ParseSimpleBody(bodyResource.ConfigBody)
	if diags.HasErrors() {
		return matchUnknown
	}

	for _, pair := range conn.AttributePairs {
		// Assign the attribute-on-body and the attribute-on-target for this
		// direction. Inbound: body=related.RelatedAttribute -> subject.SubjectAttribute.
		// Outbound: body=subject.SubjectAttribute -> related.RelatedAttribute.
		bodyAttr, targetAttr := pair.RelatedAttribute, pair.SubjectAttribute
		targetResource := subject
		if outbound {
			bodyAttr, targetAttr = pair.SubjectAttribute, pair.RelatedAttribute
			targetResource = related
		}

		// The target must actually have the attribute we compare against.
		if !targetResource.Value.Type().IsObjectType() || !targetResource.Value.Type().HasAttribute(targetAttr) {
			return matchUnknown
		}

		bodyTraversal, _ := hclsyntax.ParseTraversalAbs([]byte(bodyAttr), "", hcl.InitialPos)
		path, _ := traversalToPath(bodyTraversal)
		bodyExpr, found := getAttributeFromBody(bodySimple, path, bodyResource.Schema)
		if !found {
			// The connector attribute is not set in this config, so it cannot
			// carry a reference in this direction.
			return matchFalse
		}

		// Compare at config-address granularity (see equalRef).
		targetRef := &globalref.Reference{
			ContainerAddr: targetResource.Addr.Module,
			LocalRef: &addrs.Reference{
				Subject:   targetResource.Addr.Resource,
				Remaining: hcl.Traversal{hcl.TraverseAttr{Name: targetAttr}},
			},
		}

		// Choose the decidability rule by connector shape. A collection-valued
		// connector expression (a tuple/list literal, a splat, a `for`, or a
		// set-preserving function) is a *membership* (expansion) connector:
		// every coexisting element is a real member, so the target matches when
		// it is among them. Anything else is scalar (exactly-one) semantics.
		var outcome matchOutcome
		var reason string
		if isMembershipExpr(bodyExpr) {
			outcome, reason = c.connectorMembershipOutcome(ctx, bodyResource.Addr.Module, bodyExpr, targetRef)
		} else {
			outcome, reason = c.connectorRefOutcome(ctx, bodyResource.Addr.Module, bodyExpr, targetRef)
		}
		if outcome == matchUndecidable {
			// Record the exact source location of the complex expression so the
			// closed-by-default check can report it, then treat it as unknown
			// for the four-valued match logic (so the relationship is marked
			// partial rather than silently matched/denied).
			recordUndecidable(undecidable, bodyResource.Addr, bodyAttr, reason, bodyExpr)
			return matchUnknown
		}
		switch outcome {
		case matchFalse:
			return matchFalse
		case matchUnknown:
			return matchUnknown
		}
		// matchTrue: this pair holds; continue to the next pair (AND semantics).
	}

	return matchTrue
}

// connectorRefOutcome decides, for a single connector attribute pair, whether
// the body expression genuinely references the target resource attribute. It
// reduces the connector expression to the set of resource-attribute references
// it can yield (handling plain traversals, index/expansion steps, splats,
// conditionals, and allowlisted reference-preserving functions), resolves each
// through the reference graph, and applies the decidability rule:
//   - a single homogeneous resource reference, no constant branch -> compare to
//     the target (matchTrue / matchFalse);
//   - only constant branches -> matchFalse (a hardcoded value is not a
//     reference);
//   - an unresolved single reference (a value not yet known) -> matchUnknown
//     (defer, not a config-complexity problem);
//   - two or more distinct references, a reference mixed with a constant, or an
//     opaque/value-derived expression -> matchUndecidable, with a short reason
//     describing why (used to report the exact source lines under M11).
func (c *PolicyCallbackManager) connectorRefOutcome(ctx EvalContext, bodyModule addrs.ModuleInstance, bodyExpr hclsyntax.SimpleAttribute, targetRef *globalref.Reference) (matchOutcome, string) {
	if bodyExpr.IsLiteral() {
		return matchFalse, ""
	}

	var traversals []hcl.Traversal
	hasNonRef := false
	if bodyExpr.IsTraversal() {
		traversals = []hcl.Traversal{bodyExpr.Traversal}
	} else {
		var ok bool
		traversals, hasNonRef, ok = simplerefs.DecomposeTraversals(bodyExpr.Expr)
		if !ok {
			return matchUndecidable, "uses an unsupported function or a value-derived expression that cannot be statically resolved to a resource reference"
		}
	}

	tree := ctx.ResourceAttrRefGraph()
	var resolved []*globalref.Reference
	for _, tr := range traversals {
		ref, refDiags := globalref.ParseRef(bodyModule, tr)
		if refDiags.HasErrors() {
			log.Printf("[TRACE] global ref parse error: %s", refDiags.Err())
			return matchUnknown, ""
		}
		attrRef, ok := tree.ResolveReference(ref)
		if ok {
			resolved = append(resolved, attrRef)
			continue
		}
		// A reference that resolves through the graph to a constant is a
		// definitive non-reference branch (a hardcoded value laundered through a
		// local/output); anything else unresolved is genuinely unknown (a value
		// not yet known), which defers rather than erroring.
		if tree.ResolvesToLiteral(ref) {
			hasNonRef = true
			continue
		}
		return matchUnknown, ""
	}

	distinct := dedupRefs(resolved)
	if hasNonRef {
		if len(distinct) == 0 {
			return matchFalse, ""
		}
		// A resource reference mixed with a constant branch: the value depends
		// on inputs not known until apply, so provenance cannot be decided.
		return matchUndecidable, "may be a resource reference or a constant value depending on inputs not known until apply"
	}
	switch len(distinct) {
	case 0:
		return matchUnknown, ""
	case 1:
		if equalRef(targetRef, distinct[0]) {
			return matchTrue, ""
		}
		return matchFalse, ""
	default:
		// Multiple distinct resources reachable: the value could come from any
		// of them, so the specific target cannot be determined.
		return matchUndecidable, "may reference more than one resource, so the specific target cannot be determined"
	}
}

// isMembershipExpr reports whether the connector body expression is a
// collection-producing construct — a tuple/list literal, a splat, a `for`
// expression, or a set-preserving function (concat/flatten/tolist/…). These are
// the expansion (`any_of`/`has`) connectors, where every coexisting element is a
// genuine member. A bare traversal, a scalar function, or a conditional is not a
// membership expression (it is handled by the scalar rule), so scalar connectors
// — including ones that use element()/one()/try() to select a single reference —
// are unaffected.
func isMembershipExpr(bodyExpr hclsyntax.SimpleAttribute) bool {
	if bodyExpr.IsLiteral() || bodyExpr.IsTraversal() || bodyExpr.Expr == nil {
		return false
	}
	switch e := hcl.UnwrapExpression(bodyExpr.Expr).(type) {
	case *hclsyntax.TupleConsExpr, *hclsyntax.SplatExpr, *hclsyntax.ForExpr:
		return true
	case *hclsyntax.FunctionCallExpr:
		return simplerefs.IsMembershipPreservingFunc(e.Name)
	default:
		return false
	}
}

// connectorMembershipOutcome decides an expansion (membership) connector: the
// body is a collection whose coexisting element references are decomposed, and
// the target matches if it is among them. Unlike the scalar rule, many distinct
// references are expected (each element is a real member), so this never returns
// undecidable for "multiple resources" — only for a *choice* construct that the
// membership decomposition rejects (a conditional or a selection function), which
// cannot be resolved to a fixed set at plan time.
func (c *PolicyCallbackManager) connectorMembershipOutcome(ctx EvalContext, bodyModule addrs.ModuleInstance, bodyExpr hclsyntax.SimpleAttribute, targetRef *globalref.Reference) (matchOutcome, string) {
	traversals, _, ok := simplerefs.DecomposeMembershipTraversals(bodyExpr.Expr)
	if !ok {
		return matchUndecidable, "is a collection built from a conditional or a selection function (element/one/try/coalesce/slice), so its members cannot be determined until apply"
	}

	tree := ctx.ResourceAttrRefGraph()
	sawUnknown := false
	for _, tr := range traversals {
		ref, refDiags := globalref.ParseRef(bodyModule, tr)
		if refDiags.HasErrors() {
			log.Printf("[TRACE] global ref parse error: %s", refDiags.Err())
			sawUnknown = true
			continue
		}
		attrRef, resolved := tree.ResolveReference(ref)
		if resolved {
			if equalRef(targetRef, attrRef) {
				// The target is genuinely a member of the collection.
				return matchTrue, ""
			}
			continue
		}
		// A member that resolves to a constant is a definitive non-reference and
		// cannot be the target; any other unresolved member is a value not yet
		// known, so we cannot rule out that it is the target.
		if !tree.ResolvesToLiteral(ref) {
			sawUnknown = true
		}
	}
	if sawUnknown {
		return matchUnknown, ""
	}
	return matchFalse, ""
}

// recordUndecidable appends the source location of a structurally undecidable
// connector expression to acc (deduplicated), so the closed-by-default check can
// cite the exact lines. A nil acc (the plain GetRelatedResources entrypoint)
// records nothing.
func recordUndecidable(acc *[]undecidableRef, resource addrs.AbsResourceInstance, attr, reason string, bodyExpr hclsyntax.SimpleAttribute) {
	if acc == nil {
		return
	}
	rng := bodyExpr.Range
	if bodyExpr.Expr != nil {
		rng = bodyExpr.Expr.Range()
	}
	for _, existing := range *acc {
		if existing.resource.Equal(resource) && existing.attr == attr && existing.rng == rng {
			return
		}
	}
	*acc = append(*acc, undecidableRef{resource: resource, attr: attr, reason: reason, rng: rng})
}

// undecidableRelationshipDiagnostics builds the closed-by-default diagnostics for
// a require_reference relationship that could not be fully resolved at plan time.
// Each located undecidable connector expression becomes an error whose Subject is
// the exact source range of the offending expression, so the practitioner sees a
// snippet of the precise lines to rewrite. When no specific expression was
// located (a rare value-unknown/deferred case) a single generic error is
// returned so planning still halts.
func undecidableRelationshipDiagnostics(subject addrs.AbsResourceInstance, refs []undecidableRef) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if len(refs) == 0 {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Relationship cannot be verified at plan time",
			Detail: fmt.Sprintf(
				"The require_reference relationship for %s could not be fully determined at plan time because a related resource is deferred or has a value that is not known yet. require_reference requires the relationship to be resolvable during planning; remove require_reference to allow value-based matching, or adjust the configuration so the related resources are known at plan time.",
				subject,
			),
		})
	}
	seen := make(map[string]bool, len(refs))
	for _, r := range refs {
		key := r.rng.String() + "|" + r.attr
		if seen[key] {
			continue
		}
		seen[key] = true
		rng := r.rng
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Relationship cannot be verified at plan time",
			Detail: fmt.Sprintf(
				"The require_reference relationship for %s cannot be verified because this expression, which sets %s attribute %q, %s.\n\nPlanning cannot continue because require_reference requires every hop of the relationship to be resolvable to a genuine reference at plan time. Rewrite this to a direct reference (for example, aws_subnet.example.id), or remove require_reference to allow value-based matching.",
				subject, r.resource, r.attr, r.reason,
			),
			Subject: rng.Ptr(),
		})
	}
	return diags
}

// dedupRefs returns the input references de-duplicated at config-address
// granularity.
func dedupRefs(refs []*globalref.Reference) []*globalref.Reference {
	out := make([]*globalref.Reference, 0, len(refs))
	for _, r := range refs {
		seen := false
		for _, o := range out {
			if equalRef(r, o) {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, r)
		}
	}
	return out
}

// combineMatch ORs the outcomes of the two reference directions into a
// three-valued cty.Bool. A direction that could not be evaluated (matchNA) is
// ignored so it never forces the result to unknown; unknown only wins over a
// definitive false.
func combineMatch(a, b matchOutcome) cty.Value {
	if a == matchTrue || b == matchTrue {
		return cty.True
	}
	sawFalse := false
	for _, o := range [...]matchOutcome{a, b} {
		switch o {
		case matchUnknown:
			return cty.UnknownVal(cty.Bool)
		case matchFalse:
			sawFalse = true
		}
	}
	if sawFalse {
		return cty.False
	}
	// Both directions were not-applicable: we genuinely cannot tell.
	return cty.UnknownVal(cty.Bool)
}

// equalRef reports whether two references point at the same resource attribute
// at **config-address granularity**: the module *config* path plus the resource
// type.name plus the attribute path, with module-call instance keys and resource
// instance keys (and expansion index expressions such as count.index/each.key,
// which are dropped before parsing) normalized away.
//
// This granularity is what makes provenance detection work uniformly across
// module boundaries and expansion (count/for_each on either the resource or an
// enclosing module call): a genuine reference to the subject resource is
// detected regardless of which specific instances are involved. Instance-key
// precision (e.g. requiring acl[0] to reference bucket[0] specifically) is an
// explicit non-goal of this prototype.
func equalRef(ref *globalref.Reference, other *globalref.Reference) bool {
	if ref == nil || other == nil {
		return false
	}
	refAttr, ok := ref.ResourceAttr()
	if !ok {
		return false
	}
	otherAttr, ok := other.ResourceAttr()
	if !ok {
		return false
	}
	if !refAttr.Resource.ConfigResource().Equal(otherAttr.Resource.ConfigResource()) {
		return false
	}
	return refAttr.Attr.Equals(otherAttr.Attr)
}

// getAttributeFromBody looks up an attribute expression inside a parsed simple body
// tree using a block/attribute path.
func getAttributeFromBody(simpleBody *hclsyntax.SimpleBody, path cty.Path, resourceSchema *configschema.Block) (hclsyntax.SimpleAttribute, bool) {
	var attr hclsyntax.SimpleAttribute
	if len(path) == 0 {
		return attr, false
	}

	remaining := path[1:]
	switch step := path[0].(type) {
	case cty.GetAttrStep:
		// terminating condition
		if len(path) == 1 {
			attr, ok := simpleBody.Attributes[step.Name]
			return attr, ok
		}

		blk := resourceSchema.BlockTypes[step.Name]
		// If it is not a block, then it should have already been handled as an attribute
		if blk == nil {
			return attr, false
		}
		// If the block is expected to be a single block, we can just
		// get the first block and treat it as such
		if blk.Nesting == configschema.NestingSingle || blk.Nesting == configschema.NestingGroup {
			if len(simpleBody.Blocks) == 0 {
				return attr, false
			}
			final, ok := getAttributeFromBody(simpleBody.Blocks[0].Body, remaining, &blk.Block)
			return final, ok
		}

		// Now we treat other kinds of repeated blocks
		blocks := make(map[string][]hclsyntax.SimpleBlock)
		// group the blocks by type
		for _, block := range simpleBody.Blocks {
			if _, ok := blocks[block.Type]; !ok {
				blocks[block.Type] = make([]hclsyntax.SimpleBlock, 0, len(simpleBody.Blocks))
			}
			blocks[block.Type] = append(blocks[block.Type], block)
		}

		currentBlock, ok := blocks[step.Name]
		if !ok {
			return attr, false
		}
		// if the block is a repeated block, then the next step
		// has to be an index step.
		if len(remaining) == 0 {
			return attr, false
		}
		indexStep, ok := remaining[0].(cty.IndexStep)
		if !ok {
			return attr, false
		}

		if blk.Nesting == configschema.NestingList {
			idx, _ := indexStep.Key.AsBigFloat().Int64()
			current := currentBlock[idx]
			remaining = remaining[1:]

			final, ok := getAttributeFromBody(current.Body, remaining, &blk.Block)
			if ok {
				return final, true
			}
		} else if blk.Nesting == configschema.NestingMap {
			for _, block := range currentBlock {
				if block.Labels[0] == indexStep.Key.AsString() {
					remaining = remaining[1:]
					final, ok := getAttributeFromBody(block.Body, remaining, &blk.Block)
					return final, ok
				}
			}
		}
	default:
		return attr, false
	}

	return attr, false
}
