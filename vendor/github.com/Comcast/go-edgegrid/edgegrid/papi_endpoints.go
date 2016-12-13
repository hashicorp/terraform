package edgegrid

const papiPath = "/papi/v0/"

func papiBase(c *AuthCredentials) string {
	return concat([]string{
		c.APIHost,
		papiPath,
	})
}

func papiGroupsEndpoint(c *AuthCredentials) string {
	return concat([]string{
		papiBase(c),
		"groups/",
	})
}

func papiProductsEndpoint(c *AuthCredentials, contractID string) string {
	return concat([]string{
		papiBase(c),
		"products?contractId=",
		contractID,
	})
}

func papiCpCodesEndpoint(c *AuthCredentials, contractID, groupID string) string {
	return concat([]string{
		papiBase(c),
		"cpcodes/",
		papiQuery(contractID, groupID),
	})
}

func papiCpCodeEndpoint(c *AuthCredentials, cpCodeID, contractID, groupID string) string {
	return concat([]string{
		papiBase(c),
		"cpcodes/",
		cpCodeID,
		papiQuery(contractID, groupID),
	})
}

func papiQuery(contractID, groupID string) string {
	return concat([]string{
		"?contractId=",
		contractID,
		"&groupId=",
		groupID,
	})
}

func papiHostnamesEndpoint(c *AuthCredentials, contractID, groupID string) string {
	return concat([]string{
		papiBase(c),
		"edgehostnames",
		papiQuery(contractID, groupID),
	})
}

func papiHostnameEndpoint(c *AuthCredentials, hostID, contractID, groupID string) string {
	return concat([]string{
		papiBase(c),
		"edgehostnames/",
		hostID,
		papiQuery(contractID, groupID),
	})
}

func papiPropertiesBase(c *AuthCredentials) string {
	return concat([]string{
		papiBase(c),
		"properties/",
	})
}

func papiPropertiesEndpoint(c *AuthCredentials, contractID, groupID string) string {
	return concat([]string{
		papiPropertiesBase(c),
		papiQuery(contractID, groupID),
	})
}

func papiPropertyBase(c *AuthCredentials, propID string) string {
	return concat([]string{
		papiPropertiesBase(c),
		propID,
	})
}

func papiPropertyEndpoint(c *AuthCredentials, propID, contractID, groupID string) string {
	return concat([]string{
		papiPropertyBase(c, propID),
		papiQuery(contractID, groupID),
	})
}

func papiPropertyVersionsBase(c *AuthCredentials, propID, contractID, groupID string) string {
	return concat([]string{
		papiPropertyBase(c, propID),
		"/versions",
	})
}

func papiPropertyVersionsEndpoint(c *AuthCredentials, propID, contractID, groupID string) string {
	return concat([]string{
		papiPropertyVersionsBase(c, propID, contractID, groupID),
		papiQuery(contractID, groupID),
	})
}

func papiPropertyVersionBase(c *AuthCredentials, version, propID, contractID, groupID string) string {
	return concat([]string{
		papiPropertyVersionsBase(c, propID, contractID, groupID),
		"/",
		version,
	})
}

func papiPropertyVersionEndpoint(c *AuthCredentials, version, propID, contractID, groupID string) string {
	return concat([]string{
		papiPropertyVersionBase(c, version, propID, contractID, groupID),
		papiQuery(contractID, groupID),
	})
}

func papiPropertyLatestVersionEndpoint(c *AuthCredentials, propID, contractID, groupID string) string {
	return concat([]string{
		papiPropertyVersionsBase(c, propID, contractID, groupID),
		"/latest",
		papiQuery(contractID, groupID),
	})
}

func papiPropertyRulesEndpoint(c *AuthCredentials, propID, version, contractID, groupID string) string {
	return concat([]string{
		papiPropertyVersionBase(c, version, propID, contractID, groupID),
		"/rules/",
		papiQuery(contractID, groupID),
	})
}

func papiActivationsEndpoint(c *AuthCredentials, propID, contractID, groupID string) string {
	return concat([]string{
		papiPropertyBase(c, propID),
		"/activations",
		papiQuery(contractID, groupID),
	})
}
