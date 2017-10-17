package module

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	getter "github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/config"
)

// RootName is the name of the root tree.
const RootName = "root"

// Tree represents the module import tree of configurations.
//
// This Tree structure can be used to get (download) new modules, load
// all the modules without getting, flatten the tree into something
// Terraform can use, etc.
type Tree struct {
	name     string
	config   *config.Config
	children map[string]*Tree
	path     []string
	lock     sync.RWMutex
}

// NewTree returns a new Tree for the given config structure.
func NewTree(name string, c *config.Config) *Tree {
	return &Tree{config: c, name: name}
}

// NewEmptyTree returns a new tree that is empty (contains no configuration).
func NewEmptyTree() *Tree {
	t := &Tree{config: &config.Config{}}

	// We do this dummy load so that the tree is marked as "loaded". It
	// should never fail because this is just about a no-op. If it does fail
	// we panic so we can know its a bug.
	if err := t.Load(nil, GetModeGet); err != nil {
		panic(err)
	}

	return t
}

// NewTreeModule is like NewTree except it parses the configuration in
// the directory and gives it a specific name. Use a blank name "" to specify
// the root module.
func NewTreeModule(name, dir string) (*Tree, error) {
	c, err := config.LoadDir(dir)
	if err != nil {
		return nil, err
	}

	return NewTree(name, c), nil
}

// Config returns the configuration for this module.
func (t *Tree) Config() *config.Config {
	return t.config
}

// Child returns the child with the given path (by name).
func (t *Tree) Child(path []string) *Tree {
	if t == nil {
		return nil
	}

	if len(path) == 0 {
		return t
	}

	c := t.Children()[path[0]]
	if c == nil {
		return nil
	}

	return c.Child(path[1:])
}

// Children returns the children of this tree (the modules that are
// imported by this root).
//
// This will only return a non-nil value after Load is called.
func (t *Tree) Children() map[string]*Tree {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.children
}

// DeepEach calls the provided callback for the receiver and then all of
// its descendents in the tree, allowing an operation to be performed on
// all modules in the tree.
//
// Parents will be visited before their children but otherwise the order is
// not defined.
func (t *Tree) DeepEach(cb func(*Tree)) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	t.deepEach(cb)
}

func (t *Tree) deepEach(cb func(*Tree)) {
	cb(t)
	for _, c := range t.children {
		c.deepEach(cb)
	}
}

// Loaded says whether or not this tree has been loaded or not yet.
func (t *Tree) Loaded() bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.children != nil
}

// Modules returns the list of modules that this tree imports.
//
// This is only the imports of _this_ level of the tree. To retrieve the
// full nested imports, you'll have to traverse the tree.
func (t *Tree) Modules() []*Module {
	result := make([]*Module, len(t.config.Modules))
	for i, m := range t.config.Modules {
		result[i] = &Module{
			Name:      m.Name,
			Version:   m.Version,
			Source:    m.Source,
			Providers: m.Providers,
		}
	}

	return result
}

