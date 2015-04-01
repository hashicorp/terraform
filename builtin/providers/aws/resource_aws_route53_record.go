package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/route53"
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
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				ForceNew: true,
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
		},
	}
}

func resourceAwsRoute53RecordCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn
	zone := d.Get("zone_id").(string)

	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneRequest{ID: aws.String(zone)})
	if err != nil {
		return err
	}

	// Get the record
	rec, err := resourceAwsRoute53RecordBuildSet(d, *zoneRecord.HostedZone.Name)
	if err != nil {
		return err
	}

	// Create the new records. We abuse StateChangeConf for this to
	// retry for us since Route53 sometimes returns errors about another
	// operation happening at the same time.
	changeBatch := &route53.ChangeBatch{
		Comment: aws.String("Managed by Terraform"),
		Changes: []route53.Change{
			route53.Change{
				Action:            aws.String("UPSERT"),
				ResourceRecordSet: rec,
			},
		},
	}

	req := &route53.ChangeResourceRecordSetsRequest{
		HostedZoneID: aws.String(cleanZoneID(*zoneRecord.HostedZone.ID)),
		ChangeBatch:  changeBatch,
	}

	log.Printf("[DEBUG] Creating resource records for zone: %s, name: %s",
		zone, *rec.Name)

	wait := resource.StateChangeConf{
		Pending:    []string{"rejected"},
		Target:     "accepted",
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.ChangeResourceRecordSets(req)
			if err != nil {
				if r53err, ok := err.(aws.APIError); ok {
					if r53err.Code == "PriorRequestNotComplete" {
						// There is some pending operation, so just retry
						// in a bit.
						return nil, "rejected", nil
					}
				}

				return nil, "failure", err
			}

			return resp, "accepted", nil
		},
	}

	respRaw, err := wait.WaitForState()
	if err != nil {
		return err
	}
	changeInfo := respRaw.(*route53.ChangeResourceRecordSetsResponse).ChangeInfo

	// Generate an ID
	d.SetId(fmt.Sprintf("%s_%s_%s", zone, d.Get("name").(string), d.Get("type").(string)))

	// Wait until we are done
	wait = resource.StateChangeConf{
		Delay:      30 * time.Second,
		Pending:    []string{"PENDING"},
		Target:     "INSYNC",
		Timeout:    30 * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			changeRequest := &route53.GetChangeRequest{
				ID: aws.String(cleanChangeID(*changeInfo.ID)),
			}
			return resourceAwsGoRoute53Wait(conn, changeRequest)
		},
	}
	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsRoute53RecordRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	zone := d.Get("zone_id").(string)

	// get expanded name
	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneRequest{ID: aws.String(zone)})
	if err != nil {
		return err
	}
	en := expandRecordName(d.Get("name").(string), *zoneRecord.HostedZone.Name)

	lopts := &route53.ListResourceRecordSetsRequest{
		HostedZoneID:    aws.String(cleanZoneID(zone)),
		StartRecordName: aws.String(en),
		StartRecordType: aws.String(d.Get("type").(string)),
	}

	resp, err := conn.ListResourceRecordSets(lopts)
	if err != nil {
		return err
	}

	// Scan for a matching record
	found := false
	for _, record := range resp.ResourceRecordSets {
		name := cleanRecordName(*record.Name)
		if FQDN(name) != FQDN(*lopts.StartRecordName) {
			continue
		}
		if strings.ToUpper(*record.Type) != strings.ToUpper(*lopts.StartRecordType) {
			continue
		}

		found = true

		d.Set("records", record.ResourceRecords)
		d.Set("ttl", record.TTL)

		break
	}

	if !found {
		d.SetId("")
	}

	return nil
}

func resourceAwsRoute53RecordDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	zone := d.Get("zone_id").(string)
	log.Printf("[DEBUG] Deleting resource records for zone: %s, name: %s",
		zone, d.Get("name").(string))
	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneRequest{ID: aws.String(zone)})
	if err != nil {
		return err
	}
	// Get the records
	rec, err := resourceAwsRoute53RecordBuildSet(d, *zoneRecord.HostedZone.Name)
	if err != nil {
		return err
	}

	// Create the new records
	changeBatch := &route53.ChangeBatch{
		Comment: aws.String("Deleted by Terraform"),
		Changes: []route53.Change{
			route53.Change{
				Action:            aws.String("DELETE"),
				ResourceRecordSet: rec,
			},
		},
	}

	req := &route53.ChangeResourceRecordSetsRequest{
		HostedZoneID: aws.String(cleanZoneID(zone)),
		ChangeBatch:  changeBatch,
	}

	wait := resource.StateChangeConf{
		Pending:    []string{"rejected"},
		Target:     "accepted",
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			_, err := conn.ChangeResourceRecordSets(req)
			if err != nil {
				if r53err, ok := err.(aws.APIError); ok {
					if r53err.Code == "PriorRequestNotComplete" {
						// There is some pending operation, so just retry
						// in a bit.
						return 42, "rejected", nil
					}

					if r53err.Code == "InvalidChangeBatch" {
						// This means that the record is already gone.
						return 42, "accepted", nil
					}
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

func resourceAwsRoute53RecordBuildSet(d *schema.ResourceData, zoneName string) (*route53.ResourceRecordSet, error) {
	recs := d.Get("records").(*schema.Set).List()
	records := make([]route53.ResourceRecord, 0, len(recs))

	typeStr := d.Get("type").(string)
	for _, r := range recs {
		switch typeStr {
		case "TXT":
			str := fmt.Sprintf("\"%s\"", r.(string))
			records = append(records, route53.ResourceRecord{Value: aws.String(str)})
		default:
			records = append(records, route53.ResourceRecord{Value: aws.String(r.(string))})
		}
	}

	// get expanded name
	en := expandRecordName(d.Get("name").(string), zoneName)

	// Create the RecordSet request with the fully expanded name, e.g.
	// sub.domain.com. Route 53 requires a fully qualified domain name, but does
	// not require the trailing ".", which it will itself, so we don't call FQDN
	// here.
	rec := &route53.ResourceRecordSet{
		Name:            aws.String(en),
		Type:            aws.String(d.Get("type").(string)),
		TTL:             aws.Long(int64(d.Get("ttl").(int))),
		ResourceRecords: records,
	}
	return rec, nil
}

func FQDN(name string) string {
	n := len(name)
	if n == 0 || name[n-1] == '.' {
		return name
	} else {
		return name + "."
	}
}

// Route 53 stores the "*" wildcard indicator as ASCII 42 and returns the
// octal equivalent, "\\052". Here we look for that, and convert back to "*"
// as needed.
func cleanRecordName(name string) string {
	str := name
	if strings.HasPrefix(name, "\\052") {
		str = strings.Replace(name, "\\052", "*", 1)
		log.Printf("[DEBUG] Replacing octal \\052 for * in: %s", name)
	}
	return str
}

// Check if the current record name contains the zone suffix.
// If it does not, add the zone name to form a fully qualified name
// and keep AWS happy.
func expandRecordName(name, zone string) string {
	rn := strings.TrimSuffix(name, ".")
	zone = strings.TrimSuffix(zone, ".")
	if !strings.HasSuffix(rn, zone) {
		rn = strings.Join([]string{name, zone}, ".")
	}
	return rn
}
