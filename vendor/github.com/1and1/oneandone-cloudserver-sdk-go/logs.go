package oneandone

import (
	"net/http"
	"time"
)

type Log struct {
	ApiPtr
	idField
	typeField
	CloudPanelId string    `json:"cloudpanel_id,omitempty"`
	SiteId       string    `json:"site_id,omitempty"`
	StartDate    string    `json:"start_date,omitempty"`
	EndDate      string    `json:"end_date,omitempty"`
	Action       string    `json:"action,omitempty"`
	Duration     int       `json:"duration"`
	Status       *Status   `json:"Status,omitempty"`
	Resource     *Identity `json:"resource,omitempty"`
	User         *Identity `json:"user,omitempty"`
}

// GET /logs
func (api *API) ListLogs(period string, sd *time.Time, ed *time.Time, args ...interface{}) ([]Log, error) {
	result := []Log{}
	url, err := processQueryParamsExt(createUrl(api, logPathSegment), period, sd, ed, args...)
	if err != nil {
		return nil, err
	}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// GET /logs/{id}
func (api *API) GetLog(log_id string) (*Log, error) {
	result := new(Log)
	url := createUrl(api, logPathSegment, log_id)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}
