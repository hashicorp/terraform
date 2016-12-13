package edgegrid

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
)

// Domains returns the Akamai GTM DomainSummary for each domain the GTMClient
// is authorized to view and modify
func (c *GTMClient) Domains() ([]DomainSummary, error) {
	domains := &Domains{}
	err := resourceRequest(c, "GET", domainsEndpoint(c.GetCredentials()), nil, domains)
	if err != nil {
		return []DomainSummary{}, err
	}
	return domains.Domains, err
}

// Domain takes an Akamai GTM domain name and returns a Domain.
func (c *GTMClient) Domain(name string) (*Domain, error) {
	domain := &Domain{}
	err := resourceRequest(c, "GET", domainEndpoint(c.GetCredentials(), name), nil, domain)
	return domain, err
}

// DomainStatus takes an Akamai GTM domain name and returns its ResourceStatus.
func (c *GTMClient) DomainStatus(name string) (*ResourceStatus, error) {
	status := &ResourceStatus{}
	err := resourceRequest(c, "GET", domainStatusEndpoint(c.GetCredentials(), name), nil, status)
	return status, err
}

// DomainCreate issues a request to create a domain with the provided name and type.
// The result is returned as a DomainResponse.
func (c *GTMClient) DomainCreate(name string, domainType string) (*DomainResponse, error) {
	payload := map[string]string{
		"name": name,
		"type": domainType,
	}

	jsonRequest, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var createdDomain = &DomainResponse{}
	err = resourceRequest(c, "PUT", domainEndpoint(c.GetCredentials(), name), jsonRequest, createdDomain)
	return createdDomain, err
}

// DomainUpdate takes a domain and issues a request to update it accordingly.
// The result is returned as a DomainResponse.
func (c *GTMClient) DomainUpdate(domain *Domain) (*DomainResponse, error) {
	jsonRequest, err := json.Marshal(domain)
	if err != nil {
		return nil, err
	}
	var updatedDomain = &DomainResponse{}
	err = resourceRequest(c, "PUT", domainEndpoint(c.GetCredentials(), domain.Name), jsonRequest, updatedDomain)
	return updatedDomain, err
}

// DomainDelete takes a domain and issues a request to delete it.
func (c *GTMClient) DomainDelete(name string) error {
	resp, err := doClientReq(c, "DELETE", domainEndpoint(c.GetCredentials(), name), nil)

	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("HTTP status not OK")
	}
	return err
}

// DataCenters takes an Akamai GTM domain name and returns a []DataCenter
// representing the datacenters associated with the domain.
func (c *GTMClient) DataCenters(domain string) ([]DataCenter, error) {
	dcs := &DataCenters{}
	err := resourceRequest(c, "GET", dcsEndpoint(c.GetCredentials(), domain), nil, dcs)
	if err != nil {
		return []DataCenter{}, err
	}
	return dcs.Items, err
}

// DataCenterCreate takes an Akamai GTM domain name and a dc and
// issues a request to create the DataCenter via the Akamai GTM API.
// The result is returned as a DataCenterResponse.
func (c *GTMClient) DataCenterCreate(domain string, dc *DataCenter) (*DataCenterResponse, error) {
	jsonRequest, err := json.Marshal(dc)
	if err != nil {
		return nil, err
	}
	var resource = &DataCenterResponse{}
	err = resourceRequest(c, "POST", dcsEndpoint(c.GetCredentials(), domain), jsonRequest, resource)
	return resource, err
}

// DataCenter takes an Akamai GTM domain name and a datacenter id and
// returns the DataCenter for the given ID.
func (c *GTMClient) DataCenter(domain string, id int) (*DataCenter, error) {
	dc := &DataCenter{}
	err := resourceRequest(c, "GET", dcEndpoint(c.GetCredentials(), domain, id), nil, dc)
	return dc, err
}

// DataCenterUpdate takes an Akamai GTM domain name and a dc and issues a request
// to update the DataCenter details accordingly via the Akamai GTM API.
// The result is returned as a DataCenterResponse.
func (c *GTMClient) DataCenterUpdate(domain string, dc *DataCenter) (*DataCenterResponse, error) {
	jsonRequest, err := json.Marshal(dc)
	if err != nil {
		return nil, err
	}
	var resource = &DataCenterResponse{}
	err = resourceRequest(c, "PUT", dcEndpoint(c.GetCredentials(), domain, dc.DataCenterID), jsonRequest, resource)
	return resource, err
}

// DataCenterByName takes an Akamai GTM domain name and a datacenter name
// and returns the matching DataCenter.
func (c *GTMClient) DataCenterByName(domain string, name string) (*DataCenter, error) {
	dcs, err := c.DataCenters(domain)
	if err != nil {
		return nil, err
	}

	for _, each := range dcs {
		if each.Nickname == name {
			return &each, nil
		}
	}
	return &DataCenter{}, fmt.Errorf("DataCenter named: %s not found in domain: %s", name, domain)
}

// DataCenterDelete takes an Akamai GTM domain name and a datacenter id and
// issues a request to delete the matching datacenter via the Akamai GTM API.
func (c *GTMClient) DataCenterDelete(domain string, id int) error {
	resp, err := doClientReq(c, "DELETE", dcEndpoint(c.GetCredentials(), domain, id), nil)

	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New("HTTP status not OK")
	}
	return err
}

// Properties takes an Akamai GTM domain name and returns the Akamai GTM properties
// associated with the domain.
func (c *GTMClient) Properties(domain string) (*Properties, error) {
	props := &Properties{}
	err := resourceRequest(c, "GET", propertiesEndpoint(c.GetCredentials(), domain), nil, props)
	return props, err
}

// PropertiesSorted sorted takes an Akamai GTM domain name and returns the Akamai GTM
// properties associated with the domain name, sorted by their names.
func (c *GTMClient) PropertiesSorted(domain string) (*Properties, error) {
	props, err := c.Properties(domain)
	sort.Sort(props)
	return props, err
}

// Property takes an Akamai GTM domain name and a property name
// and returns the matching Akamai GTM property.
func (c *GTMClient) Property(domain, property string) (*Property, error) {
	prop := &Property{}
	err := resourceRequest(c, "GET", propertyEndpoint(c.GetCredentials(), domain, property), nil, prop)
	return prop, err
}

// PropertyCreate takes an Akamai GTM domain name and a property and issues a request
// to create the Property via the Akamai GTM API.
// The result is returned as a PropertyResponse.
func (c *GTMClient) PropertyCreate(domain string, property *Property) (*PropertyResponse, error) {
	jsonRequest, err := json.Marshal(property)
	if err != nil {
		return nil, err
	}

	resource := &PropertyResponse{}
	err = resourceRequest(c, "PUT", propertyEndpoint(c.GetCredentials(), domain, property.Name), jsonRequest, resource)
	return resource, err
}

// PropertyUpdate takes an Akamai GTM domain name and a property and issues a request
// to update the property accordingly via the Akamai GTM API.
// The result is returned as a PropertyResponse.
func (c *GTMClient) PropertyUpdate(domain string, property *Property) (*PropertyResponse, error) {
	return c.PropertyCreate(domain, property)
}

// PropertyDelete takes an Akamai GTM domain name and a property name and issues a request
// to delete the matching Akamai GTM property via the Akamai GTM API.
// It returns true if the action was successful and false if it was not successful.
func (c *GTMClient) PropertyDelete(domain string, property string) (bool, error) {
	url := propertyEndpoint(c.GetCredentials(), domain, property)
	resp, err := doClientReq(c, "DELETE", url, nil)

	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, errors.New("HTTP status not OK")
	}
	return true, nil
}
