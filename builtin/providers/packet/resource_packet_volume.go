package packet

import (
	"fmt"

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
				Required: false,
				Optional: true,
				Computed: true,
			},

			"facility": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"plan": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"billing_cycle": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"locked": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
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
		PlanID:       d.Get("plan").(string),
		FacilityID:   d.Get("facility").(string),
		BillingCycle: d.Get("billing_cycle").(string),
		ProjectID:    d.Get("project_id").(string),
	}

	if attr, ok := d.GetOk("description"); ok {
		createRequest.Description = attr.(string)
	}

	if attr, ok := d.GetOk("size"); ok {
		createRequest.Size = attr.(int)
	}

	snapshot_policies := d.Get("snapshot_policies.#").(int)
	if snapshot_policies > 0 {
		createRequest.SnapshotPolicies = make([]*packngo.SnapshotPolicy, 0, snapshot_policies)
		for i := 0; i < snapshot_policies; i++ {
			key := fmt.Sprintf("snapshot_policies.%d", i)
			createRequest.SnapshotPolicies = append(createRequest.SnapshotPolicies, d.Get(key).(*packngo.SnapshotPolicy))
		}
	}

	newVolume, _, err := client.Volumes.Create(createRequest)
	if err != nil {
		return friendlyError(err)
	}

	d.SetId(newVolume.ID)

	return resourcePacketVolumeRead(d, meta)
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

	snapshot_policies := make([]*packngo.SnapshotPolicy, 0, len(volume.SnapshotPolicies))
	for _, snapshot_policy := range volume.SnapshotPolicies {
		snapshot_policies = append(snapshot_policies, snapshot_policy)
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
