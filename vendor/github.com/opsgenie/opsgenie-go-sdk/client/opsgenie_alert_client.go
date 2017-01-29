package client

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/opsgenie/opsgenie-go-sdk/alerts"
	"github.com/opsgenie/opsgenie-go-sdk/logging"
)

const (
	createAlertURL          = "/v1/json/alert"
	closeAlertURL           = "/v1/json/alert/close"
	deleteAlertURL          = "/v1/json/alert"
	getAlertURL             = "/v1/json/alert"
	listAlertsURL           = "/v1/json/alert"
	listAlertNotesURL       = "/v1/json/alert/note"
	listAlertLogsURL        = "/v1/json/alert/log"
	listAlertRecipientsURL  = "/v1/json/alert/recipient"
	acknowledgeAlertURL     = "/v1/json/alert/acknowledge"
	renotifyAlertURL        = "/v1/json/alert/renotify"
	takeOwnershipAlertURL   = "/v1/json/alert/takeOwnership"
	assignOwnershipAlertURL = "/v1/json/alert/assign"
	addTeamAlertURL         = "/v1/json/alert/team"
	addRecipientAlertURL    = "/v1/json/alert/recipient"
	addNoteAlertURL         = "/v1/json/alert/note"
	addTagsAlertURL         = "/v1/json/alert/tags"
	executeActionAlertURL   = "/v1/json/alert/executeAction"
	attachFileAlertURL      = "/v1/json/alert/attach"
	countAlertURL           = "/v1/json/alert/count"
	unacknowledgeAlertURL    = "/v1/json/alert/unacknowledge"
	snoozeAlertURL 		= "/v1/json/alert/snooze"
	removeTagsAlertURL	= "/v1/json/alert/tags"
	addDetailsAlertURL	= "/v1/json/alert/details"
	removeDetailsAlertURL	= "/v1/json/alert/details"
	escalateToNextAlertURL	= "/v1/json/alert/escalateToNext"
)

// OpsGenieAlertClient is the data type to make Alert API requests.
type OpsGenieAlertClient struct {
	OpsGenieClient
}

// SetOpsGenieClient sets the embedded OpsGenieClient type of the OpsGenieAlertClient.
func (cli *OpsGenieAlertClient) SetOpsGenieClient(ogCli OpsGenieClient) {
	cli.OpsGenieClient = ogCli
}

