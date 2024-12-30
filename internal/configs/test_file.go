// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/getmodules/moduleaddrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestCommand represents the Terraform a given run block will execute, plan
// or apply. Defaults to apply.
type TestCommand rune

// TestMode represents the plan mode that Terraform will use for a given run
// block, normal or refresh-only. Defaults to normal.
type TestMode rune

const (
	// ApplyTestCommand causes the run block to execute a Terraform apply
	// operation.
	ApplyTestCommand TestCommand = 0

	// PlanTestCommand causes the run block to execute a Terraform plan
	// operation.
	PlanTestCommand TestCommand = 'P'

	// NormalTestMode causes the run block to execute in plans.NormalMode.
	NormalTestMode TestMode = 0

	// RefreshOnlyTestMode causes the run block to execute in
	// plans.RefreshOnlyMode.
	RefreshOnlyTestMode TestMode = 'R'
)

// TestFile represents a single test file within a `terraform test` execution.
//
// A test file is made up of a sequential list of run blocks, each designating
// a command to execute and a series of validations to check after the command.
type TestFile struct {
	// Variables defines a set of global variable definitions that should be set
	// for every run block within the test file.
	Variables map[string]hcl.Expression

	// Providers defines a set of providers that are available to run blocks
	// within this test file.
	//
	// Some or all of these providers may be mocked providers.
	//
	// If empty, tests should use the default providers for the module under
	// test.
	Providers map[string]*Provider

	// Overrides contains any specific overrides that should be applied for this
	// test outside any mock providers.
	Overrides addrs.Map[addrs.Targetable, *Override]

	// Runs defines the sequential list of run blocks that should be executed in
	// order.
	Runs []*TestRun

	VariablesDeclRange hcl.Range
}

// TestRun represents a single run block within a test file.
//
// Each run block represents a single Terraform command to be executed and a set
// of validations to run after the command.
type TestRun struct {
	Name string

	// Command is the Terraform command to execute.
	//
	// One of ['apply', 'plan'].
	Command TestCommand

	// Options contains the embedded plan options that will affect the given
	// Command. These should map to the options documented here:
	//   - https://developer.hashicorp.com/terraform/cli/commands/plan#planning-options
	//
	// Note, that the Variables are a top level concept and not embedded within
	// the options despite being listed as plan options in the documentation.
	Options *TestRunOptions

	// Variables defines a set of variable definitions for this command.
	//
	// Any variables specified locally that clash with the global variables will
	// take precedence over the global definition.
	Variables map[string]hcl.Expression

	// Overrides contains any specific overrides that should be applied for this
	// run block only outside any mock providers or overrides from the file.
	Overrides addrs.Map[addrs.Targetable, *Override]

	// Providers specifies the set of providers that should be loaded into the
	// module for this run block.
	//
	// Providers specified here must be configured in one of the provider blocks
	// for this file. If empty, the run block will load the default providers
	// for the module under test.
	Providers []PassedProviderConfig

	// CheckRules defines the list of assertions/validations that should be
	// checked by this run block.
	CheckRules []*CheckRule

	// Module defines an address of another module that should be loaded and
	// executed as part of this run block instead of the module under test.
	//
	// In the initial version of the testing framework we will only support
	// loading alternate modules from local directories or the registry.
	Module *TestRunModuleCall

	// ConfigUnderTest describes the configuration this run block should execute
	// against.
	//
	// In typical cases, this will be null and the config under test is the
	// configuration within the directory the terraform test command is
	// executing within. However, when Module is set the config under test is
	// whichever config is defined by Module. This field is then set during the
	// configuration load process and should be used when the test is executed.
	ConfigUnderTest *Config

	// ExpectFailures should be a list of checkable objects that are expected
	// to report a failure from their custom conditions as part of this test
	// run.
	ExpectFailures []hcl.Traversal

	NameDeclRange      hcl.Range
	VariablesDeclRange hcl.Range
	DeclRange          hcl.Range
}

