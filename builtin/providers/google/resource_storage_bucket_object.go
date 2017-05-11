package google

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/googleapi"
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

			"cache_control": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"content_disposition": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"content_encoding": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"content_language": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"content_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"content": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"source"},
			},

			"crc32c": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"md5hash": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"predefined_acl": &schema.Schema{
				Type:       schema.TypeString,
				Deprecated: "Please use resource \"storage_object_acl.predefined_acl\" instead.",
				Optional:   true,
				ForceNew:   true,
			},

			"source": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"content"},
			},

			"storage_class": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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
	var media io.Reader

	if v, ok := d.GetOk("source"); ok {
		err := error(nil)
		media, err = os.Open(v.(string))
		if err != nil {
			return err
		}
	} else if v, ok := d.GetOk("content"); ok {
		media = bytes.NewReader([]byte(v.(string)))
	} else {
		return fmt.Errorf("Error, either \"content\" or \"string\" must be specified")
	}

	objectsService := storage.NewObjectsService(config.clientStorage)
	object := &storage.Object{Bucket: bucket}

	if v, ok := d.GetOk("cache_control"); ok {
		object.CacheControl = v.(string)
	}

	if v, ok := d.GetOk("content_disposition"); ok {
		object.ContentDisposition = v.(string)
	}

	if v, ok := d.GetOk("content_encoding"); ok {
		object.ContentEncoding = v.(string)
	}

	if v, ok := d.GetOk("content_language"); ok {
		object.ContentLanguage = v.(string)
	}

	if v, ok := d.GetOk("content_type"); ok {
		object.ContentType = v.(string)
	}

	if v, ok := d.GetOk("storage_class"); ok {
		object.StorageClass = v.(string)
	}

	insertCall := objectsService.Insert(bucket, object)
	insertCall.Name(name)
	insertCall.Media(media)
	if v, ok := d.GetOk("predefined_acl"); ok {
		insertCall.PredefinedAcl(v.(string))
	}

	_, err := insertCall.Do()

	if err != nil {
		return fmt.Errorf("Error uploading object %s: %s", name, err)
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
		return handleNotFoundError(err, d, fmt.Sprintf("Storage Bucket Object %q", d.Get("name").(string)))
	}

	d.Set("md5hash", res.Md5Hash)
	d.Set("crc32c", res.Crc32c)
	d.Set("cache_control", res.CacheControl)
	d.Set("content_disposition", res.ContentDisposition)
	d.Set("content_encoding", res.ContentEncoding)
	d.Set("content_language", res.ContentLanguage)
	d.Set("content_type", res.ContentType)
	d.Set("storage_class", res.StorageClass)

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
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Bucket Object %q because it's gone", name)
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error deleting contents of object %s: %s", name, err)
	}

	return nil
}
