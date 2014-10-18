package terraform

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/depgraph"
	"github.com/hashicorp/terraform/helper/multierror"
)

// This is a function type used to implement a walker for the resource
// tree internally on the Terraform structure.
type genericWalkFunc func(*walkContext, *Resource) error

// Context represents all the context that Terraform needs in order to
// perform operations on infrastructure. This structure is built using
// ContextOpts and NewContext. See the documentation for those.
//
// Additionally, a context can be created from a Plan using Plan.Context.
type Context struct {
	module         *module.Tree
	diff           *Diff
	hooks          []Hook
	state          *State
	providerConfig map[string]map[string]map[string]interface{}
	providers      map[string]ResourceProviderFactory
	provisioners   map[string]ResourceProvisionerFactory
	variables      map[string]string
	uiInput        UIInput

	parallelSem Semaphore    // Semaphore used to limit parallelism
	l           sync.Mutex   // Lock acquired during any task
	sl          sync.RWMutex // Lock acquired to R/W internal data
	runCh       <-chan struct{}
	sh          *stopHook
}

// ContextOpts are the user-creatable configuration structure to create
// a context with NewContext.
type ContextOpts struct {
	Diff         *Diff
	Hooks        []Hook
	Module       *module.Tree
	Parallelism  int
	State        *State
	Providers    map[string]ResourceProviderFactory
	Provisioners map[string]ResourceProvisionerFactory
	Variables    map[string]string

	UIInput UIInput
}

// InputMode defines what sort of input will be asked for when Input
// is called on Context.
type InputMode byte

const (
	// InputModeVar asks for variables
	InputModeVar InputMode = 1 << iota

	// InputModeProvider asks for provider variables
	InputModeProvider

	// InputModeStd is the standard operating mode and asks for both variables
	// and providers.
	InputModeStd = InputModeVar | InputModeProvider
)

// NewContext creates a new context.
//
// Once a context is created, the pointer values within ContextOpts should
// not be mutated in any way, since the pointers are copied, not the values
// themselves.
func NewContext(opts *ContextOpts) *Context {
	sh := new(stopHook)

	// Copy all the hooks and add our stop hook. We don't append directly
	// to the Config so that we're not modifying that in-place.
	hooks := make([]Hook, len(opts.Hooks)+1)
	copy(hooks, opts.Hooks)
	hooks[len(opts.Hooks)] = sh

	// Make the parallelism channel
	par := opts.Parallelism
	if par == 0 {
		par = 10
	}

	return &Context{
		diff:           opts.Diff,
		hooks:          hooks,
		module:         opts.Module,
		state:          opts.State,
		providerConfig: make(map[string]map[string]map[string]interface{}),
		providers:      opts.Providers,
		provisioners:   opts.Provisioners,
		variables:      opts.Variables,
		uiInput:        opts.UIInput,

		parallelSem: NewSemaphore(par),
		sh:          sh,
	}
}

// Apply applies the changes represented by this context and returns
// the resulting state.
//
// In addition to returning the resulting state, this context is updated
// with the latest state.
func (c *Context) Apply() (*State, error) {
	v := c.acquireRun()
	defer c.releaseRun(v)

	// Set our state right away. No matter what, this IS our new state,
	// even if there is an error below.
	c.state = c.state.deepcopy()
	if c.state == nil {
		c.state = &State{}
	}
	c.state.init()

	// Walk
	log.Printf("[INFO] Apply walk starting")
	err := c.walkContext(walkApply, rootModulePath).Walk()
	log.Printf("[INFO] Apply walk complete")

	// Prune the state so that we have as clean a state as possible
	c.state.prune()

	return c.state, err
}

// Graph returns the graph for this context.
func (c *Context) Graph() (*depgraph.Graph, error) {
	return Graph(&GraphOpts{
		Diff:         c.diff,
		Module:       c.module,
		Providers:    c.providers,
		Provisioners: c.provisioners,
		State:        c.state,
	})
}

// Input asks for input to fill variables and provider configurations.
// This modifies the configuration in-place, so asking for Input twice
// may result in different UI output showing different current values.
func (c *Context) Input(mode InputMode) error {
	v := c.acquireRun()
	defer c.releaseRun(v)

	if mode&InputModeVar != 0 {
		// Walk the variables first for the root module. We walk them in
		// alphabetical order for UX reasons.
		rootConf := c.module.Config()
		names := make([]string, len(rootConf.Variables))
		m := make(map[string]*config.Variable)
		for i, v := range rootConf.Variables {
			names[i] = v.Name
			m[v.Name] = v
		}
		sort.Strings(names)
		for _, n := range names {
			v := m[n]
			switch v.Type() {
			case config.VariableTypeMap:
				continue
			case config.VariableTypeString:
				// Good!
			default:
				panic(fmt.Sprintf("Unknown variable type: %s", v.Type()))
			}

			var defaultString string
			if v.Default != nil {
				defaultString = v.Default.(string)
			}

			// Ask the user for a value for this variable
			var value string
			for {
				var err error
				value, err = c.uiInput.Input(&InputOpts{
					Id:          fmt.Sprintf("var.%s", n),
					Query:       fmt.Sprintf("var.%s", n),
					Default:     defaultString,
					Description: v.Description,
				})
				if err != nil {
					return fmt.Errorf(
						"Error asking for %s: %s", n, err)
				}

				if value == "" && v.Required() {
					// Redo if it is required.
					continue
				}

				if value == "" {
					// No value, just exit the loop. With no value, we just
					// use whatever is currently set in variables.
					break
				}

				break
			}

			if value != "" {
				c.variables[n] = value
			}
		}
	}

	if mode&InputModeProvider != 0 {
		// Create the walk context and walk the inputs, which will gather the
		// inputs for any resource providers.
		wc := c.walkContext(walkInput, rootModulePath)
		wc.Meta = new(walkInputMeta)
		return wc.Walk()
	}

	return nil
}

