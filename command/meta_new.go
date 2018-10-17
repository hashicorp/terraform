package command

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/terraform/plans/planfile"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/tfdiags"
)

// NOTE: Temporary file until this branch is cleaned up.

// Input returns whether or not input asking is enabled.
func (m *Meta) Input() bool {
	if test || !m.input {
		return false
	}

	if envVar := os.Getenv(InputModeEnvVar); envVar != "" {
		if v, err := strconv.ParseBool(envVar); err == nil && !v {
			return false
		}
	}

	return true
}

// Module loads the module tree for the given root path using the legacy
// configuration loader.
//
// It expects the modules to already be downloaded. This will never
// download any modules.
//
// The configuration is validated before returning, so the returned diagnostics
// may contain warnings and/or errors. If the diagnostics contains only
// warnings, the caller may treat the returned module.Tree as valid after
// presenting the warnings to the user.
func (m *Meta) Module(path string) (*module.Tree, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	mod, err := module.NewTreeModule("", path)
	if err != nil {
		// Check for the error where we have no config files
		if errwrap.ContainsType(err, new(config.ErrNoConfigsFound)) {
			return nil, nil
		}

		diags = diags.Append(err)
		return nil, diags
	}

	err = mod.Load(m.moduleStorage(m.DataDir(), module.GetModeNone))
	if err != nil {
		diags = diags.Append(errwrap.Wrapf("Error loading modules: {{err}}", err))
		return nil, diags
	}

	diags = diags.Append(mod.Validate())

	return mod, diags
}

// Config loads the root config for the path specified, using the legacy
// configuration loader.
//
// Path may be a directory or file. The absence of configuration is not an
// error and returns a nil Config.
func (m *Meta) Config(path string) (*config.Config, error) {
	// If no explicit path was given then it is okay for there to be
	// no backend configuration found.
	emptyOk := path == ""

	// If we had no path set, it is an error. We can't initialize unset
	if path == "" {
		path = "."
	}

	// Expand the path
	if !filepath.IsAbs(path) {
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf(
				"Error expanding path to backend config %q: %s", path, err)
		}
	}

	log.Printf("[DEBUG] command: loading backend config file: %s", path)

	// We first need to determine if we're loading a file or a directory.
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) && emptyOk {
			log.Printf(
				"[INFO] command: backend config not found, returning nil: %s",
				path)
			return nil, nil
		}

		return nil, err
	}

	var f func(string) (*config.Config, error) = config.LoadFile
	if fi.IsDir() {
		f = config.LoadDir
	}

	// Load the configuration
	c, err := f(path)
	if err != nil {
		// Check for the error where we have no config files and return nil
		// as the configuration type.
		if errwrap.ContainsType(err, new(config.ErrNoConfigsFound)) {
			log.Printf(
				"[INFO] command: backend config not found, returning nil: %s",
				path)
			return nil, nil
		}

		return nil, err
	}

	return c, nil
}

// PlanFile returns a reader for the plan file at the given path.
//
// If the return value and error are both nil, the given path exists but seems
// to be a configuration directory instead.
//
// Error will be non-nil if path refers to something which looks like a plan
// file and loading the file fails.
func (m *Meta) PlanFile(path string) (*planfile.Reader, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		// Looks like a configuration directory.
		return nil, nil
	}

	return planfile.Open(path)
}
