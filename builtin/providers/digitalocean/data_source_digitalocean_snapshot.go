package digitalocean

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"time"

	"github.com/digitalocean/godo"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceDigitalOceanSnapshot() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceDoSnapshotRead,

		Schema: map[string]*schema.Schema{
			"name_regex": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateNameRegex,
			},
			"most_recent": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"region_filter": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"resource_type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateResourceType,
			},
			// Computed values.
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"min_disk_size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"regions": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"size_gigabytes": {
				Type:     schema.TypeFloat,
				Computed: true,
			},
		},
	}
}

// dataSourceDoSnapshotRead performs the Snapshot lookup.
func dataSourceDoSnapshotRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*godo.Client)

	resourceType := d.Get("resource_type")
	nameRegex, nameRegexOk := d.GetOk("name_regex")
	regionFilter, regionFilterOk := d.GetOk("region_filter")

	pageOpt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	var snapshotList []godo.Snapshot
	for {
		var snapshots []godo.Snapshot
		var resp *godo.Response
		var err error

		switch resourceType {
		case "droplet":
			snapshots, resp, err = client.Snapshots.ListDroplet(pageOpt)
		case "volume":
			snapshots, resp, err = client.Snapshots.ListVolume(pageOpt)
		}

		for _, s := range snapshots {
			snapshotList = append(snapshotList, s)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return err
		}

		pageOpt.Page = page + 1
	}

	var snapshotsFilteredByName []godo.Snapshot
	if nameRegexOk {
		r := regexp.MustCompile(nameRegex.(string))
		for _, snapshot := range snapshotList {
			if r.MatchString(snapshot.Name) {
				snapshotsFilteredByName = append(snapshotsFilteredByName, snapshot)
			}
		}
	} else {
		snapshotsFilteredByName = snapshotList[:]
	}

	var snapshotsFilteredByRegion []godo.Snapshot
	if regionFilterOk {
		for _, snapshot := range snapshotsFilteredByName {
			for _, region := range snapshot.Regions {
				if region == regionFilter {
					snapshotsFilteredByRegion = append(snapshotsFilteredByRegion, snapshot)
				}
			}
		}
	} else {
		snapshotsFilteredByRegion = snapshotsFilteredByName[:]
	}

	var snapshot godo.Snapshot
	if len(snapshotsFilteredByRegion) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	if len(snapshotsFilteredByRegion) > 1 {
		recent := d.Get("most_recent").(bool)
		log.Printf("[DEBUG] do_snapshot - multiple results found and `most_recent` is set to: %t", recent)
		if recent {
			snapshot = mostRecentSnapshot(snapshotsFilteredByRegion)
		} else {
			return fmt.Errorf("Your query returned more than one result. Please try a more " +
				"specific search criteria, or set `most_recent` attribute to true.")
		}
	} else {
		// Query returned single result.
		snapshot = snapshotsFilteredByRegion[0]
	}

	log.Printf("[DEBUG] do_snapshot - Single Snapshot found: %s", snapshot.ID)
	return snapshotDescriptionAttributes(d, snapshot)
}

type snapshotSort []godo.Snapshot

func (a snapshotSort) Len() int      { return len(a) }
func (a snapshotSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a snapshotSort) Less(i, j int) bool {
	itime, _ := time.Parse(time.RFC3339, a[i].Created)
	jtime, _ := time.Parse(time.RFC3339, a[j].Created)
	return itime.Unix() < jtime.Unix()
}

// Returns the most recent Snapshot out of a slice of Snapshots.
func mostRecentSnapshot(snapshots []godo.Snapshot) godo.Snapshot {
	sortedSnapshots := snapshots
	sort.Sort(snapshotSort(sortedSnapshots))
	return sortedSnapshots[len(sortedSnapshots)-1]
}

// populate the numerous fields that the Snapshot description returns.
func snapshotDescriptionAttributes(d *schema.ResourceData, snapshot godo.Snapshot) error {
	d.SetId(snapshot.ID)
	d.Set("created_at", snapshot.Created)
	d.Set("min_disk_size", snapshot.MinDiskSize)
	d.Set("name", snapshot.Name)
	d.Set("regions", snapshot.Regions)
	d.Set("resource_id", snapshot.ResourceID)
	d.Set("size_gigabytes", snapshot.SizeGigaBytes)

	return nil
}

func validateNameRegex(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	if _, err := regexp.Compile(value); err != nil {
		errors = append(errors, fmt.Errorf(
			"%q contains an invalid regular expression: %s",
			k, err))
	}
	return
}

func validateResourceType(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	switch value {
	case
		"droplet",
		"volume":
		return
	}

	errors = append(errors, fmt.Errorf(
		"Invalid %q specified: %s",
		k, value))

	return
}
