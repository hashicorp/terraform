package google

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceComputeTargetHttpProxy() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeTargetHttpProxyCreate,
		Read:   resourceComputeTargetHttpProxyRead,
		Delete: resourceComputeTargetHttpProxyDelete,
		Update: resourceComputeTargetHttpProxyUpdate,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"url_map": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeTargetHttpProxyCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	proxy := &compute.TargetHttpProxy{
		Name:   d.Get("name").(string),
		UrlMap: d.Get("url_map").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		proxy.Description = v.(string)
	}

	log.Printf("[DEBUG] TargetHttpProxy insert request: %#v", proxy)
	op, err := config.clientCompute.TargetHttpProxies.Insert(
		project, proxy).Do()
	if err != nil {
		return fmt.Errorf("Error creating TargetHttpProxy: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Creating Target Http Proxy")
	if err != nil {
		return err
	}

	d.SetId(proxy.Name)

	return resourceComputeTargetHttpProxyRead(d, meta)
}

func resourceComputeTargetHttpProxyUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	d.Partial(true)

	if d.HasChange("url_map") {
		url_map := d.Get("url_map").(string)
		url_map_ref := &compute.UrlMapReference{UrlMap: url_map}
		op, err := config.clientCompute.TargetHttpProxies.SetUrlMap(
			project, d.Id(), url_map_ref).Do()
		if err != nil {
			return fmt.Errorf("Error updating target: %s", err)
		}

		err = computeOperationWaitGlobal(config, op, project, "Updating Target Http Proxy")
		if err != nil {
			return err
		}

		d.SetPartial("url_map")
	}

	d.Partial(false)

	return resourceComputeTargetHttpProxyRead(d, meta)
}

func resourceComputeTargetHttpProxyRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	proxy, err := config.clientCompute.TargetHttpProxies.Get(
		project, d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Target HTTP Proxy %q", d.Get("name").(string)))
	}

	d.Set("self_link", proxy.SelfLink)
	d.Set("id", strconv.FormatUint(proxy.Id, 10))

	return nil
}

func resourceComputeTargetHttpProxyDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the TargetHttpProxy
	log.Printf("[DEBUG] TargetHttpProxy delete request")
	op, err := config.clientCompute.TargetHttpProxies.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting TargetHttpProxy: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Deleting Target Http Proxy")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
