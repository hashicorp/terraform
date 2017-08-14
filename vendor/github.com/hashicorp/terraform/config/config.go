// The config package is responsible for loading and validating the
// configuration.
package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
	"github.com/hashicorp/terraform/helper/hilmapstructure"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/mitchellh/reflectwalk"
)

// NameRegexp is the regular expression that all names (modules, providers,
// resources, etc.) must follow.
var NameRegexp = regexp.MustCompile(`(?i)\A[A-Z0-9_][A-Z0-9\-\_]*\z`)

// Config is the configuration that comes from loading a collection
// of Terraform templates.
type Config struct {
	// Dir is the path to the directory where this configuration was
	// loaded from. If it is blank, this configuration wasn't loaded from
	// any meaningful directory.
	Dir string

	Terraform       *Terraform
	Atlas           *AtlasConfig
	Modules         []*Module
	ProviderConfigs []*ProviderConfig
	Resources       []*Resource
	Variables       []*Variable
	Outputs         []*Output

	// The fields below can be filled in by loaders for validation
	// purposes.
	unknownKeys []string
}

// AtlasConfig is the configuration for building in HashiCorp's Atlas.
type AtlasConfig struct {
	Name    string
	Include []string
	Exclude []string
}

// Module is a module used within a configuration.
//
// This does not represent a module itself, this represents a module
// call-site within an existing configuration.
type Module struct {
	Name      string
	Source    string
	RawConfig *RawConfig
}

// ProviderConfig is the configuration for a resource provider.
//
// For example, Terraform needs to set the AWS access keys for the AWS
// resource provider.
type ProviderConfig struct {
	Name      string
	Alias     string
	Version   string
	RawConfig *RawConfig
}

// A resource represents a single Terraform resource in the configuration.
// A Terraform resource is something that supports some or all of the
// usual "create, read, update, delete" operations, depending on
// the given Mode.
type Resource struct {
	Mode         ResourceMode // which operations the resource supports
	Name         string
	Type         string
	RawCount     *RawConfig
	RawConfig    *RawConfig
	Provisioners []*Provisioner
	Provider     string
	DependsOn    []string
	Lifecycle    ResourceLifecycle
}

// Copy returns a copy of this Resource. Helpful for avoiding shared
// config pointers across multiple pieces of the graph that need to do
// interpolation.
func (r *Resource) Copy() *Resource {
	n := &Resource{
		Mode:         r.Mode,
		Name:         r.Name,
		Type:         r.Type,
		RawCount:     r.RawCount.Copy(),
		RawConfig:    r.RawConfig.Copy(),
		Provisioners: make([]*Provisioner, 0, len(r.Provisioners)),
		Provider:     r.Provider,
		DependsOn:    make([]string, len(r.DependsOn)),
		Lifecycle:    *r.Lifecycle.Copy(),
	}
	for _, p := range r.Provisioners {
		n.Provisioners = append(n.Provisioners, p.Copy())
	}
	copy(n.DependsOn, r.DependsOn)
	return n
}

// ResourceLifecycle is used to store the lifecycle tuning parameters
// to allow customized behavior
type ResourceLifecycle struct {
	CreateBeforeDestroy bool     `mapstructure:"create_before_destroy"`
	PreventDestroy      bool     `mapstructure:"prevent_destroy"`
	IgnoreChanges       []string `mapstructure:"ignore_changes"`
}

// Copy returns a copy of this ResourceLifecycle
func (r *ResourceLifecycle) Copy() *ResourceLifecycle {
	n := &ResourceLifecycle{
		CreateBeforeDestroy: r.CreateBeforeDestroy,
		PreventDestroy:      r.PreventDestroy,
		IgnoreChanges:       make([]string, len(r.IgnoreChanges)),
	}
	copy(n.IgnoreChanges, r.IgnoreChanges)
	return n
}

// Provisioner is a configured provisioner step on a resource.
type Provisioner struct {
	Type      string
	RawConfig *RawConfig
	ConnInfo  *RawConfig

	When      ProvisionerWhen
	OnFailure ProvisionerOnFailure
}

