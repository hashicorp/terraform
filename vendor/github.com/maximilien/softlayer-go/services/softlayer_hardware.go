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

func (slhs *softLayer_Hardware_Service) AllowAccessToNetworkStorage(id int, storage datatypes.SoftLayer_Network_Storage) (bool, error) {
	parameters := datatypes.SoftLayer_Hardware_NetworkStorage_Parameters{
		Parameters: storage,
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/allowAccessToNetworkStorage.json", slhs.GetName(), id), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#allowAccessToNetworkStorage, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to allowAccessToNetworkStorage with id '%d', got '%s' as response from the API.", id, res))
	}

	return true, nil
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

	hardware := datatypes.SoftLayer_Hardware{}
	err = json.Unmarshal(response, &hardware)
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	return hardware, nil
}

func (slhs *softLayer_Hardware_Service) FindByIpAddress(ipAddress string) (datatypes.SoftLayer_Hardware, error) {

	ipAddressParameters := datatypes.SoftLayer_Hardware_String_Parameters{
		Parameters: []string{ipAddress},
	}

	requestBody, err := json.Marshal(ipAddressParameters)
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

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
		"datacenter.name",
		"datacenter.longName",
		"datacenter.id",
	}

	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/findByIpAddress.json", slhs.GetName()), objectMask, "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#findByIpAddress, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Hardware{}, errors.New(errorMessage)
	}

	hardware := datatypes.SoftLayer_Hardware{}
	err = json.Unmarshal(response, &hardware)
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	return hardware, nil
}

func (slhs *softLayer_Hardware_Service) GetObject(id int) (datatypes.SoftLayer_Hardware, error) {

	objectMask := []string{
		"bareMetalInstanceFlag",
		"domain",
		"hostname",
		"id",
		"hardwareStatusId",
		"provisionDate",
		"globalIdentifier",
		"primaryIpAddress",
		"primaryBackendIpAddress",
		"operatingSystem.passwords.password",
		"operatingSystem.passwords.username",
		"datacenter.name",
		"datacenter.longName",
		"datacenter.id",
	}

	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getObject.json", slhs.GetName(), id), objectMask, "GET", new(bytes.Buffer))
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

	hardware := datatypes.SoftLayer_Hardware{}
	err = json.Unmarshal(response, &hardware)
	if err != nil {
		return datatypes.SoftLayer_Hardware{}, err
	}

	return hardware, nil
}

func (slhs *softLayer_Hardware_Service) GetAttachedNetworkStorages(id int, nasType string) ([]datatypes.SoftLayer_Network_Storage, error) {

	nasTypeParameters := datatypes.SoftLayer_Hardware_String_Parameters{
		Parameters: []string{nasType},
	}

	requestBody, err := json.Marshal(nasTypeParameters)
	if err != nil {
		return []datatypes.SoftLayer_Network_Storage{}, err
	}

	objectMask := []string{
		"accountId",
		"capacityGb",
		"createDate",
		"guestId",
		"hardwareId",
		"hostId",
		"id",
		"fullyQualifiedDomainName",
		"nasType",
		"notes",
		"Password",
		"serviceProviderId",
		"upgradableFlag",
		"username",
		"billingItem.id",
		"billingItem.orderItem.order.id",
		"lunId",
		"serviceResourceBackendIpAddress",
	}

	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getAttachedNetworkStorages.json", slhs.GetName(), id), objectMask, "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return []datatypes.SoftLayer_Network_Storage{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#getAttachedNetworkStorages, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Network_Storage{}, errors.New(errorMessage)
	}

	err = slhs.client.GetHttpClient().CheckForHttpResponseErrors(response)
	if err != nil {
		return []datatypes.SoftLayer_Network_Storage{}, err
	}

	storageList := []datatypes.SoftLayer_Network_Storage{}
	err = json.Unmarshal(response, &storageList)
	if err != nil {
		return []datatypes.SoftLayer_Network_Storage{}, err
	}

	return storageList, nil
}

func (slhs *softLayer_Hardware_Service) GetAllowedHost(id int) (datatypes.SoftLayer_Network_Storage_Allowed_Host, error) {
	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getAllowedHost.json", slhs.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Network_Storage_Allowed_Host{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#getAllowedHost, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Network_Storage_Allowed_Host{}, errors.New(errorMessage)
	}

	allowedHost := datatypes.SoftLayer_Network_Storage_Allowed_Host{}
	err = json.Unmarshal(response, &allowedHost)
	if err != nil {
		return datatypes.SoftLayer_Network_Storage_Allowed_Host{}, err
	}

	return allowedHost, nil
}

func (slhs *softLayer_Hardware_Service) GetDatacenter(id int) (datatypes.SoftLayer_Location, error) {
	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getDatacenter.json", slhs.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Location{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#getDatacenter, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Location{}, errors.New(errorMessage)
	}

	datacenter := datatypes.SoftLayer_Location{}
	err = json.Unmarshal(response, &datacenter)
	if err != nil {
		return datatypes.SoftLayer_Location{}, err
	}

	return datacenter, nil
}

func (slhs *softLayer_Hardware_Service) GetPrimaryIpAddress(id int) (string, error) {
	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getPrimaryIpAddress.json", slhs.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return "", err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#getPrimaryIpAddress, HTTP error code: '%d'", errorCode)
		return "", errors.New(errorMessage)
	}

	return string(response[:]), nil
}

func (slhs *softLayer_Hardware_Service) GetPrimaryBackendIpAddress(id int) (string, error) {
	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getPrimaryBackendIpAddress.json", slhs.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return "", err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#getPrimaryBackendIpAddress, HTTP error code: '%d'", errorCode)
		return "", errors.New(errorMessage)
	}

	return string(response[:]), nil
}

func (slhs *softLayer_Hardware_Service) PowerOff(instanceId int) (bool, error) {
	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/powerOff.json", slhs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#powerOff, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to power off hardware with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	return true, nil
}

func (slhs *softLayer_Hardware_Service) PowerOffSoft(instanceId int) (bool, error) {
	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/powerOffSoft.json", slhs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to power off soft hardware with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#powerOffSoft, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, nil
}

func (slhs *softLayer_Hardware_Service) PowerOn(instanceId int) (bool, error) {
	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/powerOn.json", slhs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to power on hardware with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#powerOn, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, nil
}

func (slhs *softLayer_Hardware_Service) RebootDefault(instanceId int) (bool, error) {
	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/rebootDefault.json", slhs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to default reboot hardware with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#rebootDefault, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, nil
}

func (slhs *softLayer_Hardware_Service) RebootSoft(instanceId int) (bool, error) {
	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/rebootSoft.json", slhs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to soft reboot hardware with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#rebootSoft, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, nil
}

func (slhs *softLayer_Hardware_Service) RebootHard(instanceId int) (bool, error) {
	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/rebootHard.json", slhs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to hard reboot hardware with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#rebootHard, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, nil
}

func (slhs *softLayer_Hardware_Service) SetTags(instanceId int, tags []string) (bool, error) {
	var tagStringBuffer bytes.Buffer
	for i, tag := range tags {
		tagStringBuffer.WriteString(tag)
		if i != len(tags)-1 {
			tagStringBuffer.WriteString(", ")
		}
	}

	setTagsParameters := datatypes.SoftLayer_Hardware_String_Parameters{
		Parameters: []string{tagStringBuffer.String()},
	}

	requestBody, err := json.Marshal(setTagsParameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slhs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/setTags.json", slhs.GetName(), instanceId), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Hardware#setTags, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to setTags for hardware with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	return true, nil
}
