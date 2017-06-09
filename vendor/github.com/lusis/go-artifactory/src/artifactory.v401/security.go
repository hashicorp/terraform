package artifactory

func (c *ArtifactoryClient) GetSystemSecurityConfiguration() (s string, e error) {
	d, e := c.Get("/api/system/security", make(map[string]string))
	return string(d), e
}
