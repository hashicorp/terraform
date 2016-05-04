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

type softLayer_Dns_Domain_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Dns_Domain_Service(client softlayer.Client) *softLayer_Dns_Domain_Service {
	return &softLayer_Dns_Domain_Service{
		client: client,
	}
}

func (sldds *softLayer_Dns_Domain_Service) GetName() string {
	return "SoftLayer_Dns_Domain"
}

func (sldds *softLayer_Dns_Domain_Service) CreateObject(template datatypes.SoftLayer_Dns_Domain_Template) (datatypes.SoftLayer_Dns_Domain, error) {
	if template.ResourceRecords == nil {
		template.ResourceRecords = []datatypes.SoftLayer_Dns_Domain_ResourceRecord{}
	}

	parameters := datatypes.SoftLayer_Dns_Domain_Template_Parameters{
		Parameters: []datatypes.SoftLayer_Dns_Domain_Template{
			template,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain{}, err
	}

	response, errorCode, err := sldds.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s.json", sldds.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Dns_Domain#createObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Dns_Domain{}, errors.New(errorMessage)
	}

	err = sldds.client.GetHttpClient().CheckForHttpResponseErrors(response)
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain{}, err
	}

	softLayer_Dns_Domain := datatypes.SoftLayer_Dns_Domain{}
	err = json.Unmarshal(response, &softLayer_Dns_Domain)
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain{}, err
	}

	return softLayer_Dns_Domain, nil
}

func (sldds *softLayer_Dns_Domain_Service) GetObject(dnsId int) (datatypes.SoftLayer_Dns_Domain, error) {
	objectMask := []string{
		"id",
		"name",
		"serial",
		"updateDate",
		"account",
		"managedResourceFlag",
		"resourceRecordCount",
		"resourceRecords",
		"secondary",
	}

	response, errorCode, err := sldds.client.GetHttpClient().DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getObject.json", sldds.GetName(), dnsId), objectMask, "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Dns_Domain#getObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Dns_Domain{}, errors.New(errorMessage)
	}

	dns_domain := datatypes.SoftLayer_Dns_Domain{}
	err = json.Unmarshal(response, &dns_domain)
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain{}, err
	}

	return dns_domain, nil
}

func (sldds *softLayer_Dns_Domain_Service) DeleteObject(dnsId int) (bool, error) {
	response, errorCode, err := sldds.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d.json", sldds.GetName(), dnsId), "DELETE", new(bytes.Buffer))

	if response_value := string(response[:]); response_value != "true" {
		return false, errors.New(fmt.Sprintf("Failed to delete dns domain with id '%d', got '%s' as response from the API", dnsId, response_value))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Dns_Domain#deleteObject, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}
