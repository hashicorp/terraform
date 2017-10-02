package akamai

import (
	"log"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/papi-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceProperty() *schema.Resource {
	return &schema.Resource{
		Create: resourcePropertyCreate,
		Read:   resourcePropertyRead,
		Update: resourcePropertyCreate,
		Delete: resourcePropertyDelete,
		Exists: resourcePropertyExists,
		// Importer: &schema.ResourceImporter{
		// State: importRecord,
		// },
		Schema: akamaiPropertySchema,
	}
}

func resourcePropertyCreate(d *schema.ResourceData, meta interface{}) error {
	r := d.Get("rule")
	log.Printf("[DEBUG] Done with %s rules", r.(*schema.Set).Len())

	group, err := getGroup(d)
	if err != nil {
		return err
	}

	contract, err := getContract(d)
	if err != nil {
		return err
	}

	product, err := getProduct(d, contract)
	if err != nil {
		return err
	}

	_, err = createProperty(contract, group, product, d)
	if err != nil {
		return err
	}

	log.Println("[DEBUG] Done")

	return nil
}

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

func createProperty(contract *papi.Contract, group *papi.Group, product *papi.Product, d *schema.ResourceData) (*papi.Property, error) {
	log.Println("[DEBUG] Creating property")

	property, err := group.NewProperty(contract)
	property.ProductID = product.ProductID
	property.PropertyName = d.Get("name").(string)

	err = property.Save()
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Property created: %s\n", property.PropertyID)
	return property, nil
}
