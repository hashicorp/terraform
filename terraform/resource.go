package terraform

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/config"
)

// ResourceProvisionerConfig is used to pair a provisioner
// with it's provided configuration. This allows us to use singleton
// instances of each ResourceProvisioner and to keep the relevant
// configuration instead of instantiating a new Provisioner for each
// resource.
type ResourceProvisionerConfig struct {
	Provisioner ResourceProvisioner
	Config      *ResourceConfig
	RawConfig   *config.RawConfig
}

// Resource encapsulates a resource, its configuration, its provider,
// its current state, and potentially a desired diff from the state it
// wants to reach.
type Resource struct {
	Id           string
	Config       *ResourceConfig
	Diff         *ResourceDiff
	Provider     ResourceProvider
	State        *ResourceState
	Provisioners []*ResourceProvisionerConfig
}

// Vars returns the mapping of variables that should be replaced in
// configuration based on the attributes of this resource.
func (r *Resource) Vars() map[string]string {
	if r.State == nil {
		return nil
	}

	vars := make(map[string]string)
	for ak, av := range r.State.Attributes {
		vars[fmt.Sprintf("%s.%s", r.Id, ak)] = av
	}

	return vars
}

// ResourceConfig holds the configuration given for a resource. This is
// done instead of a raw `map[string]interface{}` type so that rich
// methods can be added to it to make dealing with it easier.
type ResourceConfig struct {
	ComputedKeys []string
	Raw          map[string]interface{}
	Config       map[string]interface{}

	raw *config.RawConfig
}

// NewResourceConfig creates a new ResourceConfig from a config.RawConfig.
func NewResourceConfig(c *config.RawConfig) *ResourceConfig {
	result := &ResourceConfig{raw: c}
	result.interpolate(nil)
	return result
}

// CheckSet checks that the given list of configuration keys is
// properly set. If not, errors are returned for each unset key.
//
// This is useful to be called in the Validate method of a ResourceProvider.
func (c *ResourceConfig) CheckSet(keys []string) []error {
	var errs []error

	for _, k := range keys {
		if !c.IsSet(k) {
			errs = append(errs, fmt.Errorf("%s must be set", k))
		}
	}

	return errs
}

// Get looks up a configuration value by key and returns the value.
//
// The second return value is true if the get was successful. Get will
// not succeed if the value is being computed.
func (c *ResourceConfig) Get(k string) (interface{}, bool) {
	parts := strings.Split(k, ".")

	var current interface{} = c.Raw
	for _, part := range parts {
		if current == nil {
			return nil, false
		}

		cv := reflect.ValueOf(current)
		switch cv.Kind() {
		case reflect.Map:
			v := cv.MapIndex(reflect.ValueOf(part))
			if !v.IsValid() {
				return nil, false
			}
			current = v.Interface()
		case reflect.Slice:
			i, err := strconv.ParseInt(part, 0, 0)
			if err != nil {
				return nil, false
			}
			current = cv.Index(int(i)).Interface()
		default:
			panic(fmt.Sprintf("Unknown kind: %s", cv.Kind()))
		}
	}

	return current, true
}

// IsSet checks if the key in the configuration is set. A key is set if
// it has a value or the value is being computed (is unknown currently).
//
// This function should be used rather than checking the keys of the
// raw configuration itself, since a key may be omitted from the raw
// configuration if it is being computed.
func (c *ResourceConfig) IsSet(k string) bool {
	if c == nil {
		return false
	}

	for _, ck := range c.ComputedKeys {
		if ck == k {
			return true
		}
	}

	if _, ok := c.Get(k); ok {
		return true
	}

	return false
}

func (c *ResourceConfig) interpolate(ctx *Context) error {
	if c == nil {
		return nil
	}

	if ctx != nil {
		if err := ctx.computeVars(c.raw); err != nil {
			return err
		}
	}

	if c.raw != nil {
		c.ComputedKeys = c.raw.UnknownKeys()
		c.Raw = c.raw.Raw
		c.Config = c.raw.Config()
	}

	return nil
}
