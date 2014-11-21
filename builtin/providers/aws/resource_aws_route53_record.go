package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/route53"
)

func resourceAwsRoute53Record() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53RecordCreate,
		Read:   resourceAwsRoute53RecordRead,
		Delete: resourceAwsRoute53RecordDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"records": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsRoute53RecordCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).route53

	// Get the record
	rec, err := resourceAwsRoute53RecordBuildSet(d)
	if err != nil {
		return err
	}

	// Create the new records. We abuse StateChangeConf for this to
	// retry for us since Route53 sometimes returns errors about another
	// operation happening at the same time.
	req := &route53.ChangeResourceRecordSetsRequest{
		Comment: "Managed by Terraform",
		Changes: []route53.Change{
			route53.Change{
				Action: "UPSERT",
				Record: *rec,
			},
		},
	}
	zone := d.Get("zone_id").(string)
	log.Printf("[DEBUG] Creating resource records for zone: %s, name: %s",
		zone, d.Get("name").(string))

	wait := resource.StateChangeConf{
		Pending:    []string{"rejected"},
		Target:     "accepted",
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.ChangeResourceRecordSets(zone, req)
			if err != nil {
				if strings.Contains(err.Error(), "PriorRequestNotComplete") {
					// There is some pending operation, so just retry
					// in a bit.
					return nil, "rejected", nil
				}

				return nil, "failure", err
			}

			return resp.ChangeInfo, "accepted", nil
		},
	}

	respRaw, err := wait.WaitForState()
	if err != nil {
		return err
	}
	changeInfo := respRaw.(route53.ChangeInfo)

	// Generate an ID
	d.SetId(fmt.Sprintf("%s_%s_%s", zone, d.Get("name").(string), d.Get("type").(string)))

	// Wait until we are done
	wait = resource.StateChangeConf{
		Delay:      30 * time.Second,
		Pending:    []string{"PENDING"},
		Target:     "INSYNC",
		Timeout:    10 * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			return resourceAwsRoute53Wait(conn, changeInfo.ID)
		},
	}
	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsRoute53RecordRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).route53

	zone := d.Get("zone_id").(string)
	lopts := &route53.ListOpts{
		Name: d.Get("name").(string),
		Type: d.Get("type").(string),
	}
	resp, err := conn.ListResourceRecordSets(zone, lopts)
	if err != nil {
		return err
	}

	// Scan for a matching record
	found := false
	for _, record := range resp.Records {
		if route53.FQDN(record.Name) != route53.FQDN(lopts.Name) {
			continue
		}
		if strings.ToUpper(record.Type) != strings.ToUpper(lopts.Type) {
			continue
		}

		found = true

		for i, rec := range record.Records {
			key := fmt.Sprintf("records.%d", i)
			d.Set(key, rec)
		}
		d.Set("ttl", record.TTL)

		break
	}

	if !found {
		d.SetId("")
	}

	return nil
}

func resourceAwsRoute53RecordDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).route53

	// Get the records
	rec, err := resourceAwsRoute53RecordBuildSet(d)
	if err != nil {
		return err
	}

	// Create the new records
	req := &route53.ChangeResourceRecordSetsRequest{
		Comment: "Deleted by Terraform",
		Changes: []route53.Change{
			route53.Change{
				Action: "DELETE",
				Record: *rec,
			},
		},
	}
	zone := d.Get("zone_id").(string)
	log.Printf("[DEBUG] Deleting resource records for zone: %s, name: %s",
		zone, d.Get("name").(string))

	wait := resource.StateChangeConf{
		Pending:    []string{"rejected"},
		Target:     "accepted",
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			_, err := conn.ChangeResourceRecordSets(zone, req)
			if err != nil {
				if strings.Contains(err.Error(), "PriorRequestNotComplete") {
					// There is some pending operation, so just retry
					// in a bit.
					return 42, "rejected", nil
				}

				if strings.Contains(err.Error(), "InvalidChangeBatch") {
					// This means that the record is already gone.
					return 42, "accepted", nil
				}

				return 42, "failure", err
			}

			return 42, "accepted", nil
		},
	}

	if _, err := wait.WaitForState(); err != nil {
		return err
	}

	return nil
}

func resourceAwsRoute53RecordBuildSet(d *schema.ResourceData) (*route53.ResourceRecordSet, error) {
	recs := d.Get("records.#").(int)
	records := make([]string, 0, recs)
	for i := 0; i < recs; i++ {
		key := fmt.Sprintf("records.%d", i)
		records = append(records, d.Get(key).(string))
	}

	rec := &route53.ResourceRecordSet{
		Name:    d.Get("name").(string),
		Type:    d.Get("type").(string),
		TTL:     d.Get("ttl").(int),
		Records: records,
	}
	return rec, nil
}
