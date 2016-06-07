package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TheWeatherCompany/softlayer-go/common"
	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
)

type softLayer_Security_Certificate_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Security_Certificate_Service(client softlayer.Client) *softLayer_Security_Certificate_Service {
	return &softLayer_Security_Certificate_Service{
		client: client,
	}
}

func (slscs *softLayer_Security_Certificate_Service) GetName() string {
	return "SoftLayer_Security_Certificate"
}

func (slscs *softLayer_Security_Certificate_Service) CreateSecurityCertificate(template datatypes.SoftLayer_Security_Certificate_Template) (datatypes.SoftLayer_Security_Certificate, error) {
	parameters := datatypes.SoftLayer_Security_Certificate_Parameters{
		Parameters: []datatypes.SoftLayer_Security_Certificate_Template{{
			Certificate:             template.Certificate,
			IntermediateCertificate: template.IntermediateCertificate,
			PrivateKey:              template.PrivateKey,
		}},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Security_Certificate{}, fmt.Errorf("Unable to create JSON: %s", err)
	}

	response, errorCode, err := slscs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/createObject.json", slscs.GetName()), "POST", bytes.NewBuffer(requestBody))

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not create SoftLayer_Security_Certificate, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Security_Certificate{}, errors.New(errorMessage)
	}

	if err != nil {
		return datatypes.SoftLayer_Security_Certificate{}, err
	}

	securityCertificate := datatypes.SoftLayer_Security_Certificate{}
	err = json.Unmarshal(response, &securityCertificate)
	if err != nil {
		return datatypes.SoftLayer_Security_Certificate{}, err
	}

	return securityCertificate, nil
}

func (slscs *softLayer_Security_Certificate_Service) GetObject(id int) (datatypes.SoftLayer_Security_Certificate, error) {
	response, errorCode, err := slscs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getObject.json", slscs.GetName(), id), "GET", new(bytes.Buffer))

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not retrieve SoftLayer_Security_Certificate, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Security_Certificate{}, errors.New(errorMessage)
	}

	if err != nil {
		return datatypes.SoftLayer_Security_Certificate{}, err
	}

	securityCertificate := datatypes.SoftLayer_Security_Certificate{}
	err = json.Unmarshal(response, &securityCertificate)
	if err != nil {
		return datatypes.SoftLayer_Security_Certificate{}, err
	}

	return securityCertificate, nil
}

func (slscs *softLayer_Security_Certificate_Service) DeleteObject(id int) (bool, error) {
	response, errorCode, err := slscs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d.json", slscs.GetName(), id), "DELETE", new(bytes.Buffer))

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to delete Security Certificate with id '%d', got '%s' as response from the API.", id, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not delete SoftLayer_Security_Certificate, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if err != nil {
		return false, err
	}

	return true, nil
}
