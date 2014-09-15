package module

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/config"
)

// Tree represents the module import tree of configurations.
//
// This Tree structure can be used to get (download) new modules, load
// all the modules without getting, flatten the tree into something
// Terraform can use, etc.
type Tree struct {
	config   *config.Config
	children []*Tree
	lock     sync.Mutex
}

// GetMode is an enum that describes how modules are loaded.
//
// GetModeLoad says that modules will not be downloaded or updated, they will
// only be loaded from the storage.
//
// GetModeGet says that modules can be initially downloaded if they don't
// exist, but otherwise to just load from the current version in storage.
//
// GetModeUpdate says that modules should be checked for updates and
// downloaded prior to loading. If there are no updates, we load the version
// from disk, otherwise we download first and then load.
type GetMode byte

const (
	GetModeNone GetMode = iota
	GetModeGet
	GetModeUpdate
)

// NewTree returns a new Tree for the given config structure.
func NewTree(c *config.Config) *Tree {
	return &Tree{config: c}
}

// Children returns the children of this tree (the modules that are
// imported by this root).
//
// This will only return a non-nil value after Load is called.
func (t *Tree) Children() []*Tree {
	return nil
}

// Flatten takes the entire module tree and flattens it into a single
// namespace in *config.Config with no module imports.
//
// Validate is called here implicitly, since it is important that semantic
// checks pass before flattening the configuration. Otherwise, encapsulation
// breaks in horrible ways and the errors that come out the other side
// will be surprising.
func (t *Tree) Flatten() (*config.Config, error) {
	return nil, nil
}

// Modules returns the list of modules that this tree imports.
//
// This is only the imports of _this_ level of the tree. To retrieve the
// full nested imports, you'll have to traverse the tree.
func (t *Tree) Modules() []*Module {
	result := make([]*Module, len(t.config.Modules))
	for i, m := range t.config.Modules {
		result[i] = &Module{
			Name:   m.Name,
			Source: m.Source,
		}
	}

	return result
}

// Load loads the configuration of the entire tree.
//
// The parameters are used to tell the tree where to find modules and
// whether it can download/update modules along the way.
//
// Calling this multiple times will reload the tree.
//
// Various semantic-like checks are made along the way of loading since
// module trees inherently require the configuration to be in a reasonably
// sane state: no circular dependencies, proper module sources, etc. A full
// suite of validations can be done by running Validate (after loading).
func (t *Tree) Load(s Storage, mode GetMode) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Reset the children if we have any
	t.children = nil

	modules := t.Modules()
	children := make([]*Tree, len(modules))

	// Go through all the modules and get the directory for them.
	update := mode == GetModeUpdate
	for i, m := range modules {
		source, err := Detect(m.Source, t.config.Dir)
		if err != nil {
			return fmt.Errorf("module %s: %s", m.Name, err)
		}

		if mode > GetModeNone {
			// Get the module since we specified we should
			if err := s.Get(source, update); err != nil {
				return err
			}
		}

		// Get the directory where this module is so we can load it
		dir, ok, err := s.Dir(source)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf(
				"module %s: not found, may need to be downloaded", m.Name)
		}

		// Load the configuration
		c, err := config.LoadDir(dir)
		if err != nil {
			return fmt.Errorf(
				"module %s: %s", m.Name, err)
		}
		children[i] = NewTree(c)
	}

	// Go through all the children and load them.
	for _, c := range children {
		if err := c.Load(s, mode); err != nil {
			return err
		}
	}

	// Set our tree up
	t.children = children

	return nil
}

// Validate does semantic checks on the entire tree of configurations.
//
// This will call the respective config.Config.Validate() functions as well
// as verifying things such as parameters/outputs between the various modules.
func (t *Tree) Validate() error {
	return nil
}
