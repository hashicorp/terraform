package terraform

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/terraform/config/module"
)

func init() {
	gob.Register(make([]interface{}, 0))
	gob.Register(make([]map[string]interface{}, 0))
	gob.Register(make(map[string]interface{}))
	gob.Register(make(map[string]string))
}

// Plan represents a single Terraform execution plan, which contains
// all the information necessary to make an infrastructure change.
type Plan struct {
	Diff   *Diff
	Module *module.Tree
	State  *State
	Vars   map[string]string

	once sync.Once
}

// Context returns a Context with the data encapsulated in this plan.
//
// The following fields in opts are overridden by the plan: Config,
// Diff, State, Variables.
func (p *Plan) Context(opts *ContextOpts) *Context {
	opts.Diff = p.Diff
	opts.Module = p.Module
	opts.State = p.State
	opts.Variables = p.Vars
	return NewContext(opts)
}

func (p *Plan) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString("DIFF:\n\n")
	buf.WriteString(p.Diff.String())
	buf.WriteString("\n\nSTATE:\n\n")
	buf.WriteString(p.State.String())
	return buf.String()
}

func (p *Plan) init() {
	p.once.Do(func() {
		if p.Diff == nil {
			p.Diff = new(Diff)
			p.Diff.init()
		}

		if p.State == nil {
			p.State = new(State)
			p.State.init()
		}

		if p.Vars == nil {
			p.Vars = make(map[string]string)
		}
	})
}

// The format byte is prefixed into the plan file format so that we have
// the ability in the future to change the file format if we want for any
// reason.
const planFormatMagic = "tfplan"
const planFormatVersion byte = 1

// ReadPlan reads a plan structure out of a reader in the format that
// was written by WritePlan.
func ReadPlan(src io.Reader) (*Plan, error) {
	var result *Plan
	var err error
	n := 0

	// Verify the magic bytes
	magic := make([]byte, len(planFormatMagic))
	for n < len(magic) {
		n, err = src.Read(magic[n:])
		if err != nil {
			return nil, fmt.Errorf("error while reading magic bytes: %s", err)
		}
	}
	if string(magic) != planFormatMagic {
		return nil, fmt.Errorf("not a valid plan file")
	}

	// Verify the version is something we can read
	var formatByte [1]byte
	n, err = src.Read(formatByte[:])
	if err != nil {
		return nil, err
	}
	if n != len(formatByte) {
		return nil, errors.New("failed to read plan version byte")
	}

	if formatByte[0] != planFormatVersion {
		return nil, fmt.Errorf("unknown plan file version: %d", formatByte[0])
	}

	dec := gob.NewDecoder(src)
	if err := dec.Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// WritePlan writes a plan somewhere in a binary format.
func WritePlan(d *Plan, dst io.Writer) error {
	// Write the magic bytes so we can determine the file format later
	n, err := dst.Write([]byte(planFormatMagic))
	if err != nil {
		return err
	}
	if n != len(planFormatMagic) {
		return errors.New("failed to write plan format magic bytes")
	}

	// Write a version byte so we can iterate on version at some point
	n, err = dst.Write([]byte{planFormatVersion})
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("failed to write plan version byte")
	}

	return gob.NewEncoder(dst).Encode(d)
}
