package models

type ServiceBindingRequest struct {
	AppGUID             string                 `json:"app_guid"`
	ServiceInstanceGUID string                 `json:"service_instance_guid"`
	Params              map[string]interface{} `json:"parameters,omitempty"`
}

type ServiceBindingFields struct {
	GUID    string
	URL     string
	AppGUID string
}
