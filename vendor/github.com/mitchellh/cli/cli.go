package cli

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/armon/go-radix"
	"github.com/posener/complete"
)

// CLI contains the state necessary to run subcommands and parse the
// command line arguments.
//
// CLI also supports nested subcommands, such as "cli foo bar". To use
// nested subcommands, the key in the Commands mapping below contains the
// full subcommand. In this example, it would be "foo bar".
//
// If you use a CLI with nested subcommands, some semantics change due to
// ambiguities:
//
//   * We use longest prefix matching to find a matching subcommand. This
//     means if you register "foo bar" and the user executes "cli foo qux",
//     the "foo" command will be executed with the arg "qux". It is up to
//     you to handle these args. One option is to just return the special
//     help return code `RunResultHelp` to display help and exit.
//
//   * The help flag "-h" or "-help" will look at all args to determine
//     the help function. For example: "otto apps list -h" will show the
//     help for "apps list" but "otto apps -h" will show it for "apps".
//     In the normal CLI, only the first subcommand is used.
//
//   * The help flag will list any subcommands that a command takes
//     as well as the command's help itself. If there are no subcommands,
//     it will note this. If the CLI itself has no subcommands, this entire
//     section is omitted.
//
//   * Any parent commands that don't exist are automatically created as
//     no-op commands that just show help for other subcommands. For example,
//     if you only register "foo bar", then "foo" is automatically created.
//
type CLI struct {
	// Args is the list of command-line arguments received excluding
	// the name of the app. For example, if the command "./cli foo bar"
	// was invoked, then Args should be []string{"foo", "bar"}.
	Args []string

	// Commands is a mapping of subcommand names to a factory function
	// for creating that Command implementation. If there is a command
	// with a blank string "", then it will be used as the default command
	// if no subcommand is specified.
	//
	// If the key has a space in it, this will create a nested subcommand.
	// For example, if the key is "foo bar", then to access it our CLI
	// must be accessed with "./cli foo bar". See the docs for CLI for
	// notes on how this changes some other behavior of the CLI as well.
	Commands map[string]CommandFactory

	// Name defines the name of the CLI.
	Name string

	// Version of the CLI.
	Version string

	// Autocomplete enables or disables subcommand auto-completion support.
	// This is enabled by default when NewCLI is called. Otherwise, this
	// must enabled explicitly.
	//
	// Autocomplete requires the "Name" option to be set on CLI. This name
	// should be set exactly to the binary name that is autocompleted.
	//
	// Autocompletion is supported via the github.com/posener/complete
	// library. This library supports both bash and zsh. To add support
	// for other shells, please see that library.
	//
	// AutocompleteInstall and AutocompleteUninstall are the global flag
	// names for installing and uninstalling the autocompletion handlers
	// for the user's shell. The flag should omit the hyphen(s) in front of
	// the value. Both single and double hyphens will automatically be supported
	// for the flag name. These default to `autocomplete-install` and
	// `autocomplete-uninstall` respectively.
	//
	// AutocompleteNoDefaultFlags is a boolean which controls if the default auto-
	// complete flags like -help and -version are added to the output.
	//
	// AutocompleteGlobalFlags are a mapping of global flags for
	// autocompletion. The help and version flags are automatically added.
	Autocomplete               bool
	AutocompleteInstall        string
	AutocompleteUninstall      string
	AutocompleteNoDefaultFlags bool
	AutocompleteGlobalFlags    complete.Flags
	autocompleteInstaller      autocompleteInstaller // For tests

	// HelpFunc and HelpWriter are used to output help information, if
	// requested.
	//
	// HelpFunc is the function called to generate the generic help
	// text that is shown if help must be shown for the CLI that doesn't
	// pertain to a specific command.
	//
	// HelpWriter is the Writer where the help text is outputted to. If
	// not specified, it will default to Stderr.
	HelpFunc   HelpFunc
	HelpWriter io.Writer

	//---------------------------------------------------------------
	// Internal fields set automatically

	once           sync.Once
	autocomplete   *complete.Complete
	commandTree    *radix.Tree
	commandNested  bool
	subcommand     string
	subcommandArgs []string
	topFlags       []string

	// These are true when special global flags are set. We can/should
	// probably use a bitset for this one day.
	isHelp                  bool
	isVersion               bool
	isAutocompleteInstall   bool
	isAutocompleteUninstall bool
}