// Copy returns a copy of this Provisioner
func (p *Provisioner) Copy() *Provisioner {
	return &Provisioner{
		Type:      p.Type,
		RawConfig: p.RawConfig.Copy(),
		ConnInfo:  p.ConnInfo.Copy(),
		When:      p.When,
		OnFailure: p.OnFailure,
	}
}

// Variable is a variable defined within the configuration.
type Variable struct {
	Name         string
	DeclaredType string `mapstructure:"type"`
	Default      interface{}
	Description  string
}

// Output is an output defined within the configuration. An output is
// resulting data that is highlighted by Terraform when finished. An
// output marked Sensitive will be output in a masked form following
// application, but will still be available in state.
type Output struct {
	Name        string
	DependsOn   []string
	Description string
	Sensitive   bool
	RawConfig   *RawConfig
}

// VariableType is the type of value a variable is holding, and returned
// by the Type() function on variables.
type VariableType byte

const (
	VariableTypeUnknown VariableType = iota
	VariableTypeString
	VariableTypeList
	VariableTypeMap
)

func (v VariableType) Printable() string {
	switch v {
	case VariableTypeString:
		return "string"
	case VariableTypeMap:
		return "map"
	case VariableTypeList:
		return "list"
	default:
		return "unknown"
	}
}

// ProviderConfigName returns the name of the provider configuration in
// the given mapping that maps to the proper provider configuration
// for this resource.
func ProviderConfigName(t string, pcs []*ProviderConfig) string {
	lk := ""
	for _, v := range pcs {
		k := v.Name
		if strings.HasPrefix(t, k) && len(k) > len(lk) {
			lk = k
		}
	}

	return lk
}

// A unique identifier for this module.
func (r *Module) Id() string {
	return fmt.Sprintf("%s", r.Name)
}

// Count returns the count of this resource.
func (r *Resource) Count() (int, error) {
	raw := r.RawCount.Value()
	count, ok := r.RawCount.Value().(string)
	if !ok {
		return 0, fmt.Errorf(
			"expected count to be a string or int, got %T", raw)
	}

	v, err := strconv.ParseInt(count, 0, 0)
	if err != nil {
		return 0, err
	}

	return int(v), nil
}

// A unique identifier for this resource.
func (r *Resource) Id() string {
	switch r.Mode {
	case ManagedResourceMode:
		return fmt.Sprintf("%s.%s", r.Type, r.Name)
	case DataResourceMode:
		return fmt.Sprintf("data.%s.%s", r.Type, r.Name)
	default:
		panic(fmt.Errorf("unknown resource mode %s", r.Mode))
	}
}

// ProviderFullName returns the full name of the provider for this resource,
// which may either be specified explicitly using the "provider" meta-argument
// or implied by the prefix on the resource type name.
func (r *Resource) ProviderFullName() string {
	return ResourceProviderFullName(r.Type, r.Provider)
}

// ResourceProviderFullName returns the full (dependable) name of the
// provider for a hypothetical resource with the given resource type and
// explicit provider string. If the explicit provider string is empty then
// the provider name is inferred from the resource type name.
func ResourceProviderFullName(resourceType, explicitProvider string) string {
	if explicitProvider != "" {
		return explicitProvider
	}

	idx := strings.IndexRune(resourceType, '_')
	if idx == -1 {
		// If no underscores, the resource name is assumed to be
		// also the provider name, e.g. if the provider exposes
		// only a single resource of each type.
		return resourceType
	}

	return resourceType[:idx]
}

