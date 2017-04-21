package oneandone

import "net/http"

type Role struct {
	Identity
	descField
	CreationDate string       `json:"creation_date,omitempty"`
	State        string       `json:"state,omitempty"`
	Default      *int         `json:"default,omitempty"`
	Permissions  *Permissions `json:"permissions,omitempty"`
	Users        []Identity   `json:"users,omitempty"`
	ApiPtr
}

type Permissions struct {
	Backups         *BackupPerm         `json:"backups,omitempty"`
	Firewalls       *FirewallPerm       `json:"firewall_policies,omitempty"`
	Images          *ImagePerm          `json:"images,omitempty"`
	Invoice         *InvoicePerm        `json:"interactive_invoices,omitempty"`
	IPs             *IPPerm             `json:"public_ips,omitempty"`
	LoadBalancers   *LoadBalancerPerm   `json:"load_balancers,omitempty"`
	Logs            *LogPerm            `json:"logs,omitempty"`
	MonitorCenter   *MonitorCenterPerm  `json:"monitoring_center,omitempty"`
	MonitorPolicies *MonitorPolicyPerm  `json:"monitoring_policies,omitempty"`
	PrivateNetworks *PrivateNetworkPerm `json:"private_networks,omitempty"`
	Roles           *RolePerm           `json:"roles,omitempty"`
	Servers         *ServerPerm         `json:"servers,omitempty"`
	SharedStorage   *SharedStoragePerm  `json:"shared_storages,omitempty"`
	Usages          *UsagePerm          `json:"usages,omitempty"`
	Users           *UserPerm           `json:"users,omitempty"`
	VPNs            *VPNPerm            `json:"vpn,omitempty"`
}

type BackupPerm struct {
	Create bool `json:"create"`
	Delete bool `json:"delete"`
	Show   bool `json:"show"`
}

type FirewallPerm struct {
	Clone                   bool `json:"clone"`
	Create                  bool `json:"create"`
	Delete                  bool `json:"delete"`
	ManageAttachedServerIPs bool `json:"manage_attached_server_ips"`
	ManageRules             bool `json:"manage_rules"`
	SetDescription          bool `json:"set_description"`
	SetName                 bool `json:"set_name"`
	Show                    bool `json:"show"`
}

type ImagePerm struct {
	Create            bool `json:"create"`
	Delete            bool `json:"delete"`
	DisableAutoCreate bool `json:"disable_automatic_creation"`
	SetDescription    bool `json:"set_description"`
	SetName           bool `json:"set_name"`
	Show              bool `json:"show"`
}

type InvoicePerm struct {
	Show bool `json:"show"`
}

type IPPerm struct {
	Create        bool `json:"create"`
	Delete        bool `json:"delete"`
	Release       bool `json:"release"`
	SetReverseDNS bool `json:"set_reverse_dns"`
	Show          bool `json:"show"`
}

type LoadBalancerPerm struct {
	Create                  bool `json:"create"`
	Delete                  bool `json:"delete"`
	ManageAttachedServerIPs bool `json:"manage_attached_server_ips"`
	ManageRules             bool `json:"manage_rules"`
	Modify                  bool `json:"modify"`
	SetDescription          bool `json:"set_description"`
	SetName                 bool `json:"set_name"`
	Show                    bool `json:"show"`
}

type LogPerm struct {
	Show bool `json:"show"`
}

type MonitorCenterPerm struct {
	Show bool `json:"show"`
}

type MonitorPolicyPerm struct {
	Clone                 bool `json:"clone"`
	Create                bool `json:"create"`
	Delete                bool `json:"delete"`
	ManageAttachedServers bool `json:"manage_attached_servers"`
	ManagePorts           bool `json:"manage_ports"`
	ManageProcesses       bool `json:"manage_processes"`
	ModifyResources       bool `json:"modify_resources"`
	SetDescription        bool `json:"set_description"`
	SetEmail              bool `json:"set_email"`
	SetName               bool `json:"set_name"`
	Show                  bool `json:"show"`
}

type PrivateNetworkPerm struct {
	Create                bool `json:"create"`
	Delete                bool `json:"delete"`
	ManageAttachedServers bool `json:"manage_attached_servers"`
	SetDescription        bool `json:"set_description"`
	SetName               bool `json:"set_name"`
	SetNetworkInfo        bool `json:"set_network_info"`
	Show                  bool `json:"show"`
}

