package test_helpers

import (
	"encoding/json"
	"errors"
	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
)

type MockProductPackageService struct{}

func (mock *MockProductPackageService) GetName() string {
	return "Mock_Product_Package_Service"
}

func (mock *MockProductPackageService) GetItemsByType(packageType string) ([]datatypes.SoftLayer_Product_Item, error) {
	var response []byte
	switch packageType {
	case "ADDITIONAL_SERVICES_APPLICATION_DELIVERY_APPLIANCE":
		response, _ = ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getItemsByType_vpx.json")
	case "VIRTUAL_SERVER_INSTANCE":
		response, _ = ReadJsonTestFixtures("services", "SoftLayer_Product_Package_getItemsByType_virtual_server.json")
	}

	productItems := []datatypes.SoftLayer_Product_Item{}
	json.Unmarshal(response, &productItems)

	return productItems, nil
}

func (mock *MockProductPackageService) GetItemPrices(packageId int) ([]datatypes.SoftLayer_Product_Item_Price, error) {
	return []datatypes.SoftLayer_Product_Item_Price{}, errors.New("Not supported")
}

func (mock *MockProductPackageService) GetItemPricesBySize(packageId int, size int) ([]datatypes.SoftLayer_Product_Item_Price, error) {
	return []datatypes.SoftLayer_Product_Item_Price{}, errors.New("Not supported")
}

func (mock *MockProductPackageService) GetItems(packageId int) ([]datatypes.SoftLayer_Product_Item, error) {
	return []datatypes.SoftLayer_Product_Item{}, errors.New("Not supported")
}

func (mock *MockProductPackageService) GetPackagesByType(packageType string) ([]datatypes.Softlayer_Product_Package, error) {
	return []datatypes.Softlayer_Product_Package{}, errors.New("Not supported")
}

func (mock *MockProductPackageService) GetOnePackageByType(packageType string) (datatypes.Softlayer_Product_Package, error) {
	return datatypes.Softlayer_Product_Package{}, errors.New("Not supported")
}
