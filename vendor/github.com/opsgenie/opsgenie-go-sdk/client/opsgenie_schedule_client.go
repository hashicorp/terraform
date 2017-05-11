package client

import (
	"errors"
	"fmt"

	"github.com/opsgenie/opsgenie-go-sdk/schedule"
	"github.com/opsgenie/opsgenie-go-sdk/logging"
)

const (
	scheduleURL          = "/v1/json/schedule"
)

// OpsGenieScheduleClient is the data type to make Schedule API requests.
type OpsGenieScheduleClient struct {
	OpsGenieClient
}

// SetOpsGenieClient sets the embedded OpsGenieClient type of the OpsGenieScheduleClient.
func (cli *OpsGenieScheduleClient) SetOpsGenieClient(ogCli OpsGenieClient) {
	cli.OpsGenieClient = ogCli
}

// Create method creates a schedule at OpsGenie.
func (cli *OpsGenieScheduleClient) Create(req schedule.CreateScheduleRequest) (*schedule.CreateScheduleResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(scheduleURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var createScheduleResp schedule.CreateScheduleResponse

	if err = resp.Body.FromJsonTo(&createScheduleResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &createScheduleResp, nil
}

// Update method updates a schedule at OpsGenie.
func (cli *OpsGenieScheduleClient) Update(req schedule.UpdateScheduleRequest) (*schedule.UpdateScheduleResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(scheduleURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var updateScheduleResp schedule.UpdateScheduleResponse

	if err = resp.Body.FromJsonTo(&updateScheduleResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &updateScheduleResp, nil
}

// Delete method deletes a schedule at OpsGenie.
func (cli *OpsGenieScheduleClient) Delete(req schedule.DeleteScheduleRequest) (*schedule.DeleteScheduleResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildDeleteRequest(scheduleURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var deleteScheduleResp schedule.DeleteScheduleResponse

	if err = resp.Body.FromJsonTo(&deleteScheduleResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &deleteScheduleResp, nil
}

// Get method retrieves specified schedule details from OpsGenie.
func (cli *OpsGenieScheduleClient) Get(req schedule.GetScheduleRequest) (*schedule.GetScheduleResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(scheduleURL, req))
	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()
	var getScheduleResp schedule.GetScheduleResponse

	if err = resp.Body.FromJsonTo(&getScheduleResp); err != nil {
		fmt.Println("Error parsing json")
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	fmt.Printf("%+v", getScheduleResp)
	return &getScheduleResp, nil
}

// List method retrieves schedules from OpsGenie.
func (cli *OpsGenieScheduleClient) List(req schedule.ListSchedulesRequest) (*schedule.ListSchedulesResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(scheduleURL, req))

	if resp == nil {
		return nil, errors.New(err.Error())
	}
	defer resp.Body.Close()

	var listSchedulesResp schedule.ListSchedulesResponse

	if err = resp.Body.FromJsonTo(&listSchedulesResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	
	return &listSchedulesResp, nil
}