// Name returns the name of the tree. This will be "<root>" for the root
// tree and then the module name given for any children.
func (t *Tree) Name() string {
	if t.name == "" {
		return RootName
	}

	return t.name
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
func (t *Tree) Load(s getter.Storage, mode GetMode) error {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Reset the children if we have any
	t.children = nil

	modules := t.Modules()

	children := make(map[string]*Tree)

	// Go through all the modules and get the directory for them.
	for _, m := range modules {

		if _, ok := children[m.Name]; ok {
			return fmt.Errorf(
				"module %s: duplicated. module names must be unique", m.Name)
		}

		// Determine the path to this child
		path := make([]string, len(t.path), len(t.path)+1)
		copy(path, t.path)
		path = append(path, m.Name)

		// The key is the string that will be hashed to uniquely id the Source.
		// The leading digit can be incremented to force re-fetch all existing
		// modules.
		key := fmt.Sprintf("0.root.%s-%s", strings.Join(path, "."), m.Source)

		log.Printf("[TRACE] module source: %q", m.Source)
		// Split out the subdir if we have one.
		// Terraform keeps the entire requested tree for now, so that modules can
		// reference sibling modules from the same archive or repo.
		source, subDir := getter.SourceDirSubdir(m.Source)

		// First check if we we need to download anything.
		// This is also checked by the getter.Storage implementation, but we
		// want to be able to short-circuit the detection as well, since some
		// detectors may need to make external calls.
		dir, found, err := s.Dir(key)
		if err != nil {
			return err
		}

		// looks like we already have it
		// In order to load the Tree we need to find out if there was another
		// subDir stored from discovery.
		if found && mode != GetModeUpdate {
			subDir, err := t.getSubdir(dir)
			if err != nil {
				// If there's a problem with the subdir record, we'll let the
				// recordSubdir method fix it up.  Any other errors filesystem
				// errors will turn up again below.
				log.Println("[WARN] error reading subdir record:", err)
			} else {
				dir := filepath.Join(dir, subDir)
				// Load the configurations.Dir(source)
				children[m.Name], err = NewTreeModule(m.Name, dir)
				if err != nil {
					return fmt.Errorf("module %s: %s", m.Name, err)
				}
				// Set the path of this child
				children[m.Name].path = path
				continue
			}
		}

		log.Printf("[TRACE] module source: %q", source)

		source, err = getter.Detect(source, t.config.Dir, detectors)
		if err != nil {
			return fmt.Errorf("module %s: %s", m.Name, err)
		}

		log.Printf("[TRACE] detected module source %q", source)

		// Check if the detector introduced something new.
		// For example, the registry always adds a subdir of `//*`,
		// indicating that we need to strip off the first component from the
		// tar archive, though we may not yet know what it is called.
		//
		// TODO: This can cause us to lose the previously detected subdir. It
		// was never an issue before, since none of the supported detectors
		// previously had this behavior, but we may want to add this ability to
		// registry modules.
		source, subDir2 := getter.SourceDirSubdir(source)
		if subDir2 != "" {
			subDir = subDir2
		}

		log.Printf("[TRACE] getting module source %q", source)

		dir, ok, err := getStorage(s, key, source, mode)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf(
				"module %s: not found, may need to be downloaded using 'terraform get'", m.Name)
		}

		// expand and record the subDir for later
		if subDir != "" {
			fullDir, err := getter.SubdirGlob(dir, subDir)
			if err != nil {
				return err
			}

			// +1 to account for the pathsep
			if len(dir)+1 > len(fullDir) {
				return fmt.Errorf("invalid module storage path %q", fullDir)
			}

			subDir = fullDir[len(dir)+1:]

			if err := t.recordSubdir(dir, subDir); err != nil {
				return err
			}
			dir = fullDir
		}

		// Load the configurations.Dir(source)
		children[m.Name], err = NewTreeModule(m.Name, dir)
		if err != nil {
			return fmt.Errorf("module %s: %s", m.Name, err)
		}
		// Set the path of this child
		children[m.Name].path = path
	}

	// Go through all the children and load them.
	for _, c := range children {
		if err := c.Load(s, mode); err != nil {
			return err
		}
	}

	// Set our tree up
	t.children = children

	// if we're the root module, we can now set the provider inheritance
	if len(t.path) == 0 {
		t.inheritProviderConfigs(nil)
	}

	return nil
}

