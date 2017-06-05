package compute

import (
	"fmt"
	"path/filepath"
)

// IPAddressReservationsClient is a client to manage ip address reservation resources
type IPAddressReservationsClient struct {
	*ResourceClient
}

const (
	IPAddressReservationDescription   = "IP Address Reservation"
	IPAddressReservationContainerPath = "/network/v1/ipreservation/"
	IPAddressReservationResourcePath  = "/network/v1/ipreservation"
	IPAddressReservationQualifier     = "/oracle/public"
)

// IPAddressReservations returns an IPAddressReservationsClient to manage IP address reservation
// resources
func (c *Client) IPAddressReservations() *IPAddressReservationsClient {
	return &IPAddressReservationsClient{
		ResourceClient: &ResourceClient{
			Client:              c,
			ResourceDescription: IPAddressReservationDescription,
			ContainerPath:       IPAddressReservationContainerPath,
			ResourceRootPath:    IPAddressReservationResourcePath,
		},
	}
}

// IPAddressReservation describes an IP Address reservation
type IPAddressReservation struct {
	// Description of the IP Address Reservation
	Description string `json:"description"`

	// Reserved NAT IPv4 address from the IP Address Pool
	IPAddress string `json:"ipAddress"`

	// Name of the IP Address pool to reserve the NAT IP from
	IPAddressPool string `json:"ipAddressPool"`

	// Name of the reservation
	Name string `json:"name"`

	// Tags associated with the object
	Tags []string `json:"tags"`

	// Uniform Resource Identified for the reservation
	Uri string `json:"uri"`
}

const (
	PublicIPAddressPool  = "public-ippool"
	PrivateIPAddressPool = "cloud-ippool"
)

// CreateIPAddressReservationInput defines input parameters to create an ip address reservation
type CreateIPAddressReservationInput struct {
	// Description of the IP Address Reservation
	// Optional
	Description string `json:"description"`

	// IP Address pool from which to reserve an IP Address.
	// Can be one of the following:
	//
	// 'public-ippool' - When you attach an IP Address from this pool to an instance, you enable
	//                   access between the public Internet and the instance
	// 'cloud-ippool' - When you attach an IP Address from this pool to an instance, the instance
	//                  can communicate privately with other Oracle Cloud Services
	// Optional
	IPAddressPool string `json:"ipAddressPool"`

	// The name of the reservation to create
	// Required
	Name string `json:"name"`

	// Tags to associate with the IP Reservation
	// Optional
	Tags []string `json:"tags"`
}

// Takes an input struct, creates an IP Address reservation, and returns the info struct and any errors
func (c *IPAddressReservationsClient) CreateIPAddressReservation(input *CreateIPAddressReservationInput) (*IPAddressReservation, error) {
	var ipAddrRes IPAddressReservation
	// Qualify supplied name
	input.Name = c.getQualifiedName(input.Name)
	// Qualify supplied address pool if not nil
	if input.IPAddressPool != "" {
		input.IPAddressPool = c.qualifyIPAddressPool(input.IPAddressPool)
	}

	if err := c.createResource(input, &ipAddrRes); err != nil {
		return nil, err
	}

	return c.success(&ipAddrRes)
}

// Parameters to retrieve information on an ip address reservation
type GetIPAddressReservationInput struct {
	// Name of the IP Reservation
	// Required
	Name string `json:"name"`
}

// Returns an IP Address Reservation and any errors
func (c *IPAddressReservationsClient) GetIPAddressReservation(input *GetIPAddressReservationInput) (*IPAddressReservation, error) {
	var ipAddrRes IPAddressReservation

	input.Name = c.getQualifiedName(input.Name)
	if err := c.getResource(input.Name, &ipAddrRes); err != nil {
		return nil, err
	}

	return c.success(&ipAddrRes)
}

// Parameters to update an IP Address reservation
type UpdateIPAddressReservationInput struct {
	// Description of the IP Address Reservation
	// Optional
	Description string `json:"description"`

	// IP Address pool from which to reserve an IP Address.
	// Can be one of the following:
	//
	// 'public-ippool' - When you attach an IP Address from this pool to an instance, you enable
	//                   access between the public Internet and the instance
	// 'cloud-ippool' - When you attach an IP Address from this pool to an instance, the instance
	//                  can communicate privately with other Oracle Cloud Services
	// Optional
	IPAddressPool string `json:"ipAddressPool"`

	// The name of the reservation to create
	// Required
	Name string `json:"name"`

	// Tags to associate with the IP Reservation
	// Optional
	Tags []string `json:"tags"`
}

func (c *IPAddressReservationsClient) UpdateIPAddressReservation(input *UpdateIPAddressReservationInput) (*IPAddressReservation, error) {
	var ipAddrRes IPAddressReservation

	// Qualify supplied name
	input.Name = c.getQualifiedName(input.Name)
	// Qualify supplied address pool if not nil
	if input.IPAddressPool != "" {
		input.IPAddressPool = c.qualifyIPAddressPool(input.IPAddressPool)
	}

	if err := c.updateResource(input.Name, input, &ipAddrRes); err != nil {
		return nil, err
	}

	return c.success(&ipAddrRes)
}

// Parameters to delete an IP Address Reservation
type DeleteIPAddressReservationInput struct {
	// The name of the reservation to delete
	Name string `json:"name"`
}

func (c *IPAddressReservationsClient) DeleteIPAddressReservation(input *DeleteIPAddressReservationInput) error {
	input.Name = c.getQualifiedName(input.Name)
	return c.deleteResource(input.Name)
}

func (c *IPAddressReservationsClient) success(result *IPAddressReservation) (*IPAddressReservation, error) {
	c.unqualify(&result.Name)
	if result.IPAddressPool != "" {
		result.IPAddressPool = c.unqualifyIPAddressPool(result.IPAddressPool)
	}

	return result, nil
}

func (c *IPAddressReservationsClient) qualifyIPAddressPool(input string) string {
	// Add '/oracle/public/'
	return fmt.Sprintf("%s/%s", IPAddressReservationQualifier, input)
}

func (c *IPAddressReservationsClient) unqualifyIPAddressPool(input string) string {
	// Remove '/oracle/public/'
	return filepath.Base(input)
}
