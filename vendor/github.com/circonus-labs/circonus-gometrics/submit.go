// Copyright 2016 Circonus, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package circonusgometrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/circonus-labs/circonus-gometrics/api"
	"github.com/hashicorp/go-retryablehttp"
)

func (m *CirconusMetrics) submit(output map[string]interface{}, newMetrics map[string]*api.CheckBundleMetric) {

	// update check if there are any new metrics or, if metric tags have been added since last submit
	m.check.UpdateCheck(newMetrics)

	str, err := json.Marshal(output)
	if err != nil {
		m.Log.Printf("[ERROR] marshling output %+v", err)
		return
	}

	numStats, err := m.trapCall(str)
	if err != nil {
		m.Log.Printf("[ERROR] %+v\n", err)
		return
	}

	if m.Debug {
		m.Log.Printf("[DEBUG] %d stats sent\n", numStats)
	}
}

func (m *CirconusMetrics) trapCall(payload []byte) (int, error) {
	trap, err := m.check.GetTrap()
	if err != nil {
		return 0, err
	}

	dataReader := bytes.NewReader(payload)

	req, err := retryablehttp.NewRequest("PUT", trap.URL.String(), dataReader)
	if err != nil {
		return 0, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	// keep last HTTP error in the event of retry failure
	var lastHTTPError error
	retryPolicy := func(resp *http.Response, err error) (bool, error) {
		if err != nil {
			lastHTTPError = err
			return true, err
		}
		// Check the response code. We retry on 500-range responses to allow
		// the server time to recover, as 500's are typically not permanent
		// errors and may relate to outages on the server side. This will catch
		// invalid response codes as well, like 0 and 999.
		if resp.StatusCode == 0 || resp.StatusCode >= 500 {
			body, readErr := ioutil.ReadAll(resp.Body)
			if readErr != nil {
				lastHTTPError = fmt.Errorf("- last HTTP error: %d %+v", resp.StatusCode, readErr)
			} else {
				lastHTTPError = fmt.Errorf("- last HTTP error: %d %s", resp.StatusCode, string(body))
			}
			return true, nil
		}
		return false, nil
	}

	client := retryablehttp.NewClient()
	if trap.URL.Scheme == "https" {
		client.HTTPClient.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     trap.TLS,
			DisableKeepAlives:   true,
			MaxIdleConnsPerHost: -1,
			DisableCompression:  true,
		}
	} else {
		client.HTTPClient.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   true,
			MaxIdleConnsPerHost: -1,
			DisableCompression:  true,
		}
	}
	client.RetryWaitMin = 1 * time.Second
	client.RetryWaitMax = 5 * time.Second
	client.RetryMax = 3
	// retryablehttp only groks log or no log
	// but, outputs everything as [DEBUG] messages
	if m.Debug {
		client.Logger = m.Log
	} else {
		client.Logger = log.New(ioutil.Discard, "", log.LstdFlags)
	}
	client.CheckRetry = retryPolicy

	attempts := -1
	client.RequestLogHook = func(logger *log.Logger, req *http.Request, retryNumber int) {
		attempts = retryNumber
	}

	resp, err := client.Do(req)
	if err != nil {
		if lastHTTPError != nil {
			return 0, fmt.Errorf("[ERROR] submitting: %+v %+v", err, lastHTTPError)
		}
		if attempts == client.RetryMax {
			m.check.RefreshTrap()
		}
		return 0, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		m.Log.Printf("[ERROR] reading body, proceeding. %s\n", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		m.Log.Printf("[ERROR] parsing body, proceeding. %v (%s)\n", err, body)
	}

	if resp.StatusCode != 200 {
		return 0, errors.New("[ERROR] bad response code: " + strconv.Itoa(resp.StatusCode))
	}
	switch v := response["stats"].(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	default:
	}
	return 0, errors.New("[ERROR] bad response type")
}
