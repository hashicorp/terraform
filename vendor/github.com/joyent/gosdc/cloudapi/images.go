package cloudapi

import (
	"fmt"
	"net/http"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// Image represent the software packages that will be available on newly provisioned machines
type Image struct {
	Id           string                 // Unique identifier for the image
	Name         string                 // Image friendly name
	OS           string                 // Underlying operating system
	Version      string                 // Image version
	Type         string                 // Image type, one of 'smartmachine' or 'virtualmachine'
	Description  string                 // Image description
	Requirements map[string]interface{} // Minimum requirements for provisioning a machine with this image, e.g. 'password' indicates that a password must be provided
	Homepage     string                 // URL for a web page including detailed information for this image (new in API version 7.0)
	PublishedAt  string                 `json:"published_at"` // Time this image has been made publicly available (new in API version 7.0)
	Public       bool                   // Indicates if the image is publicly available (new in API version 7.1)
	State        string                 // Current image state. One of 'active', 'unactivated', 'disabled', 'creating', 'failed' (new in API version 7.1)
	Tags         map[string]string      // A map of key/value pairs that allows clients to categorize images by any given criteria (new in API version 7.1)
	EULA         string                 // URL of the End User License Agreement (EULA) for the image (new in API version 7.1)
	ACL          []string               // An array of account UUIDs given access to a private image. The field is only relevant to private images (new in API version 7.1)
	Owner        string                 // The UUID of the user owning the image
}

// ExportImageOpts represent the option that can be specified
// when exporting an image.
type ExportImageOpts struct {
	MantaPath string `json:"manta_path"` // The Manta path prefix to use when exporting the image
}

// MantaLocation represent the properties that allow a user
// to retrieve the image file and manifest from Manta
type MantaLocation struct {
	MantaURL     string `json:"manta_url"`     // Manta datacenter URL
	ImagePath    string `json:"image_path"`    // Path to the image
	ManifestPath string `json:"manifest_path"` // Path to the image manifest
}

// CreateImageFromMachineOpts represent the option that can be specified
// when creating a new image from an existing machine.
type CreateImageFromMachineOpts struct {
	Machine     string            `json:"machine"`               // The machine UUID from which the image is to be created
	Name        string            `json:"name"`                  // Image name
	Version     string            `json:"version"`               // Image version
	Description string            `json:"description,omitempty"` // Image description
	Homepage    string            `json:"homepage,omitempty"`    // URL for a web page including detailed information for this image
	EULA        string            `json:"eula,omitempty"`        // URL of the End User License Agreement (EULA) for the image
	ACL         []string          `json:"acl,omitempty"`         // An array of account UUIDs given access to a private image. The field is only relevant to private images
	Tags        map[string]string `json:"tags,omitempty"`        // A map of key/value pairs that allows clients to categorize images by any given criteria
}

// ListImages provides a list of images available in the datacenter.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListImages
func (c *Client) ListImages(filter *Filter) ([]Image, error) {
	var resp []Image
	req := request{
		method: client.GET,
		url:    apiImages,
		filter: filter,
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of images")
	}
	return resp, nil
}

// GetImage returns the image specified by imageId.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetImage
func (c *Client) GetImage(imageID string) (*Image, error) {
	var resp Image
	req := request{
		method: client.GET,
		url:    makeURL(apiImages, imageID),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get image with id: %s", imageID)
	}
	return &resp, nil
}

// DeleteImage (Beta) Delete the image specified by imageId. Must be image owner to do so.
// See API docs: http://apidocs.joyent.com/cloudapi/#DeleteImage
func (c *Client) DeleteImage(imageID string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiImages, imageID),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete image with id: %s", imageID)
	}
	return nil
}

// ExportImage (Beta) Exports an image to the specified Manta path.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListImages
func (c *Client) ExportImage(imageID string, opts ExportImageOpts) (*MantaLocation, error) {
	var resp MantaLocation
	req := request{
		method:   client.POST,
		url:      fmt.Sprintf("%s/%s?action=%s", apiImages, imageID, actionExport),
		reqValue: opts,
		resp:     &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to export image %s to %s", imageID, opts.MantaPath)
	}
	return &resp, nil
}

// CreateImageFromMachine (Beta) Create a new custom image from a machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListImages
func (c *Client) CreateImageFromMachine(opts CreateImageFromMachineOpts) (*Image, error) {
	var resp Image
	req := request{
		method:         client.POST,
		url:            apiImages,
		reqValue:       opts,
		resp:           &resp,
		expectedStatus: http.StatusCreated,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to create image from machine %s", opts.Machine)
	}
	return &resp, nil
}
