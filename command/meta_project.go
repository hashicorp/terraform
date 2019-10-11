package command

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/projects"
	"github.com/hashicorp/terraform/tfdiags"
)

// findProjectForDir finds the project that the given start directory
// belongs to, or an error if the given directory does not seem to belong to
// a project.
func (m *Meta) findProjectForDir(dir string) (*projects.Project, tfdiags.Diagnostics) {
	return projects.FindProject(dir)
}

// findCurrentProject finds the project that the current working directory
// belongs to, or an error if either the current working directory does not
// seem to belong to any project or for some reason we can't find the current
// working directory.
func (m *Meta) findCurrentProject() (*projects.Project, tfdiags.Diagnostics) {
	dir, err := os.Getwd()
	if err != nil {
		// Failing to determine the current working directory should be a very
		// rare situation, so this error message is not super polished or
		// actionable.
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't determine current working directory",
			fmt.Sprintf("Failed to determine the current working directory: %s.", err),
		))
		return nil, diags
	}
	return projects.FindProject(dir)
}