// Validate does a very simple and cursory check across the test file to look
// for simple issues we can highlight early on.
//
// This function only returns warnings in the diagnostics. Callers of this
// function usually do not validate the returned diagnostics as a result. If
// you change this function, make sure to update the callers as well.
func (file *TestFile) Validate(config *Config) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for _, provider := range file.Providers {
		if !provider.Mock {
			continue
		}

		for _, elem := range provider.MockData.Overrides.Elems {
			if elem.Value.Source != MockProviderOverrideSource {
				// Only check overrides that are defined directly within the
				// mock provider block of this file. This is a safety check
				// against any override blocks loaded from a dedicated data
				// file, for these we won't raise warnings if they target
				// resources that don't exist.
				continue
			}

			if !file.canTarget(config, elem.Key) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagWarning,
					Summary:  "Invalid override target",
					Detail:   fmt.Sprintf("The override target %s does not exist within the configuration under test. This could indicate a typo in the target address or an unnecessary override.", elem.Key),
					Subject:  elem.Value.TargetRange.Ptr(),
				})
			}
		}
	}

	for _, elem := range file.Overrides.Elems {
		if !file.canTarget(config, elem.Key) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Invalid override target",
				Detail:   fmt.Sprintf("The override target %s does not exist within the configuration under test. This could indicate a typo in the target address or an unnecessary override.", elem.Key),
				Subject:  elem.Value.TargetRange.Ptr(),
			})
		}
	}

	return diags
}

// canTarget is a helper function, that just checks all the available
// configurations to make sure at least one contains the specified target.
func (file *TestFile) canTarget(config *Config, target addrs.Targetable) bool {
	// If the target is in the main configuration, then easy.
	if config.TargetExists(target) {
		return true
	}

	// But, we could be targeting something in configuration loaded by one of
	// the run blocks.
	for _, run := range file.Runs {
		if run.Module != nil {
			if run.ConfigUnderTest.TargetExists(target) {
				return true
			}
		}
	}

	return false
}

// Validate does a very simple and cursory check across the run block to look
// for simple issues we can highlight early on.
func (run *TestRun) Validate(config *Config) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// First, we want to make sure all the ExpectFailure references are the
	// correct kind of reference.
	for _, traversal := range run.ExpectFailures {

		reference, refDiags := addrs.ParseRefFromTestingScope(traversal)
		diags = diags.Append(refDiags)
		if refDiags.HasErrors() {
			continue
		}

		switch reference.Subject.(type) {
		// You can only reference outputs, inputs, checks, and resources.
		case addrs.OutputValue, addrs.InputVariable, addrs.Check, addrs.ResourceInstance, addrs.Resource:
			// Do nothing, these are okay!
		default:
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid `expect_failures` reference",
				Detail:   fmt.Sprintf("You cannot expect failures from %s. You can only expect failures from checkable objects such as input variables, output values, check blocks, managed resources and data sources.", reference.Subject.String()),
				Subject:  reference.SourceRange.ToHCL().Ptr(),
			})
		}

	}

	// All the overrides defined within a run block should target an existing
	// configuration block.
	for _, elem := range run.Overrides.Elems {
		if !config.TargetExists(elem.Key) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Invalid override target",
				Detail:   fmt.Sprintf("The override target %s does not exist within the configuration under test. This could indicate a typo in the target address or an unnecessary override.", elem.Key),
				Subject:  elem.Value.TargetRange.Ptr(),
			})
		}
	}

	return diags
}

// TestRunModuleCall specifies which module should be executed by a given run
// block.
type TestRunModuleCall struct {
	// Source is the source of the module to test.
	Source addrs.ModuleSource

	// Version is the version of the module to load from the registry.
	Version VersionConstraint

	DeclRange       hcl.Range
	SourceDeclRange hcl.Range
}

// TestRunOptions contains the plan options for a given run block.
type TestRunOptions struct {
	// Mode is the planning mode to run in. One of ['normal', 'refresh-only'].
	Mode TestMode

	// Refresh is analogous to the -refresh=false Terraform plan option.
	Refresh bool

	// Replace is analogous to the -replace=ADDRESS Terraform plan option.
	Replace []hcl.Traversal

	// Target is analogous to the -target=ADDRESS Terraform plan option.
	Target []hcl.Traversal

	DeclRange hcl.Range
}

