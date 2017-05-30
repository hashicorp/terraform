package service

import (
	"github.com/sky-uk/gonsx/api"
	"net/http"
)

// DeleteServiceAPI base object.
type DeleteServiceAPI struct {
	*api.BaseAPI
}

// NewDelete returns a new object of DeleteServiceAPI.
func NewDelete(serviceID string) *DeleteServiceAPI {
	this := new(DeleteServiceAPI)
	this.BaseAPI = api.NewBaseAPI(http.MethodDelete, "/api/2.0/services/application/"+serviceID, nil, nil)
	return this
}
