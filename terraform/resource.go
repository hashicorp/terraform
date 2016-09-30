package terraform

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/mitchellh/copystructure"
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
	// These are all used by the new EvalNode stuff.
	Name       string
	Type       string
	CountIndex int

	// These aren't really used anymore anywhere, but we keep them around
	// since we haven't done a proper cleanup yet.
	Id           string
	Info         *InstanceInfo
	Config       *ResourceConfig
	Dependencies []string
	Diff         *InstanceDiff
	Provider     ResourceProvider
	State        *InstanceState
	Provisioners []*ResourceProvisionerConfig
	Flags        ResourceFlag
}

// ResourceKind specifies what kind of instance we're working with, whether
// its a primary instance, a tainted instance, or an orphan.
type ResourceFlag byte

// InstanceInfo is used to hold information about the instance and/or
// resource being modified.
type InstanceInfo struct {
	// Id is a unique name to represent this instance. This is not related
	// to InstanceState.ID in any way.
	Id string

	// ModulePath is the complete path of the module containing this
	// instance.
	ModulePath []string

	// Type is the resource type of this instance
	Type string
}

// HumanId is a unique Id that is human-friendly and useful for UI elements.
func (i *InstanceInfo) HumanId() string {
	if len(i.ModulePath) <= 1 {
		return i.Id
	}

	return fmt.Sprintf(
		"module.%s.%s",
		strings.Join(i.ModulePath[1:], "."),
		i.Id)
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
	result.interpolateForce()
	return result
}

// DeepCopy performs a deep copy of the configuration. This makes it safe
// to modify any of the structures that are part of the resource config without
// affecting the original configuration.
func (c *ResourceConfig) DeepCopy() *ResourceConfig {
	// DeepCopying a nil should return a nil to avoid panics
	if c == nil {
		return nil
	}

	// Copy, this will copy all the exported attributes
	copy, err := copystructure.Config{Lock: true}.Copy(c)
	if err != nil {
		panic(err)
	}

	// Force the type
	result := copy.(*ResourceConfig)

	// For the raw configuration, we can just use its own copy method
	result.raw = c.raw.Copy()

	return result
}

// Equal checks the equality of two resource configs.
func (c *ResourceConfig) Equal(c2 *ResourceConfig) bool {
	// If either are nil, then they're only equal if they're both nil
	if c == nil || c2 == nil {
		return c == c2
	}

	// Two resource configs if their exported properties are equal.
	// We don't compare "raw" because it is never used again after
	// initialization and for all intents and purposes they are equal
	// if the exported properties are equal.
	check := [][2]interface{}{
		{c.ComputedKeys, c2.ComputedKeys},
		{c.Raw, c2.Raw},
		{c.Config, c2.Config},
	}
	for _, pair := range check {
		if !reflect.DeepEqual(pair[0], pair[1]) {
			return false
		}
	}

	return true
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

// GetRaw looks up a configuration value by key and returns the value,
// from the raw, uninterpolated config.
//
// The second return value is true if the get was successful. Get will
// not succeed if the value is being computed.
func (c *ResourceConfig) GetRaw(k string) (interface{}, bool) {
	return c.get(k, c.Raw)
}

// IsComputed returns whether the given key is computed or not.
func (c *ResourceConfig) IsComputed(k string) bool {
	_, ok := c.get(k, c.Config)
	_, okRaw := c.get(k, c.Raw)
	return !ok && okRaw
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
	if len(parts) == 1 && parts[0] == "" {
		parts = nil
	}

	var current interface{} = raw
	var previous interface{} = nil
	for i, part := range parts {
		if current == nil {
			return nil, false
		}

		cv := reflect.ValueOf(current)
		switch cv.Kind() {
		case reflect.Map:
			previous = current
			v := cv.MapIndex(reflect.ValueOf(part))
			if !v.IsValid() {
				if i > 0 && i != (len(parts)-1) {
					tryKey := strings.Join(parts[i:], ".")
					v := cv.MapIndex(reflect.ValueOf(tryKey))
					if !v.IsValid() {
						return nil, false
					}
					return v.Interface(), true
				}

				return nil, false
			}
			current = v.Interface()
		case reflect.Slice:
			previous = current
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
		case reflect.String:
			// This happens when map keys contain "." and have a common
			// prefix so were split as path components above.
			actualKey := strings.Join(parts[i-1:], ".")
			if prevMap, ok := previous.(map[string]interface{}); ok {
				return prevMap[actualKey], true
			}
			return nil, false
		default:
			panic(fmt.Sprintf("Unknown kind: %s", cv.Kind()))
		}
	}

	return current, true
}

// interpolateForce is a temporary thing. We want to get rid of interpolate
// above and likewise this, but it can only be done after the f-ast-graph
// refactor is complete.
func (c *ResourceConfig) interpolateForce() {
	if c.raw == nil {
		var err error
		c.raw, err = config.NewRawConfig(make(map[string]interface{}))
		if err != nil {
			panic(err)
		}
	}

	c.ComputedKeys = c.raw.UnknownKeys()
	c.Raw = c.raw.RawMap()
	c.Config = c.raw.Config()
}
