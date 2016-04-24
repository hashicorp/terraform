package config

import (
	"bytes"
	"encoding/gob"
	"sync"

	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
	"github.com/mitchellh/copystructure"
	"github.com/mitchellh/reflectwalk"
)

// UnknownVariableValue is a sentinel value that can be used
// to denote that the value of a variable is unknown at this time.
// RawConfig uses this information to build up data about
// unknown keys.
const UnknownVariableValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

// RawConfig is a structure that holds a piece of configuration
// where te overall structure is unknown since it will be used
// to configure a plugin or some other similar external component.
//
// RawConfigs can be interpolated with variables that come from
// other resources, user variables, etc.
//
// RawConfig supports a query-like interface to request
// information from deep within the structure.
type RawConfig struct {
	Key            string
	Raw            map[string]interface{}
	Interpolations []ast.Node
	Variables      map[string]InterpolatedVariable

	lock        sync.Mutex
	config      map[string]interface{}
	unknownKeys []string
}

// NewRawConfig creates a new RawConfig structure and populates the
// publicly readable struct fields.
func NewRawConfig(raw map[string]interface{}) (*RawConfig, error) {
	result := &RawConfig{Raw: raw}
	if err := result.init(); err != nil {
		return nil, err
	}

	return result, nil
}

// Copy returns a copy of this RawConfig, uninterpolated.
func (r *RawConfig) Copy() *RawConfig {
	r.lock.Lock()
	defer r.lock.Unlock()

	newRaw := make(map[string]interface{})
	for k, v := range r.Raw {
		newRaw[k] = v
	}

	result, err := NewRawConfig(newRaw)
	if err != nil {
		panic("copy failed: " + err.Error())
	}

	result.Key = r.Key
	return result
}

// Value returns the value of the configuration if this configuration
// has a Key set. If this does not have a Key set, nil will be returned.
func (r *RawConfig) Value() interface{} {
	if c := r.Config(); c != nil {
		if v, ok := c[r.Key]; ok {
			return v
		}
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	return r.Raw[r.Key]
}

// Config returns the entire configuration with the variables
// interpolated from any call to Interpolate.
//
// If any interpolated variables are unknown (value set to
// UnknownVariableValue), the first non-container (map, slice, etc.) element
// will be removed from the config. The keys of unknown variables
// can be found using the UnknownKeys function.
//
// By pruning out unknown keys from the configuration, the raw
// structure will always successfully decode into its ultimate
// structure using something like mapstructure.
func (r *RawConfig) Config() map[string]interface{} {
	return r.config
}

// Interpolate uses the given mapping of variable values and uses
// those as the values to replace any variables in this raw
// configuration.
//
// Any prior calls to Interpolate are replaced with this one.
//
// If a variable key is missing, this will panic.
func (r *RawConfig) Interpolate(vs map[string]ast.Variable) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	config := langEvalConfig(vs)
	return r.interpolate(func(root ast.Node) (string, error) {
		// We detect the variables again and check if the value of any
		// of the variables is the computed value. If it is, then we
		// treat this entire value as computed.
		//
		// We have to do this here before the `lang.Eval` because
		// if any of the variables it depends on are computed, then
		// the interpolation can fail at runtime for other reasons. Example:
		// `${count.index+1}`: in a world where `count.index` is computed,
		// this would fail a type check since the computed placeholder is
		// a string, but realistically the whole value is just computed.
		vars, err := DetectVariables(root)
		if err != nil {
			return "", err
		}
		for _, v := range vars {
			varVal, ok := vs[v.FullKey()]
			if ok && varVal.Value == UnknownVariableValue {
				return UnknownVariableValue, nil
			}
		}

		// None of the variables we need are computed, meaning we should
		// be able to properly evaluate.
		result, err := hil.Eval(root, config)
		if err != nil {
			return "", err
		}

		return result.Value.(string), nil
	})
}

