// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/registry"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// normalizePath normalizes a given path so that it is, if possible, relative
// to the current working directory. This is primarily used to prepare
// paths used to load configuration, because we want to prefer recording
// relative paths in source code references within the configuration.
func (m *Meta) normalizePath(path string) string {
	m.fixupMissingWorkingDir()
	return m.WorkingDir.NormalizePath(path)
}

// loadConfig reads a configuration from the given directory, which should
// contain a root module and have already have any required descendant modules
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

// loadConfigWithTests matches loadConfig, except it also loads any test files
// into the config alongside the main configuration.
func (m *Meta) loadConfigWithTests(rootDir, testDir string) (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	rootDir = m.normalizePath(rootDir)

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	config, hclDiags := loader.LoadConfigWithTests(rootDir, testDir)
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

// loadSingleModuleWithTests matches loadSingleModule except it also loads any
// tests for the target module.
func (m *Meta) loadSingleModuleWithTests(dir string, testDir string) (*configs.Module, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	dir = m.normalizePath(dir)

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	module, hclDiags := loader.Parser().LoadConfigDirWithTests(dir, testDir)
	diags = diags.Append(hclDiags)
	return module, diags
}

// dirIsConfigPath checks if the given path is a directory that contains at
// least one Terraform configuration file (.tf or .tf.json), returning true
// if so.
//
// In the unlikely event that the underlying config loader cannot be initalized,
// this function optimistically returns true, assuming that the caller will
// then do some other operation that requires the config loader and get an
// error at that point.
func (m *Meta) dirIsConfigPath(dir string) bool {
	loader, err := m.initConfigLoader()
	if err != nil {
		return true
	}

	return loader.IsConfigDir(dir)
}

// loadBackendConfig reads configuration from the given directory and returns
// the backend configuration defined by that module, if any. Nil is returned
// if the specified module does not have an explicit backend configuration.
//
// This is a convenience method for command code that will delegate to the
// configured backend to do most of its work, since in that case it is the
// backend that will do the full configuration load.
//
// Although this method returns only the backend configuration, at present it
// actually loads and validates the entire configuration first. Therefore errors
// returned may be about other aspects of the configuration. This behavior may
// change in future, so callers must not rely on it. (That is, they must expect
// that a call to loadSingleModule or loadConfig could fail on the same
// directory even if loadBackendConfig succeeded.)
func (m *Meta) loadBackendConfig(rootDir string) (*configs.Backend, tfdiags.Diagnostics) {
	mod, diags := m.loadSingleModule(rootDir)

	// Only return error diagnostics at this point. Any warnings will be caught
	// again later and duplicated in the output.
	if diags.HasErrors() {
		return nil, diags
	}

	if mod.CloudConfig != nil {
		backendConfig := mod.CloudConfig.ToBackendConfig()
		return &backendConfig, nil
	}

	return mod.Backend, nil
}

// loadHCLFile reads an arbitrary HCL file and returns the unprocessed body
// representing its toplevel. Most callers should use one of the more
// specialized "load..." methods to get a higher-level representation.
func (m *Meta) loadHCLFile(filename string) (hcl.Body, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	filename = m.normalizePath(filename)

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return nil, diags
	}

	body, hclDiags := loader.Parser().LoadHCLFile(filename)
	diags = diags.Append(hclDiags)
	return body, diags
}

// installModules reads a root module from the given directory and attempts
// recursively to install all of its descendant modules.
//
// The given hooks object will be notified of installation progress, which
// can then be relayed to the end-user. The uiModuleInstallHooks type in
// this package has a reasonable implementation for displaying notifications
// via a provided cli.Ui.
func (m *Meta) installModules(ctx context.Context, rootDir, testsDir string, upgrade, installErrsOnly bool, hooks initwd.ModuleInstallHooks) (abort bool, diags tfdiags.Diagnostics) {
	ctx, span := tracer.Start(ctx, "install modules")
	defer span.End()

	rootDir = m.normalizePath(rootDir)

	err := os.MkdirAll(m.modulesDir(), os.ModePerm)
	if err != nil {
		diags = diags.Append(fmt.Errorf("failed to create local modules directory: %s", err))
		return true, diags
	}

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return true, diags
	}

	inst := initwd.NewModuleInstaller(m.modulesDir(), loader, m.registryClient())

	_, moreDiags := inst.InstallModules(ctx, rootDir, testsDir, upgrade, installErrsOnly, hooks)
	diags = diags.Append(moreDiags)

	if ctx.Err() == context.Canceled {
		m.showDiagnostics(diags)
		diags = diags.Append(fmt.Errorf("Module installation was canceled by an interrupt signal."))
		return true, diags
	}

	return false, diags
}

