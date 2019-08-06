package terraform

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/mitchellh/copystructure"
	"github.com/mitchellh/reflectwalk"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs/configschema"
)

// Resource is a legacy way to identify a particular resource instance.
//
// New code should use addrs.ResourceInstance instead. This is still here
// only for codepaths that haven't been updated yet.
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
	Flags        ResourceFlag
}

// NewResource constructs a legacy Resource object from an
// addrs.ResourceInstance value.
//
// This is provided to shim to old codepaths that haven't been updated away
// from this type yet. Since this old type is not able to represent instances
// that have string keys, this function will panic if given a resource address
// that has a string key.
func NewResource(addr addrs.ResourceInstance) *Resource {
	ret := &Resource{
		Name: addr.Resource.Name,
		Type: addr.Resource.Type,
	}

	if addr.Key != addrs.NoKey {
		switch tk := addr.Key.(type) {
		case addrs.IntKey:
			ret.CountIndex = int(tk)
		default:
			panic(fmt.Errorf("resource instance with key %#v is not supported", addr.Key))
		}
	}

	return ret
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

	// uniqueExtra is an internal field that can be populated to supply
	// extra metadata that is used to identify a unique instance in
	// the graph walk. This will be appended to HumanID when uniqueId
	// is called.
	uniqueExtra string
}

// NewInstanceInfo constructs an InstanceInfo from an addrs.AbsResourceInstance.
//
// InstanceInfo is a legacy type, and uses of it should be gradually replaced
// by direct use of addrs.AbsResource or addrs.AbsResourceInstance as
// appropriate.
//
// The legacy InstanceInfo type cannot represent module instances with instance
// keys, so this function will panic if given such a path. Uses of this type
// should all be removed or replaced before implementing "count" and "for_each"
// arguments on modules in order to avoid such panics.
//
// This legacy type also cannot represent resource instances with string
// instance keys. It will panic if the given key is not either NoKey or an
// IntKey.
func NewInstanceInfo(addr addrs.AbsResourceInstance) *InstanceInfo {
	// We need an old-style []string module path for InstanceInfo.
	path := make([]string, len(addr.Module))
	for i, step := range addr.Module {
		if step.InstanceKey != addrs.NoKey {
			panic("NewInstanceInfo cannot convert module instance with key")
		}
		path[i] = step.Name
	}

	// This is a funny old meaning of "id" that is no longer current. It should
	// not be used for anything users might see. Note that it does not include
	// a representation of the resource mode, and so it's impossible to
	// determine from an InstanceInfo alone whether it is a managed or data
	// resource that is being referred to.
	id := fmt.Sprintf("%s.%s", addr.Resource.Resource.Type, addr.Resource.Resource.Name)
	if addr.Resource.Resource.Mode == addrs.DataResourceMode {
		id = "data." + id
	}
	if addr.Resource.Key != addrs.NoKey {
		switch k := addr.Resource.Key.(type) {
		case addrs.IntKey:
			id = id + fmt.Sprintf(".%d", int(k))
		default:
			panic(fmt.Sprintf("NewInstanceInfo cannot convert resource instance with %T instance key", addr.Resource.Key))
		}
	}

	return &InstanceInfo{
		Id:         id,
		ModulePath: path,
		Type:       addr.Resource.Resource.Type,
	}
}

// ResourceAddress returns the address of the resource that the receiver is describing.
func (i *InstanceInfo) ResourceAddress() *ResourceAddress {
	// GROSS: for tainted and deposed instances, their status gets appended
	// to i.Id to create a unique id for the graph node. Historically these
	// ids were displayed to the user, so it's designed to be human-readable:
	//   "aws_instance.bar.0 (deposed #0)"
	//
	// So here we detect such suffixes and try to interpret them back to
	// their original meaning so we can then produce a ResourceAddress
	// with a suitable InstanceType.
	id := i.Id
	instanceType := TypeInvalid
	if idx := strings.Index(id, " ("); idx != -1 {
		remain := id[idx:]
		id = id[:idx]

		switch {
		case strings.Contains(remain, "tainted"):
			instanceType = TypeTainted
		case strings.Contains(remain, "deposed"):
			instanceType = TypeDeposed
		}
	}

	addr, err := parseResourceAddressInternal(id)
	if err != nil {
		// should never happen, since that would indicate a bug in the
		// code that constructed this InstanceInfo.
		panic(fmt.Errorf("InstanceInfo has invalid Id %s", id))
	}
	if len(i.ModulePath) > 1 {
		addr.Path = i.ModulePath[1:] // trim off "root" prefix, which is implied
	}
	if instanceType != TypeInvalid {
		addr.InstanceTypeSet = true
		addr.InstanceType = instanceType
	}
	return addr
}

// ResourceConfig is a legacy type that was formerly used to represent
// interpolatable configuration blocks. It is now only used to shim to old
// APIs that still use this type, via NewResourceConfigShimmed.
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

// NewResourceConfigRaw constructs a ResourceConfig whose content is exactly
// the given value.
//
// The given value may contain hcl2shim.UnknownVariableValue to signal that
// something is computed, but it must not contain unprocessed interpolation
// sequences as we might've seen in Terraform v0.11 and prior.
func NewResourceConfigRaw(raw map[string]interface{}) *ResourceConfig {
	v := hcl2shim.HCL2ValueFromConfigValue(raw)

	// This is a little weird but we round-trip the value through the hcl2shim
	// package here for two reasons: firstly, because that reduces the risk
	// of it including something unlike what NewResourceConfigShimmed would
	// produce, and secondly because it creates a copy of "raw" just in case
	// something is relying on the fact that in the old world the raw and
	// config maps were always distinct, and thus you could in principle mutate
	// one without affecting the other. (I sure hope nobody was doing that, though!)
	cfg := hcl2shim.ConfigValueFromHCL2(v).(map[string]interface{})

	return &ResourceConfig{
		Raw:    raw,
		Config: cfg,

		ComputedKeys: newResourceConfigShimmedComputedKeys(v, ""),
	}
}

