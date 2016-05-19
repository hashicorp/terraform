package data_types

type SoftLayer_Dns_Domain_Template struct {
	Name            string                                `json:"name"`
	ResourceRecords []SoftLayer_Dns_Domain_ResourceRecord `json:"resourceRecords"`
}

type SoftLayer_Dns_Domain_Template_Parameters struct {
	Parameters []SoftLayer_Dns_Domain_Template `json:"parameters"`
}

type SoftLayer_Dns_Domain struct {
	Id                  int                                   `json:"id"`
	Name                string                                `json:"name"`
	Serial              int                                   `json:"serial"`
	UpdateDate          string                                `json:"updateDate"`
	ManagedResourceFlag bool                                  `json:"managedResourceFlag"`
	ResourceRecordCount int                                   `json:"resourceRecordCount"`
	ResourceRecords     []SoftLayer_Dns_Domain_ResourceRecord `json:"resourceRecords"`
}
