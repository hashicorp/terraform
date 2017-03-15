package puppetdb

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

type PuppetDBClient struct {
	URL string
}

type PuppetDBResp struct {
	Error                        error  `json:"error"`
	Certname                     string `json:"certname"`
	Deactivated                  string `json:"deactivated"`
	Expired                      string `json:"expired"`
	CachedCatalogStatus          string `json:"cached_catalog_status"`
	CatalogEnvironment           string `json:"catalog_environment"`
	FactsEnvironment             string `json:"facts_environment"`
	ReportEnvironment            string `json:"report_environment"`
	CatalogTimestamp             string `json:"catalog_timestamp"`
	FactsTimestamp               string `json:"facts_timestamp"`
	ReportTimestamp              string `json:"report_timestamp"`
	LatestReportCorrectiveChange string `json:"latest_report_corrective_change"`
	LatestReportHash             string `json:"latest_report_hash"`
	LatestReportNoop             bool   `json:"latest_report_noop"`
	LatestReportNoopPending      bool   `json:"latest_report_noop_pending"`
	LatestReportStatus           string `json:"latest_report_status"`
}

type commandsPayload struct {
	Command string            `json:"command"`
	Version int               `json:"version"`
	Payload map[string]string `json:"payload"`
}

func (p *PuppetDBClient) Query(query string, verb string, payload string) (pdbResp PuppetDBResp, err error) {
	url := p.URL + "/pdb/" + query
	form := strings.NewReader(payload)
	req, err := http.NewRequest(verb, url, form)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	json.Unmarshal(body, &pdbResp)

	if err = pdbResp.Error; err != nil {
		return
	}

	return pdbResp, nil
}
