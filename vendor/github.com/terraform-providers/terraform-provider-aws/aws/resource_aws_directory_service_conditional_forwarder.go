package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/directoryservice"
)

func resourceAwsDirectoryServiceConditionalForwarder() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDirectoryServiceConditionalForwarderCreate,
		Read:   resourceAwsDirectoryServiceConditionalForwarderRead,
		Update: resourceAwsDirectoryServiceConditionalForwarderUpdate,
		Delete: resourceAwsDirectoryServiceConditionalForwarderDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"directory_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"dns_ips": {
				Type:     schema.TypeList,
				Required: true,
				MinItems: 1,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"remote_domain_name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^([a-zA-Z0-9]+[\.-])+([a-zA-Z0-9])+[.]?$`), "invalid value, see the RemoteDomainName attribute documentation: https://docs.aws.amazon.com/directoryservice/latest/devguide/API_ConditionalForwarder.html"),
			},
		},
	}
}

func resourceAwsDirectoryServiceConditionalForwarderCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dsconn

	dnsIps := expandStringList(d.Get("dns_ips").([]interface{}))

	directoryId := d.Get("directory_id").(string)
	domainName := d.Get("remote_domain_name").(string)

	_, err := conn.CreateConditionalForwarder(&directoryservice.CreateConditionalForwarderInput{
		DirectoryId:      aws.String(directoryId),
		DnsIpAddrs:       dnsIps,
		RemoteDomainName: aws.String(domainName),
	})

	if err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%s:%s", directoryId, domainName))

	return nil
}

func resourceAwsDirectoryServiceConditionalForwarderRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dsconn

	directoryId, domainName, err := parseDSConditionalForwarderId(d.Id())
	if err != nil {
		return err
	}

	res, err := conn.DescribeConditionalForwarders(&directoryservice.DescribeConditionalForwardersInput{
		DirectoryId:       aws.String(directoryId),
		RemoteDomainNames: []*string{aws.String(domainName)},
	})

	if err != nil {
		if isAWSErr(err, directoryservice.ErrCodeEntityDoesNotExistException, "") {
			log.Printf("[WARN] Directory Service Conditional Forwarder (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	if len(res.ConditionalForwarders) == 0 {
		log.Printf("[WARN] Directory Service Conditional Forwarder (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	cfd := res.ConditionalForwarders[0]

	d.Set("dns_ips", flattenStringList(cfd.DnsIpAddrs))
	d.Set("directory_id", directoryId)
	d.Set("remote_domain_name", cfd.RemoteDomainName)

	return nil
}

func resourceAwsDirectoryServiceConditionalForwarderUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dsconn

	directoryId, domainName, err := parseDSConditionalForwarderId(d.Id())
	if err != nil {
		return err
	}

	dnsIps := expandStringList(d.Get("dns_ips").([]interface{}))

	_, err = conn.UpdateConditionalForwarder(&directoryservice.UpdateConditionalForwarderInput{
		DirectoryId:      aws.String(directoryId),
		DnsIpAddrs:       dnsIps,
		RemoteDomainName: aws.String(domainName),
	})

	if err != nil {
		return err
	}

	return resourceAwsDirectoryServiceConditionalForwarderRead(d, meta)
}

func resourceAwsDirectoryServiceConditionalForwarderDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).dsconn

	directoryId, domainName, err := parseDSConditionalForwarderId(d.Id())
	if err != nil {
		return err
	}

	_, err = conn.DeleteConditionalForwarder(&directoryservice.DeleteConditionalForwarderInput{
		DirectoryId:      aws.String(directoryId),
		RemoteDomainName: aws.String(domainName),
	})

	if err != nil && !isAWSErr(err, directoryservice.ErrCodeEntityDoesNotExistException, "") {
		return err
	}

	return nil
}

func parseDSConditionalForwarderId(id string) (directoryId, domainName string, err error) {
	parts := strings.SplitN(id, ":", 2)

	if len(parts) != 2 {
		return "", "", fmt.Errorf("please make sure ID is in format DIRECTORY_ID:DOMAIN_NAME")
	}

	return parts[0], parts[1], nil
}