// NewClI returns a new CLI instance with sensible defaults.
func NewCLI(app, version string) *CLI {
	return &CLI{
		Name:         app,
		Version:      version,
		HelpFunc:     BasicHelpFunc(app),
		Autocomplete: true,
	}

}

// IsHelp returns whether or not the help flag is present within the
// arguments.
func (c *CLI) IsHelp() bool {
	c.once.Do(c.init)
	return c.isHelp
}

// IsVersion returns whether or not the version flag is present within the
// arguments.
func (c *CLI) IsVersion() bool {
	c.once.Do(c.init)
	return c.isVersion
}

// Run runs the actual CLI based on the arguments given.
func (c *CLI) Run() (int, error) {
	c.once.Do(c.init)

	// If this is a autocompletion request, satisfy it. This must be called
	// first before anything else since its possible to be autocompleting
	// -help or -version or other flags and we want to show completions
	// and not actually write the help or version.
	if c.Autocomplete && c.autocomplete.Complete() {
		return 0, nil
	}

	// Just show the version and exit if instructed.
	if c.IsVersion() && c.Version != "" {
		c.HelpWriter.Write([]byte(c.Version + "\n"))
		return 0, nil
	}

	// Just print the help when only '-h' or '--help' is passed.
	if c.IsHelp() && c.Subcommand() == "" {
		c.HelpWriter.Write([]byte(c.HelpFunc(c.Commands) + "\n"))
		return 0, nil
	}

	// If we're attempting to install or uninstall autocomplete then handle
	if c.Autocomplete {
		// Autocomplete requires the "Name" to be set so that we know what
		// command to setup the autocomplete on.
		if c.Name == "" {
			return 1, fmt.Errorf(
				"internal error: CLI.Name must be specified for autocomplete to work")
		}

		// If both install and uninstall flags are specified, then error
		if c.isAutocompleteInstall && c.isAutocompleteUninstall {
			return 1, fmt.Errorf(
				"Either the autocomplete install or uninstall flag may " +
					"be specified, but not both.")
		}

		// If the install flag is specified, perform the install or uninstall
		if c.isAutocompleteInstall {
			if err := c.autocompleteInstaller.Install(c.Name); err != nil {
				return 1, err
			}

			return 0, nil
		}

		if c.isAutocompleteUninstall {
			if err := c.autocompleteInstaller.Uninstall(c.Name); err != nil {
				return 1, err
			}

			return 0, nil
		}
	}

	// Attempt to get the factory function for creating the command
	// implementation. If the command is invalid or blank, it is an error.
	raw, ok := c.commandTree.Get(c.Subcommand())
	if !ok {
		c.HelpWriter.Write([]byte(c.HelpFunc(c.helpCommands(c.subcommandParent())) + "\n"))
		return 1, nil
	}

	command, err := raw.(CommandFactory)()
	if err != nil {
		return 1, err
	}

	// If we've been instructed to just print the help, then print it
	if c.IsHelp() {
		c.commandHelp(command)
		return 0, nil
	}

	// If there is an invalid flag, then error
	if len(c.topFlags) > 0 {
		c.HelpWriter.Write([]byte(
			"Invalid flags before the subcommand. If these flags are for\n" +
				"the subcommand, please put them after the subcommand.\n\n"))
		c.commandHelp(command)
		return 1, nil
	}

	code := command.Run(c.SubcommandArgs())
	if code == RunResultHelp {
		// Requesting help
		c.commandHelp(command)
		return 1, nil
	}

	return code, nil
}

// Subcommand returns the subcommand that the CLI would execute. For
// example, a CLI from "--version version --help" would return a Subcommand
// of "version"
func (c *CLI) Subcommand() string {
	c.once.Do(c.init)
	return c.subcommand
}

