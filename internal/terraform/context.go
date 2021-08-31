package terraform

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/apparentlymart/go-versions/versions"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
	_ "github.com/hashicorp/terraform/internal/logging"
)

// InputMode defines what sort of input will be asked for when Input
// is called on Context.
type InputMode byte

const (
	// InputModeProvider asks for provider variables
	InputModeProvider InputMode = 1 << iota

	// InputModeStd is the standard operating mode and asks for both variables
	// and providers.
	InputModeStd = InputModeProvider
)

// ContextOpts are the user-configurable options to create a context with
// NewContext.
type ContextOpts struct {
	Meta         *ContextMeta
	Hooks        []Hook
	Parallelism  int
	Providers    map[addrs.Provider]providers.Factory
	Provisioners map[string]provisioners.Factory

	// If non-nil, will apply as additional constraints on the provider
	// plugins that will be requested from the provider resolver.
	ProviderSHA256s map[string][]byte

	// If non-nil, will be verified to ensure that provider requirements from
	// configuration can be satisfied by the set of locked dependencies.
	LockedDependencies *depsfile.Locks

	// Set of providers to exclude from the requirements check process, as they
	// are marked as in local development.
	ProvidersInDevelopment map[addrs.Provider]struct{}

	UIInput UIInput
}

// ContextMeta is metadata about the running context. This is information
// that this package or structure cannot determine on its own but exposes
// into Terraform in various ways. This must be provided by the Context
// initializer.
type ContextMeta struct {
	Env string // Env is the state environment

	// OriginalWorkingDir is the working directory where the Terraform CLI
	// was run from, which may no longer actually be the current working
	// directory if the user included the -chdir=... option.
	//
	// If this string is empty then the original working directory is the same
	// as the current working directory.
	//
	// In most cases we should respect the user's override by ignoring this
	// path and just using the current working directory, but this is here
	// for some exceptional cases where the original working directory is
	// needed.
	OriginalWorkingDir string
}

// Context represents all the context that Terraform needs in order to
// perform operations on infrastructure. This structure is built using
// NewContext.
type Context struct {
	// meta captures some misc. information about the working directory where
	// we're taking these actions, and thus which should remain steady between
	// operations.
	meta *ContextMeta

	plugins                *contextPlugins
	dependencyLocks        *depsfile.Locks
	providersInDevelopment map[addrs.Provider]struct{}

	hooks   []Hook
	sh      *stopHook
	uiInput UIInput

	l                   sync.Mutex // Lock acquired during any task
	parallelSem         Semaphore
	providerInputConfig map[string]map[string]cty.Value
	providerSHA256s     map[string][]byte
	runCond             *sync.Cond
	runContext          context.Context
	runContextCancel    context.CancelFunc
}

// (additional methods on Context can be found in context_*.go files.)

// NewContext creates a new Context structure.
//
// Once a Context is created, the caller must not access or mutate any of
// the objects referenced (directly or indirectly) by the ContextOpts fields.
//
// If the returned diagnostics contains errors then the resulting context is
// invalid and must not be used.
func NewContext(opts *ContextOpts) (*Context, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	log.Printf("[TRACE] terraform.NewContext: starting")

	// Copy all the hooks and add our stop hook. We don't append directly
	// to the Config so that we're not modifying that in-place.
	sh := new(stopHook)
	hooks := make([]Hook, len(opts.Hooks)+1)
	copy(hooks, opts.Hooks)
	hooks[len(opts.Hooks)] = sh

	// Determine parallelism, default to 10. We do this both to limit
	// CPU pressure but also to have an extra guard against rate throttling
	// from providers.
	// We throw an error in case of negative parallelism
	par := opts.Parallelism
	if par < 0 {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid parallelism value",
			fmt.Sprintf("The parallelism must be a positive value. Not %d.", par),
		))
		return nil, diags
	}

	if par == 0 {
		par = 10
	}

	plugins := newContextPlugins(opts.Providers, opts.Provisioners)

	log.Printf("[TRACE] terraform.NewContext: complete")

	return &Context{
		hooks:   hooks,
		meta:    opts.Meta,
		uiInput: opts.UIInput,

		plugins:                plugins,
		dependencyLocks:        opts.LockedDependencies,
		providersInDevelopment: opts.ProvidersInDevelopment,

		parallelSem:         NewSemaphore(par),
		providerInputConfig: make(map[string]map[string]cty.Value),
		providerSHA256s:     opts.ProviderSHA256s,
		sh:                  sh,
	}, diags
}

