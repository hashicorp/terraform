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

// AttachConfigTransformer goes through the graph and attaches configuration
// structures to nodes that implement the interfaces above.
//
// The attached configuration structures are directly from the configuration.
// If they're going to be modified, a copy should be made.
type AttachConfigTransformer struct {
	Module *module.Tree // Module is the root module for the config
}

func (t *AttachConfigTransformer) Transform(g *Graph) error {
	if err := t.attachProviders(g); err != nil {
		return err
	}

	return nil
}

func (t *AttachConfigTransformer) attachProviders(g *Graph) error {
	// Go through and find GraphNodeAttachProvider
	for _, v := range g.Vertices() {
		// Only care about GraphNodeAttachProvider implementations
		apn, ok := v.(GraphNodeAttachProvider)
		if !ok {
			continue
		}

		// TODO: aliases?

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
			if p.Name == name {
				log.Printf("[TRACE] Attaching provider config: %#v", p)
				apn.AttachProvider(p)
				break
			}
		}
	}

	return nil
}
