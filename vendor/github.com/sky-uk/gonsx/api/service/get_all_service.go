package service

import (
	"github.com/sky-uk/gonsx/api"
	"net/http"
)

// GetAllServiceAPI base object.
type GetAllServiceAPI struct {
	*api.BaseAPI
}

// NewGetAll returns a new object of GetAllServiceAPI.
func NewGetAll(scopeID string) *GetAllServiceAPI {
	this := new(GetAllServiceAPI)
	this.BaseAPI = api.NewBaseAPI(http.MethodGet, "/api/2.0/services/application/scope/"+scopeID, nil, new(ApplicationsList))
	return this
}

// GetResponse returns ResponseObject of GetAllServiceAPI.
func (ga GetAllServiceAPI) GetResponse() *ApplicationsList {
	return ga.ResponseObject().(*ApplicationsList)
}
