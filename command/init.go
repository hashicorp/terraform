package command

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/posener/complete"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/earlyconfig"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/projects"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// InitCommand is a Command implementation that takes a Terraform
// module and clones it to the working directory.
type InitCommand struct {
	Meta

	// getPlugins is for the -get-plugins flag
	getPlugins bool

	// providerInstaller is used to download and install providers that
	// aren't found locally. This uses a discovery.ProviderInstaller instance
	// by default, but it can be overridden here as a way to mock fetching
	// providers for tests.
	providerInstaller discovery.Installer
}

func (c *InitCommand) Run(args []string) int {
	var flagFromModule string
	var flagBackend, flagGet, flagUpgrade bool
	var flagPluginPath FlagStringSlice
	var flagVerifyPlugins bool
	flagConfigExtra := newRawFlags("-backend-config")

	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.extendedFlagSet("init")
	cmdFlags.BoolVar(&flagBackend, "backend", true, "")
	cmdFlags.Var(flagConfigExtra, "backend-config", "")
	cmdFlags.StringVar(&flagFromModule, "from-module", "", "copy the source of the given module into the directory before init")
	cmdFlags.BoolVar(&flagGet, "get", true, "")
	cmdFlags.BoolVar(&c.getPlugins, "get-plugins", true, "")
	cmdFlags.BoolVar(&c.forceInitCopy, "force-copy", false, "suppress prompts about copying state data")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.BoolVar(&c.reconfigure, "reconfigure", false, "reconfigure")
	cmdFlags.BoolVar(&flagUpgrade, "upgrade", false, "")
	cmdFlags.Var(&flagPluginPath, "plugin-dir", "plugin directory")
	cmdFlags.BoolVar(&flagVerifyPlugins, "verify-plugins", true, "verify plugins")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var diags tfdiags.Diagnostics

	if len(flagPluginPath) > 0 {
		c.pluginPath = flagPluginPath
		c.getPlugins = false
	}

	// set providerInstaller if we don't have a test version already
	if c.providerInstaller == nil {
		c.providerInstaller = &discovery.ProviderInstaller{
			Dir:                   c.pluginDir(),
			Cache:                 c.pluginCache(),
			PluginProtocolVersion: discovery.PluginInstallProtocolVersion,
			SkipVerify:            !flagVerifyPlugins,
			Ui:                    c.Ui,
			Services:              c.Services,
		}
	}

	// Validate the arg count
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The init command expects at most one argument.\n")
		cmdFlags.Usage()
		return 1
	}

	if err := c.storePluginPath(c.pluginPath); err != nil {
		c.Ui.Error(fmt.Sprintf("Error saving -plugin-path values: %s", err))
		return 1
	}

	if flagFromModule != "" {
		// -from-module doesn't make sense in a world where we're initalizing
		// whole projects rather than individual configuration directories.
		// TODO: Find a suitable replacement for this functionality that
		// still meets the use-case of allowing folks to publish example
		// projects/configurations for others to derive their repositories from.
		c.Ui.Error("The -from-module argument is no longer supported")
		return 1
	}

	if _, err := c.findCurrentProjectRoot(); err != nil {
		// Try to initialize a new project
		_, moreDiags := c.initNewProject()
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}

	// FIXME: We need to do something special here, rather than just calling
	// findCurrentProjectManager, because "terraform init" is supposed to be
	// the command that will establish the context values that the project
	// manager will use for evaluation, and we will need to pass those in here.
	projectMgr, moreDiags := c.findCurrentProjectManager()
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	projectRoot := projectMgr.Project().Config().ProjectRoot
	projectRootDisp, err := filepath.Abs(projectRoot)
	if err != nil {
		// If we can't build an absolute path then we'll just accept the
		// relative one, rather than failing. (This is unlikely)
		projectRootDisp = projectRoot
	}
	c.Ui.Output(fmt.Sprintf("Initializing project in %s...", projectRootDisp))

	workspaceAddrs := projectMgr.Project().AllWorkspaceAddrs()

	// To fully initialize, we need to load and analyze all of the configuration
	// directories, but we only need to load each distinct directory once, so
	// we'll collect them up from all the workspaces but dedupe them.
	// We need to evaluate each of the workspaces separately here because
	// if there are dependencies between them then we might not yet have enough
	// information to fully instantiate the downstream ones.
	workspacesByConfigDir := make(map[string][]*projects.Workspace)
	for _, addr := range workspaceAddrs {
		// FIXME: Loading the workspaces individually here is required to
		// allow for the fact that some of them may depend on upstream data
		// not yet available, but has the downside that we can end up
		// re-evaluating the same local value expressions multiple times and
		// reporting any errors from them more than once as a result.
		ws, moreDiags := projectMgr.LoadWorkspace(addr)
		if moreDiags.HasErrors() {
			// FIXME: This approach is problematic because when bootstrapping
			// a new project this warning will appear and the project will
			// only be partially initialized, but yet once it's fully initialized
			// we'll also mis-report this on any other error in the workspace
			// configuration. Need to think some more about how to model this
			// better so we can handle the config directory not yet being
			// determined for some workspaces, and hopefully also to separate
			// that from whether other expressions in the workspace
			// configuration can be determined.
			// This also obscures any _actual_ errors in the configuration.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Warning,
				"Workspace configuration error",
				fmt.Sprintf("Workspace %s cannot be initialized because its configuration has errors. This might be caused by dependencies on other workspaces that have not yet been applied for the first time. If so, apply those dependent workspaces first, and then run init again to complete initialization.", addr),
			))
			continue
		}
		dir := c.normalizePath(filepath.Join(projectRoot, ws.ConfigDir()))
		workspacesByConfigDir[dir] = append(workspacesByConfigDir[dir], ws)
	}
	for dir, workspaces := range workspacesByConfigDir {
		fmt.Printf("Directory %s has %d workspaces associated\n", dir, len(workspaces))

		// The descendent modules might not be initialized yet, so we'll start
		// be loading only the root module, then do our module installation,
		// before finally loading the full configuration for this directory.
		rootModEarly, earlyConfDiags := c.loadSingleModuleEarly(dir)
		if earlyConfDiags.HasErrors() {
			// Proceed with other directories too, so we can make as much
			// progress as possible before exiting.
			// Note that we only append the diags when errors are present
			// here because we're going to reload the root module as part
			// of loading the full configuration below and we don't want to
			// double up any warnings coming from loadSingleModuleEarly.
			diags = diags.Append(earlyConfDiags)
			continue
		}

		_, modsDiags := c.getModules(dir, rootModEarly, flagUpgrade)
		diags = diags.Append(modsDiags)
		if modsDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}

		// With all of the modules (hopefully) installed, we can now try to load
		// the whole configuration tree. We're using the early config loader
		// here just because it's pervasively used in the rest of "terraform init"
		// and thus avoids significant rework here while we're exploring the
		// workspaces2 prototype.
		// FIXME: Can we remove the early config loader altogether once we
		// eliminate the 0.12upgrade command in 0.13? It might still be useful
		// in allowing us to do CheckCoreVersionRequirements below, cause it
		// has more chance of reading version constraints from a 0.11-or-prior
		// config.
		earlyConfig, earlyConfDiags := c.loadConfigEarly(dir)
		diags = diags.Append(earlyConfDiags)
		if earlyConfDiags.HasErrors() {
			// Proceed with other directories too, so we can make as much
			// progress as possible before exiting.
			diags = diags.Append(earlyConfDiags)
			continue
		}

		// Before we go further, we'll check to make sure none of the modules
		// in the configuration declare that they don't support this Terraform
		// version, so we can produce a version-related error message rather
		// than potentially-confusing downstream errors.
		versionDiags := initwd.CheckCoreVersionRequirements(earlyConfig)
		diags = diags.Append(versionDiags)
		if versionDiags.HasErrors() {
			continue
		}

		// Now that we have loaded all modules, check the module tree for missing providers.
		// FIXME: We should also iterate over all the workspaces that use
		// this config, fetch their latest state snapshots, and try to install
		// providers that are only used for "orphan" resources too.
		// FIXME: This behavior isn't totally correct because it's trying
		// to install all of the providers across all configurations into a
		// single directory, but it's not taking any steps to ensure that
		// the plugins installed for one can't impact the selection of providers
		// for another. However, we're forcing the upgrade flag to true here
		// to mostly mitigate the problem by effectively asking the installer
		// to ignore anything already installed and always try to find the
		// newest available version meeting the constraints. We'll need to
		// think through what the best behavior is if we move forward with this
		// design: should we continue to pool providers in a single directory,
		// or have a separate directory per config root, or something else?
		_, providerDiags := c.getProviders(earlyConfig, nil, true)
		diags = diags.Append(providerDiags)
		if providerDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	c.Ui.Output(c.Colorize().Color(strings.TrimSpace(outputInitSuccess)))
	if !c.RunningInAutomation {
		// If we're not running in an automation wrapper, give the user
		// some more detailed next steps that are appropriate for interactive
		// shell usage.
		c.Ui.Output(c.Colorize().Color(strings.TrimSpace(outputInitSuccessCLI)))
	}

	return 0
}

