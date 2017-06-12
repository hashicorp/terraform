package alicloud

import (
	"bytes"
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"time"
)

func resourceAlicloudOssBucket() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudOssBucketCreate,
		Read:   resourceAlicloudOssBucketRead,
		Update: resourceAlicloudOssBucketUpdate,
		Delete: resourceAlicloudOssBucketDelete,

		Schema: map[string]*schema.Schema{
			"bucket": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateOssBucketName,
				Default:      resource.PrefixedUniqueId("tf-oss-bucket-"),
			},

			"acl": &schema.Schema{
				Type:         schema.TypeString,
				Default:      oss.ACLPrivate,
				Optional:     true,
				ValidateFunc: validateOssBucketAcl,
			},

			"cors_rule": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allowed_headers": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"allowed_methods": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"allowed_origins": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"expose_headers": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"max_age_seconds": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
				MaxItems: 10,
			},

			"website": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"index_document": {
							Type:     schema.TypeString,
							Required: true,
						},

						"error_document": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				MaxItems: 1,
			},

			"logging": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_bucket": {
							Type:     schema.TypeString,
							Required: true,
						},
						"target_prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["target_bucket"]))
					buf.WriteString(fmt.Sprintf("%s-", m["target_prefix"]))
					return hashcode.String(buf.String())
				},
				MaxItems: 1,
			},

			"logging_isenable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"referer_config": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"allow_empty": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"referers": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
				MaxItems: 1,
			},

			"lifecycle_rule": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:         schema.TypeString,
							Optional:     true,
							Computed:     true,
							ValidateFunc: validateOssBucketLifecycleRuleId,
						},
						"prefix": {
							Type:     schema.TypeString,
							Required: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"expiration": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"date": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateOssBucketDateTimestamp,
									},
									"days": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},
					},
				},
				MaxItems: 1000,
			},

			"creation_date": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"extranet_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"intranet_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"location": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_class": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAlicloudOssBucketCreate(d *schema.ResourceData, meta interface{}) error {
	ossconn := meta.(*AliyunClient).ossconn

	bucket := d.Get("bucket").(string)
	isExist, err := ossconn.IsBucketExist(bucket)
	if err != nil {
		return err
	}
	if isExist {
		return fmt.Errorf("[ERROR] The specified bucket name: %#v is not available. The bucket namespace is shared by all users of the OSS system. Please select a different name and try again.", bucket)
	}

	log.Printf("[DEBUG] OSS bucket create: %#v, using endpoint: %#v", bucket, ossconn.Config.Endpoint)

	retryErr := resource.Retry(3*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] Trying to create new OSS bucket: %#v", bucket)
		err := ossconn.CreateBucket(bucket)

		if err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if retryErr != nil {
		return fmt.Errorf("Error creating OSS bucket: %#v", retryErr)
	}

	// Assign the bucket name as the resource ID
	d.SetId(bucket)

	return resourceAlicloudOssBucketUpdate(d, meta)
}