func loadTestFile(body hcl.Body) (*TestFile, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := body.Content(testFileSchema)
	diags = append(diags, contentDiags...)

	tf := TestFile{
		Providers: make(map[string]*Provider),
		Overrides: addrs.MakeMap[addrs.Targetable, *Override](),
	}

	runBlockNames := make(map[string]hcl.Range)

	for _, block := range content.Blocks {
		switch block.Type {
		case "run":
			run, runDiags := decodeTestRunBlock(block)
			diags = append(diags, runDiags...)
			if !runDiags.HasErrors() {
				tf.Runs = append(tf.Runs, run)
			}

			if rng, exists := runBlockNames[run.Name]; exists {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Duplicate \"run\" block names",
					Detail:   fmt.Sprintf("This test file already has a run named %s block defined at %s.", run.Name, rng),
					Subject:  block.DefRange.Ptr(),
				})
				continue
			}
			runBlockNames[run.Name] = run.DeclRange

		case "variables":
			if tf.Variables != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Multiple \"variables\" blocks",
					Detail:   fmt.Sprintf("This test file already has a variables block defined at %s.", tf.VariablesDeclRange),
					Subject:  block.DefRange.Ptr(),
				})
				continue
			}

			tf.Variables = make(map[string]hcl.Expression)
			tf.VariablesDeclRange = block.DefRange

			vars, varsDiags := block.Body.JustAttributes()
			diags = append(diags, varsDiags...)
			for _, v := range vars {
				tf.Variables[v.Name] = v.Expr
			}
		case "provider":
			provider, providerDiags := decodeProviderBlock(block, true)
			diags = append(diags, providerDiags...)
			if provider != nil {
				key := provider.moduleUniqueKey()
				if previous, exists := tf.Providers[key]; exists {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate provider block",
						Detail:   fmt.Sprintf("A provider for %s is already defined at %s.", key, previous.NameRange),
						Subject:  provider.DeclRange.Ptr(),
					})
					continue
				}
				tf.Providers[key] = provider
			}
		case "mock_provider":
			provider, providerDiags := decodeMockProviderBlock(block)
			diags = append(diags, providerDiags...)
			if provider != nil {
				key := provider.moduleUniqueKey()
				if previous, exists := tf.Providers[key]; exists {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate provider block",
						Detail:   fmt.Sprintf("A provider for %s is already defined at %s.", key, previous.NameRange),
						Subject:  provider.DeclRange.Ptr(),
					})
					continue
				}
				tf.Providers[key] = provider
			}
		case "override_resource":
			override, overrideDiags := decodeOverrideResourceBlock(block, TestFileOverrideSource)
			diags = append(diags, overrideDiags...)

			if override != nil && override.Target != nil {
				subject := override.Target.Subject
				if previous, ok := tf.Overrides.GetOk(subject); ok {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate override_resource block",
						Detail:   fmt.Sprintf("An override_resource block targeting %s has already been defined at %s.", subject, previous.Range),
						Subject:  override.Range.Ptr(),
					})
					continue
				}
				tf.Overrides.Put(subject, override)
			}
		case "override_data":
			override, overrideDiags := decodeOverrideDataBlock(block, TestFileOverrideSource)
			diags = append(diags, overrideDiags...)

			if override != nil && override.Target != nil {
				subject := override.Target.Subject
				if previous, ok := tf.Overrides.GetOk(subject); ok {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate override_data block",
						Detail:   fmt.Sprintf("An override_data block targeting %s has already been defined at %s.", subject, previous.Range),
						Subject:  override.Range.Ptr(),
					})
					continue
				}
				tf.Overrides.Put(subject, override)
			}
		case "override_module":
			override, overrideDiags := decodeOverrideModuleBlock(block, TestFileOverrideSource)
			diags = append(diags, overrideDiags...)

			if override != nil && override.Target != nil {
				subject := override.Target.Subject
				if previous, ok := tf.Overrides.GetOk(subject); ok {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate override_module block",
						Detail:   fmt.Sprintf("An override_module block targeting %s has already been defined at %s.", subject, previous.Range),
						Subject:  override.Range.Ptr(),
					})
					continue
				}
				tf.Overrides.Put(subject, override)
			}
		}
	}

	return &tf, diags
}

