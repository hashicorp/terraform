package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcldec"
	"github.com/hashicorp/hcl2/hcltest"
	"github.com/hashicorp/terraform/command/format"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/diffs"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// PlanCommand is a Command implementation that compares a Terraform
// configuration to an actual infrastructure and shows the differences.
type PlanCommand struct {
	Meta
}

func (c *PlanCommand) Run(args []string) int {
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	fmt.Printf("temporary plan\n\n")

	reqd := discovery.PluginRequirements{
		"aws": &discovery.PluginConstraints{
			Versions: discovery.ConstraintStr(">= 0.0.0").MustParse(),
		},
	}

	providerResolver := c.providerResolver()
	factories, errs := providerResolver.ResolveProviders(reqd)
	if len(errs) > 0 {
		c.showDiagnostics(errs[0])
		return 1
	}

	provider, err := factories["aws"]()
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}

	schema, err := provider.GetSchema(&terraform.ProviderSchemaRequest{
		ResourceTypes: []string{"aws_instance"},
	})
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}

	rSchema := schema.ResourceTypes["aws_instance"]
	//ty := rSchema.ImpliedType()

	addr, _ := terraform.ParseResourceAddress("aws_instance.example[2]")

	old := cty.ObjectVal(map[string]cty.Value{
		"ami":                  cty.StringVal("ami-abcd"),
		"instance_type":        cty.StringVal("z1.weedy"),
		"ebs_optimized":        cty.True,
		"iam_instance_profile": cty.NullVal(cty.String),
		"private_dns":          cty.StringVal("127.0.0.1.example.com"),
		"private_ip":           cty.StringVal("127.0.0.1"),
		"user_data":            cty.StringVal("#!/usr/bin/bash\necho howdy\ncat /etc/foo"),
		"vpc_security_group_ids": cty.SetVal([]cty.Value{
			cty.StringVal("sg-12354"),
		}),
		"root_block_device": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"volume_type": cty.StringVal("standard"),
				"volume_size": cty.NumberIntVal(0),
				"iops":        cty.NumberIntVal(0),
			}),
		}),
		"ebs_block_device": cty.SetVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{
				"device_name": cty.StringVal("foo"),
				"volume_type": cty.StringVal("standard"),
				"volume_size": cty.NumberIntVal(0),
				"iops":        cty.NumberIntVal(0),
				"encrypted":   cty.False,
				"snapshot_id": cty.StringVal("snap-abc123"),
			}),
		}),
	})

	old, err = configschema.ForceObjectConformance(old, rSchema)
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}
	newBody := hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcl.Attributes{
			"ami": {
				Name: "ami",
				Expr: hcltest.MockExprLiteral(cty.StringVal("ami-1234")),
			},
			"instance_type": {
				Name: "instance_type",
				Expr: hcltest.MockExprLiteral(cty.StringVal("z1.weedy")),
			},
			"ebs_optimized": {
				Name: "ebs_optimized",
				Expr: hcltest.MockExprLiteral(cty.False),
			},
			"user_data": {
				Name: "user_data",
				Expr: hcltest.MockExprLiteral(cty.StringVal("#!/usr/bin/bash\necho hello\ncat /etc/foo")),
			},
			"iam_instance_profile": {
				Name: "iam_instance_profile",
				Expr: hcltest.MockExprLiteral(cty.StringVal("arn:aws:foobarbaz")),
			},
			"vpc_security_group_ids": {
				Name: "vpc_security_group_ids",
				Expr: hcltest.MockExprLiteral(cty.SetVal([]cty.Value{
					cty.StringVal("sg-12354"),
					cty.StringVal("sg-abcde"),
				})),
			},
		},
		Blocks: hcl.Blocks{
			&hcl.Block{
				Type: "root_block_device",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"volume_type": {
							Name: "volume_type",
							Expr: hcltest.MockExprLiteral(cty.StringVal("gp2")),
						},
						"volume_size": {
							Name: "volume_size",
							Expr: hcltest.MockExprLiteral(cty.NumberIntVal(0)),
						},
						"iops": {
							Name: "iops",
							Expr: hcltest.MockExprLiteral(cty.NumberIntVal(0)),
						},
					},
				}),
			},
			&hcl.Block{
				Type: "ebs_block_device",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"device_name": {
							Name: "volume_type",
							Expr: hcltest.MockExprLiteral(cty.StringVal("foo")),
						},
						"volume_type": {
							Name: "volume_type",
							Expr: hcltest.MockExprLiteral(cty.StringVal("standard")),
						},
						"volume_size": {
							Name: "volume_size",
							Expr: hcltest.MockExprLiteral(cty.NumberIntVal(0)),
						},
						"iops": {
							Name: "iops",
							Expr: hcltest.MockExprLiteral(cty.NumberIntVal(0)),
						},
						"encrypted": {
							Name: "encrypted",
							Expr: hcltest.MockExprLiteral(cty.False),
						},
					},
				}),
			},
			&hcl.Block{
				Type: "ebs_block_device",
				Body: hcltest.MockBody(&hcl.BodyContent{
					Attributes: hcl.Attributes{
						"device_name": {
							Name: "volume_type",
							Expr: hcltest.MockExprLiteral(cty.StringVal("bar")),
						},
						"volume_type": {
							Name: "volume_type",
							Expr: hcltest.MockExprLiteral(cty.StringVal("standard")),
						},
						"volume_size": {
							Name: "volume_size",
							Expr: hcltest.MockExprLiteral(cty.NumberIntVal(0)),
						},
						"iops": {
							Name: "iops",
							Expr: hcltest.MockExprLiteral(cty.NumberIntVal(0)),
						},
					},
				}),
			},
		},
	})

	forcedReplace := diffs.NewPathSet()
	forcedReplace.AddAllSteps(cty.Path{
		cty.GetAttrStep{
			Name: "ami",
		},
	})
	forcedReplace.AddAllSteps(cty.Path{
		cty.GetAttrStep{
			Name: "user_data",
		},
	})
	forcedReplace.AddAllSteps(cty.Path{
		cty.GetAttrStep{
			Name: "vpc_security_group_ids",
		},
	})
	forcedReplace.AddAllSteps(cty.Path{
		cty.GetAttrStep{
			Name: "ebs_block_device",
		},
	})
	forcedReplace.AddAllSteps(cty.Path{
		cty.GetAttrStep{
			Name: "root_block_device",
		},
		cty.IndexStep{
			Key: cty.NumberIntVal(0),
		},
		cty.GetAttrStep{
			Name: "volume_type",
		},
	})

	new, diags := hcldec.Decode(newBody, rSchema.DecoderSpec(), nil)
	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	// Propagate computed values from old into new, so that they remain
	// unchanged unless overridden.
	//new = diffs.PreserveComputedAttrs(old, new, rSchema)

	fmt.Printf("--- old %#v\n\n--- new %#v\n\n--- forcedReplace %s\n", old, new, spew.Sdump(forcedReplace))

	//change := diffs.NewCreate(new)
	//change := diffs.NewUpdate(old, new)
	//change := diffs.NewUpdate(new, new)
	change := diffs.NewReplace(old, new, forcedReplace)

	diff := format.ResourceChange(addr, change, rSchema, c.Colorize())
	fmt.Println(diff)

	return 0
}