// SubcommandArgs returns the arguments that will be passed to the
// subcommand.
func (c *CLI) SubcommandArgs() []string {
	c.once.Do(c.init)
	return c.subcommandArgs
}

// subcommandParent returns the parent of this subcommand, if there is one.
// If there isn't on, "" is returned.
func (c *CLI) subcommandParent() string {
	// Get the subcommand, if it is "" alread just return
	sub := c.Subcommand()
	if sub == "" {
		return sub
	}

	// Clear any trailing spaces and find the last space
	sub = strings.TrimRight(sub, " ")
	idx := strings.LastIndex(sub, " ")

	if idx == -1 {
		// No space means our parent is root
		return ""
	}

	return sub[:idx]
}

func (c *CLI) init() {
	if c.HelpFunc == nil {
		c.HelpFunc = BasicHelpFunc("app")

		if c.Name != "" {
			c.HelpFunc = BasicHelpFunc(c.Name)
		}
	}

	if c.HelpWriter == nil {
		c.HelpWriter = os.Stderr
	}

	// Build our command tree
	c.commandTree = radix.New()
	c.commandNested = false
	for k, v := range c.Commands {
		k = strings.TrimSpace(k)
		c.commandTree.Insert(k, v)
		if strings.ContainsRune(k, ' ') {
			c.commandNested = true
		}
	}

	// Go through the key and fill in any missing parent commands
	if c.commandNested {
		var walkFn radix.WalkFn
		toInsert := make(map[string]struct{})
		walkFn = func(k string, raw interface{}) bool {
			idx := strings.LastIndex(k, " ")
			if idx == -1 {
				// If there is no space, just ignore top level commands
				return false
			}

			// Trim up to that space so we can get the expected parent
			k = k[:idx]
			if _, ok := c.commandTree.Get(k); ok {
				// Yay we have the parent!
				return false
			}

			// We're missing the parent, so let's insert this
			toInsert[k] = struct{}{}

			// Call the walk function recursively so we check this one too
			return walkFn(k, nil)
		}

		// Walk!
		c.commandTree.Walk(walkFn)

		// Insert any that we're missing
		for k := range toInsert {
			var f CommandFactory = func() (Command, error) {
				return &MockCommand{
					HelpText:  "This command is accessed by using one of the subcommands below.",
					RunResult: RunResultHelp,
				}, nil
			}

			c.commandTree.Insert(k, f)
		}
	}

	// Setup autocomplete if we have it enabled. We have to do this after
	// the command tree is setup so we can use the radix tree to easily find
	// all subcommands.
	if c.Autocomplete {
		c.initAutocomplete()
	}

	// Process the args
	c.processArgs()
}

func (c *CLI) initAutocomplete() {
	if c.AutocompleteInstall == "" {
		c.AutocompleteInstall = defaultAutocompleteInstall
	}

	if c.AutocompleteUninstall == "" {
		c.AutocompleteUninstall = defaultAutocompleteUninstall
	}

	if c.autocompleteInstaller == nil {
		c.autocompleteInstaller = &realAutocompleteInstaller{}
	}

	// Build the root command
	cmd := c.initAutocompleteSub("")

	// For the root, we add the global flags to the "Flags". This way
	// they don't show up on every command.
	if !c.AutocompleteNoDefaultFlags {
		cmd.Flags = map[string]complete.Predictor{
			"-" + c.AutocompleteInstall:   complete.PredictNothing,
			"-" + c.AutocompleteUninstall: complete.PredictNothing,
			"-help":    complete.PredictNothing,
			"-version": complete.PredictNothing,
		}
	}
	cmd.GlobalFlags = c.AutocompleteGlobalFlags

	c.autocomplete = complete.New(c.Name, cmd)
}

