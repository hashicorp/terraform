package google

import (
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceComputeBackendBucket() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeBackendBucketCreate,
		Read:   resourceComputeBackendBucketRead,
		Update: resourceComputeBackendBucketUpdate,
		Delete: resourceComputeBackendBucketDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					re := `^(?:[a-z](?:[-a-z0-9]{0,61}[a-z0-9])?)$`
					if !regexp.MustCompile(re).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q (%q) doesn't match regexp %q", k, value, re))
					}
					return
				},
			},

			"bucket_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"enable_cdn": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
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
		},
	}
}

func resourceComputeBackendBucketCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	bucket := compute.BackendBucket{
		Name:       d.Get("name").(string),
		BucketName: d.Get("bucket_name").(string),
	}

	if v, ok := d.GetOk("description"); ok {
		bucket.Description = v.(string)
	}

	if v, ok := d.GetOk("enable_cdn"); ok {
		bucket.EnableCdn = v.(bool)
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating new Backend Bucket: %#v", bucket)
	op, err := config.clientCompute.BackendBuckets.Insert(
		project, &bucket).Do()
	if err != nil {
		return fmt.Errorf("Error creating backend bucket: %s", err)
	}

	log.Printf("[DEBUG] Waiting for new backend bucket, operation: %#v", op)

	// Store the ID now
	d.SetId(bucket.Name)

	// Wait for the operation to complete
	waitErr := computeOperationWaitGlobal(config, op, project, "Creating Backend Bucket")
	if waitErr != nil {
		// The resource didn't actually create
		d.SetId("")
		return waitErr
	}

	return resourceComputeBackendBucketRead(d, meta)
}

func resourceComputeBackendBucketRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	bucket, err := config.clientCompute.BackendBuckets.Get(
		project, d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Backend Bucket %q", d.Get("name").(string)))
	}

	d.Set("bucket_name", bucket.BucketName)
	d.Set("description", bucket.Description)
	d.Set("enable_cdn", bucket.EnableCdn)
	d.Set("self_link", bucket.SelfLink)

	return nil
}

func resourceComputeBackendBucketUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	bucket := compute.BackendBucket{
		Name:       d.Get("name").(string),
		BucketName: d.Get("bucket_name").(string),
	}

	// Optional things
	if v, ok := d.GetOk("description"); ok {
		bucket.Description = v.(string)
	}

	if v, ok := d.GetOk("enable_cdn"); ok {
		bucket.EnableCdn = v.(bool)
	}

	log.Printf("[DEBUG] Updating existing Backend Bucket %q: %#v", d.Id(), bucket)
	op, err := config.clientCompute.BackendBuckets.Update(
		project, d.Id(), &bucket).Do()
	if err != nil {
		return fmt.Errorf("Error updating backend bucket: %s", err)
	}

	d.SetId(bucket.Name)

	err = computeOperationWaitGlobal(config, op, project, "Updating Backend Bucket")
	if err != nil {
		return err
	}

	return resourceComputeBackendBucketRead(d, meta)
}

func resourceComputeBackendBucketDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Deleting backend bucket %s", d.Id())
	op, err := config.clientCompute.BackendBuckets.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting backend bucket: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Deleting Backend Bucket")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
