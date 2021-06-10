package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/configs"
)

type ActionTransformer struct {
	// Config is the module to add actions from.
	Config *configs.Config
}

var _ GraphTransformer = (*ActionTransformer)(nil)

func (t *ActionTransformer) Transform(g *Graph) error {
	// If no configuration is available, we don't do anything
	if t.Config == nil {
		return nil
	}

	// Start the transformation process
	return t.transform(g, t.Config)
}

func (t *ActionTransformer) transform(g *Graph, config *configs.Config) error {
	// If no config, do nothing
	if config == nil {
		return nil
	}

	// Add actions from this module
	if err := t.transformSingle(g, config); err != nil {
		return err
	}

	// Transform all the children
	for _, c := range config.Children {
		if err := t.transform(g, c); err != nil {
			return err
		}
	}

	return nil
}

func (t *ActionTransformer) transformSingle(g *Graph, config *configs.Config) error {
	path := config.Path
	module := config.Module
	log.Printf("[TRACE] ActionTransformer: Starting for path: %v", path)

	for _, ma := range module.MovedActions {
		g.Add(&nodeExpandPlannableMovedAction{
			Module: path,
			Config: ma,
		})
	}

	return nil
}