// Merge merges another RawConfig into this one (overriding any conflicting
// values in this config) and returns a new config. The original config
// is not modified.
func (r *RawConfig) Merge(other *RawConfig) *RawConfig {
	r.lock.Lock()
	defer r.lock.Unlock()

	// Merge the raw configurations
	raw := make(map[string]interface{})
	for k, v := range r.Raw {
		raw[k] = v
	}
	for k, v := range other.Raw {
		raw[k] = v
	}

	// Create the result
	result, err := NewRawConfig(raw)
	if err != nil {
		panic(err)
	}

	// Merge the interpolated results
	result.config = make(map[string]interface{})
	for k, v := range r.config {
		result.config[k] = v
	}
	for k, v := range other.config {
		result.config[k] = v
	}

	// Build the unknown keys
	unknownKeys := make(map[string]struct{})
	for _, k := range r.unknownKeys {
		unknownKeys[k] = struct{}{}
	}
	for _, k := range other.unknownKeys {
		unknownKeys[k] = struct{}{}
	}

	result.unknownKeys = make([]string, 0, len(unknownKeys))
	for k, _ := range unknownKeys {
		result.unknownKeys = append(result.unknownKeys, k)
	}

	return result
}

func (r *RawConfig) init() error {
	r.config = r.Raw
	r.Interpolations = nil
	r.Variables = nil

	fn := func(node ast.Node) (string, error) {
		r.Interpolations = append(r.Interpolations, node)
		vars, err := DetectVariables(node)
		if err != nil {
			return "", err
		}

		for _, v := range vars {
			if r.Variables == nil {
				r.Variables = make(map[string]InterpolatedVariable)
			}

			r.Variables[v.FullKey()] = v
		}

		return "", nil
	}

	walker := &interpolationWalker{F: fn}
	if err := reflectwalk.Walk(r.Raw, walker); err != nil {
		return err
	}

	return nil
}

func (r *RawConfig) interpolate(fn interpolationWalkerFunc) error {
	config, err := copystructure.Copy(r.Raw)
	if err != nil {
		return err
	}
	r.config = config.(map[string]interface{})

	w := &interpolationWalker{F: fn, Replace: true}
	err = reflectwalk.Walk(r.config, w)
	if err != nil {
		return err
	}

	r.unknownKeys = w.unknownKeys
	return nil
}

func (r *RawConfig) merge(r2 *RawConfig) *RawConfig {
	rawRaw, err := copystructure.Copy(r.Raw)
	if err != nil {
		panic(err)
	}

	raw := rawRaw.(map[string]interface{})
	for k, v := range r2.Raw {
		raw[k] = v
	}

	result, err := NewRawConfig(raw)
	if err != nil {
		panic(err)
	}

	return result
}

// UnknownKeys returns the keys of the configuration that are unknown
// because they had interpolated variables that must be computed.
func (r *RawConfig) UnknownKeys() []string {
	return r.unknownKeys
}

// See GobEncode
func (r *RawConfig) GobDecode(b []byte) error {
	var data gobRawConfig
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&data)
	if err != nil {
		return err
	}

	r.Key = data.Key
	r.Raw = data.Raw

	return r.init()
}

// GobEncode is a custom Gob encoder to use so that we only include the
// raw configuration. Interpolated variables and such are lost and the
// tree of interpolated variables is recomputed on decode, since it is
// referentially transparent.
func (r *RawConfig) GobEncode() ([]byte, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	data := gobRawConfig{
		Key: r.Key,
		Raw: r.Raw,
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type gobRawConfig struct {
	Key string
	Raw map[string]interface{}
}

// langEvalConfig returns the evaluation configuration we use to execute.
func langEvalConfig(vs map[string]ast.Variable) *hil.EvalConfig {
	funcMap := make(map[string]ast.Function)
	for k, v := range Funcs() {
		funcMap[k] = v
	}
	funcMap["lookup"] = interpolationFuncLookup(vs)
	funcMap["keys"] = interpolationFuncKeys(vs)
	funcMap["values"] = interpolationFuncValues(vs)

	return &hil.EvalConfig{
		GlobalScope: &ast.BasicScope{
			VarMap:  vs,
			FuncMap: funcMap,
		},
	}
}
