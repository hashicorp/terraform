// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/apparentlymart/go-shquot/shquot"
	"github.com/hashicorp/cli"
	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/cliconfig"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/didyoumean"
	"github.com/hashicorp/terraform/internal/httpclient"
	"github.com/hashicorp/terraform/internal/logging"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/version"
	"github.com/mattn/go-shellwords"
	"github.com/mitchellh/colorstring"
	"go.opentelemetry.io/otel/trace"

	backendInit "github.com/hashicorp/terraform/internal/backend/init"
)

const (
	// EnvCLI is the environment variable name to set additional CLI args.
	EnvCLI = "TF_CLI_ARGS"

	// The parent process will create a file to collect crash logs
	envTmpLogPath = "TF_TEMP_LOG_PATH"
)

// ui wraps the primary output cli.Ui, and redirects Warn calls to Output
// calls. This ensures that warnings are sent to stdout, and are properly
// serialized within the stdout stream.
type ui struct {
	cli.Ui
}

func (u *ui) Warn(msg string) {
	u.Ui.Output(msg)
}

func init() {
	Ui = &ui{&cli.BasicUi{
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
		Reader:      os.Stdin,
	}}
}

func main() {
	os.Exit(realMain())
}

func realMain() int {
	defer logging.PanicHandler()

	var err error

	err = openTelemetryInit()
	if err != nil {
		// openTelemetryInit can only fail if Terraform was run with an
		// explicit environment variable to enable telemetry collection,
		// so in typical use we cannot get here.
		Ui.Error(fmt.Sprintf("Could not initialize telemetry: %s", err))
		Ui.Error(fmt.Sprintf("Unset environment variable %s if you don't intend to collect telemetry from Terraform.", openTelemetryExporterEnvVar))
		return 1
	}
	var ctx context.Context
	var otelSpan trace.Span
	{
		// At minimum we emit a span covering the entire command execution.
		_, displayArgs := shquot.POSIXShellSplit(os.Args)
		ctx, otelSpan = tracer.Start(context.Background(), fmt.Sprintf("terraform %s", displayArgs))
		defer otelSpan.End()
	}

	tmpLogPath := os.Getenv(envTmpLogPath)
	if tmpLogPath != "" {
		f, err := os.OpenFile(tmpLogPath, os.O_RDWR|os.O_APPEND, 0666)
		if err == nil {
			defer f.Close()

			log.Printf("[DEBUG] Adding temp file log sink: %s", f.Name())
			logging.RegisterSink(f)
		} else {
			log.Printf("[ERROR] Could not open temp log file: %v", err)
		}
	}

	log.Printf(
		"[INFO] Terraform version: %s %s",
		Version, VersionPrerelease)
	for _, depMod := range version.InterestingDependencies() {
		log.Printf("[DEBUG] using %s %s", depMod.Path, depMod.Version)
	}
	log.Printf("[INFO] Go runtime version: %s", runtime.Version())
	log.Printf("[INFO] CLI args: %#v", os.Args)
	if ExperimentsAllowed() {
		log.Printf("[INFO] This build of Terraform allows using experimental features")
	}

	streams, err := terminal.Init()
	if err != nil {
		Ui.Error(fmt.Sprintf("Failed to configure the terminal: %s", err))
		return 1
	}
	if streams.Stdout.IsTerminal() {
		log.Printf("[TRACE] Stdout is a terminal of width %d", streams.Stdout.Columns())
	} else {
		log.Printf("[TRACE] Stdout is not a terminal")
	}
	if streams.Stderr.IsTerminal() {
		log.Printf("[TRACE] Stderr is a terminal of width %d", streams.Stderr.Columns())
	} else {
		log.Printf("[TRACE] Stderr is not a terminal")
	}
	if streams.Stdin.IsTerminal() {
		log.Printf("[TRACE] Stdin is a terminal")
	} else {
		log.Printf("[TRACE] Stdin is not a terminal")
	}

	// NOTE: We're intentionally calling LoadConfig _before_ handling a possible
	// -chdir=... option on the command line, so that a possible relative
	// path in the TERRAFORM_CONFIG_FILE environment variable (though probably
	// ill-advised) will be resolved relative to the true working directory,
	// not the overridden one.
	config, diags := cliconfig.LoadConfig()

	if len(diags) > 0 {
		// Since we haven't instantiated a command.Meta yet, we need to do
		// some things manually here and use some "safe" defaults for things
		// that command.Meta could otherwise figure out in smarter ways.
		Ui.Error("There are some problems with the CLI configuration:")
		for _, diag := range diags {
			earlyColor := &colorstring.Colorize{
				Colors:  colorstring.DefaultColors,
				Disable: true, // Disable color to be conservative until we know better
				Reset:   true,
			}
			// We don't currently have access to the source code cache for
			// the parser used to load the CLI config, so we can't show
			// source code snippets in early diagnostics.
			Ui.Error(format.Diagnostic(diag, nil, earlyColor, 78))
		}
		if diags.HasErrors() {
			Ui.Error("As a result of the above problems, Terraform may not behave as intended.\n\n")
			// We continue to run anyway, since Terraform has reasonable defaults.
		}
	}

	// Get any configured credentials from the config and initialize
	// a service discovery object. The slightly awkward predeclaration of
	// disco is required to allow us to pass untyped nil as the creds source
	// when creating the source fails. Otherwise we pass a typed nil which
	// breaks the nil checks in the disco object
	var services *disco.Disco
	credsSrc, err := credentialsSource(config)
	if err == nil {
		services = disco.NewWithCredentialsSource(credsSrc)
	} else {
		// Most commands don't actually need credentials, and most situations
		// that would get us here would already have been reported by the config
		// loading above, so we'll just log this one as an aid to debugging
		// in the unlikely event that it _does_ arise.
		log.Printf("[WARN] Cannot initialize remote host credentials manager: %s", err)
		// passing (untyped) nil as the creds source is okay because the disco
		// object checks that and just acts as though no credentials are present.
		services = disco.NewWithCredentialsSource(nil)
	}
	services.SetUserAgent(httpclient.TerraformUserAgent(version.String()))

	// Get the command line args.
	binName := filepath.Base(os.Args[0])
	args := os.Args[1:]

	originalWd, err := os.Getwd()
	if err != nil {
		// It would be very strange to end up here
		Ui.Error(fmt.Sprintf("Failed to determine current working directory: %s", err))
		return 1
	}

	// The arguments can begin with a -chdir option to ask Terraform to switch
	// to a different working directory for the rest of its work. If that
	// option is present then extractChdirOption returns a trimmed args with that option removed.
	overrideWd, args, err := extractChdirOption(args)
	if err != nil {
		Ui.Error(fmt.Sprintf("Invalid -chdir option: %s", err))
		return 1
	}
	if overrideWd != "" {
		err := os.Chdir(overrideWd)
		if err != nil {
			Ui.Error(fmt.Sprintf("Error handling -chdir option: %s", err))
			return 1
		}
	}

	providerSrc, diags := providerSource(config.ProviderInstallation, services)
	if len(diags) > 0 {
		Ui.Error("There are some problems with the provider_installation configuration:")
		for _, diag := range diags {
			earlyColor := &colorstring.Colorize{
				Colors:  colorstring.DefaultColors,
				Disable: true, // Disable color to be conservative until we know better
				Reset:   true,
			}
			Ui.Error(format.Diagnostic(diag, nil, earlyColor, 78))
		}
		if diags.HasErrors() {
			Ui.Error("As a result of the above problems, Terraform's provider installer may not behave as intended.\n\n")
			// We continue to run anyway, because most commands don't do provider installation.
		}
	}
	providerDevOverrides := providerDevOverrides(config.ProviderInstallation)

	// The user can declare that certain providers are being managed on
	// Terraform's behalf using this environment variable. This is used
	// primarily by the SDK's acceptance testing framework.
	unmanagedProviders, err := parseReattachProviders(os.Getenv("TF_REATTACH_PROVIDERS"))
	if err != nil {
		Ui.Error(err.Error())
		return 1
	}

	// Initialize the backends.
	backendInit.Init(services)

	// In tests, Commands may already be set to provide mock commands
	if Commands == nil {
		// Commands get to hold on to the original working directory here,
		// in case they need to refer back to it for any special reason, though
		// they should primarily be working with the override working directory
		// that we've now switched to above.
		initCommands(ctx, originalWd, streams, config, services, providerSrc, providerDevOverrides, unmanagedProviders)
	}

	// Run checkpoint
	go runCheckpoint(ctx, config)

	// Make sure we clean up any managed plugins at the end of this
	defer plugin.CleanupClients()

	// Build the CLI so far, we do this so we can query the subcommand.
	cliRunner := &cli.CLI{
		Args:       args,
		Commands:   Commands,
		HelpFunc:   helpFunc,
		HelpWriter: os.Stdout,
	}

	// Prefix the args with any args from the EnvCLI
	args, err = mergeEnvArgs(EnvCLI, cliRunner.Subcommand(), args)
	if err != nil {
		Ui.Error(err.Error())
		return 1
	}

	// Prefix the args with any args from the EnvCLI targeting this command
	suffix := strings.Replace(strings.Replace(
		cliRunner.Subcommand(), "-", "_", -1), " ", "_", -1)
	args, err = mergeEnvArgs(
		fmt.Sprintf("%s_%s", EnvCLI, suffix), cliRunner.Subcommand(), args)
	if err != nil {
		Ui.Error(err.Error())
		return 1
	}

	// We shortcut "--version" and "-v" to just show the version
	for _, arg := range args {
		if arg == "-v" || arg == "-version" || arg == "--version" {
			newArgs := make([]string, len(args)+1)
			newArgs[0] = "version"
			copy(newArgs[1:], args)
			args = newArgs
			break
		}
	}

	// Rebuild the CLI with any modified args.
	log.Printf("[INFO] CLI command args: %#v", args)
	cliRunner = &cli.CLI{
		Name:       binName,
		Args:       args,
		Commands:   Commands,
		HelpFunc:   helpFunc,
		HelpWriter: os.Stdout,

		Autocomplete:          true,
		AutocompleteInstall:   "install-autocomplete",
		AutocompleteUninstall: "uninstall-autocomplete",
	}

	// Before we continue we'll check whether the requested command is
	// actually known. If not, we might be able to suggest an alternative
	// if it seems like the user made a typo.
	// (This bypasses the built-in help handling in cli.CLI for the situation
	// where a command isn't found, because it's likely more helpful to
	// mention what specifically went wrong, rather than just printing out
	// a big block of usage information.)

	// Check if this is being run via shell auto-complete, which uses the
	// binary name as the first argument and won't be listed as a subcommand.
	autoComplete := os.Getenv("COMP_LINE") != ""

	if cmd := cliRunner.Subcommand(); cmd != "" && !autoComplete {
		// Due to the design of cli.CLI, this special error message only works
		// for typos of top-level commands. For a subcommand typo, like
		// "terraform state posh", cmd would be "state" here and thus would
		// be considered to exist, and it would print out its own usage message.
		if _, exists := Commands[cmd]; !exists {
			suggestions := make([]string, 0, len(Commands))
			for name := range Commands {
				suggestions = append(suggestions, name)
			}
			suggestion := didyoumean.NameSuggestion(cmd, suggestions)
			if suggestion != "" {
				suggestion = fmt.Sprintf(" Did you mean %q?", suggestion)
			}
			fmt.Fprintf(os.Stderr, "Terraform has no command named %q.%s\n\nTo see all of Terraform's top-level commands, run:\n  terraform -help\n\n", cmd, suggestion)
			return 1
		}
	}

	exitCode, err := cliRunner.Run()
	if err != nil {
		Ui.Error(fmt.Sprintf("Error executing CLI: %s", err.Error()))
		return 1
	}

	// if we are exiting with a non-zero code, check if it was caused by any
	// plugins crashing
	if exitCode != 0 {
		for _, panicLog := range logging.PluginPanics() {
			Ui.Error(panicLog)
		}
	}

	return exitCode
}

