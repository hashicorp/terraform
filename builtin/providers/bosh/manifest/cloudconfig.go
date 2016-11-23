package manifest

import yaml "gopkg.in/yaml.v2"

// CloudConfigManifest - The Bosh CloudConfig manifest as a serializable struct
type CloudConfigManifest struct {
	AZs          []AZ          `yaml:"azs,omitempty"`
	Networks     []Network     `yaml:"networks,omitempty"`
	VMTypes      []VMType      `yaml:"vm_types,omitempty"`
	VMExtensions []VMExtension `yaml:"vm_extensions,omitempty"`
	DiskTypes    []DiskType    `yaml:"disk_types,omitempty"`
	Compilation  Compilation   `yaml:"compilation,omitempty"`
}

// AZ - IaaS Availability Zone
type AZ struct {
	Name            string          `yaml:"name"`
	CloudProperties CloudProperties `yaml:"cloud_properties,omitempty"`
}

// Network - The logical network to target the deployment to
type Network struct {
	Name            string          `yaml:"name"`
	Type            string          `yaml:"type"`
	Subnets         []Subnet        `yaml:"subnets,omitempty"`
	CloudProperties CloudProperties `yaml:"cloud_properties,omitempty"`
}

// Subnet - Physical attributes of a subnet within the logical network
type Subnet struct {
	Range           string          `yaml:"range"`
	Gateway         string          `yaml:"gateway"`
	AZ              string          `yaml:"az,omitempty"`
	AZs             []string        `yaml:"azs,omitempty"`
	DNS             []string        `yaml:"dns,omitempty"`
	Reserved        []string        `yaml:"reserved,omitempty"`
	Static          []string        `yaml:"static,omitempty"`
	CloudProperties CloudProperties `yaml:"cloud_properties,omitempty"`
}

// VMType - VM Template describing the VM's
type VMType struct {
	Name            string          `yaml:"name"`
	CloudProperties CloudProperties `yaml:"cloud_properties,omitempty"`
}

// VMExtension - VM Extension
type VMExtension struct {
	Name            string          `yaml:"name"`
	CloudProperties CloudProperties `yaml:"cloud_properties,omitempty"`
}

// DiskType - Disk Type
type DiskType struct {
	Name            string          `yaml:"name"`
	DiskSize        int             `yaml:"disk_size,omitempty"`
	CloudProperties CloudProperties `yaml:"cloud_properties,omitempty"`
}

// Compilation - Compilation VM attributes
type Compilation struct {
	Workers             int    `yaml:"workers,omitempty"`
	AZ                  string `yaml:"az,omitempty"`
	VMType              string `yaml:"vm_type,omitempty"`
	Network             string `yaml:"network,omitempty"`
	ReuseCompilationVMs bool   `yaml:"reuse_compilation_vms,omitempty"`
}

// GetName - retrieve the AZ name
func (az *AZ) GetName() string {
	return az.Name
}

// GetName - retrieve the logical Network name
func (n *Network) GetName() string {
	return n.Name
}

// GetName - retrieve the logical VM Type name
func (vt *VMType) GetName() string {
	return vt.Name
}

// GetName - retrieve the logical VM Extension name
func (ve *VMExtension) GetName() string {
	return ve.Name
}

// GetName - retrieve the logical Disk Type name
func (dt *DiskType) GetName() string {
	return dt.Name
}

// NewCloudConfigManifest - initialize a CloudConfigManifest
// from a YAML string given as a byte array
func NewCloudConfigManifest(b []byte) *CloudConfigManifest {
	cm := new(CloudConfigManifest)
	yaml.Unmarshal(b, cm)
	return cm
}

// Bytes - returns the YAML representation of
// the CloudConfigManifest as a byte array
func (s *CloudConfigManifest) Bytes() (b []byte, err error) {
	b, err = yaml.Marshal(s)
	return
}