// initAutocompleteSub creates the complete.Command for a subcommand with
// the given prefix. This will continue recursively for all subcommands.
// The prefix "" (empty string) can be used for the root command.
func (c *CLI) initAutocompleteSub(prefix string) complete.Command {
	var cmd complete.Command
	walkFn := func(k string, raw interface{}) bool {
		if len(prefix) > 0 {
			// If we have a prefix, trim the prefix + 1 (for the space)
			// Example: turns "sub one" to "one" with prefix "sub"
			k = k[len(prefix)+1:]
		}

		// Keep track of the full key so that we can nest further if necessary
		fullKey := k

		if idx := strings.LastIndex(k, " "); idx >= 0 {
			// If there is a space, we trim up to the space
			k = k[:idx]
		}

		if idx := strings.LastIndex(k, " "); idx >= 0 {
			// This catches the scenario just in case where we see "sub one"
			// before "sub". This will let us properly setup the subcommand
			// regardless.
			k = k[idx+1:]
		}

		if _, ok := cmd.Sub[k]; ok {
			// If we already tracked this subcommand then ignore
			return false
		}

		if cmd.Sub == nil {
			cmd.Sub = complete.Commands(make(map[string]complete.Command))
		}
		subCmd := c.initAutocompleteSub(fullKey)

		// Instantiate the command so that we can check if the command is
		// a CommandAutocomplete implementation. If there is an error
		// creating the command, we just ignore it since that will be caught
		// later.
		impl, err := raw.(CommandFactory)()
		if err != nil {
			impl = nil
		}

		// Check if it implements ComandAutocomplete. If so, setup the autocomplete
		if c, ok := impl.(CommandAutocomplete); ok {
			subCmd.Args = c.AutocompleteArgs()
			subCmd.Flags = c.AutocompleteFlags()
		}

		cmd.Sub[k] = subCmd
		return false
	}

	walkPrefix := prefix
	if walkPrefix != "" {
		walkPrefix += " "
	}

	c.commandTree.WalkPrefix(walkPrefix, walkFn)
	return cmd
}

func (c *CLI) commandHelp(command Command) {
	// Get the template to use
	tpl := strings.TrimSpace(defaultHelpTemplate)
	if t, ok := command.(CommandHelpTemplate); ok {
		tpl = t.HelpTemplate()
	}
	if !strings.HasSuffix(tpl, "\n") {
		tpl += "\n"
	}

	// Parse it
	t, err := template.New("root").Parse(tpl)
	if err != nil {
		t = template.Must(template.New("root").Parse(fmt.Sprintf(
			"Internal error! Failed to parse command help template: %s\n", err)))
	}

	// Template data
	data := map[string]interface{}{
		"Name": c.Name,
		"Help": command.Help(),
	}

	// Build subcommand list if we have it
	var subcommandsTpl []map[string]interface{}
	if c.commandNested {
		// Get the matching keys
		subcommands := c.helpCommands(c.Subcommand())
		keys := make([]string, 0, len(subcommands))
		for k := range subcommands {
			keys = append(keys, k)
		}

		// Sort the keys
		sort.Strings(keys)

		// Figure out the padding length
		var longest int
		for _, k := range keys {
			if v := len(k); v > longest {
				longest = v
			}
		}

		// Go through and create their structures
		subcommandsTpl = make([]map[string]interface{}, 0, len(subcommands))
		for _, k := range keys {
			// Get the command
			raw, ok := subcommands[k]
			if !ok {
				c.HelpWriter.Write([]byte(fmt.Sprintf(
					"Error getting subcommand %q", k)))
			}
			sub, err := raw()
			if err != nil {
				c.HelpWriter.Write([]byte(fmt.Sprintf(
					"Error instantiating %q: %s", k, err)))
			}

			// Find the last space and make sure we only include that last part
			name := k
			if idx := strings.LastIndex(k, " "); idx > -1 {
				name = name[idx+1:]
			}

			subcommandsTpl = append(subcommandsTpl, map[string]interface{}{
				"Name":        name,
				"NameAligned": name + strings.Repeat(" ", longest-len(k)),
				"Help":        sub.Help(),
				"Synopsis":    sub.Synopsis(),
			})
		}
	}
	data["Subcommands"] = subcommandsTpl

	// Write
	err = t.Execute(c.HelpWriter, data)
	if err == nil {
		return
	}

	// An error, just output...
	c.HelpWriter.Write([]byte(fmt.Sprintf(
		"Internal error rendering help: %s", err)))
}

