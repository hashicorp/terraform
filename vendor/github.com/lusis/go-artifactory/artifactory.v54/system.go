package artifactory

import "encoding/json"

// VersionInfo represents the version information about Artifactory
type VersionInfo struct {
	Version  string   `json:"version"`
	Revision string   `json:"revision"`
	Addons   []string `json:"addons"`
}

// GetSystemInfo returns the general system information about Artifactory
func (c *Client) GetSystemInfo() (string, error) {
	d, e := c.Get("/api/system", make(map[string]string))
	return string(d), e
}

// GetSystemHealthPing returns a simple status response about the state of Artifactory
func (c *Client) GetSystemHealthPing() (string, error) {
	d, e := c.Get("/api/system/ping", make(map[string]string))
	return string(d), e
}

// GetGeneralConfiguration returns the general Artifactory configuration
func (c *Client) GetGeneralConfiguration() (string, error) {
	d, e := c.Get("/api/system/configuration", make(map[string]string))
	return string(d), e
}

// GetVersionAndAddOnInfo returns information about the current Artifactory version, revision, and currently installed Add-ons
func (c *Client) GetVersionAndAddOnInfo() (VersionInfo, error) {
	var res VersionInfo
	d, err := c.Get("/api/system/version", make(map[string]string))
	if err != nil {
		return res, err
	}
	err = json.Unmarshal(d, &res)

	return res, err
}
