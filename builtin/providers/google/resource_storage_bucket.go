package google

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
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
		Importer: &schema.ResourceImporter{
			State: resourceStorageBucketStateImporter,
		},

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

			"url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"storage_class": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "STANDARD",
				ForceNew: true,
			},

			"lifecycle_rule": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"action": {
							Type:     schema.TypeSet,
							Required: true,
							MaxItems: 1,
							Set:      resourceGCSBucketLifecycleRuleActionHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"storage_class": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"condition": {
							Type:     schema.TypeSet,
							Required: true,
							MaxItems: 1,
							Set:      resourceGCSBucketLifecycleRuleConditionHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"age": {
										Type:     schema.TypeInt,
										Optional: true,
									},
									"created_before": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"is_live": {
										Type:     schema.TypeBool,
										Optional: true,
									},
									"matches_storage_class": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
									"number_of_newer_versions": {
										Type:     schema.TypeInt,
										Optional: true,
									},
								},
							},
						},
					},
				},
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

			"cors": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"origin": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"method": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"response_header": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"max_age_seconds": &schema.Schema{
							Type:     schema.TypeInt,
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

	if v, ok := d.GetOk("storage_class"); ok {
		sb.StorageClass = v.(string)
	}

	if err := resourceGCSBucketLifecycleCreateOrUpdate(d, sb, false); err != nil {
		return err
	}

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

	if v, ok := d.GetOk("cors"); ok {
		sb.Cors = expandCors(v.([]interface{}))
	}

	var res *storage.Bucket

	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		call := config.clientStorage.Buckets.Insert(project, sb)
		if v, ok := d.GetOk("predefined_acl"); ok {
			call = call.PredefinedAcl(v.(string))
		}

		res, err = call.Do()
		if err == nil {
			return nil
		}
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 429 {
			return resource.RetryableError(gerr)
		}
		return resource.NonRetryableError(err)
	})

	if err != nil {
		fmt.Printf("Error creating bucket %s: %v", bucket, err)
		return err
	}

	log.Printf("[DEBUG] Created bucket %v at location %v\n\n", res.Name, res.SelfLink)

	d.SetId(res.Id)
	return resourceStorageBucketRead(d, meta)
}

func resourceStorageBucketUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	sb := &storage.Bucket{}

	if d.HasChange("lifecycle_rule") {
		if err := resourceGCSBucketLifecycleCreateOrUpdate(d, sb, true); err != nil {
			return err
		}
	}

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

	if v, ok := d.GetOk("cors"); ok {
		sb.Cors = expandCors(v.([]interface{}))
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
		return handleNotFoundError(err, d, fmt.Sprintf("Storage Bucket %q", d.Get("name").(string)))
	}

	log.Printf("[DEBUG] Read bucket %v at location %v\n\n", res.Name, res.SelfLink)

	// Update the bucket ID according to the resource ID
	d.Set("self_link", res.SelfLink)
	d.Set("url", fmt.Sprintf("gs://%s", bucket))
	d.Set("storage_class", res.StorageClass)
	d.Set("location", res.Location)
	d.Set("cors", flattenCors(res.Cors))
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
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		err := config.clientStorage.Buckets.Delete(bucket).Do()
		if err == nil {
			return nil
		}
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 429 {
			return resource.RetryableError(gerr)
		}
		return resource.NonRetryableError(err)
	})
	if err != nil {
		fmt.Printf("Error deleting bucket %s: %v\n\n", bucket, err)
		return err
	}
	log.Printf("[DEBUG] Deleted bucket %v\n\n", bucket)

	return nil
}

func resourceStorageBucketStateImporter(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	d.Set("name", d.Id())
	return []*schema.ResourceData{d}, nil
}

func expandCors(configured []interface{}) []*storage.BucketCors {
	corsRules := make([]*storage.BucketCors, 0, len(configured))
	for _, raw := range configured {
		data := raw.(map[string]interface{})
		corsRule := storage.BucketCors{
			Origin:         convertSchemaArrayToStringArray(data["origin"].([]interface{})),
			Method:         convertSchemaArrayToStringArray(data["method"].([]interface{})),
			ResponseHeader: convertSchemaArrayToStringArray(data["response_header"].([]interface{})),
			MaxAgeSeconds:  int64(data["max_age_seconds"].(int)),
		}

		corsRules = append(corsRules, &corsRule)
	}
	return corsRules
}

