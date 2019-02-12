package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/neptune"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsNeptuneSubnetGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNeptuneSubnetGroupCreate,
		Read:   resourceAwsNeptuneSubnetGroupRead,
		Update: resourceAwsNeptuneSubnetGroupUpdate,
		Delete: resourceAwsNeptuneSubnetGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
				ValidateFunc:  validateNeptuneSubnetGroupName,
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validateNeptuneSubnetGroupNamePrefix,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},

			"subnet_ids": {
				Type:     schema.TypeSet,
				Required: true,
				MinItems: 1,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsNeptuneSubnetGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn
	tags := tagsFromMapNeptune(d.Get("tags").(map[string]interface{}))

	subnetIdsSet := d.Get("subnet_ids").(*schema.Set)
	subnetIds := make([]*string, subnetIdsSet.Len())
	for i, subnetId := range subnetIdsSet.List() {
		subnetIds[i] = aws.String(subnetId.(string))
	}

	var groupName string
	if v, ok := d.GetOk("name"); ok {
		groupName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		groupName = resource.PrefixedUniqueId(v.(string))
	} else {
		groupName = resource.UniqueId()
	}

	createOpts := neptune.CreateDBSubnetGroupInput{
		DBSubnetGroupName:        aws.String(groupName),
		DBSubnetGroupDescription: aws.String(d.Get("description").(string)),
		SubnetIds:                subnetIds,
		Tags:                     tags,
	}

	log.Printf("[DEBUG] Create Neptune Subnet Group: %#v", createOpts)
	_, err := conn.CreateDBSubnetGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("Error creating Neptune Subnet Group: %s", err)
	}

	d.SetId(aws.StringValue(createOpts.DBSubnetGroupName))
	log.Printf("[INFO] Neptune Subnet Group ID: %s", d.Id())
	return resourceAwsNeptuneSubnetGroupRead(d, meta)
}

func resourceAwsNeptuneSubnetGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn

	describeOpts := neptune.DescribeDBSubnetGroupsInput{
		DBSubnetGroupName: aws.String(d.Id()),
	}

	var subnetGroups []*neptune.DBSubnetGroup
	if err := conn.DescribeDBSubnetGroupsPages(&describeOpts, func(resp *neptune.DescribeDBSubnetGroupsOutput, lastPage bool) bool {
		subnetGroups = append(subnetGroups, resp.DBSubnetGroups...)
		return !lastPage
	}); err != nil {
		if isAWSErr(err, neptune.ErrCodeDBSubnetGroupNotFoundFault, "") {
			log.Printf("[WARN] Neptune Subnet Group (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if len(subnetGroups) == 0 {
		log.Printf("[WARN] Unable to find Neptune Subnet Group: %#v, removing from state", subnetGroups)
		d.SetId("")
		return nil
	}

	var subnetGroup *neptune.DBSubnetGroup = subnetGroups[0]

	if subnetGroup.DBSubnetGroupName == nil {
		return fmt.Errorf("Unable to find Neptune Subnet Group: %#v", subnetGroups)
	}

	d.Set("name", subnetGroup.DBSubnetGroupName)
	d.Set("description", subnetGroup.DBSubnetGroupDescription)

	subnets := make([]string, 0, len(subnetGroup.Subnets))
	for _, s := range subnetGroup.Subnets {
		subnets = append(subnets, aws.StringValue(s.SubnetIdentifier))
	}
	if err := d.Set("subnet_ids", subnets); err != nil {
		return fmt.Errorf("error setting subnet_ids: %s", err)
	}

	// list tags for resource
	// set tags

	//Amazon Neptune shares the format of Amazon RDS ARNs. Neptune ARNs contain rds and not neptune.
	//https://docs.aws.amazon.com/neptune/latest/userguide/tagging.ARN.html
	d.Set("arn", subnetGroup.DBSubnetGroupArn)
	resp, err := conn.ListTagsForResource(&neptune.ListTagsForResourceInput{
		ResourceName: subnetGroup.DBSubnetGroupArn,
	})

	if err != nil {
		log.Printf("[DEBUG] Error retreiving tags for ARN: %s", aws.StringValue(subnetGroup.DBSubnetGroupArn))
	}

	d.Set("tags", tagsToMapNeptune(resp.TagList))

	return nil
}

func resourceAwsNeptuneSubnetGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn
	if d.HasChange("subnet_ids") || d.HasChange("description") {
		_, n := d.GetChange("subnet_ids")
		if n == nil {
			n = new(schema.Set)
		}
		ns := n.(*schema.Set)

		var sIds []*string
		for _, s := range ns.List() {
			sIds = append(sIds, aws.String(s.(string)))
		}

		_, err := conn.ModifyDBSubnetGroup(&neptune.ModifyDBSubnetGroupInput{
			DBSubnetGroupName:        aws.String(d.Id()),
			DBSubnetGroupDescription: aws.String(d.Get("description").(string)),
			SubnetIds:                sIds,
		})

		if err != nil {
			return err
		}
	}

	//https://docs.aws.amazon.com/neptune/latest/userguide/tagging.ARN.html
	arn := d.Get("arn").(string)
	if err := setTagsNeptune(conn, d, arn); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	return resourceAwsNeptuneSubnetGroupRead(d, meta)
}

func resourceAwsNeptuneSubnetGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn

	input := neptune.DeleteDBSubnetGroupInput{
		DBSubnetGroupName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting Neptune Subnet Group: %s", d.Id())
	_, err := conn.DeleteDBSubnetGroup(&input)
	if err != nil {
		if isAWSErr(err, neptune.ErrCodeDBSubnetGroupNotFoundFault, "") {
			return nil
		}
		return fmt.Errorf("error deleting Neptune Subnet Group (%s): %s", d.Id(), err)
	}

	return nil
}
