package dockercloud

type ActionListResponse struct {
	Meta    Meta     `json:"meta"`
	Objects []Action `json:"objects"`
}

type Action struct {
	Action       string `json:"action"`
	Body         string `json:"body"`
	End_date     string `json:"end_date"`
	Ip           string `json:"ip"`
	Location     string `json:"location"`
	Logs         string `json:"logs"`
	Method       string `json:"method"`
	Object       string `json:"object"`
	Path         string `json:"path"`
	Resource_uri string `json:"resource_uri"`
	Start_date   string `json:"start_date"`
	State        string `json:"state"`
	Uuid         string `json:"uuid"`
}

type AZListResponse struct {
	Meta    Meta `json:"meta"`
	Objects []AZ `json:"objects"`
}

type AZ struct {
	Available    bool   `json:"available"`
	Name         string `json:"name"`
	Resource_uri string `json:"resource_uri"`
}

type ContainerBinding struct {
	Container_path string `json:"container_path"`
	Host_path      string `json:"host_path"`
	Rewritable     bool   `json:"rewritable"`
	Volume         string `json:"volume"`
}

type Meta struct {
	Limit      int    `json:"limit"`
	Next       string `json:"next"`
	TotalCount int    `json:"total_count"`
}

type Metric struct {
	Cpu    float64 `json:"cpu"`
	Disk   float64 `json:"disk"`
	Memory float64 `json:"memory"`
}

type CListResponse struct {
	Meta    Meta        `json:"meta"`
	Objects []Container `json:"objects"`
}

type Container struct {
	Autodestroy         string              `json:"autodestroy"`
	Autorestart         string              `json:"autorestart"`
	Bindings            []ContainerBinding  `json:"bindings"`
	Container_envvars   []ContainerEnvvar   `json:"container_envvars"`
	Container_ports     []ContainerPortInfo `json:"container_ports"`
	Cpu_shares          int                 `json:"cpu_shares"`
	Deployed_datetime   string              `json:"deployed_datetime"`
	Destroyed_datetime  string              `json:"destroyed_datetime"`
	Entrypoint          string              `json:"entrypoint"`
	Exit_code           int                 `json:"exit_code"`
	Exit_code_message   string              `json:"exit_code_message"`
	Image_name          string              `json:"image_name"`
	Image_tag           string              `json:"image_tag"`
	Last_metric         Metric              `json:"last_metric"`
	Link_variables      map[string]string   `json:"link_variables"`
	Linked_to_container []ContainerLinkInfo `json:"linked_to_container"`
	Memory              int                 `json:"memory"`
	Name                string              `json:"name"`
	Net                 string              `json:"net"`
	Node                string              `json:"node"`
	Pid                 string              `json:"pid"`
	Private_ip          string              `json:"private_ip"`
	Privileged          bool                `json:"privileged"`
	Public_dns          string              `json:"public_dns"`
	Resource_uri        string              `json:"resource_uri"`
	Roles               []string            `json:"roles"`
	Run_command         string              `json:"run_command"`
	Service             string              `json:"service"`
	Started_datetime    string              `json:"started_datetime"`
	State               string              `json:"state"`
	Stopped_datetime    string              `json:"stopped_datetime"`
	Synchronized        bool                `json:"synchronized"`
	Uuid                string              `json:"uuid"`
	Working_dir         string              `json:"working_dir"`
}

type ContainerEnvvar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ContainerLinkInfo struct {
	Endpoints      map[string]string `json:"endpoints"`
	From_container string            `json:"from_container"`
	Name           string            `json:"name"`
	To_container   string            `json:"to_container"`
}

type ContainerPortInfo struct {
	Container  string `json:"container"`
	Inner_port int    `json:"inner_port"`
	Outer_port int    `json:"outer_port"`
	Protocol   string `json:"protocol"`
}

type Event struct {
	Type         string   `json:"type"`
	Action       string   `json:"action"`
	Parents      []string `json:"parents"`
	Resource_uri string   `json:"resource_uri"`
	State        string   `json:"state"`
}

type Exec struct {
	Type       string `json:"type"`
	Output     string `json:"output"`
	StreamType string `json:"streamType"`
}

