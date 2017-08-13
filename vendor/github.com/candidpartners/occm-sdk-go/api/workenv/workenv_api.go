// Package implements OCCM Working Environments API
package workenv

import (
  "github.com/candidpartners/occm-sdk-go/api/client"
	"github.com/pkg/errors"
)

// Working environment API
type WorkingEnvironmentAPI struct {
	*client.Client
}

// New creates a new OCCM Working Environment API client
func New(context *client.Context) (*WorkingEnvironmentAPI, error) {
  c, err := client.New(context)
  if err != nil {
    return nil, errors.Wrap(err, client.ErrClientCreationFailed)
  }

	api := &WorkingEnvironmentAPI{
		Client: c,
	}

	return api, nil
}

// GetWorkingEnvironments retrieves a list of all working environments
func (api *WorkingEnvironmentAPI) GetWorkingEnvironments() (*WorkingEnvironments, error) {
  data, _, err := api.Client.Invoke("GET", "/working-environments", nil, nil)
  if err != nil {
		return nil, errors.Wrap(err, client.ErrInvalidRequest)
	}

  result, err := ListFromJSON(data);
  if err != nil {
    return nil, errors.Wrap(err, client.ErrJSONConversion)
  }

  return result, nil
}