func resourceAlicloudOssBucketRead(d *schema.ResourceData, meta interface{}) error {
	ossconn := meta.(*AliyunClient).ossconn

	info, err := ossconn.GetBucketInfo(d.Id())
	if err != nil {
		if ossNotFoundError(err) {
			return nil
		}
		return err
	}

	d.Set("bucket", d.Id())

	d.Set("acl", info.BucketInfo.ACL)
	d.Set("creation_date", info.BucketInfo.CreationDate.Format("2016-01-01"))
	d.Set("extranet_endpoint", info.BucketInfo.ExtranetEndpoint)
	d.Set("intranet_endpoint", info.BucketInfo.IntranetEndpoint)
	d.Set("location", info.BucketInfo.Location)
	d.Set("owner", info.BucketInfo.Owner.ID)
	d.Set("storage_class", info.BucketInfo.StorageClass)

	// Read the CORS
	cors, err := ossconn.GetBucketCORS(d.Id())
	if err != nil {
		if ossNotFoundError(err) {
			log.Printf("[WARN] OSS bucket: %s, no CORS rule configuration could be found.", d.Id())
			return nil
		}
		return err
	}
	if cors.CORSRules != nil {
		rules := make([]map[string]interface{}, 0, len(cors.CORSRules))
		for _, r := range cors.CORSRules {
			rule := make(map[string]interface{})
			rule["allowed_headers"] = r.AllowedHeader
			rule["allowed_methods"] = r.AllowedMethod
			rule["allowed_origins"] = r.AllowedOrigin
			rule["expose_headers"] = r.ExposeHeader
			rule["max_age_seconds"] = r.MaxAgeSeconds

			rules = append(rules, rule)
		}
		if err := d.Set("cors_rule", rules); err != nil {
			return err
		}
	}

	// Read the website configuration
	ws, err := ossconn.GetBucketWebsite(d.Id())
	if err != nil {
		if ossNotFoundError(err) {
			log.Printf("[WARN] OSS bucket: %s, no website could be found.", d.Id())
			return nil
		}
		return fmt.Errorf("Error getting bucket website: %#v", err)
	}
	var websites []map[string]interface{}
	w := make(map[string]interface{})

	if v := &ws.IndexDocument; v != nil {
		w["index_document"] = v.Suffix
	}

	if v := &ws.ErrorDocument; v != nil {
		w["error_document"] = v.Key
	}
	websites = append(websites, w)
	if err := d.Set("website", websites); err != nil {
		return err
	}

	// Read the logging configuration
	logging, err := ossconn.GetBucketLogging(d.Id())
	if err != nil {
		if ossNotFoundError(err) {
			log.Printf("[WARN] OSS bucket: %s, no logging could be found.", d.Id())
			return nil
		}
		return fmt.Errorf("Error getting bucket logging: %#v", err)
	}

	if isEnable, ok := d.GetOk("logging_isenable"); ok {
		d.Set("logging_isenable", isEnable.(bool))
		if !isEnable.(bool) {
			d.Set("logging", logging.XMLName)
		} else {
			lgs := make([]map[string]interface{}, 0, 1)
			if &logging != nil {
				lg := make(map[string]interface{})
				// Target bucket
				if v := logging.LoggingEnabled.TargetBucket; v != "" {
					lg["target_bucket"] = v
				}
				// Target prefix
				if v := logging.LoggingEnabled.TargetPrefix; v != "" {
					lg["target_prefix"] = v
				}

				lgs = append(lgs, lg)
				if err := d.Set("logging", lgs); err != nil {
					return err
				}
			}
		}
	}

	// Read the bucket referer
	referer, err := ossconn.GetBucketReferer(d.Id())
	var referers []map[string]interface{}
	if err != nil {
		if ossNotFoundError(err) {
			log.Printf("[WARN] OSS bucket: %s, no referer configuration could be found.", d.Id())
			return nil
		}
		return fmt.Errorf("Error getting bucket referer: %#v", err)
	}
	if referers != nil && len(referers) > 0 {
		rf := make(map[string]interface{})
		// Allow empty
		if v := referer.AllowEmptyReferer; &v != nil {
			rf["allow_empty"] = v
		}
		// Referers
		if v := referer.RefererList; &v != nil {
			rf["referers"] = v
		}

		referers = append(referers, rf)
		if err := d.Set("referer_config", referers); err != nil {
			return err
		}
	}

	// Read the lifecycle rule configuration
	lifecycle, err := ossconn.GetBucketLifecycle(d.Id())
	if err != nil {
		if ossNotFoundError(err) {
			log.Printf("[WARN] OSS bucket: %s, no lifecycle could be found.", d.Id())
			return nil
		}
		return fmt.Errorf("Error getting bucket lifecycle: %#v", err)
	}
	if len(lifecycle.Rules) > 0 {
		rules := make([]map[string]interface{}, 0, len(lifecycle.Rules))

		for _, lifecycleRule := range lifecycle.Rules {
			rule := make(map[string]interface{})

			// ID
			if &lifecycleRule.ID != nil && lifecycleRule.ID != "" {
				rule["id"] = lifecycleRule.ID
			}
			// Prefix
			if &lifecycleRule.Prefix != nil && lifecycleRule.Prefix != "" {
				rule["prefix"] = lifecycleRule.Prefix
			}
			// Enabled
			if &lifecycleRule.Status != nil {
				if LifecycleRuleStatus(lifecycleRule.Status) == ExpirationStatusEnabled {
					rule["enabled"] = true
				} else {
					rule["enabled"] = false
				}
			}
			// expiration
			if &lifecycleRule.Expiration != nil {
				e := make(map[string]interface{})
				if &lifecycleRule.Expiration.Date != nil {
					e["date"] = (lifecycleRule.Expiration.Date).Format("2016-01-01")
				}
				if &lifecycleRule.Expiration.Days != nil {
					e["days"] = int(lifecycleRule.Expiration.Days)
				}
			}
			rules = append(rules, rule)
		}

		if err := d.Set("lifecycle_rule", rules); err != nil {
			return err
		}
	}

	return nil
}

