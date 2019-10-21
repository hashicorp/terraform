package openstack

import (
	"fmt"
	"log"

	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/snapshots"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceSharedFilesystemSnapshotV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceSharedFilesystemSnapshotV2Read,

		Schema: map[string]*schema.Schema{
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"project_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"status": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"share_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"size": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"share_proto": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"share_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func dataSourceSharedFilesystemSnapshotV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	sfsClient, err := config.sharedfilesystemV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack sharedfilesystem sfsClient: %s", err)
	}

	sfsClient.Microversion = minManilaShareMicroversion

	listOpts := snapshots.ListOpts{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ProjectID:   d.Get("project_id").(string),
		Status:      d.Get("status").(string),
	}

	allPages, err := snapshots.ListDetail(sfsClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query snapshots: %s", err)
	}

	allSnapshots, err := snapshots.ExtractSnapshots(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve snapshots: %s", err)
	}

	if len(allSnapshots) < 1 {
		return fmt.Errorf("Your query returned no results. " +
			"Please change your search criteria and try again.")
	}

	var share snapshots.Snapshot
	if len(allSnapshots) > 1 {
		log.Printf("[DEBUG] Multiple results found: %#v", allSnapshots)
		return fmt.Errorf("Your query returned more than one result. Please try a more " +
			"specific search criteria.")
	} else {
		share = allSnapshots[0]
	}

	return dataSourceSharedFilesystemSnapshotV2Attributes(d, &share, GetRegion(d, config))
}

func dataSourceSharedFilesystemSnapshotV2Attributes(d *schema.ResourceData, snapshot *snapshots.Snapshot, region string) error {
	d.SetId(snapshot.ID)
	d.Set("name", snapshot.Name)
	d.Set("region", region)
	d.Set("project_id", snapshot.ProjectID)
	d.Set("description", snapshot.Description)
	d.Set("size", snapshot.Size)
	d.Set("status", snapshot.Status)
	d.Set("share_id", snapshot.ShareID)
	d.Set("share_proto", snapshot.ShareProto)
	d.Set("share_size", snapshot.ShareSize)

	return nil
}
