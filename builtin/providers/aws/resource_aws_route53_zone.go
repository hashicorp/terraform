package aws

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/route53"
)

func resourceAwsRoute53Zone() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53ZoneCreate,
		Read:   resourceAwsRoute53ZoneRead,
		Update: resourceAwsRoute53ZoneUpdate,
		Delete: resourceAwsRoute53ZoneDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"comment": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"vpc_region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"delegation_set_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"name_servers": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},

			"force_destroy": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRoute53ZoneCreate(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	req := &route53.CreateHostedZoneInput{
		Name:             aws.String(d.Get("name").(string)),
		HostedZoneConfig: &route53.HostedZoneConfig{Comment: aws.String(d.Get("comment").(string))},
		CallerReference:  aws.String(time.Now().Format(time.RFC3339Nano)),
	}
	if v := d.Get("vpc_id"); v != "" {
		req.VPC = &route53.VPC{
			VPCID:     aws.String(v.(string)),
			VPCRegion: aws.String(meta.(*AWSClient).region),
		}
		if w := d.Get("vpc_region"); w != "" {
			req.VPC.VPCRegion = aws.String(w.(string))
		}
		d.Set("vpc_region", req.VPC.VPCRegion)
	}

	if v, ok := d.GetOk("delegation_set_id"); ok {
		req.DelegationSetID = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating Route53 hosted zone: %s", *req.Name)
	var err error
	resp, err := r53.CreateHostedZone(req)
	if err != nil {
		return err
	}

	// Store the zone_id
	zone := cleanZoneID(*resp.HostedZone.ID)
	d.Set("zone_id", zone)
	d.SetId(zone)

	// Wait until we are done initializing
	wait := resource.StateChangeConf{
		Delay:      30 * time.Second,
		Pending:    []string{"PENDING"},
		Target:     "INSYNC",
		Timeout:    10 * time.Minute,
		MinTimeout: 2 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			changeRequest := &route53.GetChangeInput{
				ID: aws.String(cleanChangeID(*resp.ChangeInfo.ID)),
			}
			return resourceAwsGoRoute53Wait(r53, changeRequest)
		},
	}
	_, err = wait.WaitForState()
	if err != nil {
		return err
	}
	return resourceAwsRoute53ZoneUpdate(d, meta)
}

func resourceAwsRoute53ZoneRead(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn
	zone, err := r53.GetHostedZone(&route53.GetHostedZoneInput{ID: aws.String(d.Id())})
	if err != nil {
		// Handle a deleted zone
		if r53err, ok := err.(awserr.Error); ok && r53err.Code() == "NoSuchHostedZone" {
			d.SetId("")
			return nil
		}
		return err
	}

	if !*zone.HostedZone.Config.PrivateZone {
		ns := make([]string, len(zone.DelegationSet.NameServers))
		for i := range zone.DelegationSet.NameServers {
			ns[i] = *zone.DelegationSet.NameServers[i]
		}
		sort.Strings(ns)
		if err := d.Set("name_servers", ns); err != nil {
			return fmt.Errorf("[DEBUG] Error setting name servers for: %s, error: %#v", d.Id(), err)
		}
	} else {
		ns, err := getNameServers(d.Id(), d.Get("name").(string), r53)
		if err != nil {
			return err
		}
		if err := d.Set("name_servers", ns); err != nil {
			return fmt.Errorf("[DEBUG] Error setting name servers for: %s, error: %#v", d.Id(), err)
		}

		var associatedVPC *route53.VPC
		for _, vpc := range zone.VPCs {
			if *vpc.VPCID == d.Get("vpc_id") {
				associatedVPC = vpc
			}
		}
		if associatedVPC == nil {
			return fmt.Errorf("[DEBUG] VPC: %v is not associated with Zone: %v", d.Get("vpc_id"), d.Id())
		}
	}

	if zone.DelegationSet != nil && zone.DelegationSet.ID != nil {
		d.Set("delegation_set_id", cleanDelegationSetId(*zone.DelegationSet.ID))
	}

	// get tags
	req := &route53.ListTagsForResourceInput{
		ResourceID:   aws.String(d.Id()),
		ResourceType: aws.String("hostedzone"),
	}

	resp, err := r53.ListTagsForResource(req)
	if err != nil {
		return err
	}

	var tags []*route53.Tag
	if resp.ResourceTagSet != nil {
		tags = resp.ResourceTagSet.Tags
	}

	if err := d.Set("tags", tagsToMapR53(tags)); err != nil {
		return err
	}

	return nil
}

func resourceAwsRoute53ZoneUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	if err := setTagsR53(conn, d, "hostedzone"); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	return resourceAwsRoute53ZoneRead(d, meta)
}

