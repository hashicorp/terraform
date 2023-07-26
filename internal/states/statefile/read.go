// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package statefile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	version "github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform/internal/tfdiags"
	tfversion "github.com/hashicorp/terraform/version"
)

// ErrNoState is returned by ReadState when the state file is empty.
var ErrNoState = errors.New("no state")

// ErrUnusableState is an error wrapper to indicate that we *think* the input
// represents state data, but can't use it for some reason (as explained in the
// error text). Callers can check against this type with errors.As() if they
// need to distinguish between corrupt state and more fundamental problems like
// an empty file.
type ErrUnusableState struct {
	inner error
}

func errUnusable(err error) *ErrUnusableState {
	return &ErrUnusableState{inner: err}
}
func (e *ErrUnusableState) Error() string {
	return e.inner.Error()
}
func (e *ErrUnusableState) Unwrap() error {
	return e.inner
}

// Read reads a state from the given reader.
//
// Legacy state format versions 1 through 3 are supported, but the result will
// contain object attributes in the deprecated "flatmap" format and so must
// be upgraded by the caller before use.
//
// If the state file is empty, the special error value ErrNoState is returned.
// Otherwise, the returned error might be a wrapper around tfdiags.Diagnostics
// potentially describing multiple errors.
func Read(r io.Reader) (*File, error) {
	// Some callers provide us a "typed nil" *os.File here, which would
	// cause us to panic below if we tried to use it.
	if f, ok := r.(*os.File); ok && f == nil {
		return nil, ErrNoState
	}

	var diags tfdiags.Diagnostics

	// We actually just buffer the whole thing in memory, because states are
	// generally not huge and we need to do be able to sniff for a version
	// number before full parsing.
	src, err := ioutil.ReadAll(r)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to read state file",
			fmt.Sprintf("The state file could not be read: %s", err),
		))
		return nil, diags.Err()
	}

	if len(src) == 0 {
		return nil, ErrNoState
	}

	state, err := readState(src)
	if err != nil {
		return nil, err
	}

	if state == nil {
		// Should never happen
		panic("readState returned nil state with no errors")
	}

	return state, diags.Err()
}

func readState(src []byte) (*File, error) {
	var diags tfdiags.Diagnostics

	if looksLikeVersion0(src) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			unsupportedFormat,
			"The state is stored in a legacy binary format that is not supported since Terraform v0.7. To continue, first upgrade the state using Terraform 0.6.16 or earlier.",
		))
		return nil, errUnusable(diags.Err())
	}

	version, versionDiags := sniffJSONStateVersion(src)
	diags = diags.Append(versionDiags)
	if versionDiags.HasErrors() {
		// This is the last point where there's a really good chance it's not a
		// state file at all. Past here, we'll assume errors mean it's state but
		// we can't use it.
		return nil, diags.Err()
	}

	var result *File
	var err error
	switch version {
	case 0:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			unsupportedFormat,
			"The state file uses JSON syntax but has a version number of zero. There was never a JSON-based state format zero, so this state file is invalid and cannot be processed.",
		))
	case 1:
		result, diags = readStateV1(src)
	case 2:
		result, diags = readStateV2(src)
	case 3:
		result, diags = readStateV3(src)
	case 4:
		result, diags = readStateV4(src)
	default:
		thisVersion := tfversion.SemVer.String()
		creatingVersion := sniffJSONStateTerraformVersion(src)
		switch {
		case creatingVersion != "":
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				unsupportedFormat,
				fmt.Sprintf("The state file uses format version %d, which is not supported by Terraform %s. This state file was created by Terraform %s.", version, thisVersion, creatingVersion),
			))
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				unsupportedFormat,
				fmt.Sprintf("The state file uses format version %d, which is not supported by Terraform %s. This state file may have been created by a newer version of Terraform.", version, thisVersion),
			))
		}
	}

	if diags.HasErrors() {
		err = errUnusable(diags.Err())
	}

	return result, err
}

func sniffJSONStateVersion(src []byte) (uint64, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	type VersionSniff struct {
		Version *uint64 `json:"version"`
	}
	var sniff VersionSniff
	err := json.Unmarshal(src, &sniff)
	if err != nil {
		switch tErr := err.(type) {
		case *json.SyntaxError:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				unsupportedFormat,
				fmt.Sprintf("The state file could not be parsed as JSON: syntax error at byte offset %d.", tErr.Offset),
			))
		case *json.UnmarshalTypeError:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				unsupportedFormat,
				fmt.Sprintf("The version in the state file is %s. A positive whole number is required.", tErr.Value),
			))
		default:
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				unsupportedFormat,
				"The state file could not be parsed as JSON.",
			))
		}
	}

	if sniff.Version == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			unsupportedFormat,
			"The state file does not have a \"version\" attribute, which is required to identify the format version.",
		))
		return 0, diags
	}

	return *sniff.Version, diags
}

// sniffJSONStateTerraformVersion attempts to sniff the Terraform version
// specification from the given state file source code. The result is either
// a version string or an empty string if no version number could be extracted.
//
// This is a best-effort function intended to produce nicer error messages. It
// should not be used for any real processing.
func sniffJSONStateTerraformVersion(src []byte) string {
	type VersionSniff struct {
		Version string `json:"terraform_version"`
	}
	var sniff VersionSniff

	err := json.Unmarshal(src, &sniff)
	if err != nil {
		return ""
	}

	// Attempt to parse the string as a version so we won't report garbage
	// as a version number.
	_, err = version.NewVersion(sniff.Version)
	if err != nil {
		return ""
	}

	return sniff.Version
}

// unsupportedFormat is a diagnostic summary message for when the state file
// seems to not be a state file at all, or is not a supported version.
//
// Use invalidFormat instead for the subtly-different case of "this looks like
// it's intended to be a state file but it's not structured correctly".
const unsupportedFormat = "Unsupported state file format"

const upgradeFailed = "State format upgrade failed"