func (c *InitCommand) getModules(path string, earlyRoot *tfconfig.Module, upgrade bool) (output bool, diags tfdiags.Diagnostics) {
	if len(earlyRoot.ModuleCalls) == 0 {
		// Nothing to do
		return false, nil
	}

	if upgrade {
		c.Ui.Output(c.Colorize().Color(fmt.Sprintf("[reset][bold]Upgrading modules...")))
	} else {
		c.Ui.Output(c.Colorize().Color(fmt.Sprintf("[reset][bold]Initializing modules...")))
	}

	hooks := uiModuleInstallHooks{
		Ui:             c.Ui,
		ShowLocalPaths: true,
	}
	instDiags := c.installModules(path, upgrade, hooks)
	diags = diags.Append(instDiags)

	// Since module installer has modified the module manifest on disk, we need
	// to refresh the cache of it in the loader.
	if c.configLoader != nil {
		if err := c.configLoader.RefreshModules(); err != nil {
			// Should never happen
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read module manifest",
				fmt.Sprintf("After installing modules, Terraform could not re-read the manifest of installed modules. This is a bug in Terraform. %s.", err),
			))
		}
	}

	return true, diags
}

// Load the complete module tree, and fetch any missing providers.
// This method outputs its own Ui.
func (c *InitCommand) getProviders(earlyConfig *earlyconfig.Config, state *states.State, upgrade bool) (output bool, diags tfdiags.Diagnostics) {
	var available discovery.PluginMetaSet
	if upgrade {
		// If we're in upgrade mode, we ignore any auto-installed plugins
		// in "available", causing us to reinstall and possibly upgrade them.
		available = c.providerPluginManuallyInstalledSet()
	} else {
		available = c.providerPluginSet()
	}

	configDeps, depsDiags := earlyConfig.ProviderDependencies()
	diags = diags.Append(depsDiags)
	if depsDiags.HasErrors() {
		return false, diags
	}

	configReqs := configDeps.AllPluginRequirements()
	// FIXME: This is weird because ConfigTreeDependencies was written before
	// we switched over to using earlyConfig as the main source of dependencies.
	// In future we should clean this up to be a more reasoable API.
	stateReqs := terraform.ConfigTreeDependencies(nil, state).AllPluginRequirements()

	requirements := configReqs.Merge(stateReqs)
	if len(requirements) == 0 {
		// nothing to initialize
		return false, nil
	}

	c.Ui.Output(c.Colorize().Color(
		"\n[reset][bold]Initializing provider plugins...",
	))

	missing := c.missingPlugins(available, requirements)

	if c.getPlugins {
		if len(missing) > 0 {
			c.Ui.Output("- Checking for available provider plugins...")
		}

		for provider, reqd := range missing {
			pty := addrs.ProviderType{Name: provider}
			_, providerDiags, err := c.providerInstaller.Get(pty, reqd.Versions)
			diags = diags.Append(providerDiags)

			if err != nil {
				constraint := reqd.Versions.String()
				if constraint == "" {
					constraint = "(any version)"
				}

				switch {
				case err == discovery.ErrorServiceUnreachable, err == discovery.ErrorPublicRegistryUnreachable:
					c.Ui.Error(errDiscoveryServiceUnreachable)
				case err == discovery.ErrorNoSuchProvider:
					c.Ui.Error(fmt.Sprintf(errProviderNotFound, provider, DefaultPluginVendorDir))
				case err == discovery.ErrorNoSuitableVersion:
					if reqd.Versions.Unconstrained() {
						// This should never happen, but might crop up if we catch
						// the releases server in a weird state where the provider's
						// directory is present but does not yet contain any
						// versions. We'll treat it like ErrorNoSuchProvider, then.
						c.Ui.Error(fmt.Sprintf(errProviderNotFound, provider, DefaultPluginVendorDir))
					} else {
						c.Ui.Error(fmt.Sprintf(errProviderVersionsUnsuitable, provider, reqd.Versions))
					}
				case errwrap.Contains(err, discovery.ErrorVersionIncompatible.Error()):
					// Attempt to fetch nested error to display to the user which versions
					// we considered and which versions might be compatible. Otherwise,
					// we'll just display a generic version incompatible msg
					incompatErr := errwrap.GetType(err, fmt.Errorf(""))
					if incompatErr != nil {
						c.Ui.Error(incompatErr.Error())
					} else {
						// Generic version incompatible msg
						c.Ui.Error(fmt.Sprintf(errProviderIncompatible, provider, constraint))
					}
					// Reset nested errors
					err = discovery.ErrorVersionIncompatible
				case err == discovery.ErrorNoVersionCompatible:
					// Generic version incompatible msg
					c.Ui.Error(fmt.Sprintf(errProviderIncompatible, provider, constraint))
				case err == discovery.ErrorSignatureVerification:
					c.Ui.Error(fmt.Sprintf(errSignatureVerification, provider))
				case err == discovery.ErrorChecksumVerification,
					err == discovery.ErrorMissingChecksumVerification:
					c.Ui.Error(fmt.Sprintf(errChecksumVerification, provider))
				default:
					c.Ui.Error(fmt.Sprintf(errProviderInstallError, provider, err.Error(), DefaultPluginVendorDir))
				}

				diags = diags.Append(err)
			}
		}

		if diags.HasErrors() {
			return true, diags
		}
	} else if len(missing) > 0 {
		// we have missing providers, but aren't going to try and download them
		var lines []string
		for provider, reqd := range missing {
			if reqd.Versions.Unconstrained() {
				lines = append(lines, fmt.Sprintf("* %s (any version)\n", provider))
			} else {
				lines = append(lines, fmt.Sprintf("* %s (%s)\n", provider, reqd.Versions))
			}
			diags = diags.Append(fmt.Errorf("missing provider %q", provider))
		}
		sort.Strings(lines)
		c.Ui.Error(fmt.Sprintf(errMissingProvidersNoInstall, strings.Join(lines, ""), DefaultPluginVendorDir))
		return true, diags
	}

	// With all the providers downloaded, we'll generate our lock file
	// that ensures the provider binaries remain unchanged until we init
	// again. If anything changes, other commands that use providers will
	// fail with an error instructing the user to re-run this command.
	available = c.providerPluginSet() // re-discover to see newly-installed plugins

	// internal providers were already filtered out, since we don't need to get them.
	chosen := choosePlugins(available, nil, requirements)

	digests := map[string][]byte{}
	for name, meta := range chosen {
		digest, err := meta.SHA256()
		if err != nil {
			diags = diags.Append(fmt.Errorf("Failed to read provider plugin %s: %s", meta.Path, err))
			return true, diags
		}
		digests[name] = digest
		if c.ignorePluginChecksum {
			digests[name] = nil
		}
	}
	err := c.providerPluginsLock().Write(digests)
	if err != nil {
		diags = diags.Append(fmt.Errorf("failed to save provider manifest: %s", err))
		return true, diags
	}

	{
		// Purge any auto-installed plugins that aren't being used.
		purged, err := c.providerInstaller.PurgeUnused(chosen)
		if err != nil {
			// Failure to purge old plugins is not a fatal error
			c.Ui.Warn(fmt.Sprintf("failed to purge unused plugins: %s", err))
		}
		if purged != nil {
			for meta := range purged {
				log.Printf("[DEBUG] Purged unused %s plugin %s", meta.Name, meta.Path)
			}
		}
	}

	// If any providers have "floating" versions (completely unconstrained)
	// we'll suggest the user constrain with a pessimistic constraint to
	// avoid implicitly adopting a later major release.
	constraintSuggestions := make(map[string]discovery.ConstraintStr)
	for name, meta := range chosen {
		req := requirements[name]
		if req == nil {
			// should never happen, but we don't want to crash here, so we'll
			// be cautious.
			continue
		}

		if req.Versions.Unconstrained() && meta.Version != discovery.VersionZero {
			// meta.Version.MustParse is safe here because our "chosen" metas
			// were already filtered for validity of versions.
			constraintSuggestions[name] = meta.Version.MustParse().MinorUpgradeConstraintStr()
		}
	}
	if len(constraintSuggestions) != 0 {
		names := make([]string, 0, len(constraintSuggestions))
		for name := range constraintSuggestions {
			names = append(names, name)
		}
		sort.Strings(names)

		c.Ui.Output(outputInitProvidersUnconstrained)
		for _, name := range names {
			c.Ui.Output(fmt.Sprintf("* provider.%s: version = %q", name, constraintSuggestions[name]))
		}
	}

	return true, diags
}

