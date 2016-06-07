package softlayer

import (
	"fmt"
	"log"
	"strconv"

	softlayer "github.com/TheWeatherCompany/softlayer-go/softlayer"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	NETSCALER_VPX_TYPE = "Netscaler VPX"
)

func resourceSoftLayerNetworkApplicationDeliveryController() *schema.Resource {
	return &schema.Resource{
		Create: resourceSoftLayerNetworkApplicationDeliveryControllerCreate,
		Read:   resourceSoftLayerNetworkApplicationDeliveryControllerRead,
		Delete: resourceSoftLayerNetworkApplicationDeliveryControllerDelete,
		Exists: resourceSoftLayerNetworkApplicationDeliveryControllerExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"datacenter": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"speed": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"version": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"plan": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_count": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceSoftLayerNetworkApplicationDeliveryControllerCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).networkApplicationDeliveryControllerService
	if client == nil {
		return fmt.Errorf("The client is nil.")
	}

	nadcType := NETSCALER_VPX_TYPE

	switch nadcType {
	default:
		return fmt.Errorf("[ERROR] Network application delivery controller type %s is not supported", nadcType)
	case NETSCALER_VPX_TYPE:
		// create Netscaler VPX
		opts := softlayer.NetworkApplicationDeliveryControllerCreateOptions{
			Speed:    d.Get("speed").(int),
			Version:  d.Get("version").(string),
			Plan:     d.Get("plan").(string),
			IpCount:  d.Get("ip_count").(int),
			Location: d.Get("datacenter").(string),
		}

		log.Printf("[INFO] Creating network application delivery controller")

		netscalerVPX, err := client.CreateNetscalerVPX(&opts)

		if err != nil {
			return fmt.Errorf("Error creating network application delivery controller: %s", err)
		}

		d.SetId(fmt.Sprintf("%d", netscalerVPX.Id))

		log.Printf("[INFO] Netscaler VPX ID: %s", d.Id())
	}

	return resourceSoftLayerNetworkApplicationDeliveryControllerRead(d, meta)
}

func resourceSoftLayerNetworkApplicationDeliveryControllerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).networkApplicationDeliveryControllerService
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}
	getObjectResult, err := client.GetObject(id)
	if err != nil {
		return fmt.Errorf("Error retrieving network application delivery controller: %s", err)
	}

	d.Set("name", getObjectResult.Name)
	d.Set("type", getObjectResult.Type)
	if getObjectResult.Datacenter != nil {
		d.Set("location", getObjectResult.Datacenter.Name)
	}

	return nil
}

func resourceSoftLayerNetworkApplicationDeliveryControllerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).networkApplicationDeliveryControllerService
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	_, err = client.DeleteObject(id)

	if err != nil {
		return fmt.Errorf("Error deleting network application delivery controller: %s", err)
	}

	return nil
}

func resourceSoftLayerNetworkApplicationDeliveryControllerExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*Client).networkApplicationDeliveryControllerService
	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	nadc, err := client.GetObject(id)

	if err != nil {
		return false, fmt.Errorf("Error fetching network application delivery controller: %s", err)
	}

	return nadc.Id == id && err == nil, nil
}
