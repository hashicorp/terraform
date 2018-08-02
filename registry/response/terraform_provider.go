package response

import (
	"sort"

	version "github.com/hashicorp/go-version"
)

// TerraformProvider is the response structure for all required information for
// Terraform to choose a download URL. It must include all versions and all
// platforms for Terraform to perform version and os/arch constraint matching
// locally.
type TerraformProvider struct {
	ID       string `json:"id"`
	Verified bool   `json:"verified"`

	Versions []*TerraformProviderVersion `json:"versions"`
}

// TerraformProviderVersion is the Terraform-specific response structure for a
// provider version.
type TerraformProviderVersion struct {
	Version   string   `json:"version"`
	Protocols []string `json:"protocols"`

	Platforms []*TerraformProviderPlatform `json:"platforms"`
}

// TerraformProviderVersions is the Terraform-specific response structure for an
// array of provider versions
type TerraformProviderVersions struct {
	ID       string                      `json:"id"`
	Versions []*TerraformProviderVersion `json:"versions"`
}

// TerraformProviderPlatform is the Terraform-specific response structure for a
// provider platform.
type TerraformProviderPlatform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// TerraformProviderPlatformLocation is the Terraform-specific response
// structure for a provider platform with all details required to perform a
// download.
type TerraformProviderPlatformLocation struct {
	OS                  string `json:"os"`
	Arch                string `json:"arch"`
	Filename            string `json:"filename"`
	DownloadURL         string `json:"download_url"`
	ShasumsURL          string `json:"shasums_url"`
	ShasumsSignatureURL string `json:"shasums_signature_url"`
}

// Collection type for TerraformProviderVersion
type Collection []*TerraformProviderVersion

// Sort sorts versions from newest to oldest.
func (v Collection) Sort() {
	sort.Slice(v, func(i, j int) bool {
		versionA, _ := version.NewVersion(v[i].Version)
		versionB, _ := version.NewVersion(v[j].Version)

		return versionA.GreaterThan(versionB)
	})
}
