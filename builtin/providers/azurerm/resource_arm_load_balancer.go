package azurerm

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/Azure/azure-sdk-for-go/core/http"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmLoadBalancer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerCreate,
		Read:   resourceArmLoadBalancerRead,
		Update: resourceArmLoadBalancerCreate,
		Delete: resourceArmLoadBalancerDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateArmLoadBalancerType,
			},

			"location": {
				Type:      schema.TypeString,
				Optional:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"frontend_ip_configuration": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"private_ip_allocation_method": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"private_ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"subnet": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},

				Set: resourceArmLoadBalancerFrontEndIpConfigurationHash,
			},

			"backend_address_pool": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceArmLoadBalancerBackendAddressPoolHash,
			},

			"load_balancing_rule": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"frontend_ip_configuration": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"backend_address_pool": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"probe": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"frontend_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"backend_port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"idle_timeout_in_minutes": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceArmLoadBalancerLoadBalancingRuleHash,
			},

			"probe": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"number_of_probes": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
						"interval_in_seconds": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceArmLoadBalancerProbeHash,
			},

			"tags": tagsSchema(),
		},
	}
}

func validateArmLoadBalancerType(v interface{}, k string) (ws []string, es []error) {
	value := v.(string)

	if !strings.EqualFold(value, "internal") && !strings.EqualFold(value, "public") {
		es = append(es, fmt.Errorf("%q must be either Internal or Public", k))
	}

	return
}

/*

Example:

  resource "azurerm_resource_group" "test" {
      name = "acctestrg-%d"
      location = "West US"

      tags {
    		environment = "Production"
    		cost_center = "MSFT"
      }
  }

  resource "azurerm_load_balancer" "test" {
    name = "examplelb"
    type = "internal"
    location = "${azurerm_resource_group.test.location}"
    resource_group_name = "${azurerm_resource_group.test.name}"

    frontend_ip_configuration {
      name = "examplelbfront"
      private_ip_allocation_method = "static"
      private_ip_address = "10.123.234.156"
      subnet = "subnetid"
    }

    backend_address_pool {
      name = "examplebackend"
    }

    load_balancing_rule {
      name = "examplelbrule"
      frontend_ip_configuration = "examplelbfront"
      backend_address_pool = "examplebackend"
      probe = "examplelbprobe"
      protocol = "Tcp"
      frontend_port = 80
      backend_port = 80
      idle_timeout_in_minutes = 15
    }

    probe {
      name = "examplelbprobe"
      protocol = "Tcp"
      port = 80
      number_of_probes = 2
      interval_in_seconds = 15
    }

    tags {
      environment = "Production"
      cost_center = "MSFT"
    }
  }


*/