type RolePerm struct {
	Clone          bool `json:"clone"`
	Create         bool `json:"create"`
	Delete         bool `json:"delete"`
	ManageUsers    bool `json:"manage_users"`
	Modify         bool `json:"modify"`
	SetDescription bool `json:"set_description"`
	SetName        bool `json:"set_name"`
	Show           bool `json:"show"`
}

type ServerPerm struct {
	AccessKVMConsole bool `json:"access_kvm_console"`
	AssignIP         bool `json:"assign_ip"`
	Clone            bool `json:"clone"`
	Create           bool `json:"create"`
	Delete           bool `json:"delete"`
	ManageDVD        bool `json:"manage_dvd"`
	ManageSnapshot   bool `json:"manage_snapshot"`
	Reinstall        bool `json:"reinstall"`
	Resize           bool `json:"resize"`
	Restart          bool `json:"restart"`
	SetDescription   bool `json:"set_description"`
	SetName          bool `json:"set_name"`
	Show             bool `json:"show"`
	Shutdown         bool `json:"shutdown"`
	Start            bool `json:"start"`
}

type SharedStoragePerm struct {
	Access                bool `json:"access"`
	Create                bool `json:"create"`
	Delete                bool `json:"delete"`
	ManageAttachedServers bool `json:"manage_attached_servers"`
	Resize                bool `json:"resize"`
	SetDescription        bool `json:"set_description"`
	SetName               bool `json:"set_name"`
	Show                  bool `json:"show"`
}

type UsagePerm struct {
	Show bool `json:"show"`
}

type UserPerm struct {
	ChangeRole     bool `json:"change_role"`
	Create         bool `json:"create"`
	Delete         bool `json:"delete"`
	Disable        bool `json:"disable"`
	Enable         bool `json:"enable"`
	ManageAPI      bool `json:"manage_api"`
	SetDescription bool `json:"set_description"`
	SetEmail       bool `json:"set_email"`
	SetPassword    bool `json:"set_password"`
	Show           bool `json:"show"`
}

type VPNPerm struct {
	Create         bool `json:"create"`
	Delete         bool `json:"delete"`
	DownloadFile   bool `json:"download_file"`
	SetDescription bool `json:"set_description"`
	SetName        bool `json:"set_name"`
	Show           bool `json:"show"`
}