// Validate does some basic semantic checking of the configuration.
func (c *Config) Validate() error {
	if c == nil {
		return nil
	}

	var errs []error

	for _, k := range c.unknownKeys {
		errs = append(errs, fmt.Errorf(
			"Unknown root level key: %s", k))
	}

	// Validate the Terraform config
	if tf := c.Terraform; tf != nil {
		errs = append(errs, c.Terraform.Validate()...)
	}

	vars := c.InterpolatedVariables()
	varMap := make(map[string]*Variable)
	for _, v := range c.Variables {
		if _, ok := varMap[v.Name]; ok {
			errs = append(errs, fmt.Errorf(
				"Variable '%s': duplicate found. Variable names must be unique.",
				v.Name))
		}

		varMap[v.Name] = v
	}

	for k, _ := range varMap {
		if !NameRegexp.MatchString(k) {
			errs = append(errs, fmt.Errorf(
				"variable %q: variable name must match regular expresion %s",
				k, NameRegexp))
		}
	}

	for _, v := range c.Variables {
		if v.Type() == VariableTypeUnknown {
			errs = append(errs, fmt.Errorf(
				"Variable '%s': must be a string or a map",
				v.Name))
			continue
		}

		interp := false
		fn := func(n ast.Node) (interface{}, error) {
			// LiteralNode is a literal string (outside of a ${ ... } sequence).
			// interpolationWalker skips most of these. but in particular it
			// visits those that have escaped sequences (like $${foo}) as a
			// signal that *some* processing is required on this string. For
			// our purposes here though, this is fine and not an interpolation.
			if _, ok := n.(*ast.LiteralNode); !ok {
				interp = true
			}
			return "", nil
		}

		w := &interpolationWalker{F: fn}
		if v.Default != nil {
			if err := reflectwalk.Walk(v.Default, w); err == nil {
				if interp {
					errs = append(errs, fmt.Errorf(
						"Variable '%s': cannot contain interpolations",
						v.Name))
				}
			}
		}
	}

	// Check for references to user variables that do not actually
	// exist and record those errors.
	for source, vs := range vars {
		for _, v := range vs {
			uv, ok := v.(*UserVariable)
			if !ok {
				continue
			}

			if _, ok := varMap[uv.Name]; !ok {
				errs = append(errs, fmt.Errorf(
					"%s: unknown variable referenced: '%s'. define it with 'variable' blocks",
					source,
					uv.Name))
			}
		}
	}

	// Check that all count variables are valid.
	for source, vs := range vars {
		for _, rawV := range vs {
			switch v := rawV.(type) {
			case *CountVariable:
				if v.Type == CountValueInvalid {
					errs = append(errs, fmt.Errorf(
						"%s: invalid count variable: %s",
						source,
						v.FullKey()))
				}
			case *PathVariable:
				if v.Type == PathValueInvalid {
					errs = append(errs, fmt.Errorf(
						"%s: invalid path variable: %s",
						source,
						v.FullKey()))
				}
			}
		}
	}

	// Check that providers aren't declared multiple times and that their
	// version constraints, where present, are syntactically valid.
	providerSet := make(map[string]struct{})
	for _, p := range c.ProviderConfigs {
		name := p.FullName()
		if _, ok := providerSet[name]; ok {
			errs = append(errs, fmt.Errorf(
				"provider.%s: declared multiple times, you can only declare a provider once",
				name))
			continue
		}

		if p.Version != "" {
			_, err := discovery.ConstraintStr(p.Version).Parse()
			if err != nil {
				errs = append(errs, fmt.Errorf(
					"provider.%s: invalid version constraint %q: %s",
					name, p.Version, err,
				))
			}
		}

		providerSet[name] = struct{}{}
	}

	// Check that all references to modules are valid
	modules := make(map[string]*Module)
	dupped := make(map[string]struct{})
	for _, m := range c.Modules {
		// Check for duplicates
		if _, ok := modules[m.Id()]; ok {
			if _, ok := dupped[m.Id()]; !ok {
				dupped[m.Id()] = struct{}{}

				errs = append(errs, fmt.Errorf(
					"%s: module repeated multiple times",
					m.Id()))
			}

			// Already seen this module, just skip it
			continue
		}

		modules[m.Id()] = m

		// Check that the source has no interpolations
		rc, err := NewRawConfig(map[string]interface{}{
			"root": m.Source,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"%s: module source error: %s",
				m.Id(), err))
		} else if len(rc.Interpolations) > 0 {
			errs = append(errs, fmt.Errorf(
				"%s: module source cannot contain interpolations",
				m.Id()))
		}

		// Check that the name matches our regexp
		if !NameRegexp.Match([]byte(m.Name)) {
			errs = append(errs, fmt.Errorf(
				"%s: module name can only contain letters, numbers, "+
					"dashes, and underscores",
				m.Id()))
		}

		// Check that the configuration can all be strings, lists or maps
		raw := make(map[string]interface{})
		for k, v := range m.RawConfig.Raw {
			var strVal string
			if err := hilmapstructure.WeakDecode(v, &strVal); err == nil {
				raw[k] = strVal
				continue
			}

			var mapVal map[string]interface{}
			if err := hilmapstructure.WeakDecode(v, &mapVal); err == nil {
				raw[k] = mapVal
				continue
			}

			var sliceVal []interface{}
			if err := hilmapstructure.WeakDecode(v, &sliceVal); err == nil {
				raw[k] = sliceVal
				continue
			}

			errs = append(errs, fmt.Errorf(
				"%s: variable %s must be a string, list or map value",
				m.Id(), k))
		}

		// Check for invalid count variables
		for _, v := range m.RawConfig.Variables {
			switch v.(type) {
			case *CountVariable:
				errs = append(errs, fmt.Errorf(
					"%s: count variables are only valid within resources", m.Name))
			case *SelfVariable:
				errs = append(errs, fmt.Errorf(
					"%s: self variables are only valid within resources", m.Name))
			}
		}

		// Update the raw configuration to only contain the string values
		m.RawConfig, err = NewRawConfig(raw)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"%s: can't initialize configuration: %s",
				m.Id(), err))
		}
	}
	dupped = nil

	// Check that all variables for modules reference modules that
	// exist.
	for source, vs := range vars {
		for _, v := range vs {
			mv, ok := v.(*ModuleVariable)
			if !ok {
				continue
			}

			if _, ok := modules[mv.Name]; !ok {
				errs = append(errs, fmt.Errorf(
					"%s: unknown module referenced: %s",
					source,
					mv.Name))
			}
		}
	}

	// Check that all references to resources are valid
	resources := make(map[string]*Resource)
	dupped = make(map[string]struct{})
	for _, r := range c.Resources {
		if _, ok := resources[r.Id()]; ok {
			if _, ok := dupped[r.Id()]; !ok {
				dupped[r.Id()] = struct{}{}

				errs = append(errs, fmt.Errorf(
					"%s: resource repeated multiple times",
					r.Id()))
			}
		}

		resources[r.Id()] = r
	}
	dupped = nil

	// Validate resources
	for n, r := range resources {
		// Verify count variables
		for _, v := range r.RawCount.Variables {
			switch v.(type) {
			case *CountVariable:
				errs = append(errs, fmt.Errorf(
					"%s: resource count can't reference count variable: %s",
					n,
					v.FullKey()))
			case *SimpleVariable:
				errs = append(errs, fmt.Errorf(
					"%s: resource count can't reference variable: %s",
					n,
					v.FullKey()))

			// Good
			case *ModuleVariable:
			case *ResourceVariable:
			case *TerraformVariable:
			case *UserVariable:

			default:
				errs = append(errs, fmt.Errorf(
					"Internal error. Unknown type in count var in %s: %T",
					n, v))
			}
		}

		// Interpolate with a fixed number to verify that its a number.
		r.RawCount.interpolate(func(root ast.Node) (interface{}, error) {
			// Execute the node but transform the AST so that it returns
			// a fixed value of "5" for all interpolations.
			result, err := hil.Eval(
				hil.FixedValueTransform(
					root, &ast.LiteralNode{Value: "5", Typex: ast.TypeString}),
				nil)
			if err != nil {
				return "", err
			}

			return result.Value, nil
		})
		_, err := strconv.ParseInt(r.RawCount.Value().(string), 0, 0)
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"%s: resource count must be an integer",
				n))
		}
		r.RawCount.init()

		// Validate DependsOn
		errs = append(errs, c.validateDependsOn(n, r.DependsOn, resources, modules)...)

		// Verify provisioners
		for _, p := range r.Provisioners {
			// This validation checks that there are no splat variables
			// referencing ourself. This currently is not allowed.

			for _, v := range p.ConnInfo.Variables {
				rv, ok := v.(*ResourceVariable)
				if !ok {
					continue
				}

				if rv.Multi && rv.Index == -1 && rv.Type == r.Type && rv.Name == r.Name {
					errs = append(errs, fmt.Errorf(
						"%s: connection info cannot contain splat variable "+
							"referencing itself", n))
					break
				}
			}

			for _, v := range p.RawConfig.Variables {
				rv, ok := v.(*ResourceVariable)
				if !ok {
					continue
				}

				if rv.Multi && rv.Index == -1 && rv.Type == r.Type && rv.Name == r.Name {
					errs = append(errs, fmt.Errorf(
						"%s: connection info cannot contain splat variable "+
							"referencing itself", n))
					break
				}
			}

			// Check for invalid when/onFailure values, though this should be
			// picked up by the loader we check here just in case.
			if p.When == ProvisionerWhenInvalid {
				errs = append(errs, fmt.Errorf(
					"%s: provisioner 'when' value is invalid", n))
			}
			if p.OnFailure == ProvisionerOnFailureInvalid {
				errs = append(errs, fmt.Errorf(
					"%s: provisioner 'on_failure' value is invalid", n))
			}
		}

		// Verify ignore_changes contains valid entries
		for _, v := range r.Lifecycle.IgnoreChanges {
			if strings.Contains(v, "*") && v != "*" {
				errs = append(errs, fmt.Errorf(
					"%s: ignore_changes does not support using a partial string "+
						"together with a wildcard: %s", n, v))
			}
		}

		// Verify ignore_changes has no interpolations
		rc, err := NewRawConfig(map[string]interface{}{
			"root": r.Lifecycle.IgnoreChanges,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"%s: lifecycle ignore_changes error: %s",
				n, err))
		} else if len(rc.Interpolations) > 0 {
			errs = append(errs, fmt.Errorf(
				"%s: lifecycle ignore_changes cannot contain interpolations",
				n))
		}

		// If it is a data source then it can't have provisioners
		if r.Mode == DataResourceMode {
			if _, ok := r.RawConfig.Raw["provisioner"]; ok {
				errs = append(errs, fmt.Errorf(
					"%s: data sources cannot have provisioners",
					n))
			}
		}
	}

	for source, vs := range vars {
		for _, v := range vs {
			rv, ok := v.(*ResourceVariable)
			if !ok {
				continue
			}

			id := rv.ResourceId()
			if _, ok := resources[id]; !ok {
				errs = append(errs, fmt.Errorf(
					"%s: unknown resource '%s' referenced in variable %s",
					source,
					id,
					rv.FullKey()))
				continue
			}
		}
	}

	// Check that all outputs are valid
	{
		found := make(map[string]struct{})
		for _, o := range c.Outputs {
			// Verify the output is new
			if _, ok := found[o.Name]; ok {
				errs = append(errs, fmt.Errorf(
					"%s: duplicate output. output names must be unique.",
					o.Name))
				continue
			}
			found[o.Name] = struct{}{}

			var invalidKeys []string
			valueKeyFound := false
			for k := range o.RawConfig.Raw {
				if k == "value" {
					valueKeyFound = true
					continue
				}
				if k == "sensitive" {
					if sensitive, ok := o.RawConfig.config[k].(bool); ok {
						if sensitive {
							o.Sensitive = true
						}
						continue
					}

					errs = append(errs, fmt.Errorf(
						"%s: value for 'sensitive' must be boolean",
						o.Name))
					continue
				}
				if k == "description" {
					if desc, ok := o.RawConfig.config[k].(string); ok {
						o.Description = desc
						continue
					}

					errs = append(errs, fmt.Errorf(
						"%s: value for 'description' must be string",
						o.Name))
					continue
				}
				invalidKeys = append(invalidKeys, k)
			}
			if len(invalidKeys) > 0 {
				errs = append(errs, fmt.Errorf(
					"%s: output has invalid keys: %s",
					o.Name, strings.Join(invalidKeys, ", ")))
			}
			if !valueKeyFound {
				errs = append(errs, fmt.Errorf(
					"%s: output is missing required 'value' key", o.Name))
			}

			for _, v := range o.RawConfig.Variables {
				if _, ok := v.(*CountVariable); ok {
					errs = append(errs, fmt.Errorf(
						"%s: count variables are only valid within resources", o.Name))
				}
			}
		}
	}

	// Validate the self variable
	for source, rc := range c.rawConfigs() {
		// Ignore provisioners. This is a pretty brittle way to do this,
		// but better than also repeating all the resources.
		if strings.Contains(source, "provision") {
			continue
		}

		for _, v := range rc.Variables {
			if _, ok := v.(*SelfVariable); ok {
				errs = append(errs, fmt.Errorf(
					"%s: cannot contain self-reference %s", source, v.FullKey()))
			}
		}
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	return nil
}

