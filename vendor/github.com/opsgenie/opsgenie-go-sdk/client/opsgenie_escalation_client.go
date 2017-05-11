package client

import (
	"errors"
	"github.com/opsgenie/opsgenie-go-sdk/escalation"
	"github.com/opsgenie/opsgenie-go-sdk/logging"
)

const (
	escalationURL          = "/v1/json/escalation"
)

// OpsGenieEscalationClient is the data type to make Escalation API requests.
type OpsGenieEscalationClient struct {
	OpsGenieClient
}

// SetOpsGenieClient sets the embedded OpsGenieClient type of the OpsGenieEscalationClient.
func (cli *OpsGenieEscalationClient) SetOpsGenieClient(ogCli OpsGenieClient) {
	cli.OpsGenieClient = ogCli
}

// Create method creates a escalation at OpsGenie.
func (cli *OpsGenieEscalationClient) Create(req escalation.CreateEscalationRequest) (*escalation.CreateEscalationResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(escalationURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var createEscalationResp escalation.CreateEscalationResponse

	if err = resp.Body.FromJsonTo(&createEscalationResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &createEscalationResp, nil
}

// Update method updates a escalation at OpsGenie.
func (cli *OpsGenieEscalationClient) Update(req escalation.UpdateEscalationRequest) (*escalation.UpdateEscalationResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(escalationURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updateEscalationResp escalation.UpdateEscalationResponse

	if err = resp.Body.FromJsonTo(&updateEscalationResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &updateEscalationResp, nil
}

// Delete method deletes a escalation at OpsGenie.
func (cli *OpsGenieEscalationClient) Delete(req escalation.DeleteEscalationRequest) (*escalation.DeleteEscalationResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildDeleteRequest(escalationURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var deleteEscalationResp escalation.DeleteEscalationResponse

	if err = resp.Body.FromJsonTo(&deleteEscalationResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &deleteEscalationResp, nil
}

// Get method retrieves specified escalation details from OpsGenie.
func (cli *OpsGenieEscalationClient) Get(req escalation.GetEscalationRequest) (*escalation.GetEscalationResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(escalationURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var getEscalationResp escalation.GetEscalationResponse

	if err = resp.Body.FromJsonTo(&getEscalationResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &getEscalationResp, nil
}

// List method retrieves escalations from OpsGenie.
func (cli *OpsGenieEscalationClient) List(req escalation.ListEscalationsRequest) (*escalation.ListEscalationsResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(escalationURL, req))

	if resp == nil {
		return nil, errors.New(err.Error())
	}
	defer resp.Body.Close()

	var listEscalationsResp escalation.ListEscalationsResponse

	if err = resp.Body.FromJsonTo(&listEscalationsResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &listEscalationsResp, nil
}
