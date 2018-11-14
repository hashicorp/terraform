package response

import (
	"sort"
	"strings"

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
	Shasum              string `json:"shasum"`

	SigningKeys SigningKeyList `json:"signing_keys"`
}

// SigningKeyList is the response structure for a list of signing keys.
type SigningKeyList struct {
	GPGKeys []*GPGKey `json:"gpg_public_keys"`
}

// GPGKey is the response structure for a GPG key.
type GPGKey struct {
	ASCIIArmor string  `json:"ascii_armor"`
	Source     string  `json:"source"`
	SourceURL  *string `json:"source_url"`
}

// Collection type for TerraformProviderVersion
type ProviderVersionCollection []*TerraformProviderVersion

// GPGASCIIArmor returns an ASCII-armor-formatted string for all of the gpg
// keys in the response.
func (signingKeys *SigningKeyList) GPGASCIIArmor() string {
	keys := []string{}

	for _, gpgKey := range signingKeys.GPGKeys {
		keys = append(keys, gpgKey.ASCIIArmor)
	}

	return strings.Join(keys, "\n")
}

// Sort sorts versions from newest to oldest.
func (v ProviderVersionCollection) Sort() {
	sort.Slice(v, func(i, j int) bool {
		versionA, _ := version.NewVersion(v[i].Version)
		versionB, _ := version.NewVersion(v[j].Version)

		return versionA.GreaterThan(versionB)
	})
}
