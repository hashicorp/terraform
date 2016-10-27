package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	common "github.com/maximilien/softlayer-go/common"
	datatypes "github.com/maximilien/softlayer-go/data_types"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
)

type softLayer_Virtual_Guest_Block_Device_Template_Group_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Virtual_Guest_Block_Device_Template_Group_Service(client softlayer.Client) *softLayer_Virtual_Guest_Block_Device_Template_Group_Service {
	return &softLayer_Virtual_Guest_Block_Device_Template_Group_Service{
		client: client,
	}
}

func (slvgs *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) GetName() string {
	return "SoftLayer_Virtual_Guest_Block_Device_Template_Group"
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) GetObject(id int) (datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group, error) {
	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getObject.json", slvgbdtg.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#getObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group{}, errors.New(errorMessage)
	}

	vgbdtGroup := datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group{}
	err = json.Unmarshal(response, &vgbdtGroup)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group{}, err
	}

	return vgbdtGroup, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) DeleteObject(id int) (datatypes.SoftLayer_Provisioning_Version1_Transaction, error) {
	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d.json", slvgbdtg.GetName(), id), "DELETE", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#deleteObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, errors.New(errorMessage)
	}

	transaction := datatypes.SoftLayer_Provisioning_Version1_Transaction{}
	err = json.Unmarshal(response, &transaction)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	return transaction, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) GetDatacenters(id int) ([]datatypes.SoftLayer_Location, error) {
	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getDatacenters.json", slvgbdtg.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Location{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#getDatacenters, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Location{}, errors.New(errorMessage)
	}

	locations := []datatypes.SoftLayer_Location{}
	err = json.Unmarshal(response, &locations)
	if err != nil {
		return []datatypes.SoftLayer_Location{}, err
	}

	return locations, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) GetSshKeys(id int) ([]datatypes.SoftLayer_Security_Ssh_Key, error) {
	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getSshKeys.json", slvgbdtg.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#getSshKeys, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Security_Ssh_Key{}, errors.New(errorMessage)
	}

	sshKeys := []datatypes.SoftLayer_Security_Ssh_Key{}
	err = json.Unmarshal(response, &sshKeys)
	if err != nil {
		return []datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	return sshKeys, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) GetStatus(id int) (datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_Status, error) {
	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getStatus.json", slvgbdtg.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_Status{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#getStatus, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_Status{}, errors.New(errorMessage)
	}

	status := datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_Status{}
	err = json.Unmarshal(response, &status)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_Status{}, err
	}

	return status, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) GetImageType(id int) (datatypes.SoftLayer_Image_Type, error) {
	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getImageType.json", slvgbdtg.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Image_Type{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#getImageType, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Image_Type{}, errors.New(errorMessage)
	}

	imageType := datatypes.SoftLayer_Image_Type{}
	err = json.Unmarshal(response, &imageType)
	if err != nil {
		return datatypes.SoftLayer_Image_Type{}, err
	}

	return imageType, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) GetStorageLocations(id int) ([]datatypes.SoftLayer_Location, error) {
	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getStorageLocations.json", slvgbdtg.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Location{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#getStorageLocations, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Location{}, errors.New(errorMessage)
	}

	locations := []datatypes.SoftLayer_Location{}
	err = json.Unmarshal(response, &locations)
	if err != nil {
		return []datatypes.SoftLayer_Location{}, err
	}

	return locations, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) CreateFromExternalSource(configuration datatypes.SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration) (datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group, error) {
	parameters := datatypes.SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration_Parameters{
		Parameters: []datatypes.SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration{configuration},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group{}, err
	}

	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/createFromExternalSource.json", slvgbdtg.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#createFromExternalSource, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group{}, errors.New(errorMessage)
	}

	vgbdtGroup := datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group{}
	err = json.Unmarshal(response, &vgbdtGroup)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group{}, err
	}

	return vgbdtGroup, err
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) CopyToExternalSource(configuration datatypes.SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration) (bool, error) {
	parameters := datatypes.SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration_Parameters{
		Parameters: []datatypes.SoftLayer_Container_Virtual_Guest_Block_Device_Template_Configuration{configuration},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/copyToExternalSource.json", slvgbdtg.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#copyToExternalSource, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to create virtual guest block device template group, got '%s' as response from the API.", res))
	}

	return true, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) GetImageTypeKeyName(id int) (string, error) {
	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getImageTypeKeyName.json", slvgbdtg.GetName(), id), "GET", new(bytes.Buffer))

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#getImageTypeKeyName, HTTP error code: '%d'", errorCode)
		return "", errors.New(errorMessage)
	}

	return string(response), err
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) GetTransaction(id int) (datatypes.SoftLayer_Provisioning_Version1_Transaction, error) {
	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getTransaction.json", slvgbdtg.GetName(), id), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#getTransaction, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, errors.New(errorMessage)
	}

	transaction := datatypes.SoftLayer_Provisioning_Version1_Transaction{}
	err = json.Unmarshal(response, &transaction)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	return transaction, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) DenySharingAccess(id int, accountId int) (bool, error) {
	parameters := datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_GroupInitParameters{
		Parameters: datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_GroupInitParameter{
			AccountId: accountId,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/denySharingAccess.json", slvgbdtg.GetName(), id), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#denySharingAccess, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to permit sharing access to VGDBTG with ID: %d to account ID: %d", id, accountId))
	}

	return true, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) PermitSharingAccess(id int, accountId int) (bool, error) {
	parameters := datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_GroupInitParameters{
		Parameters: datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_GroupInitParameter{
			AccountId: accountId,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/permitSharingAccess.json", slvgbdtg.GetName(), id), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#permitSharingAccess, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to permit sharing access to VGDBTG with ID: %d to account ID: %d", id, accountId))
	}

	return true, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) AddLocations(id int, locations []datatypes.SoftLayer_Location) (bool, error) {
	parameters := datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_LocationsInitParameters{
		Parameters: datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_LocationsInitParameter{
			Locations: locations,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/addLocations.json", slvgbdtg.GetName(), id), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#addLocations, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to add locations access to VGDBTG with ID: %d", id))
	}

	return true, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) RemoveLocations(id int, locations []datatypes.SoftLayer_Location) (bool, error) {
	parameters := datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_LocationsInitParameters{
		Parameters: datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_LocationsInitParameter{
			Locations: locations,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/removeLocations.json", slvgbdtg.GetName(), id), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#removeLocations, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to remove locations access to VGDBTG with ID: %d", id))
	}

	return true, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) SetAvailableLocations(id int, locations []datatypes.SoftLayer_Location) (bool, error) {
	parameters := datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_LocationsInitParameters{
		Parameters: datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_Group_LocationsInitParameter{
			Locations: locations,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/setAvailableLocations.json", slvgbdtg.GetName(), id), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#setAvailableLocations, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to set available locations access to VGDBTG with ID: %d", id))
	}

	return true, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) CreatePublicArchiveTransaction(id int, groupName string, summary string, note string, locations []datatypes.SoftLayer_Location) (int, error) {
	groupName = url.QueryEscape(groupName)
	summary = url.QueryEscape(summary)
	note = url.QueryEscape(note)

	parameters := datatypes.SoftLayer_Virtual_Guest_Block_Device_Template_GroupInitParameters2{
		Parameters: []interface{}{groupName, summary, note, locations},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return 0, err
	}

	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/createPublicArchiveTransaction.json", slvgbdtg.GetName(), id), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return 0, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#createPublicArchiveTransaction, HTTP error code: '%d'", errorCode)
		return 0, errors.New(errorMessage)
	}

	transactionId, err := strconv.Atoi(string(response[:]))
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Failed to createPublicArchiveTransaction for ID: %d, error: %s", id, string(response[:])))
	}

	return transactionId, nil
}

func (slvgbdtg *softLayer_Virtual_Guest_Block_Device_Template_Group_Service) GetGlobalIdentifier(id int) (string, error) {
	response, errorCode, err := slvgbdtg.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getGlobalIdentifier.json", slvgbdtg.GetName(), id), "GET", new(bytes.Buffer))

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest_Block_Device_Template_Group#getGlobalIdentifier, HTTP error code: '%d'", errorCode)
		return "", errors.New(errorMessage)
	}

	return string(strings.TrimSpace(string(response))), err
}
