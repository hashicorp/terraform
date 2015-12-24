package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/dot"
)

// GraphNodeCountDependent is implemented by resources for giving only
// the dependencies they have from the "count" field.
type GraphNodeCountDependent interface {
	CountDependentOn() []string
}

// GraphNodeConfigResource represents a resource within the config graph.
type GraphNodeConfigResource struct {
	Resource *config.Resource

	// If this is set to anything other than destroyModeNone, then this
	// resource represents a resource that will be destroyed in some way.
	DestroyMode GraphNodeDestroyMode

	// Used during DynamicExpand to target indexes
	Targets []ResourceAddress

	Path []string
}

func (n *GraphNodeConfigResource) ConfigType() GraphNodeConfigType {
	return GraphNodeConfigTypeResource
}

func (n *GraphNodeConfigResource) DependableName() []string {
	return []string{n.Resource.Id()}
}

// GraphNodeCountDependent impl.
func (n *GraphNodeConfigResource) CountDependentOn() []string {
	result := make([]string, 0, len(n.Resource.RawCount.Variables))
	for _, v := range n.Resource.RawCount.Variables {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}

	return result
}

// GraphNodeDependent impl.
func (n *GraphNodeConfigResource) DependentOn() []string {
	result := make([]string, len(n.Resource.DependsOn),
		(len(n.Resource.RawCount.Variables)+
			len(n.Resource.RawConfig.Variables)+
			len(n.Resource.DependsOn))*2)
	copy(result, n.Resource.DependsOn)

	for _, v := range n.Resource.RawCount.Variables {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}
	for _, v := range n.Resource.RawConfig.Variables {
		if vn := varNameForVar(v); vn != "" {
			result = append(result, vn)
		}
	}
	for _, p := range n.Resource.Provisioners {
		for _, v := range p.ConnInfo.Variables {
			if vn := varNameForVar(v); vn != "" && vn != n.Resource.Id() {
				result = append(result, vn)
			}
		}
		for _, v := range p.RawConfig.Variables {
			if vn := varNameForVar(v); vn != "" && vn != n.Resource.Id() {
				result = append(result, vn)
			}
		}
	}

	return result
}

// VarWalk calls a callback for all the variables that this resource
// depends on.
func (n *GraphNodeConfigResource) VarWalk(fn func(config.InterpolatedVariable)) {
	for _, v := range n.Resource.RawCount.Variables {
		fn(v)
	}
	for _, v := range n.Resource.RawConfig.Variables {
		fn(v)
	}
	for _, p := range n.Resource.Provisioners {
		for _, v := range p.ConnInfo.Variables {
			fn(v)
		}
		for _, v := range p.RawConfig.Variables {
			fn(v)
		}
	}
}

func (n *GraphNodeConfigResource) Name() string {
	result := n.Resource.Id()
	switch n.DestroyMode {
	case DestroyNone:
	case DestroyPrimary:
		result += " (destroy)"
	case DestroyTainted:
		result += " (destroy tainted)"
	default:
		result += " (unknown destroy type)"
	}

	return result
}

// GraphNodeDotter impl.
func (n *GraphNodeConfigResource) DotNode(name string, opts *GraphDotOpts) *dot.Node {
	if n.DestroyMode != DestroyNone && !opts.Verbose {
		return nil
	}
	return dot.NewNode(name, map[string]string{
		"label": n.Name(),
		"shape": "box",
	})
}

// GraphNodeFlattenable impl.
func (n *GraphNodeConfigResource) Flatten(p []string) (dag.Vertex, error) {
	return &GraphNodeConfigResourceFlat{
		GraphNodeConfigResource: n,
		PathValue:               p,
	}, nil
}

