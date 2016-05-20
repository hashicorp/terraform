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

type softLayer_Network_Storage_Allowed_Host_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Network_Storage_Allowed_Host_Service(client softlayer.Client) *softLayer_Network_Storage_Allowed_Host_Service {
	return &softLayer_Network_Storage_Allowed_Host_Service{
		client: client,
	}
}

func (slns *softLayer_Network_Storage_Allowed_Host_Service) GetName() string {
	return "SoftLayer_Network_Storage_Allowed_Host"
}

func (slns *softLayer_Network_Storage_Allowed_Host_Service) GetCredential(allowedHostId int) (datatypes.SoftLayer_Network_Storage_Credential, error) {
	response, errorCode, err := slns.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/getCredential.json", slns.GetName(), allowedHostId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Network_Storage_Credential{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Network_Storage_Allowed_Host#getCredential, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Network_Storage_Credential{}, errors.New(errorMessage)
	}

	credential := datatypes.SoftLayer_Network_Storage_Credential{}
	err = json.Unmarshal(response, &credential)
	if err != nil {
		return datatypes.SoftLayer_Network_Storage_Credential{}, err
	}

	return credential, nil
}
