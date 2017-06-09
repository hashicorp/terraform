package artifactory

import (
	"encoding/json"
	"errors"
)

type User struct {
	Name string `json:"name"`
	Uri  string `json:"uri"`
}

type UserDetails struct {
	Name                     string   `json:"name,omitempty"`
	Email                    string   `json:"email"`
	Password                 string   `json:"password"`
	Admin                    bool     `json:"admin,omitempty"`
	ProfileUpdatable         bool     `json:"profileUpdatable,omitempty"`
	InternalPasswordDisabled bool     `json:"internalPasswordDisabled,omitempty"`
	LastLoggedIn             string   `json:"lastLoggedIn,omitempty"`
	Realm                    string   `json:"realm,omitempty"`
	Groups                   []string `json:"groups,omitempty"`
}

func (c *ArtifactoryClient) GetUsers() ([]User, error) {
	var res []User
	d, e := c.Get("/api/security/users", make(map[string]string))
	if e != nil {
		return res, e
	} else {
		err := json.Unmarshal(d, &res)
		if err != nil {
			return res, err
		} else {
			return res, e
		}
	}
}

func (c *ArtifactoryClient) GetUserDetails(u string) (UserDetails, error) {
	var res UserDetails
	d, e := c.Get("/api/security/users/"+u, make(map[string]string))
	if e != nil {
		return res, e
	} else {
		err := json.Unmarshal(d, &res)
		if err != nil {
			return res, err
		} else {
			return res, e
		}
	}
}

func (c *ArtifactoryClient) CreateUser(uname string, u UserDetails) error {
	if &u.Email == nil || &u.Password == nil {
		return errors.New("Email and password are required to create users")
	}
	j, jerr := json.Marshal(u)
	if jerr != nil {
		return jerr
	}
	o := make(map[string]string)
	_, err := c.Put("/api/security/users/"+uname, string(j), o)
	if err != nil {
		return err
	}
	return nil
}

func (c *ArtifactoryClient) DeleteUser(uname string) error {
	err := c.Delete("/api/security/users/" + uname)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func (c *ArtifactoryClient) GetUserEncryptedPassword() (s string, err error) {
	d, err := c.Get("/api/security/encryptedPassword", make(map[string]string))
	return string(d), err
}
