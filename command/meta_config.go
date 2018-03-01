package command

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// normalizePath normalizes a given path so that it is, if possible, relative
// to the current working directory. This is primarily used to prepare
// paths used to load configuration, because we want to prefer recording
// relative paths in source code references within the configuration.
func (m *Meta) normalizePath(path string) string {
	var err error

	// First we will make it absolute so that we have a consistent place
	// to start.
	path, err = filepath.Abs(path)
	if err != nil {
		// We'll just accept what we were given, then.
		return path
	}

	cwd, err := os.Getwd()
	if err != nil || !filepath.IsAbs(cwd) {
		return path
	}

	ret, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}

	return ret
}

// loadConfig reads a configuration from the given directory, which should
// contain a root module and have already have any required descendent modules
// installed.
func (m *Meta) loadConfig(rootDir string) (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	rootDir = m.normalizePath(rootDir)

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	config, hclDiags := loader.LoadConfig(rootDir)
	diags = diags.Append(hclDiags)
	return config, diags
}

// loadSingleModule reads configuration from the given directory and returns
// a description of that module only, without attempting to assemble a module
// tree for referenced child modules.
//
// Most callers should use loadConfig. This method exists to support early
// initialization use-cases where the root module must be inspected in order
// to determine what else needs to be installed before the full configuration
// can be used.
func (m *Meta) loadSingleModule(dir string) (*configs.Module, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	dir = m.normalizePath(dir)

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	module, hclDiags := loader.Parser().LoadConfigDir(dir)
	diags = diags.Append(hclDiags)
	return module, diags
}

// installModules reads a root module from the given directory and attempts
// recursively install all of its descendent modules.
//
// The given hooks object will be notified of installation progress, which
// can then be relayed to the end-user. The moduleUiInstallHooks type in
// this package has a reasonable implementation for displaying notifications
// via a provided cli.Ui.
func (m *Meta) installModules(rootDir string, upgrade bool, hooks configload.InstallHooks) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	rootDir = m.normalizePath(rootDir)

	err := os.MkdirAll(m.modulesDir(), os.ModePerm)
	if err != nil {
		diags = diags.Append(fmt.Errorf("failed to create local modules directory: %s", err))
		return diags
	}

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return diags
	}

	hclDiags := loader.InstallModules(rootDir, upgrade, hooks)
	diags = diags.Append(hclDiags)
	return diags
}

// initDirFromModule initializes the given directory (which should be
// pre-verified as empty by the caller) by copying the source code from the
// given module address.
//
// Internally this runs similar steps to installModules.
// The given hooks object will be notified of installation progress, which
// can then be relayed to the end-user. The moduleUiInstallHooks type in
// this package has a reasonable implementation for displaying notifications
// via a provided cli.Ui.
func (m *Meta) initDirFromModule(targetDir string, addr string, hooks configload.InstallHooks) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	targetDir = m.normalizePath(targetDir)

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return diags
	}

	hclDiags := loader.InitDirFromModule(targetDir, addr, hooks)
	diags = diags.Append(hclDiags)
	return diags
}

// loadVarsFile reads a file from the given path and interprets it as a
// "vars file", returning the contained values as a map.
//
// The file is read using the parser associated with the receiver's
// configuration loader, which means that the file's contents will be added
// to the source cache that is used for config snippets in diagnostic messages.
func (m *Meta) loadVarsFile(filename string) (map[string]cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	parser := loader.Parser()
	ret, hclDiags := parser.LoadValuesFile(filename)
	diags = diags.Append(hclDiags)
	return ret, diags
}

// configSources returns the source cache from the receiver's config loader,
// which the caller must not modify.
//
// If a config loader has not yet been instantiated then no files could have
// been loaded already, so this method returns a nil map in that case.
func (m *Meta) configSources() map[string][]byte {
	if m.configLoader == nil {
		return nil
	}

	return m.configLoader.Sources()
}

func (m *Meta) modulesDir() string {
	return filepath.Join(m.DataDir(), "modules")
}

// initConfigLoader initializes the shared configuration loader if it isn't
// already initialized.
//
// If the loader cannot be created for some reason then an error is returned
// and no loader is created. Subsequent calls will presumably see the same
// error. Loader initialization errors will tend to prevent any further use
// of most Terraform features, so callers should report any error and safely
// terminate.
func (m *Meta) initConfigLoader() (*configload.Loader, error) {
	if m.configLoader == nil {
		loader, err := configload.NewLoader(&configload.Config{
			ModulesDir: m.modulesDir(),
			Services:   m.Services,
			Creds:      m.Credentials,
		})
		if err != nil {
			return nil, err
		}
		m.configLoader = loader
	}
	return m.configLoader, nil
}
