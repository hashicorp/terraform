package services

import (
	"bytes"
	"encoding/json"
	"fmt"

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

	response, err := slns.client.DoRawHttpRequest(fmt.Sprintf("%s/%d/getCredential.json", slns.GetName(), allowedHostId), "GET", new(bytes.Buffer))
	if err != nil {
		return datatypes.SoftLayer_Network_Storage_Credential{}, err
	}

	credential := datatypes.SoftLayer_Network_Storage_Credential{}
	err = json.Unmarshal(response, &credential)
	if err != nil {
		return datatypes.SoftLayer_Network_Storage_Credential{}, err
	}

	return credential, nil
}