func (c *InitCommand) AutocompleteArgs() complete.Predictor {
	return complete.PredictDirs("")
}

func (c *InitCommand) AutocompleteFlags() complete.Flags {
	return complete.Flags{
		"-backend":        completePredictBoolean,
		"-backend-config": complete.PredictFiles("*.tfvars"), // can also be key=value, but we can't "predict" that
		"-force-copy":     complete.PredictNothing,
		"-from-module":    completePredictModuleSource,
		"-get":            completePredictBoolean,
		"-get-plugins":    completePredictBoolean,
		"-input":          completePredictBoolean,
		"-lock":           completePredictBoolean,
		"-lock-timeout":   complete.PredictAnything,
		"-no-color":       complete.PredictNothing,
		"-plugin-dir":     complete.PredictDirs(""),
		"-reconfigure":    complete.PredictNothing,
		"-upgrade":        completePredictBoolean,
		"-verify-plugins": completePredictBoolean,
	}
}

// initNewProject creates an initial project configuration file in the given
// directory and then returns the project object that results from it.
//
// TODO: This should also recognize when it's initializing a working directory
// previously used with older versions of Terraform and migrate the existing
// backend configuration and workspace into the project configuration file.
func (c *InitCommand) initNewProject() (*projects.Project, tfdiags.Diagnostics) {
	err := ioutil.WriteFile(projects.ProjectConfigFilenameNative, []byte(`
workspace "default" {
  config = "."
}
`), os.ModePerm)

	var diags tfdiags.Diagnostics
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to initialize project",
			fmt.Sprintf("Could not create project configuration file: %s.", err),
		))
		return nil, diags
	}

	return c.findCurrentProject()
}

