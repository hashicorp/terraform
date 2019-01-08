package command

import (
	"bufio"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/config/module"
)

// ZeroTwelveChecklistCommand is a Command implementation that checks whether
// a configuration is ready for upgrade to Terraform 0.12, producing a list
// of remaining preparation steps if not.
type ZeroTwelveChecklistCommand struct {
	Meta
}

func (c *ZeroTwelveChecklistCommand) Help() string {
	return zeroTwelveChecklistCommandHelp
}

func (c *ZeroTwelveChecklistCommand) Synopsis() string {
	return "Checks whether the configuration is ready for Terraform v0.12"
}

func (c *ZeroTwelveChecklistCommand) Run(args []string) int {
	c.Meta.process(args, false)

	cmdFlags := c.Meta.flagSet("0.12checklist")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Load the config
	root, diags := c.Module(configPath)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}
	if root == nil {
		c.Ui.Error(fmt.Sprintf(
			"No configuration files found in the directory: %s\n\n"+
				"This command requires configuration to run.",
			configPath))
		return 1
	}

	items := make(map[string][]string)
	hasItems := zeroTwelveChecklists(root, items)
	if !hasItems {
		// TODO: Success message
		return 0
	}

	// TODO: Format the checklist.
	modKeys := make([]string, 0, len(items))
	for k := range items {
		modKeys = append(modKeys, k)
	}
	sort.Strings(modKeys)

	for _, k := range modKeys {
		modItems := items[k]
		sort.Strings(modItems)

		if k != "" {
			fmt.Printf("# Module `%q`\n\n", k)
		}

		for _, item := range modItems {
			fmt.Print("- [ ] ")
			sc := bufio.NewScanner(strings.NewReader(item))
			i := 0
			for sc.Scan() {
				if i == 0 {
					fmt.Printf("%s\n", sc.Text())
				} else {
					fmt.Printf("  %s\n", sc.Text())
				}
				i++
			}
			fmt.Printf("\n")
		}
	}

	return 1
}

func zeroTwelveChecklists(mod *module.Tree, into map[string][]string) bool {
	key := strings.Join(mod.Path(), ".")
	items := zeroTwelveChecklistForModule(mod)
	hasItems := false

	childMods := mod.Children()
	for _, modCall := range mod.Config().Modules {
		childMod, ok := childMods[modCall.Name]
		if !ok {
			// Should never happen.
			log.Printf("[WARN] Module %s declares child module %q but its tree node is missing", key, modCall.Name)
			continue
		}
		if !(strings.HasPrefix(modCall.Source, "./") || strings.HasPrefix(modCall.Source, "../")) {
			// For non-local modules we'll still run the checks but we'll roll
			// up into a single action item for our calling module if any
			// changes are needed, since the changes really need to be made
			// in the upstream repository.
			childItems := zeroTwelveChecklistForModule(childMod)
			if len(childItems) > 0 {
				items = append(items, "Upgrade child module %q to a version that passes \"terraform 0.12checklist\".")
			}
			continue
		}

		childHasItems := zeroTwelveChecklists(childMod, into)
		if childHasItems {
			hasItems = true
		}
	}

	if len(items) > 0 {
		hasItems = true
	}
	into[key] = items
	return hasItems
}

func zeroTwelveChecklistForModule(mod *module.Tree) []string {
	var items []string

	// Strings added to items must be Markdown-formatted. They can be multi-line
	// as long as all of the lines are valid to be nested inside a list item.
	// The caller above will eventually add the required initial indendation
	// to make the item's content appear as part of the item.
	//
	// In particular, items can include fenced code blocks and sub-lists.
	// However, it's best to keep Markdown metacharacters to a minimum so that
	// the result is also easy to read directly with human eyes, without
	// passing through a Markdown renderer.
	//
	// Each element of "items" will be rendered as a task list item using
	// GitHub's task list extension.

	return items
}

const zeroTwelveChecklistCommandHelp = `
Usage: terraform 0.12checklist [dir]

  Analyzes a configuration and produces a list of any preparation steps
  required before upgrading to Terraform v0.12.

  For best results, run this command with no Terraform changes pending, so that
  it can analyze your infrastructure as currently deployed, rather than as
  currently planned.

  The resulting output uses Markdown formatting so you can easily copy it into
  a Markdown-capable issue tracker. We use GitHub-flavored Markdown.
`