func convertSchemaArrayToStringArray(input []interface{}) []string {
	output := make([]string, 0, len(input))
	for _, val := range input {
		output = append(output, val.(string))
	}

	return output
}

func flattenCors(corsRules []*storage.BucketCors) []map[string]interface{} {
	corsRulesSchema := make([]map[string]interface{}, 0, len(corsRules))
	for _, corsRule := range corsRules {
		data := map[string]interface{}{
			"origin":          corsRule.Origin,
			"method":          corsRule.Method,
			"response_header": corsRule.ResponseHeader,
			"max_age_seconds": corsRule.MaxAgeSeconds,
		}

		corsRulesSchema = append(corsRulesSchema, data)
	}
	return corsRulesSchema
}

func resourceGCSBucketLifecycleCreateOrUpdate(d *schema.ResourceData, sb *storage.Bucket, update bool) error {
	if v, ok := d.GetOk("lifecycle_rule"); ok {
		lifecycle_rules := v.([]interface{})

		if len(lifecycle_rules) > 100 {
			return fmt.Errorf("At most 100 lifecycle_rule blocks are allowed")
		}

		sb.Lifecycle = &storage.BucketLifecycle{}
		sb.Lifecycle.Rule = make([]*storage.BucketLifecycleRule, 0, len(lifecycle_rules))

		for _, lifecycle_rule := range lifecycle_rules {
			lifecycle_rule := lifecycle_rule.(map[string]interface{})

			target_lifecycle_rule := &storage.BucketLifecycleRule{}

			if v, ok := lifecycle_rule["action"]; ok {
				action := v.(*schema.Set).List()[0].(map[string]interface{})

				target_lifecycle_rule.Action = &storage.BucketLifecycleRuleAction{}

				if v, ok := action["type"]; ok {
					target_lifecycle_rule.Action.Type = v.(string)
				}

				if v, ok := action["storage_class"]; ok {
					target_lifecycle_rule.Action.StorageClass = v.(string)
				}
			}

			if v, ok := lifecycle_rule["condition"]; ok {
				condition := v.(*schema.Set).List()[0].(map[string]interface{})
				condition_elements := 0

				target_lifecycle_rule.Condition = &storage.BucketLifecycleRuleCondition{}

				if v, ok := condition["age"]; ok {
					condition_elements++
					target_lifecycle_rule.Condition.Age = int64(v.(int))
				}

				if v, ok := condition["created_before"]; ok {
					condition_elements++
					target_lifecycle_rule.Condition.CreatedBefore = v.(string)
				}

				if v, ok := condition["is_live"]; ok {
					condition_elements++
					target_lifecycle_rule.Condition.IsLive = v.(bool)
				}

				if v, ok := condition["matches_storage_class"]; ok {
					matches_storage_classes := v.([]interface{})

					target_matches_storage_classes := make([]string, 0, len(matches_storage_classes))

					for _, v := range matches_storage_classes {
						target_matches_storage_classes = append(target_matches_storage_classes, v.(string))
						condition_elements++
					}

					target_lifecycle_rule.Condition.MatchesStorageClass = target_matches_storage_classes
				}

				if v, ok := condition["number_of_newer_versions"]; ok {
					condition_elements++
					target_lifecycle_rule.Condition.NumNewerVersions = int64(v.(int))
				}

				if condition_elements < 1 {
					return fmt.Errorf("At least one condition element is required")
				}

				sb.Lifecycle.Rule = append(sb.Lifecycle.Rule, target_lifecycle_rule)
			}
		}
	}

	return nil
}

func resourceGCSBucketLifecycleRuleActionHash(v interface{}) int {
	if v == nil {
		return 0
	}

	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["type"].(string)))

	if v, ok := m["storage_class"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func resourceGCSBucketLifecycleRuleConditionHash(v interface{}) int {
	if v == nil {
		return 0
	}

	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["age"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}

	if v, ok := m["created_before"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["is_live"]; ok {
		buf.WriteString(fmt.Sprintf("%t-", v.(bool)))
	}

	if v, ok := m["matches_storage_class"]; ok {
		matches_storage_classes := v.([]interface{})
		for _, matches_storage_class := range matches_storage_classes {
			buf.WriteString(fmt.Sprintf("%s-", matches_storage_class))
		}
	}

	if v, ok := m["number_of_newer_versions"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", v.(int)))
	}

	return hashcode.String(buf.String())
}
