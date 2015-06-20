package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	datatypes "github.com/maximilien/softlayer-go/data_types"
	"github.com/maximilien/softlayer-go/softlayer"
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

	response, err := sldds.client.DoRawHttpRequest(fmt.Sprintf("%s.json", sldds.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain{}, err
	}

	err = sldds.client.CheckForHttpResponseErrors(response)
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

	response, err := sldds.client.DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getObject.json", sldds.GetName(), dnsId), objectMask, "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain{}, err
	}

	dns_domain := datatypes.SoftLayer_Dns_Domain{}
	err = json.Unmarshal(response, &dns_domain)
	if err != nil {
		return datatypes.SoftLayer_Dns_Domain{}, err
	}

	return dns_domain, nil
}

func (sldds *softLayer_Dns_Domain_Service) DeleteObject(dnsId int) (bool, error) {
	response, err := sldds.client.DoRawHttpRequest(fmt.Sprintf("%s/%d.json", sldds.GetName(), dnsId), "DELETE", new(bytes.Buffer))

	if response_value := string(response[:]); response_value != "true" {
		return false, errors.New(fmt.Sprintf("Failed to delete dns domain with id '%d', got '%s' as response from the API", dnsId, response_value))
	}

	return true, err
}
