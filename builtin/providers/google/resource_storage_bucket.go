package google

import (
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/googleapi"
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

			"force_destroy": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "US",
				Optional: true,
				ForceNew: true,
			},

			"predefined_acl": &schema.Schema{
				Type:       schema.TypeString,
				Deprecated: "Please use resource \"storage_bucket_acl.predefined_acl\" instead.",
				Optional:   true,
				ForceNew:   true,
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

			"website": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"main_page_suffix": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"not_found_page": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceStorageBucketCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Get the bucket and acl
	bucket := d.Get("name").(string)
	location := d.Get("location").(string)

	// Create a bucket, setting the acl, location and name.
	sb := &storage.Bucket{Name: bucket, Location: location}

	if v, ok := d.GetOk("website"); ok {
		websites := v.([]interface{})

		if len(websites) > 1 {
			return fmt.Errorf("At most one website block is allowed")
		}

		sb.Website = &storage.BucketWebsite{}

		website := websites[0].(map[string]interface{})

		if v, ok := website["not_found_page"]; ok {
			sb.Website.NotFoundPage = v.(string)
		}

		if v, ok := website["main_page_suffix"]; ok {
			sb.Website.MainPageSuffix = v.(string)
		}
	}

	call := config.clientStorage.Buckets.Insert(project, sb)
	if v, ok := d.GetOk("predefined_acl"); ok {
		call = call.PredefinedAcl(v.(string))
	}

	res, err := call.Do()

	if err != nil {
		fmt.Printf("Error creating bucket %s: %v", bucket, err)
		return err
	}

	log.Printf("[DEBUG] Created bucket %v at location %v\n\n", res.Name, res.SelfLink)

	// Assign the bucket ID as the resource ID
	d.Set("self_link", res.SelfLink)
	d.SetId(res.Id)

	return nil
}

func resourceStorageBucketUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	sb := &storage.Bucket{}

	if d.HasChange("website") {
		if v, ok := d.GetOk("website"); ok {
			websites := v.([]interface{})

			if len(websites) > 1 {
				return fmt.Errorf("At most one website block is allowed")
			}

			// Setting fields to "" to be explicit that the PATCH call will
			// delete this field.
			if len(websites) == 0 {
				sb.Website.NotFoundPage = ""
				sb.Website.MainPageSuffix = ""
			} else {
				website := websites[0].(map[string]interface{})
				sb.Website = &storage.BucketWebsite{}
				if v, ok := website["not_found_page"]; ok {
					sb.Website.NotFoundPage = v.(string)
				} else {
					sb.Website.NotFoundPage = ""
				}

				if v, ok := website["main_page_suffix"]; ok {
					sb.Website.MainPageSuffix = v.(string)
				} else {
					sb.Website.MainPageSuffix = ""
				}
			}
		}
	}

	res, err := config.clientStorage.Buckets.Patch(d.Get("name").(string), sb).Do()

	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Patched bucket %v at location %v\n\n", res.Name, res.SelfLink)

	// Assign the bucket ID as the resource ID
	d.Set("self_link", res.SelfLink)
	d.SetId(res.Id)

	return nil
}

func resourceStorageBucketRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Get the bucket and acl
	bucket := d.Get("name").(string)
	res, err := config.clientStorage.Buckets.Get(bucket).Do()

	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Bucket %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading bucket %s: %v", bucket, err)
	}

	log.Printf("[DEBUG] Read bucket %v at location %v\n\n", res.Name, res.SelfLink)

	// Update the bucket ID according to the resource ID
	d.Set("self_link", res.SelfLink)
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
