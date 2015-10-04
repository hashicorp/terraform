package command

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/hashicorp/terraform/terraform"
	"strings"
)

// SchemasCommand is a Command implementation that reads and outputs the
// schemas of all installed Terraform providers and resource types.
type SchemasCommand struct {
	Meta
}

type providerResourceSchema struct {
	terraform.ResourceProviderSchema
	Name string `json:"name"`
	Type string `json:"type"`
}

type provisionerResourceSchemaInfo struct {
	terraform.ResourceProvisionerSchema
	Name string `json:"name"`
	Type string `json:"type"`
}

func (c *SchemasCommand) Run(args []string) int {
	var indent bool
	var inJson bool

	args = c.Meta.process(args, false)

	cmdFlags := flag.NewFlagSet("schemas", flag.ContinueOnError)
	cmdFlags.BoolVar(&indent, "indent", false, "Indent output")
	// 'inJson' ignored for now, always true
	cmdFlags.BoolVar(&inJson, "json", true, "In JSON format")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("The schemas command expects one argument with the type of provider/resource.")
		cmdFlags.Usage()
		return 1
	}

	// TODO: Use c.Ui.Output(FormatSchema ...

	for k, v := range c.Meta.ContextOpts.Providers {
		if len(args) == 1 && args[0] != k {
			continue
		}
		if provider, err := v(); err == nil {
			export, err := provider.Export()
			if err != nil {
				fmt.Printf("Cannot get schema for provider '%s': %s\n", k, err)
				continue
			}
			extended := providerResourceSchema{export, k, "provider"}
			var ser []byte
			var err2 error
			if indent {
				ser, err2 = json.MarshalIndent(extended, "", "  ")
			} else {
				ser, err2 = json.Marshal(extended)
			}
			if err2 != nil {
				fmt.Printf("Cannot serialize schema for provider '%s': %s\n", k, err)
				continue
			}
			fmt.Println(string(ser))
			break
		}
	}
	for k, v := range c.Meta.ContextOpts.Provisioners {
		if len(args) == 1 && args[0] != k {
			continue
		}
		if provisioner, err := v(); err == nil {
			export, err := provisioner.Export()
			if err != nil {
				fmt.Printf("Cannot get schema for provisioner '%s': %s\n", k, err)
				continue
			}
			extended := provisionerResourceSchemaInfo{export, k, "provisioner"}
			var ser []byte
			var err2 error
			if indent {
				ser, err2 = json.MarshalIndent(extended, "", "  ")
			} else {
				ser, err2 = json.Marshal(extended)
			}
			if err2 != nil {
				fmt.Printf("Cannot serialize schema for provisioner '%s': %s\n", k, err)
				continue
			}
			fmt.Println(string(ser))
			break
		}
	}

	return 0
}

func (c *SchemasCommand) Help() string {
	// TODO: Support more than one name element, all probably return everything at once
	helpText := `
Usage: terraform schemas [options] name

  Reads and outputs the schema of specified ('name') Terraform provider,
  provisioner or resource in machine- or human-readable form.

Options:

  -indent		      If specified, output would be indented.

  -json		          If specified, output would be in JSON format.
`
	return strings.TrimSpace(helpText)
}

func (c *SchemasCommand) Synopsis() string {
	return "Shows schemas of Terraform providers/resources"
}
