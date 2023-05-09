package terraform

import (
	"fmt"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// GenerateConfig will write any generated config from the plan into the writer
// returned by Context.generatedConfigWriter.
//
// This function returns diagnostics, but these will only be warnings. The
// failure states in here are unlikely provided everything has been set up
// correctly elsewhere. It is also possible for users to recover from errors in
// here by writing out the config themselves, so we don't consider errors to be
// showstopping.
func (c *Context) GenerateConfig(plan *plans.Plan) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	var changes []*plans.ResourceInstanceChangeSrc
	for _, change := range plan.Changes.Resources {
		if len(change.GeneratedConfig) > 0 {
			changes = append(changes, change)
		}
	}

	if c.generatedConfigWriter == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Failed to generate config for imported state",
			fmt.Sprintf(
				"Terraform had no way of writing the generated config into any destination, all imported config must be created manually.\n\nThis is a bug in Terraform; please report it!",
			),
		))
		return diags
	}

	writer, closer, err := c.generatedConfigWriter()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"Failed to generate config for imported state",
			fmt.Sprintf(
				"Terraform could not create a destination for the generated config (%v), all imported config must be created manually.\n\nThis is a bug in Terraform; please report it!",
				err,
			),
		))
		return diags
	}
	defer closer()

	for _, change := range changes {

		diag := func(err error) tfdiags.Diagnostic {
			return tfdiags.Sourceless(
				tfdiags.Warning,
				"Failed to generate config for imported state",
				fmt.Sprintf(
					"Terraform encountered an error (%v) while writing generated config, the config for %s must be created manually.\n\n`terraform state show %s` will print the existing state to help with this.\n\nThis is a bug in Terraform; please report it!",
					err, change.Addr, change.Addr,
				),
			)
		}

		if change.Importing != nil && len(change.Importing.ID) > 0 {
			if _, err := writer.Write([]byte(fmt.Sprintf("\n# __generated__ by Terraform from %q\n", change.Importing.ID))); err != nil {
				diags = diags.Append(diag(err))
				continue
			}
		} else {
			if _, err := writer.Write([]byte(fmt.Sprintf("\n# __generated__ by Terraform\n"))); err != nil {
				diags = diags.Append(diag(err))
				continue
			}
		}
		if _, err := writer.Write([]byte(fmt.Sprintln(change.GeneratedConfig))); err != nil {
			diags = diags.Append(diag(err))
			continue
		}
	}

	return diags
}