// InterpolatedVariables is a helper that returns a mapping of all the interpolated
// variables within the configuration. This is used to verify references
// are valid in the Validate step.
func (c *Config) InterpolatedVariables() map[string][]InterpolatedVariable {
	result := make(map[string][]InterpolatedVariable)
	for source, rc := range c.rawConfigs() {
		for _, v := range rc.Variables {
			result[source] = append(result[source], v)
		}
	}
	return result
}

// rawConfigs returns all of the RawConfigs that are available keyed by
// a human-friendly source.
func (c *Config) rawConfigs() map[string]*RawConfig {
	result := make(map[string]*RawConfig)
	for _, m := range c.Modules {
		source := fmt.Sprintf("module '%s'", m.Name)
		result[source] = m.RawConfig
	}

	for _, pc := range c.ProviderConfigs {
		source := fmt.Sprintf("provider config '%s'", pc.Name)
		result[source] = pc.RawConfig
	}

	for _, rc := range c.Resources {
		source := fmt.Sprintf("resource '%s'", rc.Id())
		result[source+" count"] = rc.RawCount
		result[source+" config"] = rc.RawConfig

		for i, p := range rc.Provisioners {
			subsource := fmt.Sprintf(
				"%s provisioner %s (#%d)",
				source, p.Type, i+1)
			result[subsource] = p.RawConfig
		}
	}

	for _, o := range c.Outputs {
		source := fmt.Sprintf("output '%s'", o.Name)
		result[source] = o.RawConfig
	}

	return result
}

