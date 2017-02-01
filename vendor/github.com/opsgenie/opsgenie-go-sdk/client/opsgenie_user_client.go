package client

import (
	"errors"

	"github.com/opsgenie/opsgenie-go-sdk/user"
	"github.com/opsgenie/opsgenie-go-sdk/logging"
)

const (
	userURL          = "/v1/json/user"
)

// OpsGenieUserClient is the data type to make User API requests.
type OpsGenieUserClient struct {
	OpsGenieClient
}

// SetOpsGenieClient sets the embedded OpsGenieClient type of the OpsGenieUserClient.
func (cli *OpsGenieUserClient) SetOpsGenieClient(ogCli OpsGenieClient) {
	cli.OpsGenieClient = ogCli
}

// Create method creates a user at OpsGenie.
func (cli *OpsGenieUserClient) Create(req user.CreateUserRequest) (*user.CreateUserResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(userURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var createUserResp user.CreateUserResponse

	if err = resp.Body.FromJsonTo(&createUserResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &createUserResp, nil
}

// Update method updates a user at OpsGenie.
func (cli *OpsGenieUserClient) Update(req user.UpdateUserRequest) (*user.UpdateUserResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(userURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updateUserResp user.UpdateUserResponse

	if err = resp.Body.FromJsonTo(&updateUserResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &updateUserResp, nil
}

// Delete method deletes a user at OpsGenie.
func (cli *OpsGenieUserClient) Delete(req user.DeleteUserRequest) (*user.DeleteUserResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildDeleteRequest(userURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var deleteUserResp user.DeleteUserResponse

	if err = resp.Body.FromJsonTo(&deleteUserResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &deleteUserResp, nil
}

// Get method retrieves specified user details from OpsGenie.
func (cli *OpsGenieUserClient) Get(req user.GetUserRequest) (*user.GetUserResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(userURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var getUserResp user.GetUserResponse

	if err = resp.Body.FromJsonTo(&getUserResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &getUserResp, nil
}

// List method retrieves users from OpsGenie.
func (cli *OpsGenieUserClient) List(req user.ListUsersRequest) (*user.ListUsersResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(userURL, req))

	if resp == nil {
		return nil, errors.New(err.Error())
	}
	defer resp.Body.Close()

	var listUsersResp user.ListUsersResponse

	if err = resp.Body.FromJsonTo(&listUsersResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &listUsersResp, nil
}
