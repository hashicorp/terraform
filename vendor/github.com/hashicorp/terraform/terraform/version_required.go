package terraform

import (
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
)

// checkRequiredVersion verifies that any version requirements specified by
// the configuration are met.
//
// This checks the root module as well as any additional version requirements
// from child modules.
//
// This is tested in context_test.go.
func checkRequiredVersion(m *module.Tree) error {
	// Check any children
	for _, c := range m.Children() {
		if err := checkRequiredVersion(c); err != nil {
			return err
		}
	}

	var tf *config.Terraform
	if c := m.Config(); c != nil {
		tf = c.Terraform
	}

	// If there is no Terraform config or the required version isn't set,
	// we move on.
	if tf == nil || tf.RequiredVersion == "" {
		return nil
	}

	// Path for errors
	module := "root"
	if path := normalizeModulePath(m.Path()); len(path) > 1 {
		module = modulePrefixStr(path)
	}

	// Check this version requirement of this module
	cs, err := version.NewConstraint(tf.RequiredVersion)
	if err != nil {
		return fmt.Errorf(
			"%s: terraform.required_version %q syntax error: %s",
			module,
			tf.RequiredVersion, err)
	}

	if !cs.Check(SemVersion) {
		return fmt.Errorf(
			"The currently running version of Terraform doesn't meet the\n"+
				"version requirements explicitly specified by the configuration.\n"+
				"Please use the required version or update the configuration.\n"+
				"Note that version requirements are usually set for a reason, so\n"+
				"we recommend verifying with whoever set the version requirements\n"+
				"prior to making any manual changes.\n\n"+
				"  Module: %s\n"+
				"  Required version: %s\n"+
				"  Current version: %s",
			module,
			tf.RequiredVersion,
			SemVersion)
	}

	return nil
}
