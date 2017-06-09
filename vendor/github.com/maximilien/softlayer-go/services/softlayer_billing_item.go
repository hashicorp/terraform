package services

import (
	"bytes"
	"errors"
	"fmt"
	common "github.com/maximilien/softlayer-go/common"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
)

type softLayer_Billing_Item_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Billing_Item_Service(client softlayer.Client) *softLayer_Billing_Item_Service {
	return &softLayer_Billing_Item_Service{
		client: client,
	}
}

func (slbi *softLayer_Billing_Item_Service) GetName() string {
	return "SoftLayer_Billing_Item"
}

func (slbi *softLayer_Billing_Item_Service) CancelService(billingId int) (bool, error) {
	response, errorCode, err := slbi.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/%d/cancelService.json", slbi.GetName(), billingId), "GET", new(bytes.Buffer))
	if err != nil {
		return false, err
	}

	if res := string(response[:]); res != "true" {
		return false, nil
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Billing_Item#CancelService, HTTP error code: '%d'", errorCode)
		return false, errors.New(errorMessage)
	}

	return true, err
}
