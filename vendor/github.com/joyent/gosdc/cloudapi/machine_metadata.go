package cloudapi

import (
	"net/http"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// UpdateMachineMetadata updates the metadata for a given machine.
// Any metadata keys passed in here are created if they do not exist, and
// overwritten if they do.
// See API docs: http://apidocs.joyent.com/cloudapi/#UpdateMachineMetadata
func (c *Client) UpdateMachineMetadata(machineID string, metadata map[string]string) (map[string]interface{}, error) {
	var resp map[string]interface{}
	req := request{
		method:   client.POST,
		url:      makeURL(apiMachines, machineID, apiMetadata),
		reqValue: metadata,
		resp:     &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to update metadata for machine with id %s", machineID)
	}
	return resp, nil
}

// GetMachineMetadata returns the complete set of metadata associated with the
// specified machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetMachineMetadata
func (c *Client) GetMachineMetadata(machineID string) (map[string]interface{}, error) {
	var resp map[string]interface{}
	req := request{
		method: client.GET,
		url:    makeURL(apiMachines, machineID, apiMetadata),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of metadata for machine with id %s", machineID)
	}
	return resp, nil
}

// DeleteMachineMetadata deletes a single metadata key from the specified machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#DeleteMachineMetadata
func (c *Client) DeleteMachineMetadata(machineID, metadataKey string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiMachines, machineID, apiMetadata, metadataKey),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete metadata with key %s for machine with id %s", metadataKey, machineID)
	}
	return nil
}

// DeleteAllMachineMetadata deletes all metadata keys from the specified machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#DeleteAllMachineMetadata
func (c *Client) DeleteAllMachineMetadata(machineID string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiMachines, machineID, apiMetadata),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete metadata for machine with id %s", machineID)
	}
	return nil
}