// GraphNodeDynamicExpandable impl.
func (n *GraphNodeConfigResource) DynamicExpand(ctx EvalContext) (*Graph, error) {
	state, lock := ctx.State()
	lock.RLock()
	defer lock.RUnlock()

	// Start creating the steps
	steps := make([]GraphTransformer, 0, 5)

	// Primary and non-destroy modes are responsible for creating/destroying
	// all the nodes, expanding counts.
	switch n.DestroyMode {
	case DestroyNone, DestroyPrimary:
		steps = append(steps, &ResourceCountTransformer{
			Resource: n.Resource,
			Destroy:  n.DestroyMode != DestroyNone,
			Targets:  n.Targets,
		})
	}

	// Additional destroy modifications.
	switch n.DestroyMode {
	case DestroyPrimary:
		// If we're destroying the primary instance, then we want to
		// expand orphans, which have all the same semantics in a destroy
		// as a primary.
		steps = append(steps, &OrphanTransformer{
			State:   state,
			View:    n.Resource.Id(),
			Targets: n.Targets,
		})

		steps = append(steps, &DeposedTransformer{
			State: state,
			View:  n.Resource.Id(),
		})
	case DestroyTainted:
		// If we're only destroying tainted resources, then we only
		// want to find tainted resources and destroy them here.
		steps = append(steps, &TaintedTransformer{
			State: state,
			View:  n.Resource.Id(),
		})
	}

	// Always end with the root being added
	steps = append(steps, &RootTransformer{})

	// Build the graph
	b := &BasicGraphBuilder{Steps: steps}
	return b.Build(ctx.Path())
}

// GraphNodeAddressable impl.
func (n *GraphNodeConfigResource) ResourceAddress() *ResourceAddress {
	return &ResourceAddress{
		Path:         n.Path[1:],
		Index:        -1,
		InstanceType: TypePrimary,
		Name:         n.Resource.Name,
		Type:         n.Resource.Type,
	}
}

// GraphNodeTargetable impl.
func (n *GraphNodeConfigResource) SetTargets(targets []ResourceAddress) {
	n.Targets = targets
}

// GraphNodeEvalable impl.
func (n *GraphNodeConfigResource) EvalTree() EvalNode {
	return &EvalSequence{
		Nodes: []EvalNode{
			&EvalInterpolate{Config: n.Resource.RawCount},
			&EvalOpFilter{
				Ops:  []walkOperation{walkValidate},
				Node: &EvalValidateCount{Resource: n.Resource},
			},
			&EvalCountFixZeroOneBoundary{Resource: n.Resource},
		},
	}
}

// GraphNodeProviderConsumer
func (n *GraphNodeConfigResource) ProvidedBy() []string {
	return []string{resourceProvider(n.Resource.Type, n.Resource.Provider)}
}

// GraphNodeProvisionerConsumer
func (n *GraphNodeConfigResource) ProvisionedBy() []string {
	result := make([]string, len(n.Resource.Provisioners))
	for i, p := range n.Resource.Provisioners {
		result[i] = p.Type
	}

	return result
}

// GraphNodeDestroyable
func (n *GraphNodeConfigResource) DestroyNode(mode GraphNodeDestroyMode) GraphNodeDestroy {
	// If we're already a destroy node, then don't do anything
	if n.DestroyMode != DestroyNone {
		return nil
	}

	result := &graphNodeResourceDestroy{
		GraphNodeConfigResource: *n,
		Original:                n,
	}
	result.DestroyMode = mode
	return result
}

// GraphNodeNoopPrunable
func (n *GraphNodeConfigResource) Noop(opts *NoopOpts) bool {
	// We don't have any noop optimizations for destroy nodes yet
	if n.DestroyMode != DestroyNone {
		return false
	}

	// If there is no diff, then we aren't a noop since something needs to
	// be done (such as a plan). We only check if we're a noop in a diff.
	if opts.Diff == nil || opts.Diff.Empty() {
		return false
	}

	// If we have no module diff, we're certainly a noop. This is because
	// it means there is a diff, and that the module we're in just isn't
	// in it, meaning we're not doing anything.
	if opts.ModDiff == nil || opts.ModDiff.Empty() {
		return true
	}

	// Grab the ID which is the prefix (in the case count > 0 at some point)
	prefix := n.Resource.Id()

	// Go through the diff and if there are any with our name on it, keep us
	found := false
	for k, _ := range opts.ModDiff.Resources {
		if strings.HasPrefix(k, prefix) {
			found = true
			break
		}
	}

	return !found
}

// Same as GraphNodeConfigResource, but for flattening
type GraphNodeConfigResourceFlat struct {
	*GraphNodeConfigResource

	PathValue []string
}

func (n *GraphNodeConfigResourceFlat) Name() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue), n.GraphNodeConfigResource.Name())
}