func (c *InitCommand) Help() string {
	helpText := `
Usage: terraform init [options] [DIR]

  Initialize a new or existing Terraform working directory by creating
  initial files, loading any remote state, downloading modules, etc.

  This is the first command that should be run for any new or existing
  Terraform configuration per machine. This sets up all the local data
  necessary to run Terraform that is typically not committed to version
  control.

  This command is always safe to run multiple times. Though subsequent runs
  may give errors, this command will never delete your configuration or
  state. Even so, if you have important information, please back it up prior
  to running this command, just in case.

  If no arguments are given, the configuration in this working directory
  is initialized.

Options:

  -backend=true        Configure the backend for this configuration.

  -backend-config=path This can be either a path to an HCL file with key/value
                       assignments (same format as terraform.tfvars) or a
                       'key=value' format. This is merged with what is in the
                       configuration file. This can be specified multiple
                       times. The backend type must be in the configuration
                       itself.

  -force-copy          Suppress prompts about copying state data. This is
                       equivalent to providing a "yes" to all confirmation
                       prompts.

  -from-module=SOURCE  Copy the contents of the given module into the target
                       directory before initialization.

  -get=true            Download any modules for this configuration.

  -get-plugins=true    Download any missing plugins for this configuration.

  -input=true          Ask for input if necessary. If false, will error if
                       input was required.

  -lock=true           Lock the state file when locking is supported.

  -lock-timeout=0s     Duration to retry a state lock.

  -no-color            If specified, output won't contain any color.

  -plugin-dir          Directory containing plugin binaries. This overrides all
                       default search paths for plugins, and prevents the 
                       automatic installation of plugins. This flag can be used
                       multiple times.

  -reconfigure         Reconfigure the backend, ignoring any saved
                       configuration.

  -upgrade=false       If installing modules (-get) or plugins (-get-plugins),
                       ignore previously-downloaded objects and install the
                       latest version allowed within configured constraints.

  -verify-plugins=true Verify the authenticity and integrity of automatically
                       downloaded plugins.
`
	return strings.TrimSpace(helpText)
}

