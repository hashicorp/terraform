// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	viewsjson "github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type Version interface {
	LogVersion(version string, platform string, providerSelections map[addrs.Provider]*depsfile.ProviderLock, outdated bool, latest string, diags tfdiags.Diagnostics)
	Diagnostics(diags tfdiags.Diagnostics)
}

// VersionOutput contains the information that is output by the version command.
// Either it's rendered as JSON or as a human-readable string, depending on the view type.
type VersionOutput struct {
	Version            string            `json:"terraform_version"`
	Platform           string            `json:"platform"`
	ProviderSelections map[string]string `json:"provider_selections"`
	Outdated           bool              `json:"terraform_outdated"`

	Diagnostics []*viewsjson.Diagnostic `json:"diagnostics"`
}

func NewVersion(vt arguments.ViewType, view *View) Version {
	switch vt {
	case arguments.ViewJSON:
		return &VersionJSON{
			view: view,
		}
	case arguments.ViewHuman:
		return &VersionHuman{
			view: view,
		}
	default:
		panic(fmt.Sprintf("unknown view type %v", vt))
	}
}

type VersionHuman struct {
	view *View
}

func (v *VersionHuman) Diagnostics(diags tfdiags.Diagnostics) {
	v.view.Diagnostics(diags)
}

func (v *VersionHuman) LogVersion(version string, platform string, providerSelections map[addrs.Provider]*depsfile.ProviderLock, outdated bool, latest string, diags tfdiags.Diagnostics) {
	// Log any diagnostics first, so they appear above the version output.
	// Calling code should have handled errors, so we expect only warnings here.
	v.Diagnostics(diags)

	var outputString bytes.Buffer

	// Terraform version and platform
	fmt.Fprintf(&outputString, "Terraform v%s\n", version) // May include prerelease if relevant
	fmt.Fprintf(&outputString, "on %s\n", platform)

	// For each provider selection, print the provider and version
	// The list is sorted to make output deterministic.
	var providerVersions []string
	for provider, lock := range providerSelections {
		version := lock.Version().String()
		if version == "0.0.0" {
			providerVersions = append(providerVersions, fmt.Sprintf("+ provider %s (unversioned)", provider))
		} else {
			providerVersions = append(providerVersions, fmt.Sprintf("+ provider %s v%s", provider, version))
		}
	}
	slices.Sort(providerVersions)
	for _, str := range providerVersions {
		fmt.Fprintln(&outputString, str)
	}

	if outdated {
		fmt.Fprintf(&outputString, "\nYour version of Terraform is out of date! The latest version\nis %s. You can update by downloading from https://developer.hashicorp.com/terraform/install\n", latest)
	}

	v.view.streams.Println(outputString.String())
}

type VersionJSON struct {
	view *View
}

// Diagnostics is basically the same as LogVersion but only the diagnostics data is populated.
func (v *VersionJSON) Diagnostics(diags tfdiags.Diagnostics) {
	if len(diags) == 0 {
		return
	}

	output := VersionOutput{
		// Make sure this appears as an empty object.
		ProviderSelections: make(map[string]string),

		// Make sure this always appears as an array in our output, since
		// this is easier to consume for dynamically-typed languages.
		Diagnostics: []*viewsjson.Diagnostic{},
	}

	configSources := v.view.configSources()
	for _, diag := range diags {
		output.Diagnostics = append(output.Diagnostics, viewsjson.NewDiagnostic(diag, configSources))
	}

	v.view.streams.Println(v.marshal(&output))
}

func (v *VersionJSON) LogVersion(version string, platform string, providerSelections map[addrs.Provider]*depsfile.ProviderLock, outdated bool, latest string, diags tfdiags.Diagnostics) {
	output := VersionOutput{
		Version:            version,
		Platform:           platform,
		ProviderSelections: make(map[string]string),
		Outdated:           outdated,

		// Make sure this always appears as an array in our output, since
		// this is easier to consume for dynamically-typed languages.
		Diagnostics: []*viewsjson.Diagnostic{},
	}

	// Add providers, if present
	for provider, lock := range providerSelections {
		output.ProviderSelections[provider.String()] = lock.Version().String()
	}

	// Add diagnostics, if present
	configSources := v.view.configSources()
	for _, diag := range diags {
		output.Diagnostics = append(output.Diagnostics, viewsjson.NewDiagnostic(diag, configSources))
	}

	v.view.streams.Println(v.marshal(&output))
}

func (v *VersionJSON) marshal(output *VersionOutput) string {
	j, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		// Should never happen because we fully-control the input here
		panic(err)
	}
	return string(j)
}
