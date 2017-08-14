package openstack

import (
	"fmt"
	"log"
	"sort"

	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"

	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceImagesImageV2() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceImagesImageV2Read,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"visibility": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"owner": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"size_min": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"size_max": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"sort_key": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "name",
			},

			"sort_direction": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "asc",
				ValidateFunc: dataSourceImagesImageV2SortDirection,
			},

			"tag": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"most_recent": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			// Computed values
			"container_format": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"disk_format": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"min_disk_gb": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"min_ram_mb": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"protected": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"checksum": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"size_bytes": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"metadata": {
				Type:     schema.TypeMap,
				Computed: true,
			},

			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"file": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"schema": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// dataSourceImagesImageV2Read performs the image lookup.
func dataSourceImagesImageV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	imageClient, err := config.imageV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack image client: %s", err)
	}

	visibility := resourceImagesImageV2VisibilityFromString(d.Get("visibility").(string))

	listOpts := images.ListOpts{
		Name:       d.Get("name").(string),
		Visibility: visibility,
		Owner:      d.Get("owner").(string),
		Status:     images.ImageStatusActive,
		SizeMin:    int64(d.Get("size_min").(int)),
		SizeMax:    int64(d.Get("size_max").(int)),
		SortKey:    d.Get("sort_key").(string),
		SortDir:    d.Get("sort_direction").(string),
		Tag:        d.Get("tag").(string),
	}

	log.Printf("[DEBUG] List Options: %#v", listOpts)

	var image images.Image
	allPages, err := images.List(imageClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("Unable to query images: %s", err)
	}

	allImages, err := images.ExtractImages(allPages)
	if err != nil {
		return fmt.Errorf("Unable to retrieve images: %s", err)
	}

	if len(allImages) < 1 {
		return fmt.Errorf("Your query returned no results. " +
			"Please change your search criteria and try again.")
	}

	if len(allImages) > 1 {
		recent := d.Get("most_recent").(bool)
		log.Printf("[DEBUG] Multiple results found and `most_recent` is set to: %t", recent)
		if recent {
			image = mostRecentImage(allImages)
		} else {
			return fmt.Errorf("Your query returned more than one result. Please try a more " +
				"specific search criteria, or set `most_recent` attribute to true.")
		}
	} else {
		image = allImages[0]
	}

	log.Printf("[DEBUG] Single Image found: %s", image.ID)
	return dataSourceImagesImageV2Attributes(d, &image)
}

// dataSourceImagesImageV2Attributes populates the fields of an Image resource.
func dataSourceImagesImageV2Attributes(d *schema.ResourceData, image *images.Image) error {
	log.Printf("[DEBUG] openstack_images_image details: %#v", image)

	d.SetId(image.ID)
	d.Set("name", image.Name)
	d.Set("tags", image.Tags)
	d.Set("container_format", image.ContainerFormat)
	d.Set("disk_format", image.DiskFormat)
	d.Set("min_disk_gb", image.MinDiskGigabytes)
	d.Set("min_ram_mb", image.MinRAMMegabytes)
	d.Set("owner", image.Owner)
	d.Set("protected", image.Protected)
	d.Set("visibility", image.Visibility)
	d.Set("checksum", image.Checksum)
	d.Set("size_bytes", image.SizeBytes)
	d.Set("metadata", image.Metadata)
	d.Set("created_at", image.CreatedAt)
	d.Set("updated_at", image.UpdatedAt)
	d.Set("file", image.File)
	d.Set("schema", image.Schema)

	return nil
}

type imageSort []images.Image

func (a imageSort) Len() int      { return len(a) }
func (a imageSort) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a imageSort) Less(i, j int) bool {
	itime := a[i].UpdatedAt
	jtime := a[j].UpdatedAt
	return itime.Unix() < jtime.Unix()
}

// Returns the most recent Image out of a slice of images.
func mostRecentImage(images []images.Image) images.Image {
	sortedImages := images
	sort.Sort(imageSort(sortedImages))
	return sortedImages[len(sortedImages)-1]
}

func dataSourceImagesImageV2SortDirection(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value != "asc" && value != "desc" {
		err := fmt.Errorf("%s must be either asc or desc", k)
		errors = append(errors, err)
	}
	return
}