func decodeTestRunBlock(block *hcl.Block) (*TestRun, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := block.Body.Content(testRunBlockSchema)
	diags = append(diags, contentDiags...)

	r := TestRun{
		Overrides: addrs.MakeMap[addrs.Targetable, *Override](),

		Name:          block.Labels[0],
		NameDeclRange: block.LabelRanges[0],
		DeclRange:     block.DefRange,
	}

	if !hclsyntax.ValidIdentifier(r.Name) {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid run block name",
			Detail:   badIdentifierDetail,
			Subject:  r.NameDeclRange.Ptr(),
		})
	}

	for _, block := range content.Blocks {
		switch block.Type {
		case "assert":
			cr, crDiags := decodeCheckRuleBlock(block, false)
			diags = append(diags, crDiags...)
			if !crDiags.HasErrors() {
				r.CheckRules = append(r.CheckRules, cr)
			}
		case "plan_options":
			if r.Options != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Multiple \"plan_options\" blocks",
					Detail:   fmt.Sprintf("This run block already has a plan_options block defined at %s.", r.Options.DeclRange),
					Subject:  block.DefRange.Ptr(),
				})
				continue
			}

			opts, optsDiags := decodeTestRunOptionsBlock(block)
			diags = append(diags, optsDiags...)
			if !optsDiags.HasErrors() {
				r.Options = opts
			}
		case "variables":
			if r.Variables != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Multiple \"variables\" blocks",
					Detail:   fmt.Sprintf("This run block already has a variables block defined at %s.", r.VariablesDeclRange),
					Subject:  block.DefRange.Ptr(),
				})
				continue
			}

			r.Variables = make(map[string]hcl.Expression)
			r.VariablesDeclRange = block.DefRange

			vars, varsDiags := block.Body.JustAttributes()
			diags = append(diags, varsDiags...)
			for _, v := range vars {
				r.Variables[v.Name] = v.Expr
			}
		case "module":
			if r.Module != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Multiple \"module\" blocks",
					Detail:   fmt.Sprintf("This run block already has a module block defined at %s.", r.Module.DeclRange),
					Subject:  block.DefRange.Ptr(),
				})
			}

			module, moduleDiags := decodeTestRunModuleBlock(block)
			diags = append(diags, moduleDiags...)
			if !moduleDiags.HasErrors() {
				r.Module = module
			}
		case "override_resource":
			override, overrideDiags := decodeOverrideResourceBlock(block, RunBlockOverrideSource)
			diags = append(diags, overrideDiags...)

			if override != nil && override.Target != nil {
				subject := override.Target.Subject
				if previous, ok := r.Overrides.GetOk(subject); ok {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate override_resource block",
						Detail:   fmt.Sprintf("An override_resource block targeting %s has already been defined at %s.", subject, previous.Range),
						Subject:  override.Range.Ptr(),
					})
					continue
				}
				r.Overrides.Put(subject, override)
			}
		case "override_data":
			override, overrideDiags := decodeOverrideDataBlock(block, RunBlockOverrideSource)
			diags = append(diags, overrideDiags...)

			if override != nil && override.Target != nil {
				subject := override.Target.Subject
				if previous, ok := r.Overrides.GetOk(subject); ok {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate override_data block",
						Detail:   fmt.Sprintf("An override_data block targeting %s has already been defined at %s.", subject, previous.Range),
						Subject:  override.Range.Ptr(),
					})
					continue
				}
				r.Overrides.Put(subject, override)
			}
		case "override_module":
			override, overrideDiags := decodeOverrideModuleBlock(block, RunBlockOverrideSource)
			diags = append(diags, overrideDiags...)

			if override != nil && override.Target != nil {
				subject := override.Target.Subject
				if previous, ok := r.Overrides.GetOk(subject); ok {
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate override_module block",
						Detail:   fmt.Sprintf("An override_module block targeting %s has already been defined at %s.", subject, previous.Range),
						Subject:  override.Range.Ptr(),
					})
					continue
				}
				r.Overrides.Put(subject, override)
			}
		}
	}

	if r.Variables == nil {
		// There is no distinction between a nil map of variables or an empty
		// map, but we can avoid any potential nil pointer exceptions by just
		// creating an empty map.
		r.Variables = make(map[string]hcl.Expression)
	}

	if r.Options == nil {
		// Create an options with default values if the user didn't specify
		// anything.
		r.Options = &TestRunOptions{
			Mode:    NormalTestMode,
			Refresh: true,
		}
	}

	if attr, exists := content.Attributes["command"]; exists {
		switch hcl.ExprAsKeyword(attr.Expr) {
		case "apply":
			r.Command = ApplyTestCommand
		case "plan":
			r.Command = PlanTestCommand
		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid \"command\" keyword",
				Detail:   "The \"command\" argument requires one of the following keywords without quotes: apply or plan.",
				Subject:  attr.Expr.Range().Ptr(),
			})
		}
	} else {
		r.Command = ApplyTestCommand // Default to apply
	}

	if attr, exists := content.Attributes["providers"]; exists {
		providers, providerDiags := decodePassedProviderConfigs(attr)
		diags = append(diags, providerDiags...)
		r.Providers = append(r.Providers, providers...)
	}

	if attr, exists := content.Attributes["expect_failures"]; exists {
		failures, failDiags := DecodeDependsOn(attr)
		diags = append(diags, failDiags...)
		r.ExpectFailures = failures
	}

	return &r, diags
}

