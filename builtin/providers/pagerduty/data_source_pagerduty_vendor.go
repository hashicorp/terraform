package pagerduty

import (
	"fmt"
	"log"
	"regexp"

	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourcePagerDutyVendor() *schema.Resource {
	return &schema.Resource{
		Read: dataSourcePagerDutyVendorRead,

		Schema: map[string]*schema.Schema{
			"name_regex": {
				Type:          schema.TypeString,
				Optional:      true,
				Deprecated:    "Use field name instead",
				ConflictsWith: []string{"name"},
			},
			"name": {
				Type:          schema.TypeString,
				Computed:      true,
				Optional:      true,
				ConflictsWith: []string{"name_regex"},
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourcePagerDutyVendorRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	// Check if we're doing a normal or legacy lookup
	_, ok := d.GetOk("name")
	_, legacyOk := d.GetOk("name_regex")

	if !ok && !legacyOk {
		return fmt.Errorf("Either name or name_regex must be set")
	}

	// If name_regex is set, we're doing a legacy lookup
	if legacyOk {
		return dataSourcePagerDutyVendorLegacyRead(d, meta)
	}

	log.Printf("[INFO] Reading PagerDuty vendor")

	searchName := d.Get("name").(string)

	o := &pagerduty.ListVendorOptions{
		Query: searchName,
	}

	resp, err := client.ListVendors(*o)
	if err != nil {
		return err
	}

	var found *pagerduty.Vendor

	r := regexp.MustCompile("(?i)" + searchName)

	for _, vendor := range resp.Vendors {
		if r.MatchString(vendor.Name) {
			found = &vendor
			break
		}
	}

	if found == nil {
		return fmt.Errorf("Unable to locate any vendor with the name: %s", searchName)
	}

	d.SetId(found.ID)
	d.Set("name", found.Name)
	d.Set("type", found.GenericServiceType)

	return nil
}

func dataSourcePagerDutyVendorLegacyRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*pagerduty.Client)

	log.Printf("[INFO] Reading PagerDuty vendor (legacy)")

	resp, err := getVendors(client)

	if err != nil {
		return err
	}

	r := regexp.MustCompile("(?i)" + d.Get("name_regex").(string))

	var vendors []pagerduty.Vendor
	var vendorNames []string

	for _, v := range resp {
		if r.MatchString(v.Name) {
			vendors = append(vendors, v)
			vendorNames = append(vendorNames, v.Name)
		}
	}

	if len(vendors) == 0 {
		return fmt.Errorf("Unable to locate any vendor using the regex string: %s", r.String())
	} else if len(vendors) > 1 {
		return fmt.Errorf("Your query returned more than one result using the regex string: %#v. Found vendors: %#v", r.String(), vendorNames)
	}

	vendor := vendors[0]

	genericServiceType := vendor.GenericServiceType

	switch {
	case genericServiceType == "email":
		genericServiceType = "generic_email_inbound_integration"
	case genericServiceType == "api":
		genericServiceType = "generic_events_api_inbound_integration"
	}

	d.SetId(vendor.ID)
	d.Set("name", vendor.Name)
	d.Set("type", genericServiceType)

	return nil
}
