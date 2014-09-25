package terraform

import (
	"fmt"
	"log"
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
	module       *module.Tree
	diff         *Diff
	hooks        []Hook
	state        *State
	providers    map[string]ResourceProviderFactory
	provisioners map[string]ResourceProvisionerFactory
	variables    map[string]string

	l     sync.Mutex    // Lock acquired during any task
	parCh chan struct{} // Semaphore used to limit parallelism
	sl    sync.RWMutex  // Lock acquired to R/W internal data
	runCh <-chan struct{}
	sh    *stopHook
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
}

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
	parCh := make(chan struct{}, par)

	return &Context{
		diff:         opts.Diff,
		hooks:        hooks,
		module:       opts.Module,
		state:        opts.State,
		providers:    opts.Providers,
		provisioners: opts.Provisioners,
		variables:    opts.Variables,

		parCh: parCh,
		sh:    sh,
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

	if c.Operation == walkValidate {
		// Validation is the only one that doesn't calculate outputs
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
	if c.Operation == walkApply {
		// On Apply, we prune so that we don't do outputs if we destroyed
		mod.prune()
	}
	if len(mod.Resources) == 0 {
		return nil
	}

	outputs := make(map[string]string)
	for _, o := range conf.Outputs {
		if err := c.computeVars(o.RawConfig); err != nil {
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
			if err := r.Config.interpolate(c); err != nil {
				return err
			}

			diff, err = r.Provider.Diff(r.Info, is, r.Config)
			if err != nil {
				return err
			}

			// This should never happen because we check if Diff.Empty above.
			// If this happened, then the diff above returned a bad diff.
			if diff == nil {
				return fmt.Errorf(
					"%s: diff became nil during Apply. This is a bug with "+
						"the resource provider. Please report a bug.",
					r.Id)
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
			handleHook(h.PreApply(r.Id, is, diff))
		}

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
		if applyerr == nil && is.ID != "" && len(r.Provisioners) > 0 {
			for _, h := range c.Context.hooks {
				handleHook(h.PreProvisionResource(r.Id, is))
			}

			if err := c.applyProvisioners(r, is); err != nil {
				errs = append(errs, err)
				tainted = true
			}

			for _, h := range c.Context.hooks {
				handleHook(h.PostProvisionResource(r.Id, is))
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
			handleHook(h.PostApply(r.Id, is, applyerr))
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
			handleHook(h.PreDiff(r.Id, is))
		}

		if r.Flags&FlagOrphan != 0 {
			log.Printf("[DEBUG] %s: Orphan, marking for destroy", r.Id)

			// This is an orphan (no config), so we mark it to be destroyed
			diff = &InstanceDiff{Destroy: true}
		} else {
			// Make sure the configuration is interpolated
			if err := r.Config.interpolate(c); err != nil {
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
			l.Lock()
			md := result.Diff.ModuleByPath(c.Path)
			if md == nil {
				md = result.Diff.AddModule(c.Path)
			}
			md.Resources[r.Id] = diff
			l.Unlock()
		}

		for _, h := range c.Context.hooks {
			handleHook(h.PostDiff(r.Id, diff))
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

	return func(n *depgraph.Noun) error {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			return nil
		}

		r := rn.Resource
		if r.State != nil && r.State.ID != "" {
			log.Printf("[DEBUG] %s: Making for destroy", r.Id)

			l.Lock()
			defer l.Unlock()
			result.Diff.RootModule().Resources[r.Id] = &InstanceDiff{Destroy: true}
		} else {
			log.Printf("[DEBUG] %s: Not marking for destroy, no ID", r.Id)
		}

		return nil
	}
}

func (c *walkContext) refreshWalkFn() depgraph.WalkFunc {
	cb := func(c *walkContext, r *Resource) error {
		is := r.State

		if is == nil || is.ID == "" {
			log.Printf("[DEBUG] %s: Not refreshing, ID is empty", r.Id)
			return nil
		}

		for _, h := range c.Context.hooks {
			handleHook(h.PreRefresh(r.Id, is))
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
			handleHook(h.PostRefresh(r.Id, is))
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

	return func(n *depgraph.Noun) error {
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

			// If it doesn't have a provider, that is a different problem
			if rn.Resource.Provider == nil {
				return nil
			}

			// Don't validate orphans since they never have a config
			if rn.Resource.Flags&FlagOrphan != 0 {
				return nil
			}

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

			var raw *config.RawConfig
			if sharedProvider.Config != nil {
				raw = sharedProvider.Config.RawConfig
			}

			// If we have a parent, then merge in the parent configurations
			// properly so we "inherit" the configurations.
			if sharedProvider.Parent != nil {
				var rawMap map[string]interface{}
				if raw != nil {
					rawMap = raw.Raw
				}

				parent := sharedProvider.Parent
				for parent != nil {
					if parent.Config != nil {
						if rawMap == nil {
							rawMap = parent.Config.RawConfig.Raw
						}

						for k, v := range parent.Config.RawConfig.Raw {
							rawMap[k] = v
						}
					}

					parent = parent.Parent
				}

				// Update our configuration to be the merged result
				var err error
				raw, err = config.NewRawConfig(rawMap)
				if err != nil {
					return fmt.Errorf("Error merging configurations: %s", err)
				}
			}

			rc := NewResourceConfig(raw)

			for k, p := range sharedProvider.Providers {
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
}

func (c *walkContext) genericWalkFn(cb genericWalkFunc) depgraph.WalkFunc {
	// This will keep track of whether we're stopped or not
	var stop uint32 = 0

	return func(n *depgraph.Noun) error {
		// If it is the root node, ignore
		if n.Name == GraphRootNode {
			return nil
		}

		// If we're stopped, return right away
		if atomic.LoadUint32(&stop) != 0 {
			return nil
		}

		// Limit parallelism
		c.Context.parCh <- struct{}{}
		defer func() {
			<-c.Context.parCh
		}()

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
				rc.interpolate(c)
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
		case *GraphNodeResourceMeta:
			// Skip it
			return nil
		case *GraphNodeResourceProvider:
			sharedProvider := m.Provider

			// Interpolate in the variables and configure all the providers
			var raw *config.RawConfig
			if sharedProvider.Config != nil {
				raw = sharedProvider.Config.RawConfig
			}

			// If we have a parent, then merge in the parent configurations
			// properly so we "inherit" the configurations.
			if sharedProvider.Parent != nil {
				var rawMap map[string]interface{}
				if raw != nil {
					rawMap = raw.Raw
				}

				parent := sharedProvider.Parent
				for parent != nil {
					if parent.Config != nil {
						if rawMap == nil {
							rawMap = parent.Config.RawConfig.Raw
						}

						for k, v := range parent.Config.RawConfig.Config() {
							rawMap[k] = v
						}
					}

					parent = parent.Parent
				}

				// Update our configuration to be the merged result
				var err error
				raw, err = config.NewRawConfig(rawMap)
				if err != nil {
					return fmt.Errorf("Error merging configurations: %s", err)
				}
			}

			rc := NewResourceConfig(raw)
			rc.interpolate(c)

			for k, p := range sharedProvider.Providers {
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
		if err := prov.Config.interpolate(c); err != nil {
			return err
		}

		// Interpolate the conn info, since it may contain variables
		connInfo := NewResourceConfig(prov.ConnInfo)
		if err := connInfo.interpolate(c); err != nil {
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
			handleHook(h.PreProvision(r.Id, prov.Type))
		}

		if err := prov.Provisioner.Apply(is, prov.Config); err != nil {
			return err
		}

		for _, h := range c.Context.hooks {
			handleHook(h.PostProvision(r.Id, prov.Type))
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
	if r.Flags&FlagTainted != 0 {
		if r.TaintedIndex >= 0 {
			// Tainted with a pre-existing index, just update that spot
			rs.Tainted[r.TaintedIndex] = r.State
		} else {
			// Newly tainted, so append it to the list, update the
			// index, and remove the primary.
			rs.Tainted = append(rs.Tainted, r.State)
			rs.Primary = nil
			r.TaintedIndex = len(rs.Tainted) - 1
		}
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
func (c *walkContext) computeVars(raw *config.RawConfig) error {
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
		case *config.ModuleVariable:
			value, err := c.computeModuleVariable(v)
			if err != nil {
				return err
			}

			vs[n] = value
		case *config.ResourceVariable:
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
		return "", fmt.Errorf(
			"Resource '%s' not found for variable '%s'",
			id,
			v.FullKey())
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

	var values []string
	for i := 0; i < cr.Count; i++ {
		id := fmt.Sprintf("%s.%d", v.ResourceId(), i)

		// If we're dealing with only a single resource, then the
		// ID doesn't have a trailing index.
		if cr.Count == 1 {
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

	return strings.Join(values, ","), nil
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
