package data_types

type SoftLayer_Network_LoadBalancer_VirtualIpAddress_Array []SoftLayer_Network_LoadBalancer_VirtualIpAddress

type SoftLayer_Network_LoadBalancer_VirtualIpAddress struct {
	Id                          int                                      `json:"id"`
	ConnectionLimit             int                                      `json:"connectionLimit"`
	CustomManagedFlag           bool                                     `json:"customManagedFlag"`
	LoadBalancingMethod         string                                   `json:"loadBalancingMethod"`
	LoadBalancingMethodFullName string                                   `json:"loadBalancingMethodFullName"`
	ModifyDate                  string                                   `json:"modifyDate"`
	Name                        string                                   `json:"name"`
	SecurityCertificateId       int                                      `json:"securityCertificateId"`
	SourcePort                  int                                      `json:"sourcePort"`
	Type                        string                                   `json:"type"`
	VirtualIpAddress            string                                   `json:"virtualIpAddress"`
	Services                    []SoftLayer_Network_LoadBalancer_Service `json:"services"`
}

type SoftLayer_Network_LoadBalancer_VirtualIpAddress_Template_Parameters struct {
	Parameters []SoftLayer_Network_LoadBalancer_VirtualIpAddress_Template `json:"parameters"`
}

type SoftLayer_Network_LoadBalancer_VirtualIpAddress_Template struct {
	Id                    int    `json:"id"`
	ConnectionLimit       int    `json:"connectionLimit"`
	LoadBalancingMethod   string `json:"loadBalancingMethod"`
	Name                  string `json:"name"`
	SecurityCertificateId int    `json:"securityCertificateId"`
	SourcePort            int    `json:"sourcePort"`
	Type                  string `json:"type"`
	VirtualIpAddress      string `json:"virtualIpAddress"`
}