func resourceAlicloudOssBucketUpdate(d *schema.ResourceData, meta interface{}) error {
	ossconn := meta.(*AliyunClient).ossconn

	d.Partial(true)

	if d.HasChange("acl") {
		if err := ossconn.SetBucketACL(d.Id(), oss.ACLType(d.Get("acl").(string))); err != nil {
			return fmt.Errorf("Error setting OSS bucket ACL: %#v", err)
		}
		d.SetPartial("acl")
	}

	if d.HasChange("cors_rule") {
		if err := resourceAlicloudOssBucketCorsUpdate(ossconn, d); err != nil {
			return err
		}
	}

	if d.HasChange("website") {
		if err := resourceAlicloudOssBucketWebsiteUpdate(ossconn, d); err != nil {
			return err
		}
	}

	if d.HasChange("logging") {
		if err := resourceAlicloudOssBucketLoggingUpdate(ossconn, d); err != nil {
			return err
		}
	}

	if d.HasChange("referer_config") {
		if err := resourceAlicloudOssBucketRefererUpdate(ossconn, d); err != nil {
			return err
		}
	}

	if d.HasChange("lifecycle_rule") {
		if err := resourceAlicloudOssBucketLifecycleRuleUpdate(ossconn, d); err != nil {
			return err
		}
	}

	d.Partial(false)
	return resourceAlicloudOssBucketRead(d, meta)
}
func resourceAlicloudOssBucketCorsUpdate(ossconn *oss.Client, d *schema.ResourceData) error {
	cors := d.Get("cors_rule").([]interface{})
	if cors == nil || len(cors) == 0 {
		err := resource.Retry(3*time.Minute, func() *resource.RetryError {
			if err := ossconn.DeleteBucketCORS(d.Id()); err != nil {
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error removing OSS bucket cors_rule: %#v", err)
		}
		return nil
	}
	// Put CORS
	rules := make([]oss.CORSRule, 0, len(cors))
	for _, c := range cors {
		corsMap := c.(map[string]interface{})
		rule := oss.CORSRule{}
		for k, v := range corsMap {
			log.Printf("[DEBUG] OSS bucket: %s, put CORS: %#v, %#v", d.Id(), k, v)
			if k == "max_age_seconds" {
				rule.MaxAgeSeconds = v.(int)
			} else {
				rMap := make([]string, len(v.([]interface{})))
				for i, vv := range v.([]interface{}) {
					rMap[i] = vv.(string)
				}
				switch k {
				case "allowed_headers":
					rule.AllowedHeader = rMap
				case "allowed_methods":
					rule.AllowedMethod = rMap
				case "allowed_origins":
					rule.AllowedOrigin = rMap
				case "expose_headers":
					rule.ExposeHeader = rMap
				}
			}
		}
		rules = append(rules, rule)
	}

	log.Printf("[DEBUG] Oss bucket: %s, put CORS: %#v", d.Id(), cors)
	err := ossconn.SetBucketCORS(d.Id(), rules)
	if err != nil {
		return fmt.Errorf("Error putting oss CORS: %s", err)
	}

	return nil
}
func resourceAlicloudOssBucketWebsiteUpdate(ossconn *oss.Client, d *schema.ResourceData) error {
	ws := d.Get("website").(*schema.Set)
	if ws == nil || ws.Len() == 0 {
		err := resource.Retry(3*time.Minute, func() *resource.RetryError {
			if err := ossconn.DeleteBucketWebsite(d.Id()); err != nil {
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error removing OSS bucket logging: %#v", err)
		}
		return nil
	}

	var index_document, error_document string
	w := ws.List()[0].(map[string]interface{})

	if v, ok := w["index_document"]; ok {
		index_document = v.(string)
	}
	if v, ok := w["error_document"]; ok {
		error_document = v.(string)
	}
	if err := ossconn.SetBucketWebsite(d.Id(), index_document, error_document); err != nil {
		return fmt.Errorf("Error putting OSS bucket website: %#v", err)
	}

	return nil
}

func resourceAlicloudOssBucketLoggingUpdate(ossconn *oss.Client, d *schema.ResourceData) error {
	logging := d.Get("logging").(*schema.Set)
	if logging == nil || logging.Len() == 0 {
		err := resource.Retry(3*time.Minute, func() *resource.RetryError {
			if err := ossconn.DeleteBucketLogging(d.Id()); err != nil {
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error removing OSS bucket logging: %#v", err)
		}
		return nil
	}

	c := logging.List()[0].(map[string]interface{})
	var target_bucket, target_prefix string
	if v, ok := c["target_bucket"]; ok {
		target_bucket = v.(string)
	}
	if v, ok := c["target_prefix"]; ok {
		target_prefix = v.(string)
	}
	if err := ossconn.SetBucketLogging(d.Id(), target_bucket, target_prefix, d.Get("logging_isenable").(bool)); err != nil {
		return fmt.Errorf("Error putting OSS bucket logging: %#v", err)
	}

	return nil
}

func resourceAlicloudOssBucketRefererUpdate(ossconn *oss.Client, d *schema.ResourceData) error {
	config := d.Get("referer_config").(*schema.Set)
	if config == nil || config.Len() == 0 {
		log.Printf("[DEBUG] OSS set bucket referer as nil")
		if err := ossconn.SetBucketReferer(d.Id(), nil, true); err != nil {
			return fmt.Errorf("Error deleting OSS website: %#v", err)
		}
		return nil
	}

	c := config.List()[0].(map[string]interface{})

	var allow bool
	var referers []string
	if v, ok := c["allow_empty"]; ok {
		allow = v.(bool)
	}
	if v, ok := c["referers"]; ok {
		for _, referer := range v.([]interface{}) {
			referers = append(referers, referer.(string))
		}
	}
	if err := ossconn.SetBucketReferer(d.Id(), referers, allow); err != nil {
		return fmt.Errorf("Error putting OSS bucket referer configuration: %#v", err)
	}

	return nil
}
func resourceAlicloudOssBucketLifecycleRuleUpdate(ossconn *oss.Client, d *schema.ResourceData) error {
	bucket := d.Id()
	lifecycleRules := d.Get("lifecycle_rule").([]interface{})

	if lifecycleRules == nil || len(lifecycleRules) == 0 {
		err := resource.Retry(3*time.Minute, func() *resource.RetryError {
			if err := ossconn.DeleteBucketLifecycle(bucket); err != nil {
				return resource.NonRetryableError(err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Error removing OSS bucket lifecycle: %#v", err)
		}
		return nil
	}

	rules := make([]oss.LifecycleRule, 0, len(lifecycleRules))

	for i, lifecycleRule := range lifecycleRules {
		r := lifecycleRule.(map[string]interface{})

		rule := oss.LifecycleRule{
			Prefix: r["prefix"].(string),
		}

		// ID
		if val, ok := r["id"].(string); ok && val != "" {
			rule.ID = val
		}

		// Enabled
		if val, ok := r["enabled"].(bool); ok && val {
			rule.Status = string(ExpirationStatusEnabled)
		} else {
			rule.Status = string(ExpirationStatusDisabled)
		}

		// Expiration
		expiration := d.Get(fmt.Sprintf("lifecycle_rule.%d.expiration", i)).(*schema.Set).List()
		if len(expiration) > 0 {
			e := expiration[0].(map[string]interface{})
			i := oss.LifecycleExpiration{}

			if val, ok := e["date"].(string); ok && val != "" {
				t, err := time.Parse(time.RFC3339, fmt.Sprintf("%sT00:00:00Z", val))
				if err != nil {
					return fmt.Errorf("Error Parsing Alicloud OSS Bucket Lifecycle Expiration Date: %s", err.Error())
				}
				i.Date = time.Time(t)
			} else if val, ok := e["days"].(int); ok && val > 0 {
				i.Days = int(val)
			}
			rule.Expiration = i
		}
		rules = append(rules, rule)
	}

	err := resource.Retry(3*time.Minute, func() *resource.RetryError {
		if err := ossconn.SetBucketLifecycle(bucket, rules); err != nil {
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Error putting OSS lifecycle rule: %#v", err)
	}

	return nil
}
func resourceAlicloudOssBucketDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient).ossconn

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		exist, err := client.IsBucketExist(d.Id())
		if err != nil {
			return resource.NonRetryableError(fmt.Errorf("OSS delete bucket got an error: %#v", err))
		}

		if !exist {
			return nil
		}

		if err := client.DeleteBucket(d.Id()); err != nil {
			return resource.RetryableError(fmt.Errorf("OSS Bucket %#v is in use - trying again while it is deleted.", d.Id()))
		}

		return nil
	})
}
