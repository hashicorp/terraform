package service

import (
	"github.com/sky-uk/gonsx/api"
	"net/http"
)

// CreateServiceAPI api object
type CreateServiceAPI struct {
	*api.BaseAPI
}

// NewCreate returns a new object of CreateServiceAPI.
func NewCreate(scopeID, name, desc, proto, ports string) *CreateServiceAPI {
	this := new(CreateServiceAPI)
	requestPayload := new(ApplicationService)
	requestPayload.Name = name
	requestPayload.Description = desc

	element := Element{ApplicationProtocol: proto, Value: ports}
	requestPayload.Element = []Element{element}

	this.BaseAPI = api.NewBaseAPI(http.MethodPost, "/api/2.0/services/application/"+scopeID, requestPayload, new(string))
	return this
}

// GetResponse returns a ResponseObject of CreateServiceAPI.
func (ca CreateServiceAPI) GetResponse() string {
	return ca.ResponseObject().(string)
}
