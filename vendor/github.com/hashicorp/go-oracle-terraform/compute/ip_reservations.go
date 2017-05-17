package compute

// IPReservationsClient is a client for the IP Reservations functions of the Compute API.
type IPReservationsClient struct {
	*ResourceClient
}

const (
	IPReservationDesc          = "ip reservation"
	IPReservationContainerPath = "/ip/reservation/"
	IPReservataionResourcePath = "/ip/reservation"
)

// IPReservations obtains an IPReservationsClient which can be used to access to the
// IP Reservations functions of the Compute API
func (c *Client) IPReservations() *IPReservationsClient {
	return &IPReservationsClient{
		ResourceClient: &ResourceClient{
			Client:              c,
			ResourceDescription: IPReservationDesc,
			ContainerPath:       IPReservationContainerPath,
			ResourceRootPath:    IPReservataionResourcePath,
		}}
}

type IPReservationPool string

const (
	PublicReservationPool IPReservationPool = "/oracle/public/ippool"
)

// IPReservationInput describes an existing IP reservation.
type IPReservation struct {
	// Shows the default account for your identity domain.
	Account string `json:"account"`
	// Public IP address.
	IP string `json:"ip"`
	// The three-part name of the IP Reservation (/Compute-identity_domain/user/object).
	Name string `json:"name"`
	// Pool of public IP addresses
	ParentPool IPReservationPool `json:"parentpool"`
	// Is the IP Reservation Persistent (i.e. static) or not (i.e. Dynamic)?
	Permanent bool `json:"permanent"`
	// A comma-separated list of strings which helps you to identify IP reservation.
	Tags []string `json:"tags"`
	// Uniform Resource Identifier
	Uri string `json:"uri"`
	// Is the IP reservation associated with an instance?
	Used bool `json:"used"`
}

// CreateIPReservationInput defines an IP reservation to be created.
type CreateIPReservationInput struct {
	// The name of the object
	// If you don't specify a name for this object, then the name is generated automatically.
	// Object names can contain only alphanumeric characters, hyphens, underscores, and periods.
	// Object names are case-sensitive.
	// Optional
	Name string `json:"name"`
	// Pool of public IP addresses. This must be set to `ippool`
	// Required
	ParentPool IPReservationPool `json:"parentpool"`
	// Is the IP Reservation Persistent (i.e. static) or not (i.e. Dynamic)?
	// Required
	Permanent bool `json:"permanent"`
	// A comma-separated list of strings which helps you to identify IP reservations.
	// Optional
	Tags []string `json:"tags"`
}

// CreateIPReservation creates a new IP reservation with the given parentpool, tags and permanent flag.
func (c *IPReservationsClient) CreateIPReservation(input *CreateIPReservationInput) (*IPReservation, error) {
	var ipInput IPReservation

	input.Name = c.getQualifiedName(input.Name)
	if err := c.createResource(input, &ipInput); err != nil {
		return nil, err
	}

	return c.success(&ipInput)
}

// GetIPReservationInput defines an IP Reservation to get
type GetIPReservationInput struct {
	// The name of the IP Reservation
	// Required
	Name string
}

// GetIPReservation retrieves the IP reservation with the given name.
func (c *IPReservationsClient) GetIPReservation(input *GetIPReservationInput) (*IPReservation, error) {
	var ipInput IPReservation

	input.Name = c.getQualifiedName(input.Name)
	if err := c.getResource(input.Name, &ipInput); err != nil {
		return nil, err
	}

	return c.success(&ipInput)
}

// UpdateIPReservationInput defines an IP Reservation to be updated
type UpdateIPReservationInput struct {
	// The name of the object
	// If you don't specify a name for this object, then the name is generated automatically.
	// Object names can contain only alphanumeric characters, hyphens, underscores, and periods.
	// Object names are case-sensitive.
	// Required
	Name string `json:"name"`
	// Pool of public IP addresses.
	// Required
	ParentPool IPReservationPool `json:"parentpool"`
	// Is the IP Reservation Persistent (i.e. static) or not (i.e. Dynamic)?
	// Required
	Permanent bool `json:"permanent"`
	// A comma-separated list of strings which helps you to identify IP reservations.
	// Optional
	Tags []string `json:"tags"`
}

// UpdateIPReservation updates the IP reservation.
func (c *IPReservationsClient) UpdateIPReservation(input *UpdateIPReservationInput) (*IPReservation, error) {
	var updateOutput IPReservation
	input.Name = c.getQualifiedName(input.Name)
	if err := c.updateResource(input.Name, input, &updateOutput); err != nil {
		return nil, err
	}
	return c.success(&updateOutput)
}

// DeleteIPReservationInput defines an IP Reservation to delete
type DeleteIPReservationInput struct {
	// The name of the IP Reservation
	// Required
	Name string
}

// DeleteIPReservation deletes the IP reservation with the given name.
func (c *IPReservationsClient) DeleteIPReservation(input *DeleteIPReservationInput) error {
	input.Name = c.getQualifiedName(input.Name)
	return c.deleteResource(input.Name)
}

func (c *IPReservationsClient) success(result *IPReservation) (*IPReservation, error) {
	c.unqualify(&result.Name)
	return result, nil
}
