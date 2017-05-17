package compute

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
)

// ResourceClient is an AuthenticatedClient with some additional information about the resources to be addressed.
type ResourceClient struct {
	*Client
	ResourceDescription string
	ContainerPath       string
	ResourceRootPath    string
}

func (c *ResourceClient) createResource(requestBody interface{}, responseBody interface{}) error {
	resp, err := c.executeRequest("POST", c.ContainerPath, requestBody)
	if err != nil {
		return err
	}

	return c.unmarshalResponseBody(resp, responseBody)
}

func (c *ResourceClient) updateResource(name string, requestBody interface{}, responseBody interface{}) error {
	resp, err := c.executeRequest("PUT", c.getObjectPath(c.ResourceRootPath, name), requestBody)
	if err != nil {
		return err
	}

	return c.unmarshalResponseBody(resp, responseBody)
}

func (c *ResourceClient) getResource(name string, responseBody interface{}) error {
	var objectPath string
	if name != "" {
		objectPath = c.getObjectPath(c.ResourceRootPath, name)
	} else {
		objectPath = c.ResourceRootPath
	}
	resp, err := c.executeRequest("GET", objectPath, nil)
	if err != nil {
		return err
	}

	return c.unmarshalResponseBody(resp, responseBody)
}

func (c *ResourceClient) deleteResource(name string) error {
	var objectPath string
	if name != "" {
		objectPath = c.getObjectPath(c.ResourceRootPath, name)
	} else {
		objectPath = c.ResourceRootPath
	}
	_, err := c.executeRequest("DELETE", objectPath, nil)
	if err != nil {
		return err
	}

	// No errors and no response body to write
	return nil
}

func (c *ResourceClient) unmarshalResponseBody(resp *http.Response, iface interface{}) error {
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	c.debugLogString(fmt.Sprintf("HTTP Resp (%d): %s", resp.StatusCode, buf.String()))
	// JSON decode response into interface
	var tmp interface{}
	dcd := json.NewDecoder(buf)
	if err := dcd.Decode(&tmp); err != nil {
		return err
	}

	// Use mapstructure to weakly decode into the resulting interface
	msdcd, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           iface,
		TagName:          "json",
	})
	if err != nil {
		return err
	}

	if err := msdcd.Decode(tmp); err != nil {
		return err
	}
	return nil
}
