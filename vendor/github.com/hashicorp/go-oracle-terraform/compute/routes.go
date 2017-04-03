package compute

const (
	RoutesDescription   = "IP Network Route"
	RoutesContainerPath = "/network/v1/route/"
	RoutesResourcePath  = "/network/v1/route"
)

type RoutesClient struct {
	ResourceClient
}

func (c *Client) Routes() *RoutesClient {
	return &RoutesClient{
		ResourceClient: ResourceClient{
			Client:              c,
			ResourceDescription: RoutesDescription,
			ContainerPath:       RoutesContainerPath,
			ResourceRootPath:    RoutesResourcePath,
		},
	}
}

type RouteInfo struct {
	// Admin distance associated with this route
	AdminDistance int `json:"adminDistance"`
	// Description of the route
	Description string `json:"description"`
	// CIDR IPv4 Prefix associated with this route
	IPAddressPrefix string `json:"ipAddressPrefix"`
	// Name of the route
	Name string `json:"name"`
	// Name of the VNIC set associated with the route
	NextHopVnicSet string `json:"nextHopVnicSet"`
	// Slice of Tags associated with the route
	Tags []string `json:"tags,omitempty"`
	// Uniform resource identifier associated with the route
	Uri string `json:"uri"`
}

type CreateRouteInput struct {
	// Specify 0,1, or 2 as the route's administrative distance.
	// If you do not specify a value, the default value is 0.
	// The same prefix can be used in multiple routes. In this case, packets are routed over all the matching
	// routes with the lowest administrative distance.
	// In the case multiple routes with the same lowest administrative distance match,
	// routing occurs over all these routes using ECMP.
	// Optional
	AdminDistance int `json:"adminDistance"`
	// Description of the route
	// Optional
	Description string `json:"description"`
	// The IPv4 address prefix in CIDR format, of the external network (external to the vNIC set)
	// from which you want to route traffic
	// Required
	IPAddressPrefix string `json:"ipAddressPrefix"`
	// Name of the route.
	// Names can only contain alphanumeric, underscore, dash, and period characters. Case-sensitive
	// Required
	Name string `json:"name"`
	// Name of the virtual NIC set to route matching packets to.
	// Routed flows are load-balanced among all the virtual NICs in the virtual NIC set
	// Required
	NextHopVnicSet string `json:"nextHopVnicSet"`
	// Slice of tags to be associated with the route
	// Optional
	Tags []string `json:"tags,omitempty"`
}

func (c *RoutesClient) CreateRoute(input *CreateRouteInput) (*RouteInfo, error) {
	input.Name = c.getQualifiedName(input.Name)
	input.NextHopVnicSet = c.getQualifiedName(input.NextHopVnicSet)

	var routeInfo RouteInfo
	if err := c.createResource(&input, &routeInfo); err != nil {
		return nil, err
	}

	return c.success(&routeInfo)
}

type GetRouteInput struct {
	// Name of the Route to query for. Case-sensitive
	// Required
	Name string `json:"name"`
}

func (c *RoutesClient) GetRoute(input *GetRouteInput) (*RouteInfo, error) {
	input.Name = c.getQualifiedName(input.Name)

	var routeInfo RouteInfo
	if err := c.getResource(input.Name, &routeInfo); err != nil {
		return nil, err
	}
	return c.success(&routeInfo)
}

type UpdateRouteInput struct {
	// Specify 0,1, or 2 as the route's administrative distance.
	// If you do not specify a value, the default value is 0.
	// The same prefix can be used in multiple routes. In this case, packets are routed over all the matching
	// routes with the lowest administrative distance.
	// In the case multiple routes with the same lowest administrative distance match,
	// routing occurs over all these routes using ECMP.
	// Optional
	AdminDistance int `json:"adminDistance"`
	// Description of the route
	// Optional
	Description string `json:"description"`
	// The IPv4 address prefix in CIDR format, of the external network (external to the vNIC set)
	// from which you want to route traffic
	// Required
	IPAddressPrefix string `json:"ipAddressPrefix"`
	// Name of the route.
	// Names can only contain alphanumeric, underscore, dash, and period characters. Case-sensitive
	// Required
	Name string `json:"name"`
	// Name of the virtual NIC set to route matching packets to.
	// Routed flows are load-balanced among all the virtual NICs in the virtual NIC set
	// Required
	NextHopVnicSet string `json:"nextHopVnicSet"`
	// Slice of tags to be associated with the route
	// Optional
	Tags []string `json:"tags"`
}

func (c *RoutesClient) UpdateRoute(input *UpdateRouteInput) (*RouteInfo, error) {
	input.Name = c.getQualifiedName(input.Name)
	input.NextHopVnicSet = c.getQualifiedName(input.NextHopVnicSet)

	var routeInfo RouteInfo
	if err := c.updateResource(input.Name, &input, &routeInfo); err != nil {
		return nil, err
	}

	return c.success(&routeInfo)
}

type DeleteRouteInput struct {
	// Name of the Route to delete. Case-sensitive
	// Required
	Name string `json:"name"`
}

func (c *RoutesClient) DeleteRoute(input *DeleteRouteInput) error {
	return c.deleteResource(input.Name)
}

func (c *RoutesClient) success(info *RouteInfo) (*RouteInfo, error) {
	c.unqualify(&info.Name)
	c.unqualify(&info.NextHopVnicSet)
	return info, nil
}
