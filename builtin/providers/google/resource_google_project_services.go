package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/servicemanagement/v1"
)

func resourceGoogleProjectServices() *schema.Resource {
	return &schema.Resource{
		Create: resourceGoogleProjectServicesCreate,
		Read:   resourceGoogleProjectServicesRead,
		Update: resourceGoogleProjectServicesUpdate,
		Delete: resourceGoogleProjectServicesDelete,

		Schema: map[string]*schema.Schema{
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"services": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

// These services can only be enabled as a side-effect of enabling other services,
// so don't bother storing them in the config or using them for diffing.
var ignore = map[string]struct{}{
	"containeranalysis.googleapis.com": struct{}{},
	"dataproc-control.googleapis.com":  struct{}{},
	"source.googleapis.com":            struct{}{},
}

func resourceGoogleProjectServicesCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	pid := d.Get("project").(string)

	// Get services from config
	cfgServices := getConfigServices(d)

	// Get services from API
	apiServices, err := getApiServices(pid, config)
	if err != nil {
		return fmt.Errorf("Error creating services: %v", err)
	}

	// This call disables any APIs that aren't defined in cfgServices,
	// and enables all of those that are
	err = reconcileServices(cfgServices, apiServices, config, pid)
	if err != nil {
		return fmt.Errorf("Error creating services: %v", err)
	}

	d.SetId(pid)
	return resourceGoogleProjectServicesRead(d, meta)
}

func resourceGoogleProjectServicesRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	services, err := getApiServices(d.Id(), config)
	if err != nil {
		return err
	}

	d.Set("services", services)
	return nil
}

func resourceGoogleProjectServicesUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG]: Updating google_project_services")
	config := meta.(*Config)
	pid := d.Get("project").(string)

	// Get services from config
	cfgServices := getConfigServices(d)

	// Get services from API
	apiServices, err := getApiServices(pid, config)
	if err != nil {
		return fmt.Errorf("Error updating services: %v", err)
	}

	// This call disables any APIs that aren't defined in cfgServices,
	// and enables all of those that are
	err = reconcileServices(cfgServices, apiServices, config, pid)
	if err != nil {
		return fmt.Errorf("Error updating services: %v", err)
	}

	return resourceGoogleProjectServicesRead(d, meta)
}

func resourceGoogleProjectServicesDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG]: Deleting google_project_services")
	config := meta.(*Config)
	services := resourceServices(d)
	for _, s := range services {
		disableService(s, d.Id(), config)
	}
	d.SetId("")
	return nil
}

// This function ensures that the services enabled for a project exactly match that
// in a config by disabling any services that are returned by the API but not present
// in the config
func reconcileServices(cfgServices, apiServices []string, config *Config, pid string) error {
	// Helper to convert slice to map
	m := func(vals []string) map[string]struct{} {
		sm := make(map[string]struct{})
		for _, s := range vals {
			sm[s] = struct{}{}
		}
		return sm
	}

	cfgMap := m(cfgServices)
	apiMap := m(apiServices)

	for k, _ := range apiMap {
		if _, ok := cfgMap[k]; !ok {
			// The service in the API is not in the config; disable it.
			err := disableService(k, pid, config)
			if err != nil {
				return err
			}
		} else {
			// The service exists in the config and the API, so we don't need
			// to re-enable it
			delete(cfgMap, k)
		}
	}

	for k, _ := range cfgMap {
		err := enableService(k, pid, config)
		if err != nil {
			return err
		}
	}
	return nil
}

// Retrieve services defined in a config
func getConfigServices(d *schema.ResourceData) (services []string) {
	if v, ok := d.GetOk("services"); ok {
		for _, svc := range v.(*schema.Set).List() {
			services = append(services, svc.(string))
		}
	}
	return
}

// Retrieve a project's services from the API
func getApiServices(pid string, config *Config) ([]string, error) {
	apiServices := make([]string, 0)
	// Get services from the API
	token := ""
	for paginate := true; paginate; {
		svcResp, err := config.clientServiceMan.Services.List().ConsumerId("project:" + pid).PageToken(token).Do()
		if err != nil {
			return apiServices, err
		}
		for _, v := range svcResp.Services {
			if _, ok := ignore[v.ServiceName]; !ok {
				apiServices = append(apiServices, v.ServiceName)
			}
		}
		token = svcResp.NextPageToken
		paginate = token != ""
	}
	return apiServices, nil
}

func enableService(s, pid string, config *Config) error {
	esr := newEnableServiceRequest(pid)
	sop, err := config.clientServiceMan.Services.Enable(s, esr).Do()
	if err != nil {
		return fmt.Errorf("Error enabling service %q for project %q: %v", s, pid, err)
	}
	// Wait for the operation to complete
	waitErr := serviceManagementOperationWait(config, sop, "api to enable")
	if waitErr != nil {
		return waitErr
	}
	return nil
}
func disableService(s, pid string, config *Config) error {
	dsr := newDisableServiceRequest(pid)
	sop, err := config.clientServiceMan.Services.Disable(s, dsr).Do()
	if err != nil {
		return fmt.Errorf("Error disabling service %q for project %q: %v", s, pid, err)
	}
	// Wait for the operation to complete
	waitErr := serviceManagementOperationWait(config, sop, "api to disable")
	if waitErr != nil {
		return waitErr
	}
	return nil
}

func newEnableServiceRequest(pid string) *servicemanagement.EnableServiceRequest {
	return &servicemanagement.EnableServiceRequest{ConsumerId: "project:" + pid}
}

func newDisableServiceRequest(pid string) *servicemanagement.DisableServiceRequest {
	return &servicemanagement.DisableServiceRequest{ConsumerId: "project:" + pid}
}

func resourceServices(d *schema.ResourceData) []string {
	// Calculate the tags
	var services []string
	if s := d.Get("services"); s != nil {
		ss := s.(*schema.Set)
		services = make([]string, ss.Len())
		for i, v := range ss.List() {
			services[i] = v.(string)
		}
	}
	return services
}
