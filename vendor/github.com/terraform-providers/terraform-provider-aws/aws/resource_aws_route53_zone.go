package aws

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
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
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressRoute53ZoneNameWithTrailingDot,
			},

			"comment": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},

			"vpc_id": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"delegation_set_id"},
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
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"vpc_id"},
			},

			"name_servers": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},

			"tags": tagsSchema(),

			"force_destroy": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceAwsRoute53ZoneCreate(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	req := &route53.CreateHostedZoneInput{
		Name:             aws.String(d.Get("name").(string)),
		HostedZoneConfig: &route53.HostedZoneConfig{Comment: aws.String(d.Get("comment").(string))},
		CallerReference:  aws.String(resource.UniqueId()),
	}
	if v := d.Get("vpc_id"); v != "" {
		req.VPC = &route53.VPC{
			VPCId:     aws.String(v.(string)),
			VPCRegion: aws.String(meta.(*AWSClient).region),
		}
		if w := d.Get("vpc_region"); w != "" {
			req.VPC.VPCRegion = aws.String(w.(string))
		}
		d.Set("vpc_region", req.VPC.VPCRegion)
	}

	if v, ok := d.GetOk("delegation_set_id"); ok {
		req.DelegationSetId = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating Route53 hosted zone: %s", *req.Name)
	var err error
	resp, err := r53.CreateHostedZone(req)
	if err != nil {
		return err
	}

	// Store the zone_id
	zone := cleanZoneID(*resp.HostedZone.Id)
	d.Set("zone_id", zone)
	d.SetId(zone)

	// Wait until we are done initializing
	wait := resource.StateChangeConf{
		Delay:      30 * time.Second,
		Pending:    []string{"PENDING"},
		Target:     []string{"INSYNC"},
		Timeout:    15 * time.Minute,
		MinTimeout: 2 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			changeRequest := &route53.GetChangeInput{
				Id: aws.String(cleanChangeID(*resp.ChangeInfo.Id)),
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
	zone, err := r53.GetHostedZone(&route53.GetHostedZoneInput{Id: aws.String(d.Id())})
	if err != nil {
		// Handle a deleted zone
		if r53err, ok := err.(awserr.Error); ok && r53err.Code() == "NoSuchHostedZone" {
			d.SetId("")
			return nil
		}
		return err
	}

	// In the import case this will be empty
	if _, ok := d.GetOk("zone_id"); !ok {
		d.Set("zone_id", d.Id())
	}
	if _, ok := d.GetOk("name"); !ok {
		d.Set("name", zone.HostedZone.Name)
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

		// In the import case we just associate it with the first VPC
		if _, ok := d.GetOk("vpc_id"); !ok {
			if len(zone.VPCs) > 1 {
				return fmt.Errorf(
					"Can't import a route53_zone with more than one VPC attachment")
			}

			if len(zone.VPCs) > 0 {
				d.Set("vpc_id", zone.VPCs[0].VPCId)
				d.Set("vpc_region", zone.VPCs[0].VPCRegion)
			}
		}

		var associatedVPC *route53.VPC
		for _, vpc := range zone.VPCs {
			if *vpc.VPCId == d.Get("vpc_id") {
				associatedVPC = vpc
				break
			}
		}
		if associatedVPC == nil {
			return fmt.Errorf("[DEBUG] VPC: %v is not associated with Zone: %v", d.Get("vpc_id"), d.Id())
		}
	}

	if zone.DelegationSet != nil && zone.DelegationSet.Id != nil {
		d.Set("delegation_set_id", cleanDelegationSetId(*zone.DelegationSet.Id))
	}

	if zone.HostedZone != nil && zone.HostedZone.Config != nil && zone.HostedZone.Config.Comment != nil {
		d.Set("comment", zone.HostedZone.Config.Comment)
	}

	// get tags
	req := &route53.ListTagsForResourceInput{
		ResourceId:   aws.String(d.Id()),
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

	d.Partial(true)

	if d.HasChange("comment") {
		zoneInput := route53.UpdateHostedZoneCommentInput{
			Id:      aws.String(d.Id()),
			Comment: aws.String(d.Get("comment").(string)),
		}

		_, err := conn.UpdateHostedZoneComment(&zoneInput)
		if err != nil {
			return err
		} else {
			d.SetPartial("comment")
		}
	}

	if err := setTagsR53(conn, d, "hostedzone"); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	d.Partial(false)

	return resourceAwsRoute53ZoneRead(d, meta)
}

func resourceAwsRoute53ZoneDelete(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	if d.Get("force_destroy").(bool) {
		if err := deleteAllRecordsInHostedZoneId(d.Id(), d.Get("name").(string), r53); err != nil {
			return errwrap.Wrapf("{{err}}", err)
		}
	}

	log.Printf("[DEBUG] Deleting Route53 hosted zone: %s (ID: %s)",
		d.Get("name").(string), d.Id())
	_, err := r53.DeleteHostedZone(&route53.DeleteHostedZoneInput{Id: aws.String(d.Id())})
	if err != nil {
		if r53err, ok := err.(awserr.Error); ok && r53err.Code() == "NoSuchHostedZone" {
			return nil
		}
		return err
	}

	return nil
}

func deleteAllRecordsInHostedZoneId(hostedZoneId, hostedZoneName string, conn *route53.Route53) error {
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneId),
	}

	var lastDeleteErr, lastErrorFromWaiter error
	var pageNum = 0
	err := conn.ListResourceRecordSetsPages(input, func(page *route53.ListResourceRecordSetsOutput, isLastPage bool) bool {
		sets := page.ResourceRecordSets
		pageNum += 1

		changes := make([]*route53.Change, 0)
		// 100 items per page returned by default
		for _, set := range sets {
			if strings.TrimSuffix(*set.Name, ".") == strings.TrimSuffix(hostedZoneName, ".") && (*set.Type == "NS" || *set.Type == "SOA") {
				// Zone NS & SOA records cannot be deleted
				continue
			}
			changes = append(changes, &route53.Change{
				Action:            aws.String("DELETE"),
				ResourceRecordSet: set,
			})
		}
		log.Printf("[DEBUG] Deleting %d records (page %d) from %s",
			len(changes), pageNum, hostedZoneId)

		req := &route53.ChangeResourceRecordSetsInput{
			HostedZoneId: aws.String(hostedZoneId),
			ChangeBatch: &route53.ChangeBatch{
				Comment: aws.String("Deleted by Terraform"),
				Changes: changes,
			},
		}

		var resp interface{}
		resp, lastDeleteErr = deleteRoute53RecordSet(conn, req)
		if out, ok := resp.(*route53.ChangeResourceRecordSetsOutput); ok {
			log.Printf("[DEBUG] Waiting for change batch to become INSYNC: %#v", out)
			if out.ChangeInfo != nil && out.ChangeInfo.Id != nil {
				lastErrorFromWaiter = waitForRoute53RecordSetToSync(conn, cleanChangeID(*out.ChangeInfo.Id))
			} else {
				log.Printf("[DEBUG] Change info was empty")
			}
		} else {
			log.Printf("[DEBUG] Unable to wait for change batch because of an error: %s", lastDeleteErr)
		}

		return !isLastPage
	})
	if err != nil {
		return fmt.Errorf("Failed listing/deleting record sets: %s\nLast error from deletion: %s\nLast error from waiter: %s",
			err, lastDeleteErr, lastErrorFromWaiter)
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
		HostedZoneId:    aws.String(zoneId),
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
