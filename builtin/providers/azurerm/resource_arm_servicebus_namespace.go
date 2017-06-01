package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/servicebus"
	"github.com/hashicorp/terraform/helper/schema"
)

// Default Authorization Rule/Policy created by Azure, used to populate the
// default connection strings and keys
var serviceBusNamespaceDefaultAuthorizationRule = "RootManageSharedAccessKey"

func resourceArmServiceBusNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmServiceBusNamespaceCreate,
		Read:   resourceArmServiceBusNamespaceRead,
		Update: resourceArmServiceBusNamespaceCreate,
		Delete: resourceArmServiceBusNamespaceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"sku": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateFunc:     validateServiceBusNamespaceSku,
				DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
			},

			"capacity": {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Default:      1,
				ValidateFunc: validateServiceBusNamespaceCapacity,
			},

			"default_primary_connection_string": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_secondary_connection_string": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_primary_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"default_secondary_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmServiceBusNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	namespaceClient := client.serviceBusNamespacesClient
	log.Printf("[INFO] preparing arguments for Azure ARM ServiceBus Namespace creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	sku := d.Get("sku").(string)
	capacity := int32(d.Get("capacity").(int))
	tags := d.Get("tags").(map[string]interface{})

	parameters := servicebus.NamespaceCreateOrUpdateParameters{
		Location: &location,
		Sku: &servicebus.Sku{
			Name:     servicebus.SkuName(sku),
			Tier:     servicebus.SkuTier(sku),
			Capacity: &capacity,
		},
		Tags: expandTags(tags),
	}

	_, error := namespaceClient.CreateOrUpdate(resGroup, name, parameters, make(chan struct{}))
	err := <-error
	if err != nil {
		return err
	}

	read, err := namespaceClient.Get(resGroup, name)
	if err != nil {
		return err
	}

	if read.ID == nil {
		return fmt.Errorf("Cannot read ServiceBus Namespace %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmServiceBusNamespaceRead(d, meta)
}

func resourceArmServiceBusNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	namespaceClient := meta.(*ArmClient).serviceBusNamespacesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["namespaces"]

	resp, err := namespaceClient.Get(resGroup, name)
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure ServiceBus Namespace %s: %+v", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	d.Set("name", resp.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", azureRMNormalizeLocation(*resp.Location))
	d.Set("sku", strings.ToLower(string(resp.Sku.Name)))
	d.Set("capacity", resp.Sku.Capacity)

	keys, err := namespaceClient.ListKeys(resGroup, name, serviceBusNamespaceDefaultAuthorizationRule)
	if err != nil {
		log.Printf("[ERROR] Unable to List default keys for Namespace %s: %+v", name, err)
	} else {
		d.Set("default_primary_connection_string", keys.PrimaryConnectionString)
		d.Set("default_secondary_connection_string", keys.SecondaryConnectionString)
		d.Set("default_primary_key", keys.PrimaryKey)
		d.Set("default_secondary_key", keys.SecondaryKey)
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmServiceBusNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	namespaceClient := meta.(*ArmClient).serviceBusNamespacesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["namespaces"]

	deleteResp, error := namespaceClient.Delete(resGroup, name, make(chan struct{}))
	resp := <-deleteResp
	err = <-error

	if resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("Error issuing Azure ARM delete request of ServiceBus Namespace'%s': %+v", name, err)
	}

	return nil
}

func validateServiceBusNamespaceSku(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	skus := map[string]bool{
		"basic":    true,
		"standard": true,
		"premium":  true,
	}

	if !skus[value] {
		errors = append(errors, fmt.Errorf("ServiceBus Namespace SKU can only be Basic, Standard or Premium"))
	}
	return
}

func validateServiceBusNamespaceCapacity(v interface{}, k string) (ws []string, errors []error) {
	value := v.(int)
	capacities := map[int]bool{
		1: true,
		2: true,
		4: true,
	}

	if !capacities[value] {
		errors = append(errors, fmt.Errorf("ServiceBus Namespace Capacity can only be 1, 2 or 4"))
	}
	return
}
