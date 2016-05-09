package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDbOptionGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbOptionGroupCreate,
		Read:   resourceAwsDbOptionGroupRead,
		Update: resourceAwsDbOptionGroupUpdate,
		Delete: resourceAwsDbOptionGroupDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validateDbOptionGroupName,
			},
			"engine_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"major_engine_version": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"option_group_description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"option": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"option_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"db_security_group_memberships": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"vpc_security_group_memberships": &schema.Schema{
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

	createOpts := &rds.CreateOptionGroupInput{
		EngineName:             aws.String(d.Get("engine_name").(string)),
		MajorEngineVersion:     aws.String(d.Get("major_engine_version").(string)),
		OptionGroupDescription: aws.String(d.Get("option_group_description").(string)),
		OptionGroupName:        aws.String(d.Get("name").(string)),
		Tags:                   tags,
	}

	log.Printf("[DEBUG] Create DB Option Group: %#v", createOpts)
	_, err := rdsconn.CreateOptionGroup(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating DB Option Group: %s", err)
	}

	d.SetId(d.Get("name").(string))
	log.Printf("[INFO] DB Option Group ID: %s", d.Id())

	return resourceAwsDbOptionGroupUpdate(d, meta)
}

func resourceAwsDbOptionGroupRead(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn
	params := &rds.DescribeOptionGroupsInput{
		OptionGroupName: aws.String(d.Get("name").(string)),
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
		if *ogl.OptionGroupName == d.Get("name").(string) {
			option = ogl
			break
		}
	}

	if option == nil {
		return fmt.Errorf("Unable to find Option Group: %#v", options.OptionGroupsList)
	}

	d.Set("major_engine_version", option.MajorEngineVersion)
	d.Set("engine_name", option.EngineName)
	d.Set("option_group_description", option.OptionGroupDescription)
	if len(option.Options) != 0 {
		d.Set("option", flattenOptions(option.Options))
	}

	optionGroup := options.OptionGroupsList[0]
	arn, err := buildRDSOptionGroupARN(d.Id(), meta.(*AWSClient).accountid, meta.(*AWSClient).region)
	if err != nil {
		name := "<empty>"
		if optionGroup.OptionGroupName != nil && *optionGroup.OptionGroupName != "" {
			name = *optionGroup.OptionGroupName
		}
		log.Printf("[DEBUG] Error building ARN for DB Option Group, not setting Tags for Option Group %s", name)
	} else {
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
	}

	return nil
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

		removeOptions, removeErr := flattenOptionConfigurationNames(os.Difference(ns).List())
		if removeErr != nil {
			return removeErr
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
		_, err := rdsconn.ModifyOptionGroup(modifyOpts)
		if err != nil {
			return fmt.Errorf("Error modifying DB Option Group: %s", err)
		}
		d.SetPartial("option")

	}

	if arn, err := buildRDSOptionGroupARN(d.Id(), meta.(*AWSClient).accountid, meta.(*AWSClient).region); err == nil {
		if err := setTagsRDS(rdsconn, d, arn); err != nil {
			return err
		} else {
			d.SetPartial("tags")
		}
	}

	return resourceAwsDbOptionGroupRead(d, meta)
}

func resourceAwsDbOptionGroupDelete(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn

	deleteOpts := &rds.DeleteOptionGroupInput{
		OptionGroupName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Delete DB Option Group: %#v", deleteOpts)
	_, err := rdsconn.DeleteOptionGroup(deleteOpts)
	if err != nil {
		return fmt.Errorf("Error Deleting DB Option Group: %s", err)
	}

	return nil
}

func flattenOptionConfigurationNames(configured []interface{}) ([]*string, error) {
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

	return hashcode.String(buf.String())
}

func buildRDSOptionGroupARN(identifier, accountid, region string) (string, error) {
	if accountid == "" {
		return "", fmt.Errorf("Unable to construct RDS Option Group ARN because of missing AWS Account ID")
	}
	arn := fmt.Sprintf("arn:aws:rds:%s:%s:og:%s", region, accountid, identifier)
	return arn, nil
}

func validateDbOptionGroupName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if !regexp.MustCompile(`^[0-9A-Za-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters and hyphens allowed in %q", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot end with a hyphen", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be greater than 255 characters", k))
	}
	return
}
