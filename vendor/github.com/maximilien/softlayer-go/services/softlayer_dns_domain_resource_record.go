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

type SoftLayer_Dns_Domain_ResourceRecord_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Dns_Domain_ResourceRecord_Service(client softlayer.Client) *SoftLayer_Dns_Domain_ResourceRecord_Service {
	return &SoftLayer_Dns_Domain_ResourceRecord_Service{
		client: client,
	}
}

func (sldr *SoftLayer_Dns_Domain_ResourceRecord_Service) GetName() string {
	return "SoftLayer_Dns_Domain_ResourceRecord"
}

func (sldr *SoftLayer_Dns_Domain_ResourceRecord_Service) CreateObject(template datatypes.SoftLayer_Dns_Domain_ResourceRecord_Template) (datatypes.SoftLayer_Dns_Domain_ResourceRecord, error) {
	parameters := datatypes.SoftLayer_Dns_Domain_ResourceRecord_Template_Parameters{
		Parameters: []datatypes.SoftLayer_Dns_Domain_ResourceRecord_Template{
			template,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain_ResourceRecord{}, err
	}

	response, errorCode, err := sldr.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/createObject", sldr.getNameByType(template.Type)), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain_ResourceRecord{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Dns_Domain_ResourceRecord#createObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Dns_Domain_ResourceRecord{}, errors.New(errorMessage)
	}

	err = sldr.client.GetHttpClient().CheckForHttpResponseErrors(response)
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain_ResourceRecord{}, err
	}

	dns_record := datatypes.SoftLayer_Dns_Domain_ResourceRecord{}
	err = json.Unmarshal(response, &dns_record)
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain_ResourceRecord{}, err
	}

	return dns_record, nil
}

func (sldr *SoftLayer_Dns_Domain_ResourceRecord_Service) GetObject(id int) (datatypes.SoftLayer_Dns_Domain_ResourceRecord, error) {
	objectMask := []string{
		"data",
		"domainId",
		"expire",
		"host",
		"id",
		"minimum",
		"mxPriority",
		"refresh",
		"responsiblePerson",
		"retry",
		"ttl",
		"type",
		"service",
		"priority",
		"protocol",
		"port",
		"weight",
	}

	response, errorCode, err := sldr.client.GetHttpClient().DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getObject.json", sldr.GetName(), id), objectMask, "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain_ResourceRecord{}, err
	}

	err = sldr.client.GetHttpClient().CheckForHttpResponseErrors(response)
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain_ResourceRecord{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Dns_Domain_ResourceRecord#getObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Dns_Domain_ResourceRecord{}, errors.New(errorMessage)
	}

	dns_record := datatypes.SoftLayer_Dns_Domain_ResourceRecord{}
	err = json.Unmarshal(response, &dns_record)
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain_ResourceRecord{}, err
	}

	return dns_record, nil
}

func (sldr *SoftLayer_Dns_Domain_ResourceRecord_Service) DeleteObject(recordId int) (bool, error) {
	response, errorCode, err := sldr.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d.json", sldr.GetName(), recordId), "DELETE", new(bytes.Buffer))

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to delete DNS Domain Record with id '%d', got '%s' as response from the API.", recordId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Dns_Domain_ResourceRecord#deleteObject, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (sldr *SoftLayer_Dns_Domain_ResourceRecord_Service) EditObject(recordId int, template datatypes.SoftLayer_Dns_Domain_ResourceRecord) (bool, error) {
	parameters := datatypes.SoftLayer_Dns_Domain_ResourceRecord_Parameters{
		Parameters: []datatypes.SoftLayer_Dns_Domain_ResourceRecord{
			template,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := sldr.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/editObject.json", sldr.getNameByType(template.Type), recordId), "POST", bytes.NewBuffer(requestBody))

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to edit DNS Domain Record with id: %d, got '%s' as response from the API.", recordId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Dns_Domain_ResourceRecord#editObject, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

//Private methods

func (sldr *SoftLayer_Dns_Domain_ResourceRecord_Service) getNameByType(dnsType string) string {
	switch dnsType {
	case "srv":
		// Currently only SRV record type requires additional fields for Create and Update, while all other record types
		// use basic default resource type. Therefore there is no need for now to implement each resource type as separate service
		// https://sldn.softlayer.com/reference/datatypes/SoftLayer_Dns_Domain_ResourceRecord_SrvType
		return "SoftLayer_Dns_Domain_ResourceRecord_SrvType"
	default:
		return "SoftLayer_Dns_Domain_ResourceRecord"
	}
}
