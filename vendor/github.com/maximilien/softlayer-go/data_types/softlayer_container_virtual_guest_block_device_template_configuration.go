package data_types

type SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration_Parameters struct {
	Parameters []SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration `json:"parameters"`
}

type SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration struct {
	Name                         string `json:"name"`
	Note                         string `json:"note"`
	OperatingSystemReferenceCode string `json:"operatingSystemReferenceCode"`
	Uri                          string `json:"uri"`
}
