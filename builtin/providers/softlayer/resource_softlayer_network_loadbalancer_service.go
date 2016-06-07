package softlayer

import (
	"fmt"
	"log"

	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
	"github.com/TheWeatherCompany/softlayer-go/services"
	"github.com/hashicorp/terraform/helper/schema"
	"strconv"
	"strings"
)

func resourceSoftLayerNetworkLoadBalancerService() *schema.Resource {
	return &schema.Resource{
		Create: resourceSoftLayerNetworkLoadBalancerServiceCreate,
		Read:   resourceSoftLayerNetworkLoadBalancerServiceRead,
		Update: resourceSoftLayerNetworkLoadBalancerServiceUpdate,
		Delete: resourceSoftLayerNetworkLoadBalancerServiceDelete,
		Exists: resourceSoftLayerNetworkLoadBalancerServiceExists,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},

			"vip_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"destination_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"destination_port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"weight": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"connection_limit": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"health_check": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func parseVipUniqueId(vipUniqueId string) (vipId string, nacdId int, err error) {
	nacdId, err = strconv.Atoi(strings.Split(vipUniqueId, services.ID_DELIMITER)[1])
	vipId = strings.Split(vipUniqueId, services.ID_DELIMITER)[0]

	if err != nil {
		return "", -1, fmt.Errorf("Error parsing vip id: %s", err)
	}

	return vipId, nacdId, nil
}

func resourceSoftLayerNetworkLoadBalancerServiceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).networkApplicationDeliveryControllerService

	if client == nil {
		return fmt.Errorf("The client is nil.")
	}

	vipUniqueId := d.Get("vip_id").(string)

	vipId, nacdId, err := parseVipUniqueId(vipUniqueId)

	if err != nil {
		return fmt.Errorf("Error parsing vip id: %s", err)
	}

	template := []datatypes.SoftLayer_Network_LoadBalancer_Service_Template{datatypes.SoftLayer_Network_LoadBalancer_Service_Template{
		Name:                 d.Get("name").(string),
		DestinationIpAddress: d.Get("destination_ip_address").(string),
		DestinationPort:      d.Get("destination_port").(int),
		Weight:               d.Get("weight").(int),
		HealthCheck:          d.Get("health_check").(string),
		ConnectionLimit:      d.Get("connection_limit").(int),
	}}

	log.Printf("[INFO] Creating LoadBalancer Service %s", template[0].Name)

	successFlag, err := client.CreateLoadBalancerService(vipId, nacdId, template)

	if err != nil {
		return fmt.Errorf("Error creating LoadBalancer Service: %s", err)
	}

	if !successFlag {
		return fmt.Errorf("Error creating LoadBalancer Service")
	}

	return resourceSoftLayerNetworkLoadBalancerServiceRead(d, meta)
}

func resourceSoftLayerNetworkLoadBalancerServiceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).networkApplicationDeliveryControllerService
	if client == nil {
		return fmt.Errorf("The client is nil.")
	}

	vipUniqueId := d.Get("vip_id").(string)

	vipId, nadcId, err := parseVipUniqueId(vipUniqueId)

	if err != nil {
		return fmt.Errorf("Error parsing vip id: %s", err)
	}

	service, err := client.GetLoadBalancerService(nadcId, vipId, d.Get("name").(string))

	if err != nil {
		return fmt.Errorf("Unable to get LoadBalancerService: %s", err)
	}

	d.SetId(service.Name)
	d.Set("name", service.Name)
	d.Set("destination_ip_address", service.DestinationIpAddress)
	d.Set("destination_port", service.DestinationPort)
	d.Set("weight", service.Weight)
	d.Set("health_check", service.HealthCheck)
	d.Set("connection_limit", service.ConnectionLimit)

	return nil
}

func resourceSoftLayerNetworkLoadBalancerServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).networkApplicationDeliveryControllerService

	if client == nil {
		return fmt.Errorf("The client was nil.")
	}

	vipUniqueId := d.Get("vip_id").(string)

	vipId, nadcId, err := parseVipUniqueId(vipUniqueId)

	if err != nil {
		return fmt.Errorf("Error parsing vip id: %s", err)
	}

	_, err = client.GetLoadBalancerService(nadcId, vipId, d.Get("name").(string))

	if err != nil {
		return fmt.Errorf("Error retrieving LoadBalancer Service: %s", err)
	}

	var serviceTemplate datatypes.SoftLayer_Network_LoadBalancer_Service_Template

	if data, ok := d.GetOk("name"); ok {
		serviceTemplate.Name = data.(string)
	}
	if data, ok := d.GetOk("destination_ip_address"); ok {
		serviceTemplate.DestinationIpAddress = data.(string)
	}
	if data, ok := d.GetOk("destination_port"); ok {
		serviceTemplate.DestinationPort = data.(int)
	}
	if data, ok := d.GetOk("weight"); ok {
		serviceTemplate.Weight = data.(int)
	}
	if data, ok := d.GetOk("health_check"); ok {
		serviceTemplate.HealthCheck = data.(string)
	}
	if data, ok := d.GetOk("connection_limit"); ok {
		serviceTemplate.ConnectionLimit = data.(int)
	}

	_, err = client.CreateLoadBalancerService(vipId, nadcId, []datatypes.SoftLayer_Network_LoadBalancer_Service_Template{serviceTemplate})
	if err != nil {
		return fmt.Errorf("Error editing LoadBalancer Service: %s", err)
	}

	return nil
}

func resourceSoftLayerNetworkLoadBalancerServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).networkApplicationDeliveryControllerService
	if client == nil {
		return fmt.Errorf("The client is nil.")
	}

	vipUniqueId := d.Get("vip_id").(string)

	vipId, nadcId, err := parseVipUniqueId(vipUniqueId)

	if err != nil {
		return fmt.Errorf("Error parsing vip id: %s", err)
	}

	serviceId := d.Get("name").(string)

	_, err = client.DeleteLoadBalancerService(nadcId, vipId, serviceId)
	if err != nil {
		return fmt.Errorf("Error deleting Load Balancer Service %s: %s", serviceId, err)
	}

	return nil
}

func resourceSoftLayerNetworkLoadBalancerServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*Client).networkApplicationDeliveryControllerService
	if client == nil {
		return false, fmt.Errorf("The client is nil.")
	}

	vipUniqueId := d.Get("vip_id").(string)

	vipId, nadcId, err := parseVipUniqueId(vipUniqueId)

	if err != nil {
		return false, fmt.Errorf("Error parsing vip id: %s", err)
	}

	serviceId := d.Get("name").(string)

	service, err := client.GetLoadBalancerService(nadcId, vipId, serviceId)

	if err != nil {
		return false, fmt.Errorf("Error fetching Load Balancer Service: %s", err)
	}

	return service.Name == serviceId && err == nil, nil
}