// Plan generates an execution plan for the given context.
//
// The execution plan encapsulates the context and can be stored
// in order to reinstantiate a context later for Apply.
//
// Plan also updates the diff of this context to be the diff generated
// by the plan, so Apply can be called after.
func (c *Context) Plan(opts *PlanOpts) (*Plan, error) {
	v := c.acquireRun()
	defer c.releaseRun(v)

	p := &Plan{
		Module: c.module,
		Vars:   c.variables,
		State:  c.state,
	}

	wc := c.walkContext(walkInvalid, rootModulePath)
	wc.Meta = p

	if opts != nil && opts.Destroy {
		wc.Operation = walkPlanDestroy
	} else {
		// Set our state to be something temporary. We do this so that
		// the plan can update a fake state so that variables work, then
		// we replace it back with our old state.
		old := c.state
		if old == nil {
			c.state = &State{}
			c.state.init()
		} else {
			c.state = old.deepcopy()
		}
		defer func() {
			c.state = old
		}()

		wc.Operation = walkPlan
	}

	// Walk and run the plan
	err := wc.Walk()

	// Update the diff so that our context is up-to-date
	c.diff = p.Diff

	return p, err
}

// Refresh goes through all the resources in the state and refreshes them
// to their latest state. This will update the state that this context
// works with, along with returning it.
//
// Even in the case an error is returned, the state will be returned and
// will potentially be partially updated.
func (c *Context) Refresh() (*State, error) {
	v := c.acquireRun()
	defer c.releaseRun(v)

	// Update our state
	c.state = c.state.deepcopy()

	// Walk the graph
	err := c.walkContext(walkRefresh, rootModulePath).Walk()

	// Prune the state
	c.state.prune()
	return c.state, err
}

// Stop stops the running task.
//
// Stop will block until the task completes.
func (c *Context) Stop() {
	c.l.Lock()
	ch := c.runCh

	// If we aren't running, then just return
	if ch == nil {
		c.l.Unlock()
		return
	}

	// Tell the hook we want to stop
	c.sh.Stop()

	// Wait for us to stop
	c.l.Unlock()
	<-ch
}

// Validate validates the configuration and returns any warnings or errors.
func (c *Context) Validate() ([]string, []error) {
	var rerr *multierror.Error

	// Validate the configuration itself
	if err := c.module.Validate(); err != nil {
		rerr = multierror.ErrorAppend(rerr, err)
	}

	// This only needs to be done for the root module, since inter-module
	// variables are validated in the module tree.
	if config := c.module.Config(); config != nil {
		// Validate the user variables
		if errs := smcUserVariables(config, c.variables); len(errs) > 0 {
			rerr = multierror.ErrorAppend(rerr, errs...)
		}
	}

	// Validate the entire graph
	walkMeta := new(walkValidateMeta)
	wc := c.walkContext(walkValidate, rootModulePath)
	wc.Meta = walkMeta
	if err := wc.Walk(); err != nil {
		rerr = multierror.ErrorAppend(rerr, err)
	}

	// Flatten the warns/errs so that we get all the module errors as well,
	// then aggregate.
	warns, errs := walkMeta.Flatten()
	if len(errs) > 0 {
		rerr = multierror.ErrorAppend(rerr, errs...)
	}

	errs = nil
	if rerr != nil && len(rerr.Errors) > 0 {
		errs = rerr.Errors
	}

	return warns, errs
}

func (c *Context) acquireRun() chan<- struct{} {
	c.l.Lock()
	defer c.l.Unlock()

	// Wait for no channel to exist
	for c.runCh != nil {
		c.l.Unlock()
		ch := c.runCh
		<-ch
		c.l.Lock()
	}

	ch := make(chan struct{})
	c.runCh = ch
	return ch
}

func (c *Context) releaseRun(ch chan<- struct{}) {
	c.l.Lock()
	defer c.l.Unlock()

	close(ch)
	c.runCh = nil
	c.sh.Reset()
}

func (c *Context) walkContext(op walkOperation, path []string) *walkContext {
	// Get the config structure
	m := c.module
	for _, n := range path[1:] {
		cs := m.Children()
		m = cs[n]
	}
	var conf *config.Config
	if m != nil {
		conf = m.Config()
	}

	// Calculate the default variable values
	defaultVars := make(map[string]string)
	if conf != nil {
		for _, v := range conf.Variables {
			for k, val := range v.DefaultsMap() {
				defaultVars[k] = val
			}
		}
	}

	return &walkContext{
		Context:   c,
		Operation: op,
		Path:      path,
		Variables: c.variables,

		defaultVariables: defaultVars,
	}
}

// walkContext is the context in which a graph walk is done. It stores
// much the same as a Context but works on a specific module.
type walkContext struct {
	Context   *Context
	Meta      interface{}
	Operation walkOperation
	Path      []string
	Variables map[string]string

	defaultVariables map[string]string

	// This is only set manually by subsequent context creations
	// in genericWalkFunc.
	graph *depgraph.Graph
}

// walkOperation is an enum which tells the walkContext what to do.
type walkOperation byte

const (
	walkInvalid walkOperation = iota
	walkInput
	walkApply
	walkPlan
	walkPlanDestroy
	walkRefresh
	walkValidate
)