func resourceAwsRoute53ZoneDelete(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	log.Printf("[DEBUG] Deleting Route53 hosted zone: %s (ID: %s)",
		d.Get("name").(string), d.Id())

	if ok := d.Get("force_destroy").(bool); ok {
		err := deleteZoneRecordSets(d, meta)
		if err != nil {
			return err
		}
	}

	_, err := r53.DeleteHostedZone(&route53.DeleteHostedZoneInput{ID: aws.String(d.Id())})

	if err != nil {
		if awserr, ok := err.(awserr.Error); ok {
			log.Printf("[DEBUG] AWS Error deleting hosted zone %s", awserr)
		}
		return err
	}

	return nil
}

func resourceAwsGoRoute53Wait(r53 *route53.Route53, ref *route53.GetChangeInput) (result interface{}, state string, err error) {

	status, err := r53.GetChange(ref)
	if err != nil {
		return nil, "UNKNOWN", err
	}
	return true, *status.ChangeInfo.Status, nil
}

// cleanChangeID is used to remove the leading /change/
func cleanChangeID(ID string) string {
	return cleanPrefix(ID, "/change/")
}

// cleanZoneID is used to remove the leading /hostedzone/
func cleanZoneID(ID string) string {
	return cleanPrefix(ID, "/hostedzone/")
}

// cleanPrefix removes a string prefix from an ID
func cleanPrefix(ID, prefix string) string {
	if strings.HasPrefix(ID, prefix) {
		ID = strings.TrimPrefix(ID, prefix)
	}
	return ID
}

func getNameServers(zoneId string, zoneName string, r53 *route53.Route53) ([]string, error) {
	resp, err := r53.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneID:    aws.String(zoneId),
		StartRecordName: aws.String(zoneName),
		StartRecordType: aws.String("NS"),
	})
	if err != nil {
		return nil, err
	}
	if len(resp.ResourceRecordSets) == 0 {
		return nil, nil
	}
	ns := make([]string, len(resp.ResourceRecordSets[0].ResourceRecords))
	for i := range resp.ResourceRecordSets[0].ResourceRecords {
		ns[i] = *resp.ResourceRecordSets[0].ResourceRecords[i].Value
	}
	sort.Strings(ns)
	return ns, nil
}

func deleteZoneRecordSets(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	zone := cleanZoneID(d.Get("zone_id").(string))
	log.Printf("[DEBUG] Deleting resource records for zone: %s, name: %s",
		zone, d.Get("name").(string))

	// Get the records
	recs, err := readRecordSets(d, meta)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Read %d resource records for zone: %s", len(recs), zone)

	recs = filterRequiredRecords(recs)

	log.Printf("[DEBUG] Have %d resource records after filter", len(recs))

	if len(recs) == 0 {
		return nil
	}

	changes := make([]*route53.Change, len(recs))
	for i, r := range recs {
		c := &route53.Change{
			Action:            aws.String("DELETE"),
			ResourceRecordSet: r,
		}
		changes[i] = c
	}

	// ChangeBatch for deletes
	changeBatch := &route53.ChangeBatch{
		Comment: aws.String("Deleted by Terraform"),
		Changes: changes,
	}

	req := &route53.ChangeResourceRecordSetsInput{
		HostedZoneID: aws.String(zone),
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
				if r53err, ok := err.(awserr.Error); ok {
					if r53err.Code() == "PriorRequestNotComplete" {
						// There is some pending operation, so just retry
						// in a bit.
						return 42, "rejected", nil
					}

					if r53err.Code() == "InvalidChangeBatch" {
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

func readRecordSets(d *schema.ResourceData, meta interface{}) ([]*route53.ResourceRecordSet, error) {
	conn := meta.(*AWSClient).r53conn

	zone := cleanZoneID(d.Get("zone_id").(string))

	// get expanded name
	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneInput{ID: aws.String(zone)})
	if err != nil {
		return nil, err
	}
	en := expandRecordName(zone, *zoneRecord.HostedZone.Name)
	log.Printf("[DEBUG] Expanded record name: %s", en)

	lopts := &route53.ListResourceRecordSetsInput{
		HostedZoneID: aws.String(zone),
		// StartRecordName: aws.String(en),
	}

	resp, err := conn.ListResourceRecordSets(lopts)
	if err != nil {
		return nil, err
	}
	return resp.ResourceRecordSets, nil
}

func filterRequiredRecords(rrs []*route53.ResourceRecordSet) []*route53.ResourceRecordSet {
	filtered := make([]*route53.ResourceRecordSet, 0)
	for _, rr := range rrs {
		if *rr.Type != "NS" && *rr.Type != "SOA" {
			filtered = append(filtered, rr)
		}
	}
	return filtered
}
