package client_fakes

import (
	"bytes"
)

type FakeHttpClient struct {
	Username string
	Password string

	DoRawHttpRequestInt            int
	DoRawHttpRequestError          error
	DoRawHttpRequestResponse       []byte
	DoRawHttpRequestResponses      [][]byte
	DoRawHttpRequestResponsesCount int
	DoRawHttpRequestResponsesIndex int

	//DoRawHttpRequest
	DoRawHttpRequestPath        string
	DoRawHttpRequestRequestType string
	DoRawHttpRequestRequestBody *bytes.Buffer

	//DoRawHttpRequestWithObject
	DoRawHttpRequestWithObjectMaskPath        string
	DoRawHttpRequestWithObjectMaskMasks       []string
	DoRawHttpRequestWithObjectMaskRequestType string
	DoRawHttpRequestWithObjectMaskRequestBody *bytes.Buffer

	//DoRawHttpRequestWithObjectFilter
	DoRawHttpRequestWithObjectFilterPath        string
	DoRawHttpRequestWithObjectFilterFilters     string
	DoRawHttpRequestWithObjectFilterRequestType string
	DoRawHttpRequestWithObjectFilterRequestBody *bytes.Buffer

	//DoRawHttpRequestWithObjectFilterAndObjectMask
	DoRawHttpRequestWithObjectFilterAndObjectMaskPath        string
	DoRawHttpRequestWithObjectFilterAndObjectMaskMasks       []string
	DoRawHttpRequestWithObjectFilterAndObjectMaskFilters     string
	DoRawHttpRequestWithObjectFilterAndObjectMaskRequestType string
	DoRawHttpRequestWithObjectFilterAndObjectMaskRequestBody *bytes.Buffer

	//GenerateRequest
	GenerateRequestBodyTemplateData interface{}
	GenerateRequestBodyBuffer       *bytes.Buffer
	GenerateRequestBodyError        error

	//HasErrors
	HasErrorsBody  map[string]interface{}
	HasErrorsError error

	//CheclForHttpResponseErrors
	CheckForHttpResponseErrorsData  []byte
	CheckForHttpResponseErrorsError error
}

func NewFakeHttpClient(username, Password string) *FakeHttpClient {
	return &FakeHttpClient{
		Username: username,
		Password: Password,

		DoRawHttpRequestInt:            200,
		DoRawHttpRequestError:          nil,
		DoRawHttpRequestResponses:      [][]byte{},
		DoRawHttpRequestResponsesCount: 0,
		DoRawHttpRequestResponsesIndex: 0,
	}
}

//softlayer.HttpClient interface methods

func (fhc *FakeHttpClient) DoRawHttpRequest(path string, requestType string, requestBody *bytes.Buffer) ([]byte, int, error) {
	fhc.DoRawHttpRequestPath = path
	fhc.DoRawHttpRequestRequestType = requestType
	fhc.DoRawHttpRequestRequestBody = requestBody

	return fhc.processResponse()
}

func (fhc *FakeHttpClient) DoRawHttpRequestWithObjectMask(path string, masks []string, requestType string, requestBody *bytes.Buffer) ([]byte, int, error) {
	fhc.DoRawHttpRequestWithObjectMaskPath = path
	fhc.DoRawHttpRequestWithObjectMaskMasks = masks
	fhc.DoRawHttpRequestWithObjectMaskRequestType = requestType
	fhc.DoRawHttpRequestWithObjectMaskRequestBody = requestBody

	return fhc.processResponse()
}

func (fhc *FakeHttpClient) DoRawHttpRequestWithObjectFilter(path string, filters string, requestType string, requestBody *bytes.Buffer) ([]byte, int, error) {
	fhc.DoRawHttpRequestWithObjectFilterPath = path
	fhc.DoRawHttpRequestWithObjectFilterFilters = filters
	fhc.DoRawHttpRequestWithObjectFilterRequestType = requestType
	fhc.DoRawHttpRequestWithObjectFilterRequestBody = requestBody

	return fhc.processResponse()
}

func (fhc *FakeHttpClient) DoRawHttpRequestWithObjectFilterAndObjectMask(path string, masks []string, filters string, requestType string, requestBody *bytes.Buffer) ([]byte, int, error) {
	fhc.DoRawHttpRequestWithObjectFilterAndObjectMaskPath = path
	fhc.DoRawHttpRequestWithObjectFilterAndObjectMaskMasks = masks
	fhc.DoRawHttpRequestWithObjectFilterAndObjectMaskFilters = filters
	fhc.DoRawHttpRequestWithObjectFilterAndObjectMaskRequestType = requestType
	fhc.DoRawHttpRequestWithObjectFilterAndObjectMaskRequestBody = requestBody

	return fhc.processResponse()
}

func (fhc *FakeHttpClient) GenerateRequestBody(templateData interface{}) (*bytes.Buffer, error) {
	fhc.GenerateRequestBodyTemplateData = templateData

	return fhc.GenerateRequestBodyBuffer, fhc.GenerateRequestBodyError
}

func (fhc *FakeHttpClient) HasErrors(body map[string]interface{}) error {
	fhc.HasErrorsBody = body

	return fhc.HasErrorsError
}

func (fhc *FakeHttpClient) CheckForHttpResponseErrors(data []byte) error {
	fhc.CheckForHttpResponseErrorsData = data

	return fhc.CheckForHttpResponseErrorsError
}

// private methods

func (fhc *FakeHttpClient) processResponse() ([]byte, int, error) {
	fhc.DoRawHttpRequestResponsesCount += 1

	if fhc.DoRawHttpRequestError != nil {
		return []byte{}, fhc.DoRawHttpRequestInt, fhc.DoRawHttpRequestError
	}

	if fhc.DoRawHttpRequestResponses != nil && len(fhc.DoRawHttpRequestResponses) == 0 {
		return fhc.DoRawHttpRequestResponse, fhc.DoRawHttpRequestInt, fhc.DoRawHttpRequestError
	} else {
		fhc.DoRawHttpRequestResponsesIndex = fhc.DoRawHttpRequestResponsesIndex + 1
		return fhc.DoRawHttpRequestResponses[fhc.DoRawHttpRequestResponsesIndex-1], fhc.DoRawHttpRequestInt, fhc.DoRawHttpRequestError
	}
}
