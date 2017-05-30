package service

import (
	"github.com/sky-uk/gonsx/api"
	"net/http"
)

// UpdateServiceAPI ...
type UpdateServiceAPI struct {
	*api.BaseAPI
}

// NewUpdate creates a new object of UpdateServiceAPI
func NewUpdate(serviceID string, payload *ApplicationService) *UpdateServiceAPI {
	this := new(UpdateServiceAPI)
	endpointURL := "/api/2.0/services/application/" + serviceID
	this.BaseAPI = api.NewBaseAPI(http.MethodPut, endpointURL, payload, new(ApplicationService))
	return this
}

// GetResponse returns the ResponseObject from UpdateServiceAPI
func (updateAPI UpdateServiceAPI) GetResponse() *ApplicationService {
	return updateAPI.ResponseObject().(*ApplicationService)
}
