package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
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

	// CheckRules defines the list of assertions/validations that should be
	// checked by this run block.
	CheckRules []*CheckRule

	NameDeclRange      hcl.Range
	VariablesDeclRange hcl.Range
	DeclRange          hcl.Range
}

// TestRunOptions contains the plan options for a given run block.
type TestRunOptions struct {
	// Mode is the planning mode to run in. One of ['normal', 'refresh-only'].
	Mode TestMode

	// Refresh is analogous to the -refresh=false Terraform plan option.
	Refresh bool

	// Replace is analogous to the -refresh=ADDRESS Terraform plan option.
	Replace []hcl.Traversal

	// Target is analogous to the -target=ADDRESS Terraform plan option.
	Target []hcl.Traversal

	DeclRange hcl.Range
}

func loadTestFile(body hcl.Body) (*TestFile, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := body.Content(testFileSchema)
	diags = append(diags, contentDiags...)

	tf := TestFile{}

	for _, block := range content.Blocks {
		switch block.Type {
		case "run":
			run, runDiags := decodeTestRunBlock(block)
			diags = append(diags, runDiags...)
			if !runDiags.HasErrors() {
				tf.Runs = append(tf.Runs, run)
			}
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
		}
	}

	return &tf, diags
}

func decodeTestRunBlock(block *hcl.Block) (*TestRun, hcl.Diagnostics) {
	var diags hcl.Diagnostics

	content, contentDiags := block.Body.Content(testRunBlockSchema)
	diags = append(diags, contentDiags...)

	r := TestRun{
		Name:          block.Labels[0],
		NameDeclRange: block.LabelRanges[0],
		DeclRange:     block.DefRange,
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

	return &r, diags
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
		reps, repsDiags := decodeDependsOn(attr)
		diags = append(diags, repsDiags...)
		opts.Replace = reps
	}

	if attr, exists := content.Attributes["target"]; exists {
		tars, tarsDiags := decodeDependsOn(attr)
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
			Type: "variables",
		},
	},
}

var testRunBlockSchema = &hcl.BodySchema{
	Attributes: []hcl.AttributeSchema{
		{Name: "command"},
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