// Once the tree is loaded, we can resolve all provider config inheritance.
//
// This moves the full responsibility of inheritance to the config loader,
// simplifying locating provider configuration during graph evaluation.
// The algorithm is much simpler now too. If there is a provider block without
// a config, we look in the parent's Module block for a provider, and fetch
// that provider's configuration. If that doesn't exist, we assume a default
// empty config. Implicit providers can still inherit their config all the way
// up from the root, so we walk up the tree and copy the first matching
// provider into the module.
func (t *Tree) inheritProviderConfigs(stack []*Tree) {
	stack = append(stack, t)
	for _, c := range t.children {
		c.inheritProviderConfigs(stack)
	}

	providers := make(map[string]*config.ProviderConfig)
	missingProviders := make(map[string]bool)

	for _, p := range t.config.ProviderConfigs {
		providers[p.FullName()] = p
	}

	for _, r := range t.config.Resources {
		p := r.ProviderFullName()
		if _, ok := providers[p]; !(ok || strings.Contains(p, ".")) {
			missingProviders[p] = true
		}
	}

	// Search for implicit provider configs
	// This adds an empty config is no inherited config is found, so that
	// there is always a provider config present.
	// This is done in the root module as well, just to set the providers.
	for missing := range missingProviders {
		// first create an empty provider config
		pc := &config.ProviderConfig{
			Name: missing,
		}

		// walk up the stack looking for matching providers
		for i := len(stack) - 2; i >= 0; i-- {
			pt := stack[i]
			var parentProvider *config.ProviderConfig
			for _, p := range pt.config.ProviderConfigs {
				if p.FullName() == missing {
					parentProvider = p
					break
				}
			}

			if parentProvider == nil {
				continue
			}

			pc.Path = pt.Path()
			pc.Path = append([]string{RootName}, pt.path...)
			pc.RawConfig = parentProvider.RawConfig
			log.Printf("[TRACE] provider %q inheriting config from %q",
				strings.Join(append(t.Path(), pc.FullName()), "."),
				strings.Join(append(pt.Path(), parentProvider.FullName()), "."),
			)
			break
		}

		// always set a provider config
		if pc.RawConfig == nil {
			pc.RawConfig, _ = config.NewRawConfig(map[string]interface{}{})
		}

		t.config.ProviderConfigs = append(t.config.ProviderConfigs, pc)
	}

	// After allowing the empty implicit configs to be created in root, there's nothing left to inherit
	if len(stack) == 1 {
		return
	}

	// get our parent's module config block
	parent := stack[len(stack)-2]
	var parentModule *config.Module
	for _, m := range parent.config.Modules {
		if m.Name == t.name {
			parentModule = m
			break
		}
	}

	if parentModule == nil {
		panic("can't be a module without a parent module config")
	}

	// now look for providers that need a config
	for p, pc := range providers {
		if len(pc.RawConfig.RawMap()) > 0 {
			log.Printf("[TRACE] provider %q has a config, continuing", p)
			continue
		}

		// this provider has no config yet, check for one being passed in
		parentProviderName, ok := parentModule.Providers[p]
		if !ok {
			continue
		}

		var parentProvider *config.ProviderConfig
		// there's a config for us in the parent module
		for _, pp := range parent.config.ProviderConfigs {
			if pp.FullName() == parentProviderName {
				parentProvider = pp
				break
			}
		}

		if parentProvider == nil {
			// no config found, assume defaults
			continue
		}

		// Copy it in, but set an interpolation Scope.
		// An interpolation Scope always need to have "root"
		pc.Path = append([]string{RootName}, parent.path...)
		pc.RawConfig = parentProvider.RawConfig
		log.Printf("[TRACE] provider %q inheriting config from %q",
			strings.Join(append(t.Path(), pc.FullName()), "."),
			strings.Join(append(parent.Path(), parentProvider.FullName()), "."),
		)
	}

}

func subdirRecordsPath(dir string) string {
	const filename = "module-subdir.json"
	// Get the parent directory.
	// The current FolderStorage implementation needed to be able to create
	// this directory, so we can be reasonably certain we can use it.
	parent := filepath.Dir(filepath.Clean(dir))
	return filepath.Join(parent, filename)
}

// unmarshal the records file in the parent directory. Always returns a valid map.
func loadSubdirRecords(dir string) (map[string]string, error) {
	records := map[string]string{}

	recordsPath := subdirRecordsPath(dir)
	data, err := ioutil.ReadFile(recordsPath)
	if err != nil && !os.IsNotExist(err) {
		return records, err
	}

	if len(data) == 0 {
		return records, nil
	}

	if err := json.Unmarshal(data, &records); err != nil {
		return records, err
	}
	return records, nil
}

func (t *Tree) getSubdir(dir string) (string, error) {
	records, err := loadSubdirRecords(dir)
	if err != nil {
		return "", err
	}

	return records[dir], nil
}

// Mark the location of a detected subdir in a top-level file so we
// can skip detection when not updating the module.
func (t *Tree) recordSubdir(dir, subdir string) error {
	records, err := loadSubdirRecords(dir)
	if err != nil {
		// if there was a problem with the file, we will attempt to write a new
		// one. Any non-data related error should surface there.
		log.Printf("[WARN] error reading subdir records: %s", err)
	}

	records[dir] = subdir

	js, err := json.Marshal(records)
	if err != nil {
		return err
	}

	recordsPath := subdirRecordsPath(dir)
	return ioutil.WriteFile(recordsPath, js, 0644)
}

// Path is the full path to this tree.
func (t *Tree) Path() []string {
	return t.path
}

// String gives a nice output to describe the tree.
func (t *Tree) String() string {
	var result bytes.Buffer
	path := strings.Join(t.path, ", ")
	if path != "" {
		path = fmt.Sprintf(" (path: %s)", path)
	}
	result.WriteString(t.Name() + path + "\n")

	cs := t.Children()
	if cs == nil {
		result.WriteString("  not loaded")
	} else {
		// Go through each child and get its string value, then indent it
		// by two.
		for _, c := range cs {
			r := strings.NewReader(c.String())
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				result.WriteString("  ")
				result.WriteString(scanner.Text())
				result.WriteString("\n")
			}
		}
	}

	return result.String()
}

