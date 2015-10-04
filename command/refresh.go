package command

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
)

// RefreshCommand is a cli.Command implementation that refreshes the state
// file.
type RefreshCommand struct {
	Meta
}

func walkModule(parent string, mod *module.Tree, resources map[string]bool) {
	var modName string
	if parent == "" {
		modName = mod.Name()
	} else {
		modName = fmt.Sprintf("%s.%s", parent, mod.Name())
	}
	for _, resource := range mod.Config().Resources {
		key := fmt.Sprintf("%s/%s.%s", modName, resource.Type, resource.Name)
		resources[key] = true
	}
	for _, child := range mod.Children() {
		walkModule(modName, child, resources)
	}
}

// Builds a "set" of module/resource so that we can easily lookup what's
// configured
func (c *RefreshCommand) findConfiguredResources(ctx *terraform.Context) map[string]bool {
	ret := make(map[string]bool)

	mod := ctx.Module()
	walkModule("", mod, ret)

	return ret
}

// Import existing (configured) resources by minimally adding them to their
// module's resources so that a subsequent refresh will pull down their
// details.
func (c *RefreshCommand) importResources(s *terraform.State, configuredResources map[string]bool, importPath string) bool {
	f, err := os.Open(importPath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error opening import file (%s): %s",
			importPath, err))
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			// blank line or comment
			continue
		}
		pieces := strings.Split(line, " ")
		if len(pieces) != 3 {
			c.Ui.Error(fmt.Sprintf("Error malformed import line %s", line))
			return false
		}
		// Make sure we have a config for this resource
		key := fmt.Sprintf("%s/%s", pieces[0], pieces[1])
		if _, ok := configuredResources[key]; ok {
			// if so try adding it
			log.Printf("[INFO] adding %s -> %s", key, pieces[2])

			// Find our target module
			mod := s.ModuleByPath(strings.Split(pieces[0], "."))
			if mod == nil {
				c.Ui.Error(fmt.Sprintf("Failed to find module %s", pieces[0]))
				return false
			}

			// Ignore resources that already exist
			if _, ok := mod.Resources[pieces[1]]; ok {
				log.Printf("[INFO] resource %s already exists in module %s, skipping",
					pieces[1], pieces[0])
				continue
			}

			// Minimally add it
			mod.Resources[pieces[1]] = &terraform.ResourceState{
				Type: strings.Split(pieces[1], ".")[0],
				Primary: &terraform.InstanceState{
					ID: pieces[2],
				},
			}
		}
	}
	if err = scanner.Err(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed reading import file (%s): %s",
			importPath, err))
		return false
	}

	return true
}

func (c *RefreshCommand) Run(args []string) int {
	var importPath string

	args = c.Meta.process(args, true)

	cmdFlags := c.Meta.flagSet("refresh")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.IntVar(&c.Meta.parallelism, "parallelism", 0, "parallelism")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.StringVar(&importPath, "import", "", "path")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var configPath string
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error("The refresh command expects at most one argument.")
		cmdFlags.Usage()
		return 1
	} else if len(args) == 1 {
		configPath = args[0]
	} else {
		var err error
		configPath, err = os.Getwd()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		}
	}

	// Check if remote state is enabled
	state, err := c.State()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load state: %s", err))
		return 1
	}

	// Verify that the state path exists. The "ContextArg" function below
	// will actually do this, but we want to provide a richer error message
	// if possible.
	if !state.State().IsRemote() {
		if _, err := os.Stat(c.Meta.statePath); err != nil {
			if os.IsNotExist(err) {
				c.Ui.Error(fmt.Sprintf(
					"The Terraform state file for your infrastructure does not\n"+
						"exist. The 'refresh' command only works and only makes sense\n"+
						"when there is existing state that Terraform is managing. Please\n"+
						"double-check the value given below and try again. If you\n"+
						"haven't created infrastructure with Terraform yet, use the\n"+
						"'terraform apply' command.\n\n"+
						"Path: %s",
					c.Meta.statePath))
				return 1
			}

			c.Ui.Error(fmt.Sprintf(
				"There was an error reading the Terraform state that is needed\n"+
					"for refreshing. The path and error are shown below.\n\n"+
					"Path: %s\n\nError: %s",
				c.Meta.statePath,
				err))
			return 1
		}
	}

	// Build the context based on the arguments given
	ctx, _, err := c.Context(contextOpts{
		Path:        configPath,
		StatePath:   c.Meta.statePath,
		Parallelism: c.Meta.parallelism,
	})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if !validateContext(ctx, c.Ui) {
		return 1
	}
	if err := ctx.Input(c.InputMode()); err != nil {
		c.Ui.Error(fmt.Sprintf("Error configuring: %s", err))
		return 1
	}

	if importPath != "" {
		log.Printf("[INFO] Importing resources from %s", importPath)

		configuredResources := c.findConfiguredResources(ctx)

		s := state.State()
		resourceMappings := c.importResources(s, configuredResources, importPath)
		if !resourceMappings {
			// importResources will have provided an error message
			return 1
		}

		log.Printf("[INFO] Updating context state")
		ctx.UpdateState(s)
	}

	newState, err := ctx.Refresh()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error refreshing state: %s", err))
		return 1
	}

	log.Printf("[INFO] Writing state output to: %s", c.Meta.StateOutPath())
	if err := c.Meta.PersistState(newState); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	if outputs := outputsAsString(newState); outputs != "" {
		c.Ui.Output(c.Colorize().Color(outputs))
	}

	return 0
}

func (c *RefreshCommand) Help() string {
	helpText := `
Usage: terraform refresh [options] [dir]

  Update the state file of your infrastructure with metadata that matches
  the physical resources they are tracking.

  This will not modify your infrastructure, but it can modify your
  state file to update metadata. This metadata might cause new changes
  to occur when you generate a plan or call apply next.

Options:

  -backup=path        Path to backup the existing state file before
                      modifying. Defaults to the "-state-out" path with
                      ".backup" extension. Set to "-" to disable backup.

  -import=path        Path to a file containing a mapping, one per line,
                      between module, resource, and identifier to allow
                      bringing exiting resources under terraform
                      management. E.g.

                          module resource id
                          root aws_vpc.primary vpc-24bd392c
                          root aws_subnet.public subnet-42ba370e

  -input=true         Ask for input for variables if not directly set.

  -no-color           If specified, output won't contain any color.

  -state=path         Path to read and save state (unless state-out
                      is specified). Defaults to "terraform.tfstate".

  -state-out=path     Path to write updated state file. By default, the
                      "-state" path will be used.

  -target=resource    Resource to target. Operation will be limited to this
                      resource and its dependencies. This flag can be used
                      multiple times.

  -var 'foo=bar'      Set a variable in the Terraform configuration. This
                      flag can be set multiple times.

  -var-file=foo       Set variables in the Terraform configuration from
                      a file. If "terraform.tfvars" is present, it will be
                      automatically loaded if this flag is not specified.

`
	return strings.TrimSpace(helpText)
}

func (c *RefreshCommand) Synopsis() string {
	return "Update local state file against real resources"
}
