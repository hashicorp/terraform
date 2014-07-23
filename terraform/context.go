package terraform

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/depgraph"
	"github.com/hashicorp/terraform/helper/multierror"
)

// This is a function type used to implement a walker for the resource
// tree internally on the Terraform structure.
type genericWalkFunc func(*Resource) error

// Context represents all the context that Terraform needs in order to
// perform operations on infrastructure. This structure is built using
// ContextOpts and NewContext. See the documentation for those.
//
// Additionally, a context can be created from a Plan using Plan.Context.
type Context struct {
	config       *config.Config
	diff         *Diff
	hooks        []Hook
	state        *State
	providers    map[string]ResourceProviderFactory
	provisioners map[string]ResourceProvisionerFactory
	variables    map[string]string
	defaultVars  map[string]string

	l     sync.Mutex    // Lock acquired during any task
	parCh chan struct{} // Semaphore used to limit parallelism
	sl    sync.RWMutex  // Lock acquired to R/W internal data
	runCh <-chan struct{}
	sh    *stopHook
}

// ContextOpts are the user-creatable configuration structure to create
// a context with NewContext.
type ContextOpts struct {
	Config       *config.Config
	Diff         *Diff
	Hooks        []Hook
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

	// Calculate all the default variables
	defaultVars := make(map[string]string)
	for _, v := range opts.Config.Variables {
		for k, val := range v.DefaultsMap() {
			defaultVars[k] = val
		}
	}

	return &Context{
		config:       opts.Config,
		diff:         opts.Diff,
		hooks:        hooks,
		state:        opts.State,
		providers:    opts.Providers,
		provisioners: opts.Provisioners,
		variables:    opts.Variables,
		defaultVars:  defaultVars,

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

	g, err := Graph(&GraphOpts{
		Config:       c.config,
		Diff:         c.diff,
		Providers:    c.providers,
		Provisioners: c.provisioners,
		State:        c.state,
	})
	if err != nil {
		return nil, err
	}

	// Set our state right away. No matter what, this IS our new state,
	// even if there is an error below.
	c.state = c.state.deepcopy()

	// Walk
	err = g.Walk(c.applyWalkFn())

	// Prune the state so that we have as clean a state as possible
	c.state.prune()

	// If we have no errors, then calculate the outputs if we have any
	if err == nil && len(c.config.Outputs) > 0 && len(c.state.Resources) > 0 {
		c.state.Outputs = make(map[string]string)
		for _, o := range c.config.Outputs {
			if err = c.computeVars(o.RawConfig); err != nil {
				break
			}

			c.state.Outputs[o.Name] = o.RawConfig.Config()["value"].(string)
		}
	}

	return c.state, err
}

// Graph returns the graph for this context.
func (c *Context) Graph() (*depgraph.Graph, error) {
	return c.graph()
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

	g, err := Graph(&GraphOpts{
		Config:       c.config,
		Providers:    c.providers,
		Provisioners: c.provisioners,
		State:        c.state,
	})
	if err != nil {
		return nil, err
	}

	p := &Plan{
		Config: c.config,
		Vars:   c.variables,
		State:  c.state,
	}

	var walkFn depgraph.WalkFunc

	if opts != nil && opts.Destroy {
		// If we're destroying, we use a different walk function since it
		// doesn't need as many details.
		walkFn = c.planDestroyWalkFn(p)
	} else {
		// Set our state to be something temporary. We do this so that
		// the plan can update a fake state so that variables work, then
		// we replace it back with our old state.
		old := c.state
		c.state = old.deepcopy()
		defer func() {
			c.state = old
		}()

		walkFn = c.planWalkFn(p)
	}

	// Walk and run the plan
	err = g.Walk(walkFn)

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

	g, err := Graph(&GraphOpts{
		Config:       c.config,
		Providers:    c.providers,
		Provisioners: c.provisioners,
		State:        c.state,
	})
	if err != nil {
		return c.state, err
	}

	// Update our state
	c.state = c.state.deepcopy()

	err = g.Walk(c.refreshWalkFn())
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
	if err := c.config.Validate(); err != nil {
		rerr = multierror.ErrorAppend(rerr, err)
	}

	// Validate the user variables
	if errs := smcUserVariables(c.config, c.variables); len(errs) > 0 {
		rerr = multierror.ErrorAppend(rerr, errs...)
	}

	// Validate the graph
	g, err := c.graph()
	if err != nil {
		rerr = multierror.ErrorAppend(rerr, fmt.Errorf(
			"Error creating graph: %s", err))
	}

	// Walk the graph and validate all the configs
	var warns []string
	var errs []error
	if g != nil {
		err = g.Walk(c.validateWalkFn(&warns, &errs))
		if err != nil {
			rerr = multierror.ErrorAppend(rerr, fmt.Errorf(
				"Error validating resources in graph: %s", err))
		}
		if len(errs) > 0 {
			rerr = multierror.ErrorAppend(rerr, errs...)
		}
	}

	errs = nil
	if rerr != nil && len(rerr.Errors) > 0 {
		errs = rerr.Errors
	}

	return warns, errs
}

// computeVars takes the State and given RawConfig and processes all
// the variables. This dynamically discovers the attributes instead of
// using a static map[string]string that the genericWalkFn uses.
func (c *Context) computeVars(raw *config.RawConfig) error {
	// If there isn't a raw configuration, don't do anything
	if raw == nil {
		return nil
	}

	// If there are on variables, then we're done
	if len(raw.Variables) == 0 {
		return nil
	}

	// Start building up the variables. First, defaults
	vs := make(map[string]string)
	for k, v := range c.defaultVars {
		vs[k] = v
	}

	// Next, the actual computed variables
	for n, rawV := range raw.Variables {
		switch v := rawV.(type) {
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
			val, ok := c.variables[v.Name]
			if ok {
				vs[n] = val
				continue
			}

			// Look up if we have any variables with this prefix because
			// those are map overrides. Include those.
			for k, val := range c.variables {
				if strings.HasPrefix(k, v.Name+".") {
					vs["var."+k] = val
				}
			}
		}
	}

	// Interpolate the variables
	return raw.Interpolate(vs)
}

func (c *Context) computeResourceVariable(
	v *config.ResourceVariable) (string, error) {
	id := v.ResourceId()
	if v.Multi {
		id = fmt.Sprintf("%s.%d", id, v.Index)
	}

	c.sl.RLock()
	defer c.sl.RUnlock()

	r, ok := c.state.Resources[id]
	if !ok {
		return "", fmt.Errorf(
			"Resource '%s' not found for variable '%s'",
			id,
			v.FullKey())
	}

	attr, ok := r.Attributes[v.Field]
	if !ok {
		return "", fmt.Errorf(
			"Resource '%s' does not have attribute '%s' "+
				"for variable '%s'",
			id,
			v.Field,
			v.FullKey())
	}

	return attr, nil
}

func (c *Context) computeResourceMultiVariable(
	v *config.ResourceVariable) (string, error) {
	c.sl.RLock()
	defer c.sl.RUnlock()

	// Get the resource from the configuration so we can know how
	// many of the resource there is.
	var cr *config.Resource
	for _, r := range c.config.Resources {
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

	var values []string
	for i := 0; i < cr.Count; i++ {
		id := fmt.Sprintf("%s.%d", v.ResourceId(), i)
		r, ok := c.state.Resources[id]
		if !ok {
			continue
		}

		attr, ok := r.Attributes[v.Field]
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

func (c *Context) graph() (*depgraph.Graph, error) {
	return Graph(&GraphOpts{
		Config:       c.config,
		Diff:         c.diff,
		Providers:    c.providers,
		Provisioners: c.provisioners,
		State:        c.state,
	})
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

func (c *Context) applyWalkFn() depgraph.WalkFunc {
	cb := func(r *Resource) error {
		var err error

		diff := r.Diff
		if diff.Empty() {
			log.Printf("[DEBUG] %s: Diff is empty. Will not apply.", r.Id)
			return nil
		}

		if !diff.Destroy {
			// Since we need the configuration, interpolate the variables
			if err := r.Config.interpolate(c); err != nil {
				return err
			}

			diff, err = r.Provider.Diff(r.State, r.Config)
			if err != nil {
				return err
			}
		}

		// This should never happen because we check if Diff.Empty above.
		// If this happened, then the diff above returned a bad diff.
		if diff == nil {
			return fmt.Errorf(
				"%s: diff became nil during Apply. This is a bug with " +
					"the resource provider. Please report a bug.")
		}

		// If we do not have any connection info, initialize
		if r.State.ConnInfo == nil {
			r.State.ConnInfo = make(map[string]string)
		}

		// Remove any output values from the diff
		for k, ad := range diff.Attributes {
			if ad.Type == DiffAttrOutput {
				delete(diff.Attributes, k)
			}
		}

		for _, h := range c.hooks {
			handleHook(h.PreApply(r.Id, r.State, diff))
		}

		// With the completed diff, apply!
		log.Printf("[DEBUG] %s: Executing Apply", r.Id)
		rs, applyerr := r.Provider.Apply(r.State, diff)

		var errs []error
		if applyerr != nil {
			errs = append(errs, applyerr)
		}

		// Make sure the result is instantiated
		if rs == nil {
			rs = new(ResourceState)
		}

		// Force the resource state type to be our type
		rs.Type = r.State.Type

		// Force the "id" attribute to be our ID
		if rs.ID != "" {
			if rs.Attributes == nil {
				rs.Attributes = make(map[string]string)
			}

			rs.Attributes["id"] = rs.ID
		}

		for ak, av := range rs.Attributes {
			// If the value is the unknown variable value, then it is an error.
			// In this case we record the error and remove it from the state
			if av == config.UnknownVariableValue {
				errs = append(errs, fmt.Errorf(
					"Attribute with unknown value: %s", ak))
				delete(rs.Attributes, ak)
			}
		}

		// Invoke any provisioners we have defined. This is only done
		// if the resource was created, as updates or deletes do not
		// invoke provisioners.
		//
		// Additionally, we need to be careful to not run this if there
		// was an error during the provider apply.
		tainted := false
		if applyerr == nil && r.State.ID == "" && len(r.Provisioners) > 0 {
			if err := c.applyProvisioners(r, rs); err != nil {
				errs = append(errs, err)
				tainted = true
			}
		}

		// Update the resulting diff
		c.sl.Lock()
		if rs.ID == "" {
			delete(c.state.Resources, r.Id)
		} else {
			c.state.Resources[r.Id] = rs

			if tainted {
				c.state.Tainted[r.Id] = struct{}{}
			}
		}
		c.sl.Unlock()

		// Update the state for the resource itself
		r.State = rs

		for _, h := range c.hooks {
			handleHook(h.PostApply(r.Id, r.State, applyerr))
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

// applyProvisioners is used to run any provisioners a resource has
// defined after the resource creation has already completed.
func (c *Context) applyProvisioners(r *Resource, rs *ResourceState) error {
	// Store the original connection info, restore later
	origConnInfo := rs.ConnInfo
	defer func() {
		rs.ConnInfo = origConnInfo
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
		rs.ConnInfo = overlay

		// Invoke the Provisioner
		if err := prov.Provisioner.Apply(rs, prov.Config); err != nil {
			return err
		}
	}

	return nil
}

func (c *Context) planWalkFn(result *Plan) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result
	result.init()

	cb := func(r *Resource) error {
		var diff *ResourceDiff

		for _, h := range c.hooks {
			handleHook(h.PreDiff(r.Id, r.State))
		}

		if r.Config == nil {
			log.Printf("[DEBUG] %s: Orphan, marking for destroy", r.Id)

			// This is an orphan (no config), so we mark it to be destroyed
			diff = &ResourceDiff{Destroy: true}
		} else {
			// Make sure the configuration is interpolated
			if err := r.Config.interpolate(c); err != nil {
				return err
			}

			// Get a diff from the newest state
			log.Printf("[DEBUG] %s: Executing diff", r.Id)
			var err error
			state := r.State
			if r.Tainted {
				// If we're tainted, we pretend to create a new thing.
				state = new(ResourceState)
				state.Type = r.State.Type
			}
			diff, err = r.Provider.Diff(state, r.Config)
			if err != nil {
				return err
			}
		}

		if diff == nil {
			diff = new(ResourceDiff)
		}

		if r.Tainted {
			// Tainted resources must also be destroyed
			diff.Destroy = true
		}

		if diff.RequiresNew() && r.State.ID != "" {
			// This will also require a destroy
			diff.Destroy = true
		}

		if diff.RequiresNew() || r.State.ID == "" {
			// Add diff to compute new ID
			diff.init()
			diff.Attributes["id"] = &ResourceAttrDiff{
				Old:         r.State.Attributes["id"],
				NewComputed: true,
				RequiresNew: true,
				Type:        DiffAttrOutput,
			}
		}

		l.Lock()
		if !diff.Empty() {
			result.Diff.Resources[r.Id] = diff
		}
		l.Unlock()

		for _, h := range c.hooks {
			handleHook(h.PostDiff(r.Id, diff))
		}

		// Determine the new state and update variables
		if !diff.Empty() {
			r.State = r.State.MergeDiff(diff)
		}

		// Update our internal state so that variable computation works
		c.sl.Lock()
		defer c.sl.Unlock()
		c.state.Resources[r.Id] = r.State

		return nil
	}

	return c.genericWalkFn(cb)
}

func (c *Context) planDestroyWalkFn(result *Plan) depgraph.WalkFunc {
	var l sync.Mutex

	// Initialize the result
	result.init()

	return func(n *depgraph.Noun) error {
		rn, ok := n.Meta.(*GraphNodeResource)
		if !ok {
			return nil
		}

		r := rn.Resource
		if r.State.ID != "" {
			log.Printf("[DEBUG] %s: Making for destroy", r.Id)

			l.Lock()
			defer l.Unlock()
			result.Diff.Resources[r.Id] = &ResourceDiff{Destroy: true}
		} else {
			log.Printf("[DEBUG] %s: Not marking for destroy, no ID", r.Id)
		}

		return nil
	}
}

func (c *Context) refreshWalkFn() depgraph.WalkFunc {
	cb := func(r *Resource) error {
		if r.State.ID == "" {
			log.Printf("[DEBUG] %s: Not refreshing, ID is empty", r.Id)
			return nil
		}

		for _, h := range c.hooks {
			handleHook(h.PreRefresh(r.Id, r.State))
		}

		rs, err := r.Provider.Refresh(r.State)
		if err != nil {
			return err
		}
		if rs == nil {
			rs = new(ResourceState)
		}

		// Fix the type to be the type we have
		rs.Type = r.State.Type

		c.sl.Lock()
		c.state.Resources[r.Id] = rs
		c.sl.Unlock()

		for _, h := range c.hooks {
			handleHook(h.PostRefresh(r.Id, rs))
		}

		return nil
	}

	return c.genericWalkFn(cb)
}

func (c *Context) validateWalkFn(rws *[]string, res *[]error) depgraph.WalkFunc {
	var l sync.Mutex

	return func(n *depgraph.Noun) error {
		// If it is the root node, ignore
		if n.Name == GraphRootNode {
			return nil
		}

		switch rn := n.Meta.(type) {
		case *GraphNodeResource:
			if rn.Resource == nil {
				panic("resource should never be nil")
			}

			// If it doesn't have a provider, that is a different problem
			if rn.Resource.Provider == nil {
				return nil
			}

			// Don't validate orphans since they never have a config
			if rn.Orphan {
				return nil
			}

			log.Printf("[INFO] Validating resource: %s", rn.Resource.Id)
			ws, es := rn.Resource.Provider.ValidateResource(
				rn.Type, rn.Resource.Config)
			for i, w := range ws {
				ws[i] = fmt.Sprintf("'%s' warning: %s", rn.Resource.Id, w)
			}
			for i, e := range es {
				es[i] = fmt.Errorf("'%s' error: %s", rn.Resource.Id, e)
			}

			l.Lock()
			*rws = append(*rws, ws...)
			*res = append(*res, es...)
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
				*rws = append(*rws, ws...)
				*res = append(*res, es...)
				l.Unlock()
			}

		case *GraphNodeResourceProvider:
			if rn.Config == nil {
				return nil
			}

			rc := NewResourceConfig(rn.Config.RawConfig)

			for k, p := range rn.Providers {
				log.Printf("[INFO] Validating provider: %s", k)
				ws, es := p.Validate(rc)
				for i, w := range ws {
					ws[i] = fmt.Sprintf("Provider '%s' warning: %s", k, w)
				}
				for i, e := range es {
					es[i] = fmt.Errorf("Provider '%s' error: %s", k, e)
				}

				l.Lock()
				*rws = append(*rws, ws...)
				*res = append(*res, es...)
				l.Unlock()
			}
		}

		return nil
	}
}

func (c *Context) genericWalkFn(cb genericWalkFunc) depgraph.WalkFunc {
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
		c.parCh <- struct{}{}
		defer func() {
			<-c.parCh
		}()

		switch m := n.Meta.(type) {
		case *GraphNodeResource:
			// Continue, we care about this the most
		case *GraphNodeResourceMeta:
			// Skip it
			return nil
		case *GraphNodeResourceProvider:
			// Interpolate in the variables and configure all the providers
			var raw *config.RawConfig
			if m.Config != nil {
				raw = m.Config.RawConfig
			}

			rc := NewResourceConfig(raw)
			rc.interpolate(c)

			for k, p := range m.Providers {
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
		if !rn.Orphan {
			if rn.Config == nil {
				rn.Resource.Config = new(ResourceConfig)
			} else {
				rn.Resource.Config = NewResourceConfig(rn.Config.RawConfig)
			}
		} else {
			rn.Resource.Config = nil
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
			"[INFO] Walking: %s (Graph node: %s)",
			rn.Resource.Id,
			n.Name)
		if err := cb(rn.Resource); err != nil {
			log.Printf("[ERROR] Error walking '%s': %s", rn.Resource.Id, err)
			return err
		}

		return nil
	}
}
