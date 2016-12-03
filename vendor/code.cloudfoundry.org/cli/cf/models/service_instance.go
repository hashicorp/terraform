package models

type LastOperationFields struct {
	Type        string
	State       string
	Description string
	CreatedAt   string
	UpdatedAt   string
}

type ServiceInstanceCreateRequest struct {
	Name      string                 `json:"name"`
	SpaceGUID string                 `json:"space_guid"`
	PlanGUID  string                 `json:"service_plan_guid,omitempty"`
	Params    map[string]interface{} `json:"parameters,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
}

type ServiceInstanceUpdateRequest struct {
	PlanGUID string                 `json:"service_plan_guid,omitempty"`
	Params   map[string]interface{} `json:"parameters,omitempty"`
	Tags     []string               `json:"tags"`
}

type ServiceInstanceFields struct {
	GUID             string
	Name             string
	LastOperation    LastOperationFields
	SysLogDrainURL   string
	RouteServiceURL  string
	ApplicationNames []string
	Params           map[string]interface{}
	DashboardURL     string
	Tags             []string
}

type ServiceInstance struct {
	ServiceInstanceFields
	ServiceBindings []ServiceBindingFields
	ServiceKeys     []ServiceKeyFields
	ServicePlan     ServicePlanFields
	ServiceOffering ServiceOfferingFields
}

func (inst ServiceInstance) IsUserProvided() bool {
	return inst.ServicePlan.GUID == ""
}
