package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeImage() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeImageCreate,
		Read:   resourceComputeImageRead,
		Delete: resourceComputeImageDelete,

		Schema: map[string]*schema.Schema{
			// TODO(cblecker): one of source_disk or raw_disk is required

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"family": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"source_disk": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"raw_disk": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"sha1": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"container_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "TAR",
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

func resourceComputeImageCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Build the image
	image := &compute.Image{
		Name: d.Get("name").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		image.Description = v.(string)
	}

	if v, ok := d.GetOk("family"); ok {
		image.Family = v.(string)
	}

	// Load up the source_disk for this image if specified
	if v, ok := d.GetOk("source_disk"); ok {
		image.SourceDisk = v.(string)
	}

	// Load up the raw_disk for this image if specified
	if v, ok := d.GetOk("raw_disk"); ok {
		rawDiskEle := v.([]interface{})[0].(map[string]interface{})
		imageRawDisk := &compute.ImageRawDisk{
			Source:        rawDiskEle["source"].(string),
			ContainerType: rawDiskEle["container_type"].(string),
		}
		if val, ok := rawDiskEle["sha1"]; ok {
			imageRawDisk.Sha1Checksum = val.(string)
		}

		image.RawDisk = imageRawDisk
	}

	// Insert the image
	op, err := config.clientCompute.Images.Insert(
		project, image).Do()
	if err != nil {
		return fmt.Errorf("Error creating image: %s", err)
	}

	// Store the ID
	d.SetId(image.Name)

	err = computeOperationWaitGlobal(config, op, project, "Creating Image")
	if err != nil {
		return err
	}

	return resourceComputeImageRead(d, meta)
}

func resourceComputeImageRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	image, err := config.clientCompute.Images.Get(
		project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			log.Printf("[WARN] Removing Image %q because it's gone", d.Get("name").(string))
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading image: %s", err)
	}

	d.Set("self_link", image.SelfLink)

	return nil
}

func resourceComputeImageDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the image
	log.Printf("[DEBUG] image delete request")
	op, err := config.clientCompute.Images.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting image: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Deleting image")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
