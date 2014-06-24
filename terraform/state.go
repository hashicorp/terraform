package terraform

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/hashicorp/terraform/config"
)

// State keeps track of a snapshot state-of-the-world that Terraform
// can use to keep track of what real world resources it is actually
// managing.
type State struct {
	Resources map[string]*ResourceState

	once sync.Once
}

func (s *State) init() {
	s.once.Do(func() {
		s.Resources = make(map[string]*ResourceState)
	})
}

// Orphans returns a list of keys of resources that are in the State
// but aren't present in the configuration itself. Hence, these keys
// represent the state of resources that are orphans.
func (s *State) Orphans(c *config.Config) []string {
	keys := make(map[string]struct{})
	for k, _ := range s.Resources {
		keys[k] = struct{}{}
	}

	for _, r := range c.Resources {
		delete(keys, r.Id())
	}

	result := make([]string, 0, len(keys))
	for k, _ := range keys {
		result = append(result, k)
	}

	return result
}

func (s *State) String() string {
	var buf bytes.Buffer

	names := make([]string, 0, len(s.Resources))
	for name, _ := range s.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, k := range names {
		rs := s.Resources[k]

		buf.WriteString(fmt.Sprintf("%s:\n", k))
		buf.WriteString(fmt.Sprintf("  ID = %s\n", rs.ID))

		for ak, av := range rs.Attributes {
			buf.WriteString(fmt.Sprintf("  %s = %s\n", ak, av))
		}
	}

	return buf.String()
}

// The format byte is prefixed into the state file format so that we have
// the ability in the future to change the file format if we want for any
// reason.
const stateFormatByte byte = 1

// ReadState reads a state structure out of a reader in the format that
// was written by WriteState.
func ReadState(src io.Reader) (*State, error) {
	var result *State

	var formatByte [1]byte
	n, err := src.Read(formatByte[:])
	if err != nil {
		return nil, err
	}
	if n != len(formatByte) {
		return nil, errors.New("failed to read state version byte")
	}

	if formatByte[0] != stateFormatByte {
		return nil, fmt.Errorf("unknown state file version: %d", formatByte[0])
	}

	dec := gob.NewDecoder(src)
	if err := dec.Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// WriteState writes a state somewhere in a binary format.
func WriteState(d *State, dst io.Writer) error {
	n, err := dst.Write([]byte{stateFormatByte})
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("failed to write state version byte")
	}

	return gob.NewEncoder(dst).Encode(d)
}

// ResourceState holds the state of a resource that is used so that
// a provider can find and manage an existing resource as well as for
// storing attributes that are uesd to populate variables of child
// resources.
//
// Attributes has attributes about the created resource that are
// queryable in interpolation: "${type.id.attr}"
//
// Extra is just extra data that a provider can return that we store
// for later, but is not exposed in any way to the user.
type ResourceState struct {
	ID         string
	Type       string
	Attributes map[string]string
	Extra      map[string]interface{}
}

// MergeDiff takes a ResourceDiff and merges the attributes into
// this resource state in order to generate a new state. This new
// state can be used to provide updated attribute lookups for
// variable interpolation.
//
// If the diff attribute requires computing the value, and hence
// won't be available until apply, the value is replaced with the
// computeID.
func (s *ResourceState) MergeDiff(d *ResourceDiff) *ResourceState {
	var result ResourceState
	if s != nil {
		result = *s
	}

	result.Attributes = make(map[string]string)
	if s != nil {
		for k, v := range s.Attributes {
			result.Attributes[k] = v
		}
	}
	if d != nil {
		for k, diff := range d.Attributes {
			if diff.NewComputed {
				result.Attributes[k] = config.UnknownVariableValue
				continue
			}

			result.Attributes[k] = diff.New
		}
	}

	return &result
}
