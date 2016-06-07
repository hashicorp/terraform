package data_types

type SoftLayer_Network_LoadBalancer_Service struct {
	Name                 string `json:"name"`
	DestinationIpAddress string `json:"destinationIpAddress"`
	DestinationPort      int    `json:"destinationPort"`
	Weight               int    `json:"weight"`
	HealthCheck          string `json:"healthCheck"`
	ConnectionLimit      int    `json:"connectionLimit"`
}

type SoftLayer_Network_LoadBalancer_Service_Parameters struct {
	Parameters []SoftLayer_Network_LoadBalancer_Service_VipName_Services `json:"parameters"`
}

type SoftLayer_Network_LoadBalancer_Service_VipName_Services struct {
	VipName  string                                            `json:"name"`
	Services []SoftLayer_Network_LoadBalancer_Service_Template `json:"services"`
}

type SoftLayer_Network_LoadBalancer_Service_Parameters_Delete struct {
	Parameters []SoftLayer_Network_LoadBalancer_Service_VipName_Services_Delete `json:"parameters"`
}

type SoftLayer_Network_LoadBalancer_Service_VipName_Services_Delete struct {
	ServiceName string                                         `json:"name"`
	Vip         SoftLayer_Network_LoadBalancer_Service_VipName `json:"vip"`
}

type SoftLayer_Network_LoadBalancer_Service_VipName struct {
	VipName string `json:"name"`
}

type SoftLayer_Network_LoadBalancer_Service_Template struct {
	Name                 string `json:"name"`
	DestinationIpAddress string `json:"destinationIpAddress"`
	DestinationPort      int    `json:"destinationPort"`
	Weight               int    `json:"weight"`
	HealthCheck          string `json:"healthCheck"`
	ConnectionLimit      int    `json:"connectionLimit"`
}
