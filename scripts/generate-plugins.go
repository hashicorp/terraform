// Generate Plugins is a small program that updates the lists of plugins in
// command/internal_plugin_list.go so they will be compiled into the main
// terraform binary.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const target = "command/internal_plugin_list.go"

func main() {
	wd, _ := os.Getwd()
	if filepath.Base(wd) != "terraform" {
		log.Fatalf("This program must be invoked in the terraform project root; in %s", wd)
	}

	//// Collect all of the data we need about plugins we have in the project
	//providers, err := discoverProviders()
	//if err != nil {
	//    log.Fatalf("Failed to discover providers: %s", err)
	//}

	provisioners, err := discoverProvisioners()
	if err != nil {
		log.Fatalf("Failed to discover provisioners: %s", err)
	}

	// Do some simple code generation and templating
	output := source
	output = strings.Replace(output, "IMPORTS", makeImports(nil, provisioners), 1)
	//output = strings.Replace(output, "PROVIDERS", makeProviderMap(providers), 1)
	output = strings.Replace(output, "PROVISIONERS", makeProvisionerMap(provisioners), 1)

	// TODO sort the lists of plugins so we are not subjected to random OS ordering of the plugin lists

	// Write our generated code to the command/plugin.go file
	file, err := os.Create(target)
	defer file.Close()
	if err != nil {
		log.Fatalf("Failed to open %s for writing: %s", target, err)
	}

	_, err = file.WriteString(output)
	if err != nil {
		log.Fatalf("Failed writing to %s: %s", target, err)
	}

	log.Printf("Generated %s", target)
}

type plugin struct {
	Package    string // Package name from ast  remoteexec
	PluginName string // Path via deriveName()  remote-exec
	TypeName   string // Type of plugin         provisioner
	Path       string // Relative import path   builtin/provisioners/remote-exec
	ImportName string // See deriveImport()     remoteexecprovisioner
}

// makeProviderMap creates a map of providers like this:
//
// var InternalProviders = map[string]plugin.ProviderFunc{
// 	"aws":        aws.Provider,
// 	"azurerm":    azurerm.Provider,
// 	"cloudflare": cloudflare.Provider,
func makeProviderMap(items []plugin) string {
	output := ""
	for _, item := range items {
		output += fmt.Sprintf("\t\"%s\":   %s.%s,\n", item.PluginName, item.ImportName, item.TypeName)
	}
	return output
}

// makeProvisionerMap creates a map of provisioners like this:
//
//	"chef":            chefprovisioner.Provisioner,
//	"salt-masterless": saltmasterlessprovisioner.Provisioner,
//	"file":            fileprovisioner.Provisioner,
//	"local-exec":      localexecprovisioner.Provisioner,
//	"remote-exec":     remoteexecprovisioner.Provisioner,
//
func makeProvisionerMap(items []plugin) string {
	output := ""
	for _, item := range items {
		output += fmt.Sprintf("\t\"%s\":   %s.%s,\n", item.PluginName, item.ImportName, item.TypeName)
	}
	return output
}

func makeImports(providers, provisioners []plugin) string {
	plugins := []string{}

	for _, provider := range providers {
		plugins = append(plugins, fmt.Sprintf("\t%s \"github.com/hashicorp/terraform/%s\"\n", provider.ImportName, filepath.ToSlash(provider.Path)))
	}

	for _, provisioner := range provisioners {
		plugins = append(plugins, fmt.Sprintf("\t%s \"github.com/hashicorp/terraform/%s\"\n", provisioner.ImportName, filepath.ToSlash(provisioner.Path)))
	}

	// Make things pretty
	sort.Strings(plugins)

	return strings.Join(plugins, "")
}

// listDirectories recursively lists directories under the specified path
func listDirectories(path string) ([]string, error) {
	names := []string{}
	items, err := ioutil.ReadDir(path)
	if err != nil {
		return names, err
	}

	for _, item := range items {
		// We only want directories
		if item.IsDir() {
			if item.Name() == "test-fixtures" {
				continue
			}
			currentDir := filepath.Join(path, item.Name())
			names = append(names, currentDir)

			// Do some recursion
			subNames, err := listDirectories(currentDir)
			if err == nil {
				names = append(names, subNames...)
			}
		}
	}

	return names, nil
}

