package planfile

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configload"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const tfstateFilename = "tfstate"
const tfstatePreviousFilename = "tfstate-prev"
const dependencyLocksFilename = ".terraform.lock.hcl" // matches the conventional name in an input configuration

// Reader is the main type used to read plan files. Create a Reader by calling
// Open.
//
// A plan file is a random-access file format, so methods of Reader must
// be used to access the individual portions of the file for further
// processing.
type Reader struct {
	zip *zip.ReadCloser
}

// Open creates a Reader for the file at the given filename, or returns an
// error if the file doesn't seem to be a planfile.
func Open(filename string) (*Reader, error) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		// To give a better error message, we'll sniff to see if this looks
		// like our old plan format from versions prior to 0.12.
		if b, sErr := ioutil.ReadFile(filename); sErr == nil {
			if bytes.HasPrefix(b, []byte("tfplan")) {
				return nil, fmt.Errorf("the given plan file was created by an earlier version of Terraform; plan files cannot be shared between different Terraform versions")
			}
		}
		return nil, err
	}

	// Sniff to make sure this looks like a plan file, as opposed to any other
	// random zip file the user might have around.
	var planFile *zip.File
	for _, file := range r.File {
		if file.Name == tfplanFilename {
			planFile = file
			break
		}
	}
	if planFile == nil {
		return nil, fmt.Errorf("the given file is not a valid plan file")
	}

	// For now, we'll just accept the presence of the tfplan file as enough,
	// and wait to validate the version when the caller requests the plan
	// itself.

	return &Reader{
		zip: r,
	}, nil
}

// ReadPlan reads the plan embedded in the plan file.
//
// Errors can be returned for various reasons, including if the plan file
// is not of an appropriate format version, if it was created by a different
// version of Terraform, if it is invalid, etc.
func (r *Reader) ReadPlan() (*plans.Plan, error) {
	var planFile *zip.File
	for _, file := range r.zip.File {
		if file.Name == tfplanFilename {
			planFile = file
			break
		}
	}
	if planFile == nil {
		// This should never happen because we checked for this file during
		// Open, but we'll check anyway to be safe.
		return nil, fmt.Errorf("the plan file is invalid")
	}

	pr, err := planFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve plan from plan file: %s", err)
	}
	defer pr.Close()

	// There's a slight mismatch in how plans.Plan is modeled vs. how
	// the underlying plan file format works, because the "tfplan" embedded
	// file contains only some top-level metadata and the planned changes,
	// and not the previous run or prior states. Therefore we need to
	// build this up in multiple steps.
	// This is some technical debt because historically we considered the
	// planned changes and prior state as totally separate, but later realized
	// that it made sense for a plans.Plan to include the prior state directly
	// so we can see what state the plan applies to. Hopefully later we'll
	// clean this up some more so that we don't have two different ways to
	// access the prior state (this and the ReadStateFile method).
	ret, err := readTfplan(pr)
	if err != nil {
		return nil, err
	}

	prevRunStateFile, err := r.ReadPrevStateFile()
	if err != nil {
		return nil, fmt.Errorf("failed to read previous run state from plan file: %s", err)
	}
	priorStateFile, err := r.ReadStateFile()
	if err != nil {
		return nil, fmt.Errorf("failed to read prior state from plan file: %s", err)
	}

	ret.PrevRunState = prevRunStateFile.State
	ret.PriorState = priorStateFile.State

	return ret, nil
}

// ReadStateFile reads the state file embedded in the plan file, which
// represents the "PriorState" as defined in plans.Plan.
//
// If the plan file contains no embedded state file, the returned error is
// statefile.ErrNoState.
func (r *Reader) ReadStateFile() (*statefile.File, error) {
	for _, file := range r.zip.File {
		if file.Name == tfstateFilename {
			r, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to extract state from plan file: %s", err)
			}
			return statefile.Read(r)
		}
	}
	return nil, statefile.ErrNoState
}

// ReadPrevStateFile reads the previous state file embedded in the plan file, which
// represents the "PrevRunState" as defined in plans.Plan.
//
// If the plan file contains no embedded previous state file, the returned error is
// statefile.ErrNoState.
func (r *Reader) ReadPrevStateFile() (*statefile.File, error) {
	for _, file := range r.zip.File {
		if file.Name == tfstatePreviousFilename {
			r, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to extract previous state from plan file: %s", err)
			}
			return statefile.Read(r)
		}
	}
	return nil, statefile.ErrNoState
}

// ReadConfigSnapshot reads the configuration snapshot embedded in the plan
// file.
//
// This is a lower-level alternative to ReadConfig that just extracts the
// source files, without attempting to parse them.
func (r *Reader) ReadConfigSnapshot() (*configload.Snapshot, error) {
	return readConfigSnapshot(&r.zip.Reader)
}

// ReadConfig reads the configuration embedded in the plan file.
//
// Internally this function delegates to the configs/configload package to
// parse the embedded configuration and so it returns diagnostics (rather than
// a native Go error as with other methods on Reader).
func (r *Reader) ReadConfig() (*configs.Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	snap, err := r.ReadConfigSnapshot()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to read configuration from plan file",
			fmt.Sprintf("The configuration file snapshot in the plan file could not be read: %s.", err),
		))
		return nil, diags
	}

	loader := configload.NewLoaderFromSnapshot(snap)
	rootDir := snap.Modules[""].Dir // Root module base directory
	config, configDiags := loader.LoadConfig(rootDir)
	diags = diags.Append(configDiags)

	return config, diags
}

// ReadDependencyLocks reads the dependency lock information embedded in
// the plan file.
//
// Some test codepaths create plan files without dependency lock information,
// but all main codepaths should populate this. If reading a file without
// the dependency information, this will return error diagnostics.
func (r *Reader) ReadDependencyLocks() (*depsfile.Locks, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	for _, file := range r.zip.File {
		if file.Name == dependencyLocksFilename {
			r, err := file.Open()
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to read dependency locks from plan file",
					fmt.Sprintf("Couldn't read the dependency lock information embedded in the plan file: %s.", err),
				))
				return nil, diags
			}
			src, err := ioutil.ReadAll(r)
			if err != nil {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to read dependency locks from plan file",
					fmt.Sprintf("Couldn't read the dependency lock information embedded in the plan file: %s.", err),
				))
				return nil, diags
			}
			locks, moreDiags := depsfile.LoadLocksFromBytes(src, "<saved-plan>")
			diags = diags.Append(moreDiags)
			return locks, diags
		}
	}

	// If we fall out here then this is a file without dependency information.
	diags = diags.Append(tfdiags.Sourceless(
		tfdiags.Error,
		"Saved plan has no dependency lock information",
		"The specified saved plan file does not include any dependency lock information. This is a bug in the previous run of Terraform that created this file.",
	))
	return nil, diags
}

// Close closes the file, after which no other operations may be performed.
func (r *Reader) Close() error {
	return r.zip.Close()
}
