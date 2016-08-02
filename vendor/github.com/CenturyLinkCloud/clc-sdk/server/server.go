package server

import (
	"fmt"
	"regexp"
	"time"

	"github.com/CenturyLinkCloud/clc-sdk/api"
	"github.com/CenturyLinkCloud/clc-sdk/status"
)

var (
	ErrInvalidServer = fmt.Errorf("server: server missing required field(s). (Name, CPU, MemoryGB, GroupID, SourceServerID, Type)")
)

func New(client api.HTTP) *Service {
	return &Service{
		client: client,
		config: client.Config(),
	}
}

type Service struct {
	client api.HTTP
	config *api.Config
}

func (s *Service) Get(name string) (*Response, error) {
	url := fmt.Sprintf("%s/servers/%s/%s", s.config.BaseURL, s.config.Alias, name)
	if regexp.MustCompile("^[0-9a-f]{32}$").MatchString(name) {
		url = fmt.Sprintf("%s?uuid=true", url)
	}
	resp := &Response{}
	err := s.client.Get(url, resp)
	return resp, err
}

func (s *Service) Create(server Server) (*status.QueuedResponse, error) {
	if !server.Valid() {
		return nil, ErrInvalidServer
	}

	resp := &status.QueuedResponse{}
	url := fmt.Sprintf("%s/servers/%s", s.config.BaseURL, s.config.Alias)
	err := s.client.Post(url, server, resp)
	return resp, err
}

func (s *Service) Update(name string, updates ...api.Update) (*status.Status, error) {
	resp := &status.Status{}
	url := fmt.Sprintf("%s/servers/%s/%s", s.config.BaseURL, s.config.Alias, name)
	err := s.client.Patch(url, updates, resp)
	return resp, err
}

func (s *Service) Edit(name string, updates ...api.Update) error {
	url := fmt.Sprintf("%s/servers/%s/%s", s.config.BaseURL, s.config.Alias, name)
	err := s.client.Patch(url, updates, nil)
	return err
}

func (s *Service) Delete(name string) (*status.QueuedResponse, error) {
	url := fmt.Sprintf("%s/servers/%s/%s", s.config.BaseURL, s.config.Alias, name)
	resp := &status.QueuedResponse{}
	err := s.client.Delete(url, resp)
	return resp, err
}

func (s *Service) GetCredentials(name string) (Credentials, error) {
	url := fmt.Sprintf("%s/servers/%s/%s/credentials", s.config.BaseURL, s.config.Alias, name)
	resp := Credentials{}
	err := s.client.Get(url, &resp)
	return resp, err
}

type Credentials struct {
	Username string `json:"userName"`
	Password string `json:"password"`
}

func (s *Service) Archive(servers ...string) ([]*status.QueuedResponse, error) {
	url := fmt.Sprintf("%s/operations/%s/servers/archive", s.config.BaseURL, s.config.Alias)
	var resp []*status.QueuedResponse
	err := s.client.Post(url, servers, &resp)
	return resp, err
}

func (s *Service) Restore(name, group string) (*status.Status, error) {
	restore := map[string]string{"targetGroupId": group}
	url := fmt.Sprintf("%s/servers/%s/%s/restore", s.config.BaseURL, s.config.Alias, name)
	resp := &status.Status{}
	err := s.client.Post(url, restore, resp)
	return resp, err
}

func (s *Service) CreateSnapshot(expiration int, servers ...string) ([]*status.QueuedResponse, error) {
	snapshot := Snapshot{Expiration: expiration, Servers: servers}
	url := fmt.Sprintf("%s/operations/%s/servers/createSnapshot", s.config.BaseURL, s.config.Alias)
	var resp []*status.QueuedResponse
	err := s.client.Post(url, snapshot, &resp)
	return resp, err
}

func (s *Service) DeleteSnapshot(server, id string) (*status.Status, error) {
	url := fmt.Sprintf("%s/servers/%s/%s/snapshots/%s", s.config.BaseURL, s.config.Alias, server, id)
	resp := &status.Status{}
	err := s.client.Delete(url, resp)
	return resp, err
}

func (s *Service) RevertSnapshot(server, id string) (*status.Status, error) {
	url := fmt.Sprintf("%s/servers/%s/%s/snapshots/%s/restore", s.config.BaseURL, s.config.Alias, server, id)
	resp := &status.Status{}
	err := s.client.Post(url, nil, resp)
	return resp, err
}

type Snapshot struct {
	Expiration int      `json:"snapshotExpirationDays"`
	Servers    []string `json:"serverIds"`
}

