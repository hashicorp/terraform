package models

type OrganizationFields struct {
	GUID            string
	Name            string
	QuotaDefinition QuotaFields
}

type Organization struct {
	OrganizationFields
	Spaces      []SpaceFields
	Domains     []DomainFields
	SpaceQuotas []SpaceQuota
}
