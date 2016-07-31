package data_types

type SoftLayer_Virtual_Guest_Block_Device_Template_GroupInitParameters struct {
	Parameters SoftLayer_Virtual_Guest_Block_Device_Template_GroupInitParameter `json:"parameters"`
}

type SoftLayer_Virtual_Guest_Block_Device_Template_GroupInitParameter struct {
	AccountId int `json:"accountId"`
}

type SoftLayer_Virtual_Guest_Block_Device_Template_Group_LocationsInitParameters struct {
	Parameters SoftLayer_Virtual_Guest_Block_Device_Template_Group_LocationsInitParameter `json:"parameters"`
}

type SoftLayer_Virtual_Guest_Block_Device_Template_Group_LocationsInitParameter struct {
	Locations []SoftLayer_Location `json:"locations"`
}

type SoftLayer_Virtual_Guest_Block_Device_Template_GroupInitParameters2 struct {
	Parameters []interface{} `json:"parameters"`
}
