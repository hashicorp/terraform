package cloudapi

import (
	"net/http"

	"github.com/joyent/gocommon/client"
	"github.com/joyent/gocommon/errors"
)

// Snapshot represent a point in time state of a machine.
type Snapshot struct {
	Name  string // Snapshot name
	State string // Snapshot state
}

// SnapshotOpts represent the option that can be specified
// when creating a new machine snapshot.
type SnapshotOpts struct {
	Name string `json:"name"` // Snapshot name
}

// CreateMachineSnapshot creates a new snapshot for the machine with the options specified.
// See API docs: http://apidocs.joyent.com/cloudapi/#CreateMachineSnapshot
func (c *Client) CreateMachineSnapshot(machineID string, opts SnapshotOpts) (*Snapshot, error) {
	var resp Snapshot
	req := request{
		method:         client.POST,
		url:            makeURL(apiMachines, machineID, apiSnapshots),
		reqValue:       opts,
		resp:           &resp,
		expectedStatus: http.StatusCreated,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to create snapshot %s from machine with id %s", opts.Name, machineID)
	}
	return &resp, nil
}

// StartMachineFromSnapshot starts the machine from the specified snapshot.
// Machine must be in 'stopped' state.
// See API docs: http://apidocs.joyent.com/cloudapi/#StartMachineFromSnapshot
func (c *Client) StartMachineFromSnapshot(machineID, snapshotName string) error {
	req := request{
		method:         client.POST,
		url:            makeURL(apiMachines, machineID, apiSnapshots, snapshotName),
		expectedStatus: http.StatusAccepted,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to start machine with id %s from snapshot %s", machineID, snapshotName)
	}
	return nil
}

// ListMachineSnapshots lists all snapshots for the specified machine.
// See API docs: http://apidocs.joyent.com/cloudapi/#ListMachineSnapshots
func (c *Client) ListMachineSnapshots(machineID string) ([]Snapshot, error) {
	var resp []Snapshot
	req := request{
		method: client.GET,
		url:    makeURL(apiMachines, machineID, apiSnapshots),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get list of snapshots for machine with id %s", machineID)
	}
	return resp, nil
}

// GetMachineSnapshot returns the state of the specified snapshot.
// See API docs: http://apidocs.joyent.com/cloudapi/#GetMachineSnapshot
func (c *Client) GetMachineSnapshot(machineID, snapshotName string) (*Snapshot, error) {
	var resp Snapshot
	req := request{
		method: client.GET,
		url:    makeURL(apiMachines, machineID, apiSnapshots, snapshotName),
		resp:   &resp,
	}
	if _, err := c.sendRequest(req); err != nil {
		return nil, errors.Newf(err, "failed to get snapshot %s for machine with id %s", snapshotName, machineID)
	}
	return &resp, nil
}

// DeleteMachineSnapshot deletes the specified snapshot.
// See API docs: http://apidocs.joyent.com/cloudapi/#DeleteMachineSnapshot
func (c *Client) DeleteMachineSnapshot(machineID, snapshotName string) error {
	req := request{
		method:         client.DELETE,
		url:            makeURL(apiMachines, machineID, apiSnapshots, snapshotName),
		expectedStatus: http.StatusNoContent,
	}
	if _, err := c.sendRequest(req); err != nil {
		return errors.Newf(err, "failed to delete snapshot %s for machine with id %s", snapshotName, machineID)
	}
	return nil
}