func (c *walkContext) Walk() error {
	g := c.graph
	if g == nil {
		gopts := &GraphOpts{
			Module:       c.Context.module,
			Providers:    c.Context.providers,
			Provisioners: c.Context.provisioners,
			State:        c.Context.state,
		}
		if c.Operation == walkApply {
			gopts.Diff = c.Context.diff
		}

		var err error
		g, err = Graph(gopts)
		if err != nil {
			return err
		}
	}

	var walkFn depgraph.WalkFunc
	switch c.Operation {
	case walkInput:
		walkFn = c.inputWalkFn()
	case walkApply:
		walkFn = c.applyWalkFn()
	case walkPlan:
		walkFn = c.planWalkFn()
	case walkPlanDestroy:
		walkFn = c.planDestroyWalkFn()
	case walkRefresh:
		walkFn = c.refreshWalkFn()
	case walkValidate:
		walkFn = c.validateWalkFn()
	default:
		panic(fmt.Sprintf("unknown operation: %s", c.Operation))
	}

	if err := g.Walk(walkFn); err != nil {
		return err
	}

	switch c.Operation {
	case walkInput:
		fallthrough
	case walkValidate:
		// Don't calculate outputs
		return nil
	}

	// We did an apply, so we need to calculate the outputs. If we have no
	// outputs, then we're done.
	m := c.Context.module
	for _, n := range c.Path[1:] {
		cs := m.Children()
		m = cs[n]
	}
	if m == nil {
		return nil
	}
	conf := m.Config()
	if len(conf.Outputs) == 0 {
		return nil
	}

	// Likewise, if we have no resources in our state, we're done. This
	// guards against the case that we destroyed.
	mod := c.Context.state.ModuleByPath(c.Path)
	if mod == nil {
		return nil
	}
	if c.Operation == walkApply {
		// On Apply, we prune so that we don't do outputs if we destroyed
		mod.prune()
	}
	if len(mod.Resources) == 0 {
		mod.Outputs = nil
		return nil
	}

	outputs := make(map[string]string)
	for _, o := range conf.Outputs {
		if err := c.computeVars(o.RawConfig, nil); err != nil {
			return err
		}
		vraw := o.RawConfig.Config()["value"]
		if vraw == nil {
			// This likely means that the result of the output is
			// a computed variable.
			if o.RawConfig.Raw["value"] != nil {
				vraw = config.UnknownVariableValue
			}
		}
		if vraw != nil {
			outputs[o.Name] = vraw.(string)
		}
	}

	// Assign the outputs to the root module
	mod.Outputs = outputs

	return nil
}

func (c *walkContext) inputWalkFn() depgraph.WalkFunc {
	meta := c.Meta.(*walkInputMeta)
	meta.Lock()
	if meta.Done == nil {
		meta.Done = make(map[string]struct{})
	}
	meta.Unlock()

	return func(n *depgraph.Noun) error {
		// If it is the root node, ignore
		if n.Name == GraphRootNode {
			return nil
		}

		switch rn := n.Meta.(type) {
		case *GraphNodeModule:
			// Build another walkContext for this module and walk it.
			wc := c.Context.walkContext(c.Operation, rn.Path)

			// Set the graph to specifically walk this subgraph
			wc.graph = rn.Graph

			// Preserve the meta
			wc.Meta = c.Meta

			return wc.Walk()
		case *GraphNodeResource:
			// Resources don't matter for input. Continue.
			return nil
		case *GraphNodeResourceProvider:
			// Acquire the lock the whole time so we only ask for input
			// one at a time.
			meta.Lock()
			defer meta.Unlock()

			// If we already did this provider, then we're done.
			if _, ok := meta.Done[rn.ID]; ok {
				return nil
			}

			// Get the raw configuration because this is what we
			// pass into the API.
			var raw *config.RawConfig
			sharedProvider := rn.Provider
			if sharedProvider.Config != nil {
				raw = sharedProvider.Config.RawConfig
			}
			rc := NewResourceConfig(raw)
			rc.Config = make(map[string]interface{})

			// Wrap the input into a namespace
			input := &PrefixUIInput{
				IdPrefix:    fmt.Sprintf("provider.%s", rn.ID),
				QueryPrefix: fmt.Sprintf("provider.%s.", rn.ID),
				UIInput:     c.Context.uiInput,
			}

			// Go through each provider and capture the input necessary
			// to satisfy it.
			configs := make(map[string]map[string]interface{})
			for k, p := range sharedProvider.Providers {
				newc, err := p.Input(input, rc)
				if err != nil {
					return fmt.Errorf(
						"Error configuring %s: %s", k, err)
				}
				if newc != nil && len(newc.Config) > 0 {
					configs[k] = newc.Config
				}
			}

			// Mark this provider as done
			meta.Done[rn.ID] = struct{}{}

			// Set the configuration
			c.Context.providerConfig[rn.ID] = configs
		}

		return nil
	}
}

