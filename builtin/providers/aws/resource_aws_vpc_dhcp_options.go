package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsVpcDhcpOptions() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsVpcDhcpOptionsCreate,
		Read:   resourceAwsVpcDhcpOptionsRead,
		Update: resourceAwsVpcDhcpOptionsUpdate,
		Delete: resourceAwsVpcDhcpOptionsDelete,

		Schema: map[string]*schema.Schema{
			"domain_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"domain_name_servers": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"ntp_servers": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"netbios_node_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"netbios_name_servers": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"tags": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func resourceAwsVpcDhcpOptionsCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	setDHCPOption := func(key string) *ec2.NewDHCPConfiguration {
		log.Printf("[DEBUG] Setting DHCP option %s...", key)
		tfKey := strings.Replace(key, "-", "_", -1)

		value, ok := d.GetOk(tfKey)
		if !ok {
			return nil
		}

		if v, ok := value.(string); ok {
			return &ec2.NewDHCPConfiguration{
				Key: aws.String(key),
				Values: []*string{
					aws.String(v),
				},
			}
		}

		if v, ok := value.([]interface{}); ok {
			var s []*string
			for _, attr := range v {
				s = append(s, aws.String(attr.(string)))
			}

			return &ec2.NewDHCPConfiguration{
				Key:    aws.String(key),
				Values: s,
			}
		}

		return nil
	}

	createOpts := &ec2.CreateDHCPOptionsInput{
		DHCPConfigurations: []*ec2.NewDHCPConfiguration{
			setDHCPOption("domain-name"),
			setDHCPOption("domain-name-servers"),
			setDHCPOption("ntp-servers"),
			setDHCPOption("netbios-node-type"),
			setDHCPOption("netbios-name-servers"),
		},
	}

	resp, err := conn.CreateDHCPOptions(createOpts)
	if err != nil {
		return fmt.Errorf("Error creating DHCP Options Set: %s", err)
	}

	dos := resp.DHCPOptions
	d.SetId(*dos.DHCPOptionsID)
	log.Printf("[INFO] DHCP Options Set ID: %s", d.Id())

	// Wait for the DHCP Options to become available
	log.Printf("[DEBUG] Waiting for DHCP Options (%s) to become available", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"pending"},
		Target:  "",
		Refresh: DHCPOptionsStateRefreshFunc(conn, d.Id()),
		Timeout: 1 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf(
			"Error waiting for DHCP Options (%s) to become available: %s",
			d.Id(), err)
	}

	return resourceAwsVpcDhcpOptionsUpdate(d, meta)
}

func resourceAwsVpcDhcpOptionsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	req := &ec2.DescribeDHCPOptionsInput{
		DHCPOptionsIDs: []*string{
			aws.String(d.Id()),
		},
	}

	resp, err := conn.DescribeDHCPOptions(req)
	if err != nil {
		return fmt.Errorf("Error retrieving DHCP Options: %s", err)
	}

	if len(resp.DHCPOptions) == 0 {
		return nil
	}

	opts := resp.DHCPOptions[0]
	d.Set("tags", tagsToMap(opts.Tags))

	for _, cfg := range opts.DHCPConfigurations {
		tfKey := strings.Replace(*cfg.Key, "-", "_", -1)

		if _, ok := d.Get(tfKey).(string); ok {
			d.Set(tfKey, cfg.Values[0].Value)
		} else {
			values := make([]string, 0, len(cfg.Values))
			for _, v := range cfg.Values {
				values = append(values, *v.Value)
			}

			d.Set(tfKey, values)
		}
	}

	return nil
}

func resourceAwsVpcDhcpOptionsUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn
	return setTags(conn, d)
}

func resourceAwsVpcDhcpOptionsDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	return resource.Retry(3*time.Minute, func() error {
		log.Printf("[INFO] Deleting DHCP Options ID %s...", d.Id())
		_, err := conn.DeleteDHCPOptions(&ec2.DeleteDHCPOptionsInput{
			DHCPOptionsID: aws.String(d.Id()),
		})

		if err == nil {
			return nil
		}

		log.Printf("[WARN] %s", err)

		ec2err, ok := err.(awserr.Error)
		if !ok {
			return err
		}

		switch ec2err.Code() {
		case "InvalidDhcpOptionsID.NotFound":
			return nil
		case "DependencyViolation":
			// If it is a dependency violation, we want to disassociate
			// all VPCs using the given DHCP Options ID, and retry deleting.
			vpcs, err2 := findVPCsByDHCPOptionsID(conn, d.Id())
			if err2 != nil {
				log.Printf("[ERROR] %s", err2)
				return err2
			}

			for _, vpc := range vpcs {
				log.Printf("[INFO] Disassociating DHCP Options Set %s from VPC %s...", d.Id(), *vpc.VPCID)
				if _, err := conn.AssociateDHCPOptions(&ec2.AssociateDHCPOptionsInput{
					DHCPOptionsID: aws.String("default"),
					VPCID:         vpc.VPCID,
				}); err != nil {
					return err
				}
			}
			return err //retry
		default:
			// Any other error, we want to quit the retry loop immediately
			return resource.RetryError{Err: err}
		}

		return nil
	})
}

func findVPCsByDHCPOptionsID(conn *ec2.EC2, id string) ([]*ec2.VPC, error) {
	req := &ec2.DescribeVPCsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("dhcp-options-id"),
				Values: []*string{
					aws.String(id),
				},
			},
		},
	}

	resp, err := conn.DescribeVPCs(req)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidVpcID.NotFound" {
			return nil, nil
		}
		return nil, err
	}

	return resp.VPCs, nil
}

func DHCPOptionsStateRefreshFunc(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		DescribeDhcpOpts := &ec2.DescribeDHCPOptionsInput{
			DHCPOptionsIDs: []*string{
				aws.String(id),
			},
		}

		resp, err := conn.DescribeDHCPOptions(DescribeDhcpOpts)
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidDhcpOptionsID.NotFound" {
				resp = nil
			} else {
				log.Printf("Error on DHCPOptionsStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		dos := resp.DHCPOptions[0]
		return dos, "", nil
	}
}
