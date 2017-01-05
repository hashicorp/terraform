package models

type ServiceKeyFields struct {
	Name                string
	GUID                string
	URL                 string
	ServiceInstanceGUID string
	ServiceInstanceURL  string
}

type ServiceKeyRequest struct {
	Name                string                 `json:"name"`
	ServiceInstanceGUID string                 `json:"service_instance_guid"`
	Params              map[string]interface{} `json:"parameters,omitempty"`
}

type ServiceKey struct {
	Fields      ServiceKeyFields
	Credentials map[string]interface{}
}
