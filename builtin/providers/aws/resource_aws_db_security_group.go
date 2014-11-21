package aws

/*
import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/flatmap"
	"github.com/hashicorp/terraform/helper/multierror"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/rds"
)

func resourceAwsDbSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbSecurityGroupCreate,
		Read:   resourceAwsDbSecurityGroupRead,
		Delete: resourceAwsDbSecurityGroupDelete,

		Schema: map[string]*schema.Schema{
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
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cidr": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"security_group_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
			},
		},
	}
}

func resourceAwsDbSecurityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	var err error
	var errs []error

	opts := rds.CreateDBSecurityGroup{
		DBSecurityGroupName:        d.Get("name").(string),
		DBSecurityGroupDescription: d.Get("description").(string),
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

	rules := d.Get("ingress.#").(int)
	if rules > 0 {
		for i := 0; i < ssh_keys; i++ {
			key := fmt.Sprintf("ingress.%d", i)
			err = resourceAwsDbSecurityGroupAuthorizeRule(d.Get(key), sg.Name, conn)

			if err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			return &multierror.Error{Errors: errs}
		}
	}

	log.Println(
		"[INFO] Waiting for Ingress Authorizations to be authorized")

	stateConf := &resource.StateChangeConf{
		Pending: []string{"authorizing"},
		Target:  "authorized",
		Refresh: DBSecurityGroupStateRefreshFunc(d.Id(), conn),
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
	conn := meta.(*AWSClient).rdsconn

	sg, err := resourceAwsDbSecurityGroupRetrieve(d, meta)
	if err != nil {
		return err
	}

	d.Set("name", sg.Name)
	d.Set("description", sg.Description)

	// Flatten our group values
	toFlatten := make(map[string]interface{})

	if len(v.EC2SecurityGroupOwnerIds) > 0 && v.EC2SecurityGroupOwnerIds[0] != "" {
		toFlatten["ingress_security_groups"] = v.EC2SecurityGroupOwnerIds
	}

	if len(v.CidrIps) > 0 && v.CidrIps[0] != "" {
		toFlatten["ingress_cidr"] = v.CidrIps
	}

	for k, v := range flatmap.Flatten(toFlatten) {
		s.Attributes[k] = v
	}

	return nil
}

func resourceAwsDbSecurityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	log.Printf("[DEBUG] DB Security Group destroy: %v", d.Id())

	opts := rds.DeleteDBSecurityGroup{DBSecurityGroupName: d.Id()}

	log.Printf("[DEBUG] DB Security Group destroy configuration: %v", opts)
	_, err := conn.DeleteDBSecurityGroup(&opts)

	if err != nil {
		newerr, ok := err.(*rds.Error)
		if ok && newerr.Code == "InvalidDBSecurityGroup.NotFound" {
			return nil
		}
		return err
	}

	return nil
}

func resourceAwsDbSecurityGroupRetrieve(d *schema.ResourceData, meta interface{}) (*rds.DBSecurityGroup, error) {
	conn := meta.(*AWSClient).rdsconn

	opts := rds.DescribeDBSecurityGroups{
		DBSecurityGroupName: d.Id(),
	}

	log.Printf("[DEBUG] DB Security Group describe configuration: %#v", opts)

	resp, err := conn.DescribeDBSecurityGroups(&opts)

	if err != nil {
		return nil, fmt.Errorf("Error retrieving DB Security Groups: %s", err)
	}

	if len(resp.DBSecurityGroups) != 1 ||
		resp.DBSecurityGroups[0].Name != d.Id() {
		if err != nil {
			return nil, fmt.Errorf("Unable to find DB Security Group: %#v", resp.DBSecurityGroups)
		}
	}

	v := resp.DBSecurityGroups[0]

	return &v, nil
}

// Authorizes the ingress rule on the db security group
func resourceAwsDbSecurityGroupAuthorizeRule(ingress interface{}, dbSecurityGroupName string, conn *rds.Rds) error {
	ing := ingress.(map[string]interface{})

	opts := rds.AuthorizeDBSecurityGroupIngress{
		DBSecurityGroupName: dbSecurityGroupName,
	}

	opts.Cidr = ing["cidr"].(string)
	opts.EC2SecurityGroupName = ing["security_group_name"].(string)

	if attr, ok := ing["security_group_id"].(string); ok && attr != "" {
		opts.EC2SecurityGroupId = attr
	}

	if attr, ok := ing["security_group_owner_id"].(string); ok && attr != "" {
		opts.EC2SecurityGroupOwnerId = attr
	}

	log.Printf("[DEBUG] Authorize ingress rule configuration: %#v", opts)

	_, err := conn.AuthorizeDBSecurityGroupIngress(&opts)

	if err != nil {
		return fmt.Errorf("Error authorizing security group ingress: %s", err)
	}

	return nil
}

func resourceAwsDbSecurityGroupStateRefreshFunc(id string, conn *rds.Rds) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		v, err := resource_aws_db_security_group_retrieve(id, conn)

		if err != nil {
			log.Printf("Error on retrieving DB Security Group when waiting: %s", err)
			return nil, "", err
		}

		statuses := append(v.EC2SecurityGroupStatuses, v.CidrStatuses...)

		for _, stat := range statuses {
			// Not done
			if stat != "authorized" {
				return nil, "authorizing", nil
			}
		}

		return v, "authorized", nil
	}
}
*/