func mergeEnvArgs(envName string, cmd string, args []string) ([]string, error) {
	v := os.Getenv(envName)
	if v == "" {
		return args, nil
	}

	swParser := &shellwords.Parser{
		ParseEnv:      false,
		ParseBacktick: false,
	}

	log.Printf("[INFO] %s value: %q", envName, v)
	extra, err := swParser.Parse(v)
	if err != nil {
		return nil, fmt.Errorf(
			"Error parsing extra CLI args from %s: %s",
			envName, err)
	}

	// Find the command to look for in the args. If there is a space,
	// we need to find the last part.
	search := cmd
	if idx := strings.LastIndex(search, " "); idx >= 0 {
		search = cmd[idx+1:]
	}

	// Find the index to place the flags. We put them exactly
	// after the first non-flag arg.
	idx := -1
	for i, v := range args {
		if v == search {
			idx = i
			break
		}
	}

	// idx points to the exact arg that isn't a flag. We increment
	// by one so that all the copying below expects idx to be the
	// insertion point.
	idx++

	// Copy the args
	newArgs := make([]string, len(args)+len(extra))
	copy(newArgs, args[:idx])
	copy(newArgs[idx:], extra)
	copy(newArgs[len(extra)+idx:], args[idx:])
	return newArgs, nil
}

