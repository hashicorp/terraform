package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/devtestlabs/mgmt/2016-05-15/dtl"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/validate"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func SchemaDevTestVirtualMachineInboundNatRule() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		// since these aren't returned from the API
		ForceNew: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"protocol": {
					Type:     schema.TypeString,
					Required: true,
					ValidateFunc: validation.StringInSlice([]string{
						string(dtl.TCP),
						string(dtl.UDP),
					}, false),
				},

				"backend_port": {
					Type:         schema.TypeInt,
					Required:     true,
					ForceNew:     true,
					ValidateFunc: validate.PortNumber,
				},

				"frontend_port": {
					Type:     schema.TypeInt,
					Computed: true,
				},
			},
		},
	}
}

func ExpandDevTestLabVirtualMachineNatRules(input *schema.Set) []dtl.InboundNatRule {
	rules := make([]dtl.InboundNatRule, 0)
	if input == nil {
		return rules
	}

	for _, val := range input.List() {
		v := val.(map[string]interface{})
		backendPort := v["backend_port"].(int)
		protocol := v["protocol"].(string)

		rule := dtl.InboundNatRule{
			TransportProtocol: dtl.TransportProtocol(protocol),
			BackendPort:       utils.Int32(int32(backendPort)),
		}

		rules = append(rules, rule)
	}

	return rules
}

func ExpandDevTestLabVirtualMachineGalleryImageReference(input []interface{}, osType string) *dtl.GalleryImageReference {
	if len(input) == 0 {
		return nil
	}

	v := input[0].(map[string]interface{})
	offer := v["offer"].(string)
	publisher := v["publisher"].(string)
	sku := v["sku"].(string)
	version := v["version"].(string)

	return &dtl.GalleryImageReference{
		Offer:     utils.String(offer),
		OsType:    utils.String(osType),
		Publisher: utils.String(publisher),
		Sku:       utils.String(sku),
		Version:   utils.String(version),
	}
}

func SchemaDevTestVirtualMachineGalleryImageReference() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"offer": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"publisher": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"sku": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
				"version": {
					Type:     schema.TypeString,
					Required: true,
					ForceNew: true,
				},
			},
		},
	}
}

func FlattenDevTestVirtualMachineGalleryImage(input *dtl.GalleryImageReference) []interface{} {
	results := make([]interface{}, 0)

	if input != nil {
		output := make(map[string]interface{})

		if input.Offer != nil {
			output["offer"] = *input.Offer
		}

		if input.Publisher != nil {
			output["publisher"] = *input.Publisher
		}

		if input.Sku != nil {
			output["sku"] = *input.Sku
		}

		if input.Version != nil {
			output["version"] = *input.Version
		}

		results = append(results, output)
	}

	return results
}