/*
type LoadBalancer struct {
	autorest.Response `json:"-"`
	ID                *string                       `json:"id,omitempty"`
	Name              *string                       `json:"name,omitempty"`
	Type              *string                       `json:"type,omitempty"`
	Location          *string                       `json:"location,omitempty"`
	Tags              *map[string]*string           `json:"tags,omitempty"`
	Properties        *LoadBalancerPropertiesFormat `json:"properties,omitempty"`
}

type LoadBalancerPropertiesFormat struct {
	FrontendIPConfigurations *[]FrontendIPConfiguration `json:"frontendIPConfigurations,omitempty"`
	BackendAddressPools      *[]BackendAddressPool      `json:"backendAddressPools,omitempty"`
	LoadBalancingRules       *[]LoadBalancingRule       `json:"loadBalancingRules,omitempty"`
	Probes                   *[]Probe                   `json:"probes,omitempty"`
	InboundNatRules          *[]InboundNatRule          `json:"inboundNatRules,omitempty"`
	InboundNatPools          *[]InboundNatPool          `json:"inboundNatPools,omitempty"`
	OutboundNatRules         *[]OutboundNatRule         `json:"outboundNatRules,omitempty"`
}

type FrontendIPConfiguration struct {
	ID         *string                                  `json:"id,omitempty"`
	Properties *FrontendIPConfigurationPropertiesFormat `json:"properties,omitempty"`
	Name       *string                                  `json:"name,omitempty"`
	Etag       *string                                  `json:"etag,omitempty"`
}

type FrontendIPConfigurationPropertiesFormat struct {
	InboundNatRules           *[]SubResource     `json:"inboundNatRules,omitempty"`
	InboundNatPools           *[]SubResource     `json:"inboundNatPools,omitempty"`
	OutboundNatRules          *[]SubResource     `json:"outboundNatRules,omitempty"`
	LoadBalancingRules        *[]SubResource     `json:"loadBalancingRules,omitempty"`
	PrivateIPAddress          *string            `json:"privateIPAddress,omitempty"`
	PrivateIPAllocationMethod IPAllocationMethod `json:"privateIPAllocationMethod,omitempty"`
	Subnet                    *Subnet            `json:"subnet,omitempty"`
	PublicIPAddress           *PublicIPAddress   `json:"publicIPAddress,omitempty"`
	ProvisioningState         *string            `json:"provisioningState,omitempty"`
}

type InboundNatPool struct {
	ID         *string                         `json:"id,omitempty"`
	Properties *InboundNatPoolPropertiesFormat `json:"properties,omitempty"`
	Name       *string                         `json:"name,omitempty"`
	Etag       *string                         `json:"etag,omitempty"`
}

type InboundNatPoolPropertiesFormat struct {
	FrontendIPConfiguration *SubResource      `json:"frontendIPConfiguration,omitempty"`
	Protocol                TransportProtocol `json:"protocol,omitempty"`
	FrontendPortRangeStart  *int32            `json:"frontendPortRangeStart,omitempty"`
	FrontendPortRangeEnd    *int32            `json:"frontendPortRangeEnd,omitempty"`
	BackendPort             *int32            `json:"backendPort,omitempty"`
	ProvisioningState       *string           `json:"provisioningState,omitempty"`
}

type InboundNatRule struct {
	ID         *string                         `json:"id,omitempty"`
	Properties *InboundNatRulePropertiesFormat `json:"properties,omitempty"`
	Name       *string                         `json:"name,omitempty"`
	Etag       *string                         `json:"etag,omitempty"`
}

type InboundNatRulePropertiesFormat struct {
	FrontendIPConfiguration *SubResource              `json:"frontendIPConfiguration,omitempty"`
	BackendIPConfiguration  *InterfaceIPConfiguration `json:"backendIPConfiguration,omitempty"`
	Protocol                TransportProtocol         `json:"protocol,omitempty"`
	FrontendPort            *int32                    `json:"frontendPort,omitempty"`
	BackendPort             *int32                    `json:"backendPort,omitempty"`
	IdleTimeoutInMinutes    *int32                    `json:"idleTimeoutInMinutes,omitempty"`
	EnableFloatingIP        *bool                     `json:"enableFloatingIP,omitempty"`
	ProvisioningState       *string                   `json:"provisioningState,omitempty"`
}
*/

func resourceArmLoadBalancerCreate(d *schema.ResourceData, meta interface{}) error {
	lbClient := meta.(*ArmClient).loadBalancerClient

	name := d.Get("name").(string)
	lbType := d.Get("type").(string)
	location := d.Get("location").(string)
	tags := d.Get("tags").(map[string]interface{})

	// TODO: Parse the following:
	//  frontendIPConfigurations out to a []FrontendIPConfiguration
	//  backendAddressPool out to a []BackendAddressPool
	//  loadBalancingRules out to a []LoadBalancingRules
	//  probes out to a []Probe
	//  inboundNatRules out to a []InboundNatRule
	//  inboundNatPools out to a []InboundNatPool
	//  outboundNatRules out to a []OutboundNatRules

	loadBalancer := network.LoadBalancer{
		Name:     &name,
		Type:     &lbType,
		Location: &location,
		Properties: LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: &frontendIPConfigurations,
			BackendAddressPools:      &backendAddressPool,
			LoadBalancingRules:       &loadBalancingRules,
			Probes:                   &probes,
			InboundNatRules:          &inboundNatRules,
			InboundNatPools:          &inboundNatPools,
		},
		Tags: expandTags(tags),
	}

	_, err := lbClient.CreateOrUpdate(resGroup, name, gateway, make(chan struct{}))
	if err != nil {
		return fmt.Errorf("Error creating Azure ARM Load Balancer '%s': %s", name, err)
	}

	read, err := lbClient.Get(resGroup, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Virtual Network %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmLoadBalancerRead(d, meta)
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func resourceArmLoadBalancerRead(d *schema.ResourceData, meta interface{}) error {
}

// resourceArmLoadBalancerDelete deletes the specified ARM load balancer.
func resourceArmLoadBalancerDelete(d *schema.ResourceData, meta interface{}) error {
}

// Helpers
func resourceArmLoadBalancerBackendAddressPoolHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))

	return hashcode.String(buf.String())
}

func resourceArmLoadBalancerFrontEndIpConfigurationHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["private_ip_allocation_method"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["private_ip_address"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["subnet"].(string)))

	return hashcode.String(buf.String())
}

func resourceArmLoadBalancerLoadBalancingRuleHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["frontend_ip_configuration"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["backend_address_pool"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["probe"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["frontend_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["backend_port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["idle_timeout_in_minutes"].(int)))

	return hashcode.String(buf.String())
}

func resourceArmLoadBalancerProbeHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["protocol"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["number_of_probes"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["interval_in_seconds"].(int)))

	return hashcode.String(buf.String())
}