func (c *walkContext) applyWalkFn() depgraph.WalkFunc {
	cb := func(c *walkContext, r *Resource) error {
		var err error

		diff := r.Diff
		if diff.Empty() {
			log.Printf("[DEBUG] %s: Diff is empty. Will not apply.", r.Id)
			return nil
		}

		is := r.State
		if is == nil {
			is = new(InstanceState)
		}
		is.init()

		if !diff.Destroy {
			// Since we need the configuration, interpolate the variables
			if err := r.Config.interpolate(c, r); err != nil {
				return err
			}

			diff, err = r.Provider.Diff(r.Info, is, r.Config)
			if err != nil {
				return err
			}

			// This can happen if we aren't actually applying anything
			// except an ID (the "null" provider). It is not really an issue
			// since the Same check later down will catch any real problems.
			if diff == nil {
				diff = new(InstanceDiff)
				diff.init()
			}

			// Delete id from the diff because it is dependent on
			// our internal plan function.
			delete(r.Diff.Attributes, "id")
			delete(diff.Attributes, "id")

			// Verify the diffs are the same
			if !r.Diff.Same(diff) {
				log.Printf(
					"[ERROR] Diffs don't match.\n\nDiff 1: %#v"+
						"\n\nDiff 2: %#v",
					r.Diff, diff)
				return fmt.Errorf(
					"%s: diffs didn't match during apply. This is a "+
						"bug with the resource provider, please report a bug.",
					r.Id)
			}
		}

		// Remove any output values from the diff
		for k, ad := range diff.Attributes {
			if ad.Type == DiffAttrOutput {
				delete(diff.Attributes, k)
			}
		}

		for _, h := range c.Context.hooks {
			handleHook(h.PreApply(r.Info, is, diff))
		}

		// We create a new instance if there was no ID
		// previously or the diff requires re-creating the
		// underlying instance
		createNew := (is.ID == "" && !diff.Destroy) || diff.RequiresNew()

		// With the completed diff, apply!
		log.Printf("[DEBUG] %s: Executing Apply", r.Id)
		is, applyerr := r.Provider.Apply(r.Info, is, diff)

		var errs []error
		if applyerr != nil {
			errs = append(errs, applyerr)
		}

		// Make sure the result is instantiated
		if is == nil {
			is = new(InstanceState)
		}
		is.init()

		// Force the "id" attribute to be our ID
		if is.ID != "" {
			is.Attributes["id"] = is.ID
		}

		for ak, av := range is.Attributes {
			// If the value is the unknown variable value, then it is an error.
			// In this case we record the error and remove it from the state
			if av == config.UnknownVariableValue {
				errs = append(errs, fmt.Errorf(
					"Attribute with unknown value: %s", ak))
				delete(is.Attributes, ak)
			}
		}

		// Set the result state
		r.State = is
		c.persistState(r)

		// Invoke any provisioners we have defined. This is only done
		// if the resource was created, as updates or deletes do not
		// invoke provisioners.
		//
		// Additionally, we need to be careful to not run this if there
		// was an error during the provider apply.
		tainted := false
		if createNew && len(r.Provisioners) > 0 {
			if applyerr == nil {
				// If the apply succeeded, we have to run the provisioners
				for _, h := range c.Context.hooks {
					handleHook(h.PreProvisionResource(r.Info, is))
				}

				if err := c.applyProvisioners(r, is); err != nil {
					errs = append(errs, err)
					tainted = true
				}

				for _, h := range c.Context.hooks {
					handleHook(h.PostProvisionResource(r.Info, is))
				}
			} else {
				// If we failed to create properly and we have provisioners,
				// then we have to mark ourselves as tainted to try again.
				tainted = true
			}
		}

		// If we're tainted then we need to update some flags
		if tainted && r.Flags&FlagTainted == 0 {
			r.Flags &^= FlagPrimary
			r.Flags &^= FlagHasTainted
			r.Flags |= FlagTainted
			r.TaintedIndex = -1
			c.persistState(r)
		}

		for _, h := range c.Context.hooks {
			handleHook(h.PostApply(r.Info, is, applyerr))
		}

		// Determine the new state and update variables
		err = nil
		if len(errs) > 0 {
			err = &multierror.Error{Errors: errs}
		}

		return err
	}

	return c.genericWalkFn(cb)
}

func (c *walkContext) planWalkFn() depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result
	result := c.Meta.(*Plan)
	result.init()

	cb := func(c *walkContext, r *Resource) error {
		if r.Flags&FlagTainted != 0 {
			// We don't diff tainted resources.
			return nil
		}

		var diff *InstanceDiff

		is := r.State

		for _, h := range c.Context.hooks {
			handleHook(h.PreDiff(r.Info, is))
		}

		if r.Flags&FlagOrphan != 0 {
			log.Printf("[DEBUG] %s: Orphan, marking for destroy", r.Id)

			// This is an orphan (no config), so we mark it to be destroyed
			diff = &InstanceDiff{Destroy: true}
		} else {
			// Make sure the configuration is interpolated
			if err := r.Config.interpolate(c, r); err != nil {
				return err
			}

			// Get a diff from the newest state
			log.Printf("[DEBUG] %s: Executing diff", r.Id)
			var err error

			diffIs := is
			if diffIs == nil || r.Flags&FlagHasTainted != 0 {
				// If we're tainted, we pretend to create a new thing.
				diffIs = new(InstanceState)
			}
			diffIs.init()

			diff, err = r.Provider.Diff(r.Info, diffIs, r.Config)
			if err != nil {
				return err
			}
		}

		if diff == nil {
			diff = new(InstanceDiff)
		}

		if r.Flags&FlagHasTainted != 0 {
			// This primary has a tainted resource, so just mark for
			// destroy...
			log.Printf("[DEBUG] %s: Tainted children, marking for destroy", r.Id)
			diff.DestroyTainted = true
		}

		if diff.RequiresNew() && is != nil && is.ID != "" {
			// This will also require a destroy
			diff.Destroy = true
		}

		if diff.RequiresNew() || is == nil || is.ID == "" {
			var oldID string
			if is != nil {
				oldID = is.Attributes["id"]
			}

			// Add diff to compute new ID
			diff.init()
			diff.Attributes["id"] = &ResourceAttrDiff{
				Old:         oldID,
				NewComputed: true,
				RequiresNew: true,
				Type:        DiffAttrOutput,
			}
		}

		if !diff.Empty() {
			log.Printf("[DEBUG] %s: Diff: %#v", r.Id, diff)

			l.Lock()
			md := result.Diff.ModuleByPath(c.Path)
			if md == nil {
				md = result.Diff.AddModule(c.Path)
			}
			md.Resources[r.Id] = diff
			l.Unlock()
		}

		for _, h := range c.Context.hooks {
			handleHook(h.PostDiff(r.Info, diff))
		}

		// Determine the new state and update variables
		if !diff.Empty() {
			is = is.MergeDiff(diff)
		}

		// Set it so that it can be updated
		r.State = is
		c.persistState(r)

		return nil
	}

	return c.genericWalkFn(cb)
}

