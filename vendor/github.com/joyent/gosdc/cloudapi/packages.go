package cloudapi

import (
	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// Package represents a named collections of resources that are used to describe the 'sizes'
// of either a smart machine or a virtual machine.
type Package struct {
	Name        string // Name for the package
	Memory      int    // Memory available (in Mb)
	Disk        int    // Disk space available (in Gb)
	Swap        int    // Swap memory available (in Mb)
	VCPUs       int    // Number of VCPUs for the package
	Default     bool   // Indicates whether this is the default package in the datacenter
	Id          string // Unique identifier for the package
	Version     string // Version for the package
	Group       string // Group this package belongs to
	Description string // Human friendly description for the package
}

// ListPackages provides a list of packages available in the datacenter.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListPackages
func (c *Client) ListPackages(filter *Filter) ([]Package, error) {
	var resp []Package
	req := request{
		method: client.GET,
		url:    apiPackages,
		filter: filter,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of packages")
	}
	return resp, nil
}

// GetPackage returns the package specified by packageName. NOTE: packageName can
// specify either the package name or package ID.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetPackage
func (c *Client) GetPackage(packageName string) (*Package, error) {
	var resp Package
	req := request{
		method: client.GET,
		url:    makeURL(apiPackages, packageName),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get package with name: %s", packageName)
	}
	return &resp, nil
}
