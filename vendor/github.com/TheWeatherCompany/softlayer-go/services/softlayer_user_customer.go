package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/TheWeatherCompany/softlayer-go/common"
	"github.com/TheWeatherCompany/softlayer-go/data_types"
	"github.com/TheWeatherCompany/softlayer-go/softlayer"
)

type softlayer_user_customer_service struct {
	client softlayer.Client
}

func NewSoftLayer_User_Customer_Service(client softlayer.Client) *softlayer_user_customer_service {
	return &softlayer_user_customer_service{
		client: client,
	}
}

func (slucs *softlayer_user_customer_service) GetName() string {
	return "SoftLayer_User_Customer"
}

func (slucs *softlayer_user_customer_service) GetApiAuthenticationKeys(userId int) ([]data_types.SoftLayer_User_Customer_ApiAuthentication, error) {
	path := fmt.Sprintf("%s/%d/%s", slucs.GetName(), userId, "getApiAuthenticationKeys.json")
	responseBytes, errorCode, err := slucs.client.GetHttpClient().DoRawHttpRequest(path, "GET", &bytes.Buffer{})
	if err != nil {
		errorMessage := fmt.Sprintf(
			"softlayer-go: could not %s#getApiAuthenticationKeys, error message '%s'",
			slucs.GetName(), err.Error(),
		)
		return nil, errors.New(errorMessage)
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf(
			"softlayer-go: could not %s#getApiAuthenticationKeys, HTTP error code: '%d'",
			slucs.GetName(), errorCode,
		)
		return nil, errors.New(errorMessage)
	}

	apiKeys := []data_types.SoftLayer_User_Customer_ApiAuthentication{}
	err = json.Unmarshal(responseBytes, &apiKeys)
	if err != nil {
		errorMessage := fmt.Sprintf(
			"softlayer-go: failed to decode JSON response from %s#getApiAuthenticationKeys, err message '%s'",
			slucs.GetName(), err.Error(),
		)
		err := errors.New(errorMessage)
		return nil, err
	}

	return apiKeys, nil
}

func (slucs *softlayer_user_customer_service) AddApiAuthenticationKey(userId int) error {
	path := fmt.Sprintf("%s/%d/%s", slucs.GetName(), userId, "addApiAuthenticationKey.json")
	_, errorCode, err := slucs.client.GetHttpClient().DoRawHttpRequest(path, "GET", &bytes.Buffer{})
	if err != nil {
		errorMessage := fmt.Sprintf(
			"softlayer-go: could not %s#addApiAuthenticationKey, error message '%s'",
			slucs.GetName(), err.Error(),
		)
		return errors.New(errorMessage)
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf(
			"softlayer-go: could not %s#addApiAuthenticationKey, HTTP error code: '%d'",
			slucs.GetName(), errorCode,
		)
		return errors.New(errorMessage)
	}

	return nil
}
