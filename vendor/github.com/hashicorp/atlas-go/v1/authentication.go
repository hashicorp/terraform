package atlas

import (
	"fmt"
	"log"
	"net/url"
	"strings"
)

// Login accepts a username and password as string arguments. Both username and
// password must be non-nil, non-empty values. Atlas does not permit
// passwordless authentication.
//
// If authentication is unsuccessful, an error is returned with the body of the
// error containing the server's response.
//
// If authentication is successful, this method sets the Token value on the
// Client and returns the Token as a string.
func (c *Client) Login(username, password string) (string, error) {
	log.Printf("[INFO] logging in user %s", username)

	if len(username) == 0 {
		return "", fmt.Errorf("client: missing username")
	}

	if len(password) == 0 {
		return "", fmt.Errorf("client: missing password")
	}

	// Make a request
	request, err := c.Request("POST", "/api/v1/authenticate", &RequestOptions{
		Body: strings.NewReader(url.Values{
			"user[login]":       []string{username},
			"user[password]":    []string{password},
			"user[description]": []string{"Created by the Atlas Go Client"},
		}.Encode()),
		Headers: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
	})
	if err != nil {
		return "", err
	}

	// Make the request
	response, err := checkResp(c.HTTPClient.Do(request))
	if err != nil {
		return "", err
	}

	// Decode the body
	var tResponse struct{ Token string }
	if err := decodeJSON(response, &tResponse); err != nil {
		return "", nil
	}

	// Set the token
	log.Printf("[DEBUG] setting atlas token (%s)", maskString(tResponse.Token))
	c.Token = tResponse.Token

	// Return the token
	return c.Token, nil
}

// Verify verifies that authentication and communication with Atlas
// is properly functioning.
func (c *Client) Verify() error {
	log.Printf("[INFO] verifying authentication")

	request, err := c.Request("GET", "/api/v1/authenticate", nil)
	if err != nil {
		return err
	}

	_, err = checkResp(c.HTTPClient.Do(request))
	return err
}

// maskString masks all but the first few characters of a string for display
// output. This is useful for tokens so we can display them to the user without
// showing the full output.
func maskString(s string) string {
	if len(s) <= 3 {
		return "*** (masked)"
	}

	return s[0:3] + "*** (masked)"
}
