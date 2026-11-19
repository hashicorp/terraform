// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"os"
	"strings"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type QueryCommand struct {
	Meta
}

const (
	queryPolicyEntitlementHostEnv = "TF_POLICY_ENTITLEMENT_HOST"
	queryPolicyEntitlementOrgEnv  = "TF_POLICY_ENTITLEMENT_ORG"
)

func (c *QueryCommand) Help() string {
	helpText := `
Usage: terraform [global options] query [options]

  Queries the remote infrastructure for resources.

  Terraform will search for .tfquery.hcl files within the current configuration.
  Terraform will then use the configured providers to query the remote
  infrastructure for resources that match the defined list blocks. The results
  will be printed to the terminal and optionally can be used to generate
  configuration.

Query Customization Options:

  The following options customize how Terraform will run the query.

  -policies=path        Evaluate policies from a policy set directory against
                        resources discovered by the query. Use this option more
                        than once to include multiple policy set paths.
                        The equivalent --policies=path spelling is also supported.

  -var 'foo=bar'        Set a value for one of the input variables in the query
                        file of the configuration. Use this option more than
                        once to set more than one variable.

  -var-file=filename    Load variable values from the given file, in addition
                        to the default files terraform.tfvars and *.auto.tfvars.
                        Use this option more than once to include more than one
                        variables file.

Other Options:

  -generate-config-out=path  Instructs Terraform to generate import and resource
                             blocks for any found results. The configuration is
                             written to a new file at PATH, which must not
                             already exist. When this option is used with the
                             json option, the generated configuration will be
                             part of the JSON output instead of written to a
                             file.

  -json                      If specified, machine readable output will be
                             printed in JSON format

  -no-color                  If specified, output won't contain any color.

`
	return strings.TrimSpace(helpText)
}

func (c *QueryCommand) Synopsis() string {
	return "Search and list remote infrastructure with Terraform"
}

