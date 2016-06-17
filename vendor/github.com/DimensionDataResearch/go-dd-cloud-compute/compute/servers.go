package compute

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Server represents a virtual machine.
type Server struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	OperatingSystem OperatingSystem       `json:"operatingSystem"`
	CPU             VirtualMachineCPU     `json:"cpu"`
	MemoryGB        int                   `json:"memoryGb"`
	Disks           []VirtualMachineDisk  `json:"disk"`
	Network         VirtualMachineNetwork `json:"networkInfo"`
	SourceImageID   string                `json:"sourceImageId"`
	State           string                `json:"state"`
}

// GetID returns the server's Id.
func (server *Server) GetID() string {
	return server.ID
}

// GetName returns the server's name.
func (server *Server) GetName() string {
	return server.Name
}

// GetState returns the server's current state.
func (server *Server) GetState() string {
	return server.State
}

// IsDeleted determines whether the server has been deleted (is nil).
func (server *Server) IsDeleted() bool {
	return server == nil
}

var _ Resource = &Server{}

// ServerDeploymentConfiguration represents the configuration for deploying a virtual machine.
type ServerDeploymentConfiguration struct {
	Name                  string                `json:"name"`
	Description           string                `json:"description"`
	ImageID               string                `json:"imageId"`
	AdministratorPassword string                `json:"administratorPassword"`
	CPU                   VirtualMachineCPU     `json:"cpu"`
	MemoryGB              int                   `json:"memoryGb,omitempty"`
	Disks                 []VirtualMachineDisk  `json:"disk"`
	Network               VirtualMachineNetwork `json:"networkInfo"`
	PrimaryDNS            string                `json:"primaryDns"`
	SecondaryDNS          string                `json:"secondaryDns"`
	Start                 bool                  `json:"start"`
}

// NotifyServerIPAddressChange represents the request body when notifying the system that the IP address for a server's network adapter has changed.
// Exactly at least 1 of IPv4Address or IPv6Address must be specified.
type NotifyServerIPAddressChange struct {
	// The server's network adapter Id.
	AdapterID string `json:"nicId"`

	// The server's new private IPv4 address.
	IPv4Address *string `json:"privateIpv4,omitempty"`

	// The server's new private IPv6 address.
	IPv6Address *string `json:"ipv6,omitempty"`
}

// ReconfigureServer represents the request body when updating a server's configuration (e.g. memory, CPU count).
type ReconfigureServer struct {
	ServerID string `json:"id"`
	MemoryGB *int   `json:"memoryGb,omitempty"`
	CPUCount *int   `json:"cpuCount,omitempty"`
}

// ApplyImage applies the specified image (and its default values for CPU, memory, and disks) to the ServerDeploymentConfiguration.
func (config *ServerDeploymentConfiguration) ApplyImage(image *OSImage) error {
	if image == nil {
		return fmt.Errorf("Cannot apply image defaults (no image was supplied).")
	}

	config.ImageID = image.ID
	config.CPU = image.CPU
	config.MemoryGB = image.MemoryGB
	config.Disks = make([]VirtualMachineDisk, len(image.Disks))
	for index, disk := range image.Disks {
		config.Disks[index] = disk
	}

	return nil
}

// DeleteServer represents a request to delete a compute virtual machine.
type DeleteServer struct {
	// The server Id.
	ID string `json:"id"`
}

// GetServer retrieves the server with the specified Id.
// id is the Id of the server to retrieve.
// Returns nil if no server is found with the specified Id.
func (client *Client) GetServer(id string) (server *Server, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return nil, err
	}

	requestURI := fmt.Sprintf("%s/server/server/%s", organizationID, id)
	request, err := client.newRequestV22(requestURI, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return nil, err
	}

	if statusCode != http.StatusOK {
		var apiResponse *APIResponse

		apiResponse, err = readAPIResponseAsJSON(responseBody, statusCode)
		if err != nil {
			return nil, err
		}

		if apiResponse.ResponseCode == ResponseCodeResourceNotFound {
			return nil, nil // Not an error, but was not found.
		}

		return nil, apiResponse.ToError("Request to retrieve Server failed with status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	server = &Server{}
	err = json.Unmarshal(responseBody, server)

	return server, err
}

// DeployServer deploys a new virtual machine.
func (client *Client) DeployServer(serverConfiguration ServerDeploymentConfiguration) (serverID string, err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return "", err
	}

	requestURI := fmt.Sprintf("%s/server/deployServer", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost, &serverConfiguration)
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return "", err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return "", err
	}

	if apiResponse.ResponseCode != ResponseCodeInProgress {
		return "", apiResponse.ToError("Request to deploy server '%s' failed with status code %d (%s): %s", serverConfiguration.Name, statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	// Expected: "info" { "name": "serverId", "value": "the-Id-of-the-new-server" }
	if len(apiResponse.FieldMessages) != 1 || apiResponse.FieldMessages[0].FieldName != "serverId" {
		return "", apiResponse.ToError("Received an unexpected response (missing 'serverId') with status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return apiResponse.FieldMessages[0].Message, nil
}

// DeleteServer deletes an existing Server.
// Returns an error if the operation was not successful.
func (client *Client) DeleteServer(id string) (err error) {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return err
	}

	requestURI := fmt.Sprintf("%s/server/deleteServer", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost, &DeleteServer{id})
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return err
	}

	if apiResponse.ResponseCode != ResponseCodeInProgress {
		return apiResponse.ToError("Request to delete server failed with unexpected status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return nil
}

// NotifyServerIPAddressChange notifies the system that the IP address for a server's network adapter has changed.
// serverNetworkAdapterID is the Id of the server's network adapter.
// Must specify at least one of newIPv4Address / newIPv6Address.
func (client *Client) NotifyServerIPAddressChange(networkAdapterID string, newIPv4Address *string, newIPv6Address *string) error {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return err
	}

	requestURI := fmt.Sprintf("%s/server/notifyNicIpChange", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost, &NotifyServerIPAddressChange{
		AdapterID:   networkAdapterID,
		IPv4Address: newIPv4Address,
		IPv6Address: newIPv6Address,
	})
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return err
	}

	if apiResponse.ResponseCode != ResponseCodeInProgress {
		return apiResponse.ToError("Request to notify change of server IP address failed with unexpected status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return nil
}

// ReconfigureServer updates the configuration for a server.
// serverID is the Id of the server.
// Must specify at least one of newIPv4Address / newIPv6Address.
func (client *Client) ReconfigureServer(serverID string, memoryGB *int, cpuCount *int) error {
	organizationID, err := client.getOrganizationID()
	if err != nil {
		return err
	}

	requestURI := fmt.Sprintf("%s/server/reconfigureServer", organizationID)
	request, err := client.newRequestV22(requestURI, http.MethodPost, &ReconfigureServer{
		ServerID: serverID,
		MemoryGB: memoryGB,
		CPUCount: cpuCount,
	})
	responseBody, statusCode, err := client.executeRequest(request)
	if err != nil {
		return err
	}

	apiResponse, err := readAPIResponseAsJSON(responseBody, statusCode)
	if err != nil {
		return err
	}

	if apiResponse.ResponseCode != ResponseCodeOK && apiResponse.ResponseCode != ResponseCodeInProgress {
		return apiResponse.ToError("Request to reconfigure server failed with unexpected status code %d (%s): %s", statusCode, apiResponse.ResponseCode, apiResponse.Message)
	}

	return nil
}
