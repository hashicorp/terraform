package terraform

import (
	"log"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
)

// GraphNodeAttachProvider is an interface that must be implemented by nodes
// that want provider configurations attached.
type GraphNodeAttachProvider interface {
	// Must be implemented to determine the path for the configuration
	GraphNodeSubPath

	// ProviderName with no module prefix. Example: "aws".
	ProviderName() string

	// Sets the configuration
	AttachProvider(*config.ProviderConfig)
}

// AttachProviderConfigTransformer goes through the graph and attaches
// provider configuration structures to nodes that implement the interfaces
// above.
//
// The attached configuration structures are directly from the configuration.
// If they're going to be modified, a copy should be made.
type AttachProviderConfigTransformer struct {
	Module *module.Tree // Module is the root module for the config
}

func (t *AttachProviderConfigTransformer) Transform(g *Graph) error {
	if err := t.attachProviders(g); err != nil {
		return err
	}

	return nil
}

func (t *AttachProviderConfigTransformer) attachProviders(g *Graph) error {
	// Go through and find GraphNodeAttachProvider
	for _, v := range g.Vertices() {
		// Only care about GraphNodeAttachProvider implementations
		apn, ok := v.(GraphNodeAttachProvider)
		if !ok {
			continue
		}

		// Determine what we're looking for
		path := normalizeModulePath(apn.Path())
		path = path[1:]
		name := apn.ProviderName()
		log.Printf("[TRACE] Attach provider request: %#v %s", path, name)

		// Get the configuration.
		tree := t.Module.Child(path)
		if tree == nil {
			continue
		}

		// Go through the provider configs to find the matching config
		for _, p := range tree.Config().ProviderConfigs {
			// Build the name, which is "name.alias" if an alias exists
			current := p.Name
			if p.Alias != "" {
				current += "." + p.Alias
			}

			// If the configs match then attach!
			if current == name {
				log.Printf("[TRACE] Attaching provider config: %#v", p)
				apn.AttachProvider(p)
				break
			}
		}
	}

	return nil
}
