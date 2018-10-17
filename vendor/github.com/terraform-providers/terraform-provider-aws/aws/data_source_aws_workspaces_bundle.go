package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/workspaces"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsWorkspaceBundle() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsWorkspaceBundleRead,

		Schema: map[string]*schema.Schema{
			"bundle_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"owner": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"compute_type": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"user_storage": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"capacity": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"root_storage": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"capacity": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsWorkspaceBundleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).workspacesconn

	bundleID := d.Get("bundle_id").(string)
	input := &workspaces.DescribeWorkspaceBundlesInput{
		BundleIds: []*string{aws.String(bundleID)},
	}

	resp, err := conn.DescribeWorkspaceBundles(input)
	if err != nil {
		return err
	}

	if len(resp.Bundles) != 1 {
		return fmt.Errorf("The number of Workspace Bundle (%s) should be 1, but %d", bundleID, len(resp.Bundles))
	}

	bundle := resp.Bundles[0]
	d.SetId(bundleID)
	d.Set("description", bundle.Description)
	d.Set("name", bundle.Name)
	d.Set("owner", bundle.Owner)

	computeType := make([]map[string]interface{}, 1)
	if bundle.ComputeType != nil {
		computeType[0] = map[string]interface{}{
			"name": aws.StringValue(bundle.ComputeType.Name),
		}
	}
	if err := d.Set("compute_type", computeType); err != nil {
		return fmt.Errorf("error setting compute_type: %s", err)
	}

	rootStorage := make([]map[string]interface{}, 1)
	if bundle.RootStorage != nil {
		rootStorage[0] = map[string]interface{}{
			"capacity": aws.StringValue(bundle.RootStorage.Capacity),
		}
	}
	if err := d.Set("root_storage", rootStorage); err != nil {
		return fmt.Errorf("error setting root_storage: %s", err)
	}

	userStorage := make([]map[string]interface{}, 1)
	if bundle.UserStorage != nil {
		userStorage[0] = map[string]interface{}{
			"capacity": aws.StringValue(bundle.UserStorage.Capacity),
		}
	}
	if err := d.Set("user_storage", userStorage); err != nil {
		return fmt.Errorf("error setting user_storage: %s", err)
	}

	return nil
}
