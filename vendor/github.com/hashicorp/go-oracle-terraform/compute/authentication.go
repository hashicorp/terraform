package compute

import (
	"fmt"
	"time"
)

// AuthenticationReq represents the body of an authentication request.
type AuthenticationReq struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// Get a new auth cookie for the compute client
func (c *Client) getAuthenticationCookie() error {
	req := AuthenticationReq{
		User:     c.getUserName(),
		Password: *c.password,
	}

	rsp, err := c.executeRequest("POST", "/authenticate/", req)
	if err != nil {
		return err
	}

	if len(rsp.Cookies()) == 0 {
		return fmt.Errorf("No authentication cookie found in response %#v", rsp)
	}

	c.debugLogString("Successfully authenticated to OPC")
	c.authCookie = rsp.Cookies()[0]
	c.cookieIssued = time.Now()
	return nil
}
