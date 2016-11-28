/*
(c) Copyright [2015] Hewlett Packard Enterprise Development LP

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package icsp
package icsp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

// URLEndPoint export this constant
const URLEndPointServer = "/rest/os-deployment-servers"

// StorageDevice storage device type
type StorageDevice struct {
	Capacity   int    `json:"capacity,omitempty"`   // capacity Capacity of the storage in megabytes integer
	DeviceName string `json:"deviceName,omitempty"` // deviceName Device name, such as "C:" for Windows or "sda" for Linux string
	MediaType  string `json:"mediaType,omitempty"`  // mediaType Media type, such as "CDROM", "SCSI DISK" and etc. string
	Model      string `json:"model,omitempty"`      // model Model of the device string
	Vendor     string `json:"vendor,omitempty"`     // vendor Manufacturer of the device string
}

// osdState status
type osdState int

// OK - The target server is running a production OS with a production version of the agent and is reachable;
// UNREACHABLE - The managed Server is unreachable by the appliance;
// MAINTENANCE - The Server has been booted to maintenance, and a maintenance version of the agent has been registered with the appliance.;

const (
	// OsdSateOK - The target server is running a production OS with a production version of the agent and is reachable;
	OsdSateOK osdState = iota // 0
	// OsdSateUnReachable - The managed Server is unreachable by the appliance;
	OsdSateUnReachable // 1
	// OsdSateMaintenance - The Server has been booted to maintenance, and a maintenance version of the agent has been registered with the appliance.;
	OsdSateMaintenance // 2
)

var statelist = [...]string{
	"OK",          // OK - The target server is running a production OS with a production version of the agent and is reachable;
	"UNREACHABLE", // UNREACHABLE - The managed Server is unreachable by the appliance;
	"MAINTENANCE", // MAINTENANCE - The Server has been booted to maintenance, and a maintenance version of the agent has been registered with the appliance.;
}

// String helper for state
func (o osdState) String() string { return statelist[o] }

// Equal helper for state
func (o osdState) Equal(s string) bool { return (strings.ToUpper(s) == strings.ToUpper(o.String())) }

// Stage stage const
type Stage int

const (
	// StageInDeployment -
	StageInDeployment Stage = iota // 0
	// StageLive -
	StageLive // 1
	// StageOffline -
	StageOffline // 2
	// StageOpsReady -
	StageOpsReady // 3
	// StageUnknown -
	StageUnknown // 4
)

var stagelist = [...]string{
	"IN DEPLOYMENT", // - The managed Server is in process of deployment;
	"LIVE",          // - Deployment complete, the Server is live in production;
	"OFFLINE",       // - The managed Server is off-line;
	"OPS_READY",     // - The managed Server is available to operations;
	"UNKNOWN",       // - The managed Server is in an unknown stage - this is the default value for the field;
}

// String helper for stage
func (o Stage) String() string { return stagelist[o] }

// Equal helper for stage
func (o Stage) Equal(s string) bool { return (strings.ToUpper(s) == strings.ToUpper(o.String())) }

// ServerLocationItem server location type
type ServerLocationItem struct {
	Bay       string `json:"bay,omitempty"`       // bay Slot number in a rack where the Server is located string
	Enclosure string `json:"enclosure,omitempty"` // enclosure Name of an enclosure where the Server is physically located string
	Rack      string `json:"rack,omitempty"`      // rack Name of a rack where the Server is physically located string
}

// opswLifecycle opsw lifecycle
type opswLifecycle int

// Life-cycle value for the managed Server. The following are the valid values for the life-cycle of the Server:
const (
	Deactivated       opswLifecycle = iota // 0
	Managed                                // 1
	ProvisionedFailed                      // 2
	Provisioning                           // 3
	Unprovisioned                          // 4
	PreUnProvisioned                       // 5
)

var opswlifecycle = [...]string{
	"DEACTIVATED",       // - No management activities can occur once a Server is deactivated;
	"MANAGED",           // - A production OS is installed and running on the target server. Normal management activities can occur when a Server is under management;
	"PROVISION_FAILED",  // - A managed Server enters this state when operating system installation or other provisioning activities failed;
	"PROVISIONING",      // - A managed Server is set to this state any time a job is running on the server;
	"UNPROVISIONED",     // - A managed Server in this state has booted into a service OS and is waiting to have an operating system installed;
	"PRE_UNPROVISIONED", // - A managed Server in this state is defined, but has not yet booted and registered with the appliance. An example of this is an iLO that was added without booting to maintenance;
}

// String helper for OpswLifecycle
func (o opswLifecycle) String() string { return opswlifecycle[o] }

// Equal helper for OpswLifecycle
func (o opswLifecycle) Equal(s string) bool {
	return (strings.ToUpper(s) == strings.ToUpper(o.String()))
}

// JobHistory job history type
type JobHistory struct {
	Description   string        `json:"description,omitempty"`   // description Description of the job, string
	EndDate       string        `json:"endDate,omitempty"`       // endDate Date and time when job was finished, string
	Initiator     string        `json:"initiator,omitempty"`     // initiator Name of the user who invoked the job on the Server, string
	Name          string        `json:"name,omitempty"`          // name Name of the job, string
	NameOfJobType string        `json:"nameOfJobType,omitempty"` // nameOfJobType Name of the OS Build Plan that was invoked on the Server, string
	StartDate     string        `json:"startDate,omitempty"`     // startDate Date and time when job was invoked, string
	URI           utils.Nstring `json:"uri,omitempty"`           // uri The canonical URI of the job, string
	URIOfJobType  utils.Nstring `json:"uriOfJobType,omitempty"`  // uriOfJobType The canonical URI of the OS Build Plan, string
}

// Interface struct
type Interface struct {
	DHCPEnabled bool   `json:"dhcpEnabled,omitempty"` // dhcpEnabled Flag that indicates whether the interface IP address is configured using DHCP, Boolean
	Duplex      string `json:"duplex,omitempty"`      // duplex Reported duplex of the interface, string
	IPV4Addr    string `json:"ipv4Addr,omitempty"`    // ipv4Addr IPv4 address of the interface, string
	IPV6Addr    string `json:"ipv6Addr,omitempty"`    // ipv6Addr IPv6 address of the interface, string
	MACAddr     string `json:"macAddr,omitempty"`     // macAddr Interface hardware network address, string
	Netmask     string `json:"netmask,omitempty"`     // netmask Netmask in dotted decimal notation, string
	Slot        string `json:"slot,omitempty"`        // slot Interface identity reported by the Server's operating system, string
	Speed       string `json:"speed,omitempty"`       // speed Interface speed in megabits, string
	Type        string `json:"type,omitempty"`        // type Interface type. For example, ETHERNET, string
}

// Ilo struct
type Ilo struct {
	Category       string        `json:"category,omitempty"`       // category The category is used to help identify the kind of resource, string
	Created        string        `json:"created,omitempty"`        // created Date and time when iLO was first discovered by Insight Control Server Provisioning, timestamp
	Description    string        `json:"description,omitempty"`    // description General description of the iLO, string
	ETAG           string        `json:"eTag,omitempty"`           // eTag Entity tag/version ID of the resource, the same value that is returned in the ETag header on a GET of the resource, string
	HealthStatus   string        `json:"healthStatus,omitempty"`   // healthStatus Overall health status of the resource, string
	IPAddress      string        `json:"ipAddress,omitempty"`      // ipAddress The IP address of the server’s iLO, string
	Modified       string        `json:"modified,omitempty"`       // modified Date and time when the resource was last modified, timestamp
	Name           string        `json:"name,omitempty"`           // name For servers added via iLO and booted to Intelligent Provisioning service OS, host name is determined by Intelligent Provisioning. For servers added via iLO and PXE booted to LinuxPE host name is "localhost". For servers added via iLO and PXE booted to WinPE host name is a random hostname "minint-xxx", string
	Passowrd       string        `json:"password,omitempty"`       // password ILO's password, string
	Port           int           `json:"port,omitempty"`           // port The socket on which the management service listens, integer
	ResourceStatus string        `json:"resourceStatus,omitempty"` // resourceStatus Current state of the resource, string
	Server         string        `json:"server,omitempty"`         // server The canonical URI of the hosting/managed server, string
	State          string        `json:"state,omitempty"`          // state Current state of the resource, string
	Status         string        `json:"status,omitempty"`         // status Overall health status of the resource, string
	Type           string        `json:"type,omitempty"`           // type Uniquely identifies the type of the JSON object(readonly), string
	URI            utils.Nstring `json:"uri,omitempty"`            // uri Unique numerical iLO identifier, string
	Username       string        `json:"username,omitempty"`       // username Username used to log in to iLO, string
}

// DeviceGroup struct
type DeviceGroup struct {
	Name  string        `json:"name,omitempty"`  // name Display name for the resource, string
	REFID int           `json:"refID,omitempty"` // refID The unique numerical identifier, integer
	URI   utils.Nstring `json:"uri,omitempty"`   // uri The canonical URI of the device group, string
}

// CPU struct
type CPU struct {
	CacheSize string `json:"cacheSize,omitempty"` // cacheSize CPU's cache size  , string
	Family    string `json:"family,omitempty"`    // family CPU's family. For example, "x86_64"  , string
	Model     string `json:"model,omitempty"`     // model CPU's model. For example, "Xeon"  , string
	Slot      string `json:"slot,omitempty"`      // slot CPU's slot  , string
	Speed     string `json:"speed,omitempty"`     // speed CPU's speed  , string
	Status    string `json:"status,omitempty"`    // status The last reported status of the CPU. For example, on-line, off-line  , string
	Stepping  string `json:"stepping,omitempty"`  // stepping CPU's stepping information  , string
}

// Server type
type Server struct {
	Architecture           string              `json:"architecture,omitempty"`           // architecture Server's architecture, string
	Category               string              `json:"category,omitempty"`               // category The category is used to help identify the kind of resource, string
	Cpus                   []CPU               `json:"cpus,omitempty"`                   // array of CPU's
	Created                string              `json:"created,omitempty"`                // created Date and time when the Server was discovered, timestamp
	CustomAttributes       []CustomAttribute   `json:"customAttributes,omitempty"`       // array of custom attributes
	DefaultGateway         string              `json:"defaultGateway,omitempty"`         // defaultGateway Gateway for this Server, string
	Description            string              `json:"description,omitempty"`            // description Brief description of the Server, string
	DeviceGroups           []DeviceGroup       `json:"deviceGroups,omitempty"`           // deviceGroups An array of device groups associated with the Server
	DiscoveredDate         string              `json:"discoveredDate,omitempty"`         // discoveredDate Date and time when the Server was discovered. Same as created date
	ETAG                   string              `json:"eTag,omitempty"`                   // eTag Entity tag/version ID of the resource
	Facility               string              `json:"facility,omitempty"`               // facility A facility represents the collection of servers. A facility can be all or part of a data center, Server room, or computer lab. Facilities are used as security boundaries with user groups
	HardwareModel          string              `json:"hardwareModel,omitempty"`          // hardwareModel The model name of the target server
	HostName               string              `json:"hostName,omitempty"`               // hostName The name of the server as reported by the server
	ILO                    *Ilo                `json:"ilo,omitempty"`                    // information on ilo
	Interfaces             []Interface         `json:"interfaces,omitempty"`             // list of interfaces
	JobsHistory            []JobHistory        `json:"jobsHistory,omitempty"`            // array of previous run jobs
	LastScannedDate        string              `json:"lastScannedDate,omitempty"`        // lastScannedDate Date and time when the Server was detected last , string
	Locale                 string              `json:"locale,omitempty"`                 // locale Server's configured locale , string
	LoopbackIP             string              `json:"loopbackIP,omitempty"`             // loopbackIP Server's loopback IP address in dotted decimal format, string
	ManagementIP           string              `json:"managementIP,omitempty"`           // managementIP Server's management IP address in dotted decimal format, string
	Manufacturer           string              `json:"manufacturer,omitempty"`           // manufacturer Manufacturer as reported by the Server  , string
	MID                    string              `json:"mid,omitempty"`                    // mid A unique ID assigned to the Server by Server Automation, string
	Modified               string              `json:"modified,omitempty"`               // modified Date and time when the Server was last modified , timestamp
	Name                   string              `json:"name,omitempty"`                   // name The display name of the server. This is what shows on the left hand side of the UI. It is not the same as the host name. , string
	NetBios                string              `json:"netBios,omitempty"`                // netBios Server's Net BIOS name, string
	OperatingSystem        string              `json:"operatingSystem,omitempty"`        // operatingSystem Operating system installed on the Server, string
	OperatingSystemVersion string              `json:"operatingSystemVersion,omitempty"` // operatingSystemVersion Version of the operating system installed on the Server, string
	OpswLifecycle          string              `json:"opswLifecycle,omitempty"`          // Use type OpswLifecycle
	OSFlavor               string              `json:"osFlavor,omitempty"`               // osFlavor Additional information about an operating system flavor, string
	OSSPVersion            string              `json:"osSPVersion,omitempty"`            // osSPVersion Windows Service Pack version info, string
	PeerIP                 string              `json:"peerIP,omitempty"`                 // peerIP Server's peer IP address, string
	RAM                    string              `json:"ram,omitempty"`                    // ram Amount of free memory on the Server, string
	Reporting              bool                `json:"reporting,omitempty"`              // reporting Flag that indicates if the client on the Server is reporting to the core, Boolean
	Running                string              `json:"running,omitempty"`                // running Flag that indicates whether provisioning is performed on the Server, string
	SerialNumber           string              `json:"serialNumber,omitempty"`           // serialNumber The serial number assigned to the Server, string
	ServerLocation         *ServerLocationItem `json:"serverLocation,omitempty"`         // serverLocation The Server location information such as rack and enclosure etc
	Stage                  string              `json:"stage,omitempty"`                  // stage type //stage When a managed Server is rolled out into production it typically passes to various stages of deployment.The following are the valid values for the stages of the Server:
	State                  string              `json:"state,omitempty"`                  // state Indicates the state of the agent on the target server. The following are the valid values for the state:
	Status                 string              `json:"status,omitempty"`                 // status Unified status of the target Server. Supported values:
	StorageDevices         []StorageDevice     `json:"storageDevices,omitempty"`         // storage devices on the server
	Swap                   string              `json:"swap,omitempty"`                   // swap Amount of swap space on the Server  , string
	Type                   string              `json:"type,omitempty"`                   // type Uniquely identifies the type of the JSON object  , string (readonly)
	URI                    utils.Nstring       `json:"uri,omitempty"`                    // uri The canonical URI of the Server  , string
	UUID                   string              `json:"uuid,omitempty"`                   // uuid Server's UUID  , string
}

// Clone server profile so we can submit write attributes
func (s Server) Clone() Server {
	var ca []CustomAttribute
	for _, c := range s.CustomAttributes {
		ca = append(ca, c)
	}

	return Server{
		Description:      s.Description,
		CustomAttributes: ca,
		Name:             s.Name,
		Type:             s.Type,
	}
}

// GetInterfaces get a list of interfaces that have an ip address assigned
// usually called durring provisioning, and before we apply an os build plan
func (s Server) GetInterfaces() (interfaces []Interface) {
	for _, inrface := range s.Interfaces {
		interfaces = append(interfaces, inrface)
	}
	return interfaces
}

// GetInterface get the interface from slot location
func (s Server) GetInterface(slotid int) (Interface, error) {
	var interfac Interface
	inets := s.GetInterfaces()
	log.Debugf("inets -> %+v", inets)
	for i, inet := range inets {
		if i == slotid {
			return inet, nil
		}
	}
	return interfac, errors.New("Error interface slotid not found please try another interface id.")
}

// GetInterfaceFromMac get the server interface for mac address
func (s Server) GetInterfaceFromMac(mac string) (Interface, error) {
	var intface Interface
	for _, ife := range s.Interfaces {
		if strings.ToLower(ife.MACAddr) == strings.ToLower(mac) {
			intface = ife
			return intface, nil
		}
	}
	return intface, errors.New("Error interface not found, please try a different mac address.")
}

// GetPublicIPV4 returns the public ip interface
//                 usually called after an os build plan is applied
func (s Server) GetPublicIPV4() (string, error) {
	var position int
	position, inetItem := s.GetValueItem("public_ip", "server")
	if position >= 0 {
		log.Debugf("getting ip from public_ip -> %+v", inetItem.Value)
		if inetItem.Value != "" {
			return inetItem.Value, nil
		}
	}

	log.Debugf("GetPublicIPV4 from GetPublicInterface()")
	inet, err := s.GetPublicInterface()
	if err != nil {
		return "", err
	}
	log.Debugf("inet -> %+v", inet)
	if len(inet.IPV4Addr) > 0 {
		return inet.IPV4Addr, nil
	}
	return "", nil
}

// GetPublicInterface - get public interface from public_interface for server
func (s Server) GetPublicInterface() (*Interface, error) {
	var inet *Interface
	var err error
	position, inetItem := s.GetValueItem("public_interface", "server")
	if position >= 0 {
		inetJSON := inetItem.Value
		log.Debugf("GetPublicInterface -> %s", inetJSON)
		if len(inetJSON) > 0 {
			err = json.Unmarshal([]byte(inetJSON), &inet)
			return inet, err
		}
	}
	return inet, errors.New("Error public_interface custom attribute is not found.")
}

// ReloadFull GetServers() only returns a partial object, reload it to get everything
func (s Server) ReloadFull(c *ICSPClient) (Server, error) {
	var uri = s.URI
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.GET, uri.String(), nil)
	if err != nil {
		return s, err
	}

	log.Debugf("ReloadFull %s", data)
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return s, err
	}
	return s, nil
}

// ServerList List of Servers
type ServerList struct {
	Category    string        `json:"category,omitempty"`    // Resource category used for authorizations and resource type groupings
	Count       int           `json:"count,omitempty"`       // The actual number of resources returned in the specified page
	Created     string        `json:"created,omitempty"`     // timestamp for when resource was created
	ETAG        string        `json:"eTag,omitempty"`        // entity tag version id
	Members     []Server      `json:"members,omitempty"`     // array of Server types
	Modified    string        `json:"modified,omitempty"`    // timestamp resource last modified
	NextPageURI utils.Nstring `json:"nextPageUri,omitempty"` // Next page resources
	PrevPageURI utils.Nstring `json:"prevPageUri,omitempty"` // Previous page resources
	Start       int           `json:"start,omitempty"`       // starting row of resource for current page
	Total       int           `json:"total,omitempty"`       // total number of pages
	Type        string        `json:"type,omitempty"`        // type of paging
	URI         utils.Nstring `json:"uri,omitempty"`         // uri to page
}

// ServerCreate structure for create server
type ServerCreate struct {
	Type      string `json:"type,omitempty"`      // OSDIlo
	IPAddress string `json:"ipAddress,omitempty"` // PXE managed ip address
	Port      int    `json:"port,omitempty"`      // port number to use
	UserName  string `json:"username,omitempty"`  // iLo username
	Password  string `json:"password,omitempty"`  // iLO password
}

// NewServerCreate make a new servercreate object
func (sc ServerCreate) NewServerCreate(user string, pass string, ip string, port int) ServerCreate {
	if user == "" {
		log.Error("ilo user missing, please specify with ONEVIEW_ILO_USER or --oneview-ilo-user arguments.")
	}
	if user == "" {
		log.Error("ilo password missing, please specify with ONEVIEW_ILO_PASSWORD or --oneview-ilo-password arguments.")
	}
	return ServerCreate{
		// Type:      "OSDIlo", //TODO: this causes notmal os-deployment-servers actions to fail.
		UserName:  user,
		Password:  pass,
		IPAddress: ip,
		Port:      port,
	}
}

// SubmitNewServer submit new profile template
func (c *ICSPClient) SubmitNewServer(sc ServerCreate) (jt *JobTask, err error) {
	log.Infof("Initializing creation of server for ICSP, %s.", sc.IPAddress)
	var (
		uri = "/rest/os-deployment-servers"
		// uri  = "/rest/os-deployment-ilos" //TODO: implement hidden api for server deploy that works in Houston
		juri ODSUri
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	// c.SetAuthHeaderOptions(c.GetAuthHeaderMapNoVer()) //TODO: only needed when using os-deployment-ilos

	jt = jt.NewJobTask(c)
	jt.Reset()
	data, err := c.RestAPICall(rest.POST, uri, sc)
	if err != nil {
		jt.IsDone = true
		log.Errorf("Error submitting new server request: %s", err)
		return jt, err
	}

	log.Debugf("Response submit new server %s", data)
	if err := json.Unmarshal([]byte(data), &juri); err != nil {
		jt.IsDone = true
		jt.JobURI = juri
		log.Errorf("Error with task un-marshal: %s", err)
		return jt, err
	}
	jt.JobURI = juri

	return jt, err
}

// CreateServer create profile from template
func (c *ICSPClient) CreateServer(user string, pass string, ip string, port int) error {

	var sc ServerCreate
	sc = sc.NewServerCreate(user, pass, ip, port)

	jt, err := c.SubmitNewServer(sc)
	if err != nil {
		return err
	}
	err = jt.Wait()
	if err != nil {
		return err
	}
	return nil
}

// GetServers get a servers from icsp
func (c *ICSPClient) GetServers() (ServerList, error) {
	var (
		uri     = "/rest/os-deployment-servers"
		servers ServerList
	)

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.GET, uri, nil)
	if err != nil {
		return servers, err
	}

	log.Debugf("GetServers %s", data)
	if err := json.Unmarshal([]byte(data), &servers); err != nil {
		return servers, err
	}
	return servers, nil
}

// GetServerByIP use the server ip to get the server
func (c *ICSPClient) GetServerByIP(ip string) (Server, error) {
	var (
		servers ServerList
		server  Server
	)
	servers, err := c.GetServers()
	if err != nil {
		return server, err
	}
	log.Debugf("GetServerByIP: server count: %d", servers.Count)
	// grab the target
	var srv Server
	for _, randServer := range servers.Members {
		server, err := c.GetServerByID(randServer.MID)
		if err != nil {
			return server, err
		}
		if strings.EqualFold(server.ILO.IPAddress, ip) {
			log.Debugf("server ip: %v", &server.ILO.IPAddress)
			srv = server
			srv, err = srv.ReloadFull(c)
			if err != nil {
				return srv, err
			}
			break
		}

	}
	return srv, nil
}

// GetServerByID - get a server from icsp - faster then getting all servers
// and ranging over them
func (c *ICSPClient) GetServerByID(mid string) (Server, error) {
	var (
		uri    = fmt.Sprintf("/rest/os-deployment-servers/%v", mid)
		server Server
	)

	log.Debugf("GetServer uri %s", uri)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.GET, uri, nil)
	if err != nil {
		return server, err
	}

	log.Debugf("GetServer %s", data)
	if err := json.Unmarshal([]byte(data), &server); err != nil {
		return server, err
	}
	return server, nil
}

// GetServerByName use the server name to get the server type
func (c *ICSPClient) GetServerByName(name string) (Server, error) {
	var (
		servers ServerList
		server  Server
	)
	servers, err := c.GetServers()
	if err != nil {
		return server, err
	}
	log.Debugf("GetServerByName: server count: %d", servers.Count)
	// grab the target
	var srv Server
	for _, server := range servers.Members {
		if strings.EqualFold(server.Name, name) {
			log.Debugf("server name: %v", server.Name)
			srv = server
			srv, err = srv.ReloadFull(c)
			if err != nil {
				return srv, err
			}
			break
		}
	}
	return srv, nil
}

//GetServerByHostName use the server hostname automatically assigned to get the server
func (c *ICSPClient) GetServerByHostName(hostname string) (Server, error) {
	var (
		servers ServerList
		server  Server
	)
	servers, err := c.GetServers()
	if err != nil {
		return server, err
	}
	log.Debugf("GetServerByHostName: server count: %d", servers.Count)
	// grab the target
	var srv Server
	for _, server := range servers.Members {
		log.Debugf("server host: %v", server.HostName)
		if strings.EqualFold(server.HostName, hostname) {
			log.Debugf("found server host: %v", server.HostName)
			srv = server
			srv, err = srv.ReloadFull(c)
			if err != nil {
				return srv, err
			}
			break
		}
	}
	return srv, nil
}

//GetServerBySerialNumber use the serial number to find the server
func (c *ICSPClient) GetServerBySerialNumber(serial string) (Server, error) {
	var (
		servers ServerList
		server  Server
	)
	servers, err := c.GetServers()
	if err != nil {
		return server, err
	}
	log.Debugf("GetServerBySerialNumber: server count: %d, serialnumber: %s", servers.Count, serial)
	// grab the target
	var srv Server
	for _, server := range servers.Members {
		log.Debugf("server: %v, serial : %v", server.HostName, server.SerialNumber)
		if strings.EqualFold(server.SerialNumber, serial) {
			log.Debugf("found server host: %v", server.HostName)
			srv = server
			srv, err = srv.ReloadFull(c)
			if err != nil {
				return srv, err
			}
			break
		}
	}
	return srv, nil
}

// IsServerManaged - returns true if server is managed
func (c *ICSPClient) IsServerManaged(serial string) (bool, error) {
	data, err := c.GetServerBySerialNumber(serial)
	if err != nil {
		return false, err
	}
	log.Debugf("found server host: %v, serial: %v cycle: %v", data.HostName, data.SerialNumber, data.OpswLifecycle)
	return strings.EqualFold(data.OpswLifecycle, Managed.String()), err
}

// DeleteServer - deletes a server in icsp appliance instance
func (c *ICSPClient) DeleteServer(mid string) (bool, error) {
	var (
		uri   = fmt.Sprintf("/rest/os-deployment-servers/%v", mid)
		isDel bool
	)

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	//TODO: should check to make sure server uri has a real server
	//      and it's status is managed

	_, err := c.RestAPICall(rest.DELETE, uri, nil)

	if err != nil {
		return isDel, err
	}
	isDel = true // at this point server was deleted
	//As per API
	//HTTP/1.1 204 No Content
	//Content-Type: application/json
	//...so,lets tell the consumer things went well :)
	return isDel, err
}

// SaveServer save Server, submit new profile template
func (c *ICSPClient) SaveServer(s Server) (o Server, err error) {
	log.Infof("Saving server attributes for %s.", s.Name)
	var (
		uri = s.URI
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	log.Debugf("name -> %s, description -> %s", s.Name, s.Description)
	log.Debugf("CustomAttributes -> %+v", s.CustomAttributes)

	sc := s.Clone()

	log.Debugf("options -> %+v", c.Option)
	log.Debugf("REST : %s \n %+v\n", uri, sc)
	data, err := c.RestAPICall(rest.PUT, uri.String(), sc)
	if err != nil {
		log.Errorf("Error submitting new server request: %s", err)
		return o, err
	}
	if err := json.Unmarshal([]byte(data), &o); err != nil {
		return o, err
	}

	return o, err
}
