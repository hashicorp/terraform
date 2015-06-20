package data_types

import (
	"time"
)

type SoftLayer_Virtual_Guest_Parameters struct {
	Parameters []SoftLayer_Virtual_Guest `json:"parameters"`
}

type SoftLayer_Virtual_Guest struct {
	AccountId                    int        `json:"accountId,omitempty"`
	CreateDate                   *time.Time `json:"createDate,omitempty"`
	DedicatedAccountHostOnlyFlag bool       `json:"dedicatedAccountHostOnlyFlag,omitempty"`
	Domain                       string     `json:"domain,omitempty"`
	FullyQualifiedDomainName     string     `json:"fullyQualifiedDomainName,omitempty"`
	Hostname                     string     `json:"hostname,omitempty"`
	Id                           int        `json:"id,omitempty"`
	LastPowerStateId             int        `json:"lastPowerStateId,omitempty"`
	LastVerifiedDate             *time.Time `json:"lastVerifiedDate,omitempty"`
	MaxCpu                       int        `json:"maxCpu,omitempty"`
	MaxCpuUnits                  string     `json:"maxCpuUnits,omitempty"`
	MaxMemory                    int        `json:"maxMemory,omitempty"`
	MetricPollDate               *time.Time `json:"metricPollDate,omitempty"`
	ModifyDate                   *time.Time `json:"modifyDate,omitempty"`
	Notes                        string     `json:"notes,omitempty"`
	PostInstallScriptUri         string     `json:"postInstallScriptUri,omitempty"`
	PrivateNetworkOnlyFlag       bool       `json:"privateNetworkOnlyFlag,omitempty"`
	StartCpus                    int        `json:"startCpus,omitempty"`
	StatusId                     int        `json:"statusId,omitempty"`
	Uuid                         string     `json:"uuid,omitempty"`
	LocalDiskFlag                bool       `json:"localDiskFlag,omitempty"`
	HourlyBillingFlag            bool       `json:"hourlyBillingFlag,omitempty"`

	GlobalIdentifier        string `json:"globalIdentifier,omitempty"`
	ManagedResourceFlag     bool   `json:"managedResourceFlag,omitempty"`
	PrimaryBackendIpAddress string `json:"primaryBackendIpAddress,omitempty"`
	PrimaryIpAddress        string `json:"primaryIpAddress,omitempty"`

	PrimaryNetworkComponent        *PrimaryNetworkComponent        `json:"primaryNetworkComponent,omitempty"`
	PrimaryBackendNetworkComponent *PrimaryBackendNetworkComponent `json:"primaryBackendNetworkComponent,omitempty"`

	Location          *SoftLayer_Location `json:"location"`
	Datacenter        *SoftLayer_Location `json:"datacenter"`
	NetworkComponents []NetworkComponents `json:"networkComponents,omitempty"`
	UserData          []UserData          `json:"userData,omitempty"`

	OperatingSystem *SoftLayer_Operating_System `json:"operatingSystem"`

	BlockDeviceTemplateGroup *BlockDeviceTemplateGroup `json:"blockDeviceTemplateGroup,omitempty"`
}

type SoftLayer_Operating_System struct {
	Passwords []SoftLayer_Password `json:"passwords"`
}

type SoftLayer_Password struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SoftLayer_Virtual_Guest_Template_Parameters struct {
	Parameters []SoftLayer_Virtual_Guest_Template `json:"parameters"`
}

type SoftLayer_Virtual_Guest_Template struct {
	//Required
	Hostname          string     `json:"hostname"`
	Domain            string     `json:"domain"`
	StartCpus         int        `json:"startCpus"`
	MaxMemory         int        `json:"maxMemory"`
	Datacenter        Datacenter `json:"datacenter"`
	HourlyBillingFlag bool       `json:"hourlyBillingFlag"`
	LocalDiskFlag     bool       `json:"localDiskFlag"`

	//Conditionally required
	OperatingSystemReferenceCode string                    `json:"operatingSystemReferenceCode,omitempty"`
	BlockDeviceTemplateGroup     *BlockDeviceTemplateGroup `json:"blockDeviceTemplateGroup,omitempty"`

	//Optional
	DedicatedAccountHostOnlyFlag   bool                            `json:"dedicatedAccountHostOnlyFlag,omitempty"`
	NetworkComponents              []NetworkComponents             `json:"networkComponents,omitempty"`
	PrivateNetworkOnlyFlag         bool                            `json:"privateNetworkOnlyFlag,omitempty"`
	PrimaryNetworkComponent        *PrimaryNetworkComponent        `json:"primaryNetworkComponent,omitempty"`
	PrimaryBackendNetworkComponent *PrimaryBackendNetworkComponent `json:"primaryBackendNetworkComponent,omitempty"`
	PostInstallScriptUri           string                          `json:"postInstallScriptUri,omitempty"`

	BlockDevices []BlockDevice `json:"blockDevices,omitempty"`
	UserData     []UserData    `json:"userData,omitempty"`
	SshKeys      []SshKey      `json:"sshKeys,omitempty"`
}

type Datacenter struct {
	//Required
	Name string `json:"name"`
}

type BlockDeviceTemplateGroup struct {
	//Required
	GlobalIdentifier string `json:"globalIdentifier,omitempty"`
}

type NetworkComponents struct {
	//Required, defaults to 10
	MaxSpeed int `json:"maxSpeed,omitempty"`
}

type NetworkVlan struct {
	//Required
	Id int `json:"id,omitempty"`
}

type PrimaryNetworkComponent struct {
	//Required
	NetworkVlan NetworkVlan `json:"networkVlan,omitempty"`
}

type PrimaryBackendNetworkComponent struct {
	//Required
	NetworkVlan NetworkVlan `json:"networkVlan,omitempty"`
}

type DiskImage struct {
	//Required
	Capacity int `json:"capacity,omitempty"`
}

type BlockDevice struct {
	//Required
	Device    string    `json:"device,omitempty"`
	DiskImage DiskImage `json:"diskImage,omitempty"`
}

type UserData struct {
	//Required
	Value string `json:"value,omitempty"`
}

type SshKey struct {
	//Required
	Id int `json:"id,omitempty"`
}

type SoftLayer_Virtual_Guest_SetTags_Parameters struct {
	Parameters []string `json:"parameters"`
}

type Image_Template_Config struct {
	ImageTemplateId string `json:"imageTemplateId"`
}
