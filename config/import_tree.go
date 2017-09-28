package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// configurable is an interface that must be implemented by any configuration
// formats of Terraform in order to return a *Config.
type configurable interface {
	Config() (*Config, error)
}

// importTree is the result of the first-pass load of the configuration
// files. It is a tree of raw configurables and then any children (their
// imports).
//
// An importTree can be turned into a configTree.
type importTree struct {
	Path     string
	Raw      configurable
	Children []*importTree
}

// This is the function type that must be implemented by the configuration
// file loader to turn a single file into a configurable and any additional
// imports.
type fileLoaderFunc func(path string) (configurable, []string, error)

// Set this to a non-empty value at link time to enable the HCL2 experiment.
// This is not currently enabled for release builds.
//
// For example:
//    go install -ldflags="-X github.com/hashicorp/terraform/config.enableHCL2Experiment=true" github.com/hashicorp/terraform
var enableHCL2Experiment = ""

// loadTree takes a single file and loads the entire importTree for that
// file. This function detects what kind of configuration file it is an
// executes the proper fileLoaderFunc.
func loadTree(root string) (*importTree, error) {
	var f fileLoaderFunc

	// HCL2 experiment is currently activated at build time via the linker.
	// See the comment on this variable for more information.
	if enableHCL2Experiment == "" {
		// Main-line behavior: always use the original HCL parser
		switch ext(root) {
		case ".tf", ".tf.json":
			f = loadFileHcl
		default:
		}
	} else {
		// Experimental behavior: use the HCL2 parser if the opt-in comment
		// is present.
		switch ext(root) {
		case ".tf":
			// We need to sniff the file for the opt-in comment line to decide
			// if the file is participating in the HCL2 experiment.
			cf, err := os.Open(root)
			if err != nil {
				return nil, err
			}
			sc := bufio.NewScanner(cf)
			for sc.Scan() {
				if sc.Text() == "#terraform:hcl2" {
					f = globalHCL2Loader.loadFile
				}
			}
			if f == nil {
				f = loadFileHcl
			}
		case ".tf.json":
			f = loadFileHcl
		default:
		}
	}

	if f == nil {
		return nil, fmt.Errorf(
			"%s: unknown configuration format. Use '.tf' or '.tf.json' extension",
			root)
	}

	c, imps, err := f(root)
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

// Close releases any resources we might be holding open for the importTree.
//
// This can safely be called even while ConfigTree results are alive. The
// importTree is not bound to these.
func (t *importTree) Close() error {
	if c, ok := t.Raw.(io.Closer); ok {
		c.Close()
	}
	for _, ct := range t.Children {
		ct.Close()
	}

	return nil
}

// ConfigTree traverses the importTree and turns each node into a *Config
// object, ultimately returning a *configTree.
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

	// Build the config trees for the children
	result.Children = make([]*configTree, len(t.Children))
	for i, ct := range t.Children {
		t, err := ct.ConfigTree()
		if err != nil {
			return nil, err
		}

		result.Children[i] = t
	}

	return result, nil
}
