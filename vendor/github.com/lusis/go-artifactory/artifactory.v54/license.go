package artifactory

import "encoding/json"

// LicenseInfo represents the json response from Artifactory for license information
type LicenseInfo struct {
	LicenseType  string `json:"type"`
	ValidThrough string `json:"validThrough"`
	LicensedTo   string `json:"licensedTo"`
}

// HALicense represents an element in the json response from Artifactory
type HALicense struct {
	LicenseType  string `json:"type"`
	ValidThrough string `json:"validThrough"`
	LicensedTo   string `json:"licensedTo"`
	LicenseHash  string `json:"licenseHash"`
	NodeID       string `json:"nodeId"`
	NodeURL      string `json:"nodeUrl"`
	Expired      bool   `json:"expired"`
}

// HALicenseInfo represents the json response from Artifactory for the HA license information
type HALicenseInfo struct {
	Licenses []HALicense `json:"licenses"`
}

// InstallLicense represents the json payload we send to Artifactory
type InstallLicense struct {
	LicenseKey string `json:"licenseKey"`
}

// GetLicenseInfo returns information about the currently installed license
func (c *Client) GetLicenseInfo() (LicenseInfo, error) {
	var res LicenseInfo
	d, err := c.Get("/api/system/license", make(map[string]string))
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(d, &res)

	return res, err
}

// InstallLicense installs a new license key or changes the current one
func (c *Client) InstallLicense(license InstallLicense, q map[string]string) error {
	j, err := json.Marshal(license)
	if err != nil {
		return err
	}

	_, err = c.Post("/api/system/license", j, q)
	return err
}

// GetHALicenseInfo returns information about the currently installed licenses in an HA cluster
func (c *Client) GetHALicenseInfo() (HALicenseInfo, error) {
	var res HALicenseInfo
	d, err := c.Get("/api/system/licenses", make(map[string]string))
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(d, &res)

	return res, err
}

// InstallHALicenses installs a new license key(s) on an HA cluster
func (c *Client) InstallHALicenses(licenses []InstallLicense, q map[string]string) error {
	j, err := json.Marshal(licenses)
	if err != nil {
		return err
	}

	_, err = c.Post("/api/system/licenses", j, q)
	return err
}