// GET /roles
func (api *API) ListRoles(args ...interface{}) ([]Role, error) {
	url, err := processQueryParams(createUrl(api, rolePathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []Role{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for _, role := range result {
		role.api = api
	}
	return result, nil
}

// POST /roles
func (api *API) CreateRole(name string) (string, *Role, error) {
	result := new(Role)
	url := createUrl(api, rolePathSegment)
	req := struct {
		Name string `json:"name"`
	}{name}
	err := api.Client.Post(url, &req, &result, http.StatusCreated)
	if err != nil {
		return "", nil, err
	}
	result.api = api
	return result.Id, result, nil
}

// GET /roles/{role_id}
func (api *API) GetRole(role_id string) (*Role, error) {
	result := new(Role)
	url := createUrl(api, rolePathSegment, role_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /roles/{role_id}
func (api *API) ModifyRole(role_id string, name string, description string, state string) (*Role, error) {
	result := new(Role)
	url := createUrl(api, rolePathSegment, role_id)
	req := struct {
		Name        string `json:"name,omitempty"`
		Description string `json:"description,omitempty"`
		State       string `json:"state,omitempty"`
	}{Name: name, Description: description, State: state}
	err := api.Client.Put(url, &req, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /roles/{role_id}
func (api *API) DeleteRole(role_id string) (*Role, error) {
	result := new(Role)
	url := createUrl(api, rolePathSegment, role_id)
	err := api.Client.Delete(url, nil, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /roles/{role_id}/permissions
func (api *API) GetRolePermissions(role_id string) (*Permissions, error) {
	result := new(Permissions)
	url := createUrl(api, rolePathSegment, role_id, "permissions")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// PUT /roles/{role_id}/permissions
func (api *API) ModifyRolePermissions(role_id string, perm *Permissions) (*Role, error) {
	result := new(Role)
	url := createUrl(api, rolePathSegment, role_id, "permissions")
	err := api.Client.Put(url, &perm, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /roles/{role_id}/users
func (api *API) ListRoleUsers(role_id string) ([]Identity, error) {
	result := []Identity{}
	url := createUrl(api, rolePathSegment, role_id, "users")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /roles/{role_id}/users
func (api *API) AssignRoleUsers(role_id string, user_ids []string) (*Role, error) {
	result := new(Role)
	url := createUrl(api, rolePathSegment, role_id, "users")
	req := struct {
		Users []string `json:"users"`
	}{user_ids}
	err := api.Client.Post(url, &req, &result, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /roles/{role_id}/users/{user_id}
func (api *API) GetRoleUser(role_id string, user_id string) (*Identity, error) {
	result := new(Identity)
	url := createUrl(api, rolePathSegment, role_id, "users", user_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DELETE /roles/{role_id}/users/{user_id}
func (api *API) RemoveRoleUser(role_id string, user_id string) (*Role, error) {
	result := new(Role)
	url := createUrl(api, rolePathSegment, role_id, "users", user_id)
	err := api.Client.Delete(url, nil, &result, http.StatusAccepted)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// POST /roles/{role_id}/clone
func (api *API) CloneRole(role_id string, name string) (*Role, error) {
	result := new(Role)
	url := createUrl(api, rolePathSegment, role_id, "clone")
	req := struct {
		Name string `json:"name"`
	}{name}
	err := api.Client.Post(url, &req, &result, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

func (role *Role) GetState() (string, error) {
	in, err := role.api.GetRole(role.Id)
	if in == nil {
		return "", err
	}
	return in.State, err
}

// Sets all backups' permissions
func (bp *BackupPerm) SetAll(value bool) {
	bp.Create = value
	bp.Delete = value
	bp.Show = value
}

// Sets all firewall policies' permissions
func (fp *FirewallPerm) SetAll(value bool) {
	fp.Clone = value
	fp.Create = value
	fp.Delete = value
	fp.ManageAttachedServerIPs = value
	fp.ManageRules = value
	fp.SetDescription = value
	fp.SetName = value
	fp.Show = value
}

// Sets all images' permissions
func (imp *ImagePerm) SetAll(value bool) {
	imp.Create = value
	imp.Delete = value
	imp.DisableAutoCreate = value
	imp.SetDescription = value
	imp.SetName = value
	imp.Show = value
}

// Sets all invoice's permissions
func (inp *InvoicePerm) SetAll(value bool) {
	inp.Show = value
}

// Sets all IPs' permissions
func (ipp *IPPerm) SetAll(value bool) {
	ipp.Create = value
	ipp.Delete = value
	ipp.Release = value
	ipp.SetReverseDNS = value
	ipp.Show = value
}

// Sets all load balancers' permissions
func (lbp *LoadBalancerPerm) SetAll(value bool) {
	lbp.Create = value
	lbp.Delete = value
	lbp.ManageAttachedServerIPs = value
	lbp.ManageRules = value
	lbp.Modify = value
	lbp.SetDescription = value
	lbp.SetName = value
	lbp.Show = value
}

// Sets all logs' permissions
func (lp *LogPerm) SetAll(value bool) {
	lp.Show = value
}

// Sets all monitoring center's permissions
func (mcp *MonitorCenterPerm) SetAll(value bool) {
	mcp.Show = value
}

// Sets all monitoring policies' permissions
func (mpp *MonitorPolicyPerm) SetAll(value bool) {
	mpp.Clone = value
	mpp.Create = value
	mpp.Delete = value
	mpp.ManageAttachedServers = value
	mpp.ManagePorts = value
	mpp.ManageProcesses = value
	mpp.ModifyResources = value
	mpp.SetDescription = value
	mpp.SetEmail = value
	mpp.SetName = value
	mpp.Show = value
}

// Sets all private networks' permissions
func (pnp *PrivateNetworkPerm) SetAll(value bool) {
	pnp.Create = value
	pnp.Delete = value
	pnp.ManageAttachedServers = value
	pnp.SetDescription = value
	pnp.SetName = value
	pnp.SetNetworkInfo = value
	pnp.Show = value
}

// Sets all roles' permissions
func (rp *RolePerm) SetAll(value bool) {
	rp.Clone = value
	rp.Create = value
	rp.Delete = value
	rp.ManageUsers = value
	rp.Modify = value
	rp.SetDescription = value
	rp.SetName = value
	rp.Show = value
}

// Sets all servers' permissions
func (sp *ServerPerm) SetAll(value bool) {
	sp.AccessKVMConsole = value
	sp.AssignIP = value
	sp.Clone = value
	sp.Create = value
	sp.Delete = value
	sp.ManageDVD = value
	sp.ManageSnapshot = value
	sp.Reinstall = value
	sp.Resize = value
	sp.Restart = value
	sp.SetDescription = value
	sp.SetName = value
	sp.Show = value
	sp.Shutdown = value
	sp.Start = value
}

// Sets all shared storages' permissions
func (ssp *SharedStoragePerm) SetAll(value bool) {
	ssp.Access = value
	ssp.Create = value
	ssp.Delete = value
	ssp.ManageAttachedServers = value
	ssp.Resize = value
	ssp.SetDescription = value
	ssp.SetName = value
	ssp.Show = value
}

// Sets all usages' permissions
func (up *UsagePerm) SetAll(value bool) {
	up.Show = value
}

// Sets all users' permissions
func (up *UserPerm) SetAll(value bool) {
	up.ChangeRole = value
	up.Create = value
	up.Delete = value
	up.Disable = value
	up.Enable = value
	up.ManageAPI = value
	up.SetDescription = value
	up.SetEmail = value
	up.SetPassword = value
	up.Show = value
}

// Sets all VPNs' permissions
func (vpnp *VPNPerm) SetAll(value bool) {
	vpnp.Create = value
	vpnp.Delete = value
	vpnp.DownloadFile = value
	vpnp.SetDescription = value
	vpnp.SetName = value
	vpnp.Show = value
}

// Sets all available permissions
func (p *Permissions) SetAll(v bool) {
	if p.Backups == nil {
		p.Backups = &BackupPerm{v, v, v}
	} else {
		p.Backups.SetAll(v)
	}
	if p.Firewalls == nil {
		p.Firewalls = &FirewallPerm{v, v, v, v, v, v, v, v}
	} else {
		p.Firewalls.SetAll(v)
	}
	if p.Images == nil {
		p.Images = &ImagePerm{v, v, v, v, v, v}
	} else {
		p.Images.SetAll(v)
	}
	if p.Invoice == nil {
		p.Invoice = &InvoicePerm{v}
	} else {
		p.Invoice.SetAll(v)
	}
	if p.IPs == nil {
		p.IPs = &IPPerm{v, v, v, v, v}
	} else {
		p.IPs.SetAll(v)
	}
	if p.LoadBalancers == nil {
		p.LoadBalancers = &LoadBalancerPerm{v, v, v, v, v, v, v, v}
	} else {
		p.LoadBalancers.SetAll(v)
	}
	if p.Logs == nil {
		p.Logs = &LogPerm{v}
	} else {
		p.Logs.SetAll(v)
	}
	if p.MonitorCenter == nil {
		p.MonitorCenter = &MonitorCenterPerm{v}
	} else {
		p.MonitorCenter.SetAll(v)
	}
	if p.MonitorPolicies == nil {
		p.MonitorPolicies = &MonitorPolicyPerm{v, v, v, v, v, v, v, v, v, v, v}
	} else {
		p.MonitorPolicies.SetAll(v)
	}
	if p.PrivateNetworks == nil {
		p.PrivateNetworks = &PrivateNetworkPerm{v, v, v, v, v, v, v}
	} else {
		p.PrivateNetworks.SetAll(v)
	}
	if p.Roles == nil {
		p.Roles = &RolePerm{v, v, v, v, v, v, v, v}
	} else {
		p.Roles.SetAll(v)
	}
	if p.Servers == nil {
		p.Servers = &ServerPerm{v, v, v, v, v, v, v, v, v, v, v, v, v, v, v}
	} else {
		p.Servers.SetAll(v)
	}
	if p.SharedStorage == nil {
		p.SharedStorage = &SharedStoragePerm{v, v, v, v, v, v, v, v}
	} else {
		p.SharedStorage.SetAll(v)
	}
	if p.Usages == nil {
		p.Usages = &UsagePerm{v}
	} else {
		p.Usages.SetAll(v)
	}
	if p.Users == nil {
		p.Users = &UserPerm{v, v, v, v, v, v, v, v, v, v}
	} else {
		p.Users.SetAll(v)
	}
	if p.VPNs == nil {
		p.VPNs = &VPNPerm{v, v, v, v, v, v}
	} else {
		p.VPNs.SetAll(v)
	}
}
