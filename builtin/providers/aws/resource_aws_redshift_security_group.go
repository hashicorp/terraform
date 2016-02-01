package aws

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRedshiftSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRedshiftSecurityGroupCreate,
		Read:   resourceAwsRedshiftSecurityGroupRead,
		Delete: resourceAwsRedshiftSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateRedshiftSecurityGroupName,
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

						"security_group_owner_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceAwsRedshiftSecurityGroupIngressHash,
			},
		},
	}
}

func resourceAwsRedshiftSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	var err error
	var errs []error

	name := d.Get("name").(string)
	desc := d.Get("description").(string)
	sgInput := &redshift.CreateClusterSecurityGroupInput{
		ClusterSecurityGroupName: aws.String(name),
		Description:              aws.String(desc),
	}
	log.Printf("[DEBUG] Redshift security group create: name: %s, description: %s", name, desc)
	_, err = conn.CreateClusterSecurityGroup(sgInput)
	if err != nil {
		return fmt.Errorf("Error creating RedshiftSecurityGroup: %s", err)
	}

	d.SetId(d.Get("name").(string))

	log.Printf("[INFO] Redshift Security Group ID: %s", d.Id())
	sg, err := resourceAwsRedshiftSecurityGroupRetrieve(d, meta)
	if err != nil {
		return err
	}

	ingresses := d.Get("ingress").(*schema.Set)
	for _, ing := range ingresses.List() {
		err := resourceAwsRedshiftSecurityGroupAuthorizeRule(ing, *sg.ClusterSecurityGroupName, conn)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return &multierror.Error{Errors: errs}
	}

	log.Println("[INFO] Waiting for Redshift Security Group Ingress Authorizations to be authorized")
	stateConf := &resource.StateChangeConf{
		Pending: []string{"authorizing"},
		Target:  []string{"authorized"},
		Refresh: resourceAwsRedshiftSecurityGroupStateRefreshFunc(d, meta),
		Timeout: 10 * time.Minute,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsRedshiftSecurityGroupRead(d, meta)
}

func resourceAwsRedshiftSecurityGroupRead(d *schema.ResourceData, meta interface{}) error {
	sg, err := resourceAwsRedshiftSecurityGroupRetrieve(d, meta)
	if err != nil {
		return err
	}

	rules := &schema.Set{
		F: resourceAwsRedshiftSecurityGroupIngressHash,
	}

	for _, v := range sg.IPRanges {
		rule := map[string]interface{}{"cidr": *v.CIDRIP}
		rules.Add(rule)
	}

	for _, g := range sg.EC2SecurityGroups {
		rule := map[string]interface{}{
			"security_group_name":     *g.EC2SecurityGroupName,
			"security_group_owner_id": *g.EC2SecurityGroupOwnerId,
		}
		rules.Add(rule)
	}

	d.Set("ingress", rules)
	d.Set("name", *sg.ClusterSecurityGroupName)
	d.Set("description", *sg.Description)

	return nil
}

func resourceAwsRedshiftSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	log.Printf("[DEBUG] Redshift Security Group destroy: %v", d.Id())
	opts := redshift.DeleteClusterSecurityGroupInput{
		ClusterSecurityGroupName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Redshift Security Group destroy configuration: %v", opts)
	_, err := conn.DeleteClusterSecurityGroup(&opts)

	if err != nil {
		newerr, ok := err.(awserr.Error)
		if ok && newerr.Code() == "InvalidRedshiftSecurityGroup.NotFound" {
			return nil
		}
		return err
	}

	return nil
}

func resourceAwsRedshiftSecurityGroupRetrieve(d *schema.ResourceData, meta interface{}) (*redshift.ClusterSecurityGroup, error) {
	conn := meta.(*AWSClient).redshiftconn

	opts := redshift.DescribeClusterSecurityGroupsInput{
		ClusterSecurityGroupName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Redshift Security Group describe configuration: %#v", opts)

	resp, err := conn.DescribeClusterSecurityGroups(&opts)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving Redshift Security Groups: %s", err)
	}

	if len(resp.ClusterSecurityGroups) != 1 ||
		*resp.ClusterSecurityGroups[0].ClusterSecurityGroupName != d.Id() {
		return nil, fmt.Errorf("Unable to find Redshift Security Group: %#v", resp.ClusterSecurityGroups)
	}

	return resp.ClusterSecurityGroups[0], nil
}

func validateRedshiftSecurityGroupName(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if value == "default" {
		errors = append(errors, fmt.Errorf("the Redshift Security Group name cannot be %q", value))
	}
	if !regexp.MustCompile(`^[0-9a-z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only lowercase alphanumeric characters and hyphens allowed in %q: %q",
			k, value))
	}
	if len(value) > 255 {
		errors = append(errors, fmt.Errorf(
			"%q cannot be longer than 32 characters: %q", k, value))
	}
	return

}

func resourceAwsRedshiftSecurityGroupIngressHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["cidr"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["security_group_name"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	if v, ok := m["security_group_owner_id"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func resourceAwsRedshiftSecurityGroupAuthorizeRule(ingress interface{}, redshiftSecurityGroupName string, conn *redshift.Redshift) error {
	ing := ingress.(map[string]interface{})

	opts := redshift.AuthorizeClusterSecurityGroupIngressInput{
		ClusterSecurityGroupName: aws.String(redshiftSecurityGroupName),
	}

	if attr, ok := ing["cidr"]; ok && attr != "" {
		opts.CIDRIP = aws.String(attr.(string))
	}

	if attr, ok := ing["security_group_name"]; ok && attr != "" {
		opts.EC2SecurityGroupName = aws.String(attr.(string))
	}

	if attr, ok := ing["security_group_owner_id"]; ok && attr != "" {
		opts.EC2SecurityGroupOwnerId = aws.String(attr.(string))
	}

	log.Printf("[DEBUG] Authorize ingress rule configuration: %#v", opts)
	_, err := conn.AuthorizeClusterSecurityGroupIngress(&opts)

	if err != nil {
		return fmt.Errorf("Error authorizing security group ingress: %s", err)
	}

	return nil
}

func resourceAwsRedshiftSecurityGroupStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resourceAwsRedshiftSecurityGroupRetrieve(d, meta)

		if err != nil {
			log.Printf("Error on retrieving Redshift Security Group when waiting: %s", err)
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
