// Copyright (C) 2015 Scaleway. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE.md file.

// Interact with Scaleway API

// Package api contains client and functions to interact with Scaleway API
package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

	"golang.org/x/sync/errgroup"
)

// Default values
var (
	AccountAPI     = "https://account.scaleway.com/"
	MetadataAPI    = "http://169.254.42.42/"
	MarketplaceAPI = "https://api-marketplace.scaleway.com"
	ComputeAPIPar1 = "https://cp-par1.scaleway.com/"
	ComputeAPIAms1 = "https://cp-ams1.scaleway.com"

	URLPublicDNS  = ".pub.cloud.scaleway.com"
	URLPrivateDNS = ".priv.cloud.scaleway.com"
)

func init() {
	if url := os.Getenv("SCW_ACCOUNT_API"); url != "" {
		AccountAPI = url
	}
	if url := os.Getenv("SCW_METADATA_API"); url != "" {
		MetadataAPI = url
	}
	if url := os.Getenv("SCW_MARKETPLACE_API"); url != "" {
		MarketplaceAPI = url
	}
}

const (
	perPage = 50
)

// ScalewayAPI is the interface used to communicate with the Scaleway API
type ScalewayAPI struct {
	// Organization is the identifier of the Scaleway organization
	Organization string

	// Token is the authentication token for the Scaleway organization
	Token string

	// Password is the authentication password
	password string

	userAgent string

	// Cache is used to quickly resolve identifiers from names
	Cache *ScalewayCache

	client     *http.Client
	verbose    bool
	computeAPI string

	Region string
	//
	Logger
}

// ScalewayAPIError represents a Scaleway API Error
type ScalewayAPIError struct {
	// Message is a human-friendly error message
	APIMessage string `json:"message,omitempty"`

	// Type is a string code that defines the kind of error
	Type string `json:"type,omitempty"`

	// Fields contains detail about validation error
	Fields map[string][]string `json:"fields,omitempty"`

	// StatusCode is the HTTP status code received
	StatusCode int `json:"-"`

	// Message
	Message string `json:"-"`
}

// Error returns a string representing the error
func (e ScalewayAPIError) Error() string {
	var b bytes.Buffer

	fmt.Fprintf(&b, "StatusCode: %v, ", e.StatusCode)
	fmt.Fprintf(&b, "Type: %v, ", e.Type)
	fmt.Fprintf(&b, "APIMessage: \x1b[31m%v\x1b[0m", e.APIMessage)
	if len(e.Fields) > 0 {
		fmt.Fprintf(&b, ", Details: %v", e.Fields)
	}
	return b.String()
}

// HideAPICredentials removes API credentials from a string
func (s *ScalewayAPI) HideAPICredentials(input string) string {
	output := input
	if s.Token != "" {
		output = strings.Replace(output, s.Token, "00000000-0000-4000-8000-000000000000", -1)
	}
	if s.Organization != "" {
		output = strings.Replace(output, s.Organization, "00000000-0000-5000-9000-000000000000", -1)
	}
	if s.password != "" {
		output = strings.Replace(output, s.password, "XX-XX-XX-XX", -1)
	}
	return output
}

// ScalewayIPAddress represents a Scaleway IP address
type ScalewayIPAddress struct {
	// Identifier is a unique identifier for the IP address
	Identifier string `json:"id,omitempty"`

	// IP is an IPv4 address
	IP string `json:"address,omitempty"`

	// Dynamic is a flag that defines an IP that change on each reboot
	Dynamic *bool `json:"dynamic,omitempty"`
}