// parse information on reattaching to unmanaged providers out of a
// JSON-encoded environment variable.
func parseReattachProviders(in string) (map[addrs.Provider]*plugin.ReattachConfig, error) {
	unmanagedProviders := map[addrs.Provider]*plugin.ReattachConfig{}
	if in != "" {
		type reattachConfig struct {
			Protocol        string
			ProtocolVersion int
			Addr            struct {
				Network string
				String  string
			}
			Pid  int
			Test bool
		}
		var m map[string]reattachConfig
		err := json.Unmarshal([]byte(in), &m)
		if err != nil {
			return unmanagedProviders, fmt.Errorf("Invalid format for TF_REATTACH_PROVIDERS: %w", err)
		}
		for p, c := range m {
			a, diags := addrs.ParseProviderSourceString(p)
			if diags.HasErrors() {
				return unmanagedProviders, fmt.Errorf("Error parsing %q as a provider address: %w", a, diags.Err())
			}
			var addr net.Addr
			switch c.Addr.Network {
			case "unix":
				addr, err = net.ResolveUnixAddr("unix", c.Addr.String)
				if err != nil {
					return unmanagedProviders, fmt.Errorf("Invalid unix socket path %q for %q: %w", c.Addr.String, p, err)
				}
			case "tcp":
				addr, err = net.ResolveTCPAddr("tcp", c.Addr.String)
				if err != nil {
					return unmanagedProviders, fmt.Errorf("Invalid TCP address %q for %q: %w", c.Addr.String, p, err)
				}
			default:
				return unmanagedProviders, fmt.Errorf("Unknown address type %q for %q", c.Addr.Network, p)
			}
			unmanagedProviders[a] = &plugin.ReattachConfig{
				Protocol:        plugin.Protocol(c.Protocol),
				ProtocolVersion: c.ProtocolVersion,
				Pid:             c.Pid,
				Test:            c.Test,
				Addr:            addr,
			}
		}
	}
	return unmanagedProviders, nil
}

