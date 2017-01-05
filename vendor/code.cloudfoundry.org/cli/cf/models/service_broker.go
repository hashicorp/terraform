package models

type ServiceBroker struct {
	GUID     string
	Name     string
	Username string
	Password string
	URL      string
	Services []ServiceOffering
}
