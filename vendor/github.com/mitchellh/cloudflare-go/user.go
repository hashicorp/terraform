package cloudflare

import (
	"encoding/json"

	"github.com/pkg/errors"
)

/*
Information about the logged-in user.

API reference: https://api.cloudflare.com/#user-user-details
*/
func (api API) UserDetails() (User, error) {
	var r UserResponse
	res, err := api.makeRequest("GET", "/user", nil)
	if err != nil {
		return User{}, errors.Wrap(err, errMakeRequestError)
	}
	err = json.Unmarshal(res, &r)
	if err != nil {
		return User{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

/*
Update user properties.

API reference: https://api.cloudflare.com/#user-update-user
*/
func (api API) UpdateUser() (User, error) {
	// api.makeRequest("PATCH", "/user", user)
	return User{}, nil
}
