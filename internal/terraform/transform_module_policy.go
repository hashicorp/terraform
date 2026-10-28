// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// ModulePolicyTransformer adds a nodeModulePolicy for every module call in
// the configuration, and makes everything within each module depend on it so
// that the module policy is evaluated before any of the module's objects.
type ModulePolicyTransformer struct {
	Config       *configs.Config
	PolicyClient policy.Client
}

var _ GraphTransformer = (*ModulePolicyTransformer)(nil)

func (t *ModulePolicyTransformer) Transform(g *Graph) error {
	if t.PolicyClient == nil || t.Config == nil {
		return nil
	}

	moduleNodes := make(map[string]*nodeModulePolicy, 0)
	for config := range t.Config.AllModules() {
		// root modules need no policy evaluation
		if config.Path.IsRoot() {
			continue
		}
		n := &nodeModulePolicy{Config: config}
		log.Printf("[TRACE] ModulePolicyTransformer: adding %s", n.Name())
		g.Add(n)
		moduleNodes[config.Path.String()] = n
	}

	if len(moduleNodes) == 0 {
		return nil
	}

	// make every object in the module object wait for the for the module policy to
	// be evaluated.
	for node := range dag.SelectSeq[GraphNodeModulePath](g.VerticesSeq()) {
		path := node.ModulePath()

		moduleNode, ok := moduleNodes[path.String()]
		if !ok {
			continue
		}
		g.Connect(node, moduleNode)
	}

	return nil
}

// nodeModulePolicy evaluates the policy for a module call.
type nodeModulePolicy struct {
	// Config is the configuration of the called module.
	Config *configs.Config
}

var _ GraphNodeExecutable = (*nodeModulePolicy)(nil)

func (n *nodeModulePolicy) Name() string {
	return n.Config.Path.String() + " (policy)"
}

func (n *nodeModulePolicy) Execute(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	config := n.Config
	addr := config.Path
	client := ctx.PolicyClient()
	if client == nil {
		log.Printf("[DEBUG] No policy client configured, skipping policy evaluation for %s", addr)
		return nil
	}

	_, call := addr.Call()
	moduleCall := config.Parent.Module.ModuleCalls[call.Name]
	source := config.SourceAddr.String()

	// We will evaluate the module policy with just the metadata and not the module inputs.
	// Terraform modules act as namespace scopes and do not directly participate
	// in the reference flow in the graph.
	// This allows references in their inputs to be deferred until
	// they are used by a concrete object in the module.
	// Since policy evaluation is supposed to be done before evaluating the module's
	// objects, evaluating module inputs here would be incorrect as
	// it could lead to cyclic dependencies.
	// For example: Given the below module call,
	//
	// module "child" {
	// 	 source = "./child"
	// 	 input  = "child-value"
	// 	 input2 = module.child.output
	// }
	//
	// resource "test_instance" "test" {
	// 	 value = var.input
	// }
	// resource "test_instance" "test2" {
	// 	 value = var.input2
	// }
	// output "output" {
	// 	 value = resource.test_instance.test.value
	// }
	//
	// This is a valid terraform module call, but the input2 value depends on the module's output,
	// which is not available until after the module is expanded, and potentially resources have been created.
	// In the above example, the reference to `module.child.output` is deferred until the resource
	// `test_instance.test2` is evaluated, and any attempt to evaluate it here would result in a cyclic dependency error.
	result := client.EvaluateModule(ctx.StopCtx(), policy.EvaluationRequest[*proto.PolicyEvaluateModuleRequest_ModuleMetadata]{
		Attrs:  policy.CtyToPolicyValue(cty.DynamicVal),
		Target: source,
		Meta: &proto.PolicyEvaluateModuleRequest_ModuleMetadata{
			Address: addr.String(),
			Version: func() string {
				// local modules do not have versions, so we just return
				// an empty string in that case.
				if config.Version == nil {
					return ""
				}
				return config.Version.String()
			}(),
		},
	})

	// add local range to diagnostics if the module call has a config body.
	if moduleCall.Config != nil {
		result = result.WithLocalRange(moduleCall.DeclRange.Ptr())
	}

	var diags tfdiags.Diagnostics
	if !result.Empty() {
		hookErr := ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PolicyResult(addr.String(), result)
		})
		diags = diags.Append(hookErr)
	}

	return diags
}
