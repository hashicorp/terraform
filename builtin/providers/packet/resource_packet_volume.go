package packet

import (
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/packethost/packngo"
)

func resourcePacketVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourcePacketVolumeCreate,
		Read:   resourcePacketVolumeRead,
		Update: resourcePacketVolumeUpdate,
		Delete: resourcePacketVolumeDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"project_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
				Optional: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"facility": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"plan": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"billing_cycle": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},

			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"locked": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"snapshot_policies": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"snapshot_frequency": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"snapshot_count": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"attachments": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"href": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"created": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"updated": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourcePacketVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	createRequest := &packngo.VolumeCreateRequest{
		PlanID:     d.Get("plan").(string),
		FacilityID: d.Get("facility").(string),
		ProjectID:  d.Get("project_id").(string),
		Size:       d.Get("size").(int),
	}

	if attr, ok := d.GetOk("billing_cycle"); ok {
		createRequest.BillingCycle = attr.(string)
	} else {
		createRequest.BillingCycle = "hourly"
	}

	if attr, ok := d.GetOk("description"); ok {
		createRequest.Description = attr.(string)
	}

	snapshot_count := d.Get("snapshot_policies.#").(int)
	if snapshot_count > 0 {
		createRequest.SnapshotPolicies = make([]*packngo.SnapshotPolicy, 0, snapshot_count)
		for i := 0; i < snapshot_count; i++ {
			policy := new(packngo.SnapshotPolicy)
			policy.SnapshotFrequency = d.Get(fmt.Sprintf("snapshot_policies.%d.snapshot_frequency", i)).(string)
			policy.SnapshotCount = d.Get(fmt.Sprintf("snapshot_policies.%d.snapshot_count", i)).(int)
			createRequest.SnapshotPolicies = append(createRequest.SnapshotPolicies, policy)
		}
	}

	newVolume, _, err := client.Volumes.Create(createRequest)
	if err != nil {
		return friendlyError(err)
	}

	d.SetId(newVolume.ID)

	_, err = waitForVolumeAttribute(d, "active", []string{"queued", "provisioning"}, "state", meta)
	if err != nil {
		if isForbidden(err) {
			// If the volume doesn't get to the active state, we can't recover it from here.
			d.SetId("")

			return errors.New("provisioning time limit exceeded; the Packet team will investigate")
		}
		return err
	}

	return resourcePacketVolumeRead(d, meta)
}

func waitForVolumeAttribute(d *schema.ResourceData, target string, pending []string, attribute string, meta interface{}) (interface{}, error) {
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     []string{target},
		Refresh:    newVolumeStateRefreshFunc(d, attribute, meta),
		Timeout:    60 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}
	return stateConf.WaitForState()
}

func newVolumeStateRefreshFunc(d *schema.ResourceData, attribute string, meta interface{}) resource.StateRefreshFunc {
	client := meta.(*packngo.Client)

	return func() (interface{}, string, error) {
		if err := resourcePacketVolumeRead(d, meta); err != nil {
			return nil, "", err
		}

		if attr, ok := d.GetOk(attribute); ok {
			volume, _, err := client.Volumes.Get(d.Id())
			if err != nil {
				return nil, "", friendlyError(err)
			}
			return &volume, attr.(string), nil
		}

		return nil, "", nil
	}
}

func resourcePacketVolumeRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	volume, _, err := client.Volumes.Get(d.Id())
	if err != nil {
		err = friendlyError(err)

		// If the volume somehow already destroyed, mark as succesfully gone.
		if isNotFound(err) {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", volume.Name)
	d.Set("description", volume.Description)
	d.Set("size", volume.Size)
	d.Set("plan", volume.Plan.Slug)
	d.Set("facility", volume.Facility.Code)
	d.Set("state", volume.State)
	d.Set("billing_cycle", volume.BillingCycle)
	d.Set("locked", volume.Locked)
	d.Set("created", volume.Created)
	d.Set("updated", volume.Updated)

	snapshot_policies := make([]map[string]interface{}, 0, len(volume.SnapshotPolicies))
	for _, snapshot_policy := range volume.SnapshotPolicies {
		policy := map[string]interface{}{
			"snapshot_frequency": snapshot_policy.SnapshotFrequency,
			"snapshot_count":     snapshot_policy.SnapshotCount,
		}
		snapshot_policies = append(snapshot_policies, policy)
	}
	d.Set("snapshot_policies", snapshot_policies)

	attachments := make([]*packngo.Attachment, 0, len(volume.Attachments))
	for _, attachment := range volume.Attachments {
		attachments = append(attachments, attachment)
	}
	d.Set("attachments", attachments)

	return nil
}

func resourcePacketVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	updateRequest := &packngo.VolumeUpdateRequest{
		ID: d.Get("id").(string),
	}

	if attr, ok := d.GetOk("description"); ok {
		updateRequest.Description = attr.(string)
	}

	if attr, ok := d.GetOk("plan"); ok {
		updateRequest.Plan = attr.(string)
	}

	_, _, err := client.Volumes.Update(updateRequest)
	if err != nil {
		return friendlyError(err)
	}

	return resourcePacketVolumeRead(d, meta)
}

func resourcePacketVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*packngo.Client)

	if _, err := client.Volumes.Delete(d.Id()); err != nil {
		return friendlyError(err)
	}

	return nil
}
