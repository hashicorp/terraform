package terraform

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
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
//
// A plan has to contain basically the entire state of the world
// necessary to make a change: the state, diff, config, backend config, etc.
// This is so that it can run alone without any other data.
type Plan struct {
	// Diff describes the resource actions that must be taken when this
	// plan is applied.
	Diff *Diff

	// Module represents the entire configuration that was present when this
	// plan was created.
	Module *module.Tree

	// State is the Terraform state that was current when this plan was
	// created.
	//
	// It is not allowed to apply a plan that has a stale state, since its
	// diff could be outdated.
	State *State

	// Vars retains the variables that were set when creating the plan, so
	// that the same variables can be applied during apply.
	Vars map[string]interface{}

	// Targets, if non-empty, contains a set of resource address strings that
	// identify graph nodes that were selected as targets for plan.
	//
	// When targets are set, any graph node that is not directly targeted or
	// indirectly targeted via dependencies is excluded from the graph.
	Targets []string

	// TerraformVersion is the version of Terraform that was used to create
	// this plan.
	//
	// It is not allowed to apply a plan created with a different version of
	// Terraform, since the other fields of this structure may be interpreted
	// in different ways between versions.
	TerraformVersion string

	// ProviderSHA256s is a map giving the SHA256 hashes of the exact binaries
	// used as plugins for each provider during plan.
	//
	// These must match between plan and apply to ensure that the diff is
	// correctly interpreted, since different provider versions may have
	// different attributes or attribute value constraints.
	ProviderSHA256s map[string][]byte

	// Backend is the backend that this plan should use and store data with.
	Backend *BackendState

	// Destroy indicates that this plan was created for a full destroy operation
	Destroy bool

	once sync.Once
}

// Context returns a Context with the data encapsulated in this plan.
//
// The following fields in opts are overridden by the plan: Config,
// Diff, Variables.
//
// If State is not provided, it is set from the plan. If it _is_ provided,
// it must be Equal to the state stored in plan, but may have a newer
// serial.
func (p *Plan) Context(opts *ContextOpts) (*Context, error) {
	var err error
	opts, err = p.contextOpts(opts)
	if err != nil {
		return nil, err
	}
	return NewContext(opts)
}

// contextOpts mutates the given base ContextOpts in place to use input
// objects obtained from the receiving plan.
func (p *Plan) contextOpts(base *ContextOpts) (*ContextOpts, error) {
	opts := base

	opts.Diff = p.Diff
	opts.Module = p.Module
	opts.Targets = p.Targets
	opts.ProviderSHA256s = p.ProviderSHA256s
	opts.Destroy = p.Destroy

	if opts.State == nil {
		opts.State = p.State
	} else if !opts.State.Equal(p.State) {
		// Even if we're overriding the state, it should be logically equal
		// to what's in plan. The only valid change to have made by the time
		// we get here is to have incremented the serial.
		//
		// Due to the fact that serialization may change the representation of
		// the state, there is little chance that these aren't actually equal.
		// Log the error condition for reference, but continue with the state
		// we have.
		log.Println("[WARNING] Plan state and ContextOpts state are not equal")
	}

	thisVersion := VersionString()
	if p.TerraformVersion != "" && p.TerraformVersion != thisVersion {
		return nil, fmt.Errorf(
			"plan was created with a different version of Terraform (created with %s, but running %s)",
			p.TerraformVersion, thisVersion,
		)
	}

	opts.Variables = make(map[string]interface{})
	for k, v := range p.Vars {
		opts.Variables[k] = v
	}

	return opts, nil
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
			p.Vars = make(map[string]interface{})
		}
	})
}

// The format byte is prefixed into the plan file format so that we have
// the ability in the future to change the file format if we want for any
// reason.
const planFormatMagic = "tfplan"
const planFormatVersion byte = 2

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