func decodeTestRunModuleBlock(block *hcl.Block) (*TestRunModuleCall, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := block.Body.Content(testRunModuleBlockSchema)
	diags = append(diags, contentDiags...)

	module := TestRunModuleCall{
		DeclRange: block.DefRange,
	}

	haveVersionArg := false
	if attr, exists := content.Attributes["version"]; exists {
		var versionDiags hcl.Diagnostics
		module.Version, versionDiags = decodeVersionConstraint(attr)
		diags = append(diags, versionDiags...)
		haveVersionArg = true
	}

	if attr, exists := content.Attributes["source"]; exists {
		module.SourceDeclRange = attr.Range

		var raw string
		rawDiags := gohcl.DecodeExpression(attr.Expr, nil, &raw)
		diags = append(diags, rawDiags...)
		if !rawDiags.HasErrors() {
			var err error
			if haveVersionArg {
				module.Source, err = moduleaddrs.ParseModuleSourceRegistry(raw)
			} else {
				module.Source, err = moduleaddrs.ParseModuleSource(raw)
			}
			if err != nil {
				// NOTE: We leave mc.SourceAddr as nil for any situation where the
				// source attribute is invalid, so any code which tries to carefully
				// use the partial result of a failed config decode must be
				// resilient to that.
				module.Source = nil

				// NOTE: In practice it's actually very unlikely to end up here,
				// because our source address parser can turn just about any string
				// into some sort of remote package address, and so for most errors
				// we'll detect them only during module installation. There are
				// still a _few_ purely-syntax errors we can catch at parsing time,
				// though, mostly related to remote package sub-paths and local
				// paths.
				switch err := err.(type) {
				case *moduleaddrs.MaybeRelativePathErr:
					diags = append(diags, &hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Invalid module source address",
						Detail: fmt.Sprintf(
							"Terraform failed to determine your intended installation method for remote module package %q.\n\nIf you intended this as a path relative to the current module, use \"./%s\" instead. The \"./\" prefix indicates that the address is a relative filesystem path.",
							err.Addr, err.Addr,
						),
						Subject: module.SourceDeclRange.Ptr(),
					})
				default:
					if haveVersionArg {
						// In this case we'll include some extra context that
						// we assumed a registry source address due to the
						// version argument.
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid registry module source address",
							Detail:   fmt.Sprintf("Failed to parse module registry address: %s.\n\nTerraform assumed that you intended a module registry source address because you also set the argument \"version\", which applies only to registry modules.", err),
							Subject:  module.SourceDeclRange.Ptr(),
						})
					} else {
						diags = append(diags, &hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Invalid module source address",
							Detail:   fmt.Sprintf("Failed to parse module source address: %s.", err),
							Subject:  module.SourceDeclRange.Ptr(),
						})
					}
				}
			}

			switch module.Source.(type) {
			case addrs.ModuleSourceRemote:
				// We only support local or registry modules when loading
				// modules directly from alternate sources during a test
				// execution.
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid module source address",
					Detail:   "Only local or registry module sources are currently supported from within test run blocks.",
					Subject:  module.SourceDeclRange.Ptr(),
				})
			}
		}
	} else {
		// Must have a source attribute.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Missing \"source\" attribute for module block",
			Detail:   "You must specify a source attribute when executing alternate modules during test executions.",
			Subject:  module.DeclRange.Ptr(),
		})
	}

	return &module, diags
}

