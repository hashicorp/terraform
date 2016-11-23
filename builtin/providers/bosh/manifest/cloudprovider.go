package manifest

// CloudProvider - CloudProvider section of a Bosh manifest as a serializable struct
type CloudProvider struct {
	Template   Template                `yaml:"template"`
	SSHTunnel  SSHTunnel               `yaml:"ssh_tunnel"`
	MBus       string                  `yaml:"mbus"`
	Properties CloudProviderProperties `yaml:"properties"`
}

// Templates - A release job template
type Template struct {
	Name    string `yaml:"name"`
	Release string `yaml:"release"`
}

// SSHTunnel - Bosh SSH tunnel endpoint and credentials
type SSHTunnel struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	User       string `yaml:"user"`
	PrivateKey string `yaml:"private_key"`
}

// CloudProviderProperties - Properties of Cloud Provider CPI
type CloudProviderProperties struct {
	VCenter   *VCenter         `yaml:"vcenter,omitempty"`
	OpenStack *OpenStack       `yaml:"openstack,omitempty"`
	AWS       *AWS             `yaml:"aws,omitempty"`
	Azure     *Azure           `yaml:"azure,omitempty"`
	CPI       *CloudProperties `yaml:"cpi,omitempty"`

	Agent struct {
		MBUS string `yaml:"mbus"`
	} `yaml:"agent,flow"`

	BlobStore struct {
		Provider string `yaml:"provider"`
		Path     string `yaml:"path"`
	} `yaml:"blobstore,flow"`

	NTP []string `yaml:"ntp"`
}

// VCenter - properties to connect to VSphere IaaS
type VCenter struct {
	Address     string              `yaml:"address"`
	User        string              `yaml:"user"`
	Password    string              `yaml:"password"`
	Datacenters []VSphereDatacenter `yaml:"datacenters"`
}

// VSphereDatacenter - properties identifying a VSPhere datacenter
type VSphereDatacenter struct {
	Name                       string   `yaml:"name"`
	VMFolder                   string   `yaml:"vm_folder"`
	TemplateFolder             string   `yaml:"template_folder"`
	DatastorePattern           string   `yaml:"datastore_pattern"`
	PersistentDatastorePattern string   `yaml:"persistent_datastore_pattern"`
	DiskPath                   string   `yaml:"disk_path"`
	Clusters                   []string `yaml:"clusters"`
}

// OpenStack - properties to connect to OpenStack IaaS
type OpenStack struct {
	AuthURL               string   `yaml:"auth_url"`
	Project               string   `yaml:"project"`
	Domain                string   `yaml:"domain"`
	Username              string   `yaml:"username"`
	APIKey                string   `yaml:"api_key"`
	DefaulteKeyName       string   `yaml:"default_key_name"`
	DefaultSecurityGroups []string `yaml:"default_security_groups"`
}

// AWS - properties to connect to AWS IaaS
type AWS struct {
	Region                string   `yaml:"region"`
	AccessKeyID           string   `yaml:"access_key_id"`
	SecretAcccessKey      string   `yaml:"secret_access_key"`
	DefaultKeyName        string   `yaml:"default_key_name"`
	DefaultSecurityGroups []string `yaml:"default_security_groups"`
}

// Azure - properties to connect to Azure IaaS
type Azure struct {
	Environment          string `yaml:"environment"`
	SubscriptionID       string `yaml:"subscription_id"`
	TenantID             string `yaml:"tenant_id"`
	ClientID             string `yaml:"client_id"`
	ClientSecret         string `yaml:"client_secret"`
	ResourceGroupName    string `yaml:"resource_group_name"`
	StorageAccountName   string `yaml:"storage_account_name"`
	DefaultSecurityGroup string `yaml:"default_security_group"`
	SSHUser              string `yaml:"ssh_user"`
	SSHPublicKey         string `yaml:"ssh_public_key"`
}
