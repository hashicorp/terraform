package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	datatypes "github.com/maximilien/softlayer-go/data_types"
	softlayer "github.com/maximilien/softlayer-go/softlayer"
)

const (
	OUTLET_PACKAGE = "OUTLET"
)

type softLayer_Product_Package_Service struct {
	client softlayer.Client
}

func NewSoftLayer_Product_Package_Service(client softlayer.Client) *softLayer_Product_Package_Service {
	return &softLayer_Product_Package_Service{
		client: client,
	}
}

func (slpp *softLayer_Product_Package_Service) GetName() string {
	return "SoftLayer_Product_Package"
}

func (slpp *softLayer_Product_Package_Service) GetItemPrices(packageId int) ([]datatypes.SoftLayer_Product_Item_Price, error) {
	response, err := slpp.client.DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getItemPrices.json", slpp.GetName(), packageId), []string{"id", "item.id", "item.description", "item.capacity"}, "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Product_Item_Price{}, err
	}

	itemPrices := []datatypes.SoftLayer_Product_Item_Price{}
	err = json.Unmarshal(response, &itemPrices)
	if err != nil {
		return []datatypes.SoftLayer_Product_Item_Price{}, err
	}

	return itemPrices, nil
}

func (slpp *softLayer_Product_Package_Service) GetItemPricesBySize(packageId int, size int) ([]datatypes.SoftLayer_Product_Item_Price, error) {
	keyName := strconv.Itoa(size) + "_GB_PERFORMANCE_STORAGE_SPACE"
	filter := string(`{"itemPrices":{"item":{"keyName":{"operation":"` + keyName + `"}}}}`)

	response, err := slpp.client.DoRawHttpRequestWithObjectFilterAndObjectMask(fmt.Sprintf("%s/%d/getItemPrices.json", slpp.GetName(), packageId), []string{"id", "locationGroupId", "item.id", "item.keyName", "item.units", "item.description", "item.capacity"}, fmt.Sprintf(string(filter)), "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Product_Item_Price{}, err
	}

	itemPrices := []datatypes.SoftLayer_Product_Item_Price{}
	err = json.Unmarshal(response, &itemPrices)
	if err != nil {
		return []datatypes.SoftLayer_Product_Item_Price{}, err
	}

	return itemPrices, nil
}

func (slpp *softLayer_Product_Package_Service) GetItemsByType(packageType string) ([]datatypes.SoftLayer_Product_Item, error) {
	productPackage, err := slpp.GetOnePackageByType(packageType)
	if err != nil {
		return []datatypes.SoftLayer_Product_Item{}, err
	}

	return slpp.GetItems(productPackage.Id)
}

func (slpp *softLayer_Product_Package_Service) GetItems(packageId int) ([]datatypes.SoftLayer_Product_Item, error) {
	objectMasks := []string{
		"id",
		"capacity",
		"description",
		"prices.id",
		"prices.categories.id",
		"prices.categories.name",
	}

	response, err := slpp.client.DoRawHttpRequestWithObjectMask(fmt.Sprintf("%s/%d/getItems.json", slpp.GetName(), packageId), objectMasks, "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.SoftLayer_Product_Item{}, err
	}

	productItems := []datatypes.SoftLayer_Product_Item{}
	err = json.Unmarshal(response, &productItems)
	if err != nil {
		return []datatypes.SoftLayer_Product_Item{}, err
	}

	return productItems, nil
}

func (slpp *softLayer_Product_Package_Service) GetOnePackageByType(packageType string) (datatypes.Softlayer_Product_Package, error) {
	productPackages, err := slpp.GetPackagesByType(packageType)
	if err != nil {
		return datatypes.Softlayer_Product_Package{}, err
	}

	if len(productPackages) == 0 {
		return datatypes.Softlayer_Product_Package{}, errors.New(fmt.Sprintf("No packages available for type '%s'.", packageType))
	}

	return productPackages[0], nil
}

func (slpp *softLayer_Product_Package_Service) GetPackagesByType(packageType string) ([]datatypes.Softlayer_Product_Package, error) {
	objectMasks := []string{
		"id",
		"name",
		"description",
		"isActive",
		"type.keyName",
	}

	filterObject := string(`{"type":{"keyName":{"operation":"` + packageType + `"}}}`)

	response, err := slpp.client.DoRawHttpRequestWithObjectFilterAndObjectMask(fmt.Sprintf("%s/getAllObjects.json", slpp.GetName()), objectMasks, filterObject, "GET", new(bytes.Buffer))
	if err != nil {
		return []datatypes.Softlayer_Product_Package{}, err
	}

	productPackages := []*datatypes.Softlayer_Product_Package{}
	err = json.Unmarshal(response, &productPackages)
	if err != nil {
		return []datatypes.Softlayer_Product_Package{}, err
	}

	// Remove packages designated as OUTLET
	// See method "#get_packages_of_type" in SoftLayer Python client for details: https://github.com/softlayer/softlayer-python/blob/master/SoftLayer/managers/ordering.py
	nonOutletPackages := slpp.filterProducts(productPackages, func(productPackage *datatypes.Softlayer_Product_Package) bool {
		return !strings.Contains(productPackage.Description, OUTLET_PACKAGE) && !strings.Contains(productPackage.Name, OUTLET_PACKAGE)
	})

	return nonOutletPackages, nil
}

//Private methods

func (slpp *softLayer_Product_Package_Service) filterProducts(array []*datatypes.Softlayer_Product_Package, predicate func(*datatypes.Softlayer_Product_Package) bool) []datatypes.Softlayer_Product_Package {
	filtered := make([]datatypes.Softlayer_Product_Package, 0)
	for _, element := range array {
		if predicate(element) {
			filtered = append(filtered, *element)
		}
	}
	return filtered
}
