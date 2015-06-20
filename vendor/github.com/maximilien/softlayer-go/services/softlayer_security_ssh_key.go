package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	datatypes "github.com/maximilien/softlayer-go/data_types"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
)

type softLayer_Security_Ssh_Key_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Security_Ssh_Key_Service(client softlayer.Client) *softLayer_Security_Ssh_Key_Service {
	return &softLayer_Security_Ssh_Key_Service{
		client: client,
	}
}

func (slssks *softLayer_Security_Ssh_Key_Service) GetName() string {
	return "SoftLayer_Security_Ssh_Key"
}

func (slssks *softLayer_Security_Ssh_Key_Service) CreateObject(template datatypes.SoftLayer_Security_Ssh_Key) (datatypes.SoftLayer_Security_Ssh_Key, error) {
	parameters := datatypes.SoftLayer_Shh_Key_Parameters{
		Parameters: []datatypes.SoftLayer_Security_Ssh_Key{
			template,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	data, err := slssks.client.DoRawHttpRequest(fmt.Sprintf("%s/createObject", slssks.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	err = slssks.client.CheckForHttpResponseErrors(data)
	if err != nil {
		return datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	softLayer_Ssh_Key := datatypes.SoftLayer_Security_Ssh_Key{}
	err = json.Unmarshal(data, &softLayer_Ssh_Key)
	if err != nil {
		return datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	return softLayer_Ssh_Key, nil
}

func (slssks *softLayer_Security_Ssh_Key_Service) GetObject(sshKeyId int) (datatypes.SoftLayer_Security_Ssh_Key, error) {
	objectMask := []string{
		"createDate",
		"fingerprint",
		"id",
		"key",
		"label",
		"modifyDate",
		"notes",
	}

	response, err := slssks.client.DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getObject.json", slssks.GetName(), sshKeyId), objectMask, "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	sshKey := datatypes.SoftLayer_Security_Ssh_Key{}
	err = json.Unmarshal(response, &sshKey)
	if err != nil {
		return datatypes.SoftLayer_Security_Ssh_Key{}, err
	}

	return sshKey, nil
}

func (slssks *softLayer_Security_Ssh_Key_Service) EditObject(sshKeyId int, template datatypes.SoftLayer_Security_Ssh_Key) (bool, error) {
	parameters := datatypes.SoftLayer_Shh_Key_Parameters{
		Parameters: []datatypes.SoftLayer_Security_Ssh_Key{
			template,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return false, err
	}

	response, err := slssks.client.DoRawHttpRequest(fmt.Sprintf("%s/%d/editObject.json", slssks.GetName(), sshKeyId), "POST", bytes.NewBuffer(requestBody))

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to edit SSH key with id: %d, got '%s' as response from the API.", sshKeyId, res))
	}

	return true, err
}

func (slssks *softLayer_Security_Ssh_Key_Service) DeleteObject(sshKeyId int) (bool, error) {
	response, err := slssks.client.DoRawHttpRequest(fmt.Sprintf("%s/%d.json", slssks.GetName(), sshKeyId), "DELETE", new(bytes.Buffer))

	if res := string(response[:]); res != "true" {
		return false, errors.New(fmt.Sprintf("Failed to destroy ssh key with id '%d', got '%s' as response from the API.", sshKeyId, res))
	}

	return true, err
}

func (slssks *softLayer_Security_Ssh_Key_Service) GetSoftwarePasswords(sshKeyId int) ([]datatypes.SoftLayer_Software_Component_Password, error) {
	response, err := slssks.client.DoRawHttpRequest(fmt.Sprintf("%s/%d/getSoftwarePasswords.json", slssks.GetName(), sshKeyId), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Software_Component_Password{}, err
	}

	passwords := []datatypes.SoftLayer_Software_Component_Password{}
	err = json.Unmarshal(response, &passwords)
	if err != nil {
		return []datatypes.SoftLayer_Software_Component_Password{}, err
	}

	return passwords, nil
}