func (c *Config) validateDependsOn(
	n string,
	v []string,
	resources map[string]*Resource,
	modules map[string]*Module) []error {
	// Verify depends on points to resources that all exist
	var errs []error
	for _, d := range v {
		// Check if we contain interpolations
		rc, err := NewRawConfig(map[string]interface{}{
			"value": d,
		})
		if err == nil && len(rc.Variables) > 0 {
			errs = append(errs, fmt.Errorf(
				"%s: depends on value cannot contain interpolations: %s",
				n, d))
			continue
		}

		// If it is a module, verify it is a module
		if strings.HasPrefix(d, "module.") {
			name := d[len("module."):]
			if _, ok := modules[name]; !ok {
				errs = append(errs, fmt.Errorf(
					"%s: resource depends on non-existent module '%s'",
					n, name))
			}

			continue
		}

		// Check resources
		if _, ok := resources[d]; !ok {
			errs = append(errs, fmt.Errorf(
				"%s: resource depends on non-existent resource '%s'",
				n, d))
		}
	}

	return errs
}

func (m *Module) mergerName() string {
	return m.Id()
}

func (m *Module) mergerMerge(other merger) merger {
	m2 := other.(*Module)

	result := *m
	result.Name = m2.Name
	result.RawConfig = result.RawConfig.merge(m2.RawConfig)

	if m2.Source != "" {
		result.Source = m2.Source
	}

	return &result
}

