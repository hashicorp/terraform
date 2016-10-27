package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	common "github.com/maximilien/softlayer-go/common"
	datatypes "github.com/maximilien/softlayer-go/data_types"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
)

const (
	EPHEMERAL_DISK_CATEGORY_CODE = "guest_disk1"
	// Package type for virtual servers: http://sldn.softlayer.com/reference/services/SoftLayer_Product_Order/placeOrder
	VIRTUAL_SERVER_PACKAGE_TYPE = "VIRTUAL_SERVER_INSTANCE"
	MAINTENANCE_WINDOW_PROPERTY = "MAINTENANCE_WINDOW"
	// Described in the following link: http://sldn.softlayer.com/reference/datatypes/SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade
	UPGRADE_VIRTUAL_SERVER_ORDER_TYPE = "SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade"
)

type softLayer_Virtual_Guest_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Virtual_Guest_Service(client softlayer.Client) *softLayer_Virtual_Guest_Service {
	return &softLayer_Virtual_Guest_Service{
		client: client,
	}
}

func (slvgs *softLayer_Virtual_Guest_Service) GetName() string {
	return "SoftLayer_Virtual_Guest"
}

func (slvgs *softLayer_Virtual_Guest_Service) CreateObject(template datatypes.SoftLayer_Virtual_Guest_Template) (datatypes.SoftLayer_Virtual_Guest, error) {
	err := slvgs.checkCreateObjectRequiredValues(template)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	parameters := datatypes.SoftLayer_Virtual_Guest_Template_Parameters{
		Parameters: []datatypes.SoftLayer_Virtual_Guest_Template{
			template,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s.json", slvgs.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#createObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Virtual_Guest{}, errors.New(errorMessage)
	}

	err = slvgs.client.GetHttpClient().CheckForHttpResponseErrors(response)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	softLayer_Virtual_Guest := datatypes.SoftLayer_Virtual_Guest{}
	err = json.Unmarshal(response, &softLayer_Virtual_Guest)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	return softLayer_Virtual_Guest, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) ReloadOperatingSystem(instanceId int, template datatypes.Image_Template_Config) error {
	parameter := [2]interface{}{"FORCE", template}
	parameters := map[string]interface{}{
		"parameters": parameter,
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return err
	}

	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/reloadOperatingSystem.json", slvgs.GetName(), instanceId), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#reloadOperatingSystem, HTTP error code: '%d'", errorCode)
		return errors.New(errorMessage)
	}

	if res := string(response[:]); res != `"1"` {
		return errors.New(fmt.Sprintf("Failed to reload OS on instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	return nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetObject(instanceId int) (datatypes.SoftLayer_Virtual_Guest, error) {

	objectMask := []string{
		"accountId",
		"createDate",
		"dedicatedAccountHostOnlyFlag",
		"domain",
		"fullyQualifiedDomainName",
		"hostname",
		"hourlyBillingFlag",
		"id",
		"lastPowerStateId",
		"lastVerifiedDate",
		"maxCpu",
		"maxCpuUnits",
		"maxMemory",
		"metricPollDate",
		"modifyDate",
		"notes",
		"postInstallScriptUri",
		"privateNetworkOnlyFlag",
		"startCpus",
		"statusId",
		"uuid",
		"userData.value",
		"localDiskFlag",

		"globalIdentifier",
		"managedResourceFlag",
		"primaryBackendIpAddress",
		"primaryIpAddress",

		"location.name",
		"location.longName",
		"location.id",
		"datacenter.name",
		"datacenter.longName",
		"datacenter.id",
		"networkComponents.maxSpeed",
		"operatingSystem.passwords.password",
		"operatingSystem.passwords.username",

		"blockDeviceTemplateGroup.globalIdentifier",
		"primaryNetworkComponent.networkVlan.id",
		"primaryBackendNetworkComponent.networkVlan.id",
	}

	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getObject.json", slvgs.GetName(), instanceId), objectMask, "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getObject, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Virtual_Guest{}, errors.New(errorMessage)
	}

	virtualGuest := datatypes.SoftLayer_Virtual_Guest{}
	err = json.Unmarshal(response, &virtualGuest)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	return virtualGuest, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetObjectByPrimaryIpAddress(ipAddress string) (datatypes.SoftLayer_Virtual_Guest, error) {

	ObjectFilter := string(`{"virtualGuests":{"primaryIpAddress":{"operation":"` + ipAddress + `"}}}`)

	accountService, err := slvgs.client.GetSoftLayer_Account_Service()
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	virtualGuests, err := accountService.GetVirtualGuestsByFilter(ObjectFilter)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	if len(virtualGuests) == 1 {
		return virtualGuests[0], nil
	}

	return datatypes.SoftLayer_Virtual_Guest{}, errors.New(fmt.Sprintf("Cannot find virtual guest with primary ip: %s", ipAddress))
}

func (slvgs *softLayer_Virtual_Guest_Service) GetObjectByPrimaryBackendIpAddress(ipAddress string) (datatypes.SoftLayer_Virtual_Guest, error) {

	ObjectFilter := string(`{"virtualGuests":{"primaryBackendIpAddress":{"operation":"` + ipAddress + `"}}}`)

	accountService, err := slvgs.client.GetSoftLayer_Account_Service()
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	virtualGuests, err := accountService.GetVirtualGuestsByFilter(ObjectFilter)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest{}, err
	}

	if len(virtualGuests) == 1 {
		return virtualGuests[0], nil
	}

	return datatypes.SoftLayer_Virtual_Guest{}, errors.New(fmt.Sprintf("Cannot find virtual guest with primary backend ip: %s", ipAddress))
}

func (slvgs *softLayer_Virtual_Guest_Service) EditObject(instanceId int, template datatypes.SoftLayer_Virtual_Guest) (bool, error) {
	parameters := datatypes.SoftLayer_Virtual_Guest_Parameters{
		Parameters: []datatypes.SoftLayer_Virtual_Guest{template},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/editObject.json", slvgs.GetName(), instanceId), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to edit virtual guest with id: %d, got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#editObject, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slvgs *softLayer_Virtual_Guest_Service) DeleteObject(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d.json", slvgs.GetName(), instanceId), "DELETE", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to delete instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#deleteObject, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slvgs *softLayer_Virtual_Guest_Service) GetPowerState(instanceId int) (datatypes.SoftLayer_Virtual_Guest_Power_State, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getPowerState.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Power_State{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getPowerState, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Virtual_Guest_Power_State{}, errors.New(errorMessage)
	}

	vgPowerState := datatypes.SoftLayer_Virtual_Guest_Power_State{}
	err = json.Unmarshal(response, &vgPowerState)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Power_State{}, err
	}

	return vgPowerState, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetPrimaryIpAddress(instanceId int) (string, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getPrimaryIpAddress.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return "", err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getPrimaryIpAddress, HTTP error code: '%d'", errorCode)
		return "", errors.New(errorMessage)
	}

	vgPrimaryIpAddress := strings.TrimSpace(string(response))
	if vgPrimaryIpAddress == "" {
		return "", errors.New(fmt.Sprintf("Failed to get primary IP address for instance with id '%d', got '%s' as response from the API.", instanceId, response))
	}

	return vgPrimaryIpAddress, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetPrimaryBackendIpAddress(instanceId int) (string, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getPrimaryBackendIpAddress.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return "", err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getPrimaryBackendIpAddress, HTTP error code: '%d'", errorCode)
		return "", errors.New(errorMessage)
	}

	vgPrimaryBackendIpAddress := strings.TrimSpace(string(response))
	if vgPrimaryBackendIpAddress == "" {
		return "", errors.New(fmt.Sprintf("Failed to get primary IP address for instance with id '%d', got '%s' as response from the API.", instanceId, response))
	}

	return vgPrimaryBackendIpAddress, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetActiveTransaction(instanceId int) (datatypes.SoftLayer_Provisioning_Version1_Transaction, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getActiveTransaction.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getActiveTransaction, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, errors.New(errorMessage)
	}

	activeTransaction := datatypes.SoftLayer_Provisioning_Version1_Transaction{}
	err = json.Unmarshal(response, &activeTransaction)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	return activeTransaction, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetLastTransaction(instanceId int) (datatypes.SoftLayer_Provisioning_Version1_Transaction, error) {
	objectMask := []string{
		"transactionGroup",
	}

	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getLastTransaction.json", slvgs.GetName(), instanceId), objectMask, "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getLastTransaction, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, errors.New(errorMessage)
	}

	lastTransaction := datatypes.SoftLayer_Provisioning_Version1_Transaction{}
	err = json.Unmarshal(response, &lastTransaction)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	return lastTransaction, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetActiveTransactions(instanceId int) ([]datatypes.SoftLayer_Provisioning_Version1_Transaction, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getActiveTransactions.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getActiveTransactions, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Provisioning_Version1_Transaction{}, errors.New(errorMessage)
	}

	activeTransactions := []datatypes.SoftLayer_Provisioning_Version1_Transaction{}
	err = json.Unmarshal(response, &activeTransactions)
	if err != nil {
		return []datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	return activeTransactions, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetSshKeys(instanceId int) ([]datatypes.SoftLayer_Security_Ssh_Key, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getSshKeys.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getSshKeys, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Security_Ssh_Key{}, errors.New(errorMessage)
	}

	sshKeys := []datatypes.SoftLayer_Security_Ssh_Key{}
	err = json.Unmarshal(response, &sshKeys)
	if err != nil {
		return []datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	return sshKeys, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) PowerCycle(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/powerCycle.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to power cycle instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#powerCycle, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slvgs *softLayer_Virtual_Guest_Service) PowerOff(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/powerOff.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to power off instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#powerOff, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slvgs *softLayer_Virtual_Guest_Service) PowerOffSoft(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/powerOffSoft.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to power off soft instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#powerOffSoft, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slvgs *softLayer_Virtual_Guest_Service) PowerOn(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/powerOn.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to power on instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#powerOn, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slvgs *softLayer_Virtual_Guest_Service) RebootDefault(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/rebootDefault.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to default reboot instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#rebootDefault, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slvgs *softLayer_Virtual_Guest_Service) RebootSoft(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/rebootSoft.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to soft reboot instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#rebootSoft, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slvgs *softLayer_Virtual_Guest_Service) RebootHard(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/rebootHard.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to hard reboot instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#rebootHard, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slvgs *softLayer_Virtual_Guest_Service) SetMetadata(instanceId int, metadata string) (bool, error) {
	dataBytes := []byte(metadata)
	base64EncodedMetadata := base64.StdEncoding.EncodeToString(dataBytes)

	parameters := datatypes.SoftLayer_SetUserMetadata_Parameters{
		Parameters: []datatypes.UserMetadataArray{
			[]datatypes.UserMetadata{datatypes.UserMetadata(base64EncodedMetadata)},
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/setUserMetadata.json", slvgs.GetName(), instanceId), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to setUserMetadata for instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#setUserMetadata, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}

func (slvgs *softLayer_Virtual_Guest_Service) ConfigureMetadataDisk(instanceId int) (datatypes.SoftLayer_Provisioning_Version1_Transaction, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/configureMetadataDisk.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#setUserMetadata, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, errors.New(errorMessage)
	}

	transaction := datatypes.SoftLayer_Provisioning_Version1_Transaction{}
	err = json.Unmarshal(response, &transaction)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	return transaction, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetUserData(instanceId int) ([]datatypes.SoftLayer_Virtual_Guest_Attribute, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getUserData.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Virtual_Guest_Attribute{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getUserData, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Virtual_Guest_Attribute{}, errors.New(errorMessage)
	}

	attributes := []datatypes.SoftLayer_Virtual_Guest_Attribute{}
	err = json.Unmarshal(response, &attributes)
	if err != nil {
		return []datatypes.SoftLayer_Virtual_Guest_Attribute{}, err
	}

	return attributes, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) IsPingable(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/isPingable.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#isPingable, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	res := string(response)

	if res == "true" {
		return true, nil
	}

	if res == "false" {
		return false, nil
	}

	return false, errors.New(fmt.Sprintf("Failed to checking that virtual guest is pingable for instance with id '%d', got '%s' as response from the API.", instanceId, res))
}

func (slvgs *softLayer_Virtual_Guest_Service) IsBackendPingable(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/isBackendPingable.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#isBackendPingable, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	res := string(response)

	if res == "true" {
		return true, nil
	}

	if res == "false" {
		return false, nil
	}

	return false, errors.New(fmt.Sprintf("Failed to checking that virtual guest backend is pingable for instance with id '%d', got '%s' as response from the API.", instanceId, res))
}

func (slvgs *softLayer_Virtual_Guest_Service) AttachEphemeralDisk(instanceId int, diskSize int) (datatypes.SoftLayer_Container_Product_Order_Receipt, error) {
	diskItemPrice, err := slvgs.findUpgradeItemPriceForEphemeralDisk(instanceId, diskSize)
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	orderService, err := slvgs.client.GetSoftLayer_Product_Order_Service()
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	order := datatypes.SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade{
		VirtualGuests: []datatypes.VirtualGuest{
			datatypes.VirtualGuest{
				Id: instanceId,
			},
		},
		Prices: []datatypes.SoftLayer_Product_Item_Price{
			datatypes.SoftLayer_Product_Item_Price{
				Id: diskItemPrice.Id,
				Categories: []datatypes.Category{
					datatypes.Category{
						CategoryCode: EPHEMERAL_DISK_CATEGORY_CODE,
					},
				},
			},
		},
		ComplexType: UPGRADE_VIRTUAL_SERVER_ORDER_TYPE,
		Properties: []datatypes.Property{
			datatypes.Property{
				Name:  MAINTENANCE_WINDOW_PROPERTY,
				Value: time.Now().UTC().Format(time.RFC3339),
			},
			datatypes.Property{
				Name:  "NOTE_GENERAL",
				Value: "addingdisks",
			},
		},
	}

	receipt, err := orderService.PlaceContainerOrderVirtualGuestUpgrade(order)
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}
	return receipt, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) UpgradeObject(instanceId int, options *softlayer.UpgradeOptions) (bool, error) {
	prices, err := slvgs.GetAvailableUpgradeItemPrices(options)
	if err != nil {
		return false, err
	}

	if len(prices) == 0 {
		// Nothing to order, as all the values are up to date
		return false, nil
	}

	orderService, err := slvgs.client.GetSoftLayer_Product_Order_Service()
	if err != nil {
		return false, err
	}

	order := datatypes.SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade{
		VirtualGuests: []datatypes.VirtualGuest{
			datatypes.VirtualGuest{
				Id: instanceId,
			},
		},
		Prices:      prices,
		ComplexType: UPGRADE_VIRTUAL_SERVER_ORDER_TYPE,
		Properties: []datatypes.Property{
			datatypes.Property{
				Name:  MAINTENANCE_WINDOW_PROPERTY,
				Value: time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	_, err = orderService.PlaceContainerOrderVirtualGuestUpgrade(order)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetAvailableUpgradeItemPrices(upgradeOptions *softlayer.UpgradeOptions) ([]datatypes.SoftLayer_Product_Item_Price, error) {
	itemsCapacity := make(map[string]int)
	if upgradeOptions.Cpus > 0 {
		itemsCapacity["cpus"] = upgradeOptions.Cpus
	}
	if upgradeOptions.MemoryInGB > 0 {
		itemsCapacity["memory"] = upgradeOptions.MemoryInGB
	}
	if upgradeOptions.NicSpeed > 0 {
		itemsCapacity["nic_speed"] = upgradeOptions.NicSpeed
	}

	virtualServerPackageItems, err := slvgs.getVirtualServerItems()
	if err != nil {
		return []datatypes.SoftLayer_Product_Item_Price{}, err
	}

	prices := make([]datatypes.SoftLayer_Product_Item_Price, 0)

	for item, amount := range itemsCapacity {
		price, err := slvgs.filterProductItemPrice(virtualServerPackageItems, item, amount)
		if err != nil {
			return []datatypes.SoftLayer_Product_Item_Price{}, err
		}

		prices = append(prices, price)
	}

	return prices, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetUpgradeItemPrices(instanceId int) ([]datatypes.SoftLayer_Product_Item_Price, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getUpgradeItemPrices.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Product_Item_Price{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getUpgradeItemPrices, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Product_Item_Price{}, errors.New(errorMessage)
	}

	itemPrices := []datatypes.SoftLayer_Product_Item_Price{}
	err = json.Unmarshal(response, &itemPrices)
	if err != nil {
		return []datatypes.SoftLayer_Product_Item_Price{}, err
	}

	return itemPrices, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) SetTags(instanceId int, tags []string) (bool, error) {
	var tagStringBuffer bytes.Buffer
	for i, tag := range tags {
		tagStringBuffer.WriteString(tag)
		if i != len(tags)-1 {
			tagStringBuffer.WriteString(", ")
		}
	}

	setTagsParameters := datatypes.SoftLayer_Virtual_Guest_SetTags_Parameters{
		Parameters: []string{tagStringBuffer.String()},
	}

	requestBody, err := json.Marshal(setTagsParameters)
	if err != nil {
		return false, err
	}

	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/setTags.json", slvgs.GetName(), instanceId), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#setTags, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to setTags for instance with id '%d', got '%s' as response from the API.", instanceId, res))
	}

	return true, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetTagReferences(instanceId int) ([]datatypes.SoftLayer_Tag_Reference, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getTagReferences.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Tag_Reference{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getTagReferences, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Tag_Reference{}, errors.New(errorMessage)
	}

	tagReferences := []datatypes.SoftLayer_Tag_Reference{}
	err = json.Unmarshal(response, &tagReferences)
	if err != nil {
		return []datatypes.SoftLayer_Tag_Reference{}, err
	}

	return tagReferences, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) AttachDiskImage(instanceId int, imageId int) (datatypes.SoftLayer_Provisioning_Version1_Transaction, error) {
	parameters := datatypes.SoftLayer_Virtual_GuestInit_ImageId_Parameters{
		Parameters: datatypes.ImageId_Parameter{
			ImageId: imageId,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/attachDiskImage.json", slvgs.GetName(), instanceId), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#attachDiskImage, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, errors.New(errorMessage)
	}

	transaction := datatypes.SoftLayer_Provisioning_Version1_Transaction{}
	err = json.Unmarshal(response, &transaction)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	return transaction, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) DetachDiskImage(instanceId int, imageId int) (datatypes.SoftLayer_Provisioning_Version1_Transaction, error) {
	parameters := datatypes.SoftLayer_Virtual_GuestInit_ImageId_Parameters{
		Parameters: datatypes.ImageId_Parameter{
			ImageId: imageId,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/detachDiskImage.json", slvgs.GetName(), instanceId), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#detachDiskImage, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, errors.New(errorMessage)
	}

	transaction := datatypes.SoftLayer_Provisioning_Version1_Transaction{}
	err = json.Unmarshal(response, &transaction)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	return transaction, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) ActivatePrivatePort(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/activatePrivatePort.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#activatePrivatePort, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	res := string(response)

	if res == "true" {
		return true, nil
	}

	if res == "false" {
		return false, nil
	}

	return false, errors.New(fmt.Sprintf("Failed to activate private port for virtual guest is pingable for instance with id '%d', got '%s' as response from the API.", instanceId, res))
}

func (slvgs *softLayer_Virtual_Guest_Service) ActivatePublicPort(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/activatePublicPort.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#activatePublicPort, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	res := string(response)

	if res == "true" {
		return true, nil
	}

	if res == "false" {
		return false, nil
	}

	return false, errors.New(fmt.Sprintf("Failed to activate public port for virtual guest is pingable for instance with id '%d', got '%s' as response from the API.", instanceId, res))
}

func (slvgs *softLayer_Virtual_Guest_Service) ShutdownPrivatePort(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/shutdownPrivatePort.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#shutdownPrivatePort, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	res := string(response)

	if res == "true" {
		return true, nil
	}

	if res == "false" {
		return false, nil
	}

	return false, errors.New(fmt.Sprintf("Failed to shutdown private port for virtual guest is pingable for instance with id '%d', got '%s' as response from the API.", instanceId, res))
}

func (slvgs *softLayer_Virtual_Guest_Service) ShutdownPublicPort(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/shutdownPublicPort.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#shutdownPublicPort, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	res := string(response)

	if res == "true" {
		return true, nil
	}

	if res == "false" {
		return false, nil
	}

	return false, errors.New(fmt.Sprintf("Failed to shutdown public port for virtual guest is pingable for instance with id '%d', got '%s' as response from the API.", instanceId, res))
}

func (slvgs *softLayer_Virtual_Guest_Service) GetAllowedHost(instanceId int) (datatypes.SoftLayer_Network_Storage_Allowed_Host, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getAllowedHost.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Network_Storage_Allowed_Host{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getAllowedHost, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Network_Storage_Allowed_Host{}, errors.New(errorMessage)
	}

	allowedHost := datatypes.SoftLayer_Network_Storage_Allowed_Host{}
	err = json.Unmarshal(response, &allowedHost)
	if err != nil {
		return datatypes.SoftLayer_Network_Storage_Allowed_Host{}, err
	}

	return allowedHost, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetNetworkVlans(instanceId int) ([]datatypes.SoftLayer_Network_Vlan, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getNetworkVlans.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Network_Vlan{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getNetworkVlans, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Network_Vlan{}, errors.New(errorMessage)
	}

	networkVlans := []datatypes.SoftLayer_Network_Vlan{}
	err = json.Unmarshal(response, &networkVlans)
	if err != nil {
		return []datatypes.SoftLayer_Network_Vlan{}, err
	}

	return networkVlans, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetNetworkComponents(instanceId int) ([]datatypes.SoftLayer_Virtual_Guest_Network_Component, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getNetworkComponents.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Virtual_Guest_Network_Component{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: SoftLayer_Virtual_Guest#getNetworkComponents failed, HTTP error code: '%d'", errorCode)
		return []datatypes.SoftLayer_Virtual_Guest_Network_Component{}, errors.New(errorMessage)
	}

	networkComponents := []datatypes.SoftLayer_Virtual_Guest_Network_Component{}
	err = json.Unmarshal(response, &networkComponents)
	if err != nil {
		return []datatypes.SoftLayer_Virtual_Guest_Network_Component{}, err
	}

	return networkComponents, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetPrimaryBackendNetworkComponent(instanceId int) (datatypes.SoftLayer_Virtual_Guest_Network_Component, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getPrimaryBackendNetworkComponent.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Network_Component{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: SoftLayer_Virtual_Guest#getPrimaryBackendNetworkComponent failed, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Virtual_Guest_Network_Component{}, errors.New(errorMessage)
	}

	networkComponent := datatypes.SoftLayer_Virtual_Guest_Network_Component{}
	err = json.Unmarshal(response, &networkComponent)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Network_Component{}, err
	}

	return networkComponent, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetPrimaryNetworkComponent(instanceId int) (datatypes.SoftLayer_Virtual_Guest_Network_Component, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getPrimaryNetworkComponent.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Network_Component{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: SoftLayer_Virtual_Guest#getPrimaryNetworkComponent failed, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Virtual_Guest_Network_Component{}, errors.New(errorMessage)
	}

	networkComponent := datatypes.SoftLayer_Virtual_Guest_Network_Component{}
	err = json.Unmarshal(response, &networkComponent)
	if err != nil {
		return datatypes.SoftLayer_Virtual_Guest_Network_Component{}, err
	}

	return networkComponent, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) CheckHostDiskAvailability(instanceId int, diskCapacity int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/checkHostDiskAvailability/%d", slvgs.GetName(), instanceId, diskCapacity), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#checkHostDiskAvailability, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	res := string(response)

	if res == "true" {
		return true, nil
	}

	if res == "false" {
		return false, nil
	}

	return false, errors.New(fmt.Sprintf("Failed to check host disk availability for instance '%d', got '%s' as response from the API.", instanceId, res))
}

func (slvgs *softLayer_Virtual_Guest_Service) CaptureImage(instanceId int) (datatypes.SoftLayer_Container_Disk_Image_Capture_Template, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/captureImage.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Container_Disk_Image_Capture_Template{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#captureImage, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Container_Disk_Image_Capture_Template{}, errors.New(errorMessage)
	}

	diskImageTemplate := datatypes.SoftLayer_Container_Disk_Image_Capture_Template{}
	err = json.Unmarshal(response, &diskImageTemplate)
	if err != nil {
		return datatypes.SoftLayer_Container_Disk_Image_Capture_Template{}, err
	}

	return diskImageTemplate, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) CreateArchiveTransaction(instanceId int, groupName string, blockDevices []datatypes.SoftLayer_Virtual_Guest_Block_Device, note string) (datatypes.SoftLayer_Provisioning_Version1_Transaction, error) {
	groupName = url.QueryEscape(groupName)
	note = url.QueryEscape(note)

	parameters := datatypes.SoftLayer_Virtual_GuestInitParameters{
		Parameters: []interface{}{groupName, blockDevices, note},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/createArchiveTransaction.json", slvgs.GetName(), instanceId), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#createArchiveTransaction, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, errors.New(errorMessage)
	}

	transaction := datatypes.SoftLayer_Provisioning_Version1_Transaction{}
	err = json.Unmarshal(response, &transaction)
	if err != nil {
		return datatypes.SoftLayer_Provisioning_Version1_Transaction{}, err
	}

	return transaction, nil
}

func (slvgs *softLayer_Virtual_Guest_Service) GetLocalDiskFlag(instanceId int) (bool, error) {
	response, errorCode, err := slvgs.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getLocalDiskFlag.json", slvgs.GetName(), instanceId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Virtual_Guest#getLocalDiskFlag, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	res := string(response)

	if res == "true" {
		return true, nil
	}

	if res == "false" {
		return false, nil
	}

	return false, errors.New(fmt.Sprintf("Failed to check the disk type (local or SAN) of that virtual guest with id '%d', got '%s' as response from the API.", instanceId, res))
}

//Private methods

func (slvgs *softLayer_Virtual_Guest_Service) getVirtualServerItems() ([]datatypes.SoftLayer_Product_Item, error) {
	service, err := slvgs.client.GetSoftLayer_Product_Package_Service()
	if err != nil {
		return []datatypes.SoftLayer_Product_Item{}, err
	}

	return service.GetItemsByType(VIRTUAL_SERVER_PACKAGE_TYPE)
}

func (slvgs *softLayer_Virtual_Guest_Service) filterProductItemPrice(packageItems []datatypes.SoftLayer_Product_Item, option string, amount int) (datatypes.SoftLayer_Product_Item_Price, error) {
	// for now use hardcoded values in the same "style" as Python client does
	// refer to corresponding Python method #_get_item_id_for_upgrade: https://github.com/softlayer/softlayer-python/blob/master/SoftLayer/managers/vs.py
	vsId := map[string]int{
		"memory":    3,
		"cpus":      80,
		"nic_speed": 26,
	}

	for _, packageItem := range packageItems {
		categories := packageItem.Prices[0].Categories
		for _, category := range categories {

			if packageItem.Capacity == "" {
				continue
			}

			capacity, err := strconv.Atoi(packageItem.Capacity)
			if err != nil {
				return datatypes.SoftLayer_Product_Item_Price{}, err
			}

			if category.Id != vsId[option] || capacity != amount {
				continue
			}

			switch option {
			case "cpus":
				if !strings.Contains(packageItem.Description, "Private") {
					return packageItem.Prices[0], nil
				}
			case "nic_speed":
				if strings.Contains(packageItem.Description, "Public") {
					return packageItem.Prices[0], nil
				}
			default:
				return packageItem.Prices[0], nil
			}
		}
	}

	return datatypes.SoftLayer_Product_Item_Price{}, errors.New(fmt.Sprintf("Failed to find price for '%s' (of size %d)", option, amount))
}

func (slvgs *softLayer_Virtual_Guest_Service) checkCreateObjectRequiredValues(template datatypes.SoftLayer_Virtual_Guest_Template) error {
	var err error
	errorMessage, errorTemplate := "", "* %s is required and cannot be empty\n"

	if template.Hostname == "" {
		errorMessage += fmt.Sprintf(errorTemplate, "Hostname for the computing instance")
	}

	if template.Domain == "" {
		errorMessage += fmt.Sprintf(errorTemplate, "Domain for the computing instance")
	}

	if template.StartCpus <= 0 {
		errorMessage += fmt.Sprintf(errorTemplate, "StartCpus: the number of CPU cores to allocate")
	}

	if template.MaxMemory <= 0 {
		errorMessage += fmt.Sprintf(errorTemplate, "MaxMemory: the amount of memory to allocate in megabytes")
	}

	for _, device := range template.BlockDevices {
		if device.DiskImage.Capacity <= 0 {
			errorMessage += fmt.Sprintf("Disk size must be positive number, the size of block device %s is set to be %dGB.", device.Device, device.DiskImage.Capacity)
		}
	}

	if template.Datacenter.Name == "" {
		errorMessage += fmt.Sprintf(errorTemplate, "Datacenter.Name: specifies which datacenter the instance is to be provisioned in")
	}

	if errorMessage != "" {
		err = errors.New(errorMessage)
	}

	return err
}

func (slvgs *softLayer_Virtual_Guest_Service) findUpgradeItemPriceForEphemeralDisk(instanceId int, ephemeralDiskSize int) (datatypes.SoftLayer_Product_Item_Price, error) {
	if ephemeralDiskSize <= 0 {
		return datatypes.SoftLayer_Product_Item_Price{}, fmt.Errorf("Ephemeral disk size can not be negative: %d", ephemeralDiskSize)
	}

	itemPrices, err := slvgs.GetUpgradeItemPrices(instanceId)
	if err != nil {
		return datatypes.SoftLayer_Product_Item_Price{}, err
	}

	var currentDiskCapacity int
	var currentItemPrice datatypes.SoftLayer_Product_Item_Price
	var diskType string

	diskTypeBool, err := slvgs.GetLocalDiskFlag(instanceId)
	if err != nil {
		return datatypes.SoftLayer_Product_Item_Price{}, err
	}
	if diskTypeBool {
		diskType = "(LOCAL)"
	} else {
		diskType = "(SAN)"
	}

	for _, itemPrice := range itemPrices {
		flag := false
		for _, category := range itemPrice.Categories {
			if category.CategoryCode == EPHEMERAL_DISK_CATEGORY_CODE {
				flag = true
				break
			}
		}

		if flag && strings.Contains(itemPrice.Item.Description, diskType) {
			capacity, _ := strconv.Atoi(itemPrice.Item.Capacity)

			if capacity >= ephemeralDiskSize {
				if currentItemPrice.Id == 0 || currentDiskCapacity >= capacity {
					currentItemPrice = itemPrice
					currentDiskCapacity = capacity
				}
			}
		}
	}

	if currentItemPrice.Id == 0 {
		return datatypes.SoftLayer_Product_Item_Price{}, fmt.Errorf("No proper local disk for size %d", ephemeralDiskSize)
	}

	return currentItemPrice, nil
}
