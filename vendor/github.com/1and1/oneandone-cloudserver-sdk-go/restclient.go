package oneandone

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	p_url "net/url"
	"time"
)

type restClient struct {
	token string
}

func newRestClient(token string) *restClient {
	restClient := new(restClient)
	restClient.token = token
	return restClient
}

func (c *restClient) Get(url string, result interface{}, expectedStatus int) error {
	return c.doRequest(url, "GET", nil, result, expectedStatus)
}

func (c *restClient) Delete(url string, requestBody interface{}, result interface{}, expectedStatus int) error {
	return c.doRequest(url, "DELETE", requestBody, result, expectedStatus)
}

func (c *restClient) Post(url string, requestBody interface{}, result interface{}, expectedStatus int) error {
	return c.doRequest(url, "POST", requestBody, result, expectedStatus)
}

func (c *restClient) Put(url string, requestBody interface{}, result interface{}, expectedStatus int) error {
	return c.doRequest(url, "PUT", requestBody, result, expectedStatus)
}

func (c *restClient) doRequest(url string, method string, requestBody interface{}, result interface{}, expectedStatus int) error {
	var bodyData io.Reader
	if requestBody != nil {
		data, _ := json.Marshal(requestBody)
		bodyData = bytes.NewBuffer(data)
	}

	request, err := http.NewRequest(method, url, bodyData)
	if err != nil {
		return err
	}

	request.Header.Add("X-Token", c.token)
	request.Header.Add("Content-Type", "application/json")
	client := http.Client{}
	response, err := client.Do(request)
	if err = isError(response, expectedStatus, err); err != nil {
		return err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return c.unmarshal(body, result)
}

func (c *restClient) unmarshal(data []byte, result interface{}) error {
	err := json.Unmarshal(data, result)
	if err != nil {
		// handle the case when the result is an empty array instead of an object
		switch err.(type) {
		case *json.UnmarshalTypeError:
			var ra []interface{}
			e := json.Unmarshal(data, &ra)
			if e != nil {
				return e
			} else if len(ra) > 0 {
				return err
			}
			return nil
		default:
			return err
		}
	}

	return nil
}

func isError(response *http.Response, expectedStatus int, err error) error {
	if err != nil {
		return err
	}
	if response != nil {
		if response.StatusCode == expectedStatus {
			// we got a response with the expected HTTP status code, hence no error
			return nil
		}
		body, _ := ioutil.ReadAll(response.Body)
		// extract the API's error message to be returned later
		er_resp := new(errorResponse)
		err = json.Unmarshal(body, er_resp)
		if err != nil {
			return err
		}

		return apiError{response.StatusCode, fmt.Sprintf("Type: %s; Message: %s", er_resp.Type, er_resp.Message)}
	}
	return errors.New("Generic error - no response from the REST API service.")
}

func createUrl(api *API, sections ...interface{}) string {
	url := api.Endpoint
	for _, section := range sections {
		url += "/" + fmt.Sprint(section)
	}
	return url
}

func makeParameterMap(args ...interface{}) (map[string]interface{}, error) {
	qps := make(map[string]interface{}, len(args))
	var is_true bool
	var page, per_page int
	var sort, query, fields string

	for i, p := range args {
		switch i {
		case 0:
			page, is_true = p.(int)
			if !is_true {
				return nil, errors.New("1st parameter must be a page number (integer).")
			} else if page > 0 {
				qps["page"] = page
			}
		case 1:
			per_page, is_true = p.(int)
			if !is_true {
				return nil, errors.New("2nd parameter must be a per_page number (integer).")
			} else if per_page > 0 {
				qps["per_page"] = per_page
			}
		case 2:
			sort, is_true = p.(string)
			if !is_true {
				return nil, errors.New("3rd parameter must be a sorting property string (e.g. 'name' or '-name').")
			} else if sort != "" {
				qps["sort"] = sort
			}
		case 3:
			query, is_true = p.(string)
			if !is_true {
				return nil, errors.New("4th parameter must be a query string to look for the response.")
			} else if query != "" {
				qps["q"] = query
			}
		case 4:
			fields, is_true = p.(string)
			if !is_true {
				return nil, errors.New("5th parameter must be fields properties string (e.g. 'id,name').")
			} else if fields != "" {
				qps["fields"] = fields
			}
		default:
			return nil, errors.New("Wrong number of parameters.")
		}
	}
	return qps, nil
}

func processQueryParams(url string, args ...interface{}) (string, error) {
	if len(args) > 0 {
		params, err := makeParameterMap(args...)
		if err != nil {
			return "", err
		}
		url = appendQueryParams(url, params)
	}
	return url, nil
}

func processQueryParamsExt(url string, period string, sd *time.Time, ed *time.Time, args ...interface{}) (string, error) {
	var qm map[string]interface{}
	var err error
	if len(args) > 0 {
		qm, err = makeParameterMap(args...)
		if err != nil {
			return "", err
		}
	} else {
		qm = make(map[string]interface{}, 3)
	}
	qm["period"] = period
	if sd != nil && ed != nil {
		if sd.After(*ed) {
			return "", errors.New("Start date cannot be after end date.")
		}
		qm["start_date"] = sd.Format(time.RFC3339)
		qm["end_date"] = ed.Format(time.RFC3339)
	}
	url = appendQueryParams(url, qm)
	return url, nil
}

func appendQueryParams(url string, params map[string]interface{}) string {
	queryUrl, _ := p_url.Parse(url)
	parameters := p_url.Values{}
	for key, value := range params {
		parameters.Add(key, fmt.Sprintf("%v", value))
	}
	queryUrl.RawQuery = parameters.Encode()
	return queryUrl.String()
}
