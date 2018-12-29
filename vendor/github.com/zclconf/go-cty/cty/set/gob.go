package set

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// GobEncode is an implementation of the interface gob.GobEncoder, allowing
// sets to be included in structures encoded via gob.
//
// The set rules are included in the serialized value, so the caller must
// register its concrete rules type with gob.Register before using a
// set in a gob, and possibly also implement GobEncode/GobDecode to customize
// how any parameters are persisted.
//
// The set elements are also included, so if they are of non-primitive types
// they too must be registered with gob.
//
// If the produced gob values will persist for a long time, the caller must
// ensure compatibility of the rules implementation. In particular, if the
// definition of element equivalence changes between encoding and decoding
// then two distinct stored elements may be considered equivalent on decoding,
// causing the recovered set to have fewer elements than when it was stored.
func (s Set) GobEncode() ([]byte, error) {
	gs := gobSet{
		Version: 0,
		Rules:   s.rules,
		Values:  s.Values(),
	}

	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(gs)
	if err != nil {
		return nil, fmt.Errorf("error encoding set.Set: %s", err)
	}

	return buf.Bytes(), nil
}

// GobDecode is the opposite of GobEncode. See GobEncode for information
// on the requirements for and caveats of including set values in gobs.
func (s *Set) GobDecode(buf []byte) error {
	r := bytes.NewReader(buf)
	dec := gob.NewDecoder(r)

	var gs gobSet
	err := dec.Decode(&gs)
	if err != nil {
		return fmt.Errorf("error decoding set.Set: %s", err)
	}
	if gs.Version != 0 {
		return fmt.Errorf("unsupported set.Set encoding version %d; need 0", gs.Version)
	}

	victim := NewSetFromSlice(gs.Rules, gs.Values)
	s.vals = victim.vals
	s.rules = victim.rules
	return nil
}

type gobSet struct {
	Version int
	Rules   Rules

	// The bucket-based representation is for efficient in-memory access, but
	// for serialization it's enough to just retain the values themselves,
	// which we can re-bucket using the rules (which may have changed!) when
	// we re-inflate.
	Values []interface{}
}

func init() {
	gob.Register([]interface{}(nil))
}
