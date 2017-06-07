package azurerm

import (
	"bytes"
	"log"
	"strings"

	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceArmExpressRouteCircuit() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmExpressRouteCircuitCreateOrUpdate,
		Read:   resourceArmExpressRouteCircuitRead,
		Update: resourceArmExpressRouteCircuitCreateOrUpdate,
		Delete: resourceArmExpressRouteCircuitDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"service_provider_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
			},

			"peering_location": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
			},

			"bandwidth_in_mbps": {
				Type:     schema.TypeInt,
				Required: true,
			},

			"sku": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"tier": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.ExpressRouteCircuitSkuTierStandard),
								string(network.ExpressRouteCircuitSkuTierPremium),
							}, true),
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},

						"family": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								string(network.MeteredData),
								string(network.UnlimitedData),
							}, true),
							DiffSuppressFunc: ignoreCaseDiffSuppressFunc,
						},
					},
				},
				Set: resourceArmExpressRouteCircuitSkuHash,
			},

			"allow_classic_operations": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"service_provider_provisioning_state": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"service_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmExpressRouteCircuitCreateOrUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	ercClient := client.expressRouteCircuitClient

	log.Printf("[INFO] preparing arguments for Azure ARM ExpressRouteCircuit creation.")

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)
	location := d.Get("location").(string)
	serviceProviderName := d.Get("service_provider_name").(string)
	peeringLocation := d.Get("peering_location").(string)
	bandwidthInMbps := int32(d.Get("bandwidth_in_mbps").(int))
	sku := expandExpressRouteCircuitSku(d)
	allowRdfeOps := d.Get("allow_classic_operations").(bool)
	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	erc := network.ExpressRouteCircuit{
		Name:     &name,
		Location: &location,
		Sku:      sku,
		ExpressRouteCircuitPropertiesFormat: &network.ExpressRouteCircuitPropertiesFormat{
			AllowClassicOperations: &allowRdfeOps,
			ServiceProviderProperties: &network.ExpressRouteCircuitServiceProviderProperties{
				ServiceProviderName: &serviceProviderName,
				PeeringLocation:     &peeringLocation,
				BandwidthInMbps:     &bandwidthInMbps,
			},
		},
		Tags: expandedTags,
	}

	_, error := ercClient.CreateOrUpdate(resGroup, name, erc, make(chan struct{}))
	err := <-error
	if err != nil {
		return errwrap.Wrapf("Error Creating/Updating ExpressRouteCircuit {{err}}", err)
	}

	read, err := ercClient.Get(resGroup, name)
	if err != nil {
		return errwrap.Wrapf("Error Getting ExpressRouteCircuit {{err}}", err)
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read ExpressRouteCircuit %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmExpressRouteCircuitRead(d, meta)
}

func resourceArmExpressRouteCircuitRead(d *schema.ResourceData, meta interface{}) error {
	erc, resGroup, err := retrieveErcByResourceId(d.Id(), meta)
	if err != nil {
		return err
	}

	if erc == nil {
		d.SetId("")
		log.Printf("[INFO] Express Route Circuit %q not found. Removing from state", d.Get("name").(string))
		return nil
	}

	d.Set("name", erc.Name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", erc.Location)

	if erc.ServiceProviderProperties != nil {
		d.Set("service_provider_name", erc.ServiceProviderProperties.ServiceProviderName)
		d.Set("peering_location", erc.ServiceProviderProperties.PeeringLocation)
		d.Set("bandwidth_in_mbps", erc.ServiceProviderProperties.BandwidthInMbps)
	}

	if erc.Sku != nil {
		d.Set("sku", schema.NewSet(resourceArmExpressRouteCircuitSkuHash, flattenExpressRouteCircuitSku(erc.Sku)))
	}

	d.Set("service_provider_provisioning_state", string(erc.ServiceProviderProvisioningState))
	d.Set("service_key", erc.ServiceKey)
	d.Set("allow_classic_operations", erc.AllowClassicOperations)

	flattenAndSetTags(d, erc.Tags)

	return nil
}

func resourceArmExpressRouteCircuitDelete(d *schema.ResourceData, meta interface{}) error {
	ercClient := meta.(*ArmClient).expressRouteCircuitClient

	resGroup, name, err := extractResourceGroupAndErcName(d.Id())
	if err != nil {
		return errwrap.Wrapf("Error Parsing Azure Resource ID {{err}}", err)
	}

	_, error := ercClient.Delete(resGroup, name, make(chan struct{}))
	err = <-error
	return err
}

func expandExpressRouteCircuitSku(d *schema.ResourceData) *network.ExpressRouteCircuitSku {
	skuSettings := d.Get("sku").(*schema.Set)
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