func (c *Context) Schemas(config *configs.Config, state *states.State) (*Schemas, tfdiags.Diagnostics) {
	// TODO: This method gets called multiple times on the same context with
	// the same inputs by different parts of Terraform that all need the
	// schemas, and it's typically quite expensive because it has to spin up
	// plugins to gather their schemas, so it'd be good to have some caching
	// here to remember plugin schemas we already loaded since the plugin
	// selections can't change during the life of a *Context object.

	var diags tfdiags.Diagnostics

	// If we have a configuration and a set of locked dependencies, verify that
	// the provider requirements from the configuration can be satisfied by the
	// locked dependencies.
	if c.dependencyLocks != nil && config != nil {
		reqs, providerDiags := config.ProviderRequirements()
		diags = diags.Append(providerDiags)

		locked := c.dependencyLocks.AllProviders()
		unmetReqs := make(getproviders.Requirements)
		for provider, versionConstraints := range reqs {
			// Builtin providers are not listed in the locks file
			if provider.IsBuiltIn() {
				continue
			}
			// Development providers must be excluded from this check
			if _, ok := c.providersInDevelopment[provider]; ok {
				continue
			}
			// If the required provider doesn't exist in the lock, or the
			// locked version doesn't meet the constraints, mark the
			// requirement unmet
			acceptable := versions.MeetingConstraints(versionConstraints)
			if lock, ok := locked[provider]; !ok || !acceptable.Has(lock.Version()) {
				unmetReqs[provider] = versionConstraints
			}
		}

		if len(unmetReqs) > 0 {
			var buf strings.Builder
			for provider, versionConstraints := range unmetReqs {
				fmt.Fprintf(&buf, "\n- %s", provider)
				if len(versionConstraints) > 0 {
					fmt.Fprintf(&buf, " (%s)", getproviders.VersionConstraintsString(versionConstraints))
				}
			}
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Provider requirements cannot be satisfied by locked dependencies",
				fmt.Sprintf("The following required providers are not installed:\n%s\n\nPlease run \"terraform init\".", buf.String()),
			))
			return nil, diags
		}
	}

	ret, err := loadSchemas(config, state, c.plugins)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to load plugin schemas",
			fmt.Sprintf("Error while loading schemas for plugin components: %s.", err),
		))
		return nil, diags
	}
	return ret, diags
}

type ContextGraphOpts struct {
	// If true, validates the graph structure (checks for cycles).
	Validate bool

	// Legacy graphs only: won't prune the graph
	Verbose bool
}

// Stop stops the running task.
//
// Stop will block until the task completes.
func (c *Context) Stop() {
	log.Printf("[WARN] terraform: Stop called, initiating interrupt sequence")

	c.l.Lock()
	defer c.l.Unlock()

	// If we're running, then stop
	if c.runContextCancel != nil {
		log.Printf("[WARN] terraform: run context exists, stopping")

		// Tell the hook we want to stop
		c.sh.Stop()

		// Stop the context
		c.runContextCancel()
		c.runContextCancel = nil
	}

	// Grab the condition var before we exit
	if cond := c.runCond; cond != nil {
		log.Printf("[INFO] terraform: waiting for graceful stop to complete")
		cond.Wait()
	}

	log.Printf("[WARN] terraform: stop complete")
}

func (c *Context) acquireRun(phase string) func() {
	// With the run lock held, grab the context lock to make changes
	// to the run context.
	c.l.Lock()
	defer c.l.Unlock()

	// Wait until we're no longer running
	for c.runCond != nil {
		c.runCond.Wait()
	}

	// Build our lock
	c.runCond = sync.NewCond(&c.l)

	// Create a new run context
	c.runContext, c.runContextCancel = context.WithCancel(context.Background())

	// Reset the stop hook so we're not stopped
	c.sh.Reset()

	return c.releaseRun
}

func (c *Context) releaseRun() {
	// Grab the context lock so that we can make modifications to fields
	c.l.Lock()
	defer c.l.Unlock()

	// End our run. We check if runContext is non-nil because it can be
	// set to nil if it was cancelled via Stop()
	if c.runContextCancel != nil {
		c.runContextCancel()
	}

	// Unlock all waiting our condition
	cond := c.runCond
	c.runCond = nil
	cond.Broadcast()

	// Unset the context
	c.runContext = nil
}

// watchStop immediately returns a `stop` and a `wait` chan after dispatching
// the watchStop goroutine. This will watch the runContext for cancellation and
// stop the providers accordingly.  When the watch is no longer needed, the
// `stop` chan should be closed before waiting on the `wait` chan.
// The `wait` chan is important, because without synchronizing with the end of
// the watchStop goroutine, the runContext may also be closed during the select
// incorrectly causing providers to be stopped. Even if the graph walk is done
// at that point, stopping a provider permanently cancels its StopContext which
// can cause later actions to fail.
func (c *Context) watchStop(walker *ContextGraphWalker) (chan struct{}, <-chan struct{}) {
	stop := make(chan struct{})
	wait := make(chan struct{})

	// get the runContext cancellation channel now, because releaseRun will
	// write to the runContext field.
	done := c.runContext.Done()

	go func() {
		defer close(wait)
		// Wait for a stop or completion
		select {
		case <-done:
			// done means the context was canceled, so we need to try and stop
			// providers.
		case <-stop:
			// our own stop channel was closed.
			return
		}

		// If we're here, we're stopped, trigger the call.
		log.Printf("[TRACE] Context: requesting providers and provisioners to gracefully stop")

		{
			// Copy the providers so that a misbehaved blocking Stop doesn't
			// completely hang Terraform.
			walker.providerLock.Lock()
			ps := make([]providers.Interface, 0, len(walker.providerCache))
			for _, p := range walker.providerCache {
				ps = append(ps, p)
			}
			defer walker.providerLock.Unlock()

			for _, p := range ps {
				// We ignore the error for now since there isn't any reasonable
				// action to take if there is an error here, since the stop is still
				// advisory: Terraform will exit once the graph node completes.
				p.Stop()
			}
		}

		{
			// Call stop on all the provisioners
			walker.provisionerLock.Lock()
			ps := make([]provisioners.Interface, 0, len(walker.provisionerCache))
			for _, p := range walker.provisionerCache {
				ps = append(ps, p)
			}
			defer walker.provisionerLock.Unlock()

			for _, p := range ps {
				// We ignore the error for now since there isn't any reasonable
				// action to take if there is an error here, since the stop is still
				// advisory: Terraform will exit once the graph node completes.
				p.Stop()
			}
		}
	}()

	return stop, wait
}
