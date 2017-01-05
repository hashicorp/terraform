package resources

import (
	"encoding/json"
	"strconv"

	"code.cloudfoundry.org/cli/cf/models"
)

type PaginatedServiceOfferingResources struct {
	Resources []ServiceOfferingResource
}

type ServiceOfferingResource struct {
	Resource
	Entity ServiceOfferingEntity
}

type ServiceOfferingEntity struct {
	Label        string                `json:"label"`
	Version      string                `json:"version"`
	Description  string                `json:"description"`
	Provider     string                `json:"provider"`
	BrokerGUID   string                `json:"service_broker_guid"`
	Requires     []string              `json:"requires"`
	ServicePlans []ServicePlanResource `json:"service_plans"`
	Extra        ServiceOfferingExtra
}

type ServiceOfferingExtra struct {
	DocumentationURL string `json:"documentationURL"`
}

func (resource ServiceOfferingResource) ToFields() models.ServiceOfferingFields {
	return models.ServiceOfferingFields{
		Label:            resource.Entity.Label,
		Version:          resource.Entity.Version,
		Provider:         resource.Entity.Provider,
		Description:      resource.Entity.Description,
		BrokerGUID:       resource.Entity.BrokerGUID,
		GUID:             resource.Metadata.GUID,
		DocumentationURL: resource.Entity.Extra.DocumentationURL,
		Requires:         resource.Entity.Requires,
	}
}

func (resource ServiceOfferingResource) ToModel() models.ServiceOffering {
	offering := models.ServiceOffering{
		ServiceOfferingFields: resource.ToFields(),
	}

	for _, p := range resource.Entity.ServicePlans {
		offering.Plans = append(offering.Plans,
			models.ServicePlanFields{
				Name: p.Entity.Name,
				GUID: p.Metadata.GUID,
			},
		)
	}

	return offering
}

type serviceOfferingExtra ServiceOfferingExtra

func (resource *ServiceOfferingExtra) UnmarshalJSON(rawData []byte) error {
	if string(rawData) == "null" {
		return nil
	}

	extra := serviceOfferingExtra{}

	unquoted, err := strconv.Unquote(string(rawData))
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(unquoted), &extra)
	if err != nil {
		return err
	}

	*resource = ServiceOfferingExtra(extra)

	return nil
}