type BuildSettings struct {
	Autobuild    bool   `json:"autobuild,omitempty"`
	Branch       string `json:"branch"`
	Dockerfile   string `json:"dockerfile"`
	Image        string `json:"image"`
	Resource_uri string `json:"resource_uri"`
	State        string `json:"state"`
	Tag          string `json:"tag"`
}

type BuildSource struct {
	Autotest       string   `json:"autotest,omitempty"`
	Build_Settings []string `json:"build_settings,omitempty"`
	Image          []string `json:"Image"`
	Owner          string   `json:"owner,omitempty"`
	Repository     string   `json:"repository,omitempty"`
	Type           string   `json:"type,omitempty"`
	Uuid           string   `json:"uuid"`
}

type RepositoryListResponse struct {
	Meta    Meta              `json:"meta"`
	Objects []RepositoryShort `json:"objects"`
}

type RepositoryShort struct {
	Build_Source     string   `json:"build_source"`
	Description      string   `json:"description"`
	Icon_url         string   `json:"icon_url"`
	In_use           bool     `json:"in_use"`
	Is_private_image bool     `json:"is_private_image"`
	Jumpstart        bool     `json:"jumpstart"`
	Last_build_date  string   `json:"last_build_date"`
	Name             string   `json:"name"`
	Public_url       string   `json:"public_url"`
	Registry         string   `json:"registry"`
	Resource_uri     string   `json:"resource_uri"`
	Star_count       int      `json:"star_count"`
	State            string   `json:"state"`
	Tags             []string `json:"tags"`
}

type Repository struct {
	Build_Source     BuildSource `json:"build_source,omitempty"`
	Description      string      `json:"description"`
	Icon_url         string      `json:"icon_url"`
	In_use           bool        `json:"in_use"`
	Is_private_image bool        `json:"is_private_image"`
	Jumpstart        bool        `json:"jumpstart"`
	Last_build_date  string      `json:"last_build_date"`
	Name             string      `json:"name"`
	Public_url       string      `json:"public_url"`
	Registry         string      `json:"registry"`
	Resource_uri     string      `json:"resource_uri"`
	Star_count       int         `json:"star_count"`
	State            string      `json:"state"`
	Tags             []string    `json:"tags"`
}