// Validate does semantic checks on the entire tree of configurations.
//
// This will call the respective config.Config.Validate() functions as well
// as verifying things such as parameters/outputs between the various modules.
//
// Load must be called prior to calling Validate or an error will be returned.
func (t *Tree) Validate() error {
	if !t.Loaded() {
		return fmt.Errorf("tree must be loaded before calling Validate")
	}

	// If something goes wrong, here is our error template
	newErr := &treeError{Name: []string{t.Name()}}

	// Terraform core does not handle root module children named "root".
	// We plan to fix this in the future but this bug was brought up in
	// the middle of a release and we don't want to introduce wide-sweeping
	// changes at that time.
	if len(t.path) == 1 && t.name == "root" {
		return fmt.Errorf("root module cannot contain module named 'root'")
	}

	// Validate our configuration first.
	if err := t.config.Validate(); err != nil {
		newErr.Add(err)
	}

	// If we're the root, we do extra validation. This validation usually
	// requires the entire tree (since children don't have parent pointers).
	if len(t.path) == 0 {
		if err := t.validateProviderAlias(); err != nil {
			newErr.Add(err)
		}
	}

	// Get the child trees
	children := t.Children()

	// Validate all our children
	for _, c := range children {
		err := c.Validate()
		if err == nil {
			continue
		}

		verr, ok := err.(*treeError)
		if !ok {
			// Unknown error, just return...
			return err
		}

		// Append ourselves to the error and then return
		verr.Name = append(verr.Name, t.Name())
		newErr.AddChild(verr)
	}

	// Go over all the modules and verify that any parameters are valid
	// variables into the module in question.
	for _, m := range t.config.Modules {
		tree, ok := children[m.Name]
		if !ok {
			// This should never happen because Load watches us
			panic("module not found in children: " + m.Name)
		}

		// Build the variables that the module defines
		requiredMap := make(map[string]struct{})
		varMap := make(map[string]struct{})
		for _, v := range tree.config.Variables {
			varMap[v.Name] = struct{}{}

			if v.Required() {
				requiredMap[v.Name] = struct{}{}
			}
		}

		// Compare to the keys in our raw config for the module
		for k, _ := range m.RawConfig.Raw {
			if _, ok := varMap[k]; !ok {
				newErr.Add(fmt.Errorf(
					"module %s: %s is not a valid parameter",
					m.Name, k))
			}

			// Remove the required
			delete(requiredMap, k)
		}

		// If we have any required left over, they aren't set.
		for k, _ := range requiredMap {
			newErr.Add(fmt.Errorf(
				"module %s: required variable %q not set",
				m.Name, k))
		}
	}

	// Go over all the variables used and make sure that any module
	// variables represent outputs properly.
	for source, vs := range t.config.InterpolatedVariables() {
		for _, v := range vs {
			mv, ok := v.(*config.ModuleVariable)
			if !ok {
				continue
			}

			tree, ok := children[mv.Name]
			if !ok {
				newErr.Add(fmt.Errorf(
					"%s: undefined module referenced %s",
					source, mv.Name))
				continue
			}

			found := false
			for _, o := range tree.config.Outputs {
				if o.Name == mv.Field {
					found = true
					break
				}
			}
			if !found {
				newErr.Add(fmt.Errorf(
					"%s: %s is not a valid output for module %s",
					source, mv.Field, mv.Name))
			}
		}
	}

	return newErr.ErrOrNil()
}

// treeError is an error use by Tree.Validate to accumulates all
// validation errors.
type treeError struct {
	Name     []string
	Errs     []error
	Children []*treeError
}

func (e *treeError) Add(err error) {
	e.Errs = append(e.Errs, err)
}

func (e *treeError) AddChild(err *treeError) {
	e.Children = append(e.Children, err)
}

func (e *treeError) ErrOrNil() error {
	if len(e.Errs) > 0 || len(e.Children) > 0 {
		return e
	}
	return nil
}

func (e *treeError) Error() string {
	name := strings.Join(e.Name, ".")
	var out bytes.Buffer
	fmt.Fprintf(&out, "module %s: ", name)

	if len(e.Errs) == 1 {
		// single like error
		out.WriteString(e.Errs[0].Error())
	} else {
		// multi-line error
		for _, err := range e.Errs {
			fmt.Fprintf(&out, "\n    %s", err)
		}
	}

	if len(e.Children) > 0 {
		// start the next error on a new line
		out.WriteString("\n  ")
	}
	for _, child := range e.Children {
		out.WriteString(child.Error())
	}

	return out.String()
}
