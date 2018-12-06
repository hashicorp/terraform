package configupgrade

import (
	"fmt"
	"log"
	"strings"

	hcl1 "github.com/hashicorp/hcl"
	hcl1ast "github.com/hashicorp/hcl/hcl/ast"
	hcl1parser "github.com/hashicorp/hcl/hcl/parser"
	hcl1token "github.com/hashicorp/hcl/hcl/token"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/moduledeps"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/terraform"
)

// analysis is a container for the various different information gathered
// by Upgrader.analyze.
type analysis struct {
	ProviderSchemas      map[string]*terraform.ProviderSchema
	ProvisionerSchemas   map[string]*configschema.Block
	ResourceProviderType map[addrs.Resource]string
	ResourceHasCount     map[addrs.Resource]bool
	VariableTypes        map[string]string
}

// analyze processes the configuration files included inside the receiver
// and returns an assortment of information required to make decisions during
// a configuration upgrade.
func (u *Upgrader) analyze(ms ModuleSources) (*analysis, error) {
	ret := &analysis{
		ProviderSchemas:      make(map[string]*terraform.ProviderSchema),
		ProvisionerSchemas:   make(map[string]*configschema.Block),
		ResourceProviderType: make(map[addrs.Resource]string),
		ResourceHasCount:     make(map[addrs.Resource]bool),
		VariableTypes:        make(map[string]string),
	}

	m := &moduledeps.Module{
		Providers: make(moduledeps.Providers),
	}

	// This is heavily based on terraform.ModuleTreeDependencies but
	// differs in that it works directly with the HCL1 AST rather than
	// the legacy config structs (and can thus outlive those) and that
	// it only works on one module at a time, and so doesn't need to
	// recurse into child calls.
	for name, src := range ms {
		if ext := fileExt(name); ext != ".tf" {
			continue
		}

		log.Printf("[TRACE] configupgrade: Analyzing %q", name)

		f, err := hcl1parser.Parse(src)
		if err != nil {
			// If we encounter a syntax error then we'll just skip for now
			// and assume that we'll catch this again when we do the upgrade.
			// If not, we'll break the upgrade step of renaming .tf files to
			// .tf.json if they seem to be JSON syntax.
			log.Printf("[ERROR] Failed to parse %q: %s", name, err)
			continue
		}

		list, ok := f.Node.(*hcl1ast.ObjectList)
		if !ok {
			return nil, fmt.Errorf("error parsing: file doesn't contain a root object")
		}

		if providersList := list.Filter("provider"); len(providersList.Items) > 0 {
			providerObjs := providersList.Children()
			for _, providerObj := range providerObjs.Items {
				if len(providerObj.Keys) != 1 {
					return nil, fmt.Errorf("provider block has wrong number of labels")
				}
				name := providerObj.Keys[0].Token.Value().(string)

				var listVal *hcl1ast.ObjectList
				if ot, ok := providerObj.Val.(*hcl1ast.ObjectType); ok {
					listVal = ot.List
				} else {
					return nil, fmt.Errorf("provider %q: must be a block", name)
				}

				var versionStr string
				if a := listVal.Filter("version"); len(a.Items) > 0 {
					err := hcl1.DecodeObject(&versionStr, a.Items[0].Val)
					if err != nil {
						return nil, fmt.Errorf("Error reading version for provider %q: %s", name, err)
					}
				}
				var constraints discovery.Constraints
				if versionStr != "" {
					constraints, err = discovery.ConstraintStr(versionStr).Parse()
					if err != nil {
						return nil, fmt.Errorf("Error parsing version for provider %q: %s", name, err)
					}
				}

				var alias string
				if a := listVal.Filter("alias"); len(a.Items) > 0 {
					err := hcl1.DecodeObject(&alias, a.Items[0].Val)
					if err != nil {
						return nil, fmt.Errorf("Error reading alias for provider %q: %s", name, err)
					}
				}

				inst := moduledeps.ProviderInstance(name)
				if alias != "" {
					inst = moduledeps.ProviderInstance(name + "." + alias)
				}
				log.Printf("[TRACE] Provider block requires provider %q", inst)
				m.Providers[inst] = moduledeps.ProviderDependency{
					Constraints: constraints,
					Reason:      moduledeps.ProviderDependencyExplicit,
				}
			}
		}

		{
			resourceConfigsList := list.Filter("resource")
			dataResourceConfigsList := list.Filter("data")
			// list.Filter annoyingly strips off the key used for matching,
			// so we'll put it back here so we can distinguish our two types
			// of blocks below.
			for _, obj := range resourceConfigsList.Items {
				obj.Keys = append([]*hcl1ast.ObjectKey{
					{Token: hcl1token.Token{Type: hcl1token.IDENT, Text: "resource"}},
				}, obj.Keys...)
			}
			for _, obj := range dataResourceConfigsList.Items {
				obj.Keys = append([]*hcl1ast.ObjectKey{
					{Token: hcl1token.Token{Type: hcl1token.IDENT, Text: "data"}},
				}, obj.Keys...)
			}
			// Now we can merge the two lists together, since we can distinguish
			// them just by their keys[0].
			resourceConfigsList.Items = append(resourceConfigsList.Items, dataResourceConfigsList.Items...)

			resourceObjs := resourceConfigsList.Children()
			for _, resourceObj := range resourceObjs.Items {
				if len(resourceObj.Keys) != 3 {
					return nil, fmt.Errorf("resource or data block has wrong number of labels")
				}
				typeName := resourceObj.Keys[1].Token.Value().(string)
				name := resourceObj.Keys[2].Token.Value().(string)
				rAddr := addrs.Resource{
					Mode: addrs.ManagedResourceMode,
					Type: typeName,
					Name: name,
				}
				if resourceObj.Keys[0].Token.Value() == "data" {
					rAddr.Mode = addrs.DataResourceMode
				}

				var listVal *hcl1ast.ObjectList
				if ot, ok := resourceObj.Val.(*hcl1ast.ObjectType); ok {
					listVal = ot.List
				} else {
					return nil, fmt.Errorf("config for %q must be a block", rAddr)
				}

				if o := listVal.Filter("count"); len(o.Items) > 0 {
					ret.ResourceHasCount[rAddr] = true
				} else {
					ret.ResourceHasCount[rAddr] = false
				}

				var providerKey string
				if o := listVal.Filter("provider"); len(o.Items) > 0 {
					err := hcl1.DecodeObject(&providerKey, o.Items[0].Val)
					if err != nil {
						return nil, fmt.Errorf("Error reading provider for resource %s: %s", rAddr, err)
					}
				}

				if providerKey == "" {
					providerKey = rAddr.DefaultProviderConfig().StringCompact()
				}

				inst := moduledeps.ProviderInstance(providerKey)
				log.Printf("[TRACE] Resource block for %s requires provider %q", rAddr, inst)
				if _, exists := m.Providers[inst]; !exists {
					m.Providers[inst] = moduledeps.ProviderDependency{
						Reason: moduledeps.ProviderDependencyImplicit,
					}
				}
				ret.ResourceProviderType[rAddr] = inst.Type()
			}
		}

		if variablesList := list.Filter("variable"); len(variablesList.Items) > 0 {
			variableObjs := variablesList.Children()
			for _, variableObj := range variableObjs.Items {
				if len(variableObj.Keys) != 1 {
					return nil, fmt.Errorf("variable block has wrong number of labels")
				}
				name := variableObj.Keys[0].Token.Value().(string)

				var listVal *hcl1ast.ObjectList
				if ot, ok := variableObj.Val.(*hcl1ast.ObjectType); ok {
					listVal = ot.List
				} else {
					return nil, fmt.Errorf("variable %q: must be a block", name)
				}

				var typeStr string
				if a := listVal.Filter("type"); len(a.Items) > 0 {
					err := hcl1.DecodeObject(&typeStr, a.Items[0].Val)
					if err != nil {
						return nil, fmt.Errorf("Error reading type for variable %q: %s", name, err)
					}
				} else if a := listVal.Filter("default"); len(a.Items) > 0 {
					switch a.Items[0].Val.(type) {
					case *hcl1ast.ObjectType:
						typeStr = "map"
					case *hcl1ast.ListType:
						typeStr = "list"
					default:
						typeStr = "string"
					}
				} else {
					typeStr = "string"
				}

				ret.VariableTypes[name] = strings.TrimSpace(typeStr)
			}
		}
	}

	providerFactories, err := u.Providers.ResolveProviders(m.PluginRequirements())
	if err != nil {
		return nil, fmt.Errorf("error resolving providers: %s", err)
	}

	for name, fn := range providerFactories {
		log.Printf("[TRACE] Fetching schema from provider %q", name)
		provider, err := fn()
		if err != nil {
			return nil, fmt.Errorf("failed to load provider %q: %s", name, err)
		}

		resp := provider.GetSchema()
		if resp.Diagnostics.HasErrors() {
			return nil, resp.Diagnostics.Err()
		}

		schema := &terraform.ProviderSchema{
			Provider:      resp.Provider.Block,
			ResourceTypes: map[string]*configschema.Block{},
			DataSources:   map[string]*configschema.Block{},
		}
		for t, s := range resp.ResourceTypes {
			schema.ResourceTypes[t] = s.Block
		}
		for t, s := range resp.DataSources {
			schema.DataSources[t] = s.Block
		}
		ret.ProviderSchemas[name] = schema
	}

	// TODO: Also ProvisionerSchemas

	return ret, nil
}