func (s *Service) ExecutePackage(pkg Package, servers ...string) ([]*status.QueuedResponse, error) {
	url := fmt.Sprintf("%s/operations/%s/servers/executePackage", s.config.BaseURL, s.config.Alias)
	var resp []*status.QueuedResponse
	exec := executePackage{Servers: servers, Package: pkg}
	err := s.client.Post(url, exec, &resp)
	return resp, err
}

type executePackage struct {
	Servers []string `json:"servers"`
	Package Package  `json:"package"`
}

type Package struct {
	ID     string            `json:"packageId"`
	Params map[string]string `json:"parameters"`
}

func (s *Service) PowerState(state PowerState, servers ...string) ([]*status.QueuedResponse, error) {
	url := fmt.Sprintf("%s/operations/%s/servers/%s", s.config.BaseURL, s.config.Alias, state)
	var resp []*status.QueuedResponse
	err := s.client.Post(url, servers, &resp)
	return resp, err
}

type PowerState int

const (
	On = iota
	Off
	Pause
	Reboot
	Reset
	ShutDown
	StartMaintenance
	StopMaintenance
)

func (p PowerState) String() string {
	switch p {
	case On:
		return "powerOn"
	case Off:
		return "powerOff"
	case Pause:
		return "pause"
	case Reboot:
		return "reboot"
	case Reset:
		return "reset"
	case ShutDown:
		return "shutDown"
	case StartMaintenance:
		return "startMaintenance"
	case StopMaintenance:
		return "stopMaintenance"
	}
	return ""
}

func (s *Service) GetPublicIP(name string, ip string) (*PublicIP, error) {
	url := fmt.Sprintf("%s/servers/%s/%s/publicIPAddresses/%s", s.config.BaseURL, s.config.Alias, name, ip)
	resp := &PublicIP{}
	err := s.client.Get(url, resp)
	return resp, err
}

func (s *Service) AddPublicIP(name string, ip PublicIP) (*status.Status, error) {
	url := fmt.Sprintf("%s/servers/%s/%s/publicIPAddresses", s.config.BaseURL, s.config.Alias, name)
	resp := &status.Status{}
	err := s.client.Post(url, ip, resp)
	return resp, err
}

func (s *Service) UpdatePublicIP(name string, public string, ip PublicIP) (*status.Status, error) {
	url := fmt.Sprintf("%s/servers/%s/%s/publicIPAddresses/%s", s.config.BaseURL, s.config.Alias, name, public)
	resp := &status.Status{}
	err := s.client.Put(url, ip, resp)
	return resp, err
}

func (s *Service) DeletePublicIP(name, ip string) (*status.Status, error) {
	url := fmt.Sprintf("%s/servers/%s/%s/publicIPAddresses/%s", s.config.BaseURL, s.config.Alias, name, ip)
	resp := &status.Status{}
	err := s.client.Delete(url, resp)
	return resp, err
}

type PublicIP struct {
	InternalIP         string              `json:"internalIPAddress,omitempty"`
	Ports              []Port              `json:"ports,omitempty"`
	SourceRestrictions []SourceRestriction `json:"sourceRestrictions,omitempty"`
}

type Port struct {
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	PortTo   int    `json:"portTo,omitempty"`
}

type SourceRestriction struct {
	CIDR string `json:"cidr"`
}

type Disk struct {
	DiskID string `json:"diskId,omitempty"`
	Path   string `json:"path,omitempty"`
	SizeGB int    `json:"sizeGB,omitempty"`
	Type   string `json:"type,omitempty"`
}

func (s *Service) AddSecondaryNetwork(name, networkId, ip string) (*status.Status, error) {
	url := fmt.Sprintf("%s/servers/%s/%s/networks", s.config.BaseURL, s.config.Alias, name)
	req := &SecondaryNetwork{
		NetworkID: networkId,
		IPAddress: ip,
	}
	// returned a non-standard status object, repackage into a proper one
	resp := &status.QueuedOperation{}
	err := s.client.Post(url, req, resp)
	if err != nil {
		return nil, err
	}
	return resp.Status(), nil
}