func (c *InitCommand) Synopsis() string {
	return "Initialize a Terraform working directory"
}

const errInitConfigError = `
There are some problems with the configuration, described below.

The Terraform configuration must be valid before initialization so that
Terraform can determine which modules and providers need to be installed.
`

const errInitCopyNotEmpty = `
The working directory already contains files. The -from-module option requires
an empty directory into which a copy of the referenced module will be placed.

To initialize the configuration already in this working directory, omit the
-from-module option.
`

const outputInitEmpty = `
[reset][bold]Terraform initialized in an empty directory![reset]

The directory has no Terraform configuration files. You may begin working
with Terraform immediately by creating Terraform configuration files.
`

const outputInitSuccess = `
[reset][bold][green]Terraform has been successfully initialized![reset][green]
`

const outputInitSuccessCLI = `[reset][green]
You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
`

const outputInitSuccessConfigUpgrade = `
[reset][bold]Terraform has initialized, but configuration upgrades may be needed.[reset]

Terraform found syntax errors in the configuration that prevented full
initialization. If you've recently upgraded to Terraform v0.12, this may be
because your configuration uses syntax constructs that are no longer valid,
and so must be updated before full initialization is possible.

Run terraform init for this configuration at a shell prompt for more information
on how to update it for Terraform v0.12 compatibility.
`

