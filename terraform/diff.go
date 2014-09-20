package terraform

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// The format byte is prefixed into the diff file format so that we have
// the ability in the future to change the file format if we want for any
// reason.
const diffFormatByte byte = 1

// Diff tracks the differences between resources to apply.
type Diff struct {
	Resources map[string]*InstanceDiff
	once      sync.Once
}

// ReadDiff reads a diff structure out of a reader in the format that
// was written by WriteDiff.
func ReadDiff(src io.Reader) (*Diff, error) {
	var result *Diff

	var formatByte [1]byte
	n, err := src.Read(formatByte[:])
	if err != nil {
		return nil, err
	}
	if n != len(formatByte) {
		return nil, errors.New("failed to read diff version byte")
	}

	if formatByte[0] != diffFormatByte {
		return nil, fmt.Errorf("unknown diff file version: %d", formatByte[0])
	}

	dec := gob.NewDecoder(src)
	if err := dec.Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// WriteDiff writes a diff somewhere in a binary format.
func WriteDiff(d *Diff, dst io.Writer) error {
	n, err := dst.Write([]byte{diffFormatByte})
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("failed to write diff version byte")
	}

	return gob.NewEncoder(dst).Encode(d)
}

func (d *Diff) init() {
	d.once.Do(func() {
		if d.Resources == nil {
			d.Resources = make(map[string]*InstanceDiff)
		}
	})
}

// Empty returns true if the diff has no changes.
func (d *Diff) Empty() bool {
	if len(d.Resources) == 0 {
		return true
	}

	for _, rd := range d.Resources {
		if !rd.Empty() {
			return false
		}
	}

	return true
}

// String outputs the diff in a long but command-line friendly output
// format that users can read to quickly inspect a diff.
func (d *Diff) String() string {
	var buf bytes.Buffer

	names := make([]string, 0, len(d.Resources))
	for name, _ := range d.Resources {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		rdiff := d.Resources[name]

		crud := "UPDATE"
		if rdiff.RequiresNew() && (rdiff.Destroy || rdiff.DestroyTainted) {
			crud = "DESTROY/CREATE"
		} else if rdiff.Destroy {
			crud = "DESTROY"
		} else if rdiff.RequiresNew() {
			crud = "CREATE"
		}

		buf.WriteString(fmt.Sprintf(
			"%s: %s\n",
			crud,
			name))

		keyLen := 0
		keys := make([]string, 0, len(rdiff.Attributes))
		for key, _ := range rdiff.Attributes {
			if key == "id" {
				continue
			}

			keys = append(keys, key)
			if len(key) > keyLen {
				keyLen = len(key)
			}
		}
		sort.Strings(keys)

		for _, attrK := range keys {
			attrDiff := rdiff.Attributes[attrK]

			v := attrDiff.New
			if attrDiff.NewComputed {
				v = "<computed>"
			}

			newResource := ""
			if attrDiff.RequiresNew {
				newResource = " (forces new resource)"
			}

			buf.WriteString(fmt.Sprintf(
				"  %s:%s %#v => %#v%s\n",
				attrK,
				strings.Repeat(" ", keyLen-len(attrK)),
				attrDiff.Old,
				v,
				newResource))
		}
	}

	return buf.String()
}

// InstanceDiff is the diff of a resource from some state to another.
type InstanceDiff struct {
	Attributes     map[string]*ResourceAttrDiff
	Destroy        bool
	DestroyTainted bool

	once sync.Once
}

// ResourceAttrDiff is the diff of a single attribute of a resource.
type ResourceAttrDiff struct {
	Old         string      // Old Value
	New         string      // New Value
	NewComputed bool        // True if new value is computed (unknown currently)
	NewRemoved  bool        // True if this attribute is being removed
	NewExtra    interface{} // Extra information for the provider
	RequiresNew bool        // True if change requires new resource
	Type        DiffAttrType
}

func (d *ResourceAttrDiff) GoString() string {
	return fmt.Sprintf("*%#v", *d)
}

// DiffAttrType is an enum type that says whether a resource attribute
// diff is an input attribute (comes from the configuration) or an
// output attribute (comes as a result of applying the configuration). An
// example input would be "ami" for AWS and an example output would be
// "private_ip".
type DiffAttrType byte

const (
	DiffAttrUnknown DiffAttrType = iota
	DiffAttrInput
	DiffAttrOutput
)

func (d *InstanceDiff) init() {
	d.once.Do(func() {
		if d.Attributes == nil {
			d.Attributes = make(map[string]*ResourceAttrDiff)
		}
	})
}

// Empty returns true if this diff encapsulates no changes.
func (d *InstanceDiff) Empty() bool {
	if d == nil {
		return true
	}

	return !d.Destroy && len(d.Attributes) == 0
}

// RequiresNew returns true if the diff requires the creation of a new
// resource (implying the destruction of the old).
func (d *InstanceDiff) RequiresNew() bool {
	if d == nil {
		return false
	}

	for _, rd := range d.Attributes {
		if rd != nil && rd.RequiresNew {
			return true
		}
	}

	return false
}

// Same checks whether or not to InstanceDiff are the "same." When
// we say "same", it is not necessarily exactly equal. Instead, it is
// just checking that the same attributes are changing, a destroy
// isn't suddenly happening, etc.
func (d *InstanceDiff) Same(d2 *InstanceDiff) bool {
	if d == nil && d2 == nil {
		return true
	} else if d == nil || d2 == nil {
		return false
	}

	if d.Destroy != d2.Destroy {
		return false
	}
	if d.RequiresNew() != d2.RequiresNew() {
		return false
	}
	if len(d.Attributes) != len(d2.Attributes) {
		return false
	}

	ks := make(map[string]struct{})
	for k, _ := range d.Attributes {
		ks[k] = struct{}{}
	}
	for k, _ := range d2.Attributes {
		delete(ks, k)
	}

	if len(ks) > 0 {
		return false
	}

	return true
}
