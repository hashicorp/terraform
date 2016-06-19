package command

import (
	"flag"
	"fmt"
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/terraform"
	"strings"
)

// SchemasCommand is a Command implementation that reads and outputs the
// schemas of all installed Terraform providers and resource types.
type SchemasCommand struct {
	Meta
}

type resultBase struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type providerResourceSchema struct {
	resultBase
	terraform.ResourceProviderSchema
}

type resourceResourceSchema struct {
	resultBase
	Provider string               `json:"provider"`
	Schema   terraform.SchemaInfo `json:"schema"`
}

type provisionerResourceSchemaInfo struct {
	resultBase
	*terraform.ResourceProvisionerSchema
}

type errorResult struct {
	resultBase
	Error string `json:"error"`
}

type FunctionInfo struct {
	Name         string
	ArgTypes     []string
	ReturnType   string
	Variadic     bool   `json:",omitempty"`
	VariadicType string `json:",omitempty"`
}

type functionSchema struct {
	resultBase
	FunctionInfo `json:"schema"`
}

type functionsSchema struct {
	resultBase
	Functions map[string]FunctionInfo `json:"schema"`
}

func (c *SchemasCommand) Run(args []string) int {
	var indent bool
	var inJson bool
	var inXml bool
	var expectedType string

	args = c.Meta.process(args, false)

	cmdFlags := flag.NewFlagSet("schemas", flag.ContinueOnError)
	cmdFlags.BoolVar(&indent, "indent", false, "Indent output")
	cmdFlags.BoolVar(&inJson, "json", false, "In JSON format")
	cmdFlags.StringVar(&expectedType, "type", "any", "In JSON format")
	expectedType = strings.ToLower(expectedType)
	// Temporarily disabled due to not-implemented xml serializer for SchemaInfo (which is map[string]interface{})
	//cmdFlags.BoolVar(&inXml, "xml", false, "In XML format")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error("Cannot parser command line arguments" + err.Error())
		cmdFlags.Usage()
		return 1
	}

	if inXml && inJson {
		c.Ui.Error("Cannot produce output in both xml in json formats at the same time. Either use -josn or -xml flags")
		return 1
	}

	if inXml || inJson {
		c.color = false
	}
	var format string
	if inJson {
		format = "json"
	} else if inXml {
		format = "xml"
	} else {
		format = "plain"
	}

	args = cmdFlags.Args()
	if len(args) != 1 {
		c.Ui.Error("The schemas command expects one argument with the type of provider/resource.")
		cmdFlags.Usage()
		return 1
	}

	var s interface{}
	s = getOrErrorResult(c.Meta.ContextOpts, args[0], expectedType)

	c.Ui.Output(FormatSchema(&FormatSchemaOpts{
		Name:      args[0],
		Schema:    &s,
		Colorize:  c.color,
		Colorizer: c.Colorize(),
		Format:    format,
		Indent:    indent,
	}))

	// Return non-zero xit code in case of error (error result)
	switch s.(type) {
	case errorResult:
		return 1
	default:
		return 0
	}
}

func (c *SchemasCommand) Help() string {
	// TODO: Support more than one name element, all probably return everything at once
	helpText := `
Usage: terraform schemas [options] name

  Reads and outputs the schema of specified ('name') Terraform provider,
  provisioner or resource in machine- or human-readable form.

Options:

  -indent		      If specified, output would be indented.

  -json		          If specified, output would be in JSON format. Implies '--no-color'.

  -type	<type>            If specified, would search for specific type, e.g. provider, resource, provisioner or function.
`
	return strings.TrimSpace(helpText)
}

func (c *SchemasCommand) Synopsis() string {
	return "Shows schemas of Terraform providers/resources"
}

func getOrErrorResult(context *terraform.ContextOpts, name string, expectedType string) interface{} {
	var s interface{}
	var e error
	switch expectedType {
	case "function":
		s = getFunctionSchema(name)
	case "provider":
		s, e = getProviderSchema(context.Providers, name)
	case "resource":
		s, e = getResourceSchema(context.Providers, name)
	case "provisioner":
		s, e = getProvisionerSchema(context.Provisioners, name)
	case "any":
		s = getAnythingOrErrorResult(context, name)
	default:
		return errorResult{resultBase{name, "unknown"}, "Unexpected type " + expectedType}
	}
	if e != nil {
		return errorResult{resultBase{name, expectedType}, e.Error()}
	} else if s != nil {
		return s
	}
	return errorResult{resultBase{name, expectedType}, "Not found"}
}