type RepositoryCreateRequest struct {
	Name     string `json:"name,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type LayerStruct struct {
	Author       string            `json:"author"`
	Creation     string            `json:"creation"`
	Docker_id    string            `json:"docker_id"`
	Entrypoint   string            `json:"entrypoint"`
	Envvars      []ContainerEnvvar `json:"envvars"`
	Ports        []Port            `json:"ports"`
	Resource_uri string            `json:"resource_uri"`
	Run_command  string            `json:"run_command"`
	Volumes      []VolumePath      `json:"volumes"`
}

type Logs struct {
	Type       string `json:"type"`
	Log        string `json:"log"`
	StreamType string `json:"streamType"`
	Timestamp  int    `json:"timestamp"`
}

type Network struct {
	Name string `json:"name"`
	CIDR string `json:"cidr"`
}

type NodeListResponse struct {
	Meta    Meta   `json:"meta"`
	Objects []Node `json:"objects"`
}

type Node struct {
	Availability_zone  string    `json:"availability_zone,omniempty"`
	Deployed_datetime  string    `json:"deployed_datetime,omitempty"`
	Destroyed_datetime string    `json:"destroyed_datetime,omitempty"`
	Docker_version     string    `json:"docker_version,omitempty"`
	Last_seen          string    `json:"last_seen,omitempty"`
	Node_cluster       string    `json:"node_cluster,omitempty"`
	Public_ip          string    `json:"public_ip,omitempty"`
	Private_ips        []Network `json:"private_ips,omitempty"`
	Region             string    `json:"region,omitempty"`
	Resource_uri       string    `json:"resource_uri,omitempty"`
	State              string    `json:"state,omitempty"`
	Tags               []NodeTag `json:"tags,omitempty"`
	Uuid               string    `json:"uuid,omitempty"`
}

type NodeEvent struct {
	Type string `json:"type"`
	Log  string `json:"log"`
}

type NodeTag struct {
	Name string `json:"name"`
}

type NodeClusterListResponse struct {
	Meta    Meta          `json:"meta"`
	Objects []NodeCluster `json:"objects"`
}

type VPC struct {
	Id              string   `json:"id"`
	Subnets         []string `json:"subnets,omitempty"`
	Security_groups []string `json:"security_groups,omitempty"`
}

type IAM struct {
	Instance_profile_name string `json:"instance_profile_name,omitempty"`
}

type ProviderOption struct {
	Vpc VPC `json:"vpc,omitempty"`
	Iam IAM `json:"iam,omitempty"`
}

type NodeCluster struct {
	Current_num_nodes  int            `json:"current_num_nodes"`
	Deployed_datetime  string         `json:"deployed_datetime"`
	Destroyed_datetime string         `json:"destroyed_datetime"`
	Disk               int            `json:"disk"`
	Name               string         `json:"name"`
	Nodes              []string       `json:"nodes"`
	NodeType           string         `json:"node_type"`
	Provider_options   ProviderOption `json:"provider_options"`
	Region             string         `json:"region"`
	Resource_uri       string         `json:"resource_uri"`
	State              string         `json:"state"`
	Tags               []NodeTag      `json:"tags,omitempty"`
	Target_num_nodes   int            `json:"target_num_nodes"`
	Uuid               string         `json:"uuid"`
}

type NodeCreateRequest struct {
	Disk             int       `json:"disk,omitempty"`
	Name             string    `json:"name,omitempty"`
	NodeType         string    `json:"node_type,omitempty"`
	Region           string    `json:"region,omitempty"`
	Target_num_nodes int       `json:"target_num_nodes,omitempty"`
	Tags             []NodeTag `json:"tags,omitempty"`
}

type NodeTypeListResponse struct {
	Meta    Meta       `json:"meta"`
	Objects []NodeType `json:"objects"`
}

type NodeType struct {
	Available    bool     `json:"available"`
	Label        string   `json:"label"`
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Regions      []string `json:"regions"`
	Resource_uri string   `json:"resource_uri"`
}

type Port struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
}

type ProviderListResponse struct {
	Meta    Meta       `json:"meta"`
	Objects []Provider `json:"objects"`
}

type Provider struct {
	Available    bool     `json:"available"`
	Label        string   `json:"label"`
	Name         string   `json:"name"`
	Regions      []string `json:"regions"`
	Resource_uri string   `json:"resource_uri"`
}

type RegionListResponse struct {
	Meta    Meta     `json:"meta"`
	Objects []Region `json:"objects"`
}

type Region struct {
	Available    bool     `json:"available"`
	Label        string   `json:"label"`
	Name         string   `json:"name"`
	Node_types   []string `json:"node_types"`
	Provider     string   `json:"provider"`
	Resource_uri string   `json:"resource_uri"`
}

type RegistryListResponse struct {
	Meta    Meta       `json:"meta"`
	Objects []Registry `json:"objects"`
}

type Registry struct {
	Host              string `json:"host"`
	Icon_url          string `json:"icon_url"`
	Is_ssl            bool   `json:"is_ssl"`
	Is_tutum_registry bool   `json:"is_tutum_registry"`
	Name              string `json:"name"`
	Resource_uri      string `json:"resource_uri"`
}

type ReuseVolumesOption struct {
	Reuse bool
}

type SListResponse struct {
	Meta    Meta      `json:"meta"`
	Objects []Service `json:"objects"`
}

type Service struct {
	Autodestroy            string              `json:"autodestroy"`
	Autoredeploy           bool                `json:"autoredeploy"`
	Autorestart            string              `json:"autorestart"`
	Bindings               []ServiceBinding    `json:"bindings"`
	Container_envvars      []ContainerEnvvar   `json:"container_envvars"`
	Container_ports        []ContainerPortInfo `json:"container_ports"`
	Containers             []string            `json:"containers"`
	Cpu_shares             int                 `json:"cpu_shares"`
	Current_num_containers int                 `json:"current_num_containers"`
	Deployed_datetime      string              `json:"deployed_datetime"`
	Deployment_strategy    string              `json:"deployment_strategy"`
	Destroyed_datetime     string              `json:"destroyed_datetime"`
	Entrypoint             string              `json:"entrypoint"`
	Image_name             string              `json:"image_name"`
	Image_tag              string              `json:"image_tag"`
	Link_variables         map[string]string   `json:"link_variables"`
	Linked_from_service    []ServiceLinkInfo   `json:"linked_from_service"`
	Linked_to_service      []ServiceLinkInfo   `json:"linked_to_service"`
	Memory                 int                 `json:"memory"`
	Name                   string              `json:"name"`
	Net                    string              `json:"net"`
	Pid                    string              `json:"pid"`
	Privileged             bool                `json:"privileged"`
	Public_dns             string              `json:"public_dns"`
	Resource_uri           string              `json:"resource_uri"`
	Roles                  []string            `json:"roles"`
	Run_command            string              `json:"run_command"`
	Running_num_containers int                 `json:"running_num_containers"`
	Sequential_deployment  bool                `json:"sequential_deployment"`
	Stack                  string              `json:"stack"`
	Started_datetime       string              `json:"started_datetime"`
	State                  string              `json:"state"`
	Stopped_datetime       string              `json:"stopped_datetime"`
	Stopped_num_containers int                 `json:"stopped_num_containers"`
	Synchronized           bool                `json:"synchronized"`
	Tags                   []ServiceTag        `json:"tags"`
	Target_num_containers  int                 `json:"target_num_containers"`
	Uuid                   string              `json:"uuid"`
	Working_dir            string              `json:"working_dir"`
}

type ServiceBinding struct {
	Container_path string `json:"container_path"`
	Host_path      string `json:"host_path"`
	Rewritable     bool   `json:"rewritable"`
	Volumes_from   string `json:"volumes_from"`
}

type ServiceCreateRequest struct {
	Autodestroy           string              `json:"autodestroy,omitempty"`
	Autoredeploy          bool                `json:"autoredeploy,omitempty"`
	Autorestart           string              `json:"autorestart,omitempty"`
	Bindings              []ServiceBinding    `json:"bindings,omitempty"`
	Container_envvars     []ContainerEnvvar   `json:"container_envvars,omitempty"`
	Container_ports       []ContainerPortInfo `json:"container_ports,omitempty"`
	Deployment_strategy   string              `json:"deployment_strategy,omitempty"`
	Entrypoint            string              `json:"entrypoint,omitempty"`
	Image                 string              `json:"image,omitempty"`
	Linked_to_service     []ServiceLinkInfo   `json:"linked_to_service,omitempty"`
	Name                  string              `json:"name,omitempty"`
	Net                   string              `json:"net,omitempty"`
	Pid                   string              `json:"pid,omitempty"`
	Privileged            bool                `json:"privileged,omitempty"`
	Roles                 []string            `json:"roles,omitempty"`
	Run_command           string              `json:"run_command,omitempty"`
	Sequential_deployment bool                `json:"sequential_deployment,omitempty"`
	Tags                  []string            `json:"tags,omitempty"`
	Target_num_containers int                 `json:"target_num_containers,omitempty"`
	Working_dir           string              `json:"working_dir,omitempty"`
}

type ServiceLinkInfo struct {
	From_service string `json:"from_service,omitempty"`
	Name         string `json:"name"`
	To_service   string `json:"to_service"`
}

type ServiceTag struct {
	Name string `json:"name"`
}

type StackListResponse struct {
	Meta    Meta    `json:"meta"`
	Objects []Stack `json:"objects"`
}

type Stack struct {
	Deployed_datetime  string   `json:"deployed_datetime"`
	Destroyed_datetime string   `json:"destroyed_datetime"`
	Name               string   `json:"name"`
	Resource_uri       string   `json:"resource_uri"`
	Services           []string `json:"services"`
	State              string   `json:"state"`
	Synchronized       bool     `json:"synchronized"`
	Uuid               string   `json:"uuid"`
}

type StackCreateRequest struct {
	Name     string                 `json:"name,omitempty"`
	Services []ServiceCreateRequest `json:"services,omitempty"`
}

type Token struct {
	Token string `json:"token"`
}

type TriggerListResponse struct {
	Meta    Meta      `json:"meta"`
	Objects []Trigger `json:"objects"`
}

type Trigger struct {
	Url          string `json:"url,omitempty"`
	Name         string `json:"name"`
	Operation    string `json:"operation"`
	Resource_uri string `json:"resource_uri,omitempty"`
}

type TriggerCreateRequest struct {
	Name      string `json:"name"`
	Operation string `json:"operation"`
}

type VolumePath struct {
	Container_path string `json:"container_path"`
}
