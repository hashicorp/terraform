package localenv

import (
	"github.com/zclconf/go-cty/cty"
)

// Storage represents a "storage" block in an environment definition file.
//
// Storage values are immutable once created, to allow the containing
// DefinitionFile type to handle modifications and thus make sure they will
// be reflected correctly in the on-disk representation.
type Storage struct {
	typeAddr string
	config   map[string]cty.Value
}

func NewStorage(typeAddr string, config map[string]cty.Value) *Storage {
	ret := &Storage{
		typeAddr: typeAddr,
		config:   make(map[string]cty.Value, len(config)),
	}

	// We'll copy the given config to make sure the caller can't modify
	// our internal state after we return.
	for k, v := range config {
		ret.config[k] = v
	}

	return ret
}

func (s *Storage) TypeAddr() string {
	return s.typeAddr
}

func (s *Storage) ConfigVal() cty.Value {
	// We wrap the config up in a cty object to communicate that it's intended
	// as an opaque bag of settings intended to be passed verbatim to a
	// storage implementation in a provider plugin, and not modified directly.
	return cty.ObjectVal(s.config)
}

// WithNewConfig creates a new Storage object that has the same storage type
// address but a new configuration map.
func (s *Storage) WithNewConfig(newConfig map[string]cty.Value) *Storage {
	return NewStorage(s.typeAddr, newConfig)
}
