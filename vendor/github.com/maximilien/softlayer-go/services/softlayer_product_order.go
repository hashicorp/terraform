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

type softLayer_Product_Order_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Product_Order_Service(client softlayer.Client) *softLayer_Product_Order_Service {
	return &softLayer_Product_Order_Service{
		client: client,
	}
}

func (slpo *softLayer_Product_Order_Service) GetName() string {
	return "SoftLayer_Product_Order"
}

func (slpo *softLayer_Product_Order_Service) PlaceOrder(order datatypes.SoftLayer_Container_Product_Order) (datatypes.SoftLayer_Container_Product_Order_Receipt, error) {
	parameters := datatypes.SoftLayer_Container_Product_Order_Parameters{
		Parameters: []datatypes.SoftLayer_Container_Product_Order{
			order,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	responseBytes, errorCode, err := slpo.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/placeOrder.json", slpo.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Account#getAccountStatus, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, errors.New(errorMessage)
	}

	receipt := datatypes.SoftLayer_Container_Product_Order_Receipt{}
	err = json.Unmarshal(responseBytes, &receipt)
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	return receipt, nil
}

func (slpo *softLayer_Product_Order_Service) PlaceContainerOrderNetworkPerformanceStorageIscsi(order datatypes.SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi) (datatypes.SoftLayer_Container_Product_Order_Receipt, error) {
	parameters := datatypes.SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi_Parameters{
		Parameters: []datatypes.SoftLayer_Container_Product_Order_Network_PerformanceStorage_Iscsi{
			order,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	responseBytes, errorCode, err := slpo.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/placeOrder.json", slpo.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Product_Order#placeContainerOrderNetworkPerformanceStorageIscsi, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, errors.New(errorMessage)
	}

	receipt := datatypes.SoftLayer_Container_Product_Order_Receipt{}
	err = json.Unmarshal(responseBytes, &receipt)
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	return receipt, nil
}

func (slpo *softLayer_Product_Order_Service) PlaceContainerOrderVirtualGuestUpgrade(order datatypes.SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade) (datatypes.SoftLayer_Container_Product_Order_Receipt, error) {
	parameters := datatypes.SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade_Parameters{
		Parameters: []datatypes.SoftLayer_Container_Product_Order_Virtual_Guest_Upgrade{
			order,
		},
	}

	requestBody, err := json.Marshal(parameters)
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	responseBytes, errorCode, err := slpo.client.GetHttpClient().DoRawHttpRequest(fmt.Sprintf("%s/placeOrder.json", slpo.GetName()), "POST", bytes.NewBuffer(requestBody))
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	if common.IsHttpErrorCode(errorCode) {
		errorMessage := fmt.Sprintf("softlayer-go: could not SoftLayer_Product_Order#placeOrder, HTTP error code: '%d'", errorCode)
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, errors.New(errorMessage)
	}

	receipt := datatypes.SoftLayer_Container_Product_Order_Receipt{}
	err = json.Unmarshal(responseBytes, &receipt)
	if err != nil {
		return datatypes.SoftLayer_Container_Product_Order_Receipt{}, err
	}

	return receipt, nil
}
