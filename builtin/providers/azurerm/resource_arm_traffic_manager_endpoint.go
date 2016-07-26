package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/Azure/azure-sdk-for-go/arm/trafficmanager"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmTrafficManagerEndpoint() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmTrafficManagerEndpointCreate,
		Read:   resourceArmTrafficManagerEndpointRead,
		Update: resourceArmTrafficManagerEndpointCreate,
		Delete: resourceArmTrafficManagerEndpointDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

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
				ValidateFunc: validateAzureRMTrafficManagerEndpointType,
			},

			"profile_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"target": {
				Type:     schema.TypeString,
				Optional: true,
				// when targeting an Azure resource the FQDN of that resource will be set as the target
				Computed: true,
			},

			"target_resource_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"endpoint_status": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"weight": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateAzureRMTrafficManagerEndpointWeight,
			},

			"priority": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateAzureRMTrafficManagerEndpointPriority,
			},

			"endpoint_location": {
				Type:     schema.TypeString,
				Optional: true,
				// when targeting an Azure resource the location of that resource will be set on the endpoint
				Computed:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"min_child_endpoints": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceArmTrafficManagerEndpointCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).trafficManagerEndpointsClient

	log.Printf("[INFO] preparing arguments for ARM TrafficManager Endpoint creation.")

	name := d.Get("name").(string)
	endpointType := d.Get("type").(string)
	profileName := d.Get("profile_name").(string)
	resGroup := d.Get("resource_group_name").(string)

	params := trafficmanager.Endpoint{
		Name:       &name,
		Type:       &endpointType,
		Properties: getArmTrafficManagerEndpointProperties(d),
	}

	_, err := client.CreateOrUpdate(resGroup, profileName, path.Base(endpointType), name, params)
	if err != nil {
		return err
	}

	read, err := client.Get(resGroup, profileName, path.Base(endpointType), name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read TrafficManager endpoint %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmTrafficManagerEndpointRead(d, meta)
}

func resourceArmTrafficManagerEndpointRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).trafficManagerEndpointsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	endpointType := d.Get("type").(string)
	profileName := id.Path["trafficManagerProfiles"]

	// endpoint name is keyed by endpoint type in ARM ID
	name := id.Path[path.Base(endpointType)]

	resp, err := client.Get(resGroup, profileName, path.Base(endpointType), name)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on TrafficManager Endpoint %s: %s", name, err)
	}
	endpoint := *resp.Properties

	d.Set("name", resp.Name)
	d.Set("endpoint_status", *endpoint.EndpointStatus)

	if endpoint.TargetResourceID != nil {
		d.Set("target_resource_id", *endpoint.TargetResourceID)
	}

	if endpoint.Target != nil {
		d.Set("target", *endpoint.Target)
	}

	if endpoint.Weight != nil {
		d.Set("weight", int(*endpoint.Weight))
	}

	if endpoint.Priority != nil {
		d.Set("priority", int(*endpoint.Priority))
	}

	if endpoint.EndpointLocation != nil {
		d.Set("endpoint_location", *endpoint.EndpointLocation)
	}

	if endpoint.EndpointMonitorStatus != nil {
		d.Set("endpoint_monitor_status", *endpoint.EndpointMonitorStatus)
	}

	if endpoint.MinChildEndpoints != nil {
		d.Set("min_child_endpoints", *endpoint.MinChildEndpoints)
	}

	return nil
}

func resourceArmTrafficManagerEndpointDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).trafficManagerEndpointsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	endpointType := d.Get("type").(string)
	profileName := id.Path["trafficManagerProfiles"]

	// endpoint name is keyed by endpoint type in ARM ID
	name := id.Path[path.Base(endpointType)]

	_, err = client.Delete(resGroup, profileName, path.Base(endpointType), name)

	return err
}

func getArmTrafficManagerEndpointProperties(d *schema.ResourceData) *trafficmanager.EndpointProperties {
	var endpointProps trafficmanager.EndpointProperties

	if targetResID := d.Get("target_resource_id").(string); targetResID != "" {
		endpointProps.TargetResourceID = &targetResID
	}

	if target := d.Get("target").(string); target != "" {
		endpointProps.Target = &target
	}

	if status := d.Get("endpoint_status").(string); status != "" {
		endpointProps.EndpointStatus = &status
	}

	if weight := d.Get("weight").(int); weight != 0 {
		w64 := int64(weight)
		endpointProps.Weight = &w64
	}

	if priority := d.Get("priority").(int); priority != 0 {
		p64 := int64(priority)
		endpointProps.Priority = &p64
	}

	if location := d.Get("endpoint_location").(string); location != "" {
		endpointProps.EndpointLocation = &location
	}

	if minChildEndpoints := d.Get("min_child_endpoints").(int); minChildEndpoints != 0 {
		mci64 := int64(minChildEndpoints)
		endpointProps.MinChildEndpoints = &mci64
	}

	return &endpointProps
}

func validateAzureRMTrafficManagerEndpointType(i interface{}, k string) (s []string, errors []error) {
	valid := map[string]struct{}{
		"Microsoft.Network/TrafficManagerProfiles/azureEndpoints":    struct{}{},
		"Microsoft.Network/TrafficManagerProfiles/externalEndpoints": struct{}{},
		"Microsoft.Network/TrafficManagerProfiles/nestedEndpoints":   struct{}{},
	}

	if _, ok := valid[i.(string)]; !ok {
		errors = append(errors, fmt.Errorf("endpoint type invalid, got %s", i.(string)))
	}
	return
}

func validateAzureRMTrafficManagerEndpointWeight(i interface{}, k string) (s []string, errors []error) {
	w := i.(int)
	if w < 1 || w > 1000 {
		errors = append(errors, fmt.Errorf("endpoint weight must be between 1-1000 inclusive"))
	}
	return
}

func validateAzureRMTrafficManagerEndpointPriority(i interface{}, k string) (s []string, errors []error) {
	p := i.(int)
	if p < 1 || p > 1000 {
		errors = append(errors, fmt.Errorf("endpoint priority must be between 1-1000 inclusive"))
	}
	return
}
