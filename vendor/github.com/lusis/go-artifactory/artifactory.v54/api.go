package artifactory

import "encoding/json"

// UserAPIKey represents the JSON returned for a user's API Key in Artifactory
type UserAPIKey struct {
	APIKey string `json:"apiKey"`
}

// GetUserAPIKey returns the current user's api key
func (c *Client) GetUserAPIKey() (string, error) {
	var res UserAPIKey
	d, err := c.Get("/api/security/apiKey", make(map[string]string))
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(d, &res)
	if err != nil {
		return "", err
	}
	return res.APIKey, nil
}

// CreateUserAPIKey creates an apikey for the current user
func (c *Client) CreateUserAPIKey() (string, error) {
	var res UserAPIKey
	d, err := c.Post("/api/security/apiKey", nil, make(map[string]string))
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(d, &res)
	if err != nil {
		return "", err
	}
	return res.APIKey, nil
}