type SecondaryNetwork struct {
	NetworkID string `json:"networkId,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

func UpdateCPU(num int) api.Update {
	return api.Update{
		Op:     "set",
		Member: "cpu",
		Value:  num,
	}
}

func UpdateMemory(num int) api.Update {
	return api.Update{
		Op:     "set",
		Member: "memory",
		Value:  num,
	}
}

func UpdateCredentials(current, updated string) api.Update {
	return api.Update{
		Op:     "set",
		Member: "password",
		Value: struct {
			Current  string `json:"current"`
			Password string `json:"password"`
		}{
			current,
			updated,
		},
	}
}

func UpdateGroup(group string) api.Update {
	return api.Update{
		Op:     "set",
		Member: "groupId",
		Value:  group,
	}
}

func UpdateDescription(desc string) api.Update {
	return api.Update{
		Op:     "set",
		Member: "description",
		Value:  desc,
	}
}

func UpdateAdditionaldisks(disks []Disk) api.Update {
	return api.Update{
		Op:     "set",
		Member: "disks",
		Value:  disks,
	}
}

func UpdateCustomfields(fields []api.Customfields) api.Update {
	return api.Update{
		Op:     "set",
		Member: "customFields",
		Value:  fields,
	}
}

type Server struct {
	Name                 string             `json:"name"`
	Description          string             `json:"description,omitempty"`
	GroupID              string             `json:"groupId"`
	SourceServerID       string             `json:"sourceServerId"`
	IsManagedOS          bool               `json:"isManagedOS,omitempty"`
	IsManagedBackup      bool               `json:"isManagedBackup,omitempty"`
	PrimaryDNS           string             `json:"primaryDns,omitempty"`
	SecondaryDNS         string             `json:"secondaryDns,omitempty"`
	NetworkID            string             `json:"networkId,omitempty"`
	IPaddress            string             `json:"ipAddress,omitempty"`
	Password             string             `json:"password,omitempty"`
	SourceServerPassword string             `json:"sourceServerPassword,omitempty"`
	CPU                  int                `json:"cpu"`
	CPUAutoscalePolicyID string             `json:"cpuAutoscalePolicyId,omitempty"`
	MemoryGB             int                `json:"memoryGB"`
	Type                 string             `json:"type"`
	Storagetype          string             `json:"storageType,omitempty"`
	AntiAffinityPolicyID string             `json:"antiAffinityPolicyId,omitempty"`
	Customfields         []api.Customfields `json:"customFields,omitempty"`
	Additionaldisks      []Disk             `json:"additionalDisks,omitempty"`
	TTL                  *time.Time         `json:"ttl,omitempty"`
	Packages             []Package          `json:"packages,omitempty"`
	ConfigurationID      string             `json:"configurationId,omitempty"`
	OSType               string             `json:"osType,omitempty"`
}

func (s *Server) Valid() bool {
	return s.Name != "" && s.CPU != 0 && s.MemoryGB != 0 && s.GroupID != "" && s.SourceServerID != "" && s.Type != ""
}

type Response struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	GroupID     string `json:"groupId"`
	IsTemplate  bool   `json:"isTemplate"`
	LocationID  string `json:"locationId"`
	OStype      string `json:"osType"`
	Status      string `json:"status"`
	Details     struct {
		IPaddresses []struct {
			Internal string `json:"internal"`
			Public   string `json:"public"`
		} `json:"ipAddresses"`
		AlertPolicies []struct {
			ID    string    `json:"id"`
			Name  string    `json:"name"`
			Links api.Links `json:"links"`
		} `json:"alertPolicies"`
		CPU               int    `json:"cpu"`
		Diskcount         int    `json:"diskCount"`
		Hostname          string `json:"hostName"`
		InMaintenanceMode bool   `json:"inMaintenanceMode"`
		MemoryMB          int    `json:"memoryMB"`
		Powerstate        string `json:"powerState"`
		Storagegb         int    `json:"storageGB"`
		Disks             []struct {
			ID             string        `json:"id"`
			SizeGB         int           `json:"sizeGB"`
			PartitionPaths []interface{} `json:"partitionPaths"`
		} `json:"disks"`
		Partitions []struct {
			SizeGB float64 `json:"sizeGB"`
			Path   string  `json:"path"`
		} `json:"partitions"`
		Snapshots []struct {
			Name  string    `json:"name"`
			Links api.Links `json:"links"`
		} `json:"snapshots"`
		Customfields []api.Customfields `json:"customFields,omitempty"`
	} `json:"details"`
	Type        string `json:"type"`
	Storagetype string `json:"storageType"`
	ChangeInfo  struct {
		CreatedDate  string `json:"createdDate"`
		CreatedBy    string `json:"createdBy"`
		ModifiedDate string `json:"modifiedDate"`
		ModifiedBy   string `json:"modifiedBy"`
	} `json:"changeInfo"`
	Links api.Links `json:"links"`
}
