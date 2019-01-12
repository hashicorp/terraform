package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRedshiftSubnetGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRedshiftSubnetGroupCreate,
		Read:   resourceAwsRedshiftSubnetGroupRead,
		Update: resourceAwsRedshiftSubnetGroupUpdate,
		Delete: resourceAwsRedshiftSubnetGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validateRedshiftSubnetGroupName,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},

			"subnet_ids": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRedshiftSubnetGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	subnetIdsSet := d.Get("subnet_ids").(*schema.Set)
	subnetIds := make([]*string, subnetIdsSet.Len())
	for i, subnetId := range subnetIdsSet.List() {
		subnetIds[i] = aws.String(subnetId.(string))
	}
	tags := tagsFromMapRedshift(d.Get("tags").(map[string]interface{}))

	createOpts := redshift.CreateClusterSubnetGroupInput{
		ClusterSubnetGroupName: aws.String(d.Get("name").(string)),
		Description:            aws.String(d.Get("description").(string)),
		SubnetIds:              subnetIds,
		Tags:                   tags,
	}

	log.Printf("[DEBUG] Create Redshift Subnet Group: %#v", createOpts)
	_, err := conn.CreateClusterSubnetGroup(&createOpts)
	if err != nil {
		return fmt.Errorf("Error creating Redshift Subnet Group: %s", err)
	}

	d.SetId(*createOpts.ClusterSubnetGroupName)
	log.Printf("[INFO] Redshift Subnet Group ID: %s", d.Id())
	return resourceAwsRedshiftSubnetGroupRead(d, meta)
}

func resourceAwsRedshiftSubnetGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	describeOpts := redshift.DescribeClusterSubnetGroupsInput{
		ClusterSubnetGroupName: aws.String(d.Id()),
	}

	describeResp, err := conn.DescribeClusterSubnetGroups(&describeOpts)
	if err != nil {
		if isAWSErr(err, "ClusterSubnetGroupNotFoundFault", "") {
			log.Printf("[INFO] Redshift Subnet Group: %s was not found", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if len(describeResp.ClusterSubnetGroups) == 0 {
		return fmt.Errorf("Unable to find Redshift Subnet Group: %#v", describeResp.ClusterSubnetGroups)
	}

	d.Set("name", d.Id())
	d.Set("description", describeResp.ClusterSubnetGroups[0].Description)
	d.Set("subnet_ids", subnetIdsToSlice(describeResp.ClusterSubnetGroups[0].Subnets))
	if err := d.Set("tags", tagsToMapRedshift(describeResp.ClusterSubnetGroups[0].Tags)); err != nil {
		return fmt.Errorf("Error setting Redshift Subnet Group Tags: %#v", err)
	}

	return nil
}

func resourceAwsRedshiftSubnetGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "redshift",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("subnetgroup:%s", d.Id()),
	}.String()
	if tagErr := setTagsRedshift(conn, d, arn); tagErr != nil {
		return tagErr
	}

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

		_, err := conn.ModifyClusterSubnetGroup(&redshift.ModifyClusterSubnetGroupInput{
			ClusterSubnetGroupName: aws.String(d.Id()),
			Description:            aws.String(d.Get("description").(string)),
			SubnetIds:              sIds,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func resourceAwsRedshiftSubnetGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	_, err := conn.DeleteClusterSubnetGroup(&redshift.DeleteClusterSubnetGroupInput{
		ClusterSubnetGroupName: aws.String(d.Id()),
	})
	if err != nil && isAWSErr(err, "ClusterSubnetGroupNotFoundFault", "") {
		return nil
	}

	return err
}

func subnetIdsToSlice(subnetIds []*redshift.Subnet) []string {
	subnetsSlice := make([]string, 0, len(subnetIds))
	for _, s := range subnetIds {
		subnetsSlice = append(subnetsSlice, *s.SubnetIdentifier)
	}
	return subnetsSlice
}

func validateRedshiftSubnetGroupName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q", k))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 255 characters", k))
	}
	if regexp.MustCompile(`(?i)^default$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q is not allowed as %q", "Default", k))
	}
	return
}
