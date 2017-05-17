package cloudflare

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// User describes a user account.
type User struct {
	ID            string         `json:"id"`
	Email         string         `json:"email"`
	FirstName     string         `json:"first_name"`
	LastName      string         `json:"last_name"`
	Username      string         `json:"username"`
	Telephone     string         `json:"telephone"`
	Country       string         `json:"country"`
	Zipcode       string         `json:"zipcode"`
	CreatedOn     time.Time      `json:"created_on"`
	ModifiedOn    time.Time      `json:"modified_on"`
	APIKey        string         `json:"api_key"`
	TwoFA         bool           `json:"two_factor_authentication_enabled"`
	Betas         []string       `json:"betas"`
	Organizations []Organization `json:"organizations"`
}

// UserResponse wraps a response containing User accounts.
type UserResponse struct {
	Response
	Result User `json:"result"`
}

// UserDetails provides information about the logged-in user.
// API reference:
// 	https://api.cloudflare.com/#user-user-details
//	GET /user
func (api *API) UserDetails() (User, error) {
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

// UpdateUser updates the properties of the given user.
// API reference:
// 	https://api.cloudflare.com/#user-update-user
//	PATCH /user
func (api *API) UpdateUser() (User, error) {
	// api.makeRequest("PATCH", "/user", user)
	return User{}, nil
}