func getAnythingOrErrorResult(context *terraform.ContextOpts, name string) interface{} {
	if name == "functions" {
		return functionsSchema{resultBase{"functions", "functions"}, getInterpolationFunctions()}
	}
	var a interface{}
	var e error
	a, e = getProviderSchema(context.Providers, name)
	if e != nil {
		return errorResult{resultBase{name, "provider"}, e.Error()}
	} else if a != nil {
		return a
	}
	a, e = getResourceSchema(context.Providers, name)
	if e != nil {
		return errorResult{resultBase{name, "resource"}, e.Error()}
	} else if a != nil {
		return a
	}
	a, e = getProvisionerSchema(context.Provisioners, name)
	if e != nil {
		return errorResult{resultBase{name, "provisioner"}, e.Error()}
	} else if a != nil {
		return a
	}
	return errorResult{resultBase{name, "unknown"}, "Not found"}
}

func getInterpolationFunctions() map[string]FunctionInfo {
	vars := make(map[string]ast.Variable)
	cfg := config.LangEvalConfig(vars)
	result := make(map[string]FunctionInfo)
	for name, fun := range cfg.GlobalScope.FuncMap {
		args := make([]string, len(fun.ArgTypes))
		for i, at := range fun.ArgTypes {
			args[i] = at.String()
		}
		vt := ""
		if fun.Variadic {
			vt = fun.VariadicType.String()
		}
		result[name] = FunctionInfo{name, args, fun.ReturnType.String(), fun.Variadic, vt}
	}

	return result
}

func getInterpolationFunction(name string) (*FunctionInfo, bool) {
	vars := make(map[string]ast.Variable)
	cfg := config.LangEvalConfig(vars)
	fm := cfg.GlobalScope.FuncMap
	if fun, ok := fm[name]; ok {
		args := make([]string, len(fun.ArgTypes))
		for i, at := range fun.ArgTypes {
			args[i] = at.String()
		}
		vt := ""
		if fun.Variadic {
			vt = fun.VariadicType.String()
		}
		return &FunctionInfo{name, args, fun.ReturnType.String(), fun.Variadic, vt}, true
	}
	return nil, false
}

func getFunctionSchema(name string) interface{} {
	function, found := getInterpolationFunction(name)
	if !found {
		return nil
	}
	return functionSchema{resultBase{name, "function"}, *function}
}

func getProviderSchema(providers map[string]terraform.ResourceProviderFactory, name string) (interface{}, error) {
	for k, v := range providers {
		if name != k {
			continue
		}
		if provider, err := v(); err == nil {
			export, err := provider.Export()
			if err != nil {
				return nil, fmt.Errorf("Cannot get schema for provider '%s': %s\n", k, err)
			}
			extended := providerResourceSchema{resultBase{k, "provider"}, export}
			return extended, nil
		}
	}
	return nil, nil
}

func getResourceSchema(providers map[string]terraform.ResourceProviderFactory, name string) (interface{}, error) {
	for k, v := range providers {
		if provider, err := v(); err == nil {
			resources := provider.Resources()
			for _, n := range resources {
				if n.Name != name {
					continue
				}
				export, err := provider.Export()
				if err != nil {
					return nil, fmt.Errorf("Cannot get schema for resource '%s': %s\n", k, err)
				}
				extended := resourceResourceSchema{resultBase{k, "resource"}, k, export.Resources[n.Name]}
				return extended, nil
			}
		}
	}
	return nil, nil
}

func getProvisionerSchema(providers map[string]terraform.ResourceProvisionerFactory, name string) (interface{}, error) {
	for k, v := range providers {
		if name != k {
			continue
		}
		if provisioner, err := v(); err == nil {
			export, err := provisioner.Export()
			if err != nil {
				return nil, fmt.Errorf("Cannot get schema for provisioner '%s': %s\n", k, err)
			}
			extended := provisionerResourceSchemaInfo{resultBase{k, "provisioner"}, export}
			return extended, nil
		}
	}
	return nil, nil
}
