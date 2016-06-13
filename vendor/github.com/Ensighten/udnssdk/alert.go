package udnssdk

import (
	"log"
	"net/http"
	"time"
)

// AlertsService manages Alerts
type AlertsService struct {
	client *Client
}

// ProbeAlertDataDTO wraps a probe alert response
type ProbeAlertDataDTO struct {
	PoolRecord      string    `json:"poolRecord"`
	ProbeType       string    `json:"probeType"`
	ProbeStatus     string    `json:"probeStatus"`
	AlertDate       time.Time `json:"alertDate"`
	FailoverOccured bool      `json:"failoverOccured"`
	OwnerName       string    `json:"ownerName"`
	Status          string    `json:"status"`
}

// Equal compares to another ProbeAlertDataDTO, but uses time.Equals to compare semantic equvalance of AlertDate
func (a ProbeAlertDataDTO) Equal(b ProbeAlertDataDTO) bool {
	return a.PoolRecord == b.PoolRecord &&
		a.ProbeType == b.ProbeType &&
		a.ProbeStatus == b.ProbeStatus &&
		a.AlertDate.Equal(b.AlertDate) &&
		a.FailoverOccured == b.FailoverOccured &&
		a.OwnerName == b.OwnerName &&
		a.Status == b.Status
}

// ProbeAlertDataListDTO wraps the response for an index of probe alerts
type ProbeAlertDataListDTO struct {
	Alerts     []ProbeAlertDataDTO `json:"alerts"`
	Queryinfo  QueryInfo           `json:"queryInfo"`
	Resultinfo ResultInfo          `json:"resultInfo"`
}

// Select returns all probe alerts with a RRSetKey
func (s *AlertsService) Select(k RRSetKey) ([]ProbeAlertDataDTO, error) {
	// TODO: Sane Configuration for timeouts / retries
	maxerrs := 5
	waittime := 5 * time.Second

	// init accumulators
	as := []ProbeAlertDataDTO{}
	offset := 0
	errcnt := 0

	for {
		reqAlerts, ri, res, err := s.SelectWithOffset(k, offset)
		if err != nil {
			if res != nil && res.StatusCode >= 500 {
				errcnt = errcnt + 1
				if errcnt < maxerrs {
					time.Sleep(waittime)
					continue
				}
			}
			return as, err
		}

		log.Printf("ResultInfo: %+v\n", ri)
		for _, a := range reqAlerts {
			as = append(as, a)
		}
		if ri.ReturnedCount+ri.Offset >= ri.TotalCount {
			return as, nil
		}
		offset = ri.ReturnedCount + ri.Offset
		continue
	}
}

// SelectWithOffset returns the probe alerts with a RRSetKey, accepting an offset
func (s *AlertsService) SelectWithOffset(k RRSetKey, offset int) ([]ProbeAlertDataDTO, ResultInfo, *http.Response, error) {
	var ald ProbeAlertDataListDTO

	uri := k.AlertsQueryURI(offset)
	res, err := s.client.get(uri, &ald)

	as := []ProbeAlertDataDTO{}
	for _, a := range ald.Alerts {
		as = append(as, a)
	}
	return as, ald.Resultinfo, res, err
}
