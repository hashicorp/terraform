package data_types

type SoftLayer_Network_Application_Delivery_Controller struct {
	Id                    int                       `json:"id"`
	Name                  string                    `json:"name"`
	TypeId                int                       `json:"typeId"`
	ModifyDate            string                    `json:"modifyDate"`
	CreateDate            string                    `json:"createDate"`
	Description           string                    `json:"description"`
	ManagedResourceFlag   bool                      `json:"managedResourceFlag"`
	ManagementIpAddress   string                    `json:"managementIpAddress"`
	PrimaryIpAddress      string                    `json:"primaryIpAddress"`
	Password              SoftLayer_Password        `json:"password"`
	Notes                 string                    `json:"notes"`
	Datacenter            *SoftLayer_Location       `json:"datacenter"`
	NetworkVlan           *SoftLayer_Network_Vlan   `json:"networkVlan"`
	NetworkVlanCount      int                       `json:"networkVlanCount"`
	NetworkVlans          []SoftLayer_Network_Vlan  `json:"networkVlans"`
	TagReferenceCount     int                       `json:"tagReferenceCount"`
	TagReferences         []SoftLayer_Tag_Reference `json:"tagReferences"`
	VirtualAddressCount   int                       `json:"virtualIpAddressCount"`
	SubnetCount           int                       `json:"subnetCount"`
	LicenseExpirationDate string                    `json:"licenseExpirationDate"`

	Type *SoftLayer_Network_Application_Delivery_Controller_Type `json:"type"`
}

type SoftLayer_Network_Application_Delivery_Controller_Type struct {
	KeyName string `json:"keyName"`
	Name    string `json:"name"`
}
