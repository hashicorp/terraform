package client

import (
	"errors"

	"github.com/opsgenie/opsgenie-go-sdk/team"
	"github.com/opsgenie/opsgenie-go-sdk/logging"
)

const (
	teamURL          = "/v1/json/team"
	teamLogsURL        = "/v1/json/team/log"
)

// OpsGenieTeamClient is the data type to make Team API requests.
type OpsGenieTeamClient struct {
	OpsGenieClient
}

// SetOpsGenieClient sets the embedded OpsGenieClient type of the OpsGenieTeamClient.
func (cli *OpsGenieTeamClient) SetOpsGenieClient(ogCli OpsGenieClient) {
	cli.OpsGenieClient = ogCli
}

// Create method creates a team at OpsGenie.
func (cli *OpsGenieTeamClient) Create(req team.CreateTeamRequest) (*team.CreateTeamResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(teamURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var createTeamResp team.CreateTeamResponse

	if err = resp.Body.FromJsonTo(&createTeamResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &createTeamResp, nil
}

// Update method updates a team at OpsGenie.
func (cli *OpsGenieTeamClient) Update(req team.UpdateTeamRequest) (*team.UpdateTeamResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(teamURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updateTeamResp team.UpdateTeamResponse

	if err = resp.Body.FromJsonTo(&updateTeamResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &updateTeamResp, nil
}

// Delete method deletes a team at OpsGenie.
func (cli *OpsGenieTeamClient) Delete(req team.DeleteTeamRequest) (*team.DeleteTeamResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildDeleteRequest(teamURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var deleteTeamResp team.DeleteTeamResponse

	if err = resp.Body.FromJsonTo(&deleteTeamResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &deleteTeamResp, nil
}

// Get method retrieves specified team details from OpsGenie.
func (cli *OpsGenieTeamClient) Get(req team.GetTeamRequest) (*team.GetTeamResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(teamURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var getTeamResp team.GetTeamResponse

	if err = resp.Body.FromJsonTo(&getTeamResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &getTeamResp, nil
}

// List method retrieves teams from OpsGenie.
func (cli *OpsGenieTeamClient) List(req team.ListTeamsRequest) (*team.ListTeamsResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(teamURL,req))
	if resp == nil {
		return nil, errors.New(err.Error())
	}
	defer resp.Body.Close()

	var listTeamsResp team.ListTeamsResponse

	if err = resp.Body.FromJsonTo(&listTeamsResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &listTeamsResp, nil
}

// ListLogs method retrieves team logs from OpsGenie.
func (cli *OpsGenieTeamClient) ListLogs(req team.ListTeamLogsRequest) (*team.ListTeamLogsResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(teamLogsURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listTeamLogsResp team.ListTeamLogsResponse

	if err = resp.Body.FromJsonTo(&listTeamLogsResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &listTeamLogsResp, nil
}
