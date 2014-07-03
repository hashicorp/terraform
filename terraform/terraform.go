package terraform

// Terraform is the primary structure that is used to interact with
// Terraform from code, and can perform operations such as returning
// all resources, a resource tree, a specific resource, etc.
type Terraform struct {
	hooks     []Hook
	providers map[string]ResourceProviderFactory
	stopHook  *stopHook
}

// This is a function type used to implement a walker for the resource
// tree internally on the Terraform structure.
type genericWalkFunc func(*Resource) (map[string]string, error)

// Config is the configuration that must be given to instantiate
// a Terraform structure.
type Config struct {
	Hooks     []Hook
	Providers map[string]ResourceProviderFactory
}

/*
// New creates a new Terraform structure, initializes resource providers
// for the given configuration, etc.
//
// Semantic checks of the entire configuration structure are done at this
// time, as well as richer checks such as verifying that the resource providers
// can be properly initialized, can be configured, etc.
func New(c *Config) (*Terraform, error) {
	sh := new(stopHook)
	sh.Lock()
	sh.reset()
	sh.Unlock()

	// Copy all the hooks and add our stop hook. We don't append directly
	// to the Config so that we're not modifying that in-place.
	hooks := make([]Hook, len(c.Hooks)+1)
	copy(hooks, c.Hooks)
	hooks[len(c.Hooks)] = sh

	return &Terraform{
		hooks:     hooks,
		stopHook:  sh,
		providers: c.Providers,
	}, nil
}

/*
// Stop stops all running tasks (applies, plans, refreshes).
//
// This will block until all running tasks are stopped. While Stop is
// blocked, any new calls to Apply, Plan, Refresh, etc. will also block. New
// calls, however, will start once this Stop has returned.
func (t *Terraform) Stop() {
	log.Printf("[INFO] Terraform stopping tasks")

	t.stopHook.Lock()
	defer t.stopHook.Unlock()

	// Setup the stoppedCh
	stoppedCh := make(chan struct{}, t.stopHook.count)
	t.stopHook.stoppedCh = stoppedCh

	// Close the channel to signal that we're done
	close(t.stopHook.ch)

	// Expect the number of count stops...
	log.Printf("[DEBUG] Waiting for %d tasks to stop", t.stopHook.count)
	for i := 0; i < t.stopHook.count; i++ {
		<-stoppedCh
	}
	log.Printf("[DEBUG] Stopped!")

	// Success, everything stopped, reset everything
	t.stopHook.reset()
}
*/