// initDirFromModule initializes the given directory (which should be
// pre-verified as empty by the caller) by copying the source code from the
// given module address.
//
// Internally this runs similar steps to installModules.
// The given hooks object will be notified of installation progress, which
// can then be relayed to the end-user. The uiModuleInstallHooks type in
// this package has a reasonable implementation for displaying notifications
// via a provided cli.Ui.
func (m *Meta) initDirFromModule(ctx context.Context, targetDir string, addr string, hooks initwd.ModuleInstallHooks) (abort bool, diags tfdiags.Diagnostics) {
	ctx, span := tracer.Start(ctx, "initialize directory from module", trace.WithAttributes(
		attribute.String("source_addr", addr),
	))
	defer span.End()

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return true, diags
	}

	targetDir = m.normalizePath(targetDir)
	moreDiags := initwd.DirFromModule(ctx, loader, targetDir, m.modulesDir(), addr, m.registryClient(), hooks)
	diags = diags.Append(moreDiags)
	if ctx.Err() == context.Canceled {
		m.showDiagnostics(diags)
		diags = diags.Append(fmt.Errorf("Module initialization was canceled by an interrupt signal."))
		return true, diags
	}
	return false, diags
}

// inputForSchema uses interactive prompts to try to populate any
// not-yet-populated required attributes in the given object value to
// comply with the given schema.
//
// An error will be returned if input is disabled for this meta or if
// values cannot be obtained for some other operational reason. Errors are
// not returned for invalid input since the input loop itself will report
// that interactively.
//
// It is not guaranteed that the result will be valid, since certain attribute
// types and nested blocks are not supported for input.
//
// The given value must conform to the given schema. If not, this method will
// panic.
func (m *Meta) inputForSchema(given cty.Value, schema *configschema.Block) (cty.Value, error) {
	if given.IsNull() || !given.IsKnown() {
		// This is not reasonable input, but we'll tolerate it anyway and
		// just pass it through for the caller to handle downstream.
		return given, nil
	}

	retVals := given.AsValueMap()
	names := make([]string, 0, len(schema.Attributes))
	for name, attrS := range schema.Attributes {
		if attrS.Required && retVals[name].IsNull() && attrS.Type.IsPrimitiveType() {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	input := m.UIInput()
	for _, name := range names {
		attrS := schema.Attributes[name]

		for {
			strVal, err := input.Input(context.Background(), &terraform.InputOpts{
				Id:          name,
				Query:       name,
				Description: attrS.Description,
			})
			if err != nil {
				return cty.UnknownVal(schema.ImpliedType()), fmt.Errorf("%s: %s", name, err)
			}

			val := cty.StringVal(strVal)
			val, err = convert.Convert(val, attrS.Type)
			if err != nil {
				m.showDiagnostics(fmt.Errorf("Invalid value: %s", err))
				continue
			}

			retVals[name] = val
			break
		}
	}

	return cty.ObjectVal(retVals), nil
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

// registerSynthConfigSource allows commands to add synthetic additional source
// buffers to the config loader's cache of sources (as returned by
// configSources), which is useful when a command is directly parsing something
// from the command line that may produce diagnostics, so that diagnostic
// snippets can still be produced.
//
// If this is called before a configLoader has been initialized then it will
// try to initialize the loader but ignore any initialization failure, turning
// the call into a no-op. (We presume that a caller will later call a different
// function that also initializes the config loader as a side effect, at which
// point those errors can be returned.)
func (m *Meta) registerSynthConfigSource(filename string, src []byte) {
	loader, err := m.initConfigLoader()
	if err != nil || loader == nil {
		return // treated as no-op, since this is best-effort
	}
	loader.Parser().ForceFileSource(filename, src)
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
		})
		if err != nil {
			return nil, err
		}
		loader.AllowLanguageExperiments(m.AllowExperimentalFeatures)
		m.configLoader = loader
		if m.View != nil {
			m.View.SetConfigSources(loader.Sources)
		}
	}
	return m.configLoader, nil
}

// registryClient instantiates and returns a new Terraform Registry client.
func (m *Meta) registryClient() *registry.Client {
	return registry.NewClient(m.Services, nil)
}

// configValueFromCLI parses a configuration value that was provided in a
// context in the CLI where only strings can be provided, such as on the
// command line or in an environment variable, and returns the resulting
// value.
func configValueFromCLI(synthFilename, rawValue string, wantType cty.Type) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	switch {
	case wantType.IsPrimitiveType():
		// Primitive types are handled as conversions from string.
		val := cty.StringVal(rawValue)
		var err error
		val, err = convert.Convert(val, wantType)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid backend configuration value",
				fmt.Sprintf("Invalid backend configuration argument %s: %s", synthFilename, err),
			))
			val = cty.DynamicVal // just so we return something valid-ish
		}
		return val, diags
	default:
		// Non-primitives are parsed as HCL expressions
		src := []byte(rawValue)
		expr, hclDiags := hclsyntax.ParseExpression(src, synthFilename, hcl.Pos{Line: 1, Column: 1})
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			return cty.DynamicVal, diags
		}
		val, hclDiags := expr.Value(nil)
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			val = cty.DynamicVal
		}
		return val, diags
	}
}
