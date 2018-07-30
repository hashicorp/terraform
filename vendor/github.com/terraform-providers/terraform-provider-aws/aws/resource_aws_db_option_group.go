package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDbOptionGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbOptionGroupCreate,
		Read:   resourceAwsDbOptionGroupRead,
		Update: resourceAwsDbOptionGroupUpdate,
		Delete: resourceAwsDbOptionGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(15 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateDbOptionGroupName,
			},
			"name_prefix": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateDbOptionGroupNamePrefix,
			},
			"engine_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"major_engine_version": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"option_group_description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "Managed by Terraform",
			},

			"option": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"option_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"option_settings": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"value": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"port": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"db_security_group_memberships": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"vpc_security_group_memberships": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
					},
				},
				Set: resourceAwsDbOptionHash,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDbOptionGroupCreate(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	var groupName string
	if v, ok := d.GetOk("name"); ok {
		groupName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		groupName = resource.PrefixedUniqueId(v.(string))
	} else {
		groupName = resource.UniqueId()
	}

	createOpts := &rds.CreateOptionGroupInput{
		EngineName:             aws.String(d.Get("engine_name").(string)),
		MajorEngineVersion:     aws.String(d.Get("major_engine_version").(string)),
		OptionGroupDescription: aws.String(d.Get("option_group_description").(string)),
		OptionGroupName:        aws.String(groupName),
		Tags:                   tags,
	}

	log.Printf("[DEBUG] Create DB Option Group: %#v", createOpts)
	_, err := rdsconn.CreateOptionGroup(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating DB Option Group: %s", err)
	}

	d.SetId(strings.ToLower(groupName))
	log.Printf("[INFO] DB Option Group ID: %s", d.Id())

	return resourceAwsDbOptionGroupUpdate(d, meta)
}

func resourceAwsDbOptionGroupRead(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn
	params := &rds.DescribeOptionGroupsInput{
		OptionGroupName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Describe DB Option Group: %#v", params)
	options, err := rdsconn.DescribeOptionGroups(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if "OptionGroupNotFoundFault" == awsErr.Code() {
				d.SetId("")
				log.Printf("[DEBUG] DB Option Group (%s) not found", d.Get("name").(string))
				return nil
			}
		}
		return fmt.Errorf("Error Describing DB Option Group: %s", err)
	}

	var option *rds.OptionGroup
	for _, ogl := range options.OptionGroupsList {
		if *ogl.OptionGroupName == d.Id() {
			option = ogl
			break
		}
	}

	if option == nil {
		return fmt.Errorf("Unable to find Option Group: %#v", options.OptionGroupsList)
	}

	d.Set("name", option.OptionGroupName)
	d.Set("major_engine_version", option.MajorEngineVersion)
	d.Set("engine_name", option.EngineName)
	d.Set("option_group_description", option.OptionGroupDescription)
	if len(option.Options) != 0 {
		d.Set("option", flattenOptions(option.Options))
	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "rds",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("og:%s", d.Id()),
	}.String()
	d.Set("arn", arn)
	resp, err := rdsconn.ListTagsForResource(&rds.ListTagsForResourceInput{
		ResourceName: aws.String(arn),
	})

	if err != nil {
		log.Printf("[DEBUG] Error retrieving tags for ARN: %s", arn)
	}

	var dt []*rds.Tag
	if len(resp.TagList) > 0 {
		dt = resp.TagList
	}
	d.Set("tags", tagsToMapRDS(dt))

	return nil
}

func optionInList(optionName string, list []*string) bool {
	for _, opt := range list {
		if *opt == optionName {
			return true
		}
	}
	return false
}

func resourceAwsDbOptionGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn
	if d.HasChange("option") {
		o, n := d.GetChange("option")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		addOptions, addErr := expandOptionConfiguration(ns.Difference(os).List())
		if addErr != nil {
			return addErr
		}

		addingOptionNames, err := flattenOptionNames(ns.Difference(os).List())
		if err != nil {
			return err
		}

		removeOptions := []*string{}
		opts, err := flattenOptionNames(os.Difference(ns).List())
		if err != nil {
			return err
		}

		for _, optionName := range opts {
			if optionInList(*optionName, addingOptionNames) {
				continue
			}
			removeOptions = append(removeOptions, optionName)
		}

		modifyOpts := &rds.ModifyOptionGroupInput{
			OptionGroupName:  aws.String(d.Id()),
			ApplyImmediately: aws.Bool(true),
		}

		if len(addOptions) > 0 {
			modifyOpts.OptionsToInclude = addOptions
		}

		if len(removeOptions) > 0 {
			modifyOpts.OptionsToRemove = removeOptions
		}

		log.Printf("[DEBUG] Modify DB Option Group: %s", modifyOpts)
		_, err = rdsconn.ModifyOptionGroup(modifyOpts)
		if err != nil {
			return fmt.Errorf("Error modifying DB Option Group: %s", err)
		}
		d.SetPartial("option")

	}

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "rds",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("og:%s", d.Id()),
	}.String()
	if err := setTagsRDS(rdsconn, d, arn); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	return resourceAwsDbOptionGroupRead(d, meta)
}

func resourceAwsDbOptionGroupDelete(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn

	deleteOpts := &rds.DeleteOptionGroupInput{
		OptionGroupName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Delete DB Option Group: %#v", deleteOpts)
	ret := resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		_, err := rdsconn.DeleteOptionGroup(deleteOpts)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "InvalidOptionGroupStateFault" {
					log.Printf("[DEBUG] AWS believes the RDS Option Group is still in use, retrying")
					return resource.RetryableError(awsErr)
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if ret != nil {
		return fmt.Errorf("Error Deleting DB Option Group: %s", ret)
	}
	return nil
}

func flattenOptionNames(configured []interface{}) ([]*string, error) {
	var optionNames []*string
	for _, pRaw := range configured {
		data := pRaw.(map[string]interface{})
		optionNames = append(optionNames, aws.String(data["option_name"].(string)))
	}

	return optionNames, nil
}

func resourceAwsDbOptionHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["option_name"].(string)))
	if _, ok := m["port"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", m["port"].(int)))
	}

	for _, oRaw := range m["option_settings"].(*schema.Set).List() {
		o := oRaw.(map[string]interface{})
		buf.WriteString(fmt.Sprintf("%s-", o["name"].(string)))
		buf.WriteString(fmt.Sprintf("%s-", o["value"].(string)))
	}

	for _, vpcRaw := range m["vpc_security_group_memberships"].(*schema.Set).List() {
		buf.WriteString(fmt.Sprintf("%s-", vpcRaw.(string)))
	}

	for _, sgRaw := range m["db_security_group_memberships"].(*schema.Set).List() {
		buf.WriteString(fmt.Sprintf("%s-", sgRaw.(string)))
	}
	return hashcode.String(buf.String())
}