// ScalewayVolume represents a Scaleway Volume
type ScalewayVolume struct {
	// Identifier is a unique identifier for the volume
	Identifier string `json:"id,omitempty"`

	// Size is the allocated size of the volume
	Size uint64 `json:"size,omitempty"`

	// CreationDate is the creation date of the volume
	CreationDate string `json:"creation_date,omitempty"`

	// ModificationDate is the date of the last modification of the volume
	ModificationDate string `json:"modification_date,omitempty"`

	// Organization is the organization owning the volume
	Organization string `json:"organization,omitempty"`

	// Name is the name of the volume
	Name string `json:"name,omitempty"`

	// Server is the server using this image
	Server *struct {
		Identifier string `json:"id,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"server,omitempty"`

	// VolumeType is a Scaleway identifier for the kind of volume (default: l_ssd)
	VolumeType string `json:"volume_type,omitempty"`

	// ExportURI represents the url used by initrd/scripts to attach the volume
	ExportURI string `json:"export_uri,omitempty"`
}

// ScalewayOneVolume represents the response of a GET /volumes/UUID API call
type ScalewayOneVolume struct {
	Volume ScalewayVolume `json:"volume,omitempty"`
}

// ScalewayVolumes represents a group of Scaleway volumes
type ScalewayVolumes struct {
	// Volumes holds scaleway volumes of the response
	Volumes []ScalewayVolume `json:"volumes,omitempty"`
}

// ScalewayVolumeDefinition represents a Scaleway volume definition
type ScalewayVolumeDefinition struct {
	// Name is the user-defined name of the volume
	Name string `json:"name"`

	// Image is the image used by the volume
	Size uint64 `json:"size"`

	// Bootscript is the bootscript used by the volume
	Type string `json:"volume_type"`

	// Organization is the owner of the volume
	Organization string `json:"organization"`
}

// ScalewayVolumePutDefinition represents a Scaleway volume with nullable fields (for PUT)
type ScalewayVolumePutDefinition struct {
	Identifier       *string `json:"id,omitempty"`
	Size             *uint64 `json:"size,omitempty"`
	CreationDate     *string `json:"creation_date,omitempty"`
	ModificationDate *string `json:"modification_date,omitempty"`
	Organization     *string `json:"organization,omitempty"`
	Name             *string `json:"name,omitempty"`
	Server           struct {
		Identifier *string `json:"id,omitempty"`
		Name       *string `json:"name,omitempty"`
	} `json:"server,omitempty"`
	VolumeType *string `json:"volume_type,omitempty"`
	ExportURI  *string `json:"export_uri,omitempty"`
}

// ScalewayImage represents a Scaleway Image
type ScalewayImage struct {
	// Identifier is a unique identifier for the image
	Identifier string `json:"id,omitempty"`

	// Name is a user-defined name for the image
	Name string `json:"name,omitempty"`

	// CreationDate is the creation date of the image
	CreationDate string `json:"creation_date,omitempty"`

	// ModificationDate is the date of the last modification of the image
	ModificationDate string `json:"modification_date,omitempty"`

	// RootVolume is the root volume bound to the image
	RootVolume ScalewayVolume `json:"root_volume,omitempty"`

	// Public is true for public images and false for user images
	Public bool `json:"public,omitempty"`

	// Bootscript is the bootscript bound to the image
	DefaultBootscript *ScalewayBootscript `json:"default_bootscript,omitempty"`

	// Organization is the owner of the image
	Organization string `json:"organization,omitempty"`

	// Arch is the architecture target of the image
	Arch string `json:"arch,omitempty"`

	// FIXME: extra_volumes
}

// ScalewayImageIdentifier represents a Scaleway Image Identifier
type ScalewayImageIdentifier struct {
	Identifier string
	Arch       string
	Region     string
	Owner      string
}

// ScalewayOneImage represents the response of a GET /images/UUID API call
type ScalewayOneImage struct {
	Image ScalewayImage `json:"image,omitempty"`
}

// ScalewayImages represents a group of Scaleway images
type ScalewayImages struct {
	// Images holds scaleway images of the response
	Images []ScalewayImage `json:"images,omitempty"`
}

// ScalewaySnapshot represents a Scaleway Snapshot
type ScalewaySnapshot struct {
	// Identifier is a unique identifier for the snapshot
	Identifier string `json:"id,omitempty"`

	// Name is a user-defined name for the snapshot
	Name string `json:"name,omitempty"`

	// CreationDate is the creation date of the snapshot
	CreationDate string `json:"creation_date,omitempty"`

	// ModificationDate is the date of the last modification of the snapshot
	ModificationDate string `json:"modification_date,omitempty"`

	// Size is the allocated size of the volume
	Size uint64 `json:"size,omitempty"`

	// Organization is the owner of the snapshot
	Organization string `json:"organization"`

	// State is the current state of the snapshot
	State string `json:"state"`

	// VolumeType is the kind of volume behind the snapshot
	VolumeType string `json:"volume_type"`

	// BaseVolume is the volume from which the snapshot inherits
	BaseVolume ScalewayVolume `json:"base_volume,omitempty"`
}

// ScalewayOneSnapshot represents the response of a GET /snapshots/UUID API call
type ScalewayOneSnapshot struct {
	Snapshot ScalewaySnapshot `json:"snapshot,omitempty"`
}

// ScalewaySnapshots represents a group of Scaleway snapshots
type ScalewaySnapshots struct {
	// Snapshots holds scaleway snapshots of the response
	Snapshots []ScalewaySnapshot `json:"snapshots,omitempty"`
}

// ScalewayBootscript represents a Scaleway Bootscript
type ScalewayBootscript struct {
	Bootcmdargs string `json:"bootcmdargs,omitempty"`
	Dtb         string `json:"dtb,omitempty"`
	Initrd      string `json:"initrd,omitempty"`
	Kernel      string `json:"kernel,omitempty"`

	// Arch is the architecture target of the bootscript
	Arch string `json:"architecture,omitempty"`

	// Identifier is a unique identifier for the bootscript
	Identifier string `json:"id,omitempty"`

	// Organization is the owner of the bootscript
	Organization string `json:"organization,omitempty"`

	// Name is a user-defined name for the bootscript
	Title string `json:"title,omitempty"`

	// Public is true for public bootscripts and false for user bootscripts
	Public bool `json:"public,omitempty"`

	Default bool `json:"default,omitempty"`
}

// ScalewayOneBootscript represents the response of a GET /bootscripts/UUID API call
type ScalewayOneBootscript struct {
	Bootscript ScalewayBootscript `json:"bootscript,omitempty"`
}

// ScalewayBootscripts represents a group of Scaleway bootscripts
type ScalewayBootscripts struct {
	// Bootscripts holds Scaleway bootscripts of the response
	Bootscripts []ScalewayBootscript `json:"bootscripts,omitempty"`
}

// ScalewayTask represents a Scaleway Task
type ScalewayTask struct {
	// Identifier is a unique identifier for the task
	Identifier string `json:"id,omitempty"`

	// StartDate is the start date of the task
	StartDate string `json:"started_at,omitempty"`

	// TerminationDate is the termination date of the task
	TerminationDate string `json:"terminated_at,omitempty"`

	HrefFrom string `json:"href_from,omitempty"`

	Description string `json:"description,omitempty"`

	Status string `json:"status,omitempty"`

	Progress int `json:"progress,omitempty"`
}

// ScalewayOneTask represents the response of a GET /tasks/UUID API call
type ScalewayOneTask struct {
	Task ScalewayTask `json:"task,omitempty"`
}

// ScalewayTasks represents a group of Scaleway tasks
type ScalewayTasks struct {
	// Tasks holds scaleway tasks of the response
	Tasks []ScalewayTask `json:"tasks,omitempty"`
}

// ScalewaySecurityGroupRule definition
type ScalewaySecurityGroupRule struct {
	Direction    string `json:"direction"`
	Protocol     string `json:"protocol"`
	IPRange      string `json:"ip_range"`
	DestPortFrom int    `json:"dest_port_from,omitempty"`
	Action       string `json:"action"`
	Position     int    `json:"position"`
	DestPortTo   string `json:"dest_port_to"`
	Editable     bool   `json:"editable"`
	ID           string `json:"id"`
}

// ScalewayGetSecurityGroupRules represents the response of a GET /security_group/{groupID}/rules
type ScalewayGetSecurityGroupRules struct {
	Rules []ScalewaySecurityGroupRule `json:"rules"`
}

// ScalewayGetSecurityGroupRule represents the response of a GET /security_group/{groupID}/rules/{ruleID}
type ScalewayGetSecurityGroupRule struct {
	Rules ScalewaySecurityGroupRule `json:"rule"`
}

// ScalewayNewSecurityGroupRule definition POST/PUT request /security_group/{groupID}
type ScalewayNewSecurityGroupRule struct {
	Action       string `json:"action"`
	Direction    string `json:"direction"`
	IPRange      string `json:"ip_range"`
	Protocol     string `json:"protocol"`
	DestPortFrom int    `json:"dest_port_from,omitempty"`
}

// ScalewaySecurityGroups definition
type ScalewaySecurityGroups struct {
	Description           string                  `json:"description"`
	ID                    string                  `json:"id"`
	Organization          string                  `json:"organization"`
	Name                  string                  `json:"name"`
	Servers               []ScalewaySecurityGroup `json:"servers"`
	EnableDefaultSecurity bool                    `json:"enable_default_security"`
	OrganizationDefault   bool                    `json:"organization_default"`
}

// ScalewayGetSecurityGroups represents the response of a GET /security_groups/
type ScalewayGetSecurityGroups struct {
	SecurityGroups []ScalewaySecurityGroups `json:"security_groups"`
}

// ScalewayGetSecurityGroup represents the response of a GET /security_groups/{groupID}
type ScalewayGetSecurityGroup struct {
	SecurityGroups ScalewaySecurityGroups `json:"security_group"`
}

// ScalewayIPDefinition represents the IP's fields
type ScalewayIPDefinition struct {
	Organization string  `json:"organization"`
	Reverse      *string `json:"reverse"`
	ID           string  `json:"id"`
	Server       *struct {
		Identifier string `json:"id,omitempty"`
		Name       string `json:"name,omitempty"`
	} `json:"server"`
	Address string `json:"address"`
}

// ScalewayGetIPS represents the response of a GET /ips/
type ScalewayGetIPS struct {
	IPS []ScalewayIPDefinition `json:"ips"`
}

// ScalewayGetIP represents the response of a GET /ips/{id_ip}
type ScalewayGetIP struct {
	IP ScalewayIPDefinition `json:"ip"`
}

// ScalewaySecurityGroup represents a Scaleway security group
type ScalewaySecurityGroup struct {
	// Identifier is a unique identifier for the security group
	Identifier string `json:"id,omitempty"`

	// Name is the user-defined name of the security group
	Name string `json:"name,omitempty"`
}

// ScalewayNewSecurityGroup definition POST request /security_groups
type ScalewayNewSecurityGroup struct {
	Organization string `json:"organization"`
	Name         string `json:"name"`
	Description  string `json:"description"`
}

// ScalewayUpdateSecurityGroup definition PUT request /security_groups
type ScalewayUpdateSecurityGroup struct {
	Organization        string `json:"organization"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	OrganizationDefault bool   `json:"organization_default"`
}

// ScalewayServer represents a Scaleway server
type ScalewayServer struct {
	// Arch is the architecture target of the server
	Arch string `json:"arch,omitempty"`

	// Identifier is a unique identifier for the server
	Identifier string `json:"id,omitempty"`

	// Name is the user-defined name of the server
	Name string `json:"name,omitempty"`

	// CreationDate is the creation date of the server
	CreationDate string `json:"creation_date,omitempty"`

	// ModificationDate is the date of the last modification of the server
	ModificationDate string `json:"modification_date,omitempty"`

	// Image is the image used by the server
	Image ScalewayImage `json:"image,omitempty"`

	// DynamicIPRequired is a flag that defines a server with a dynamic ip address attached
	DynamicIPRequired *bool `json:"dynamic_ip_required,omitempty"`

	// PublicIP is the public IP address bound to the server
	PublicAddress ScalewayIPAddress `json:"public_ip,omitempty"`

	// State is the current status of the server
	State string `json:"state,omitempty"`

	// StateDetail is the detailed status of the server
	StateDetail string `json:"state_detail,omitempty"`

	// PrivateIP represents the private IPV4 attached to the server (changes on each boot)
	PrivateIP string `json:"private_ip,omitempty"`

	// Bootscript is the unique identifier of the selected bootscript
	Bootscript *ScalewayBootscript `json:"bootscript,omitempty"`

	// Hostname represents the ServerName in a format compatible with unix's hostname
	Hostname string `json:"hostname,omitempty"`

	// Tags represents user-defined tags
	Tags []string `json:"tags,omitempty"`

	// Volumes are the attached volumes
	Volumes map[string]ScalewayVolume `json:"volumes,omitempty"`

	// SecurityGroup is the selected security group object
	SecurityGroup ScalewaySecurityGroup `json:"security_group,omitempty"`

	// Organization is the owner of the server
	Organization string `json:"organization,omitempty"`

	// CommercialType is the commercial type of the server (i.e: C1, C2[SML], VC1S)
	CommercialType string `json:"commercial_type,omitempty"`

	// Location of the server
	Location struct {
		Platform   string `json:"platform_id,omitempty"`
		Chassis    string `json:"chassis_id,omitempty"`
		Cluster    string `json:"cluster_id,omitempty"`
		Hypervisor string `json:"hypervisor_id,omitempty"`
		Blade      string `json:"blade_id,omitempty"`
		Node       string `json:"node_id,omitempty"`
		ZoneID     string `json:"zone_id,omitempty"`
	} `json:"location,omitempty"`

	IPV6 *ScalewayIPV6Definition `json:"ipv6,omitempty"`

	EnableIPV6 bool `json:"enable_ipv6,omitempty"`

	// This fields are not returned by the API, we generate it
	DNSPublic  string `json:"dns_public,omitempty"`
	DNSPrivate string `json:"dns_private,omitempty"`
}

// ScalewayIPV6Definition represents a Scaleway ipv6
type ScalewayIPV6Definition struct {
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway"`
	Address string `json:"address"`
}

// ScalewayServerPatchDefinition represents a Scaleway server with nullable fields (for PATCH)
type ScalewayServerPatchDefinition struct {
	Arch              *string                    `json:"arch,omitempty"`
	Name              *string                    `json:"name,omitempty"`
	CreationDate      *string                    `json:"creation_date,omitempty"`
	ModificationDate  *string                    `json:"modification_date,omitempty"`
	Image             *ScalewayImage             `json:"image,omitempty"`
	DynamicIPRequired *bool                      `json:"dynamic_ip_required,omitempty"`
	PublicAddress     *ScalewayIPAddress         `json:"public_ip,omitempty"`
	State             *string                    `json:"state,omitempty"`
	StateDetail       *string                    `json:"state_detail,omitempty"`
	PrivateIP         *string                    `json:"private_ip,omitempty"`
	Bootscript        *string                    `json:"bootscript,omitempty"`
	Hostname          *string                    `json:"hostname,omitempty"`
	Volumes           *map[string]ScalewayVolume `json:"volumes,omitempty"`
	SecurityGroup     *ScalewaySecurityGroup     `json:"security_group,omitempty"`
	Organization      *string                    `json:"organization,omitempty"`
	Tags              *[]string                  `json:"tags,omitempty"`
	IPV6              *ScalewayIPV6Definition    `json:"ipv6,omitempty"`
	EnableIPV6        *bool                      `json:"enable_ipv6,omitempty"`
}

// ScalewayServerDefinition represents a Scaleway server with image definition
type ScalewayServerDefinition struct {
	// Name is the user-defined name of the server
	Name string `json:"name"`

	// Image is the image used by the server
	Image *string `json:"image,omitempty"`

	// Volumes are the attached volumes
	Volumes map[string]string `json:"volumes,omitempty"`

	// DynamicIPRequired is a flag that defines a server with a dynamic ip address attached
	DynamicIPRequired *bool `json:"dynamic_ip_required,omitempty"`

	// Bootscript is the bootscript used by the server
	Bootscript *string `json:"bootscript"`

	// Tags are the metadata tags attached to the server
	Tags []string `json:"tags,omitempty"`

	// Organization is the owner of the server
	Organization string `json:"organization"`

	// CommercialType is the commercial type of the server (i.e: C1, C2[SML], VC1S)
	CommercialType string `json:"commercial_type"`

	PublicIP string `json:"public_ip,omitempty"`

	EnableIPV6 bool `json:"enable_ipv6,omitempty"`

	SecurityGroup string `json:"security_group,omitempty"`
}

// ScalewayOneServer represents the response of a GET /servers/UUID API call
type ScalewayOneServer struct {
	Server ScalewayServer `json:"server,omitempty"`
}

// ScalewayServers represents a group of Scaleway servers
type ScalewayServers struct {
	// Servers holds scaleway servers of the response
	Servers []ScalewayServer `json:"servers,omitempty"`
}

// ScalewayServerAction represents an action to perform on a Scaleway server
type ScalewayServerAction struct {
	// Action is the name of the action to trigger
	Action string `json:"action,omitempty"`
}

// ScalewaySnapshotDefinition represents a Scaleway snapshot definition
type ScalewaySnapshotDefinition struct {
	VolumeIDentifier string `json:"volume_id"`
	Name             string `json:"name,omitempty"`
	Organization     string `json:"organization"`
}

// ScalewayImageDefinition represents a Scaleway image definition
type ScalewayImageDefinition struct {
	SnapshotIDentifier string  `json:"root_volume"`
	Name               string  `json:"name,omitempty"`
	Organization       string  `json:"organization"`
	Arch               string  `json:"arch"`
	DefaultBootscript  *string `json:"default_bootscript,omitempty"`
}

// ScalewayRoleDefinition represents a Scaleway Token UserId Role
type ScalewayRoleDefinition struct {
	Organization ScalewayOrganizationDefinition `json:"organization,omitempty"`
	Role         string                         `json:"role,omitempty"`
}

// ScalewayTokenDefinition represents a Scaleway Token
type ScalewayTokenDefinition struct {
	UserID             string                 `json:"user_id"`
	Description        string                 `json:"description,omitempty"`
	Roles              ScalewayRoleDefinition `json:"roles"`
	Expires            string                 `json:"expires"`
	InheritsUsersPerms bool                   `json:"inherits_user_perms"`
	ID                 string                 `json:"id"`
}

// ScalewayTokensDefinition represents a Scaleway Tokens
type ScalewayTokensDefinition struct {
	Token ScalewayTokenDefinition `json:"token"`
}

// ScalewayGetTokens represents a list of Scaleway Tokens
type ScalewayGetTokens struct {
	Tokens []ScalewayTokenDefinition `json:"tokens"`
}

// ScalewayContainerData represents a Scaleway container data (S3)
type ScalewayContainerData struct {
	LastModified string `json:"last_modified"`
	Name         string `json:"name"`
	Size         string `json:"size"`
}

// ScalewayGetContainerDatas represents a list of Scaleway containers data (S3)
type ScalewayGetContainerDatas struct {
	Container []ScalewayContainerData `json:"container"`
}

// ScalewayContainer represents a Scaleway container (S3)
type ScalewayContainer struct {
	ScalewayOrganizationDefinition `json:"organization"`
	Name                           string `json:"name"`
	Size                           string `json:"size"`
}

// ScalewayGetContainers represents a list of Scaleway containers (S3)
type ScalewayGetContainers struct {
	Containers []ScalewayContainer `json:"containers"`
}

// ScalewayConnectResponse represents the answer from POST /tokens
type ScalewayConnectResponse struct {
	Token ScalewayTokenDefinition `json:"token"`
}

// ScalewayConnect represents the data to connect
type ScalewayConnect struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Description string `json:"description"`
	Expires     bool   `json:"expires"`
}

