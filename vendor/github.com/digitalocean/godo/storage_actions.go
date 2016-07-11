package godo

import "fmt"

// StorageActionsService is an interface for interfacing with the
// storage actions endpoints of the Digital Ocean API.
// See: https://developers.digitalocean.com/documentation/v2#storage-actions
type StorageActionsService interface {
	Attach(volumeID string, dropletID int) (*Action, *Response, error)
	Detach(volumeID string) (*Action, *Response, error)
}

// StorageActionsServiceOp handles communication with the floating IPs
// action related methods of the DigitalOcean API.
type StorageActionsServiceOp struct {
	client *Client
}

// StorageAttachment represents the attachement of a block storage
// volume to a specific droplet under the device name.
type StorageAttachment struct {
	DropletID int `json:"droplet_id"`
}

// Attach a storage volume to a droplet.
func (s *StorageActionsServiceOp) Attach(volumeID string, dropletID int) (*Action, *Response, error) {
	request := &ActionRequest{
		"type":       "attach",
		"droplet_id": dropletID,
	}
	return s.doAction(volumeID, request)
}

// Detach a storage volume from a droplet.
func (s *StorageActionsServiceOp) Detach(volumeID string) (*Action, *Response, error) {
	request := &ActionRequest{
		"type": "detach",
	}
	return s.doAction(volumeID, request)
}

func (s *StorageActionsServiceOp) doAction(volumeID string, request *ActionRequest) (*Action, *Response, error) {
	path := storageAllocationActionPath(volumeID)

	req, err := s.client.NewRequest("POST", path, request)
	if err != nil {
		return nil, nil, err
	}

	root := new(actionRoot)
	resp, err := s.client.Do(req, root)
	if err != nil {
		return nil, resp, err
	}

	return &root.Event, resp, err
}

func storageAllocationActionPath(volumeID string) string {
	return fmt.Sprintf("%s/%s/actions", storageAllocPath, volumeID)
}
