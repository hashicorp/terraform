// Package implements OCCM Auth API
package auth

import (
  "github.com/candidpartners/occm-sdk-go/api/client"
	"github.com/pkg/errors"
)

// Auth API
type AuthAPI struct {
	*client.Client
}

// New creates a new OCCM Auth API client
func New(context *client.Context) (*AuthAPI, error) {
  c, err := client.New(context)
  if err != nil {
    return nil, errors.Wrap(err, client.ErrClientCreationFailed)
  }

	api := &AuthAPI{
		Client: c,
	}

	return api, nil
}

// Login attempts to log a user in
func (api *AuthAPI) Login(email, password string) error {
  if email == "" || password == "" {
		return errors.New(client.ErrInvalidCredentials)
	}

  _, _, err := api.Client.Invoke("POST", "/auth/login",
    nil,
    map[string]interface{}{
      "email": email,
      "password": password,
    },
  )
  if err != nil {
		return errors.Wrap(err, client.ErrInvalidRequest)
	}

  return nil
}

// Logouts attempts to log a user out
func (api *AuthAPI) Logout() error {

  return nil
}
