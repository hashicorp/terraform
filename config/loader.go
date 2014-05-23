package config

import (
	"fmt"
)

// Load loads the Terraform configuration from a given file.
func Load(path string) (*Config, error) {
	importTree, err := loadTree(path)
	if err != nil {
		return nil, err
	}

	configTree, err := importTree.ConfigTree()
	if err != nil {
		return nil, err
	}

	return configTree.Flatten()
}


type configurable interface {
	Config() (*Config, error)
}

type importTree struct {
	Path     string
	Raw      configurable
	Children []*importTree
}

type fileLoaderFunc func(path string) (configurable, []string, error)

func loadTree(root string) (*importTree, error) {
	c, imps, err := loadFileLibucl(root)
	if err != nil {
		return nil, err
	}

	children := make([]*importTree, len(imps))
	for i, imp := range imps {
		t, err := loadTree(imp)
		if err != nil {
			return nil, err
		}

		children[i] = t
	}

	return &importTree{
		Path:     root,
		Raw:      c,
		Children: children,
	}, nil
}

func (t *importTree) ConfigTree() (*configTree, error) {
	config, err := t.Raw.Config()
	if err != nil {
		return nil, fmt.Errorf(
			"Error loading %s: %s",
			t.Path,
			err)
	}

	// Build our result
	result := &configTree{
		Path:   t.Path,
		Config: config,
	}

	// TODO: Follow children and load them

	return result, nil
}
