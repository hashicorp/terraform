package azurerm

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

func resourceArmLoadBalancer() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmLoadBalancerCreate,
		Read:   resourceArmLoadBalancerRead,
		Update: resourceArmLoadBalancerUpdate,
		Delete: resourceArmLoadBalancerDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				StateFunc: azureRMNormalizeLocation,
			},
			"frontend_ip_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"frontend_ip_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"frontend_ip_subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"frontend_ip_private_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"frontend_ip_public_ip_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"frontend_ip_private_ip_allocation": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceArmLoadBalancerCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmLoadBalancer] resourceArmLoadBalancerCreate[enter]")
	defer log.Printf("[resourceArmLoadBalancer] resourceArmLoadBalancerCreate[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	// first; fetch a bunch of fields:
	typ := d.Get("type").(string)
	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGrp := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	ipConfs, err := expandArmFrontendIp(d)
	ipProps := network.LoadBalancerPropertiesFormat{FrontendIPConfigurations: ipConfs}

	if err != nil {
		return err
	}

	loadBalancer := network.LoadBalancer{
		Name:       &name,
		Type:       &typ,
		Location:   &location,
		Tags:       expandTags(tags),
		Properties: &ipProps,
	}

	resp, err := lbClient.CreateOrUpdate(resGrp, name, loadBalancer)
	if err != nil {
		log.Printf("[resourceArmLoadBalancer] ERROR LB got status %s", err.Error())
		return fmt.Errorf("Error issuing Azure ARM creation request for load balancer '%s': %s", name, err)
	}

	return ResourceArmLoadBalancerSetFromResponse(d, &resp)
}

func resourceArmLoadBalancerUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmSimpleLb] resourceArmLoadBalancerUpdate[enter]")
	defer log.Printf("[resourceArmSimpleLb] resourceArmLoadBalancerUpdate[exit]")
	return resourceArmLoadBalancerCreate(d, meta)
}

func resourceArmLoadBalancerDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmLoadBalancer] resourceArmLoadBalancerDelete[enter]")
	defer log.Printf("[resourceArmLoadBalancer] resourceArmLoadBalancerDelete[exit]")

	lbClient := meta.(*ArmClient).loadBalancerClient

	name := d.Get("name").(string)
	resGroup := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Issuing deletion request to Azure ARM for load balancer '%s'.", name)

	resp, err := lbClient.Delete(resGroup, name)
	if err != nil {
		return fmt.Errorf("Error issuing Azure ARM delete request for load balancer '%s': %s", name, err)
	}

	log.Printf("[resourceArmLoadBalancer] delete response %d %s", resp.StatusCode, resp.Status)

	return nil
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func ResourceArmLoadBalancerSetFromResponse(d *schema.ResourceData, loadBalancer *network.LoadBalancer) error {
	log.Printf("[resourceArmLoadBalancer] iResourceArmLoadBalancerRead[enter]")
	defer log.Printf("[resourceArmLoadBalancer] iResourceArmLoadBalancerRead[exit]")

	d.Set("location", loadBalancer.Location)
	d.Set("type", loadBalancer.Type)
	flattenAndSetTags(d, loadBalancer.Tags)
	err := flattenArmFrontendIps(d, loadBalancer.Properties.FrontendIPConfigurations)
	if err != nil {
		return err
	}
	d.SetId(*loadBalancer.ID)

	return nil
}

// resourceArmLoadBalancerRead goes ahead and reads the state of the corresponding ARM load balancer.
func resourceArmLoadBalancerRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[resourceArmLoadBalancer] resourceArmSimpleLbRead[enter]")
	defer log.Printf("[resourceArmLoadBalancer] resourceArmSimpleLbRead[exit]")
	lbClient := meta.(*ArmClient).loadBalancerClient

	name := d.Get("name").(string)
	resGrp := d.Get("resource_group_name").(string)

	log.Printf("[INFO] Issuing read request of load balancer '%s' off Azure.", name)

	loadBalancer, err := lbClient.Get(resGrp, name, "")
	if err != nil {
		return fmt.Errorf("Error reading the state of the load balancer off Azure: %s", err)
	}
	return ResourceArmLoadBalancerSetFromResponse(d, &loadBalancer)
}

func expandArmFrontendIp(d *schema.ResourceData) (*[]network.FrontendIPConfiguration, error) {
	frontendProperties := network.FrontendIPConfigurationPropertiesFormat{}
	frontendName := d.Get("frontend_ip_name").(string)
	subnetId := d.Get("frontend_ip_subnet_id").(string)
	publicIpId := d.Get("frontend_ip_public_ip_id").(string)
	privateIpAddress := d.Get("frontend_ip_private_ip_address").(string)
	privateIpAllocation := d.Get("frontend_ip_private_ip_address").(string)

	if subnetId != "" && publicIpId != "" {
		return nil, fmt.Errorf("The frontend IP configuration %s cannot use both a subnet and a public IP address", frontendName)
	}

	if subnetId != "" {
		frontendProperties.Subnet = &network.Subnet{ID: &subnetId}
	}
	if publicIpId != "" {
		frontendProperties.PublicIPAddress = &network.PublicIPAddress{ID: &publicIpId}
	}
	if privateIpAddress != "" {
		frontendProperties.PrivateIPAddress = &privateIpAddress
	}
	if privateIpAllocation != "" {
		frontendProperties.PrivateIPAllocationMethod = network.IPAllocationMethod(privateIpAllocation)
	}

	frontendIpConf := network.FrontendIPConfiguration{
		Name:       &frontendName,
		Properties: &frontendProperties,
	}

	return &[]network.FrontendIPConfiguration{frontendIpConf}, nil
}

func flattenArmFrontendIps(d *schema.ResourceData, frontendIps *[]network.FrontendIPConfiguration) error {

	if len(*frontendIps) != 1 {
		return fmt.Errorf("The load balancer must have exactly 1 frontend IP")
	}

	fIpConf := (*frontendIps)[0]
	if fIpConf.Properties.Subnet != nil && fIpConf.Properties.Subnet.ID != nil {
		d.Set("frontend_ip_subnet_id", *fIpConf.Properties.Subnet.ID)
	}
	if fIpConf.Properties.PublicIPAddress != nil && fIpConf.Properties.PublicIPAddress.ID != nil {
		d.Set("frontend_ip_public_ip_id", *fIpConf.Properties.PublicIPAddress.ID)
	}
	if fIpConf.Properties.PrivateIPAddress != nil {
		d.Set("frontend_ip_private_ip_address", *fIpConf.Properties.PrivateIPAddress)
	}

	d.Set("frontend_ip_name", *fIpConf.Name)
	d.Set("frontend_ip_id", *fIpConf.ID)
	d.Set("frontend_ip_private_ip_allocation", string(fIpConf.Properties.PrivateIPAllocationMethod))

	return nil
}
