package models

type ServicePlanVisibilityFields struct {
	GUID             string `json:"guid"`
	ServicePlanGUID  string `json:"service_plan_guid"`
	OrganizationGUID string `json:"organization_guid"`
}
