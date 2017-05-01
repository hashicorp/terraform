package oneandone

import (
	"errors"
	"net/http"
	"time"
)

type MonServerUsageSummary struct {
	Identity
	Agent  *monitoringAgent  `json:"agent,omitempty"`
	Alerts *monitoringAlerts `json:"alerts,omitempty"`
	Status *monitoringStatus `json:"status,omitempty"`
	ApiPtr
}

type MonServerUsageDetails struct {
	Identity
	Status         *statusState       `json:"status,omitempty"`
	Agent          *monitoringAgent   `json:"agent,omitempty"`
	Alerts         *monitoringAlerts  `json:"alerts,omitempty"`
	CpuStatus      *utilizationStatus `json:"cpu,omitempty"`
	DiskStatus     *utilizationStatus `json:"disk,omitempty"`
	RamStatus      *utilizationStatus `json:"ram,omitempty"`
	PingStatus     *pingStatus        `json:"internal_ping,omitempty"`
	TransferStatus *transferStatus    `json:"transfer,omitempty"`
	ApiPtr
}

type monitoringStatus struct {
	State        string       `json:"state,omitempty"`
	Cpu          *statusState `json:"cpu,omitempty"`
	Disk         *statusState `json:"disk,omitempty"`
	InternalPing *statusState `json:"internal_ping,omitempty"`
	Ram          *statusState `json:"ram,omitempty"`
	Transfer     *statusState `json:"transfer,omitempty"`
}

type utilizationStatus struct {
	CriticalThreshold int         `json:"critical,omitempty"`
	WarningThreshold  int         `json:"warning,omitempty"`
	Status            string      `json:"status,omitempty"`
	Data              []usageData `json:"data,omitempty"`
	Unit              *usageUnit  `json:"unit,omitempty"`
}

type pingStatus struct {
	CriticalThreshold int        `json:"critical,omitempty"`
	WarningThreshold  int        `json:"warning,omitempty"`
	Status            string     `json:"status,omitempty"`
	Data              []pingData `json:"data,omitempty"`
	Unit              *pingUnit  `json:"unit,omitempty"`
}

type transferStatus struct {
	CriticalThreshold int            `json:"critical,omitempty"`
	WarningThreshold  int            `json:"warning,omitempty"`
	Status            string         `json:"status,omitempty"`
	Data              []transferData `json:"data,omitempty"`
	Unit              *transferUnit  `json:"unit,omitempty"`
}

type monitoringAgent struct {
	AgentInstalled       bool `json:"agent_installed"`
	MissingAgentAlert    bool `json:"missing_agent_alert"`
	MonitoringNeedsAgent bool `json:"monitoring_needs_agent"`
}

type monitoringAlerts struct {
	Ports     *monitoringAlertInfo `json:"ports,omitempty"`
	Process   *monitoringAlertInfo `json:"process,omitempty"`
	Resources *monitoringAlertInfo `json:"resources,omitempty"`
}

type monitoringAlertInfo struct {
	Ok       int `json:"ok"`
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
}

type usageData struct {
	Date        string  `json:"date,omitempty"`
	UsedPercent float32 `json:"used_percent"`
}

type usageUnit struct {
	UsedPercent string `json:"used_percent,omitempty"`
}

type pingUnit struct {
	PackagesLost string `json:"pl,omitempty"`
	AccessTime   string `json:"rta,omitempty"`
}

type pingData struct {
	Date         string  `json:"date,omitempty"`
	PackagesLost int     `json:"pl"`
	AccessTime   float32 `json:"rta"`
}

type transferUnit struct {
	Downstream string `json:"downstream,omitempty"`
	Upstream   string `json:"upstream,omitempty"`
}

type transferData struct {
	Date       string `json:"date,omitempty"`
	Downstream int    `json:"downstream"`
	Upstream   int    `json:"upstream"`
}

// GET /monitoring_center
func (api *API) ListMonitoringServersUsages(args ...interface{}) ([]MonServerUsageSummary, error) {
	url, err := processQueryParams(createUrl(api, monitorCenterPathSegment), args...)
	if err != nil {
		return nil, err
	}
	result := []MonServerUsageSummary{}
	err = api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	for index, _ := range result {
		result[index].api = api
	}
	return result, nil
}

// GET /monitoring_center/{server_id}
func (api *API) GetMonitoringServerUsage(ser_id string, period string, dates ...time.Time) (*MonServerUsageDetails, error) {
	if period == "" {
		return nil, errors.New("Time period must be provided.")
	}

	params := make(map[string]interface{}, len(dates)+1)
	params["period"] = period

	if len(dates) == 2 {
		if dates[0].After(dates[1]) {
			return nil, errors.New("Start date cannot be after end date.")
		}

		params["start_date"] = dates[0].Format(time.RFC3339)
		params["end_date"] = dates[1].Format(time.RFC3339)

	} else if len(dates) > 0 {
		return nil, errors.New("Start and end dates must be provided.")
	}
	url := createUrl(api, monitorCenterPathSegment, ser_id)
	url = appendQueryParams(url, params)
	result := new(MonServerUsageDetails)
	err := api.Client.Get(url, &result, http.StatusOK)
	if err != nil {
		return nil, err
	}
	result.api = api
	return result, nil
}
