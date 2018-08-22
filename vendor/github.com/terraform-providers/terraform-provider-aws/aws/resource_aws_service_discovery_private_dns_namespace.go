package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsServiceDiscoveryPrivateDnsNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsServiceDiscoveryPrivateDnsNamespaceCreate,
		Read:   resourceAwsServiceDiscoveryPrivateDnsNamespaceRead,
		Delete: resourceAwsServiceDiscoveryPrivateDnsNamespaceDelete,

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
			"vpc": {
				Type:     schema.TypeString,
				Required: true,
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

func resourceAwsServiceDiscoveryPrivateDnsNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).sdconn

	name := d.Get("name").(string)
	// The CreatorRequestId has a limit of 64 bytes
	var requestId string
	if len(name) > (64 - resource.UniqueIDSuffixLength) {
		requestId = resource.PrefixedUniqueId(name[0:(64 - resource.UniqueIDSuffixLength - 1)])
	} else {
		requestId = resource.PrefixedUniqueId(name)
	}

	input := &servicediscovery.CreatePrivateDnsNamespaceInput{
		Name:             aws.String(name),
		Vpc:              aws.String(d.Get("vpc").(string)),
		CreatorRequestId: aws.String(requestId),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	resp, err := conn.CreatePrivateDnsNamespace(input)
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
	return resourceAwsServiceDiscoveryPrivateDnsNamespaceRead(d, meta)
}

func resourceAwsServiceDiscoveryPrivateDnsNamespaceRead(d *schema.ResourceData, meta interface{}) error {
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

	d.Set("description", resp.Namespace.Description)
	d.Set("arn", resp.Namespace.Arn)
	if resp.Namespace.Properties != nil {
		d.Set("hosted_zone", resp.Namespace.Properties.DnsProperties.HostedZoneId)
	}
	return nil
}

func resourceAwsServiceDiscoveryPrivateDnsNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
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
