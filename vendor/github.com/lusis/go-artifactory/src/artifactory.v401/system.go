package artifactory

func (c *ArtifactoryClient) GetGeneralConfiguration() (s string, e error) {
	d, e := c.Get("/api/system/configuration", make(map[string]string))
	return string(d), e
}
