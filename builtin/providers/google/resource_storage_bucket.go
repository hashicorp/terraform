package google

import (
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/storage/v1"
)

func resourceStorageBucket() *schema.Resource {
	return &schema.Resource{
		Create: resourceStorageBucketCreate,
		Read:   resourceStorageBucketRead,
		Update: resourceStorageBucketUpdate,
		Delete: resourceStorageBucketDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"predefined_acl": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "projectPrivate",
				Optional: true,
				ForceNew: true,
			},
			"location": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "US",
				Optional: true,
				ForceNew: true,
			},
			"force_destroy": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceStorageBucketCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Get the bucket and acl
	bucket := d.Get("name").(string)
	acl := d.Get("predefined_acl").(string)
	location := d.Get("location").(string)

	// Create a bucket, setting the acl, location and name.
	sb := &storage.Bucket{Name: bucket, Location: location}
	res, err := config.clientStorage.Buckets.Insert(config.Project, sb).PredefinedAcl(acl).Do()

	if err != nil {
		fmt.Printf("Error creating bucket %s: %v", bucket, err)
		return err
	}

	log.Printf("[DEBUG] Created bucket %v at location %v\n\n", res.Name, res.SelfLink)

	// Assign the bucket ID as the resource ID
	d.SetId(res.Id)

	return nil
}

func resourceStorageBucketUpdate(d *schema.ResourceData, meta interface{}) error {
	// Only thing you can currently change is force_delete (all other properties have ForceNew)
	// which is just terraform object state change, so nothing to do here
	return nil
}

func resourceStorageBucketRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Get the bucket and acl
	bucket := d.Get("name").(string)
	res, err := config.clientStorage.Buckets.Get(bucket).Do()

	if err != nil {
		fmt.Printf("Error reading bucket %s: %v", bucket, err)
		return err
	}

	log.Printf("[DEBUG] Read bucket %v at location %v\n\n", res.Name, res.SelfLink)

	// Update the bucket ID according to the resource ID
	d.SetId(res.Id)

	return nil
}

func resourceStorageBucketDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Get the bucket
	bucket := d.Get("name").(string)

	for {
		res, err := config.clientStorage.Objects.List(bucket).Do()
		if err != nil {
			fmt.Printf("Error Objects.List failed: %v", err)
			return err
		}

		if len(res.Items) != 0 {
			if d.Get("force_destroy").(bool) {
				// purge the bucket...
				log.Printf("[DEBUG] GCS Bucket attempting to forceDestroy\n\n")

				for _, object := range res.Items {
					log.Printf("[DEBUG] Found %s", object.Name)
					if err := config.clientStorage.Objects.Delete(bucket, object.Name).Do(); err != nil {
						log.Fatalf("Error trying to delete object: %s %s\n\n", object.Name, err)
					} else {
						log.Printf("Object deleted: %s \n\n", object.Name)
					}
				}

			} else {
				delete_err := errors.New("Error trying to delete a bucket containing objects without `force_destroy` set to true")
				log.Printf("Error! %s : %s\n\n", bucket, delete_err)
				return delete_err
			}
		} else {
			break // 0 items, bucket empty
		}
	}

	// remove empty bucket
	err := config.clientStorage.Buckets.Delete(bucket).Do()
	if err != nil {
		fmt.Printf("Error deleting bucket %s: %v\n\n", bucket, err)
		return err
	}
	log.Printf("[DEBUG] Deleted bucket %v\n\n", bucket)

	return nil
}