func (c *walkContext) planDestroyWalkFn() depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result
	result := c.Meta.(*Plan)
	result.init()

	var walkFn depgraph.WalkFunc
	walkFn = func(n *depgraph.Noun) error {
		switch m := n.Meta.(type) {
		case *GraphNodeModule:
			// Build another walkContext for this module and walk it.
			wc := c.Context.walkContext(c.Operation, m.Path)

			// Set the graph to specifically walk this subgraph
			wc.graph = m.Graph

			// Preserve the meta
			wc.Meta = c.Meta

			return wc.Walk()
		case *GraphNodeResource:
			// If we're expanding, then expand the nodes, and then rewalk the graph
			if m.ExpandMode > ResourceExpandNone {
				return c.genericWalkResource(m, walkFn)
			}

			r := m.Resource

			if r.State != nil && r.State.ID != "" {
				log.Printf("[DEBUG] %s: Making for destroy", r.Id)

				l.Lock()
				defer l.Unlock()
				md := result.Diff.ModuleByPath(c.Path)
				if md == nil {
					md = result.Diff.AddModule(c.Path)
				}
				md.Resources[r.Id] = &InstanceDiff{Destroy: true}
			} else {
				log.Printf("[DEBUG] %s: Not marking for destroy, no ID", r.Id)
			}
		}

		return nil
	}

	return walkFn
}

func (c *walkContext) refreshWalkFn() depgraph.WalkFunc {
	cb := func(c *walkContext, r *Resource) error {
		is := r.State

		if is == nil || is.ID == "" {
			log.Printf("[DEBUG] %s: Not refreshing, ID is empty", r.Id)
			return nil
		}

		for _, h := range c.Context.hooks {
			handleHook(h.PreRefresh(r.Info, is))
		}

		is, err := r.Provider.Refresh(r.Info, is)
		if err != nil {
			return err
		}
		if is == nil {
			is = new(InstanceState)
			is.init()
		}

		// Set the updated state
		r.State = is
		c.persistState(r)

		for _, h := range c.Context.hooks {
			handleHook(h.PostRefresh(r.Info, is))
		}

		return nil
	}

	return c.genericWalkFn(cb)
}

func (c *walkContext) validateWalkFn() depgraph.WalkFunc {
	var l sync.Mutex

	meta := c.Meta.(*walkValidateMeta)
	if meta.Children == nil {
		meta.Children = make(map[string]*walkValidateMeta)
	}

	var walkFn depgraph.WalkFunc
	walkFn = func(n *depgraph.Noun) error {
		// If it is the root node, ignore
		if n.Name == GraphRootNode {
			return nil
		}

		switch rn := n.Meta.(type) {
		case *GraphNodeModule:
			// Build another walkContext for this module and walk it.
			wc := c.Context.walkContext(walkValidate, rn.Path)

			// Set the graph to specifically walk this subgraph
			wc.graph = rn.Graph

			// Build the meta parameter. Do this by sharing the Children
			// reference but copying the rest into our own Children list.
			newMeta := new(walkValidateMeta)
			newMeta.Children = meta.Children
			wc.Meta = newMeta

			if err := wc.Walk(); err != nil {
				return err
			}

			newMeta.Children = nil
			meta.Children[strings.Join(rn.Path, ".")] = newMeta
			return nil
		case *GraphNodeResource:
			if rn.Resource == nil {
				panic("resource should never be nil")
			}

			// If we're expanding, then expand the nodes, and then rewalk the graph
			if rn.ExpandMode > ResourceExpandNone {
				// Interpolate the count and verify it is non-negative
				rc := NewResourceConfig(rn.Config.RawCount)
				rc.interpolate(c, rn.Resource)
				if !rc.IsComputed(rn.Config.RawCount.Key) {
					count, err := rn.Config.Count()
					if err == nil {
						if count < 0 {
							err = fmt.Errorf(
								"%s error: count must be positive", rn.Resource.Id)
						}
					}
					if err != nil {
						l.Lock()
						defer l.Unlock()
						meta.Errs = append(meta.Errs, err)
						return nil
					}
				}

				return c.genericWalkResource(rn, walkFn)
			}

			// If it doesn't have a provider, that is a different problem
			if rn.Resource.Provider == nil {
				return nil
			}

			// Don't validate orphans or tainted since they never have a config
			if rn.Resource.Flags&FlagOrphan != 0 {
				return nil
			}
			if rn.Resource.Flags&FlagTainted != 0 {
				return nil
			}

			// If the resouce name doesn't match the name regular
			// expression, show a warning.
			if !config.NameRegexp.Match([]byte(rn.Config.Name)) {
				l.Lock()
				meta.Warns = append(meta.Warns, fmt.Sprintf(
					"%s: module name can only contain letters, numbers, "+
						"dashes, and underscores.\n"+
						"This will be an error in Terraform 0.4",
					rn.Resource.Id))
				l.Unlock()
			}

			// Compute the variables in this resource
			rn.Resource.Config.interpolate(c, rn.Resource)

			log.Printf("[INFO] Validating resource: %s", rn.Resource.Id)
			ws, es := rn.Resource.Provider.ValidateResource(
				rn.Resource.Info.Type, rn.Resource.Config)
			for i, w := range ws {
				ws[i] = fmt.Sprintf("'%s' warning: %s", rn.Resource.Id, w)
			}
			for i, e := range es {
				es[i] = fmt.Errorf("'%s' error: %s", rn.Resource.Id, e)
			}

			l.Lock()
			meta.Warns = append(meta.Warns, ws...)
			meta.Errs = append(meta.Errs, es...)
			l.Unlock()

			for idx, p := range rn.Resource.Provisioners {
				ws, es := p.Provisioner.Validate(p.Config)
				for i, w := range ws {
					ws[i] = fmt.Sprintf("'%s.provisioner.%d' warning: %s", rn.Resource.Id, idx, w)
				}
				for i, e := range es {
					es[i] = fmt.Errorf("'%s.provisioner.%d' error: %s", rn.Resource.Id, idx, e)
				}

				l.Lock()
				meta.Warns = append(meta.Warns, ws...)
				meta.Errs = append(meta.Errs, es...)
				l.Unlock()
			}

		case *GraphNodeResourceProvider:
			sharedProvider := rn.Provider

			// Check if we have an override
			cs, ok := c.Context.providerConfig[rn.ID]
			if !ok {
				cs = make(map[string]map[string]interface{})
			}

			for k, p := range sharedProvider.Providers {
				// Merge the configurations to get what we use to configure with
				rc := sharedProvider.MergeConfig(false, cs[k])
				rc.interpolate(c, nil)

				log.Printf("[INFO] Validating provider: %s", k)
				ws, es := p.Validate(rc)
				for i, w := range ws {
					ws[i] = fmt.Sprintf("Provider '%s' warning: %s", k, w)
				}
				for i, e := range es {
					es[i] = fmt.Errorf("Provider '%s' error: %s", k, e)
				}

				l.Lock()
				meta.Warns = append(meta.Warns, ws...)
				meta.Errs = append(meta.Errs, es...)
				l.Unlock()
			}
		}

		return nil
	}

	return walkFn
}

