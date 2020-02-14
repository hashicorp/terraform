package command

import (
	"fmt"
	"io/ioutil"
	"strings"

	// "github.com/hashicorp/terraform/command/jsonconfig"
	"github.com/hashicorp/terraform/lang/langserver"
)

// ShowConfigCommand is a Command implementation that reads and outputs the
// contents of a Terraform plan or state file.
type ShowConfigCommand struct {
	Meta
}

func (c *ShowConfigCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("show-config")
	var jsonOutput bool
	var byteOffset int
	cmdFlags.BoolVar(&jsonOutput, "json", false, "produce JSON output")
	cmdFlags.IntVar(&byteOffset, "offset", -1, "byte offset")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	path := cmdFlags.Arg(0)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		c.Ui.Error(err.Error())
	}

	// find addrs.Reference via offset first
	f := langserver.NewFile(path, content)
	ref := f.ResolveRefAtByteOffset(byteOffset)

	c.Ui.Output(ref.Subject.String())
	// // parse the config
	// p := configs.NewParser(dummyFs())
	// f := p.LoadConfigFile(path)
	// m, diags := configs.NewModule([]*configs.File{f})

	// // TODO: Generic reference finder
	// // rawCfg := m.ConfigByAddr(ref)

	// // find the relevant part via reference
	// rawResource := m.ResourceByAddr(ref)

	// schemas := ctx.Schemas()

	// // TODO: Get schema via reference, so we don't need to be passing around all schemas
	// // schema := schemas.SchemaForResourceAddr(ref)

	// jsonResource, err := jsonconfig.MarshalResource(rawResource, schemas)
	// if err != nil {
	// 	c.Ui.Error(fmt.Sprintf("Failed to marshal config to json: %s", err))
	// 	return 1
	// }

	// c.Ui.Output(string(jsonResource))

	return 0
}

func (c *ShowConfigCommand) Help() string {
	helpText := `
Usage: terraform show-config [options] [path]

  Reads and outputs Terraform configuration.
  If no path is specified, the configuration
  from the current workdir will be shown.

Options:

  -offset Byte offset within the file

  -json   If specified, output the configuration in
          a machine-readable form.

`
	return strings.TrimSpace(helpText)
}

func (c *ShowConfigCommand) Synopsis() string {
	return "Inspect Terraform configuration"
}
