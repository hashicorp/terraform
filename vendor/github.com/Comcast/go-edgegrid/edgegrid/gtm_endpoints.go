package edgegrid

import (
	"fmt"
)

const gtmPath = "/config-gtm/v1/"

func gtmBase(c *AuthCredentials) string {
	return concat([]string{
		c.APIHost,
		gtmPath,
	})
}

func domainsEndpoint(c *AuthCredentials) string {
	return concat([]string{
		gtmBase(c),
		"domains",
	})
}

func domainEndpoint(c *AuthCredentials, domain string) string {
	return concat([]string{
		domainsEndpoint(c),
		"/",
		domain,
	})
}

func domainStatusEndpoint(c *AuthCredentials, domain string) string {
	return concat([]string{
		domainsEndpoint(c),
		"/",
		domain,
		"/status/current",
	})
}

func dcEndpoint(c *AuthCredentials, domain string, id int) string {
	return fmt.Sprintf("%s/%d", dcsEndpoint(c, domain), id)
}

func dcsEndpoint(c *AuthCredentials, domain string) string {
	return concat([]string{
		domainEndpoint(c, domain),
		"/datacenters",
	})
}

func propertiesEndpoint(c *AuthCredentials, domain string) string {
	return concat([]string{
		domainEndpoint(c, domain),
		"/properties",
	})
}

func propertyEndpoint(c *AuthCredentials, domain, property string) string {
	return concat([]string{
		propertiesEndpoint(c, domain),
		"/",
		property,
	})
}