func (c *walkContext) genericWalkFn(cb genericWalkFunc) depgraph.WalkFunc {
	// This will keep track of whether we're stopped or not
	var stop uint32 = 0

	var walkFn depgraph.WalkFunc
	walkFn = func(n *depgraph.Noun) error {
		// If it is the root node, ignore
		if n.Name == GraphRootNode {
			return nil
		}

		// If we're stopped, return right away
		if atomic.LoadUint32(&stop) != 0 {
			return nil
		}

		switch m := n.Meta.(type) {
		case *GraphNodeModule:
			// Build another walkContext for this module and walk it.
			wc := c.Context.walkContext(c.Operation, m.Path)

			// Set the graph to specifically walk this subgraph
			wc.graph = m.Graph

			// Preserve the meta
			wc.Meta = c.Meta

			// Set the variables
			if m.Config != nil {
				wc.Variables = make(map[string]string)

				rc := NewResourceConfig(m.Config.RawConfig)
				rc.interpolate(c, nil)
				for k, v := range rc.Config {
					wc.Variables[k] = v.(string)
				}
				for k, _ := range rc.Raw {
					if _, ok := wc.Variables[k]; !ok {
						wc.Variables[k] = config.UnknownVariableValue
					}
				}
			}

			return wc.Walk()
		case *GraphNodeResource:
			// Continue, we care about this the most
		case *GraphNodeResourceProvider:
			sharedProvider := m.Provider

			// Check if we have an override
			cs, ok := c.Context.providerConfig[m.ID]
			if !ok {
				cs = make(map[string]map[string]interface{})
			}

			for k, p := range sharedProvider.Providers {
				// Interpolate our own configuration before merging
				if sharedProvider.Config != nil {
					rc := NewResourceConfig(sharedProvider.Config.RawConfig)
					rc.interpolate(c, nil)
				}

				// Merge the configurations to get what we use to configure
				// with. We don't need to interpolate this because the
				// lines above verify that all parents are interpolated
				// properly.
				rc := sharedProvider.MergeConfig(false, cs[k])

				log.Printf("[INFO] Configuring provider: %s", k)
				err := p.Configure(rc)
				if err != nil {
					return err
				}
			}

			return nil
		default:
			panic(fmt.Sprintf("unknown graph node: %#v", n.Meta))
		}

		rn := n.Meta.(*GraphNodeResource)

		// If we're expanding, then expand the nodes, and then rewalk the graph
		if rn.ExpandMode > ResourceExpandNone {
			return c.genericWalkResource(rn, walkFn)
		}

		// Make sure that at least some resource configuration is set
		if rn.Config == nil {
			rn.Resource.Config = new(ResourceConfig)
		} else {
			rn.Resource.Config = NewResourceConfig(rn.Config.RawConfig)
		}

		// Handle recovery of special panic scenarios
		defer func() {
			if v := recover(); v != nil {
				if v == HookActionHalt {
					atomic.StoreUint32(&stop, 1)
				} else {
					panic(v)
				}
			}
		}()

		// Limit parallelism
		c.Context.parallelSem.Acquire()
		defer c.Context.parallelSem.Release()

		// Call the callack
		log.Printf(
			"[INFO] Module %s walking: %s (Graph node: %s)",
			strings.Join(c.Path, "."),
			rn.Resource.Id,
			n.Name)
		if err := cb(c, rn.Resource); err != nil {
			log.Printf("[ERROR] Error walking '%s': %s", rn.Resource.Id, err)
			return err
		}

		return nil
	}

	return walkFn
}

func (c *walkContext) genericWalkResource(
	rn *GraphNodeResource, fn depgraph.WalkFunc) error {
	// Interpolate the count
	rc := NewResourceConfig(rn.Config.RawCount)
	rc.interpolate(c, rn.Resource)

	// If we're validating, then we set the count to 1 if it is computed
	if c.Operation == walkValidate {
		if key := rn.Config.RawCount.Key; rc.IsComputed(key) {
			// Preserve the old value so that we reset it properly
			old := rn.Config.RawCount.Raw[key]
			defer func() {
				rn.Config.RawCount.Raw[key] = old
			}()

			// Set th count to 1 for validation purposes
			rn.Config.RawCount.Raw[key] = "1"
		}
	}

	// Expand the node to the actual resources
	g, err := rn.Expand()
	if err != nil {
		return err
	}

	// Walk the graph with our function
	if err := g.Walk(fn); err != nil {
		return err
	}

	return nil
}

