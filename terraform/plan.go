package terraform

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs"
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

	// Config represents the entire configuration that was present when this
	// plan was created.
	Config *configs.Config

	// State is the Terraform state that was current when this plan was
	// created.
	//
	// It is not allowed to apply a plan that has a stale state, since its
	// diff could be outdated.
	State *State

	// Vars retains the variables that were set when creating the plan, so
	// that the same variables can be applied during apply.
	Vars map[string]cty.Value

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
			p.Vars = make(map[string]cty.Value)
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
	return nil, fmt.Errorf("terraform.ReadPlan is no longer in use; use planfile.Open instead")
}

// WritePlan writes a plan somewhere in a binary format.
func WritePlan(d *Plan, dst io.Writer) error {
	return fmt.Errorf("terraform.WritePlan is no longer in use; use planfile.Create instead")
}
