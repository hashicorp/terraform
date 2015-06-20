package services

import (
	"bytes"
	"encoding/json"
	"fmt"

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

	response, err := slhs.client.DoRawHttpRequest(fmt.Sprintf("%s.json", slhs.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	err = slhs.client.CheckForHttpResponseErrors(response)
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

	response, err := slhs.client.DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%s.json", slhs.GetName(), id), objectMask, "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	err = slhs.client.CheckForHttpResponseErrors(response)
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