// applyProvisioners is used to run any provisioners a resource has
// defined after the resource creation has already completed.
func (c *walkContext) applyProvisioners(r *Resource, is *InstanceState) error {
	// Store the original connection info, restore later
	origConnInfo := is.Ephemeral.ConnInfo
	defer func() {
		is.Ephemeral.ConnInfo = origConnInfo
	}()

	for _, prov := range r.Provisioners {
		// Interpolate since we may have variables that depend on the
		// local resource.
		if err := prov.Config.interpolate(c, r); err != nil {
			return err
		}

		// Interpolate the conn info, since it may contain variables
		connInfo := NewResourceConfig(prov.ConnInfo)
		if err := connInfo.interpolate(c, r); err != nil {
			return err
		}

		// Merge the connection information
		overlay := make(map[string]string)
		if origConnInfo != nil {
			for k, v := range origConnInfo {
				overlay[k] = v
			}
		}
		for k, v := range connInfo.Config {
			switch vt := v.(type) {
			case string:
				overlay[k] = vt
			case int64:
				overlay[k] = strconv.FormatInt(vt, 10)
			case int32:
				overlay[k] = strconv.FormatInt(int64(vt), 10)
			case int:
				overlay[k] = strconv.FormatInt(int64(vt), 10)
			case float32:
				overlay[k] = strconv.FormatFloat(float64(vt), 'f', 3, 32)
			case float64:
				overlay[k] = strconv.FormatFloat(vt, 'f', 3, 64)
			case bool:
				overlay[k] = strconv.FormatBool(vt)
			default:
				overlay[k] = fmt.Sprintf("%v", vt)
			}
		}
		is.Ephemeral.ConnInfo = overlay

		// Invoke the Provisioner
		for _, h := range c.Context.hooks {
			handleHook(h.PreProvision(r.Info, prov.Type))
		}

		output := ProvisionerUIOutput{
			Info:  r.Info,
			Type:  prov.Type,
			Hooks: c.Context.hooks,
		}
		err := prov.Provisioner.Apply(&output, is, prov.Config)
		if err != nil {
			return err
		}

		for _, h := range c.Context.hooks {
			handleHook(h.PostProvision(r.Info, prov.Type))
		}
	}

	return nil
}

// persistState persists the state in a Resource to the actual final
// state location.
func (c *walkContext) persistState(r *Resource) {
	// Acquire a state lock around this whole thing since we're updating that
	c.Context.sl.Lock()
	defer c.Context.sl.Unlock()

	// If we have no state, then we don't persist.
	if c.Context.state == nil {
		return
	}

	// Get the state for this resource. The resource state should always
	// exist because we call graphInitState before anything that could
	// potentially call this.
	module := c.Context.state.ModuleByPath(c.Path)
	if module == nil {
		module = c.Context.state.AddModule(c.Path)
	}
	rs := module.Resources[r.Id]
	if rs == nil {
		rs = &ResourceState{Type: r.Info.Type}
		rs.init()
		module.Resources[r.Id] = rs
	}
	rs.Dependencies = r.Dependencies

	// Assign the instance state to the proper location
	if r.Flags&FlagDeposed != 0 {
		// We were previously the primary and have been deposed, so
		// now we are the final tainted resource
		r.TaintedIndex = len(rs.Tainted) - 1
		rs.Tainted[r.TaintedIndex] = r.State

	} else if r.Flags&FlagTainted != 0 {
		if r.TaintedIndex >= 0 {
			// Tainted with a pre-existing index, just update that spot
			rs.Tainted[r.TaintedIndex] = r.State

		} else if r.Flags&FlagReplacePrimary != 0 {
			// We just replaced the primary, so restore the primary
			rs.Primary = rs.Tainted[len(rs.Tainted)-1]

			// Set ourselves as tainted
			rs.Tainted[len(rs.Tainted)-1] = r.State

		} else {
			// Newly tainted, so append it to the list, update the
			// index, and remove the primary.
			rs.Tainted = append(rs.Tainted, r.State)
			r.TaintedIndex = len(rs.Tainted) - 1
			rs.Primary = nil
		}

	} else if r.Flags&FlagReplacePrimary != 0 {
		// If the ID is blank (there was an error), then we leave
		// the primary that exists, and do not store this as a tainted
		// instance
		if r.State.ID == "" {
			return
		}

		// Push the old primary into the tainted state
		rs.Tainted = append(rs.Tainted, rs.Primary)

		// Set this as the new primary
		rs.Primary = r.State

	} else {
		// The primary instance, so just set it directly
		rs.Primary = r.State
	}

	// Do a pruning so that empty resources are not saved
	rs.prune()
}

// computeVars takes the State and given RawConfig and processes all
// the variables. This dynamically discovers the attributes instead of
// using a static map[string]string that the genericWalkFn uses.
func (c *walkContext) computeVars(
	raw *config.RawConfig, r *Resource) error {
	// If there isn't a raw configuration, don't do anything
	if raw == nil {
		return nil
	}

	// Copy the default variables
	vs := make(map[string]string)
	for k, v := range c.defaultVariables {
		vs[k] = v
	}

	// Next, the actual computed variables
	for n, rawV := range raw.Variables {
		switch v := rawV.(type) {
		case *config.CountVariable:
			switch v.Type {
			case config.CountValueIndex:
				if r != nil {
					vs[n] = strconv.FormatInt(int64(r.CountIndex), 10)
				}
			}
		case *config.ModuleVariable:
			if c.Operation == walkValidate {
				vs[n] = config.UnknownVariableValue
				continue
			}

			value, err := c.computeModuleVariable(v)
			if err != nil {
				return err
			}

			vs[n] = value
		case *config.PathVariable:
			switch v.Type {
			case config.PathValueCwd:
				wd, err := os.Getwd()
				if err != nil {
					return fmt.Errorf(
						"Couldn't get cwd for var %s: %s",
						v.FullKey(), err)
				}

				vs[n] = wd
			case config.PathValueModule:
				if t := c.Context.module.Child(c.Path[1:]); t != nil {
					vs[n] = t.Config().Dir
				}
			case config.PathValueRoot:
				vs[n] = c.Context.module.Config().Dir
			}
		case *config.ResourceVariable:
			if c.Operation == walkValidate {
				vs[n] = config.UnknownVariableValue
				continue
			}

			var attr string
			var err error
			if v.Multi && v.Index == -1 {
				attr, err = c.computeResourceMultiVariable(v)
			} else {
				attr, err = c.computeResourceVariable(v)
			}
			if err != nil {
				return err
			}

			vs[n] = attr
		case *config.UserVariable:
			val, ok := c.Variables[v.Name]
			if ok {
				vs[n] = val
				continue
			}

			if _, ok := vs[n]; !ok && c.Operation == walkValidate {
				vs[n] = config.UnknownVariableValue
				continue
			}

			// Look up if we have any variables with this prefix because
			// those are map overrides. Include those.
			for k, val := range c.Variables {
				if strings.HasPrefix(k, v.Name+".") {
					vs["var."+k] = val
				}
			}
		}
	}

	// Interpolate the variables
	return raw.Interpolate(vs)
}

