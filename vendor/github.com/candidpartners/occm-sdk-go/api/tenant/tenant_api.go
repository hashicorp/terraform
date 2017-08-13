// Package implements OCCM Tenant API
package tenant

import (
  "github.com/candidpartners/occm-sdk-go/api/client"
	"github.com/pkg/errors"
)

// Tenant API
type TenantAPI struct {
	*client.Client
}

// New creates a new OCCM Tenant API client
func New(context *client.Context) (*TenantAPI, error) {
  c, err := client.New(context)
  if err != nil {
    return nil, errors.Wrap(err, client.ErrClientCreationFailed)
  }

	api := &TenantAPI{
		Client: c,
	}

	return api, nil
}

// GetTenants retrieves a list of all tenants
func (api *TenantAPI) GetTenants() ([]Tenant, error) {
  data, _, err := api.Client.Invoke("GET", "/tenants", nil, nil)
  if err != nil {
		return nil, errors.Wrap(err, client.ErrInvalidRequest)
	}

  result, err := ListFromJSON(data);
  if err != nil {
    return nil, errors.Wrap(err, client.ErrJSONConversion)
  }

  return result, nil
}
