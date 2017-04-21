package oneandone

import "net/http"

type User struct {
	Identity
	descField
	CreationDate string    `json:"creation_date,omitempty"`
	Email        string    `json:"email,omitempty"`
	State        string    `json:"state,omitempty"`
	Role         *Identity `json:"role,omitempty"`
	Api          *UserApi  `json:"api,omitempty"`
	ApiPtr
}

type UserApi struct {
	Active     bool     `json:"active"`
	AllowedIps []string `json:"allowed_ips,omitempty"`
	UserApiKey
	ApiPtr
}

type UserApiKey struct {
	Key string `json:"key,omitempty"`
}

type UserRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Password    string `json:"password,omitempty"`
	Email       string `json:"email,omitempty"`
	State       string `json:"state,omitempty"`
}

// GET /users
func (api *API) ListUsers(args ...interface{}) ([]User, error) {
	url, err := processQueryParams(createUrl(api, userPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []User{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// POST /users
func (api *API) CreateUser(user *UserRequest) (string, *User, error) {
	result := new(User)
	url := createUrl(api, userPathSegment)
	err := api.Client.Post(url, &user, &result, http.StatusCreated)
	if err != nil {
		return "", nil, err
	}
	result.api = api
	return result.Id, result, nil
}

// GET /users/{id}
func (api *API) GetUser(user_id string) (*User, error) {
	result := new(User)
	url := createUrl(api, userPathSegment, user_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /users/{id}
func (api *API) DeleteUser(user_id string) (*User, error) {
	result := new(User)
	url := createUrl(api, userPathSegment, user_id)
	err := api.Client.Delete(url, nil, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /users/{id}
func (api *API) ModifyUser(user_id string, user *UserRequest) (*User, error) {
	result := new(User)
	url := createUrl(api, userPathSegment, user_id)
	err := api.Client.Put(url, &user, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /users/{id}/api
func (api *API) GetUserApi(user_id string) (*UserApi, error) {
	result := new(UserApi)
	url := createUrl(api, userPathSegment, user_id, "api")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// PUT /users/{id}/api
func (api *API) ModifyUserApi(user_id string, active bool) (*User, error) {
	result := new(User)
	req := struct {
		Active bool `json:"active"`
	}{active}
	url := createUrl(api, userPathSegment, user_id, "api")
	err := api.Client.Put(url, &req, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /users/{id}/api/key
func (api *API) GetUserApiKey(user_id string) (*UserApiKey, error) {
	result := new(UserApiKey)
	url := createUrl(api, userPathSegment, user_id, "api/key")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// PUT /users/{id}/api/key
func (api *API) RenewUserApiKey(user_id string) (*User, error) {
	result := new(User)
	url := createUrl(api, userPathSegment, user_id, "api/key")
	err := api.Client.Put(url, nil, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /users/{id}/api/ips
func (api *API) ListUserApiAllowedIps(user_id string) ([]string, error) {
	result := []string{}
	url := createUrl(api, userPathSegment, user_id, "api/ips")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// POST /users/{id}/api/ips
func (api *API) AddUserApiAlowedIps(user_id string, ips []string) (*User, error) {
	result := new(User)
	req := struct {
		Ips []string `json:"ips"`
	}{ips}
	url := createUrl(api, userPathSegment, user_id, "api/ips")
	err := api.Client.Post(url, &req, &result, http.StatusCreated)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// DELETE /users/{id}/api/ips/{ip}
func (api *API) RemoveUserApiAllowedIp(user_id string, ip string) (*User, error) {
	result := new(User)
	url := createUrl(api, userPathSegment, user_id, "api/ips", ip)
	err := api.Client.Delete(url, nil, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}

// GET /users/{id}/api/ips
func (api *API) GetCurrentUserPermissions() (*Permissions, error) {
	result := new(Permissions)
	url := createUrl(api, userPathSegment, "current_user_permissions")
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (u *User) GetState() (string, error) {
	in, err := u.api.GetUser(u.Id)
	if in == nil {
		return "", err
	}
	return in.State, err
}