func (c *walkContext) computeModuleVariable(
	v *config.ModuleVariable) (string, error) {
	// Build the path to our child
	path := make([]string, len(c.Path), len(c.Path)+1)
	copy(path, c.Path)
	path = append(path, v.Name)

	// Grab some locks
	c.Context.sl.RLock()
	defer c.Context.sl.RUnlock()

	// Get that module from our state
	mod := c.Context.state.ModuleByPath(path)
	if mod == nil {
		return "", fmt.Errorf(
			"Module '%s' not found for variable '%s'",
			strings.Join(path[1:], "."),
			v.FullKey())
	}

	value, ok := mod.Outputs[v.Field]
	if !ok {
		return "", fmt.Errorf(
			"Output field '%s' not found for variable '%s'",
			v.Field,
			v.FullKey())
	}

	return value, nil
}

func (c *walkContext) computeResourceVariable(
	v *config.ResourceVariable) (string, error) {
	id := v.ResourceId()
	if v.Multi {
		id = fmt.Sprintf("%s.%d", id, v.Index)
	}

	c.Context.sl.RLock()
	defer c.Context.sl.RUnlock()

	// Get the relevant module
	module := c.Context.state.ModuleByPath(c.Path)

	r, ok := module.Resources[id]
	if !ok {
		if v.Multi && v.Index == 0 {
			r, ok = module.Resources[v.ResourceId()]
		}
		if !ok {
			return "", fmt.Errorf(
				"Resource '%s' not found for variable '%s'",
				id,
				v.FullKey())
		}
	}

	if r.Primary == nil {
		goto MISSING
	}

	if attr, ok := r.Primary.Attributes[v.Field]; ok {
		return attr, nil
	}

	// We didn't find the exact field, so lets separate the dots
	// and see if anything along the way is a computed set. i.e. if
	// we have "foo.0.bar" as the field, check to see if "foo" is
	// a computed list. If so, then the whole thing is computed.
	if parts := strings.Split(v.Field, "."); len(parts) > 1 {
		for i := 1; i < len(parts); i++ {
			key := fmt.Sprintf("%s.#", strings.Join(parts[:i], "."))
			if attr, ok := r.Primary.Attributes[key]; ok {
				return attr, nil
			}
		}
	}

MISSING:
	return "", fmt.Errorf(
		"Resource '%s' does not have attribute '%s' "+
			"for variable '%s'",
		id,
		v.Field,
		v.FullKey())
}

func (c *walkContext) computeResourceMultiVariable(
	v *config.ResourceVariable) (string, error) {
	c.Context.sl.RLock()
	defer c.Context.sl.RUnlock()

	// Get the resource from the configuration so we can know how
	// many of the resource there is.
	var cr *config.Resource
	for _, r := range c.Context.module.Config().Resources {
		if r.Id() == v.ResourceId() {
			cr = r
			break
		}
	}
	if cr == nil {
		return "", fmt.Errorf(
			"Resource '%s' not found for variable '%s'",
			v.ResourceId(),
			v.FullKey())
	}

	// Get the relevant module
	// TODO: Not use only root module
	module := c.Context.state.RootModule()

	count, err := cr.Count()
	if err != nil {
		return "", fmt.Errorf(
			"Error reading %s count: %s",
			v.ResourceId(),
			err)
	}

	// If we have no count, return empty
	if count == 0 {
		return "", nil
	}

	var values []string
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("%s.%d", v.ResourceId(), i)

		// If we're dealing with only a single resource, then the
		// ID doesn't have a trailing index.
		if count == 1 {
			id = v.ResourceId()
		}

		r, ok := module.Resources[id]
		if !ok {
			continue
		}

		if r.Primary == nil {
			continue
		}

		attr, ok := r.Primary.Attributes[v.Field]
		if !ok {
			continue
		}

		values = append(values, attr)
	}

	if len(values) == 0 {
		return "", fmt.Errorf(
			"Resource '%s' does not have attribute '%s' "+
				"for variable '%s'",
			v.ResourceId(),
			v.Field,
			v.FullKey())
	}

	return strings.Join(values, config.InterpSplitDelim), nil
}

type walkInputMeta struct {
	sync.Mutex

	Done map[string]struct{}
}

type walkValidateMeta struct {
	Errs     []error
	Warns    []string
	Children map[string]*walkValidateMeta
}

func (m *walkValidateMeta) Flatten() ([]string, []error) {
	// Prune out the empty children
	for k, m2 := range m.Children {
		if len(m2.Errs) == 0 && len(m2.Warns) == 0 {
			delete(m.Children, k)
		}
	}

	// If we have no children, then just return what we have
	if len(m.Children) == 0 {
		return m.Warns, m.Errs
	}

	// Otherwise, copy the errors and warnings
	errs := make([]error, len(m.Errs))
	warns := make([]string, len(m.Warns))
	for i, err := range m.Errs {
		errs[i] = err
	}
	for i, warn := range m.Warns {
		warns[i] = warn
	}

	// Now go through each child and copy it in...
	for k, c := range m.Children {
		for _, err := range c.Errs {
			errs = append(errs, fmt.Errorf(
				"Module %s: %s", k, err))
		}
		for _, warn := range c.Warns {
			warns = append(warns, fmt.Sprintf(
				"Module %s: %s", k, warn))
		}
	}

	return warns, errs
}
