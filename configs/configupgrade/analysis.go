package configupgrade

import (
	"fmt"

	hcl1 "github.com/hashicorp/hcl"
	hcl1ast "github.com/hashicorp/hcl/hcl/ast"
	hcl1parser "github.com/hashicorp/hcl/hcl/parser"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config/configschema"
	"github.com/hashicorp/terraform/moduledeps"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/terraform"
)

// analysis is a container for the various different information gathered
// by ModuleSources.analyze.
type analysis struct {
	ProviderSchemas      map[string]*terraform.ProviderSchema
	ProvisionerSchemas   map[string]*configschema.Block
	ResourceProviderType map[addrs.Resource]string
	ResourceHasCount     map[addrs.Resource]bool
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

		f, err := hcl1parser.Parse(src)
		if err != nil {
			// If we encounter a syntax error then we'll just skip for now
			// and assume that we'll catch this again when we do the upgrade.
			// If not, we'll break the upgrade step of renaming .tf files to
			// .tf.json if they seem to be JSON syntax.
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
				m.Providers[inst] = moduledeps.ProviderDependency{
					Constraints: constraints,
					Reason:      moduledeps.ProviderDependencyExplicit,
				}
			}
		}

		{
			// For our purposes here we don't need to distinguish "resource"
			// and "data" blocks -- provider references are the same for
			// both of them -- so we'll just merge them together into a
			// single list and iterate it.
			resourceConfigsList := list.Filter("resource")
			dataResourceConfigsList := list.Filter("data")
			resourceConfigsList.Items = append(resourceConfigsList.Items, dataResourceConfigsList.Items...)

			resourceObjs := resourceConfigsList.Children()
			for _, resourceObj := range resourceObjs.Items {
				if len(resourceObj.Keys) != 2 {
					return nil, fmt.Errorf("resource or data block has wrong number of labels")
				}
				typeName := resourceObj.Keys[0].Token.Value().(string)
				name := resourceObj.Keys[1].Token.Value().(string)
				rAddr := addrs.Resource{
					Mode: addrs.ManagedResourceMode, // not necessarily true, but good enough for our purposes here
					Type: typeName,
					Name: name,
				}

				var listVal *hcl1ast.ObjectList
				if ot, ok := resourceObj.Val.(*hcl1ast.ObjectType); ok {
					listVal = ot.List
				} else {
					return nil, fmt.Errorf("resource %q %q must be a block", typeName, name)
				}

				if o := listVal.Filter("count"); len(o.Items) > 0 {
					ret.ResourceHasCount[rAddr] = true
				}

				var providerKey string
				if o := listVal.Filter("provider"); len(o.Items) > 0 {
					err := hcl1.DecodeObject(&providerKey, o.Items[0].Val)
					if err != nil {
						return nil, fmt.Errorf("Error reading provider for resource %q %q: %s", typeName, name, err)
					}
				}

				if providerKey == "" {
					providerKey = rAddr.DefaultProviderConfig().StringCompact()
				}

				inst := moduledeps.ProviderInstance(providerKey)
				if _, exists := m.Providers[inst]; !exists {
					m.Providers[inst] = moduledeps.ProviderDependency{
						Reason: moduledeps.ProviderDependencyImplicit,
					}
				}
				ret.ResourceProviderType[rAddr] = inst.Type()
			}
		}
	}

	providerFactories, err := u.Providers.ResolveProviders(m.PluginRequirements())
	if err != nil {
		return nil, fmt.Errorf("error resolving providers: %s", err)
	}

	for name, fn := range providerFactories {
		provider, err := fn()
		if err != nil {
			return nil, fmt.Errorf("failed to load provider %q: %s", name, err)
		}

		// The current GetSchema interface is non-ideal. We're going to make
		// this much simpler in the new interface, but we'll need to shim
		// it for now.
		resourceTypes := provider.Resources()
		dataSources := provider.DataSources()
		var resourceTypeNames, dataSourceNames []string
		for _, t := range resourceTypes {
			resourceTypeNames = append(resourceTypeNames, t.Name)
		}
		for _, t := range dataSources {
			dataSourceNames = append(dataSourceNames, t.Name)
		}
		schema, err := provider.GetSchema(&terraform.ProviderSchemaRequest{
			DataSources:   dataSourceNames,
			ResourceTypes: resourceTypeNames,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get schema from provider %q: %s", name, err)
		}

		ret.ProviderSchemas[name] = schema
	}

	// TODO: Also ProvisionerSchemas

	return ret, nil
}
