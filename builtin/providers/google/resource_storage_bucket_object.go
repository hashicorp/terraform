package google

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/storage/v1"
)

func resourceStorageBucketObject() *schema.Resource {
	return &schema.Resource{
		Create: resourceStorageBucketObjectCreate,
		Read:   resourceStorageBucketObjectRead,
		Delete: resourceStorageBucketObjectDelete,

		Schema: map[string]*schema.Schema{
			"bucket": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"source": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"predefined_acl": &schema.Schema{
				Type:       schema.TypeString,
				Deprecated: "Please use resource \"storage_object_acl.predefined_acl\" instead.",
				Optional:   true,
				ForceNew:   true,
			},
			"md5hash": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"crc32c": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func objectGetId(object *storage.Object) string {
	return object.Bucket + "-" + object.Name
}

func resourceStorageBucketObjectCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	bucket := d.Get("bucket").(string)
	name := d.Get("name").(string)
	source := d.Get("source").(string)

	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("Error opening %s: %s", source, err)
	}

	objectsService := storage.NewObjectsService(config.clientStorage)
	object := &storage.Object{Bucket: bucket}

	insertCall := objectsService.Insert(bucket, object)
	insertCall.Name(name)
	insertCall.Media(file)
	if v, ok := d.GetOk("predefined_acl"); ok {
		insertCall.PredefinedAcl(v.(string))
	}

	_, err = insertCall.Do()

	if err != nil {
		return fmt.Errorf("Error uploading contents of object %s from %s: %s", name, source, err)
	}

	return resourceStorageBucketObjectRead(d, meta)
}

func resourceStorageBucketObjectRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	bucket := d.Get("bucket").(string)
	name := d.Get("name").(string)

	objectsService := storage.NewObjectsService(config.clientStorage)
	getCall := objectsService.Get(bucket, name)

	res, err := getCall.Do()

	if err != nil {
		return fmt.Errorf("Error retrieving contents of object %s: %s", name, err)
	}

	d.Set("md5hash", res.Md5Hash)
	d.Set("crc32c", res.Crc32c)

	d.SetId(objectGetId(res))

	return nil
}

func resourceStorageBucketObjectDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	bucket := d.Get("bucket").(string)
	name := d.Get("name").(string)

	objectsService := storage.NewObjectsService(config.clientStorage)

	DeleteCall := objectsService.Delete(bucket, name)
	err := DeleteCall.Do()

	if err != nil {
		return fmt.Errorf("Error deleting contents of object %s: %s", name, err)
	}

	return nil
}
