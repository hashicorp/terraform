package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/route53"
)

func resourceAwsRoute53AliasTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53AliasTargetCreate,
		Read:   resourceAwsRoute53AliasTargetRead,
		Delete: resourceAwsRoute53AliasTargetDelete,

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

			"evaluate_health": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"target": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"target_zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsRoute53AliasTargetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn
	zone := d.Get("zone_id").(string)

	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneRequest{ID: aws.String(zone)})
	if err != nil {
		return err
	}

	// Get the record
	rec, err := resourceAwsRoute53AliasTargetBuildSet(d, *zoneRecord.HostedZone.Name)
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

func resourceAwsRoute53AliasTargetRead(d *schema.ResourceData, meta interface{}) error {
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

		aliasTarget := record.AliasTarget
		d.Set("target", strings.TrimSuffix(string(*aliasTarget.DNSName), "."))
		d.Set("target_zone_id", aliasTarget.HostedZoneID)

		break
	}

	if !found {
		d.SetId("")
	}

	return nil
}

func resourceAwsRoute53AliasTargetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	zone := d.Get("zone_id").(string)
	log.Printf("[DEBUG] Deleting resource records for zone: %s, name: %s",
		zone, d.Get("name").(string))
	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneRequest{ID: aws.String(zone)})
	if err != nil {
		return err
	}
	// Get the records
	rec, err := resourceAwsRoute53AliasTargetBuildSet(d, *zoneRecord.HostedZone.Name)
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

func resourceAwsRoute53AliasTargetBuildSet(d *schema.ResourceData, zoneName string) (*route53.ResourceRecordSet, error) {

	// Get expanded name
	en := expandRecordName(d.Get("name").(string), zoneName)
	target := strings.TrimSuffix(d.Get("target").(string), ".") + "."

	// Expand the name if it's in the same view
	if d.Get("zone_id").(string) == d.Get("target_zone_id").(string) {
		target = expandRecordName(target, zoneName)
	}

	// Create the RecordSet request with the fully expanded name, e.g.
	// sub.domain.com. Route 53 requires a fully qualified domain name, but does
	// not require the trailing ".", which it will itself, so we don't call FQDN
	// here.
	aliasTarget := &route53.AliasTarget{
		DNSName:              aws.String(target),
		HostedZoneID:         aws.String(d.Get("target_zone_id").(string)),
		EvaluateTargetHealth: aws.Boolean(d.Get("evaluate_health").(bool)),
	}

	rec := &route53.ResourceRecordSet{
		Name:        aws.String(en),
		Type:        aws.String(d.Get("type").(string)),
		AliasTarget: aliasTarget,
	}
	return rec, nil
}
