package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsServiceDiscoveryPublicDnsNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsServiceDiscoveryPublicDnsNamespaceCreate,
		Read:   resourceAwsServiceDiscoveryPublicDnsNamespaceRead,
		Delete: resourceAwsServiceDiscoveryPublicDnsNamespaceDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hosted_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsServiceDiscoveryPublicDnsNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	requestId := resource.PrefixedUniqueId(fmt.Sprintf("tf-%s", d.Get("name").(string)))
	input := &servicediscovery.CreatePublicDnsNamespaceInput{
		Name:             aws.String(d.Get("name").(string)),
		CreatorRequestId: aws.String(requestId),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	resp, err := conn.CreatePublicDnsNamespace(input)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{servicediscovery.OperationStatusSubmitted, servicediscovery.OperationStatusPending},
		Target:  []string{servicediscovery.OperationStatusSuccess},
		Refresh: servicediscoveryOperationRefreshStatusFunc(conn, *resp.OperationId),
		Timeout: 5 * time.Minute,
	}

	opresp, err := stateConf.WaitForState()
	if err != nil {
		return err
	}

	d.SetId(*opresp.(*servicediscovery.GetOperationOutput).Operation.Targets["NAMESPACE"])
	return resourceAwsServiceDiscoveryPublicDnsNamespaceRead(d, meta)
}

func resourceAwsServiceDiscoveryPublicDnsNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	input := &servicediscovery.GetNamespaceInput{
		Id: aws.String(d.Id()),
	}

	resp, err := conn.GetNamespace(input)
	if err != nil {
		if isAWSErr(err, servicediscovery.ErrCodeNamespaceNotFound, "") {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", resp.Namespace.Name)
	d.Set("description", resp.Namespace.Description)
	d.Set("arn", resp.Namespace.Arn)
	if resp.Namespace.Properties != nil {
		d.Set("hosted_zone", resp.Namespace.Properties.DnsProperties.HostedZoneId)
	}
	return nil
}

func resourceAwsServiceDiscoveryPublicDnsNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	input := &servicediscovery.DeleteNamespaceInput{
		Id: aws.String(d.Id()),
	}

	resp, err := conn.DeleteNamespace(input)
	if err != nil {
		return err
	}

	stateConf := &resource.StateChangeConf{
		Pending: []string{servicediscovery.OperationStatusSubmitted, servicediscovery.OperationStatusPending},
		Target:  []string{servicediscovery.OperationStatusSuccess},
		Refresh: servicediscoveryOperationRefreshStatusFunc(conn, *resp.OperationId),
		Timeout: 5 * time.Minute,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return nil
}

func servicediscoveryOperationRefreshStatusFunc(conn *servicediscovery.ServiceDiscovery, oid string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &servicediscovery.GetOperationInput{
			OperationId: aws.String(oid),
		}
		resp, err := conn.GetOperation(input)
		if err != nil {
			return nil, "failed", err
		}
		return resp, *resp.Operation.Status, nil
	}
}