func (n *GraphNodeConfigResourceFlat) Path() []string {
	return n.PathValue
}

func (n *GraphNodeConfigResourceFlat) DependableName() []string {
	return modulePrefixList(
		n.GraphNodeConfigResource.DependableName(),
		modulePrefixStr(n.PathValue))
}

func (n *GraphNodeConfigResourceFlat) DependentOn() []string {
	prefix := modulePrefixStr(n.PathValue)
	return modulePrefixList(
		n.GraphNodeConfigResource.DependentOn(),
		prefix)
}

func (n *GraphNodeConfigResourceFlat) ProvidedBy() []string {
	prefix := modulePrefixStr(n.PathValue)
	return modulePrefixList(
		n.GraphNodeConfigResource.ProvidedBy(),
		prefix)
}

func (n *GraphNodeConfigResourceFlat) ProvisionedBy() []string {
	prefix := modulePrefixStr(n.PathValue)
	return modulePrefixList(
		n.GraphNodeConfigResource.ProvisionedBy(),
		prefix)
}

// GraphNodeDestroyable impl.
func (n *GraphNodeConfigResourceFlat) DestroyNode(mode GraphNodeDestroyMode) GraphNodeDestroy {
	// Get our parent destroy node. If we don't have any, just return
	raw := n.GraphNodeConfigResource.DestroyNode(mode)
	if raw == nil {
		return nil
	}

	node, ok := raw.(*graphNodeResourceDestroy)
	if !ok {
		panic(fmt.Sprintf("unknown destroy node: %s %T", dag.VertexName(raw), raw))
	}

	// Otherwise, wrap it so that it gets the proper module treatment.
	return &graphNodeResourceDestroyFlat{
		graphNodeResourceDestroy: node,
		PathValue:                n.PathValue,
		FlatCreateNode:           n,
	}
}

type graphNodeResourceDestroyFlat struct {
	*graphNodeResourceDestroy

	PathValue []string

	// Needs to be able to properly yield back a flattened create node to prevent
	FlatCreateNode *GraphNodeConfigResourceFlat
}

func (n *graphNodeResourceDestroyFlat) Name() string {
	return fmt.Sprintf(
		"%s.%s", modulePrefixStr(n.PathValue), n.graphNodeResourceDestroy.Name())
}

func (n *graphNodeResourceDestroyFlat) Path() []string {
	return n.PathValue
}

func (n *graphNodeResourceDestroyFlat) CreateNode() dag.Vertex {
	return n.FlatCreateNode
}

func (n *graphNodeResourceDestroyFlat) ProvidedBy() []string {
	prefix := modulePrefixStr(n.PathValue)
	return modulePrefixList(
		n.GraphNodeConfigResource.ProvidedBy(),
		prefix)
}

// graphNodeResourceDestroy represents the logical destruction of a
// resource. This node doesn't mean it will be destroyed for sure, but
// instead that if a destroy were to happen, it must happen at this point.
type graphNodeResourceDestroy struct {
	GraphNodeConfigResource
	Original *GraphNodeConfigResource
}

func (n *graphNodeResourceDestroy) CreateBeforeDestroy() bool {
	// CBD is enabled if the resource enables it in addition to us
	// being responsible for destroying the primary state. The primary
	// state destroy node is the only destroy node that needs to be
	// "shuffled" according to the CBD rules, since tainted resources
	// don't have the same inverse dependencies.
	return n.Original.Resource.Lifecycle.CreateBeforeDestroy &&
		n.DestroyMode == DestroyPrimary
}

func (n *graphNodeResourceDestroy) CreateNode() dag.Vertex {
	return n.Original
}

func (n *graphNodeResourceDestroy) DestroyInclude(d *ModuleDiff, s *ModuleState) bool {
	switch n.DestroyMode {
	case DestroyPrimary:
		return n.destroyIncludePrimary(d, s)
	case DestroyTainted:
		return n.destroyIncludeTainted(d, s)
	default:
		return true
	}
}

func (n *graphNodeResourceDestroy) destroyIncludeTainted(
	d *ModuleDiff, s *ModuleState) bool {
	// If there is no state, there can't by any tainted.
	if s == nil {
		return false
	}

	// Grab the ID which is the prefix (in the case count > 0 at some point)
	prefix := n.Original.Resource.Id()

	// Go through the resources and find any with our prefix. If there
	// are any tainted, we need to keep it.
	for k, v := range s.Resources {
		if !strings.HasPrefix(k, prefix) {
			continue
		}

		if len(v.Tainted) > 0 {
			return true
		}
	}

	// We didn't find any tainted nodes, return
	return false
}