func (c *PlanCommand) NormalRun(args []string) int {
	var destroy, refresh, detailed bool
	var outPath string
	var moduleDepth int

	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.flagSet("plan")
	cmdFlags.BoolVar(&destroy, "destroy", false, "destroy")
	cmdFlags.BoolVar(&refresh, "refresh", true, "refresh")
	c.addModuleDepthFlag(cmdFlags, &moduleDepth)
	cmdFlags.StringVar(&outPath, "out", "", "path")
	cmdFlags.IntVar(
		&c.Meta.parallelism, "parallelism", DefaultParallelism, "parallelism")
	cmdFlags.StringVar(&c.Meta.statePath, "state", "", "path")
	cmdFlags.BoolVar(&detailed, "detailed-exitcode", false, "detailed-exitcode")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	// Check if the path is a plan
	plan, err := c.Plan(configPath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if plan != nil {
		// Disable refreshing no matter what since we only want to show the plan
		refresh = false

		// Set the config path to empty for backend loading
		configPath = ""
	}

	var diags tfdiags.Diagnostics

	// Load the module if we don't have one yet (not running from plan)
	var mod *module.Tree
	if plan == nil {
		var modDiags tfdiags.Diagnostics
		mod, modDiags = c.Module(configPath)
		diags = diags.Append(modDiags)
		if modDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}

	var conf *config.Config
	if mod != nil {
		conf = mod.Config()
	}
	// Load the backend
	b, err := c.Backend(&BackendOpts{
		Config: conf,
		Plan:   plan,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	// Build the operation
	opReq := c.Operation()
	opReq.Destroy = destroy
	opReq.Module = mod
	opReq.Plan = plan
	opReq.PlanRefresh = refresh
	opReq.PlanOutPath = outPath
	opReq.Type = backend.OperationTypePlan

	// Perform the operation
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	op, err := b.Operation(ctx, opReq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error starting operation: %s", err))
		return 1
	}

	select {
	case <-c.ShutdownCh:
		// Cancel our context so we can start gracefully exiting
		ctxCancel()

		// Notify the user
		c.Ui.Output(outputInterrupt)

		// Still get the result, since there is still one
		select {
		case <-c.ShutdownCh:
			c.Ui.Error(
				"Two interrupts received. Exiting immediately")
			return 1
		case <-op.Done():
		}
	case <-op.Done():
		if err := op.Err; err != nil {
			diags = diags.Append(err)
		}
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	if detailed && !op.PlanEmpty {
		return 2
	}

	return 0
}

func (c *PlanCommand) Help() string {
	helpText := `
Usage: terraform plan [options] [DIR-OR-PLAN]

  Generates an execution plan for Terraform.

  This execution plan can be reviewed prior to running apply to get a
  sense for what Terraform will do. Optionally, the plan can be saved to
  a Terraform plan file, and apply can take this plan file to execute
  this plan exactly.

  If a saved plan is passed as an argument, this command will output
  the saved plan contents. It will not modify the given plan.

Options:

  -destroy            If set, a plan will be generated to destroy all resources
                      managed by the given configuration and state.

  -detailed-exitcode  Return detailed exit codes when the command exits. This
                      will change the meaning of exit codes to:
                      0 - Succeeded, diff is empty (no changes)
                      1 - Errored
                      2 - Succeeded, there is a diff

  -input=true         Ask for input for variables if not directly set.

  -lock=true          Lock the state file when locking is supported.

  -lock-timeout=0s    Duration to retry a state lock.

  -module-depth=n     Specifies the depth of modules to show in the output.
                      This does not affect the plan itself, only the output
                      shown. By default, this is -1, which will expand all.

  -no-color           If specified, output won't contain any color.

  -out=path           Write a plan file to the given path. This can be used as
                      input to the "apply" command.

  -parallelism=n      Limit the number of concurrent operations. Defaults to 10.

  -refresh=true       Update state prior to checking for differences.

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default it will
                      use the state "terraform.tfstate" if it exists.

  -target=resource    Resource to target. Operation will be limited to this
                      resource and its dependencies. This flag can be used
                      multiple times.

  -var 'foo=bar'      Set a variable in the Terraform configuration. This
                      flag can be set multiple times.

  -var-file=foo       Set variables in the Terraform configuration from
                      a file. If "terraform.tfvars" or any ".auto.tfvars"
                      files are present, they will be automatically loaded.
`
	return strings.TrimSpace(helpText)
}

func (c *PlanCommand) Synopsis() string {
	return "Generate and show an execution plan"
}