const outputInitSuccessConfigUpgradeCLI = `[reset][green]
[reset][bold]Terraform has initialized, but configuration upgrades may be needed.[reset]

Terraform found syntax errors in the configuration that prevented full
initialization. If you've recently upgraded to Terraform v0.12, this may be
because your configuration uses syntax constructs that are no longer valid,
and so must be updated before full initialization is possible.

Terraform has installed the required providers to support the configuration
upgrade process. To begin upgrading your configuration, run the following:
    terraform 0.12upgrade

To see the full set of errors that led to this message, run:
    terraform validate
`

const outputInitProvidersUnconstrained = `
The following providers do not have any version constraints in configuration,
so the latest version was installed.

To prevent automatic upgrades to new major versions that may contain breaking
changes, it is recommended to add version = "..." constraints to the
corresponding provider blocks in configuration, with the constraint strings
suggested below.
`

const errDiscoveryServiceUnreachable = `
[reset][bold][red]Registry service unreachable.[reset][red]

This may indicate a network issue, or an issue with the requested Terraform Registry.
`

const errProviderNotFound = `
[reset][bold][red]Provider %[1]q not available for installation.[reset][red]

A provider named %[1]q could not be found in the Terraform Registry.

This may result from mistyping the provider name, or the given provider may
be a third-party provider that cannot be installed automatically.

In the latter case, the plugin must be installed manually by locating and
downloading a suitable distribution package and placing the plugin's executable
file in the following directory:
    %[2]s

Terraform detects necessary plugins by inspecting the configuration and state.
To view the provider versions requested by each module, run
"terraform providers".
`

