package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/go-plugin"
	"github.com/hashicorp/terraform/command/format"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-shellwords"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/panicwrap"
	"github.com/mitchellh/prefixedio"
)

const (
	// EnvCLI is the environment variable name to set additional CLI args.
	EnvCLI = "TF_CLI_ARGS"
)

func main() {
	// Override global prefix set by go-dynect during init()
	log.SetPrefix("")
	os.Exit(realMain())
}

func realMain() int {
	var wrapConfig panicwrap.WrapConfig

	// don't re-exec terraform as a child process for easier debugging
	if os.Getenv("TF_FORK") == "0" {
		return wrappedMain()
	}

	if !panicwrap.Wrapped(&wrapConfig) {
		// Determine where logs should go in general (requested by the user)
		logWriter, err := logging.LogOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't setup log output: %s", err)
			return 1
		}

		// We always send logs to a temporary file that we use in case
		// there is a panic. Otherwise, we delete it.
		logTempFile, err := ioutil.TempFile("", "terraform-log")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't setup logging tempfile: %s", err)
			return 1
		}
		defer os.Remove(logTempFile.Name())
		defer logTempFile.Close()

		// Setup the prefixed readers that send data properly to
		// stdout/stderr.
		doneCh := make(chan struct{})
		outR, outW := io.Pipe()
		go copyOutput(outR, doneCh)

		// Create the configuration for panicwrap and wrap our executable
		wrapConfig.Handler = panicHandler(logTempFile)
		wrapConfig.Writer = io.MultiWriter(logTempFile, logWriter)
		wrapConfig.Stdout = outW
		wrapConfig.IgnoreSignals = ignoreSignals
		wrapConfig.ForwardSignals = forwardSignals
		exitStatus, err := panicwrap.Wrap(&wrapConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't start Terraform: %s", err)
			return 1
		}

		// If >= 0, we're the parent, so just exit
		if exitStatus >= 0 {
			// Close the stdout writer so that our copy process can finish
			outW.Close()

			// Wait for the output copying to finish
			<-doneCh

			return exitStatus
		}

		// We're the child, so just close the tempfile we made in order to
		// save file handles since the tempfile is only used by the parent.
		logTempFile.Close()
	}

	// Call the real main
	return wrappedMain()
}

func init() {
	Ui = &cli.PrefixedUi{
		AskPrefix:    OutputPrefix,
		OutputPrefix: OutputPrefix,
		InfoPrefix:   OutputPrefix,
		ErrorPrefix:  ErrorPrefix,
		Ui:           &cli.BasicUi{Writer: os.Stdout},
	}
}

func wrappedMain() int {
	// We always need to close the DebugInfo before we exit.
	defer terraform.CloseDebugInfo()

	log.SetOutput(os.Stderr)
	log.Printf(
		"[INFO] Terraform version: %s %s %s",
		Version, VersionPrerelease, GitCommit)
	log.Printf("[INFO] Go runtime version: %s", runtime.Version())
	log.Printf("[INFO] CLI args: %#v", os.Args)

	// Load the configuration
	config := BuiltinConfig

	// Load the configuration file if we have one, that can be used to
	// define extra providers and provisioners.
	clicfgFile, err := cliConfigFile()
	if err != nil {
		Ui.Error(fmt.Sprintf("Error loading CLI configuration: \n\n%s", err))
		return 1
	}

	if clicfgFile != "" {
		usrcfg, err := LoadConfig(clicfgFile)
		if err != nil {
			Ui.Error(fmt.Sprintf("Error loading CLI configuration: \n\n%s", err))
			return 1
		}

		config = *config.Merge(usrcfg)
	}

	if envConfig := EnvConfig(); envConfig != nil {
		// envConfig takes precedence
		config = *envConfig.Merge(&config)
	}

	log.Printf("[DEBUG] CLI Config is %#v", config)

	{
		var diags tfdiags.Diagnostics
		diags = diags.Append(config.Validate())
		if len(diags) > 0 {
			Ui.Error("There are some problems with the CLI configuration:")
			for _, diag := range diags {
				earlyColor := &colorstring.Colorize{
					Colors:  colorstring.DefaultColors,
					Disable: true, // Disable color to be conservative until we know better
					Reset:   true,
				}
				Ui.Error(format.Diagnostic(diag, earlyColor, 78))
			}
			if diags.HasErrors() {
				Ui.Error("As a result of the above problems, Terraform may not behave as intended.\n\n")
			}
		}
	}

	// In tests, Commands may already be set to provide mock commands
	if Commands == nil {
		initCommands(&config)
	}

	// Run checkpoint
	go runCheckpoint(&config)

	// Make sure we clean up any managed plugins at the end of this
	defer plugin.CleanupClients()

	// Get the command line args.
	binName := filepath.Base(os.Args[0])
	args := os.Args[1:]

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

	// Pass in the overriding plugin paths from config
	PluginOverrides.Providers = config.Providers
	PluginOverrides.Provisioners = config.Provisioners

	exitCode, err := cliRunner.Run()
	if err != nil {
		Ui.Error(fmt.Sprintf("Error executing CLI: %s", err.Error()))
		return 1
	}

	return exitCode
}

func cliConfigFile() (string, error) {
	mustExist := true
	configFilePath := os.Getenv("TERRAFORM_CONFIG")
	if configFilePath == "" {
		var err error
		configFilePath, err = ConfigFile()
		mustExist = false

		if err != nil {
			log.Printf(
				"[ERROR] Error detecting default CLI config file path: %s",
				err)
		}
	}

	log.Printf("[DEBUG] Attempting to open CLI config file: %s", configFilePath)
	f, err := os.Open(configFilePath)
	if err == nil {
		f.Close()
		return configFilePath, nil
	}

	if mustExist || !os.IsNotExist(err) {
		return "", err
	}

	log.Println("[DEBUG] File doesn't exist, but doesn't need to. Ignoring.")
	return "", nil
}

// copyOutput uses output prefixes to determine whether data on stdout
// should go to stdout or stderr. This is due to panicwrap using stderr
// as the log and error channel.
func copyOutput(r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)

	pr, err := prefixedio.NewReader(r)
	if err != nil {
		panic(err)
	}

	stderrR, err := pr.Prefix(ErrorPrefix)
	if err != nil {
		panic(err)
	}
	stdoutR, err := pr.Prefix(OutputPrefix)
	if err != nil {
		panic(err)
	}
	defaultR, err := pr.Prefix("")
	if err != nil {
		panic(err)
	}

	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr

	if runtime.GOOS == "windows" {
		stdout = colorable.NewColorableStdout()
		stderr = colorable.NewColorableStderr()

		// colorable is not concurrency-safe when stdout and stderr are the
		// same console, so we need to add some synchronization to ensure that
		// we can't be concurrently writing to both stderr and stdout at
		// once, or else we get intermingled writes that create gibberish
		// in the console.
		wrapped := synchronizedWriters(stdout, stderr)
		stdout = wrapped[0]
		stderr = wrapped[1]
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		io.Copy(stderr, stderrR)
	}()
	go func() {
		defer wg.Done()
		io.Copy(stdout, stdoutR)
	}()
	go func() {
		defer wg.Done()
		io.Copy(stdout, defaultR)
	}()

	wg.Wait()
}

func mergeEnvArgs(envName string, cmd string, args []string) ([]string, error) {
	v := os.Getenv(envName)
	if v == "" {
		return args, nil
	}

	log.Printf("[INFO] %s value: %q", envName, v)
	extra, err := shellwords.Parse(v)
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