func decodeTestRunOptionsBlock(block *hcl.Block) (*TestRunOptions, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := block.Body.Content(testRunOptionsBlockSchema)
	diags = append(diags, contentDiags...)

	opts := TestRunOptions{
		DeclRange: block.DefRange,
	}

	if attr, exists := content.Attributes["mode"]; exists {
		switch hcl.ExprAsKeyword(attr.Expr) {
		case "refresh-only":
			opts.Mode = RefreshOnlyTestMode
		case "normal":
			opts.Mode = NormalTestMode
		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid \"mode\" keyword",
				Detail:   "The \"mode\" argument requires one of the following keywords without quotes: normal or refresh-only",
				Subject:  attr.Expr.Range().Ptr(),
			})
		}
	} else {
		opts.Mode = NormalTestMode // Default to normal
	}

	if attr, exists := content.Attributes["refresh"]; exists {
		diags = append(diags, gohcl.DecodeExpression(attr.Expr, nil, &opts.Refresh)...)
	} else {
		// Defaults to true.
		opts.Refresh = true
	}

	if attr, exists := content.Attributes["replace"]; exists {
		reps, repsDiags := DecodeDependsOn(attr)
		diags = append(diags, repsDiags...)
		opts.Replace = reps
	}

	if attr, exists := content.Attributes["target"]; exists {
		tars, tarsDiags := DecodeDependsOn(attr)
		diags = append(diags, tarsDiags...)
		opts.Target = tars
	}

	if !opts.Refresh && opts.Mode == RefreshOnlyTestMode {
		// These options are incompatible.
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incompatible plan options",
			Detail:   "The \"refresh\" option cannot be set to false when running a test in \"refresh-only\" mode.",
			Subject:  content.Attributes["refresh"].Range.Ptr(),
		})
	}

	return &opts, diags
}

var testFileSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "run",
			LabelNames: []string{"name"},
		},
		{
			Type:       "provider",
			LabelNames: []string{"name"},
		},
		{
			Type:       "mock_provider",
			LabelNames: []string{"name"},
		},
		{
			Type: "variables",
		},
		{
			Type: "override_resource",
		},
		{
			Type: "override_data",
		},
		{
			Type: "override_module",
		},
	},
}

var testRunBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "command"},
		{Name: "providers"},
		{Name: "expect_failures"},
	},
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "plan_options",
		},
		{
			Type: "assert",
		},
		{
			Type: "variables",
		},
		{
			Type: "module",
		},
		{
			Type: "override_resource",
		},
		{
			Type: "override_data",
		},
		{
			Type: "override_module",
		},
	},
}

var testRunOptionsBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "mode"},
		{Name: "refresh"},
		{Name: "replace"},
		{Name: "target"},
	},
}

var testRunModuleBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "source"},
		{Name: "version"},
	},
}
