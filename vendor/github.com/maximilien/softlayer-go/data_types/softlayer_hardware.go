package data_types

import (
	"time"
)

type SoftLayer_Hardware_Template_Parameters struct {
	Parameters []SoftLayer_Hardware_Template `json:"parameters"`
}

type SoftLayer_Hardware_Parameters struct {
	Parameters []SoftLayer_Hardware `json:"parameters"`
}

type SoftLayer_Hardware_Template struct {
	Hostname                     string `json:"hostname"`
	Domain                       string `json:"domain"`
	ProcessorCoreAmount          int    `json:"processorCoreAmount"`
	MemoryCapacity               int    `json:"memoryCapacity"`
	HourlyBillingFlag            bool   `json:"hourlyBillingFlag"`
	OperatingSystemReferenceCode string `json:"operatingSystemReferenceCode"`

	Datacenter *Datacenter `json:"datacenter"`
}

type SoftLayer_Hardware struct {
	BareMetalInstanceFlag    int        `json:"bareMetalInstanceFlag"`
	Domain                   string     `json:"domain"`
	Hostname                 string     `json:"hostname"`
	Id                       int        `json:"id"`
	HardwareStatusId         int        `json:"hardwareStatusId"`
	ProvisionDate            *time.Time `json:"provisionDate"`
	GlobalIdentifier         string     `json:"globalIdentifier"`
	PrimaryIpAddress         string     `json:"primaryIpAddress"`
	PrimaryBackendIpAddress  string     `json:"primaryBackendIpAddress"`
	FullyQualifiedDomainName string     `json:"fullyQualifiedDomainName,omitempty"`

	OperatingSystem *SoftLayer_Operating_System `json:"operatingSystem"`

	Location   *SoftLayer_Location `json:"location"`
	Datacenter *SoftLayer_Location `json:"datacenter"`
}

type SoftLayer_Hardware_String_Parameters struct {
	Parameters []string `json:"parameters"`
}

type SoftLayer_Hardware_NetworkStorage_Parameters struct {
	Parameters SoftLayer_Network_Storage `json:"parameters"`
}
