package models

type SpaceFields struct {
	GUID     string
	Name     string
	AllowSSH bool
}

type Space struct {
	SpaceFields
	Organization     OrganizationFields
	Applications     []ApplicationFields
	ServiceInstances []ServiceInstanceFields
	Domains          []DomainFields
	SecurityGroups   []SecurityGroupFields
	SpaceQuotaGUID   string
}
