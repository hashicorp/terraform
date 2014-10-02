package config

import (
	"bytes"
	"encoding/gob"

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
	Interpolations []Interpolation
	Variables      map[string]InterpolatedVariable

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

// Value returns the value of the configuration if this configuration
// has a Key set. If this does not have a Key set, nil will be returned.
func (r *RawConfig) Value() interface{} {
	if c := r.Config(); c != nil {
		if v, ok := c[r.Key]; ok {
			return v
		}
	}

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
func (r *RawConfig) Interpolate(vs map[string]string) error {
	config, err := copystructure.Copy(r.Raw)
	if err != nil {
		return err
	}
	r.config = config.(map[string]interface{})

	fn := func(i Interpolation) (string, error) {
		return i.Interpolate(vs)
	}

	w := &interpolationWalker{F: fn, Replace: true}
	err = reflectwalk.Walk(r.config, w)
	if err != nil {
		return err
	}

	r.unknownKeys = w.unknownKeys
	return nil
}

func (r *RawConfig) init() error {
	r.config = r.Raw
	r.Interpolations = nil
	r.Variables = nil

	fn := func(i Interpolation) (string, error) {
		r.Interpolations = append(r.Interpolations, i)

		for k, v := range i.Variables() {
			if r.Variables == nil {
				r.Variables = make(map[string]InterpolatedVariable)
			}

			r.Variables[k] = v
		}

		return "", nil
	}

	walker := &interpolationWalker{F: fn}
	if err := reflectwalk.Walk(r.Raw, walker); err != nil {
		return err
	}

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
	err := gob.NewDecoder(bytes.NewReader(b)).Decode(&r.Raw)
	if err != nil {
		return err
	}

	return r.init()
}

// GobEncode is a custom Gob encoder to use so that we only include the
// raw configuration. Interpolated variables and such are lost and the
// tree of interpolated variables is recomputed on decode, since it is
// referentially transparent.
func (r *RawConfig) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(r.Raw); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
