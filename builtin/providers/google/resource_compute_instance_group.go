package google

import (
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceComputeInstanceGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeInstanceGroupCreate,
		Read:   resourceComputeInstanceGroupRead,
		Update: resourceComputeInstanceGroupUpdate,
		Delete: resourceComputeInstanceGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"instances": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"named_port": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"network": &schema.Schema{
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

			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func getInstanceReferences(instanceUrls []string) (refs []*compute.InstanceReference) {
	for _, v := range instanceUrls {
		refs = append(refs, &compute.InstanceReference{
			Instance: v,
		})
	}
	return refs
}

func validInstanceURLs(instanceUrls []string) bool {
	for _, v := range instanceUrls {
		if !strings.HasPrefix(v, "https://www.googleapis.com/compute/v1/") {
			return false
		}
	}
	return true
}

func resourceComputeInstanceGroupCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Build the parameter
	instanceGroup := &compute.InstanceGroup{
		Name: d.Get("name").(string),
	}

	// Set optional fields
	if v, ok := d.GetOk("description"); ok {
		instanceGroup.Description = v.(string)
	}

	if v, ok := d.GetOk("named_port"); ok {
		instanceGroup.NamedPorts = getNamedPorts(v.([]interface{}))
	}

	log.Printf("[DEBUG] InstanceGroup insert request: %#v", instanceGroup)
	op, err := config.clientCompute.InstanceGroups.Insert(
		project, d.Get("zone").(string), instanceGroup).Do()
	if err != nil {
		return fmt.Errorf("Error creating InstanceGroup: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(instanceGroup.Name)

	// Wait for the operation to complete
	err = computeOperationWaitZone(config, op, d.Get("zone").(string), "Creating InstanceGroup")
	if err != nil {
		return err
	}

	if v, ok := d.GetOk("instances"); ok {
		instanceUrls := convertStringArr(v.([]interface{}))
		if !validInstanceURLs(instanceUrls) {
			return fmt.Errorf("Error invalid instance URLs: %v", instanceUrls)
		}

		addInstanceReq := &compute.InstanceGroupsAddInstancesRequest{
			Instances: getInstanceReferences(instanceUrls),
		}

		log.Printf("[DEBUG] InstanceGroup add instances request: %#v", addInstanceReq)
		op, err := config.clientCompute.InstanceGroups.AddInstances(
			project, d.Get("zone").(string), d.Id(), addInstanceReq).Do()
		if err != nil {
			return fmt.Errorf("Error adding instances to InstanceGroup: %s", err)
		}

		// Wait for the operation to complete
		err = computeOperationWaitZone(config, op, d.Get("zone").(string), "Adding instances to InstanceGroup")
		if err != nil {
			return err
		}
	}

	return resourceComputeInstanceGroupRead(d, meta)
}

func resourceComputeInstanceGroupRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// retreive instance group
	instanceGroup, err := config.clientCompute.InstanceGroups.Get(
		project, d.Get("zone").(string), d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading InstanceGroup: %s", err)
	}

	// retreive instance group members
	var memberUrls []string
	members, err := config.clientCompute.InstanceGroups.ListInstances(
		project, d.Get("zone").(string), d.Id(), &compute.InstanceGroupsListInstancesRequest{
			InstanceState: "ALL",
		}).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't have any instances
			d.Set("instances", nil)
		} else {
			// any other errors return them
			return fmt.Errorf("Error reading InstanceGroup Members: %s", err)
		}
	} else {
		for _, member := range members.Items {
			memberUrls = append(memberUrls, member.Instance)
		}
		log.Printf("[DEBUG] InstanceGroup members: %v", memberUrls)
		d.Set("instances", memberUrls)
	}

	// Set computed fields
	d.Set("network", instanceGroup.Network)
	d.Set("size", instanceGroup.Size)
	d.Set("self_link", instanceGroup.SelfLink)

	return nil
}
func resourceComputeInstanceGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// refresh the state incase referenced instances have been removed earlier in the run
	err = resourceComputeInstanceGroupRead(d, meta)
	if err != nil {
		return fmt.Errorf("Error reading InstanceGroup: %s", err)
	}

	d.Partial(true)

	if d.HasChange("instances") {
		// to-do check for no instances
		from_, to_ := d.GetChange("instances")

		from := convertStringArr(from_.([]interface{}))
		to := convertStringArr(to_.([]interface{}))

		if !validInstanceURLs(from) {
			return fmt.Errorf("Error invalid instance URLs: %v", from)
		}
		if !validInstanceURLs(to) {
			return fmt.Errorf("Error invalid instance URLs: %v", to)
		}

		add, remove := calcAddRemove(from, to)

		if len(remove) > 0 {
			removeReq := &compute.InstanceGroupsRemoveInstancesRequest{
				Instances: getInstanceReferences(remove),
			}

			log.Printf("[DEBUG] InstanceGroup remove instances request: %#v", removeReq)
			removeOp, err := config.clientCompute.InstanceGroups.RemoveInstances(
				project, d.Get("zone").(string), d.Id(), removeReq).Do()
			if err != nil {
				return fmt.Errorf("Error removing instances from InstanceGroup: %s", err)
			}

			// Wait for the operation to complete
			err = computeOperationWaitZone(config, removeOp, d.Get("zone").(string), "Updating InstanceGroup")
			if err != nil {
				return err
			}
		}

		if len(add) > 0 {

			addReq := &compute.InstanceGroupsAddInstancesRequest{
				Instances: getInstanceReferences(add),
			}

			log.Printf("[DEBUG] InstanceGroup adding instances request: %#v", addReq)
			addOp, err := config.clientCompute.InstanceGroups.AddInstances(
				project, d.Get("zone").(string), d.Id(), addReq).Do()
			if err != nil {
				return fmt.Errorf("Error adding instances from InstanceGroup: %s", err)
			}

			// Wait for the operation to complete
			err = computeOperationWaitZone(config, addOp, d.Get("zone").(string), "Updating InstanceGroup")
			if err != nil {
				return err
			}
		}

		d.SetPartial("instances")
	}

	if d.HasChange("named_port") {
		namedPorts := getNamedPorts(d.Get("named_port").([]interface{}))

		namedPortsReq := &compute.InstanceGroupsSetNamedPortsRequest{
			NamedPorts: namedPorts,
		}

		log.Printf("[DEBUG] InstanceGroup updating named ports request: %#v", namedPortsReq)
		op, err := config.clientCompute.InstanceGroups.SetNamedPorts(
			project, d.Get("zone").(string), d.Id(), namedPortsReq).Do()
		if err != nil {
			return fmt.Errorf("Error updating named ports for InstanceGroup: %s", err)
		}

		err = computeOperationWaitZone(config, op, d.Get("zone").(string), "Updating InstanceGroup")
		if err != nil {
			return err
		}
		d.SetPartial("named_port")
	}

	d.Partial(false)

	return resourceComputeInstanceGroupRead(d, meta)
}

func resourceComputeInstanceGroupDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	zone := d.Get("zone").(string)
	op, err := config.clientCompute.InstanceGroups.Delete(project, zone, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting InstanceGroup: %s", err)
	}

	err = computeOperationWaitZone(config, op, zone, "Deleting InstanceGroup")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
