package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDbSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbSecurityGroupCreate,
		Read:   resourceAwsDbSecurityGroupRead,
		Update: resourceAwsDbSecurityGroupUpdate,
		Delete: resourceAwsDbSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ingress": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidr": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},

						"security_group_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"security_group_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"security_group_owner_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceAwsDbSecurityGroupIngressHash,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDbSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	var err error
	var errs []error

	opts := rds.CreateDBSecurityGroupInput{
		DBSecurityGroupName:        aws.String(d.Get("name").(string)),
		DBSecurityGroupDescription: aws.String(d.Get("description").(string)),
		Tags: tags,
	}

	log.Printf("[DEBUG] DB Security Group create configuration: %#v", opts)
	_, err = conn.CreateDBSecurityGroup(&opts)
	if err != nil {
		return fmt.Errorf("Error creating DB Security Group: %s", err)
	}

	d.SetId(d.Get("name").(string))

	log.Printf("[INFO] DB Security Group ID: %s", d.Id())

	sg, err := resourceAwsDbSecurityGroupRetrieve(d, meta)
	if err != nil {
		return err
	}

	ingresses := d.Get("ingress").(*schema.Set)
	for _, ing := range ingresses.List() {
		err := resourceAwsDbSecurityGroupAuthorizeRule(ing, *sg.DBSecurityGroupName, conn)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	log.Println(
		"[INFO] Waiting for Ingress Authorizations to be authorized")

	stateConf := &resource.StateChangeConf{
		Pending: []string{"authorizing"},
		Target:  []string{"authorized"},
		Refresh: resourceAwsDbSecurityGroupStateRefreshFunc(d, meta),
		Timeout: 10 * time.Minute,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsDbSecurityGroupRead(d, meta)
}

func resourceAwsDbSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	sg, err := resourceAwsDbSecurityGroupRetrieve(d, meta)
	if err != nil {
		return err
	}

	d.Set("name", *sg.DBSecurityGroupName)
	d.Set("description", *sg.DBSecurityGroupDescription)

	// Create an empty schema.Set to hold all ingress rules
	rules := &schema.Set{
		F: resourceAwsDbSecurityGroupIngressHash,
	}

	for _, v := range sg.IPRanges {
		rule := map[string]interface{}{"cidr": *v.CIDRIP}
		rules.Add(rule)
	}

	for _, g := range sg.EC2SecurityGroups {
		rule := map[string]interface{}{
			"security_group_name":     *g.EC2SecurityGroupName,
			"security_group_id":       *g.EC2SecurityGroupId,
			"security_group_owner_id": *g.EC2SecurityGroupOwnerId,
		}
		rules.Add(rule)
	}

	d.Set("ingress", rules)

	conn := meta.(*AWSClient).rdsconn
	arn, err := buildRDSSecurityGroupARN(d, meta)
	if err != nil {
		name := "<empty>"
		if sg.DBSecurityGroupName != nil && *sg.DBSecurityGroupName != "" {
			name = *sg.DBSecurityGroupName
		}
		log.Printf("[DEBUG] Error building ARN for DB Security Group, not setting Tags for DB Security Group %s", name)
	} else {
		d.Set("arn", arn)
		resp, err := conn.ListTagsForResource(&rds.ListTagsForResourceInput{
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

func resourceAwsDbSecurityGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	d.Partial(true)
	if arn, err := buildRDSSecurityGroupARN(d, meta); err == nil {
		if err := setTagsRDS(conn, d, arn); err != nil {
			return err
		} else {
			d.SetPartial("tags")
		}
	}
	d.Partial(false)

	return resourceAwsDbSecurityGroupRead(d, meta)
}

func resourceAwsDbSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	log.Printf("[DEBUG] DB Security Group destroy: %v", d.Id())

	opts := rds.DeleteDBSecurityGroupInput{DBSecurityGroupName: aws.String(d.Id())}

	log.Printf("[DEBUG] DB Security Group destroy configuration: %v", opts)
	_, err := conn.DeleteDBSecurityGroup(&opts)

	if err != nil {
		newerr, ok := err.(awserr.Error)
		if ok && newerr.Code() == "InvalidDBSecurityGroup.NotFound" {
			return nil
		}
		return err
	}

	return nil
}

func resourceAwsDbSecurityGroupRetrieve(d *schema.ResourceData, meta interface{}) (*rds.DBSecurityGroup, error) {
	conn := meta.(*AWSClient).rdsconn

	opts := rds.DescribeDBSecurityGroupsInput{
		DBSecurityGroupName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] DB Security Group describe configuration: %#v", opts)

	resp, err := conn.DescribeDBSecurityGroups(&opts)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving DB Security Groups: %s", err)
	}

	if len(resp.DBSecurityGroups) != 1 ||
		*resp.DBSecurityGroups[0].DBSecurityGroupName != d.Id() {
		return nil, fmt.Errorf("Unable to find DB Security Group: %#v", resp.DBSecurityGroups)
	}

	return resp.DBSecurityGroups[0], nil
}

// Authorizes the ingress rule on the db security group
func resourceAwsDbSecurityGroupAuthorizeRule(ingress interface{}, dbSecurityGroupName string, conn *rds.RDS) error {
	ing := ingress.(map[string]interface{})

	opts := rds.AuthorizeDBSecurityGroupIngressInput{
		DBSecurityGroupName: aws.String(dbSecurityGroupName),
	}

	if attr, ok := ing["cidr"]; ok && attr != "" {
		opts.CIDRIP = aws.String(attr.(string))
	}

	if attr, ok := ing["security_group_name"]; ok && attr != "" {
		opts.EC2SecurityGroupName = aws.String(attr.(string))
	}

	if attr, ok := ing["security_group_id"]; ok && attr != "" {
		opts.EC2SecurityGroupId = aws.String(attr.(string))
	}

	if attr, ok := ing["security_group_owner_id"]; ok && attr != "" {
		opts.EC2SecurityGroupOwnerId = aws.String(attr.(string))
	}

	log.Printf("[DEBUG] Authorize ingress rule configuration: %#v", opts)

	_, err := conn.AuthorizeDBSecurityGroupIngress(&opts)

	if err != nil {
		return fmt.Errorf("Error authorizing security group ingress: %s", err)
	}

	return nil
}

func resourceAwsDbSecurityGroupIngressHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["cidr"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["security_group_name"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["security_group_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["security_group_owner_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func resourceAwsDbSecurityGroupStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resourceAwsDbSecurityGroupRetrieve(d, meta)

		if err != nil {
			log.Printf("Error on retrieving DB Security Group when waiting: %s", err)
			return nil, "", err
		}

		statuses := make([]string, 0, len(v.EC2SecurityGroups)+len(v.IPRanges))
		for _, ec2g := range v.EC2SecurityGroups {
			statuses = append(statuses, *ec2g.Status)
		}
		for _, ips := range v.IPRanges {
			statuses = append(statuses, *ips.Status)
		}

		for _, stat := range statuses {
			// Not done
			if stat != "authorized" {
				return nil, "authorizing", nil
			}
		}

		return v, "authorized", nil
	}
}

func buildRDSSecurityGroupARN(d *schema.ResourceData, meta interface{}) (string, error) {
	iamconn := meta.(*AWSClient).iamconn
	region := meta.(*AWSClient).region
	// An zero value GetUserInput{} defers to the currently logged in user
	resp, err := iamconn.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}
	userARN := *resp.User.Arn
	accountID := strings.Split(userARN, ":")[4]
	arn := fmt.Sprintf("arn:aws:rds:%s:%s:secgrp:%s", region, accountID, d.Id())
	return arn, nil
}