// Create method creates an alert at OpsGenie.
func (cli *OpsGenieAlertClient) Create(req alerts.CreateAlertRequest) (*alerts.CreateAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(createAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var createAlertResp alerts.CreateAlertResponse

	if err = resp.Body.FromJsonTo(&createAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &createAlertResp, nil
}

// Count method counts alerts at OpsGenie.
func (cli *OpsGenieAlertClient) Count(req alerts.CountAlertRequest) (*alerts.CountAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(countAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var countAlertResp alerts.CountAlertResponse

	if err = resp.Body.FromJsonTo(&countAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &countAlertResp, nil
}


// Close method closes an alert at OpsGenie.
func (cli *OpsGenieAlertClient) Close(req alerts.CloseAlertRequest) (*alerts.CloseAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(closeAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var closeAlertResp alerts.CloseAlertResponse

	if err = resp.Body.FromJsonTo(&closeAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &closeAlertResp, nil
}

// Delete method deletes an alert at OpsGenie.
func (cli *OpsGenieAlertClient) Delete(req alerts.DeleteAlertRequest) (*alerts.DeleteAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildDeleteRequest(deleteAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var deleteAlertResp alerts.DeleteAlertResponse

	if err = resp.Body.FromJsonTo(&deleteAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &deleteAlertResp, nil
}

// Get method retrieves specified alert details from OpsGenie.
func (cli *OpsGenieAlertClient) Get(req alerts.GetAlertRequest) (*alerts.GetAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(getAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var getAlertResp alerts.GetAlertResponse

	if err = resp.Body.FromJsonTo(&getAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &getAlertResp, nil
}

// List method retrieves alerts from OpsGenie.
func (cli *OpsGenieAlertClient) List(req alerts.ListAlertsRequest) (*alerts.ListAlertsResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(listAlertsURL, req))

	if resp == nil {
		return nil, errors.New(err.Error())
	}
	defer resp.Body.Close()

	var listAlertsResp alerts.ListAlertsResponse

	if err = resp.Body.FromJsonTo(&listAlertsResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &listAlertsResp, nil
}

// ListNotes method retrieves notes of an alert from OpsGenie.
func (cli *OpsGenieAlertClient) ListNotes(req alerts.ListAlertNotesRequest) (*alerts.ListAlertNotesResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(listAlertNotesURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listAlertNotesResp alerts.ListAlertNotesResponse

	if err = resp.Body.FromJsonTo(&listAlertNotesResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &listAlertNotesResp, nil
}

// ListLogs method retrieves activity logs of an alert from OpsGenie.
func (cli *OpsGenieAlertClient) ListLogs(req alerts.ListAlertLogsRequest) (*alerts.ListAlertLogsResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(listAlertLogsURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listAlertLogsResp alerts.ListAlertLogsResponse

	if err = resp.Body.FromJsonTo(&listAlertLogsResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &listAlertLogsResp, nil
}

// ListRecipients method retrieves recipients of an alert from OpsGenie.
func (cli *OpsGenieAlertClient) ListRecipients(req alerts.ListAlertRecipientsRequest) (*alerts.ListAlertRecipientsResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildGetRequest(listAlertRecipientsURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var listAlertRecipientsResp alerts.ListAlertRecipientsResponse

	if err = resp.Body.FromJsonTo(&listAlertRecipientsResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &listAlertRecipientsResp, nil
}

// Acknowledge method acknowledges an alert at OpsGenie.
func (cli *OpsGenieAlertClient) Acknowledge(req alerts.AcknowledgeAlertRequest) (*alerts.AcknowledgeAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(acknowledgeAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var acknowledgeAlertResp alerts.AcknowledgeAlertResponse

	if err = resp.Body.FromJsonTo(&acknowledgeAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &acknowledgeAlertResp, nil
}

// Renotify re-notifies recipients at OpsGenie.
func (cli *OpsGenieAlertClient) Renotify(req alerts.RenotifyAlertRequest) (*alerts.RenotifyAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(renotifyAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var renotifyAlertResp alerts.RenotifyAlertResponse

	if err = resp.Body.FromJsonTo(&renotifyAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &renotifyAlertResp, nil
}

// TakeOwnership method takes the ownership of an alert at OpsGenie.
func (cli *OpsGenieAlertClient) TakeOwnership(req alerts.TakeOwnershipAlertRequest) (*alerts.TakeOwnershipAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(takeOwnershipAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var takeOwnershipResp alerts.TakeOwnershipAlertResponse

	if err = resp.Body.FromJsonTo(&takeOwnershipResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &takeOwnershipResp, nil
}

// AssignOwner method assigns the specified user as the owner of the alert at OpsGenie.
func (cli *OpsGenieAlertClient) AssignOwner(req alerts.AssignOwnerAlertRequest) (*alerts.AssignOwnerAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(assignOwnershipAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var assignOwnerAlertResp alerts.AssignOwnerAlertResponse

	if err = resp.Body.FromJsonTo(&assignOwnerAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &assignOwnerAlertResp, nil
}

// AddTeam method adds a team to an alert at OpsGenie.
func (cli *OpsGenieAlertClient) AddTeam(req alerts.AddTeamAlertRequest) (*alerts.AddTeamAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(addTeamAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var addTeamAlertResp alerts.AddTeamAlertResponse

	if err = resp.Body.FromJsonTo(&addTeamAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &addTeamAlertResp, nil
}

// AddRecipient method adds recipient to an alert at OpsGenie.
func (cli *OpsGenieAlertClient) AddRecipient(req alerts.AddRecipientAlertRequest) (*alerts.AddRecipientAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(addRecipientAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var addRecipientAlertResp alerts.AddRecipientAlertResponse

	if err = resp.Body.FromJsonTo(&addRecipientAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &addRecipientAlertResp, nil
}

// AddNote method adds a note to an alert at OpsGenie.
func (cli *OpsGenieAlertClient) AddNote(req alerts.AddNoteAlertRequest) (*alerts.AddNoteAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(addNoteAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var addNoteAlertResp alerts.AddNoteAlertResponse

	if err = resp.Body.FromJsonTo(&addNoteAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &addNoteAlertResp, nil
}

// AddTags method adds tags to an alert at OpsGenie.
func (cli *OpsGenieAlertClient) AddTags(req alerts.AddTagsAlertRequest) (*alerts.AddTagsAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(addTagsAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var addTagsAlertResp alerts.AddTagsAlertResponse

	if err = resp.Body.FromJsonTo(&addTagsAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &addTagsAlertResp, nil
}

// ExecuteAction method executes a custom action on an alert at OpsGenie.
func (cli *OpsGenieAlertClient) ExecuteAction(req alerts.ExecuteActionAlertRequest) (*alerts.ExecuteActionAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(executeActionAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var executeActionAlertResp alerts.ExecuteActionAlertResponse

	if err = resp.Body.FromJsonTo(&executeActionAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &executeActionAlertResp, nil
}

// UnAcknowledge method unacknowledges an alert at OpsGenie.
func (cli *OpsGenieAlertClient) UnAcknowledge(req alerts.UnAcknowledgeAlertRequest) (*alerts.UnAcknowledgeAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(unacknowledgeAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var unacknowledgeAlertResp alerts.UnAcknowledgeAlertResponse

	if err = resp.Body.FromJsonTo(&unacknowledgeAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &unacknowledgeAlertResp, nil
}

// Snooze method snoozes an alert at OpsGenie.
func (cli *OpsGenieAlertClient) Snooze(req alerts.SnoozeAlertRequest) (*alerts.SnoozeAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(snoozeAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var snoozeAlertResp alerts.SnoozeAlertResponse

	if err = resp.Body.FromJsonTo(&snoozeAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &snoozeAlertResp, nil
}

// RemoveTags method removes tags from an alert at OpsGenie.
func (cli *OpsGenieAlertClient) RemoveTags(req alerts.RemoveTagsAlertRequest) (*alerts.RemoveTagsAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildDeleteRequest(removeTagsAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var removeTagsAlertResp alerts.RemoveTagsAlertResponse

	if err = resp.Body.FromJsonTo(&removeTagsAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &removeTagsAlertResp, nil
}

// AddDetails method adds details to an alert at OpsGenie.
func (cli *OpsGenieAlertClient) AddDetails(req alerts.AddDetailsAlertRequest) (*alerts.AddDetailsAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(addDetailsAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var addDetailsAlertResp alerts.AddDetailsAlertResponse

	if err = resp.Body.FromJsonTo(&addDetailsAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &addDetailsAlertResp, nil
}

// RemoveDetails method removes details from an alert at OpsGenie.
func (cli *OpsGenieAlertClient) RemoveDetails(req alerts.RemoveDetailsAlertRequest) (*alerts.RemoveDetailsAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildDeleteRequest(removeDetailsAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var removeDetailsAlertResp alerts.RemoveDetailsAlertResponse

	if err = resp.Body.FromJsonTo(&removeDetailsAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &removeDetailsAlertResp, nil
}

// UnAcknowledge method unacknowledges an alert at OpsGenie.
func (cli *OpsGenieAlertClient) EscalateToNext(req alerts.EscalateToNextAlertRequest) (*alerts.EscalateToNextAlertResponse, error) {
	req.APIKey = cli.apiKey
	resp, err := cli.sendRequest(cli.buildPostRequest(escalateToNextAlertURL, req))

	if resp == nil {
		return nil, err
	}
	defer resp.Body.Close()

	var escalateToNextAlertResp alerts.EscalateToNextAlertResponse

	if err = resp.Body.FromJsonTo(&escalateToNextAlertResp); err != nil {
		message := "Server response can not be parsed, " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	return &escalateToNextAlertResp, nil
}

// AttachFile method attaches a file to an alert at OpsGenie.
func (cli *OpsGenieAlertClient) AttachFile(req alerts.AttachFileAlertRequest) (*alerts.AttachFileAlertResponse, error) {
	req.APIKey = cli.apiKey
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	path := req.Attachment.Name()
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		message := "Attachment can not be opened for reading. " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	// add the attachment
	fw, err := w.CreateFormFile("attachment", filepath.Base(path))
	if err != nil {
		message := "Can not build the request with the field attachment. " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	if _, err := io.Copy(fw, file); err != nil {
		message := "Can not copy the attachment into the request. " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	// Add the other fields
	// empty fields should not be placed into the request
	// otherwise it yields an incomplete boundary exception
	if req.APIKey != "" {
		if err = writeField(*w, "apiKey", req.APIKey); err != nil {
			return nil, err
		}
	}
	if req.ID != "" {
		if err = writeField(*w, "id", req.ID); err != nil {
			return nil, err
		}
	}
	if req.Alias != "" {
		if err = writeField(*w, "alias", req.Alias); err != nil {
			return nil, err
		}
	}
	if req.User != "" {
		if err = writeField(*w, "user", req.User); err != nil {
			return nil, err
		}
	}
	if req.Source != "" {
		if err = writeField(*w, "source", req.Source); err != nil {
			return nil, err
		}
	}
	if req.IndexFile != "" {
		if err = writeField(*w, "indexFile", req.IndexFile); err != nil {
			return nil, err
		}
	}
	if req.Note != "" {
		if err = writeField(*w, "note", req.Note); err != nil {
			return nil, err
		}
	}

	w.Close()
	httpReq, err := http.NewRequest("POST", cli.opsGenieAPIURL+attachFileAlertURL, &b)
	if err != nil {
		message := "Can not create the multipart/form-data request. " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}
	httpReq.Header.Set("Content-Type", w.FormDataContentType())
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		Proxy:           http.ProxyFromEnvironment,
		Dial: func(netw, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(netw, addr, cli.httpTransportSettings.ConnectionTimeout)
			if err != nil {
				message := "Error occurred while connecting: " + err.Error()
				logging.Logger().Warn(message)
				return nil, errors.New(message)
			}
			conn.SetDeadline(time.Now().Add(cli.httpTransportSettings.RequestTimeout))
			return conn, nil
		},
	}
	client := &http.Client{Transport: transport}
	// proxy settings
	if cli.proxy != nil {
		proxyURL, proxyErr := url.Parse(cli.proxy.toString())
		if proxyErr != nil {
			message := "Can not set the proxy configuration " + proxyErr.Error()
			logging.Logger().Warn(message)
			return nil, errors.New(message)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	url := httpReq.URL.String()
	logging.Logger().Info("Executing OpsGenie request to [" + url + "] with multipart data.")
	var res *http.Response
	for i := 0; i < cli.httpTransportSettings.MaxRetryAttempts; i++ {
		res, err = client.Do(httpReq)
		if err == nil {
			defer res.Body.Close()
			break
		}
		if res != nil {
			logging.Logger().Info(fmt.Sprintf("Retrying request [%s] ResponseCode:[%d]. RetryCount: %d", url, res.StatusCode, (i + 1)))
		} else {
			logging.Logger().Info(fmt.Sprintf("Retrying request [%s] Reason:[%s]. RetryCount: %d", url, err.Error(), (i + 1)))
		}
		time.Sleep(timeSleepBetweenRequests)
	}

	if err != nil {
		message := "Can not attach the file, unable to send the request. " + err.Error()
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	httpStatusCode := res.StatusCode
	if httpStatusCode >= 400 {
		body, err := ioutil.ReadAll(res.Body)
		if err == nil {
			return nil, errorMessage(httpStatusCode, string(body[:]))
		}
		message := fmt.Sprint("Couldn't read the response, %s", err.Error())
		logging.Logger().Warn(message)
		return nil, errors.New(message)
	}

	attachFileAlertResp := alerts.AttachFileAlertResponse{Status: res.Status, Code: res.StatusCode}
	return &attachFileAlertResp, nil
}

func writeField(w multipart.Writer, fieldName string, fieldVal string) error {
	if err := w.WriteField(fieldName, fieldVal); err != nil {
		message := "Can not write field " + fieldName + " into the request. Reason: " + err.Error()
		logging.Logger().Warn(message)
		return errors.New(message)
	}
	return nil
}
