package rancher

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	rancher "github.com/rancher/go-rancher/client"
)

// ro_labels are used internally by Rancher
// They are not documented and should not be set in Terraform
var ro_labels = []string{
	"io.rancher.host.agent_image",
	"io.rancher.host.docker_version",
	"io.rancher.host.kvm",
	"io.rancher.host.linux_kernel_version",
}

func resourceRancherHost() *schema.Resource {
	return &schema.Resource{
		Create: resourceRancherHostCreate,
		Read:   resourceRancherHostRead,
		Update: resourceRancherHostUpdate,
		Delete: resourceRancherHostDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"environment_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func resourceRancherHostCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO][rancher] Creating Host: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	hosts, _ := client.Host.List(NewListOpts())
	hostname := d.Get("hostname").(string)
	var host rancher.Host

	for _, h := range hosts.Data {
		if h.Hostname == hostname {
			host = h
			break
		}
	}

	if host.Hostname == "" {
		return fmt.Errorf("Failed to find host %s", hostname)
	}

	d.SetId(host.Id)

	return resourceRancherHostUpdate(d, meta)
}

func resourceRancherHostRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Refreshing Host: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	host, err := client.Host.ById(d.Id())
	if err != nil {
		return err
	}

	if host == nil {
		log.Printf("[INFO] Host %s not found", d.Id())
		d.SetId("")
		return nil
	}

	if removed(host.State) {
		log.Printf("[INFO] Host %s was removed on %v", d.Id(), host.Removed)
		d.SetId("")
		return nil
	}

	log.Printf("[INFO] Host Name: %s", host.Name)

	d.Set("description", host.Description)
	d.Set("name", host.Name)
	d.Set("hostname", host.Hostname)

	labels := host.Labels
	// Remove read-only labels
	for _, lbl := range ro_labels {
		delete(labels, lbl)
	}
	d.Set("labels", host.Labels)

	return nil
}

func resourceRancherHostUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Updating Host: %s", d.Id())
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	// Process labels: merge ro_labels into new labels
	labels := d.Get("labels").(map[string]interface{})
	host, err := client.Host.ById(d.Id())
	if err != nil {
		return err
	}
	for _, lbl := range ro_labels {
		labels[lbl] = host.Labels[lbl]
	}

	data := map[string]interface{}{
		"name":        &name,
		"description": &description,
		"labels":      &labels,
	}

	var newHost rancher.Host
	if err := client.Update("host", &host.Resource, data, &newHost); err != nil {
		return err
	}

	return resourceRancherHostRead(d, meta)
}

func resourceRancherHostDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting Host: %s", d.Id())
	id := d.Id()
	client, err := meta.(*Config).EnvironmentClient(d.Get("environment_id").(string))
	if err != nil {
		return err
	}

	host, err := client.Host.ById(id)
	if err != nil {
		return err
	}

	if err := client.Host.Delete(host); err != nil {
		return fmt.Errorf("Error deleting Host: %s", err)
	}

	log.Printf("[DEBUG] Waiting for host (%s) to be removed", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"active", "removed", "removing"},
		Target:     []string{"removed"},
		Refresh:    HostStateRefreshFunc(client, id),
		Timeout:    10 * time.Minute,
		Delay:      1 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, waitErr := stateConf.WaitForState()
	if waitErr != nil {
		return fmt.Errorf(
			"Error waiting for host (%s) to be removed: %s", id, waitErr)
	}

	d.SetId("")
	return nil
}

// HostStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Rancher Host.
func HostStateRefreshFunc(client *rancher.RancherClient, hostID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		host, err := client.Host.ById(hostID)

		if err != nil {
			return nil, "", err
		}

		return host, host.State, nil
	}
}