// helpCommands returns the subcommands for the HelpFunc argument.
// This will only contain immediate subcommands.
func (c *CLI) helpCommands(prefix string) map[string]CommandFactory {
	// If our prefix isn't empty, make sure it ends in ' '
	if prefix != "" && prefix[len(prefix)-1] != ' ' {
		prefix += " "
	}

	// Get all the subkeys of this command
	var keys []string
	c.commandTree.WalkPrefix(prefix, func(k string, raw interface{}) bool {
		// Ignore any sub-sub keys, i.e. "foo bar baz" when we want "foo bar"
		if !strings.Contains(k[len(prefix):], " ") {
			keys = append(keys, k)
		}

		return false
	})

	// For each of the keys return that in the map
	result := make(map[string]CommandFactory, len(keys))
	for _, k := range keys {
		raw, ok := c.commandTree.Get(k)
		if !ok {
			// We just got it via WalkPrefix above, so we just panic
			panic("not found: " + k)
		}

		result[k] = raw.(CommandFactory)
	}

	return result
}

func (c *CLI) processArgs() {
	for i, arg := range c.Args {
		if arg == "--" {
			break
		}

		// Check for help flags.
		if arg == "-h" || arg == "-help" || arg == "--help" {
			c.isHelp = true
			continue
		}

		// Check for autocomplete flags
		if c.Autocomplete {
			if arg == "-"+c.AutocompleteInstall || arg == "--"+c.AutocompleteInstall {
				c.isAutocompleteInstall = true
				continue
			}

			if arg == "-"+c.AutocompleteUninstall || arg == "--"+c.AutocompleteUninstall {
				c.isAutocompleteUninstall = true
				continue
			}
		}

		if c.subcommand == "" {
			// Check for version flags if not in a subcommand.
			if arg == "-v" || arg == "-version" || arg == "--version" {
				c.isVersion = true
				continue
			}

			if arg != "" && arg[0] == '-' {
				// Record the arg...
				c.topFlags = append(c.topFlags, arg)
			}
		}

		// If we didn't find a subcommand yet and this is the first non-flag
		// argument, then this is our subcommand.
		if c.subcommand == "" && arg != "" && arg[0] != '-' {
			c.subcommand = arg
			if c.commandNested {
				// Nested CLI, the subcommand is actually the entire
				// arg list up to a flag that is still a valid subcommand.
				searchKey := strings.Join(c.Args[i:], " ")
				k, _, ok := c.commandTree.LongestPrefix(searchKey)
				if ok {
					// k could be a prefix that doesn't contain the full
					// command such as "foo" instead of "foobar", so we
					// need to verify that we have an entire key. To do that,
					// we look for an ending in a space or an end of string.
					reVerify := regexp.MustCompile(regexp.QuoteMeta(k) + `( |$)`)
					if reVerify.MatchString(searchKey) {
						c.subcommand = k
						i += strings.Count(k, " ")
					}
				}
			}

			// The remaining args the subcommand arguments
			c.subcommandArgs = c.Args[i+1:]
		}
	}

	// If we never found a subcommand and support a default command, then
	// switch to using that.
	if c.subcommand == "" {
		if _, ok := c.Commands[""]; ok {
			args := c.topFlags
			args = append(args, c.subcommandArgs...)
			c.topFlags = nil
			c.subcommandArgs = args
		}
	}
}

// defaultAutocompleteInstall and defaultAutocompleteUninstall are the
// default values for the autocomplete install and uninstall flags.
const defaultAutocompleteInstall = "autocomplete-install"
const defaultAutocompleteUninstall = "autocomplete-uninstall"

const defaultHelpTemplate = `
{{.Help}}{{if gt (len .Subcommands) 0}}

Subcommands:
{{- range $value := .Subcommands }}
    {{ $value.NameAligned }}    {{ $value.Synopsis }}{{ end }}
{{- end }}
`