func (o *Output) mergerName() string {
	return o.Name
}

func (o *Output) mergerMerge(m merger) merger {
	o2 := m.(*Output)

	result := *o
	result.Name = o2.Name
	result.Description = o2.Description
	result.RawConfig = result.RawConfig.merge(o2.RawConfig)
	result.Sensitive = o2.Sensitive
	result.DependsOn = o2.DependsOn

	return &result
}

func (c *ProviderConfig) GoString() string {
	return fmt.Sprintf("*%#v", *c)
}

func (c *ProviderConfig) FullName() string {
	if c.Alias == "" {
		return c.Name
	}

	return fmt.Sprintf("%s.%s", c.Name, c.Alias)
}

func (c *ProviderConfig) mergerName() string {
	return c.Name
}

func (c *ProviderConfig) mergerMerge(m merger) merger {
	c2 := m.(*ProviderConfig)

	result := *c
	result.Name = c2.Name
	result.RawConfig = result.RawConfig.merge(c2.RawConfig)

	if c2.Alias != "" {
		result.Alias = c2.Alias
	}

	return &result
}

func (r *Resource) mergerName() string {
	return r.Id()
}

func (r *Resource) mergerMerge(m merger) merger {
	r2 := m.(*Resource)

	result := *r
	result.Mode = r2.Mode
	result.Name = r2.Name
	result.Type = r2.Type
	result.RawConfig = result.RawConfig.merge(r2.RawConfig)

	if r2.RawCount.Value() != "1" {
		result.RawCount = r2.RawCount
	}

	if len(r2.Provisioners) > 0 {
		result.Provisioners = r2.Provisioners
	}

	return &result
}

