package terraform

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/terraform/config/module"
)

// Plan represents a single Terraform execution plan, which contains
// all the information necessary to make an infrastructure change.
type Plan struct {
	Diff    *Diff             `json:"diff"`
	Module  *module.Tree      `json:"module"`
	State   *State            `json:"state"`
	Vars    map[string]string `json:"variables"`
	Version string            `json:"version"`

	once sync.Once `json:"-"`
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

// Our old binary format used a magic prefix to identify plan files.
// We use this to recognize and reject old plan files with a helpful
// error message.
const planOldFormatMagic = "tfplan"

func planFileVersion() string {
	// Since plan files are short-lived and easy to recreate, we'll reject
	// any plan file that was created by a different version of Terraform.
	if VersionPrerelease == "" {
		return Version
	} else {
		return fmt.Sprintf("%s-%s", Version, VersionPrerelease)
	}
}

// ReadPlan reads a plan structure out of a reader in the format that
// was written by WritePlan.
func ReadPlan(src io.Reader) (*Plan, error) {
	buf := bufio.NewReader(src)

	// Check if this is the legacy binary format
	start, err := buf.Peek(len(planOldFormatMagic))
	if err != nil {
		return nil, fmt.Errorf("Failed to check for magic bytes: %v", err)
	}
	if string(start) == stateFormatMagic {
		return nil, fmt.Errorf(
			"Plan was created with an earlier Terraform version; please create a new plan",
		)
	}

	// Otherwise, assumed to be our JSON format
	dec := json.NewDecoder(buf)
	plan := &Plan{}
	if err := dec.Decode(plan); err != nil {
		return nil, fmt.Errorf("Decoding plan file failed: %v", err)
	}

	// Check the version, this to ensure we don't read a future
	// version that we don't understand
	if plan.Version > planFileVersion() {
		return nil, fmt.Errorf(
			"Plan was created with a different Terraform version; please create a new plan.",
		)
	}

	if plan.State != nil {
		if err := plan.State.prepareAfterRead(); err != nil {
			return nil, fmt.Errorf("Error in state from plan: %s", err)
		}
	}

	return plan, nil
}

// WritePlan writes a plan somewhere in a binary format.
func WritePlan(d *Plan, dst io.Writer) error {

	// Note the version so we can reject incompatible versions
	d.Version = planFileVersion()

	// Normalize and prepare the state portion of the plan, in
	// a manner compatible with the state file format.
	if d.State != nil {
		d.State.prepareForWrite()
	}

	data, err := json.MarshalIndent(d, "", "    ")
	if err != nil {
		return fmt.Errorf("Failed to encode plan: %s", err)
	}

	// We append a newline to the data because MarshalIndent doesn't
	data = append(data, '\n')

	// Write the data out to the dst
	if _, err := io.Copy(dst, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("Failed to write plan: %v", err)
	}

	return nil
}
