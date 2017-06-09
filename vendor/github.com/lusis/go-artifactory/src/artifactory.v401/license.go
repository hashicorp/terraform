package artifactory

import (
	"encoding/json"
)

type LicenseInformation struct {
	LicenseType  string `json:"type"`
	ValidThrough string `json:"validThrough"`
	LicensedTo   string `json:"licensedTo"`
}

func (c *ArtifactoryClient) GetLicenseInformation() (LicenseInformation, error) {
	o := make(map[string]string, 0)
	var l LicenseInformation
	d, e := c.Get("/api/system/license", o)
	if e != nil {
		return l, e
	} else {
		err := json.Unmarshal(d, &l)
		return l, err
	}
}