const errProviderVersionsUnsuitable = `
[reset][bold][red]No provider %[1]q plugins meet the constraint %[2]q.[reset][red]

The version constraint is derived from the "version" argument within the
provider %[1]q block in configuration. Child modules may also apply
provider version constraints. To view the provider versions requested by each
module in the current configuration, run "terraform providers".

To proceed, the version constraints for this provider must be relaxed by
either adjusting or removing the "version" argument in the provider blocks
throughout the configuration.
`

const errProviderIncompatible = `
[reset][bold][red]No available provider %[1]q plugins are compatible with this Terraform version.[reset][red]

From time to time, new Terraform major releases can change the requirements for
plugins such that older plugins become incompatible.

Terraform checked all of the plugin versions matching the given constraint:
    %[2]s

Unfortunately, none of the suitable versions are compatible with this version
of Terraform. If you have recently upgraded Terraform, it may be necessary to
move to a newer major release of this provider. Alternatively, if you are
attempting to upgrade the provider to a new major version you may need to
also upgrade Terraform to support the new version.

Consult the documentation for this provider for more information on
compatibility between provider versions and Terraform versions.
`

const errProviderInstallError = `
[reset][bold][red]Error installing provider %[1]q: %[2]s.[reset][red]

Terraform analyses the configuration and state and automatically downloads
plugins for the providers used. However, when attempting to download this
plugin an unexpected error occurred.

This may be caused if for some reason Terraform is unable to reach the
plugin repository. The repository may be unreachable if access is blocked
by a firewall.

If automatic installation is not possible or desirable in your environment,
you may alternatively manually install plugins by downloading a suitable
distribution package and placing the plugin's executable file in the
following directory:
    %[3]s
`

const errMissingProvidersNoInstall = `
[reset][bold][red]Missing required providers.[reset][red]

The following provider constraints are not met by the currently-installed
provider plugins:

%[1]s
Terraform can automatically download and install plugins to meet the given
constraints, but this step was skipped due to the use of -get-plugins=false
and/or -plugin-dir on the command line.

If automatic installation is not possible or desirable in your environment,
you may manually install plugins by downloading a suitable distribution package
and placing the plugin's executable file in one of the directories given in
by -plugin-dir on the command line, or in the following directory if custom
plugin directories are not set:
    %[2]s
`

const errChecksumVerification = `
[reset][bold][red]Error verifying checksum for provider %[1]q[reset][red]
The checksum for provider distribution from the Terraform Registry
did not match the source. This may mean that the distributed files
were changed after this version was released to the Registry.
`

const errSignatureVerification = `
[reset][bold][red]Error verifying GPG signature for provider %[1]q[reset][red]
Terraform was unable to verify the GPG signature of the downloaded provider
files using the keys downloaded from the Terraform Registry. This may mean that
the publisher of the provider removed the key it was signed with, or that the
distributed files were changed after this version was released.
`
