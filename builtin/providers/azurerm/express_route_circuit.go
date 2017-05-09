package azurerm

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
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

func retrieveErcByResourceId(resourceId string, meta interface{}) (erc *network.ExpressRouteCircuit, resourceGroup string, e error) {
	ercClient := meta.(*ArmClient).expressRouteCircuitClient

	resGroup, name, err := extractResourceGroupAndErcName(resourceId)
	if err != nil {
		return nil, "", errwrap.Wrapf("Error Parsing Azure Resource ID - {{err}}", err)
	}

	resp, err := ercClient.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, "", nil
		}
		return nil, "", errwrap.Wrapf(fmt.Sprintf("Error making Read request on Express Route Circuit %s: {{err}}", name), err)
	}

	return &resp, resGroup, nil
}

func expandExpressRouteCircuitSku(skuSettings *schema.Set) *network.ExpressRouteCircuitSku {
	v := skuSettings.List()[0].(map[string]interface{}) // [0] is guarded by MinItems in schema.
	tier := v["tier"].(string)
	family := v["family"].(string)
	name := fmt.Sprintf("%s_%s", tier, family)

	return &network.ExpressRouteCircuitSku{
		Name:   &name,
		Tier:   network.ExpressRouteCircuitSkuTier(tier),
		Family: network.ExpressRouteCircuitSkuFamily(family),
	}
}

func flattenExpressRouteCircuitSku(sku *network.ExpressRouteCircuitSku) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"tier":   string(sku.Tier),
			"family": string(sku.Family),
		},
	}
}

func resourceArmExpressRouteCircuitSkuHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", strings.ToLower(m["tier"].(string))))
	buf.WriteString(fmt.Sprintf("%s-", strings.ToLower(m["family"].(string))))

	return hashcode.String(buf.String())
}