// deriveName determines the name of the plugin relative to the specified root
// path.
func deriveName(root, full string) string {
	short, _ := filepath.Rel(root, full)
	bits := strings.Split(short, string(os.PathSeparator))
	return strings.Join(bits, "-")
}

// deriveImport will build a unique import identifier based on packageName and
// the result of deriveName(). This is important for disambigutating between
// providers and provisioners that have the same name. This will be something
// like:
//
//	remote-exec -> remoteexecprovisioner
//
// which is long, but is deterministic and unique.
func deriveImport(typeName, derivedName string) string {
	return strings.Replace(derivedName, "-", "", -1) + strings.ToLower(typeName)
}

// discoverTypesInPath searches for types of typeID in path using go's ast and
// returns a list of plugins it finds.
func discoverTypesInPath(path, typeID, typeName string) ([]plugin, error) {
	pluginTypes := []plugin{}

	dirs, err := listDirectories(path)
	if err != nil {
		return pluginTypes, err
	}

	for _, dir := range dirs {
		fset := token.NewFileSet()
		goPackages, err := parser.ParseDir(fset, dir, nil, parser.AllErrors)
		if err != nil {
			return pluginTypes, fmt.Errorf("Failed parsing directory %s: %s", dir, err)
		}

		for _, goPackage := range goPackages {
			ast.PackageExports(goPackage)
			ast.Inspect(goPackage, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					// If we get a function then we will check the function name
					// against typeName and the function return type (Results)
					// against typeID.
					//
					// There may be more than one return type but in the target
					// case there should only be one. Also the return type is a
					// ast.SelectorExpr which means we have multiple nodes.
					// We'll read all of them as ast.Ident (identifier), join
					// them via . to get a string like terraform.ResourceProvider
					// and see if it matches our expected typeID
					//
					// This is somewhat verbose but prevents us from identifying
					// the wrong types if the function name is amiguous or if
					// there are other subfolders added later.
					if x.Name.Name == typeName && len(x.Type.Results.List) == 1 {
						node := x.Type.Results.List[0].Type
						typeIdentifiers := []string{}
						ast.Inspect(node, func(m ast.Node) bool {
							switch y := m.(type) {
							case *ast.Ident:
								typeIdentifiers = append(typeIdentifiers, y.Name)
							}
							// We need all of the identifiers to join so we
							// can't break early here.
							return true
						})
						if strings.Join(typeIdentifiers, ".") == typeID {
							derivedName := deriveName(path, dir)
							pluginTypes = append(pluginTypes, plugin{
								Package:    goPackage.Name,
								PluginName: derivedName,
								ImportName: deriveImport(x.Name.Name, derivedName),
								TypeName:   x.Name.Name,
								Path:       dir,
							})
						}
					}
				case *ast.TypeSpec:
					// In the simpler case we will simply check whether the type
					// declaration has the name we were looking for.
					if x.Name.Name == typeID {
						derivedName := deriveName(path, dir)
						pluginTypes = append(pluginTypes, plugin{
							Package:    goPackage.Name,
							PluginName: derivedName,
							ImportName: deriveImport(x.Name.Name, derivedName),
							TypeName:   x.Name.Name,
							Path:       dir,
						})
						// The AST stops parsing when we return false. Once we
						// find the symbol we want we can stop parsing.
						return false
					}
				}
				return true
			})
		}
	}

	return pluginTypes, nil
}

func discoverProviders() ([]plugin, error) {
	path := "./builtin/providers"
	typeID := "terraform.ResourceProvider"
	typeName := "Provider"
	return discoverTypesInPath(path, typeID, typeName)
}

func discoverProvisioners() ([]plugin, error) {
	path := "./builtin/provisioners"
	typeID := "terraform.ResourceProvisioner"
	typeName := "Provisioner"
	return discoverTypesInPath(path, typeID, typeName)
}

const source = `//
// This file is automatically generated by scripts/generate-plugins.go -- Do not edit!
//
package command

import (
IMPORTS
	"github.com/hashicorp/terraform/plugin"
)

var InternalProviders = map[string]plugin.ProviderFunc{}

var InternalProvisioners = map[string]plugin.ProvisionerFunc{
PROVISIONERS
}
`