func (c *QueryCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Propagate -no-color for legacy use of Ui.  The remote backend and
	// cloud package use this; it should be removed when/if they are
	// migrated to views.
	c.Meta.color = !common.NoColor
	c.Meta.Color = c.Meta.color
	c.Meta.includeQueryFiles = true

	// Parse and validate flags
	args, diags := arguments.ParseQuery(rawArgs)
	diags = diags.Append(c.Validate(args))

	// Instantiate the view, even if there are flag errors, so that we render
	// diagnostics according to the desired view
	view := views.NewQuery(args.ViewType, c.View)

	// Log version output
	view.Version()

	loader, err := c.initConfigLoader()
	if err != nil {
		diags = diags.Append(fmt.Errorf("Failed to initialize config loader: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	// We need to collect variables early in this command to ensure that the
	// values are available when preparing the backend. A PSS backend will need
	// the values to resolve the provider requirements.
	var varDiags tfdiags.Diagnostics
	c.Meta.VariableValues, varDiags = args.Vars.CollectValues(func(filename string, src []byte) {
		loader.Parser().ForceFileSource(filename, src)
	})
	diags = diags.Append(varDiags)

	if diags.HasErrors() {
		view.Diagnostics(diags)
		view.HelpPrompt()
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	// We currently don't support the paralleism flag in the query command,
	// so we set it to the default value here. This avoids backend errors
	// that check for deviant values.
	c.Meta.parallelism = DefaultParallelism

	// Prepare the backend with the backend-specific arguments
	be, beDiags := c.PrepareBackend(args.State, args.ViewType)
	b, isRemoteBackend := be.(BackendWithRemoteTerraformVersion)
	if isRemoteBackend && !b.IsLocalOperations() {
		diags = diags.Append(c.providerDevOverrideRuntimeWarningsRemoteExecution())
	} else {
		diags = diags.Append(c.providerDevOverrideRuntimeWarnings())
	}
	diags = diags.Append(beDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Build the operation request
	opReq, opDiags := c.OperationRequest(be, view, args.ViewType, args.GenerateConfigPath, args.PolicyPaths)
	diags = diags.Append(opDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	stopPolicyClient, policySetupOK := c.configureQueryPolicyClient(be, opReq)
	defer stopPolicyClient()
	if !policySetupOK {
		view.Diagnostics(diags)
		return 1
	}

	// Assign config loader and variables to the operation request
	opReq.ConfigLoader = loader
	opReq.Variables = c.Meta.VariableValues

	// Before we delegate to the backend, we'll print any warning diagnostics
	// we've accumulated here, since the backend will start fresh with its own
	// diagnostics.
	view.Diagnostics(diags)
	if diags.HasErrors() {
		return 1
	}
	diags = nil

	// Perform the operation
	op, err := c.RunOperation(be, opReq)
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}

	if op.Result != backendrun.OperationSuccess {
		return op.Result.ExitStatus()
	}

	return op.Result.ExitStatus()
}

func (c *QueryCommand) configureQueryPolicyClient(be backendrun.OperationsBackend, opReq *backendrun.Operation) (stop func(), ok bool) {
	remote, isRemoteBackend := be.(BackendWithRemoteTerraformVersion)
	// Remote execution forwards policy paths to the server; only local
	// execution needs an in-process policy client.
	if len(opReq.PolicyPaths) == 0 || (isRemoteBackend && !remote.IsLocalOperations()) {
		return func() {}, true
	}

	entitlement, fromEnv, err := c.queryPolicyEntitlement(be)
	if err != nil {
		if opReq.View != nil {
			opReq.View.PolicyDiagnostics(policy.Diagnostics{policy.NewErrorDiagnostic(
				"Failed to configure query policy entitlement", err.Error(), policy.SetupErrorResult,
			)})
		}
		return func() {}, false
	}
	client, policyDiags, stopClient := c.PolicyClient(c.CommandContext(), opReq.PolicyPaths, entitlement)
	if opReq.View != nil && policyDiags != nil {
		opReq.View.PolicyDiagnostics(policyDiags)
	}
	opReq.PolicyClient = client
	// An explicit entitlement request must not silently run without policy.
	return stopClient, !fromEnv || (client != nil && !policyDiags.HasErrors())
}

func (c *QueryCommand) queryPolicyEntitlement(be backendrun.OperationsBackend) (entitlement *policy.Entitlement, fromEnv bool, err error) {
	if entitlement := backendPolicyEntitlement(be, policy.PlanEvaluationStage); entitlement != nil {
		return entitlement, false, nil
	}
	// Local execution can wrap a remote state backend. Only standalone local
	// storage is eligible for entitlement independent of the backend.
	local, ok := be.(*backendLocal.Local)
	if !ok || local.Backend != nil {
		return nil, false, nil
	}

	host, hasHost := os.LookupEnv(queryPolicyEntitlementHostEnv)
	org, hasOrg := os.LookupEnv(queryPolicyEntitlementOrgEnv)
	if !hasHost && !hasOrg {
		return nil, false, nil
	}
	if strings.TrimSpace(host) == "" || strings.TrimSpace(org) == "" {
		return nil, false, fmt.Errorf("%s and %s must both be set to non-empty values", queryPolicyEntitlementHostEnv, queryPolicyEntitlementOrgEnv)
	}
	hostname, err := svchost.ForComparison(host)
	if err != nil {
		return nil, false, fmt.Errorf("%s must be a valid hostname", queryPolicyEntitlementHostEnv)
	}
	if c.Services == nil {
		return nil, false, fmt.Errorf("Terraform CLI credentials are unavailable")
	}
	creds, err := c.Services.CredentialsForHost(hostname)
	if err != nil || creds == nil || creds.Token() == "" {
		return nil, false, fmt.Errorf("could not resolve Terraform CLI credentials for the policy entitlement host %q", hostname.ForDisplay())
	}
	return &policy.Entitlement{
		Host:            string(hostname),
		Org:             org,
		Token:           creds.Token(),
		EvaluationStage: policy.PlanEvaluationStage,
	}, true, nil
}

func (c *QueryCommand) Validate(args *arguments.Query) (diags tfdiags.Diagnostics) {
	// The query command's -policies option takes policy set directories, so we
	// reject anything that isn't a directory here rather than letting it reach
	// the policy client.
	return diags.Append(validatePolicySetDirs(args.PolicyPaths))
}

func (c *QueryCommand) PrepareBackend(args *arguments.State, viewType arguments.ViewType) (backendrun.OperationsBackend, tfdiags.Diagnostics) {
	// Load the backend
	be, diags := c.backend(".", viewType)
	return be, diags
}

func (c *QueryCommand) OperationRequest(
	be backendrun.OperationsBackend,
	view views.Query,
	viewType arguments.ViewType,
	generateConfigOut string,
	policyPaths []string,
) (*backendrun.Operation, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Build the operation
	opReq := c.Operation(be, viewType)
	opReq.Hooks = view.Hooks()
	opReq.ConfigDir = "."
	opReq.Type = backendrun.OperationTypePlan
	opReq.GenerateConfigOut = generateConfigOut
	opReq.View = view.Operation()
	opReq.Query = true
	opReq.PolicyPaths = policyPaths

	return opReq, diags
}