func (n *graphNodeResourceDestroy) destroyIncludePrimary(
	d *ModuleDiff, s *ModuleState) bool {
	// Get the count, and specifically the raw value of the count
	// (with interpolations and all). If the count is NOT a static "1",
	// then we keep the destroy node no matter what.
	//
	// The reasoning for this is complicated and not intuitively obvious,
	// but I attempt to explain it below.
	//
	// The destroy transform works by generating the worst case graph,
	// with worst case being the case that every resource already exists
	// and needs to be destroy/created (force-new). There is a single important
	// edge case where this actually results in a real-life cycle: if a
	// create-before-destroy (CBD) resource depends on a non-CBD resource.
	// Imagine a EC2 instance "foo" with CBD depending on a security
	// group "bar" without CBD, and conceptualize the worst case destroy
	// order:
	//
	//   1.) SG must be destroyed (non-CBD)
	//   2.) SG must be created/updated
	//   3.) EC2 instance must be created (CBD, requires the SG be made)
	//   4.) EC2 instance must be destroyed (requires SG be destroyed)
	//
	// Except, #1 depends on #4, since the SG can't be destroyed while
	// an EC2 instance is using it (AWS API requirements). As you can see,
	// this is a real life cycle that can't be automatically reconciled
	// except under two conditions:
	//
	//   1.) SG is also CBD. This doesn't work 100% of the time though
	//       since the non-CBD resource might not support CBD. To make matters
	//       worse, the entire transitive closure of dependencies must be
	//       CBD (if the SG depends on a VPC, you have the same problem).
	//   2.) EC2 must not CBD. This can't happen automatically because CBD
	//       is used as a way to ensure zero (or minimal) downtime Terraform
	//       applies, and it isn't acceptable for TF to ignore this request,
	//       since it can result in unexpected downtime.
	//
	// Therefore, we compromise with this edge case here: if there is
	// a static count of "1", we prune the diff to remove cycles during a
	// graph optimization path if we don't see the resource in the diff.
	// If the count is set to ANYTHING other than a static "1" (variable,
	// computed attribute, static number greater than 1), then we keep the
	// destroy, since it is required for dynamic graph expansion to find
	// orphan/tainted count objects.
	//
	// This isn't ideal logic, but its strictly better without introducing
	// new impossibilities. It breaks the cycle in practical cases, and the
	// cycle comes back in no cases we've found to be practical, but just
	// as the cycle would already exist without this anyways.
	count := n.Original.Resource.RawCount
	if raw := count.Raw[count.Key]; raw != "1" {
		return true
	}

	// Okay, we're dealing with a static count. There are a few ways
	// to include this resource.
	prefix := n.Original.Resource.Id()

	// If we're present in the diff proper, then keep it. We're looking
	// only for resources in the diff that match our resource or a count-index
	// of our resource that are marked for destroy.
	if d != nil {
		for k, d := range d.Resources {
			match := k == prefix || strings.HasPrefix(k, prefix+".")
			if match && d.Destroy {
				return true
			}
		}
	}

	// If we're in the state as a primary in any form, then keep it.
	// This does a prefix check so it will also catch orphans on count
	// decreases to "1".
	if s != nil {
		for k, v := range s.Resources {
			// Ignore exact matches
			if k == prefix {
				continue
			}

			// Ignore anything that doesn't have a "." afterwards so that
			// we only get our own resource and any counts on it.
			if !strings.HasPrefix(k, prefix+".") {
				continue
			}

			// Ignore exact matches and the 0'th index. We only care
			// about if there is a decrease in count.
			if k == prefix+".0" {
				continue
			}

			if v.Primary != nil {
				return true
			}
		}

		// If we're in the state as _both_ "foo" and "foo.0", then
		// keep it, since we treat the latter as an orphan.
		_, okOne := s.Resources[prefix]
		_, okTwo := s.Resources[prefix+".0"]
		if okOne && okTwo {
			return true
		}
	}

	return false
}