// ScalewayOrganizationDefinition represents a Scaleway Organization
type ScalewayOrganizationDefinition struct {
	ID    string                   `json:"id"`
	Name  string                   `json:"name"`
	Users []ScalewayUserDefinition `json:"users"`
}

// ScalewayOrganizationsDefinition represents a Scaleway Organizations
type ScalewayOrganizationsDefinition struct {
	Organizations []ScalewayOrganizationDefinition `json:"organizations"`
}

// ScalewayUserDefinition represents a Scaleway User
type ScalewayUserDefinition struct {
	Email         string                           `json:"email"`
	Firstname     string                           `json:"firstname"`
	Fullname      string                           `json:"fullname"`
	ID            string                           `json:"id"`
	Lastname      string                           `json:"lastname"`
	Organizations []ScalewayOrganizationDefinition `json:"organizations"`
	Roles         []ScalewayRoleDefinition         `json:"roles"`
	SSHPublicKeys []ScalewayKeyDefinition          `json:"ssh_public_keys"`
}

// ScalewayUsersDefinition represents the response of a GET /user
type ScalewayUsersDefinition struct {
	User ScalewayUserDefinition `json:"user"`
}

// ScalewayKeyDefinition represents a key
type ScalewayKeyDefinition struct {
	Key         string `json:"key"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// ScalewayUserPatchSSHKeyDefinition represents a User Patch
type ScalewayUserPatchSSHKeyDefinition struct {
	SSHPublicKeys []ScalewayKeyDefinition `json:"ssh_public_keys"`
}

// ScalewayDashboardResp represents a dashboard received from the API
type ScalewayDashboardResp struct {
	Dashboard ScalewayDashboard
}

// ScalewayDashboard represents a dashboard
type ScalewayDashboard struct {
	VolumesCount        int `json:"volumes_count"`
	RunningServersCount int `json:"running_servers_count"`
	ImagesCount         int `json:"images_count"`
	SnapshotsCount      int `json:"snapshots_count"`
	ServersCount        int `json:"servers_count"`
	IPsCount            int `json:"ips_count"`
}

// ScalewayPermissions represents the response of GET /permissions
type ScalewayPermissions map[string]ScalewayPermCategory

// ScalewayPermCategory represents ScalewayPermissions's fields
type ScalewayPermCategory map[string][]string

// ScalewayPermissionDefinition represents the permissions
type ScalewayPermissionDefinition struct {
	Permissions ScalewayPermissions `json:"permissions"`
}

// ScalewayUserdatas represents the response of a GET /user_data
type ScalewayUserdatas struct {
	UserData []string `json:"user_data"`
}

// ScalewayQuota represents a map of quota (name, value)
type ScalewayQuota map[string]int

// ScalewayGetQuotas represents the response of GET /organizations/{orga_id}/quotas
type ScalewayGetQuotas struct {
	Quotas ScalewayQuota `json:"quotas"`
}

// ScalewayUserdata represents []byte
type ScalewayUserdata []byte

// FuncMap used for json inspection
var FuncMap = template.FuncMap{
	"json": func(v interface{}) string {
		a, _ := json.Marshal(v)
		return string(a)
	},
}

// MarketLocalImageDefinition represents localImage of marketplace version
type MarketLocalImageDefinition struct {
	Arch string `json:"arch"`
	ID   string `json:"id"`
	Zone string `json:"zone"`
}

// MarketLocalImages represents an array of local images
type MarketLocalImages struct {
	LocalImages []MarketLocalImageDefinition `json:"local_images"`
}

// MarketLocalImage represents local image
type MarketLocalImage struct {
	LocalImages MarketLocalImageDefinition `json:"local_image"`
}

// MarketVersionDefinition represents version of marketplace image
type MarketVersionDefinition struct {
	CreationDate string `json:"creation_date"`
	ID           string `json:"id"`
	Image        struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"image"`
	ModificationDate string `json:"modification_date"`
	Name             string `json:"name"`
	MarketLocalImages
}

// MarketVersions represents an array of marketplace image versions
type MarketVersions struct {
	Versions []MarketVersionDefinition `json:"versions"`
}

// MarketVersion represents version of marketplace image
type MarketVersion struct {
	Version MarketVersionDefinition `json:"version"`
}

