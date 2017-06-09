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

type softLayer_Billing_Item_Cancellation_Request_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Billing_Item_Cancellation_Request_Service(client softlayer.Client) *softLayer_Billing_Item_Cancellation_Request_Service {
	return &softLayer_Billing_Item_Cancellation_Request_Service{
		client: client,
	}
}

func (slbicr *softLayer_Billing_Item_Cancellation_Request_Service) GetName() string {
	return "SoftLayer_Billing_Item_Cancellation_Request"
}

func (slbicr *softLayer_Billing_Item_Cancellation_Request_Service) CreateObject(request datatypes.SoftLayer_Billing_Item_Cancellation_Request) (datatypes.SoftLayer_Billing_Item_Cancellation_Request, error) {
	parameters := datatypes.SoftLayer_Billing_Item_Cancellation_Request_Parameters{
		Parameters: []datatypes.SoftLayer_Billing_Item_Cancellation_Request{
			request,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Billing_Item_Cancellation_Request{}, err
	}

	responseBytes, errorCode, err := slbicr.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/createObject.json", slbicr.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Billing_Item_Cancellation_Request{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Billing_Item_Cancellation_Request#createObject TTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Billing_Item_Cancellation_Request{}, errors.New(errorMessage)
	}

	result := datatypes.SoftLayer_Billing_Item_Cancellation_Request{}
	err = json.Unmarshal(responseBytes, &result)
	if err != nil {
		return datatypes.SoftLayer_Billing_Item_Cancellation_Request{}, err
	}

	return result, nil
}
