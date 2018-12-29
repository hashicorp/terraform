package aws

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsVpcDhcpOptions() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsVpcDhcpOptionsRead,

		Schema: map[string]*schema.Schema{
			"dhcp_options_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"domain_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"domain_name_servers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"filter": ec2CustomFiltersSchema(),
			"netbios_name_servers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"netbios_node_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ntp_servers": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsVpcDhcpOptionsRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	input := &ec2.DescribeDhcpOptionsInput{}

	if v, ok := d.GetOk("dhcp_options_id"); ok && v.(string) != "" {
		input.DhcpOptionsIds = []*string{aws.String(v.(string))}
	}

	input.Filters = append(input.Filters, buildEC2CustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)
	if len(input.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		input.Filters = nil
	}

	log.Printf("[DEBUG] Reading EC2 DHCP Options: %s", input)
	output, err := conn.DescribeDhcpOptions(input)
	if err != nil {
		if isNoSuchDhcpOptionIDErr(err) {
			return errors.New("No matching EC2 DHCP Options found")
		}
		return fmt.Errorf("error reading EC2 DHCP Options: %s", err)
	}

	if len(output.DhcpOptions) == 0 {
		return errors.New("No matching EC2 DHCP Options found")
	}

	if len(output.DhcpOptions) > 1 {
		return errors.New("Multiple matching EC2 DHCP Options found")
	}

	dhcpOptionID := aws.StringValue(output.DhcpOptions[0].DhcpOptionsId)
	d.SetId(dhcpOptionID)
	d.Set("dhcp_options_id", dhcpOptionID)

	dhcpConfigurations := output.DhcpOptions[0].DhcpConfigurations

	for _, dhcpConfiguration := range dhcpConfigurations {
		key := aws.StringValue(dhcpConfiguration.Key)
		tfKey := strings.Replace(key, "-", "_", -1)

		if len(dhcpConfiguration.Values) == 0 {
			continue
		}

		switch key {
		case "domain-name":
			d.Set(tfKey, aws.StringValue(dhcpConfiguration.Values[0].Value))
		case "domain-name-servers":
			if err := d.Set(tfKey, flattenEc2AttributeValues(dhcpConfiguration.Values)); err != nil {
				return fmt.Errorf("error setting %s: %s", tfKey, err)
			}
		case "netbios-name-servers":
			if err := d.Set(tfKey, flattenEc2AttributeValues(dhcpConfiguration.Values)); err != nil {
				return fmt.Errorf("error setting %s: %s", tfKey, err)
			}
		case "netbios-node-type":
			d.Set(tfKey, aws.StringValue(dhcpConfiguration.Values[0].Value))
		case "ntp-servers":
			if err := d.Set(tfKey, flattenEc2AttributeValues(dhcpConfiguration.Values)); err != nil {
				return fmt.Errorf("error setting %s: %s", tfKey, err)
			}
		}
	}

	if err := d.Set("tags", d.Set("tags", tagsToMap(output.DhcpOptions[0].Tags))); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	return nil
}