// MarketImage represents MarketPlace image
type MarketImage struct {
	Categories           []string `json:"categories"`
	CreationDate         string   `json:"creation_date"`
	CurrentPublicVersion string   `json:"current_public_version"`
	Description          string   `json:"description"`
	ID                   string   `json:"id"`
	Logo                 string   `json:"logo"`
	ModificationDate     string   `json:"modification_date"`
	Name                 string   `json:"name"`
	Organization         struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"organization"`
	Public bool `json:"-"`
	MarketVersions
}

// MarketImages represents MarketPlace images
type MarketImages struct {
	Images []MarketImage `json:"images"`
}

// NewScalewayAPI creates a ready-to-use ScalewayAPI client
func NewScalewayAPI(organization, token, userAgent, region string, options ...func(*ScalewayAPI)) (*ScalewayAPI, error) {
	s := &ScalewayAPI{
		// exposed
		Organization: organization,
		Token:        token,
		Logger:       NewDefaultLogger(),

		// internal
		client:    &http.Client{},
		verbose:   os.Getenv("SCW_VERBOSE_API") != "",
		password:  "",
		userAgent: userAgent,
	}
	for _, option := range options {
		option(s)
	}
	cache, err := NewScalewayCache(func() { s.Logger.Debugf("Writing cache file to disk") })
	if err != nil {
		return nil, err
	}
	s.Cache = cache
	if os.Getenv("SCW_TLSVERIFY") == "0" {
		s.client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	switch region {
	case "par1", "":
		s.computeAPI = ComputeAPIPar1
	case "ams1":
		s.computeAPI = ComputeAPIAms1
	default:
		return nil, fmt.Errorf("%s isn't a valid region", region)
	}
	s.Region = region
	if url := os.Getenv("SCW_COMPUTE_API"); url != "" {
		s.computeAPI = url
	}
	return s, nil
}

// ClearCache clears the cache
func (s *ScalewayAPI) ClearCache() {
	s.Cache.Clear()
}

// Sync flushes out the cache to the disk
func (s *ScalewayAPI) Sync() {
	s.Cache.Save()
}

func (s *ScalewayAPI) response(method, uri string, content io.Reader) (resp *http.Response, err error) {
	var (
		req *http.Request
	)

	req, err = http.NewRequest(method, uri, content)
	if err != nil {
		err = fmt.Errorf("response %s %s", method, uri)
		return
	}
	req.Header.Set("X-Auth-Token", s.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", s.userAgent)
	s.LogHTTP(req)
	if s.verbose {
		dump, _ := httputil.DumpRequest(req, true)
		s.Debugf("%v", string(dump))
	} else {
		s.Debugf("[%s]: %v", method, uri)
	}
	resp, err = s.client.Do(req)
	return
}

// GetResponsePaginate fetchs all resources and returns an http.Response object for the requested resource
func (s *ScalewayAPI) GetResponsePaginate(apiURL, resource string, values url.Values) (*http.Response, error) {
	resp, err := s.response("HEAD", fmt.Sprintf("%s/%s?%s", strings.TrimRight(apiURL, "/"), resource, values.Encode()), nil)
	if err != nil {
		return nil, err
	}

	count := resp.Header.Get("X-Total-Count")
	var maxElem int
	if count == "" {
		maxElem = 0
	} else {
		maxElem, err = strconv.Atoi(count)
		if err != nil {
			return nil, err
		}
	}

	get := maxElem / perPage
	if (float32(maxElem) / perPage) > float32(get) {
		get++
	}

	if get <= 1 { // If there is 0 or 1 page of result, the response is not paginated
		if len(values) == 0 {
			return s.response("GET", fmt.Sprintf("%s/%s", strings.TrimRight(apiURL, "/"), resource), nil)
		}
		return s.response("GET", fmt.Sprintf("%s/%s?%s", strings.TrimRight(apiURL, "/"), resource, values.Encode()), nil)
	}

	fetchAll := !(values.Get("per_page") != "" || values.Get("page") != "")
	if fetchAll {
		var g errgroup.Group

		ch := make(chan *http.Response, get)
		for i := 1; i <= get; i++ {
			i := i // closure tricks
			g.Go(func() (err error) {
				var resp *http.Response

				val := url.Values{}
				val.Set("per_page", fmt.Sprintf("%v", perPage))
				val.Set("page", fmt.Sprintf("%v", i))
				resp, err = s.response("GET", fmt.Sprintf("%s/%s?%s", strings.TrimRight(apiURL, "/"), resource, val.Encode()), nil)
				ch <- resp
				return
			})
		}
		if err = g.Wait(); err != nil {
			return nil, err
		}
		newBody := make(map[string][]json.RawMessage)
		body := make(map[string][]json.RawMessage)
		key := ""
		for i := 0; i < get; i++ {
			res := <-ch
			if res.StatusCode != http.StatusOK {
				return res, nil
			}
			content, err := ioutil.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(content, &body); err != nil {
				return nil, err
			}

			if i == 0 {
				resp = res
				for k := range body {
					key = k
					break
				}
			}
			newBody[key] = append(newBody[key], body[key]...)
		}
		payload := new(bytes.Buffer)
		if err := json.NewEncoder(payload).Encode(newBody); err != nil {
			return nil, err
		}
		resp.Body = ioutil.NopCloser(payload)
	} else {
		resp, err = s.response("GET", fmt.Sprintf("%s/%s?%s", strings.TrimRight(apiURL, "/"), resource, values.Encode()), nil)
	}
	return resp, err
}

// PostResponse returns an http.Response object for the updated resource
func (s *ScalewayAPI) PostResponse(apiURL, resource string, data interface{}) (*http.Response, error) {
	payload := new(bytes.Buffer)
	if err := json.NewEncoder(payload).Encode(data); err != nil {
		return nil, err
	}
	return s.response("POST", fmt.Sprintf("%s/%s", strings.TrimRight(apiURL, "/"), resource), payload)
}

// PatchResponse returns an http.Response object for the updated resource
func (s *ScalewayAPI) PatchResponse(apiURL, resource string, data interface{}) (*http.Response, error) {
	payload := new(bytes.Buffer)
	if err := json.NewEncoder(payload).Encode(data); err != nil {
		return nil, err
	}
	return s.response("PATCH", fmt.Sprintf("%s/%s", strings.TrimRight(apiURL, "/"), resource), payload)
}

// PutResponse returns an http.Response object for the updated resource
func (s *ScalewayAPI) PutResponse(apiURL, resource string, data interface{}) (*http.Response, error) {
	payload := new(bytes.Buffer)
	if err := json.NewEncoder(payload).Encode(data); err != nil {
		return nil, err
	}
	return s.response("PUT", fmt.Sprintf("%s/%s", strings.TrimRight(apiURL, "/"), resource), payload)
}

// DeleteResponse returns an http.Response object for the deleted resource
func (s *ScalewayAPI) DeleteResponse(apiURL, resource string) (*http.Response, error) {
	return s.response("DELETE", fmt.Sprintf("%s/%s", strings.TrimRight(apiURL, "/"), resource), nil)
}

// handleHTTPError checks the statusCode and displays the error
func (s *ScalewayAPI) handleHTTPError(goodStatusCode []int, resp *http.Response) ([]byte, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if s.verbose {
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			var js bytes.Buffer

			err = json.Indent(&js, body, "", "  ")
			if err != nil {
				s.Debugf("[Response]: [%v]\n%v", resp.StatusCode, string(dump))
			} else {
				s.Debugf("[Response]: [%v]\n%v", resp.StatusCode, js.String())
			}
		}
	} else {
		s.Debugf("[Response]: [%v]\n%v", resp.StatusCode, string(body))
	}

	if resp.StatusCode >= http.StatusInternalServerError {
		return nil, errors.New(string(body))
	}
	good := false
	for _, code := range goodStatusCode {
		if code == resp.StatusCode {
			good = true
		}
	}
	if !good {
		var scwError ScalewayAPIError

		if err := json.Unmarshal(body, &scwError); err != nil {
			return nil, err
		}
		scwError.StatusCode = resp.StatusCode
		s.Debugf("%s", scwError.Error())
		return nil, scwError
	}
	return body, nil
}

func (s *ScalewayAPI) fetchServers(api string, query url.Values, out chan<- ScalewayServers) func() error {
	return func() error {
		resp, err := s.GetResponsePaginate(api, "servers", query)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
		if err != nil {
			return err
		}
		var servers ScalewayServers

		if err = json.Unmarshal(body, &servers); err != nil {
			return err
		}
		out <- servers
		return nil
	}
}

// GetServers gets the list of servers from the ScalewayAPI
func (s *ScalewayAPI) GetServers(all bool, limit int) (*[]ScalewayServer, error) {
	query := url.Values{}
	if !all {
		query.Set("state", "running")
	}
	if limit > 0 {
		// FIXME: wait for the API to be ready
		// query.Set("per_page", strconv.Itoa(limit))
		panic("Not implemented yet")
	}
	if all && limit == 0 {
		s.Cache.ClearServers()
	}
	var (
		g    errgroup.Group
		apis = []string{
			ComputeAPIPar1,
			ComputeAPIAms1,
		}
	)

	serverChan := make(chan ScalewayServers, 2)
	for _, api := range apis {
		g.Go(s.fetchServers(api, query, serverChan))
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(serverChan)
	var servers ScalewayServers

	for server := range serverChan {
		servers.Servers = append(servers.Servers, server.Servers...)
	}

	for i, server := range servers.Servers {
		servers.Servers[i].DNSPublic = server.Identifier + URLPublicDNS
		servers.Servers[i].DNSPrivate = server.Identifier + URLPrivateDNS
		s.Cache.InsertServer(server.Identifier, server.Location.ZoneID, server.Arch, server.Organization, server.Name)
	}
	return &servers.Servers, nil
}

// ScalewaySortServers represents a wrapper to sort by CreationDate the servers
type ScalewaySortServers []ScalewayServer

func (s ScalewaySortServers) Len() int {
	return len(s)
}

func (s ScalewaySortServers) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ScalewaySortServers) Less(i, j int) bool {
	date1, _ := time.Parse("2006-01-02T15:04:05.000000+00:00", s[i].CreationDate)
	date2, _ := time.Parse("2006-01-02T15:04:05.000000+00:00", s[j].CreationDate)
	return date2.Before(date1)
}

// GetServer gets a server from the ScalewayAPI
func (s *ScalewayAPI) GetServer(serverID string) (*ScalewayServer, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "servers/"+serverID, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}

	var oneServer ScalewayOneServer

	if err = json.Unmarshal(body, &oneServer); err != nil {
		return nil, err
	}
	// FIXME arch, owner, title
	oneServer.Server.DNSPublic = oneServer.Server.Identifier + URLPublicDNS
	oneServer.Server.DNSPrivate = oneServer.Server.Identifier + URLPrivateDNS
	s.Cache.InsertServer(oneServer.Server.Identifier, oneServer.Server.Location.ZoneID, oneServer.Server.Arch, oneServer.Server.Organization, oneServer.Server.Name)
	return &oneServer.Server, nil
}

// PostServerAction posts an action on a server
func (s *ScalewayAPI) PostServerAction(serverID, action string) error {
	data := ScalewayServerAction{
		Action: action,
	}
	resp, err := s.PostResponse(s.computeAPI, fmt.Sprintf("servers/%s/action", serverID), data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusAccepted}, resp)
	return err
}

// DeleteServer deletes a server
func (s *ScalewayAPI) DeleteServer(serverID string) error {
	defer s.Cache.RemoveServer(serverID)
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("servers/%s", serverID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err = s.handleHTTPError([]int{http.StatusNoContent}, resp); err != nil {
		return err
	}
	return nil
}

// PostServer creates a new server
func (s *ScalewayAPI) PostServer(definition ScalewayServerDefinition) (string, error) {
	definition.Organization = s.Organization

	resp, err := s.PostResponse(s.computeAPI, "servers", definition)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusCreated}, resp)
	if err != nil {
		return "", err
	}
	var server ScalewayOneServer

	if err = json.Unmarshal(body, &server); err != nil {
		return "", err
	}
	// FIXME arch, owner, title
	s.Cache.InsertServer(server.Server.Identifier, server.Server.Location.ZoneID, server.Server.Arch, server.Server.Organization, server.Server.Name)
	return server.Server.Identifier, nil
}

// PatchUserSSHKey updates a user
func (s *ScalewayAPI) PatchUserSSHKey(UserID string, definition ScalewayUserPatchSSHKeyDefinition) error {
	resp, err := s.PatchResponse(AccountAPI, fmt.Sprintf("users/%s", UserID), definition)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if _, err := s.handleHTTPError([]int{http.StatusOK}, resp); err != nil {
		return err
	}
	return nil
}

// PatchServer updates a server
func (s *ScalewayAPI) PatchServer(serverID string, definition ScalewayServerPatchDefinition) error {
	resp, err := s.PatchResponse(s.computeAPI, fmt.Sprintf("servers/%s", serverID), definition)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := s.handleHTTPError([]int{http.StatusOK}, resp); err != nil {
		return err
	}
	return nil
}

// PostSnapshot creates a new snapshot
func (s *ScalewayAPI) PostSnapshot(volumeID string, name string) (string, error) {
	definition := ScalewaySnapshotDefinition{
		VolumeIDentifier: volumeID,
		Name:             name,
		Organization:     s.Organization,
	}
	resp, err := s.PostResponse(s.computeAPI, "snapshots", definition)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusCreated}, resp)
	if err != nil {
		return "", err
	}
	var snapshot ScalewayOneSnapshot

	if err = json.Unmarshal(body, &snapshot); err != nil {
		return "", err
	}
	// FIXME arch, owner, title
	s.Cache.InsertSnapshot(snapshot.Snapshot.Identifier, "", "", snapshot.Snapshot.Organization, snapshot.Snapshot.Name)
	return snapshot.Snapshot.Identifier, nil
}

// PostImage creates a new image
func (s *ScalewayAPI) PostImage(volumeID string, name string, bootscript string, arch string) (string, error) {
	definition := ScalewayImageDefinition{
		SnapshotIDentifier: volumeID,
		Name:               name,
		Organization:       s.Organization,
		Arch:               arch,
	}
	if bootscript != "" {
		definition.DefaultBootscript = &bootscript
	}

	resp, err := s.PostResponse(s.computeAPI, "images", definition)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusCreated}, resp)
	if err != nil {
		return "", err
	}
	var image ScalewayOneImage

	if err = json.Unmarshal(body, &image); err != nil {
		return "", err
	}
	// FIXME region, arch, owner, title
	s.Cache.InsertImage(image.Image.Identifier, "", image.Image.Arch, image.Image.Organization, image.Image.Name, "")
	return image.Image.Identifier, nil
}

// PostVolume creates a new volume
func (s *ScalewayAPI) PostVolume(definition ScalewayVolumeDefinition) (string, error) {
	definition.Organization = s.Organization
	if definition.Type == "" {
		definition.Type = "l_ssd"
	}

	resp, err := s.PostResponse(s.computeAPI, "volumes", definition)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusCreated}, resp)
	if err != nil {
		return "", err
	}
	var volume ScalewayOneVolume

	if err = json.Unmarshal(body, &volume); err != nil {
		return "", err
	}
	// FIXME: s.Cache.InsertVolume(volume.Volume.Identifier, volume.Volume.Name)
	return volume.Volume.Identifier, nil
}

// PutVolume updates a volume
func (s *ScalewayAPI) PutVolume(volumeID string, definition ScalewayVolumePutDefinition) error {
	resp, err := s.PutResponse(s.computeAPI, fmt.Sprintf("volumes/%s", volumeID), definition)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// ResolveServer attempts to find a matching Identifier for the input string
func (s *ScalewayAPI) ResolveServer(needle string) (ScalewayResolverResults, error) {
	servers, err := s.Cache.LookUpServers(needle, true)
	if err != nil {
		return servers, err
	}
	if len(servers) == 0 {
		if _, err = s.GetServers(true, 0); err != nil {
			return nil, err
		}
		servers, err = s.Cache.LookUpServers(needle, true)
	}
	return servers, err
}

// ResolveVolume attempts to find a matching Identifier for the input string
func (s *ScalewayAPI) ResolveVolume(needle string) (ScalewayResolverResults, error) {
	volumes, err := s.Cache.LookUpVolumes(needle, true)
	if err != nil {
		return volumes, err
	}
	if len(volumes) == 0 {
		if _, err = s.GetVolumes(); err != nil {
			return nil, err
		}
		volumes, err = s.Cache.LookUpVolumes(needle, true)
	}
	return volumes, err
}

// ResolveSnapshot attempts to find a matching Identifier for the input string
func (s *ScalewayAPI) ResolveSnapshot(needle string) (ScalewayResolverResults, error) {
	snapshots, err := s.Cache.LookUpSnapshots(needle, true)
	if err != nil {
		return snapshots, err
	}
	if len(snapshots) == 0 {
		if _, err = s.GetSnapshots(); err != nil {
			return nil, err
		}
		snapshots, err = s.Cache.LookUpSnapshots(needle, true)
	}
	return snapshots, err
}

// ResolveImage attempts to find a matching Identifier for the input string
func (s *ScalewayAPI) ResolveImage(needle string) (ScalewayResolverResults, error) {
	images, err := s.Cache.LookUpImages(needle, true)
	if err != nil {
		return images, err
	}
	if len(images) == 0 {
		if _, err = s.GetImages(); err != nil {
			return nil, err
		}
		images, err = s.Cache.LookUpImages(needle, true)
	}
	return images, err
}

// ResolveBootscript attempts to find a matching Identifier for the input string
func (s *ScalewayAPI) ResolveBootscript(needle string) (ScalewayResolverResults, error) {
	bootscripts, err := s.Cache.LookUpBootscripts(needle, true)
	if err != nil {
		return bootscripts, err
	}
	if len(bootscripts) == 0 {
		if _, err = s.GetBootscripts(); err != nil {
			return nil, err
		}
		bootscripts, err = s.Cache.LookUpBootscripts(needle, true)
	}
	return bootscripts, err
}

// GetImages gets the list of images from the ScalewayAPI
func (s *ScalewayAPI) GetImages() (*[]MarketImage, error) {
	images, err := s.GetMarketPlaceImages("")
	if err != nil {
		return nil, err
	}
	s.Cache.ClearImages()
	for i, image := range images.Images {
		if image.CurrentPublicVersion != "" {
			for _, version := range image.Versions {
				if version.ID == image.CurrentPublicVersion {
					for _, localImage := range version.LocalImages {
						images.Images[i].Public = true
						s.Cache.InsertImage(localImage.ID, localImage.Zone, localImage.Arch, image.Organization.ID, image.Name, image.CurrentPublicVersion)
					}
				}
			}
		}
	}
	values := url.Values{}
	values.Set("organization", s.Organization)
	resp, err := s.GetResponsePaginate(s.computeAPI, "images", values)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var OrgaImages ScalewayImages

	if err = json.Unmarshal(body, &OrgaImages); err != nil {
		return nil, err
	}

	for _, orgaImage := range OrgaImages.Images {
		images.Images = append(images.Images, MarketImage{
			Categories:           []string{"MyImages"},
			CreationDate:         orgaImage.CreationDate,
			CurrentPublicVersion: orgaImage.Identifier,
			ModificationDate:     orgaImage.ModificationDate,
			Name:                 orgaImage.Name,
			Public:               false,
			MarketVersions: MarketVersions{
				Versions: []MarketVersionDefinition{
					{
						CreationDate:     orgaImage.CreationDate,
						ID:               orgaImage.Identifier,
						ModificationDate: orgaImage.ModificationDate,
						MarketLocalImages: MarketLocalImages{
							LocalImages: []MarketLocalImageDefinition{
								{
									Arch: orgaImage.Arch,
									ID:   orgaImage.Identifier,
									// TODO: fecth images from ams1 and par1
									Zone: s.Region,
								},
							},
						},
					},
				},
			},
		})
		s.Cache.InsertImage(orgaImage.Identifier, s.Region, orgaImage.Arch, orgaImage.Organization, orgaImage.Name, "")
	}
	return &images.Images, nil
}

// GetImage gets an image from the ScalewayAPI
func (s *ScalewayAPI) GetImage(imageID string) (*ScalewayImage, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "images/"+imageID, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var oneImage ScalewayOneImage

	if err = json.Unmarshal(body, &oneImage); err != nil {
		return nil, err
	}
	// FIXME owner, title
	s.Cache.InsertImage(oneImage.Image.Identifier, s.Region, oneImage.Image.Arch, oneImage.Image.Organization, oneImage.Image.Name, "")
	return &oneImage.Image, nil
}

// DeleteImage deletes a image
func (s *ScalewayAPI) DeleteImage(imageID string) error {
	defer s.Cache.RemoveImage(imageID)
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("images/%s", imageID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := s.handleHTTPError([]int{http.StatusNoContent}, resp); err != nil {
		return err
	}
	return nil
}

// DeleteSnapshot deletes a snapshot
func (s *ScalewayAPI) DeleteSnapshot(snapshotID string) error {
	defer s.Cache.RemoveSnapshot(snapshotID)
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("snapshots/%s", snapshotID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := s.handleHTTPError([]int{http.StatusNoContent}, resp); err != nil {
		return err
	}
	return nil
}

// DeleteVolume deletes a volume
func (s *ScalewayAPI) DeleteVolume(volumeID string) error {
	defer s.Cache.RemoveVolume(volumeID)
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("volumes/%s", volumeID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if _, err := s.handleHTTPError([]int{http.StatusNoContent}, resp); err != nil {
		return err
	}
	return nil
}

// GetSnapshots gets the list of snapshots from the ScalewayAPI
func (s *ScalewayAPI) GetSnapshots() (*[]ScalewaySnapshot, error) {
	query := url.Values{}
	s.Cache.ClearSnapshots()

	resp, err := s.GetResponsePaginate(s.computeAPI, "snapshots", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var snapshots ScalewaySnapshots

	if err = json.Unmarshal(body, &snapshots); err != nil {
		return nil, err
	}
	for _, snapshot := range snapshots.Snapshots {
		// FIXME region, arch, owner, title
		s.Cache.InsertSnapshot(snapshot.Identifier, "", "", snapshot.Organization, snapshot.Name)
	}
	return &snapshots.Snapshots, nil
}

// GetSnapshot gets a snapshot from the ScalewayAPI
func (s *ScalewayAPI) GetSnapshot(snapshotID string) (*ScalewaySnapshot, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "snapshots/"+snapshotID, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var oneSnapshot ScalewayOneSnapshot

	if err = json.Unmarshal(body, &oneSnapshot); err != nil {
		return nil, err
	}
	// FIXME region, arch, owner, title
	s.Cache.InsertSnapshot(oneSnapshot.Snapshot.Identifier, "", "", oneSnapshot.Snapshot.Organization, oneSnapshot.Snapshot.Name)
	return &oneSnapshot.Snapshot, nil
}

// GetVolumes gets the list of volumes from the ScalewayAPI
func (s *ScalewayAPI) GetVolumes() (*[]ScalewayVolume, error) {
	query := url.Values{}
	s.Cache.ClearVolumes()

	resp, err := s.GetResponsePaginate(s.computeAPI, "volumes", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}

	var volumes ScalewayVolumes

	if err = json.Unmarshal(body, &volumes); err != nil {
		return nil, err
	}
	for _, volume := range volumes.Volumes {
		// FIXME region, arch, owner, title
		s.Cache.InsertVolume(volume.Identifier, "", "", volume.Organization, volume.Name)
	}
	return &volumes.Volumes, nil
}

// GetVolume gets a volume from the ScalewayAPI
func (s *ScalewayAPI) GetVolume(volumeID string) (*ScalewayVolume, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "volumes/"+volumeID, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var oneVolume ScalewayOneVolume

	if err = json.Unmarshal(body, &oneVolume); err != nil {
		return nil, err
	}
	// FIXME region, arch, owner, title
	s.Cache.InsertVolume(oneVolume.Volume.Identifier, "", "", oneVolume.Volume.Organization, oneVolume.Volume.Name)
	return &oneVolume.Volume, nil
}

// GetBootscripts gets the list of bootscripts from the ScalewayAPI
func (s *ScalewayAPI) GetBootscripts() (*[]ScalewayBootscript, error) {
	query := url.Values{}

	s.Cache.ClearBootscripts()
	resp, err := s.GetResponsePaginate(s.computeAPI, "bootscripts", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var bootscripts ScalewayBootscripts

	if err = json.Unmarshal(body, &bootscripts); err != nil {
		return nil, err
	}
	for _, bootscript := range bootscripts.Bootscripts {
		// FIXME region, arch, owner, title
		s.Cache.InsertBootscript(bootscript.Identifier, "", bootscript.Arch, bootscript.Organization, bootscript.Title)
	}
	return &bootscripts.Bootscripts, nil
}

// GetBootscript gets a bootscript from the ScalewayAPI
func (s *ScalewayAPI) GetBootscript(bootscriptID string) (*ScalewayBootscript, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "bootscripts/"+bootscriptID, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var oneBootscript ScalewayOneBootscript

	if err = json.Unmarshal(body, &oneBootscript); err != nil {
		return nil, err
	}
	// FIXME region, arch, owner, title
	s.Cache.InsertBootscript(oneBootscript.Bootscript.Identifier, "", oneBootscript.Bootscript.Arch, oneBootscript.Bootscript.Organization, oneBootscript.Bootscript.Title)
	return &oneBootscript.Bootscript, nil
}

// GetUserdatas gets list of userdata for a server
func (s *ScalewayAPI) GetUserdatas(serverID string, metadata bool) (*ScalewayUserdatas, error) {
	var uri, endpoint string

	endpoint = s.computeAPI
	if metadata {
		uri = "/user_data"
		endpoint = MetadataAPI
	} else {
		uri = fmt.Sprintf("servers/%s/user_data", serverID)
	}

	resp, err := s.GetResponsePaginate(endpoint, uri, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var userdatas ScalewayUserdatas

	if err = json.Unmarshal(body, &userdatas); err != nil {
		return nil, err
	}
	return &userdatas, nil
}

func (s *ScalewayUserdata) String() string {
	return string(*s)
}

// GetUserdata gets a specific userdata for a server
func (s *ScalewayAPI) GetUserdata(serverID, key string, metadata bool) (*ScalewayUserdata, error) {
	var uri, endpoint string

	endpoint = s.computeAPI
	if metadata {
		uri = fmt.Sprintf("/user_data/%s", key)
		endpoint = MetadataAPI
	} else {
		uri = fmt.Sprintf("servers/%s/user_data/%s", serverID, key)
	}

	var err error
	resp, err := s.GetResponsePaginate(endpoint, uri, url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("no such user_data %q (%d)", key, resp.StatusCode)
	}
	var data ScalewayUserdata
	data, err = ioutil.ReadAll(resp.Body)
	return &data, err
}

// PatchUserdata sets a user data
func (s *ScalewayAPI) PatchUserdata(serverID, key string, value []byte, metadata bool) error {
	var resource, endpoint string

	endpoint = s.computeAPI
	if metadata {
		resource = fmt.Sprintf("/user_data/%s", key)
		endpoint = MetadataAPI
	} else {
		resource = fmt.Sprintf("servers/%s/user_data/%s", serverID, key)
	}

	uri := fmt.Sprintf("%s/%s", strings.TrimRight(endpoint, "/"), resource)
	payload := new(bytes.Buffer)
	payload.Write(value)

	req, err := http.NewRequest("PATCH", uri, payload)
	if err != nil {
		return err
	}

	req.Header.Set("X-Auth-Token", s.Token)
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("User-Agent", s.userAgent)

	s.LogHTTP(req)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	return fmt.Errorf("cannot set user_data (%d)", resp.StatusCode)
}

// DeleteUserdata deletes a server user_data
func (s *ScalewayAPI) DeleteUserdata(serverID, key string, metadata bool) error {
	var url, endpoint string

	endpoint = s.computeAPI
	if metadata {
		url = fmt.Sprintf("/user_data/%s", key)
		endpoint = MetadataAPI
	} else {
		url = fmt.Sprintf("servers/%s/user_data/%s", serverID, key)
	}

	resp, err := s.DeleteResponse(endpoint, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// GetTasks get the list of tasks from the ScalewayAPI
func (s *ScalewayAPI) GetTasks() (*[]ScalewayTask, error) {
	query := url.Values{}
	resp, err := s.GetResponsePaginate(s.computeAPI, "tasks", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var tasks ScalewayTasks

	if err = json.Unmarshal(body, &tasks); err != nil {
		return nil, err
	}
	return &tasks.Tasks, nil
}

// CheckCredentials performs a dummy check to ensure we can contact the API
func (s *ScalewayAPI) CheckCredentials() error {
	query := url.Values{}

	resp, err := s.GetResponsePaginate(AccountAPI, "tokens", query)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return err
	}
	found := false
	var tokens ScalewayGetTokens

	if err = json.Unmarshal(body, &tokens); err != nil {
		return err
	}
	for _, token := range tokens.Tokens {
		if token.ID == s.Token {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Invalid token %v", s.Token)
	}
	return nil
}

// GetUserID returns the userID
func (s *ScalewayAPI) GetUserID() (string, error) {
	resp, err := s.GetResponsePaginate(AccountAPI, fmt.Sprintf("tokens/%s", s.Token), url.Values{})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return "", err
	}
	var token ScalewayTokensDefinition

	if err = json.Unmarshal(body, &token); err != nil {
		return "", err
	}
	return token.Token.UserID, nil
}

// GetOrganization returns Organization
func (s *ScalewayAPI) GetOrganization() (*ScalewayOrganizationsDefinition, error) {
	resp, err := s.GetResponsePaginate(AccountAPI, "organizations", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var data ScalewayOrganizationsDefinition

	if err = json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// GetUser returns the user
func (s *ScalewayAPI) GetUser() (*ScalewayUserDefinition, error) {
	userID, err := s.GetUserID()
	if err != nil {
		return nil, err
	}
	resp, err := s.GetResponsePaginate(AccountAPI, fmt.Sprintf("users/%s", userID), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var user ScalewayUsersDefinition

	if err = json.Unmarshal(body, &user); err != nil {
		return nil, err
	}
	return &user.User, nil
}

// GetPermissions returns the permissions
func (s *ScalewayAPI) GetPermissions() (*ScalewayPermissionDefinition, error) {
	resp, err := s.GetResponsePaginate(AccountAPI, fmt.Sprintf("tokens/%s/permissions", s.Token), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var permissions ScalewayPermissionDefinition

	if err = json.Unmarshal(body, &permissions); err != nil {
		return nil, err
	}
	return &permissions, nil
}

// GetDashboard returns the dashboard
func (s *ScalewayAPI) GetDashboard() (*ScalewayDashboard, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "dashboard", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var dashboard ScalewayDashboardResp

	if err = json.Unmarshal(body, &dashboard); err != nil {
		return nil, err
	}
	return &dashboard.Dashboard, nil
}

// GetServerID returns exactly one server matching
func (s *ScalewayAPI) GetServerID(needle string) (string, error) {
	// Parses optional type prefix, i.e: "server:name" -> "name"
	_, needle = parseNeedle(needle)

	servers, err := s.ResolveServer(needle)
	if err != nil {
		return "", fmt.Errorf("Unable to resolve server %s: %s", needle, err)
	}
	if len(servers) == 1 {
		return servers[0].Identifier, nil
	}
	if len(servers) == 0 {
		return "", fmt.Errorf("No such server: %s", needle)
	}
	return "", showResolverResults(needle, servers)
}

func showResolverResults(needle string, results ScalewayResolverResults) error {
	w := tabwriter.NewWriter(os.Stderr, 20, 1, 3, ' ', 0)
	defer w.Flush()
	sort.Sort(results)
	fmt.Fprintf(w, "  IMAGEID\tFROM\tNAME\tZONE\tARCH\n")
	for _, result := range results {
		if result.Arch == "" {
			result.Arch = "n/a"
		}
		fmt.Fprintf(w, "- %s\t%s\t%s\t%s\t%s\n", result.TruncIdentifier(), result.CodeName(), result.Name, result.Region, result.Arch)
	}
	return fmt.Errorf("Too many candidates for %s (%d)", needle, len(results))
}

// GetVolumeID returns exactly one volume matching
func (s *ScalewayAPI) GetVolumeID(needle string) (string, error) {
	// Parses optional type prefix, i.e: "volume:name" -> "name"
	_, needle = parseNeedle(needle)

	volumes, err := s.ResolveVolume(needle)
	if err != nil {
		return "", fmt.Errorf("Unable to resolve volume %s: %s", needle, err)
	}
	if len(volumes) == 1 {
		return volumes[0].Identifier, nil
	}
	if len(volumes) == 0 {
		return "", fmt.Errorf("No such volume: %s", needle)
	}
	return "", showResolverResults(needle, volumes)
}

// GetSnapshotID returns exactly one snapshot matching
func (s *ScalewayAPI) GetSnapshotID(needle string) (string, error) {
	// Parses optional type prefix, i.e: "snapshot:name" -> "name"
	_, needle = parseNeedle(needle)

	snapshots, err := s.ResolveSnapshot(needle)
	if err != nil {
		return "", fmt.Errorf("Unable to resolve snapshot %s: %s", needle, err)
	}
	if len(snapshots) == 1 {
		return snapshots[0].Identifier, nil
	}
	if len(snapshots) == 0 {
		return "", fmt.Errorf("No such snapshot: %s", needle)
	}
	return "", showResolverResults(needle, snapshots)
}

// FilterImagesByArch removes entry that doesn't match with architecture
func FilterImagesByArch(res ScalewayResolverResults, arch string) (ret ScalewayResolverResults) {
	if arch == "*" {
		return res
	}
	for _, result := range res {
		if result.Arch == arch {
			ret = append(ret, result)
		}
	}
	return
}

// FilterImagesByRegion removes entry that doesn't match with region
func FilterImagesByRegion(res ScalewayResolverResults, region string) (ret ScalewayResolverResults) {
	if region == "*" {
		return res
	}
	for _, result := range res {
		if result.Region == region {
			ret = append(ret, result)
		}
	}
	return
}

// GetImageID returns exactly one image matching
func (s *ScalewayAPI) GetImageID(needle, arch string) (*ScalewayImageIdentifier, error) {
	// Parses optional type prefix, i.e: "image:name" -> "name"
	_, needle = parseNeedle(needle)

	images, err := s.ResolveImage(needle)
	if err != nil {
		return nil, fmt.Errorf("Unable to resolve image %s: %s", needle, err)
	}
	images = FilterImagesByArch(images, arch)
	images = FilterImagesByRegion(images, s.Region)
	if len(images) == 1 {
		return &ScalewayImageIdentifier{
			Identifier: images[0].Identifier,
			Arch:       images[0].Arch,
			// FIXME region, owner hardcoded
			Region: images[0].Region,
			Owner:  "",
		}, nil
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("No such image (zone %s, arch %s) : %s", s.Region, arch, needle)
	}
	return nil, showResolverResults(needle, images)
}

// GetSecurityGroups returns a ScalewaySecurityGroups
func (s *ScalewayAPI) GetSecurityGroups() (*ScalewayGetSecurityGroups, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "security_groups", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var securityGroups ScalewayGetSecurityGroups

	if err = json.Unmarshal(body, &securityGroups); err != nil {
		return nil, err
	}
	return &securityGroups, nil
}

// GetSecurityGroupRules returns a ScalewaySecurityGroupRules
func (s *ScalewayAPI) GetSecurityGroupRules(groupID string) (*ScalewayGetSecurityGroupRules, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, fmt.Sprintf("security_groups/%s/rules", groupID), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var securityGroupRules ScalewayGetSecurityGroupRules

	if err = json.Unmarshal(body, &securityGroupRules); err != nil {
		return nil, err
	}
	return &securityGroupRules, nil
}

// GetASecurityGroupRule returns a ScalewaySecurityGroupRule
func (s *ScalewayAPI) GetASecurityGroupRule(groupID string, rulesID string) (*ScalewayGetSecurityGroupRule, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, fmt.Sprintf("security_groups/%s/rules/%s", groupID, rulesID), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var securityGroupRules ScalewayGetSecurityGroupRule

	if err = json.Unmarshal(body, &securityGroupRules); err != nil {
		return nil, err
	}
	return &securityGroupRules, nil
}

// GetASecurityGroup returns a ScalewaySecurityGroup
func (s *ScalewayAPI) GetASecurityGroup(groupsID string) (*ScalewayGetSecurityGroup, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, fmt.Sprintf("security_groups/%s", groupsID), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var securityGroups ScalewayGetSecurityGroup

	if err = json.Unmarshal(body, &securityGroups); err != nil {
		return nil, err
	}
	return &securityGroups, nil
}

// PostSecurityGroup posts a group on a server
func (s *ScalewayAPI) PostSecurityGroup(group ScalewayNewSecurityGroup) error {
	resp, err := s.PostResponse(s.computeAPI, "security_groups", group)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusCreated}, resp)
	return err
}

// PostSecurityGroupRule posts a rule on a server
func (s *ScalewayAPI) PostSecurityGroupRule(SecurityGroupID string, rules ScalewayNewSecurityGroupRule) error {
	resp, err := s.PostResponse(s.computeAPI, fmt.Sprintf("security_groups/%s/rules", SecurityGroupID), rules)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusCreated}, resp)
	return err
}

// DeleteSecurityGroup deletes a SecurityGroup
func (s *ScalewayAPI) DeleteSecurityGroup(securityGroupID string) error {
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("security_groups/%s", securityGroupID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// PutSecurityGroup updates a SecurityGroup
func (s *ScalewayAPI) PutSecurityGroup(group ScalewayUpdateSecurityGroup, securityGroupID string) error {
	resp, err := s.PutResponse(s.computeAPI, fmt.Sprintf("security_groups/%s", securityGroupID), group)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// PutSecurityGroupRule updates a SecurityGroupRule
func (s *ScalewayAPI) PutSecurityGroupRule(rules ScalewayNewSecurityGroupRule, securityGroupID, RuleID string) error {
	resp, err := s.PutResponse(s.computeAPI, fmt.Sprintf("security_groups/%s/rules/%s", securityGroupID, RuleID), rules)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// DeleteSecurityGroupRule deletes a SecurityGroupRule
func (s *ScalewayAPI) DeleteSecurityGroupRule(SecurityGroupID, RuleID string) error {
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("security_groups/%s/rules/%s", SecurityGroupID, RuleID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// GetContainers returns a ScalewayGetContainers
func (s *ScalewayAPI) GetContainers() (*ScalewayGetContainers, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "containers", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var containers ScalewayGetContainers

	if err = json.Unmarshal(body, &containers); err != nil {
		return nil, err
	}
	return &containers, nil
}

// GetContainerDatas returns a ScalewayGetContainerDatas
func (s *ScalewayAPI) GetContainerDatas(container string) (*ScalewayGetContainerDatas, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, fmt.Sprintf("containers/%s", container), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var datas ScalewayGetContainerDatas

	if err = json.Unmarshal(body, &datas); err != nil {
		return nil, err
	}
	return &datas, nil
}

// GetIPS returns a ScalewayGetIPS
func (s *ScalewayAPI) GetIPS() (*ScalewayGetIPS, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, "ips", url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ips ScalewayGetIPS

	if err = json.Unmarshal(body, &ips); err != nil {
		return nil, err
	}
	return &ips, nil
}

// NewIP returns a new IP
func (s *ScalewayAPI) NewIP() (*ScalewayGetIP, error) {
	var orga struct {
		Organization string `json:"organization"`
	}
	orga.Organization = s.Organization
	resp, err := s.PostResponse(s.computeAPI, "ips", orga)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusCreated}, resp)
	if err != nil {
		return nil, err
	}
	var ip ScalewayGetIP

	if err = json.Unmarshal(body, &ip); err != nil {
		return nil, err
	}
	return &ip, nil
}

// AttachIP attachs an IP to a server
func (s *ScalewayAPI) AttachIP(ipID, serverID string) error {
	var update struct {
		Address      string  `json:"address"`
		ID           string  `json:"id"`
		Reverse      *string `json:"reverse"`
		Organization string  `json:"organization"`
		Server       string  `json:"server"`
	}

	ip, err := s.GetIP(ipID)
	if err != nil {
		return err
	}
	update.Address = ip.IP.Address
	update.ID = ip.IP.ID
	update.Organization = ip.IP.Organization
	update.Server = serverID
	resp, err := s.PutResponse(s.computeAPI, fmt.Sprintf("ips/%s", ipID), update)
	if err != nil {
		return err
	}
	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// DetachIP detaches an IP from a server
func (s *ScalewayAPI) DetachIP(ipID string) error {
	ip, err := s.GetIP(ipID)
	if err != nil {
		return err
	}
	ip.IP.Server = nil
	resp, err := s.PutResponse(s.computeAPI, fmt.Sprintf("ips/%s", ipID), ip.IP)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// DeleteIP deletes an IP
func (s *ScalewayAPI) DeleteIP(ipID string) error {
	resp, err := s.DeleteResponse(s.computeAPI, fmt.Sprintf("ips/%s", ipID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// GetIP returns a ScalewayGetIP
func (s *ScalewayAPI) GetIP(ipID string) (*ScalewayGetIP, error) {
	resp, err := s.GetResponsePaginate(s.computeAPI, fmt.Sprintf("ips/%s", ipID), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ip ScalewayGetIP

	if err = json.Unmarshal(body, &ip); err != nil {
		return nil, err
	}
	return &ip, nil
}

// GetQuotas returns a ScalewayGetQuotas
func (s *ScalewayAPI) GetQuotas() (*ScalewayGetQuotas, error) {
	resp, err := s.GetResponsePaginate(AccountAPI, fmt.Sprintf("organizations/%s/quotas", s.Organization), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var quotas ScalewayGetQuotas

	if err = json.Unmarshal(body, &quotas); err != nil {
		return nil, err
	}
	return &quotas, nil
}

// GetBootscriptID returns exactly one bootscript matching
func (s *ScalewayAPI) GetBootscriptID(needle, arch string) (string, error) {
	// Parses optional type prefix, i.e: "bootscript:name" -> "name"
	_, needle = parseNeedle(needle)

	bootscripts, err := s.ResolveBootscript(needle)
	if err != nil {
		return "", fmt.Errorf("Unable to resolve bootscript %s: %s", needle, err)
	}
	bootscripts.FilterByArch(arch)
	if len(bootscripts) == 1 {
		return bootscripts[0].Identifier, nil
	}
	if len(bootscripts) == 0 {
		return "", fmt.Errorf("No such bootscript: %s", needle)
	}
	return "", showResolverResults(needle, bootscripts)
}

func rootNetDial(network, addr string) (net.Conn, error) {
	dialer := net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 10 * time.Second,
	}

	// bruteforce privileged ports
	var localAddr net.Addr
	var err error
	for port := 1; port <= 1024; port++ {
		localAddr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))

		// this should never happen
		if err != nil {
			return nil, err
		}

		dialer.LocalAddr = localAddr

		conn, err := dialer.Dial(network, addr)

		// if err is nil, dialer.Dial succeed, so let's go
		// else, err != nil, but we don't care
		if err == nil {
			return conn, nil
		}
	}
	// if here, all privileged ports were tried without success
	return nil, fmt.Errorf("bind: permission denied, are you root ?")
}

// SetPassword register the password
func (s *ScalewayAPI) SetPassword(password string) {
	s.password = password
}

// GetMarketPlaceImages returns images from marketplace
func (s *ScalewayAPI) GetMarketPlaceImages(uuidImage string) (*MarketImages, error) {
	resp, err := s.GetResponsePaginate(MarketplaceAPI, fmt.Sprintf("images/%s", uuidImage), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ret MarketImages

	if uuidImage != "" {
		ret.Images = make([]MarketImage, 1)

		var img MarketImage

		if err = json.Unmarshal(body, &img); err != nil {
			return nil, err
		}
		ret.Images[0] = img
	} else {
		if err = json.Unmarshal(body, &ret); err != nil {
			return nil, err
		}
	}
	return &ret, nil
}

// GetMarketPlaceImageVersions returns image version
func (s *ScalewayAPI) GetMarketPlaceImageVersions(uuidImage, uuidVersion string) (*MarketVersions, error) {
	resp, err := s.GetResponsePaginate(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%s", uuidImage, uuidVersion), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ret MarketVersions

	if uuidImage != "" {
		var version MarketVersion
		ret.Versions = make([]MarketVersionDefinition, 1)

		if err = json.Unmarshal(body, &version); err != nil {
			return nil, err
		}
		ret.Versions[0] = version.Version
	} else {
		if err = json.Unmarshal(body, &ret); err != nil {
			return nil, err
		}
	}
	return &ret, nil
}

// GetMarketPlaceImageCurrentVersion return the image current version
func (s *ScalewayAPI) GetMarketPlaceImageCurrentVersion(uuidImage string) (*MarketVersion, error) {
	resp, err := s.GetResponsePaginate(MarketplaceAPI, fmt.Sprintf("images/%v/versions/current", uuidImage), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ret MarketVersion

	if err = json.Unmarshal(body, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

// GetMarketPlaceLocalImages returns images from local region
func (s *ScalewayAPI) GetMarketPlaceLocalImages(uuidImage, uuidVersion, uuidLocalImage string) (*MarketLocalImages, error) {
	resp, err := s.GetResponsePaginate(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%s/local_images/%s", uuidImage, uuidVersion, uuidLocalImage), url.Values{})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := s.handleHTTPError([]int{http.StatusOK}, resp)
	if err != nil {
		return nil, err
	}
	var ret MarketLocalImages
	if uuidLocalImage != "" {
		var localImage MarketLocalImage
		ret.LocalImages = make([]MarketLocalImageDefinition, 1)

		if err = json.Unmarshal(body, &localImage); err != nil {
			return nil, err
		}
		ret.LocalImages[0] = localImage.LocalImages
	} else {
		if err = json.Unmarshal(body, &ret); err != nil {
			return nil, err
		}
	}
	return &ret, nil
}

// PostMarketPlaceImage adds new image
func (s *ScalewayAPI) PostMarketPlaceImage(images MarketImage) error {
	resp, err := s.PostResponse(MarketplaceAPI, "images/", images)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusAccepted}, resp)
	return err
}

// PostMarketPlaceImageVersion adds new image version
func (s *ScalewayAPI) PostMarketPlaceImageVersion(uuidImage string, version MarketVersion) error {
	resp, err := s.PostResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions", uuidImage), version)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusAccepted}, resp)
	return err
}

// PostMarketPlaceLocalImage adds new local image
func (s *ScalewayAPI) PostMarketPlaceLocalImage(uuidImage, uuidVersion, uuidLocalImage string, local MarketLocalImage) error {
	resp, err := s.PostResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%s/local_images/%v", uuidImage, uuidVersion, uuidLocalImage), local)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusAccepted}, resp)
	return err
}

// PutMarketPlaceImage updates image
func (s *ScalewayAPI) PutMarketPlaceImage(uudiImage string, images MarketImage) error {
	resp, err := s.PutResponse(MarketplaceAPI, fmt.Sprintf("images/%v", uudiImage), images)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// PutMarketPlaceImageVersion updates image version
func (s *ScalewayAPI) PutMarketPlaceImageVersion(uuidImage, uuidVersion string, version MarketVersion) error {
	resp, err := s.PutResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%v", uuidImage, uuidVersion), version)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// PutMarketPlaceLocalImage updates local image
func (s *ScalewayAPI) PutMarketPlaceLocalImage(uuidImage, uuidVersion, uuidLocalImage string, local MarketLocalImage) error {
	resp, err := s.PostResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%s/local_images/%v", uuidImage, uuidVersion, uuidLocalImage), local)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusOK}, resp)
	return err
}

// DeleteMarketPlaceImage deletes image
func (s *ScalewayAPI) DeleteMarketPlaceImage(uudImage string) error {
	resp, err := s.DeleteResponse(MarketplaceAPI, fmt.Sprintf("images/%v", uudImage))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// DeleteMarketPlaceImageVersion delete image version
func (s *ScalewayAPI) DeleteMarketPlaceImageVersion(uuidImage, uuidVersion string) error {
	resp, err := s.DeleteResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%v", uuidImage, uuidVersion))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// DeleteMarketPlaceLocalImage deletes local image
func (s *ScalewayAPI) DeleteMarketPlaceLocalImage(uuidImage, uuidVersion, uuidLocalImage string) error {
	resp, err := s.DeleteResponse(MarketplaceAPI, fmt.Sprintf("images/%v/versions/%s/local_images/%v", uuidImage, uuidVersion, uuidLocalImage))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = s.handleHTTPError([]int{http.StatusNoContent}, resp)
	return err
}

// ResolveTTYUrl return an URL to get a tty
func (s *ScalewayAPI) ResolveTTYUrl() string {
	switch s.Region {
	case "par1", "":
		return "https://tty-par1.scaleway.com/v2/"
	case "ams1":
		return "https://tty-ams1.scaleway.com"
	}
	return ""
}
