package compute

import (
	"fmt"
	"strings"
)

// IPAssociationsClient is a client for the IP Association functions of the Compute API.
type IPAssociationsClient struct {
	*ResourceClient
}

// IPAssociations obtains a IPAssociationsClient which can be used to access to the
// IP Association functions of the Compute API
func (c *Client) IPAssociations() *IPAssociationsClient {
	return &IPAssociationsClient{
		ResourceClient: &ResourceClient{
			Client:              c,
			ResourceDescription: "ip association",
			ContainerPath:       "/ip/association/",
			ResourceRootPath:    "/ip/association",
		}}
}

// IPAssociationInfo describes an existing IP association.
type IPAssociationInfo struct {
	// TODO: it'd probably make sense to expose the `ip` field here too?

	// The three-part name of the object (/Compute-identity_domain/user/object).
	Name string `json:"name"`

	// The three-part name of the IP reservation object in the format (/Compute-identity_domain/user/object).
	// An IP reservation is a public IP address which is attached to an Oracle Compute Cloud Service instance that requires access to or from the Internet.
	Reservation string `json:"reservation"`

	// The type of IP Address to associate with this instance
	// for a Dynamic IP address specify `ippool:/oracle/public/ippool`.
	// for a Static IP address specify the three part name of the existing IP reservation
	ParentPool string `json:"parentpool"`

	// Uniform Resource Identifier for the IP Association
	URI string `json:"uri"`

	// The three-part name of a vcable ID of an instance that is associated with the IP reservation.
	VCable string `json:"vcable"`
}

type CreateIPAssociationInput struct {
	// The type of IP Address to associate with this instance
	// for a Dynamic IP address specify `ippool:/oracle/public/ippool`.
	// for a Static IP address specify the three part name of the existing IP reservation
	// Required
	ParentPool string `json:"parentpool"`

	// The three-part name of the vcable ID of the instance that you want to associate with an IP address. The three-part name is in the format: /Compute-identity_domain/user/object.
	// Required
	VCable string `json:"vcable"`
}

// CreateIPAssociation creates a new IP association with the supplied vcable and parentpool.
func (c *IPAssociationsClient) CreateIPAssociation(input *CreateIPAssociationInput) (*IPAssociationInfo, error) {
	input.VCable = c.getQualifiedName(input.VCable)
	input.ParentPool = c.getQualifiedParentPoolName(input.ParentPool)
	var assocInfo IPAssociationInfo
	if err := c.createResource(input, &assocInfo); err != nil {
		return nil, err
	}

	return c.success(&assocInfo)
}

type GetIPAssociationInput struct {
	// The three-part name of the IP Association
	// Required.
	Name string `json:"name"`
}

// GetIPAssociation retrieves the IP association with the given name.
func (c *IPAssociationsClient) GetIPAssociation(input *GetIPAssociationInput) (*IPAssociationInfo, error) {
	var assocInfo IPAssociationInfo
	if err := c.getResource(input.Name, &assocInfo); err != nil {
		return nil, err
	}

	return c.success(&assocInfo)
}

type DeleteIPAssociationInput struct {
	// The three-part name of the IP Association
	// Required.
	Name string `json:"name"`
}

// DeleteIPAssociation deletes the IP association with the given name.
func (c *IPAssociationsClient) DeleteIPAssociation(input *DeleteIPAssociationInput) error {
	return c.deleteResource(input.Name)
}

func (c *IPAssociationsClient) getQualifiedParentPoolName(parentpool string) string {
	parts := strings.Split(parentpool, ":")
	pooltype := parts[0]
	name := parts[1]
	return fmt.Sprintf("%s:%s", pooltype, c.getQualifiedName(name))
}

func (c *IPAssociationsClient) unqualifyParentPoolName(parentpool *string) {
	parts := strings.Split(*parentpool, ":")
	pooltype := parts[0]
	name := parts[1]
	*parentpool = fmt.Sprintf("%s:%s", pooltype, c.getUnqualifiedName(name))
}

// Unqualifies identifiers
func (c *IPAssociationsClient) success(assocInfo *IPAssociationInfo) (*IPAssociationInfo, error) {
	c.unqualify(&assocInfo.Name, &assocInfo.VCable)
	c.unqualifyParentPoolName(&assocInfo.ParentPool)
	return assocInfo, nil
}
