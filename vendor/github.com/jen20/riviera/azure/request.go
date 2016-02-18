package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	URI      *string
	location *string
	tags     *map[string]*string
	etag     *string
	Command  APICall

	client *Client
}

func readTaggedFields(command interface{}) map[string]interface{} {
	var value reflect.Value
	if reflect.ValueOf(command).Kind() == reflect.Ptr {
		value = reflect.ValueOf(command).Elem()
	} else {
		value = reflect.ValueOf(command)
	}

	result := make(map[string]interface{})

	for i := 0; i < value.NumField(); i++ { // iterates through every struct type field
		tag := value.Type().Field(i).Tag // returns the tag string
		tagValue := tag.Get("riviera")
		if tagValue != "" {
			result[tagValue] = value.Field(i).Interface()
		}
	}
	return result
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

func defaultARMRequestStruct(request *Request, properties interface{}) interface{} {
	body := make(map[string]interface{})

	envelopeFields := readTaggedFields(properties)
	for k, v := range envelopeFields {
		body[k] = v
	}

	body["properties"] = properties
	return body
}

func defaultARMRequestSerialize(body interface{}) (io.ReadSeeker, error) {
	jsonEncodedRequest, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(jsonEncodedRequest), nil
}

func (request *Request) Execute() (*Response, error) {
	apiInfo := request.Command.APIInfo()

	var urlString string

	// Base URL should already be validated by now so Parse is safe without error handling
	urlObj, _ := url.Parse(request.client.BaseURL)

	// Determine whether to use the URLPathFunc or the URI explicitly set in the request
	if request.URI == nil {
		urlObj.Path = fmt.Sprintf("/subscriptions/%s/%s", request.client.subscriptionID, strings.TrimPrefix(apiInfo.URLPathFunc(), "/"))
		urlString = urlObj.String()
	} else {
		urlObj.Path = *request.URI
		urlString = urlObj.String()
	}

	// Encode the request body if necessary
	var body io.ReadSeeker
	if apiInfo.HasBody() {
		var bodyStruct interface{}
		if apiInfo.RequestPropertiesFunc != nil {
			bodyStruct = defaultARMRequestStruct(request, apiInfo.RequestPropertiesFunc())
		} else {
			bodyStruct = defaultARMRequestStruct(request, request.Command)
		}

		serialized, err := defaultARMRequestSerialize(bodyStruct)
		if err != nil {
			return nil, err
		}

		body = serialized
	} else {

		body = bytes.NewReader([]byte{})
	}

	// Create an HTTP request
	req, err := retryablehttp.NewRequest(apiInfo.Method, urlString, body)
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	query.Set("api-version", apiInfo.APIVersion)
	req.URL.RawQuery = query.Encode()

	if apiInfo.HasBody() {
		req.Header.Add("Content-Type", "application/json")
	}

	err = request.client.tokenRequester.addAuthorizationToRequest(req)
	if err != nil {
		return nil, err
	}

	httpResponse, err := request.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	// This is safe to use for every request: we check for it being http.StatusAccepted
	httpResponse, err = request.pollForAsynchronousResponse(httpResponse)
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
