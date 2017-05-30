package google

import (
	"fmt"
	"log"

	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeRouter() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeRouterCreate,
		Read:   resourceComputeRouterRead,
		Delete: resourceComputeRouterDelete,
		Importer: &schema.ResourceImporter{
			State: resourceComputeRouterImportState,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: linkDiffSuppress,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"bgp": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{

						"asn": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeRouterCreate(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	routerLock := getRouterLockName(region, name)
	mutexKV.Lock(routerLock)
	defer mutexKV.Unlock(routerLock)

	network, err := getNetworkLink(d, config, "network")
	if err != nil {
		return err
	}
	routersService := config.clientCompute.Routers

	router := &compute.Router{
		Name:    name,
		Network: network,
	}

	if v, ok := d.GetOk("description"); ok {
		router.Description = v.(string)
	}

	if _, ok := d.GetOk("bgp"); ok {
		prefix := "bgp.0"
		if v, ok := d.GetOk(prefix + ".asn"); ok {
			asn := v.(int)
			bgp := &compute.RouterBgp{
				Asn: int64(asn),
			}
			router.Bgp = bgp
		}
	}

	op, err := routersService.Insert(project, region, router).Do()
	if err != nil {
		return fmt.Errorf("Error Inserting Router %s into network %s: %s", name, network, err)
	}
	d.SetId(fmt.Sprintf("%s/%s", region, name))
	err = computeOperationWaitRegion(config, op, project, region, "Inserting Router")
	if err != nil {
		d.SetId("")
		return fmt.Errorf("Error Waiting to Insert Router %s into network %s: %s", name, network, err)
	}

	return resourceComputeRouterRead(d, meta)
}

func resourceComputeRouterRead(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	routersService := config.clientCompute.Routers
	router, err := routersService.Get(project, region, name).Do()

	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing router %s/%s because it is gone", region, name)
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error Reading Router %s: %s", name, err)
	}

	d.Set("self_link", router.SelfLink)
	d.Set("network", router.Network)

	d.Set("name", router.Name)
	d.Set("description", router.Description)
	d.Set("region", region)
	d.Set("project", project)
	d.Set("bgp", flattenAsn(router.Bgp.Asn))
	d.SetId(fmt.Sprintf("%s/%s", region, name))

	return nil
}

func resourceComputeRouterDelete(d *schema.ResourceData, meta interface{}) error {

	config := meta.(*Config)

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	name := d.Get("name").(string)

	routerLock := getRouterLockName(region, name)
	mutexKV.Lock(routerLock)
	defer mutexKV.Unlock(routerLock)

	routersService := config.clientCompute.Routers

	op, err := routersService.Delete(project, region, name).Do()
	if err != nil {
		return fmt.Errorf("Error Reading Router %s: %s", name, err)
	}

	err = computeOperationWaitRegion(config, op, project, region, "Deleting Router")
	if err != nil {
		return fmt.Errorf("Error Waiting to Delete Router %s: %s", name, err)
	}

	d.SetId("")
	return nil
}

func resourceComputeRouterImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("Invalid router specifier. Expecting {region}/{name}")
	}

	d.Set("region", parts[0])
	d.Set("name", parts[1])

	return []*schema.ResourceData{d}, nil
}

func getRouterLink(config *Config, project string, region string, router string) (string, error) {

	if !strings.HasPrefix(router, "https://www.googleapis.com/compute/") {
		// Router value provided is just the name, lookup the router SelfLink
		routerData, err := config.clientCompute.Routers.Get(
			project, region, router).Do()
		if err != nil {
			return "", fmt.Errorf("Error reading router: %s", err)
		}
		router = routerData.SelfLink
	}

	return router, nil

}

func flattenAsn(asn int64) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, 1)
	r := make(map[string]interface{})
	r["asn"] = asn
	result = append(result, r)
	return result
}