func extractChdirOption(args []string) (string, []string, error) {
	if len(args) == 0 {
		return "", args, nil
	}

	const argName = "-chdir"
	const argPrefix = argName + "="
	var argValue string
	var argPos int

	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			// Because the chdir option is a subcommand-agnostic one, we require
			// it to appear before any subcommand argument, so if we find a
			// non-option before we find -chdir then we are finished.
			break
		}
		if arg == argName || arg == argPrefix {
			return "", args, fmt.Errorf("must include an equals sign followed by a directory path, like -chdir=example")
		}
		if strings.HasPrefix(arg, argPrefix) {
			argPos = i
			argValue = arg[len(argPrefix):]
		}
	}

	// When we fall out here, we'll have populated argValue with a non-empty
	// string if the -chdir=... option was present and valid, or left it
	// empty if it wasn't present.
	if argValue == "" {
		return "", args, nil
	}

	// If we did find the option then we'll need to produce a new args that
	// doesn't include it anymore.
	if argPos == 0 {
		// Easy case: we can just slice off the front
		return argValue, args[1:], nil
	}
	// Otherwise we need to construct a new array and copy to it.
	newArgs := make([]string, len(args)-1)
	copy(newArgs, args[:argPos])
	copy(newArgs[argPos:], args[argPos+1:])
	return argValue, newArgs, nil
}
