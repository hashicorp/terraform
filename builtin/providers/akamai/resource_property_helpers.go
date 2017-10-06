package akamai

import (
	"errors"
	"log"
	"strings"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func getGroup(d *schema.ResourceData) (*papi.Group, error) {
	log.Println("[DEBUG] Fetching groups")
	groupId := d.Get("group_id").(string)

	groups := papi.NewGroups()
	e := groups.GetGroups()
	if e != nil {
		return nil, e
	}

	group, e := groups.FindGroup(groupId)
	if e != nil {
		return nil, e
	}

	log.Printf("[DEBUG] Group found: %s\n", group.GroupID)
	return group, nil
}

func getContract(d *schema.ResourceData) (*papi.Contract, error) {
	log.Println("[DEBUG] Fetching contract")
	contractId := d.Get("contract_id").(string)

	contracts := papi.NewContracts()
	e := contracts.GetContracts()
	if e != nil {
		return nil, e
	}

	contract, e := contracts.FindContract(contractId)
	if e != nil {
		return nil, e
	}

	log.Printf("[DEBUG] Contract found: %s\n", contract.ContractID)
	return contract, nil
}

func getProduct(d *schema.ResourceData, contract *papi.Contract) (*papi.Product, error) {
	log.Println("[DEBUG] Fetching product")
	productId := d.Get("product_id").(string)

	products := papi.NewProducts()
	e := products.GetProducts(contract)
	if e != nil {
		return nil, e
	}

	product, e := products.FindProduct(productId)
	if e != nil {
		return nil, e
	}

	log.Printf("[DEBUG] Product found: %s\n", product.ProductID)
	return product, nil
}

func createCpCode(contract *papi.Contract, group *papi.Group, product *papi.Product, d *schema.ResourceData) (*papi.CpCode, error) {
	log.Println("[DEBUG] Setting up CPCode")
	cpCodes := papi.NewCpCodes(contract, group)
	cpCode := papi.NewCpCode(cpCodes)
	cpCode.CpcodeID = d.Get("cp_code").(string)
	if !strings.HasPrefix(cpCode.CpcodeID, "cpc_") {
		cpCode.CpcodeID = "cpc_" + cpCode.CpcodeID
	}
	if err := cpCode.GetCpCode(); err != nil {
		cpCode.CpcodeID = ""
		cpCodes.GetCpCodes()
		cpCode, err := cpCodes.FindCpCode(d.Get("cp_code").(string))
		if err != nil {
			return nil, err
		}

		if cpCode == nil {
			log.Println("[DEBUG] CPCode not found, creating a new one")
			cpCode = papi.NewCpCode(cpCodes)
			cpCode.ProductID = product.ProductID
			cpCode.CpcodeName = d.Get("cp_code").(string)
			err := cpCode.Save()
			if err != nil {
				return nil, err
			}
			log.Println("[DEBUG] CPCode created")
		}
	}
	log.Println("[DEBUG] CPCode set up")

	return cpCode, nil
}

func createOrigin(d *schema.ResourceData) (*papi.OptionValue, error) {
	log.Println("[DEBUG] Setting origin")
	if origin, ok := d.GetOk("origin"); ok {
		originConfig := origin.([]interface{})[0].(map[string]interface{})
		forwardHostname := originConfig["forward_hostname"].(string)
		var originValues *papi.OptionValue
		if forwardHostname == "ORIGIN_HOSTNAME" || forwardHostname == "REQUEST_HOST_HEADER" {
			log.Println("[DEBUG] Setting non-custom forward hostname")
			originValues = &papi.OptionValue{
				"originType":         "CUSTOMER",
				"hostname":           originConfig["hostname"].(string),
				"httpPort":           originConfig["port"].(int),
				"forwardHostHeader":  forwardHostname,
				"cacheKeyHostname":   originConfig["cache_key_hostname"].(string),
				"compress":           originConfig["gzip_compression"].(bool),
				"enableTrueClientIp": originConfig["true_client_ip_header"].(bool),
			}
		} else {
			log.Println("[DEBUG] Setting custom forward hostname")
			originValues = &papi.OptionValue{
				"originType":              "CUSTOMER",
				"hostname":                originConfig["hostname"].(string),
				"httpPort":                originConfig["port"].(string),
				"forwardHostHeader":       "CUSTOM",
				"customForwardHostHeader": forwardHostname,
				"cacheKeyHostname":        originConfig["cache_key_hostname"].(string),
				"compress":                originConfig["gzip_compression"].(bool),
				"enableTrueClientIp":      originConfig["true_client_ip_header"].(bool),
			}
		}
		return originValues, nil
	}
	return nil, errors.New("No origin config found")
}

func addStandardBehaviors(rules *papi.Rules, cpCode *papi.CpCode, origin *papi.OptionValue) {
	b := papi.NewBehavior()
	b.Name = "cpCode"
	b.Options = &papi.OptionValue{
		"value": &papi.OptionValue{
			"id": cpCode.ID(),
		},
	}
	rules.Rule.AddBehavior(b)

	b = papi.NewBehavior()
	b.Name = "origin"
	b.Options = origin
	rules.Rule.AddBehavior(b)

	log.Println("[DEBUG] Setting Performance")
	r := papi.NewRule()
	r.Name = "Performance"

	b = papi.NewBehavior()
	b.Name = "sureRoute"
	b.Options = &papi.OptionValue{
		"testObjectUrl":   "/akamai/sureroute-testobject.html",
		"enableCustomKey": false,
		"enabled":         false,
	}
	r.AddBehavior(b)

	// log.Println("[DEBUG] Fixing Image compression settings")
	// b = papi.NewBehavior()
	// b.Name = "adaptiveImageCompression"
	// b.Options = &papi.OptionValue{
	// 	"compressMobile":               true,
	// 	"tier1MobileCompressionMethod": "BYPASS",
	// 	"tier2MobileCompressionMethod": "COMPRESS",
	// 	"tier2MobileCompressionValue":  60,
	// }
	// r.AddBehavior(b)
	rules.Rule.AddChildRule(r)
}