// Merge merges two variables to create a new third variable.
func (v *Variable) Merge(v2 *Variable) *Variable {
	// Shallow copy the variable
	result := *v

	// The names should be the same, but the second name always wins.
	result.Name = v2.Name

	if v2.DeclaredType != "" {
		result.DeclaredType = v2.DeclaredType
	}
	if v2.Default != nil {
		result.Default = v2.Default
	}
	if v2.Description != "" {
		result.Description = v2.Description
	}

	return &result
}

var typeStringMap = map[string]VariableType{
	"string": VariableTypeString,
	"map":    VariableTypeMap,
	"list":   VariableTypeList,
}

// Type returns the type of variable this is.
func (v *Variable) Type() VariableType {
	if v.DeclaredType != "" {
		declaredType, ok := typeStringMap[v.DeclaredType]
		if !ok {
			return VariableTypeUnknown
		}

		return declaredType
	}

	return v.inferTypeFromDefault()
}

// ValidateTypeAndDefault ensures that default variable value is compatible
// with the declared type (if one exists), and that the type is one which is
// known to Terraform
func (v *Variable) ValidateTypeAndDefault() error {
	// If an explicit type is declared, ensure it is valid
	if v.DeclaredType != "" {
		if _, ok := typeStringMap[v.DeclaredType]; !ok {
			validTypes := []string{}
			for k := range typeStringMap {
				validTypes = append(validTypes, k)
			}
			return fmt.Errorf(
				"Variable '%s' type must be one of [%s] - '%s' is not a valid type",
				v.Name,
				strings.Join(validTypes, ", "),
				v.DeclaredType,
			)
		}
	}

	if v.DeclaredType == "" || v.Default == nil {
		return nil
	}

	if v.inferTypeFromDefault() != v.Type() {
		return fmt.Errorf("'%s' has a default value which is not of type '%s' (got '%s')",
			v.Name, v.DeclaredType, v.inferTypeFromDefault().Printable())
	}

	return nil
}

func (v *Variable) mergerName() string {
	return v.Name
}

func (v *Variable) mergerMerge(m merger) merger {
	return v.Merge(m.(*Variable))
}

// Required tests whether a variable is required or not.
func (v *Variable) Required() bool {
	return v.Default == nil
}

// inferTypeFromDefault contains the logic for the old method of inferring
// variable types - we can also use this for validating that the declared
// type matches the type of the default value
func (v *Variable) inferTypeFromDefault() VariableType {
	if v.Default == nil {
		return VariableTypeString
	}

	var s string
	if err := hilmapstructure.WeakDecode(v.Default, &s); err == nil {
		v.Default = s
		return VariableTypeString
	}

	var m map[string]interface{}
	if err := hilmapstructure.WeakDecode(v.Default, &m); err == nil {
		v.Default = m
		return VariableTypeMap
	}

	var l []interface{}
	if err := hilmapstructure.WeakDecode(v.Default, &l); err == nil {
		v.Default = l
		return VariableTypeList
	}

	return VariableTypeUnknown
}

func (m ResourceMode) Taintable() bool {
	switch m {
	case ManagedResourceMode:
		return true
	case DataResourceMode:
		return false
	default:
		panic(fmt.Errorf("unsupported ResourceMode value %s", m))
	}
}
