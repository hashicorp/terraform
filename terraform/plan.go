package terraform

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io"

	"github.com/hashicorp/terraform/config"
)

// Plan represents a single Terraform execution plan, which contains
// all the information necessary to make an infrastructure change.
type Plan struct {
	Config *config.Config
	Diff   *Diff
	State  *State
	Vars   map[string]string
}

// The format byte is prefixed into the plan file format so that we have
// the ability in the future to change the file format if we want for any
// reason.
const planFormatByte byte = 1

// ReadPlan reads a plan structure out of a reader in the format that
// was written by WritePlan.
func ReadPlan(src io.Reader) (*Plan, error) {
	var result *Plan

	var formatByte [1]byte
	n, err := src.Read(formatByte[:])
	if err != nil {
		return nil, err
	}
	if n != len(formatByte) {
		return nil, errors.New("failed to read plan version byte")
	}

	if formatByte[0] != planFormatByte {
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
	n, err := dst.Write([]byte{planFormatByte})
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("failed to write plan version byte")
	}

	return gob.NewEncoder(dst).Encode(d)
}
