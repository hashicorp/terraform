package models

type ServicePlanFields struct {
	GUID                string
	Name                string
	Free                bool
	Public              bool
	Description         string
	Active              bool
	ServiceOfferingGUID string
	OrgNames            []string
}

type ServicePlan struct {
	ServicePlanFields
	ServiceOffering ServiceOfferingFields
}

type ServicePlanSummary struct {
	GUID string
	Name string
}

func (servicePlanFields ServicePlanFields) OrgHasVisibility(orgName string) bool {
	if servicePlanFields.Public {
		return true
	}
	for _, org := range servicePlanFields.OrgNames {
		if org == orgName {
			return true
		}
	}
	return false
}
