package config

import (
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
	Raw       map[string]interface{}
	Variables map[string]InterpolatedVariable

	config      map[string]interface{}
	unknownKeys []string
}

// NewRawConfig creates a new RawConfig structure and populates the
// publicly readable struct fields.
func NewRawConfig(raw map[string]interface{}) (*RawConfig, error) {
	walker := new(variableDetectWalker)
	if err := reflectwalk.Walk(raw, walker); err != nil {
		return nil, err
	}

	return &RawConfig{
		Raw:       raw,
		Variables: walker.Variables,
		config:    raw,
	}, nil
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

	w := &variableReplaceWalker{Values: vs}
	r.config = config.(map[string]interface{})
	err = reflectwalk.Walk(r.config, w)
	if err != nil {
		return err
	}

	r.unknownKeys = w.UnknownKeys
	return nil
}

// UnknownKeys returns the keys of the configuration that are unknown
// because they had interpolated variables that must be computed.
func (r *RawConfig) UnknownKeys() []string {
	return r.unknownKeys
}
