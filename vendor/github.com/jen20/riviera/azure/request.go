package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/mitchellh/mapstructure"
)

type Request struct {
	URI      *string             `json:"-"`
	location *string             `json:"location,omitempty"`
	tags     *map[string]*string `json:"tags,omitempty"`
	etag     *string             `json:"etag,omitempty"`
	Command  ApiCall             `json:"properties,omitempty"`

	client *Client
}

func readLocation(req interface{}) (string, bool) {
	var value reflect.Value
	if reflect.ValueOf(req).Kind() == reflect.Ptr {
		value = reflect.ValueOf(req).Elem()
	} else {
		value = reflect.ValueOf(req)
	}

	for i := 0; i < value.NumField(); i++ { // iterates through every struct type field
		tag := value.Type().Field(i).Tag // returns the tag string
		if tag.Get("riviera") == "location" {
			return value.Field(i).String(), true
		}
	}
	return "", false
}

func readTags(req interface{}) (map[string]*string, bool) {
	var value reflect.Value
	if reflect.ValueOf(req).Kind() == reflect.Ptr {
		value = reflect.ValueOf(req).Elem()
	} else {
		value = reflect.ValueOf(req)
	}

	for i := 0; i < value.NumField(); i++ { // iterates through every struct type field
		tag := value.Type().Field(i).Tag // returns the tag string
		if tag.Get("riviera") == "tags" {
			tags := value.Field(i)
			return tags.Interface().(map[string]*string), true
		}
	}
	return make(map[string]*string), false
}

func (request *Request) pollForAsynchronousResponse(acceptedResponse *http.Response) (*http.Response, error) {
	var resp *http.Response = acceptedResponse

	for {
		if resp.StatusCode != http.StatusAccepted {
			return resp, nil
		}

		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			retryTime, err := strconv.Atoi(strings.TrimSpace(retryAfter))
			if err != nil {
				return nil, err
			}

			request.client.logger.Printf("[INFO] Polling pausing for %d seconds as per Retry-After header", retryTime)
			time.Sleep(time.Duration(retryTime) * time.Second)
		}

		pollLocation, err := resp.Location()
		if err != nil {
			return nil, err
		}

		request.client.logger.Printf("[INFO] Polling %q for operation completion", pollLocation.String())
		req, err := retryablehttp.NewRequest("GET", pollLocation.String(), bytes.NewReader([]byte{}))
		if err != nil {
			return nil, err
		}

		err = request.client.tokenRequester.addAuthorizationToRequest(req)
		if err != nil {
			return nil, err
		}

		resp, err := request.client.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusAccepted {
			continue
		}

		return resp, err
	}
}

func (r *Request) Execute() (*Response, error) {
	apiInfo := r.Command.ApiInfo()

	var urlString string

	// Base URL should already be validated by now so Parse is safe without error handling
	urlObj, _ := url.Parse(r.client.BaseUrl)

	// Determine whether to use the URLPathFunc or the URI explictly set in the request
	if r.URI == nil {
		urlObj.Path = fmt.Sprintf("/subscriptions/%s/%s", r.client.subscriptionID, strings.TrimPrefix(apiInfo.URLPathFunc(), "/"))
		urlString = urlObj.String()
	} else {
		urlObj.Path = *r.URI
		urlString = urlObj.String()
	}

	// Encode the request body if necessary
	body := bytes.NewReader([]byte{})
	if apiInfo.HasBody() {
		bodyStruct := struct {
			Location   *string             `json:"location,omitempty"`
			Tags       *map[string]*string `json:"tags,omitempty"`
			Properties interface{}         `json:"properties"`
		}{
			Properties: r.Command,
		}

		if location, hasLocation := readLocation(r.Command); hasLocation {
			bodyStruct.Location = &location
		}
		if tags, hasTags := readTags(r.Command); hasTags {
			if len(tags) > 0 {
				bodyStruct.Tags = &tags
			}
		}

		jsonEncodedRequest, err := json.Marshal(bodyStruct)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(jsonEncodedRequest)
	}

	// Create an HTTP request
	req, err := retryablehttp.NewRequest(apiInfo.Method, urlString, body)
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	query.Set("api-version", apiInfo.ApiVersion)
	req.URL.RawQuery = query.Encode()

	if apiInfo.HasBody() {
		req.Header.Add("Content-Type", "application/json")
	}

	err = r.client.tokenRequester.addAuthorizationToRequest(req)
	if err != nil {
		return nil, err
	}

	httpResponse, err := r.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// This is safe to use for every request: we check for it being http.StatusAccepted
	httpResponse, err = r.pollForAsynchronousResponse(httpResponse)
	if err != nil {
		return nil, err
	}

	var responseObj interface{}
	var errorObj *Error

	if isSuccessCode(httpResponse.StatusCode) {
		responseObj = apiInfo.ResponseTypeFunc()
		// The response factory func returns nil as a signal that there is no body
		if responseObj != nil {
			responseMap, err := unmarshalFlattenPropertiesAndClose(httpResponse)
			if err != nil {
				return nil, err
			}

			err = mapstructure.WeakDecode(responseMap, responseObj)
			if err != nil {
				return nil, err
			}
		}
	} else {
		responseMap, err := unmarshalFlattenErrorAndClose(httpResponse)

		err = mapstructure.WeakDecode(responseMap, &errorObj)
		if err != nil {
			return nil, err
		}

		errorObj.StatusCode = httpResponse.StatusCode
	}

	return &Response{
		HTTP:   httpResponse,
		Parsed: responseObj,
		Error:  errorObj,
	}, nil
}
