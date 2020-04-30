package tfe

import (
	"context"
	"testing"
)

// TestAccountDetails represents the basic account information
// of a TFE/TFC user.
//
// See FetchTestAccountDetails for more information.
type TestAccountDetails struct {
	ID       string `json:"id" jsonapi:"primary,users"`
	Username string `jsonapi:"attr,username"`
	Email    string `jsonapi:"attr,email"`
}

// FetchTestAccountDetails returns TestAccountDetails
// of the user running the tests.
//
// Use this helper to fetch the username and email
// address associated with the token used to run the tests.
func FetchTestAccountDetails(t *testing.T, client *Client) *TestAccountDetails {
	tad := &TestAccountDetails{}
	req, err := client.newRequest("GET", "account/details", nil)
	if err != nil {
		t.Fatalf("could not create account details request: %v", err)
	}

	ctx := context.Background()
	err = client.do(ctx, req, tad)
	if err != nil {
		t.Fatalf("could not fetch test user details: %v", err)
	}
	return tad
}
