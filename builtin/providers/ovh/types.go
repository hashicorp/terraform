package ovh

import (
	"fmt"
	"time"
)

// Opts
type PublicCloudPrivateNetworkCreateOpts struct {
	ProjectId string   `json:"serviceName"`
	VlanId    int      `json:"vlanId"`
	Name      string   `json:"name"`
	Regions   []string `json:"regions"`
}

func (p *PublicCloudPrivateNetworkCreateOpts) String() string {
	return fmt.Sprintf("projectId: %s, vlanId:%d, name: %s, regions: %s", p.ProjectId, p.VlanId, p.Name, p.Regions)
}

// Opts
type PublicCloudPrivateNetworkUpdateOpts struct {
	Name string `json:"name"`
}

type PublicCloudPrivateNetworkRegion struct {
	Status string `json:"status"`
	Region string `json:"region"`
}

func (p *PublicCloudPrivateNetworkRegion) String() string {
	return fmt.Sprintf("Status:%s, Region: %s", p.Status, p.Region)
}

type PublicCloudPrivateNetworkResponse struct {
	Id      string                             `json:"id"`
	Status  string                             `json:"status"`
	Vlanid  int                                `json:"vlanId"`
	Name    string                             `json:"name"`
	Type    string                             `json:"type"`
	Regions []*PublicCloudPrivateNetworkRegion `json:"regions"`
}

func (p *PublicCloudPrivateNetworkResponse) String() string {
	return fmt.Sprintf("Id: %s, Status: %s, Name: %s, Vlanid: %d, Type: %s, Regions: %s", p.Id, p.Status, p.Name, p.Vlanid, p.Type, p.Regions)
}

// Opts
type PublicCloudPrivateNetworksCreateOpts struct {
	ProjectId string `json:"serviceName"`
	NetworkId string `json:"networkId"`
	Dhcp      bool   `json:"dhcp"`
	NoGateway bool   `json:"noGateway"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Network   string `json:"network"`
	Region    string `json:"region"`
}

func (p *PublicCloudPrivateNetworksCreateOpts) String() string {
	return fmt.Sprintf("PCPNSCreateOpts[projectId: %s, networkId:%s, dchp: %v, noGateway: %v, network: %s, start: %s, end: %s, region: %s]",
		p.ProjectId, p.NetworkId, p.Dhcp, p.NoGateway, p.Network, p.Start, p.End, p.Region)
}

type IPPool struct {
	Network string `json:"network"`
	Region  string `json:"region"`
	Dhcp    bool   `json:"dhcp"`
	Start   string `json:"start"`
	End     string `json:"end"`
}

func (p *IPPool) String() string {
	return fmt.Sprintf("IPPool[Network: %s, Region: %s, Dhcp: %v, Start: %s, End: %s]", p.Network, p.Region, p.Dhcp, p.Start, p.End)
}

type PublicCloudPrivateNetworksResponse struct {
	Id        string    `json:"id"`
	GatewayIp string    `json:"gatewayIp"`
	Cidr      string    `json:"cidr"`
	IPPools   []*IPPool `json:"ipPools"`
}

func (p *PublicCloudPrivateNetworksResponse) String() string {
	return fmt.Sprintf("PCPNSResponse[Id: %s, GatewayIp: %s, Cidr: %s, IPPools: %s]", p.Id, p.GatewayIp, p.Cidr, p.IPPools)
}

// Opts
type PublicCloudUserCreateOpts struct {
	ProjectId   string `json:"serviceName"`
	Description string `json:"description"`
}

func (p *PublicCloudUserCreateOpts) String() string {
	return fmt.Sprintf("UserOpts[projectId: %s, description:%s]", p.ProjectId, p.Description)
}

type PublicCloudUserResponse struct {
	Id           int    `json:"id"`
	Username     string `json:"username"`
	Status       string `json:"status"`
	Description  string `json:"description"`
	Password     string `json:"password"`
	CreationDate string `json:"creationDate"`
}

func (p *PublicCloudUserResponse) String() string {
	return fmt.Sprintf("UserResponse[Id: %v, Username: %s, Status: %s, Description: %s, CreationDate: %s]", p.Id, p.Username, p.Status, p.Description, p.CreationDate)
}

type PublicCloudUserOpenstackRC struct {
	Content string `json:"content"`
}

// Opts
type VRackAttachOpts struct {
	Project string `json:"project"`
}

// Task Opts
type TaskOpts struct {
	ServiceName string `json:"serviceName"`
	TaskId      string `json:"taskId"`
}

type VRackAttachTaskResponse struct {
	Id           int       `json:"id"`
	Function     string    `json:"function"`
	TargetDomain string    `json:"targetDomain"`
	Status       string    `json:"status"`
	ServiceName  string    `json:"serviceName"`
	OrderId      int       `json:"orderId"`
	LastUpdate   time.Time `json:"lastUpdate"`
	TodoDate     time.Time `json:"TodoDate"`
}

type PublicCloudRegionResponse struct {
	ContinentCode      string                             `json:"continentCode"`
	DatacenterLocation string                             `json:"datacenterLocation"`
	Name               string                             `json:"name"`
	Services           []PublicCloudServiceStatusResponse `json:"services"`
}

func (r *PublicCloudRegionResponse) String() string {
	return fmt.Sprintf("Region: %s, Services: %s", r.Name, r.Services)
}

type PublicCloudServiceStatusResponse struct {
	Status string `json:"status"`
	Name   string `json:"name"`
}

func (s *PublicCloudServiceStatusResponse) String() string {
	return fmt.Sprintf("%s: %s", s.Name, s.Status)
}
