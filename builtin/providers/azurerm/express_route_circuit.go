package azurerm

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/errwrap"
)

func extractResourceGroupAndErcName(resourceId string) (resourceGroup string, name string, err error) {
	id, err := parseAzureResourceID(resourceId)

	if err != nil {
		return "", "", err
	}
	resourceGroup = id.ResourceGroup
	name = id.Path["expressRouteCircuits"]

	return
}

func retrieveErcByResourceId(resourceId string, meta interface{}) (*network.ExpressRouteCircuit, string, bool, error) {
	ercClient := meta.(*ArmClient).expressRouteCircuitClient

	resGroup, name, err := extractResourceGroupAndErcName(resourceId)
	if err != nil {
		return nil, "", false, errwrap.Wrapf("Error Parsing Azure Resource ID {{err}}", err)
	}

	resp, err := ercClient.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, "", false, nil
		}
		return nil, "", false, fmt.Errorf("Error making Read request on Express Route Circuit %s: %s", name, err)
	}

	return &resp, resGroup, true, nil
}

func validateSkuTier(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	allowedValues := []string{
		string(network.ExpressRouteCircuitSkuTierStandard),
		string(network.ExpressRouteCircuitSkuTierPremium),
	}

	if !isStringValueAllowed(value, allowedValues) {
		errors = append(errors, fmt.Errorf(`Allowed sku_tier value(s) are %+v, provided value is "%s"`, allowedValues, value))
	}
	return
}

func validateSkuFamily(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	allowedValues := []string{
		string(network.MeteredData),
		string(network.UnlimitedData),
	}

	if !isStringValueAllowed(value, allowedValues) {
		errors = append(errors, fmt.Errorf(`Allowed sku_family value(s) are %+v, provided value is "%s"`, allowedValues, value))
	}
	return
}

func isStringValueAllowed(v string, allowedValues []string) bool {
	for _, allowed := range allowedValues {
		if strings.ToLower(v) == strings.ToLower(allowed) {
			return true
		}
	}
	return false
}
