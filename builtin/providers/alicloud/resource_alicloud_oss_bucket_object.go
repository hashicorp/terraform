package alicloud

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/go-homedir"
	"time"
)

func resourceAlicloudOssBucketObject() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudOssBucketObjectPut,
		Read:   resourceAlicloudOssBucketObjectRead,
		Update: resourceAlicloudOssBucketObjectPut,
		Delete: resourceAlicloudOssBucketObjectDelete,

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"key": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"source": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"content"},
			},

			"content": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"source"},
			},

			"acl": {
				Type:         schema.TypeString,
				Default:      oss.ACLPrivate,
				Optional:     true,
				ValidateFunc: validateOssBucketAcl,
			},

			"content_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"content_length": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cache_control": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_disposition": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_encoding": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"content_md5": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"expires": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"server_side_encryption": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateOssBucketObjectServerSideEncryption,
				Computed:     true,
			},

			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAlicloudOssBucketObjectPut(d *schema.ResourceData, meta interface{}) error {

	bucket, err := meta.(*AliyunClient).ossconn.Bucket(d.Get("bucket").(string))
	if err != nil {
		return fmt.Errorf("Error getting bucket: %#v", err)
	}

	var filePath string
	var body io.Reader

	if v, ok := d.GetOk("source"); ok {
		source := v.(string)
		path, err := homedir.Expand(source)
		if err != nil {
			return fmt.Errorf("Error expanding homedir in source (%s): %s", source, err)
		}

		filePath = path
	} else if v, ok := d.GetOk("content"); ok {
		content := v.(string)
		body = bytes.NewReader([]byte(content))
	} else {
		return fmt.Errorf("[ERROR] Must specify \"source\" or \"content\" field")
	}

	key := d.Get("key").(string)
	options, err := buildObjectHeaderOptions(d)
	if err != nil {
		return err
	}
	if filePath != "" {
		err = bucket.PutObjectFromFile(key, filePath, options...)
	}

	if body != nil {
		err = bucket.PutObject(key, body, options...)
	}

	if err != nil {
		return fmt.Errorf("Error putting object in Oss bucket (%#v): %s", bucket, err)
	}

	d.SetId(key)
	return resourceAlicloudOssBucketObjectRead(d, meta)
}

func resourceAlicloudOssBucketObjectRead(d *schema.ResourceData, meta interface{}) error {
	bucket, err := meta.(*AliyunClient).ossconn.Bucket(d.Get("bucket").(string))
	if err != nil {
		return fmt.Errorf("Error getting bucket: %#v", err)
	}

	options, err := buildObjectHeaderOptions(d)
	if err != nil {
		return fmt.Errorf("Error building object header options: %#v", err)
	}

	object, err := bucket.GetObjectDetailedMeta(d.Get("key").(string), options...)
	if err != nil {
		return fmt.Errorf("Error Reading Object: %#v", err)
	}

	if object == nil {
		log.Printf("[WARN] Reading Object: %#v, object does not exist", object)
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Reading Oss Bucket Object meta: %s", object)

	d.Set("content_type", object.Get("Content-Type"))
	d.Set("content_length", object.Get("Content-Length"))
	d.Set("cache_control", object.Get("Cache-Control"))
	d.Set("content_disposition", object.Get("Content-Disposition"))
	d.Set("content_encoding", object.Get("Content-Encoding"))
	d.Set("expires", object.Get("Expires"))
	d.Set("server_side_encryption", object.Get("ServerSideEncryption"))
	d.Set("etag", strings.Trim(object.Get("ETag"), `"`))

	return nil
}

func resourceAlicloudOssBucketObjectDelete(d *schema.ResourceData, meta interface{}) error {
	bucket, err := meta.(*AliyunClient).ossconn.Bucket(d.Get("bucket").(string))
	if err != nil {
		return fmt.Errorf("Error getting bucket: %#v", err)
	}
	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		exist, err := bucket.IsObjectExist(d.Id())
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("OSS delete object got an error: %#v", err))
		}

		if !exist {
			return nil
		}

		if err := bucket.DeleteObject(d.Id()); err != nil {
			return resource.RetryableError(fmt.Errorf("OSS object %#v is in use - trying again while it is deleted.", d.Id()))
		}

		return nil
	})

}

func buildObjectHeaderOptions(d *schema.ResourceData) (options []oss.Option, err error) {

	if v, ok := d.GetOk("acl"); ok {
		options = append(options, oss.ACL(oss.ACLType(v.(string))))
	}

	if v, ok := d.GetOk("content_type"); ok {
		options = append(options, oss.ContentType(v.(string)))
	}

	if v, ok := d.GetOk("cache_control"); ok {
		options = append(options, oss.CacheControl(v.(string)))
	}

	if v, ok := d.GetOk("content_disposition"); ok {
		options = append(options, oss.ContentDisposition(v.(string)))
	}

	if v, ok := d.GetOk("content_encoding"); ok {
		options = append(options, oss.ContentEncoding(v.(string)))
	}

	if v, ok := d.GetOk("content_md5"); ok {
		options = append(options, oss.ContentMD5(v.(string)))
	}

	if v, ok := d.GetOk("expires"); ok {
		options = append(options, oss.Expires(v.(time.Time)))
	}

	if v, ok := d.GetOk("server_side_encryption"); ok {
		options = append(options, oss.ServerSideEncryption(v.(string)))
	}
	if options == nil || len(options) == 0 {
		log.Printf("[WARN] Object header options is nil.")
	}
	return options, nil
}
