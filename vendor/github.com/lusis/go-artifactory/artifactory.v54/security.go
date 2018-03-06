package artifactory

// GetSystemSecurityConfiguration returns the security configuration for the artifactory server
func (c *Client) GetSystemSecurityConfiguration() (s string, e error) {
	d, e := c.Get("/api/system/security", make(map[string]string))
	return string(d), e
}
