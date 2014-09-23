package terraform

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/config"
)

// ResourceProvisionerConfig is used to pair a provisioner
// with its provided configuration. This allows us to use singleton
// instances of each ResourceProvisioner and to keep the relevant
// configuration instead of instantiating a new Provisioner for each
// resource.
type ResourceProvisionerConfig struct {
	Type        string
	Provisioner ResourceProvisioner
	Config      *ResourceConfig
	RawConfig   *config.RawConfig
	ConnInfo    *config.RawConfig
}

// Resource encapsulates a resource, its configuration, its provider,
// its current state, and potentially a desired diff from the state it
// wants to reach.
type Resource struct {
	Id           string
	Info         *InstanceInfo
	Config       *ResourceConfig
	Diff         *InstanceDiff
	Provider     ResourceProvider
	State        *InstanceState
	Provisioners []*ResourceProvisionerConfig
	Flags        ResourceFlag
	TaintedIndex int
}

// ResourceKind specifies what kind of instance we're working with, whether
// its a primary instance, a tainted instance, or an orphan.
type ResourceFlag byte

const (
	FlagPrimary ResourceFlag = 1 << iota
	FlagTainted
	FlagOrphan
	FlagHasTainted
)

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
	// First try to get it from c.Config since that has interpolated values
	result, ok := c.get(k, c.Config)
	if ok {
		return result, ok
	}

	// Otherwise, just get it from the raw config
	return c.get(k, c.Raw)
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

func (c *ResourceConfig) get(
	k string, raw map[string]interface{}) (interface{}, bool) {
	parts := strings.Split(k, ".")

	var current interface{} = raw
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
			if part == "#" {
				current = cv.Len()
			} else {
				i, err := strconv.ParseInt(part, 0, 0)
				if err != nil {
					return nil, false
				}
				if i >= int64(cv.Len()) {
					return nil, false
				}
				current = cv.Index(int(i)).Interface()
			}
		default:
			panic(fmt.Sprintf("Unknown kind: %s", cv.Kind()))
		}
	}

	return current, true
}

func (c *ResourceConfig) interpolate(ctx *walkContext) error {
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
