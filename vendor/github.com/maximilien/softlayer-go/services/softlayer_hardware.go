package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	common "github.com/maximilien/softlayer-go/common"
	datatypes "github.com/maximilien/softlayer-go/data_types"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
)

type softLayer_Hardware_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Hardware_Service(client softlayer.Client) *softLayer_Hardware_Service {
	return &softLayer_Hardware_Service{
		client: client,
	}
}

func (slhs *softLayer_Hardware_Service) GetName() string {
	return "SoftLayer_Hardware"
}

func (slhs *softLayer_Hardware_Service) CreateObject(template datatypes.SoftLayer_Hardware_Template) (datatypes.SoftLayer_Hardware, error) {
	parameters := datatypes.SoftLayer_Hardware_Template_Parameters{
		Parameters: []datatypes.SoftLayer_Hardware_Template{
			template,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s.json", slhs.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#createObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Hardware{}, errors.New(errorMessage)
	}

	err = slhs.client.GetHttpClient().CheckForHttpResponseErrors(response)
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	bare_metal_server := datatypes.SoftLayer_Hardware{}
	err = json.Unmarshal(response, &bare_metal_server)
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	return bare_metal_server, nil
}

func (slhs *softLayer_Hardware_Service) GetObject(id string) (datatypes.SoftLayer_Hardware, error) {

	objectMask := []string{
		"bareMetalInstanceFlag",
		"domain",
		"hostname",
		"id",
		"hardwareStatusId",
		"provisionDate",
		"globalIdentifier",
		"primaryIpAddress",
		"operatingSystem.passwords.password",
		"operatingSystem.passwords.username",
	}

	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%s.json", slhs.GetName(), id), objectMask, "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#getObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Hardware{}, errors.New(errorMessage)
	}

	err = slhs.client.GetHttpClient().CheckForHttpResponseErrors(response)
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	bare_metal_server := datatypes.SoftLayer_Hardware{}
	err = json.Unmarshal(response, &bare_metal_server)
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	return bare_metal_server, nil
}