// NewResourceConfigShimmed wraps a cty.Value of object type in a legacy
// ResourceConfig object, so that it can be passed to older APIs that expect
// this wrapping.
//
// The returned ResourceConfig is already interpolated and cannot be
// re-interpolated. It is, therefore, useful only to functions that expect
// an already-populated ResourceConfig which they then treat as read-only.
//
// If the given value is not of an object type that conforms to the given
// schema then this function will panic.
func NewResourceConfigShimmed(val cty.Value, schema *configschema.Block) *ResourceConfig {
	if !val.Type().IsObjectType() {
		panic(fmt.Errorf("NewResourceConfigShimmed given %#v; an object type is required", val.Type()))
	}
	ret := &ResourceConfig{}

	legacyVal := hcl2shim.ConfigValueFromHCL2Block(val, schema)
	if legacyVal != nil {
		ret.Config = legacyVal

		// Now we need to walk through our structure and find any unknown values,
		// producing the separate list ComputedKeys to represent these. We use the
		// schema here so that we can preserve the expected invariant
		// that an attribute is always either wholly known or wholly unknown, while
		// a child block can be partially unknown.
		ret.ComputedKeys = newResourceConfigShimmedComputedKeys(val, "")
	} else {
		ret.Config = make(map[string]interface{})
	}
	ret.Raw = ret.Config

	return ret
}

// Record the any config values in ComputedKeys. This field had been unused in
// helper/schema, but in the new protocol we're using this so that the SDK can
// now handle having an unknown collection. The legacy diff code doesn't
// properly handle the unknown, because it can't be expressed in the same way
// between the config and diff.
func newResourceConfigShimmedComputedKeys(val cty.Value, path string) []string {
	var ret []string
	ty := val.Type()

	if val.IsNull() {
		return ret
	}

	if !val.IsKnown() {
		// we shouldn't have an entirely unknown resource, but prevent empty
		// strings just in case
		if len(path) > 0 {
			ret = append(ret, path)
		}
		return ret
	}

	if path != "" {
		path += "."
	}
	switch {
	case ty.IsListType(), ty.IsTupleType(), ty.IsSetType():
		i := 0
		for it := val.ElementIterator(); it.Next(); i++ {
			_, subVal := it.Element()
			keys := newResourceConfigShimmedComputedKeys(subVal, fmt.Sprintf("%s%d", path, i))
			ret = append(ret, keys...)
		}

	case ty.IsMapType(), ty.IsObjectType():
		for it := val.ElementIterator(); it.Next(); {
			subK, subVal := it.Element()
			keys := newResourceConfigShimmedComputedKeys(subVal, fmt.Sprintf("%s%s", path, subK.AsString()))
			ret = append(ret, keys...)
		}
	}

	return ret
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

	return result
}

// Equal checks the equality of two resource configs.
func (c *ResourceConfig) Equal(c2 *ResourceConfig) bool {
	// If either are nil, then they're only equal if they're both nil
	if c == nil || c2 == nil {
		return c == c2
	}

	// Sort the computed keys so they're deterministic
	sort.Strings(c.ComputedKeys)
	sort.Strings(c2.ComputedKeys)

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
// return the raw value if the key is computed, so you should pair this
// with IsComputed.
func (c *ResourceConfig) Get(k string) (interface{}, bool) {
	// We aim to get a value from the configuration. If it is computed,
	// then we return the pure raw value.
	source := c.Config
	if c.IsComputed(k) {
		source = c.Raw
	}

	return c.get(k, source)
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
	// The next thing we do is check the config if we get a computed
	// value out of it.
	v, ok := c.get(k, c.Config)
	if !ok {
		return false
	}

	// If value is nil, then it isn't computed
	if v == nil {
		return false
	}

	// Test if the value contains an unknown value
	var w unknownCheckWalker
	if err := reflectwalk.Walk(v, &w); err != nil {
		panic(err)
	}

	return w.Unknown
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

	if c.IsComputed(k) {
		return true
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
				// If any value in a list is computed, this whole thing
				// is computed and we can't read any part of it.
				for i := 0; i < cv.Len(); i++ {
					if v := cv.Index(i).Interface(); v == hcl2shim.UnknownVariableValue {
						return v, true
					}
				}

				current = cv.Len()
			} else {
				i, err := strconv.ParseInt(part, 0, 0)
				if err != nil {
					return nil, false
				}
				if int(i) < 0 || int(i) >= cv.Len() {
					return nil, false
				}
				current = cv.Index(int(i)).Interface()
			}
		case reflect.String:
			// This happens when map keys contain "." and have a common
			// prefix so were split as path components above.
			actualKey := strings.Join(parts[i-1:], ".")
			if prevMap, ok := previous.(map[string]interface{}); ok {
				v, ok := prevMap[actualKey]
				return v, ok
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
		// If we don't have a lowercase "raw" but we _do_ have the uppercase
		// Raw populated then this indicates that we're recieving a shim
		// ResourceConfig created by NewResourceConfigShimmed, which is already
		// fully evaluated and thus this function doesn't need to do anything.
		if c.Raw != nil {
			return
		}

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

// unknownCheckWalker
type unknownCheckWalker struct {
	Unknown bool
}

func (w *unknownCheckWalker) Primitive(v reflect.Value) error {
	if v.Interface() == hcl2shim.UnknownVariableValue {
		w.Unknown = true
	}

	return nil
}
