/*
 * Datadog API for Go
 *
 * Please see the included LICENSE file for licensing information.
 *
 * Copyright 2013 by authors and contributors.
 */

package datadog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
)

const (
	RateLimitHeader     = "X-RateLimit-Limit"
	RatePeriodHeader    = "X-RateLimit-Period"
	RateRemainingHeader = "X-RateLimit-Remaining"
	RateResetHeader     = "X-RateLimit-Reset"
)

// uriForAPI is to be called with something like "/v1/events" and it will give
// the proper request URI to be posted to.
func (client *Client) uriForAPI(api string) string {
	url := os.Getenv("DATADOG_HOST")
	if url == "" {
		url = "https://app.datadoghq.com"
	}
	if strings.Index(api, "?") > -1 {
		return url + "/api" + api + "&api_key=" +
			client.apiKey + "&application_key=" + client.appKey
	} else {
		return url + "/api" + api + "?api_key=" +
			client.apiKey + "&application_key=" + client.appKey
	}
}

// doJsonRequest is the simplest type of request: a method on a URI that returns
// some JSON result which we unmarshal into the passed interface.
func (client *Client) doJsonRequest(method, api string,
	reqbody, out interface{}) error {
	// Handle the body if they gave us one.
	var bodyreader io.Reader
	if method != "GET" && reqbody != nil {
		bjson, err := json.Marshal(reqbody)
		if err != nil {
			return err
		}
		bodyreader = bytes.NewReader(bjson)
	}

	req, err := http.NewRequest(method, client.uriForAPI(api), bodyreader)
	if err != nil {
		return err
	}
	if bodyreader != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	// Perform the request and retry it if it's not a POST request
	var resp *http.Response
	if method == "POST" {
		resp, err = client.HttpClient.Do(req)
	} else {
		resp, err = client.doRequestWithRetries(req, client.RetryTimeout)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := client.getRateLimit(resp); err != nil {
		if err.(*RateLimit).Remaining == 0 {
			return err
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("API error %s: %s", resp.Status, body)
	}

	// If they don't care about the body, then we don't care to give them one,
	// so bail out because we're done.
	if out == nil {
		return nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// If we got no body, by default let's just make an empty JSON dict. This
	// saves us some work in other parts of the code.
	if len(body) == 0 {
		body = []byte{'{', '}'}
	}

	if err := json.Unmarshal(body, &out); err != nil {
		return err
	}
	return nil
}

func (client *Client) getRateLimit(response *http.Response) error {

	ratelimit := response.Header.Get(RateLimitHeader)
	if ratelimit == "" {
		return nil
	}

	rateperiod := response.Header.Get(RatePeriodHeader)
	rateremaining := response.Header.Get(RateRemainingHeader)
	ratereset := response.Header.Get(RateResetHeader)

	limit, _ := strconv.Atoi(ratelimit)
	period, _ := strconv.Atoi(rateperiod)
	remaining, _ := strconv.Atoi(rateremaining)
	reset, _ := strconv.Atoi(ratereset)

	ratelimiterror := NewRateLimit(limit, period, remaining, reset)

	return error(ratelimiterror)
}

// doRequestWithRetries performs an HTTP request repeatedly for maxTime or until
// no error and no HTTP response code higher than 299 is returned.
func (client *Client) doRequestWithRetries(req *http.Request, maxTime time.Duration) (*http.Response, error) {
	var (
		err  error
		resp *http.Response
		bo   = backoff.NewExponentialBackOff()
	)
	bo.MaxElapsedTime = maxTime

	err = backoff.Retry(func() error {
		resp, err = client.HttpClient.Do(req)
		if err != nil {
			return err
		}

		if err := client.getRateLimit(resp); err != nil {
			if err.(*RateLimit).Remaining == 0 {
				return err
			}
		}

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return errors.New("API error: " + resp.Status)
		}
		return nil
	}, bo)

	return resp, err
}
