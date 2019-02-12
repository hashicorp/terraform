package aws

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
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
			"name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: suppressRoute53ZoneNameWithTrailingDot,
			},

			"comment": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Managed by Terraform",
			},

			"vpc": {
				Type:     schema.TypeSet,
				Optional: true,
				// Deprecated: Remove Computed: true in next major version of the provider
				Computed:      true,
				MinItems:      1,
				ConflictsWith: []string{"delegation_set_id", "vpc_id", "vpc_region"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"vpc_id": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.NoZeroValues,
						},
						"vpc_region": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: route53HostedZoneVPCHash,
			},

			"vpc_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"delegation_set_id", "vpc"},
				Deprecated:    "use 'vpc' attribute instead",
			},

			"vpc_region": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Computed:      true,
				ConflictsWith: []string{"delegation_set_id", "vpc"},
				Deprecated:    "use 'vpc' attribute instead",
			},

			"zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"delegation_set_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"vpc_id"},
			},

			"name_servers": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},

			"tags": tagsSchema(),

			"force_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func resourceAwsRoute53ZoneCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn
	region := meta.(*AWSClient).region

	input := &route53.CreateHostedZoneInput{
		CallerReference: aws.String(resource.UniqueId()),
		Name:            aws.String(d.Get("name").(string)),
		HostedZoneConfig: &route53.HostedZoneConfig{
			Comment: aws.String(d.Get("comment").(string)),
		},
	}

	if v, ok := d.GetOk("delegation_set_id"); ok {
		input.DelegationSetId = aws.String(v.(string))
	}

	// Private Route53 Hosted Zones can only be created with their first VPC association,
	// however we need to associate the remaining after creation.

	var vpcs []*route53.VPC = expandRoute53VPCs(d.Get("vpc").(*schema.Set).List(), region)

	// Backwards compatibility
	if vpcID, ok := d.GetOk("vpc_id"); ok {
		vpc := &route53.VPC{
			VPCId:     aws.String(vpcID.(string)),
			VPCRegion: aws.String(region),
		}

		if vpcRegion, ok := d.GetOk("vpc_region"); ok {
			vpc.VPCRegion = aws.String(vpcRegion.(string))
		}

		vpcs = []*route53.VPC{vpc}
	}

	if len(vpcs) > 0 {
		input.VPC = vpcs[0]
	}

	log.Printf("[DEBUG] Creating Route53 hosted zone: %s", input)
	output, err := conn.CreateHostedZone(input)

	if err != nil {
		return fmt.Errorf("error creating Route53 Hosted Zone: %s", err)
	}

	d.SetId(cleanZoneID(aws.StringValue(output.HostedZone.Id)))

	if err := route53WaitForChangeSynchronization(conn, cleanChangeID(aws.StringValue(output.ChangeInfo.Id))); err != nil {
		return fmt.Errorf("error waiting for Route53 Hosted Zone (%s) creation: %s", d.Id(), err)
	}

	if err := setTagsR53(conn, d, route53.TagResourceTypeHostedzone); err != nil {
		return fmt.Errorf("error setting tags for Route53 Hosted Zone (%s): %s", d.Id(), err)
	}

	// Associate additional VPCs beyond the first
	if len(vpcs) > 1 {
		for _, vpc := range vpcs[1:] {
			err := route53HostedZoneVPCAssociate(conn, d.Id(), vpc)

			if err != nil {
				return err
			}
		}
	}

	return resourceAwsRoute53ZoneRead(d, meta)
}

func resourceAwsRoute53ZoneRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	input := &route53.GetHostedZoneInput{
		Id: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Getting Route53 Hosted Zone: %s", input)
	output, err := conn.GetHostedZone(input)

	if isAWSErr(err, route53.ErrCodeNoSuchHostedZone, "") {
		log.Printf("[WARN] Route53 Hosted Zone (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error getting Route53 Hosted Zone (%s): %s", d.Id(), err)
	}

	if output == nil || output.HostedZone == nil {
		log.Printf("[WARN] Route53 Hosted Zone (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("comment", "")
	d.Set("delegation_set_id", "")
	d.Set("name", output.HostedZone.Name)
	d.Set("zone_id", cleanZoneID(aws.StringValue(output.HostedZone.Id)))

	var nameServers []string

	if output.DelegationSet != nil {
		d.Set("delegation_set_id", cleanDelegationSetId(aws.StringValue(output.DelegationSet.Id)))

		nameServers = aws.StringValueSlice(output.DelegationSet.NameServers)
	}

	if output.HostedZone.Config != nil {
		d.Set("comment", output.HostedZone.Config.Comment)

		if aws.BoolValue(output.HostedZone.Config.PrivateZone) {
			var err error
			nameServers, err = getNameServers(d.Id(), d.Get("name").(string), conn)

			if err != nil {
				return fmt.Errorf("error getting Route53 Hosted Zone (%s) name servers: %s", d.Id(), err)
			}
		}
	}

	sort.Strings(nameServers)
	if err := d.Set("name_servers", nameServers); err != nil {
		return fmt.Errorf("error setting name_servers: %s", err)
	}

	// Backwards compatibility: only set vpc_id/vpc_region if either is true:
	//  * Previously configured
	//  * Only one VPC association
	existingVpcID := d.Get("vpc_id").(string)

	// Detect drift in configuration
	d.Set("vpc_id", "")
	d.Set("vpc_region", "")

	if len(output.VPCs) == 1 && output.VPCs[0] != nil {
		d.Set("vpc_id", output.VPCs[0].VPCId)
		d.Set("vpc_region", output.VPCs[0].VPCRegion)
	} else if len(output.VPCs) > 1 {
		for _, vpc := range output.VPCs {
			if vpc == nil {
				continue
			}
			if aws.StringValue(vpc.VPCId) == existingVpcID {
				d.Set("vpc_id", vpc.VPCId)
				d.Set("vpc_region", vpc.VPCRegion)
			}
		}
	}

	if err := d.Set("vpc", flattenRoute53VPCs(output.VPCs)); err != nil {
		return fmt.Errorf("error setting vpc: %s", err)
	}

	// get tags
	req := &route53.ListTagsForResourceInput{
		ResourceId:   aws.String(d.Id()),
		ResourceType: aws.String(route53.TagResourceTypeHostedzone),
	}

	log.Printf("[DEBUG] Listing tags for Route53 Hosted Zone: %s", req)
	resp, err := conn.ListTagsForResource(req)

	if err != nil {
		return fmt.Errorf("error listing tags for Route53 Hosted Zone (%s): %s", d.Id(), err)
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
	region := meta.(*AWSClient).region

	d.Partial(true)

	if d.HasChange("comment") {
		input := route53.UpdateHostedZoneCommentInput{
			Id:      aws.String(d.Id()),
			Comment: aws.String(d.Get("comment").(string)),
		}

		_, err := conn.UpdateHostedZoneComment(&input)

		if err != nil {
			return fmt.Errorf("error updating Route53 Hosted Zone (%s) comment: %s", d.Id(), err)
		}

		d.SetPartial("comment")
	}

	if d.HasChange("tags") {
		if err := setTagsR53(conn, d, route53.TagResourceTypeHostedzone); err != nil {
			return err
		}

		d.SetPartial("tags")
	}

	if d.HasChange("vpc") {
		o, n := d.GetChange("vpc")
		oldVPCs := o.(*schema.Set)
		newVPCs := n.(*schema.Set)

		// VPCs cannot be empty, so add first and then remove
		for _, vpcRaw := range newVPCs.Difference(oldVPCs).List() {
			if vpcRaw == nil {
				continue
			}

			vpc := expandRoute53VPC(vpcRaw.(map[string]interface{}), region)
			err := route53HostedZoneVPCAssociate(conn, d.Id(), vpc)

			if err != nil {
				return err
			}
		}

		for _, vpcRaw := range oldVPCs.Difference(newVPCs).List() {
			if vpcRaw == nil {
				continue
			}

			vpc := expandRoute53VPC(vpcRaw.(map[string]interface{}), region)
			err := route53HostedZoneVPCDisassociate(conn, d.Id(), vpc)

			if err != nil {
				return err
			}
		}

		d.SetPartial("vpc")
	}

	d.Partial(false)

	return resourceAwsRoute53ZoneRead(d, meta)
}

func resourceAwsRoute53ZoneDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	if d.Get("force_destroy").(bool) {
		if err := deleteAllRecordsInHostedZoneId(d.Id(), d.Get("name").(string), conn); err != nil {
			return fmt.Errorf("error deleting records in Route53 Hosted Zone (%s): %s", d.Id(), err)
		}
	}

	input := &route53.DeleteHostedZoneInput{
		Id: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting Route53 Hosted Zone: %s", input)
	_, err := conn.DeleteHostedZone(input)

	if isAWSErr(err, route53.ErrCodeNoSuchHostedZone, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting Route53 Hosted Zone (%s): %s", d.Id(), err)
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

func expandRoute53VPCs(l []interface{}, currentRegion string) []*route53.VPC {
	vpcs := []*route53.VPC{}

	for _, mRaw := range l {
		if mRaw == nil {
			continue
		}

		vpcs = append(vpcs, expandRoute53VPC(mRaw.(map[string]interface{}), currentRegion))
	}

	return vpcs
}

func expandRoute53VPC(m map[string]interface{}, currentRegion string) *route53.VPC {
	vpc := &route53.VPC{
		VPCId:     aws.String(m["vpc_id"].(string)),
		VPCRegion: aws.String(currentRegion),
	}

	if v, ok := m["vpc_region"]; ok && v.(string) != "" {
		vpc.VPCRegion = aws.String(v.(string))
	}

	return vpc
}

func flattenRoute53VPCs(vpcs []*route53.VPC) []interface{} {
	l := []interface{}{}

	for _, vpc := range vpcs {
		if vpc == nil {
			continue
		}

		m := map[string]interface{}{
			"vpc_id":     aws.StringValue(vpc.VPCId),
			"vpc_region": aws.StringValue(vpc.VPCRegion),
		}

		l = append(l, m)
	}

	return l
}

func route53HostedZoneVPCAssociate(conn *route53.Route53, zoneID string, vpc *route53.VPC) error {
	input := &route53.AssociateVPCWithHostedZoneInput{
		HostedZoneId: aws.String(zoneID),
		VPC:          vpc,
	}

	log.Printf("[DEBUG] Associating Route53 Hosted Zone with VPC: %s", input)
	output, err := conn.AssociateVPCWithHostedZone(input)

	if err != nil {
		return fmt.Errorf("error associating Route53 Hosted Zone (%s) to VPC (%s): %s", zoneID, aws.StringValue(vpc.VPCId), err)
	}

	if err := route53WaitForChangeSynchronization(conn, cleanChangeID(aws.StringValue(output.ChangeInfo.Id))); err != nil {
		return fmt.Errorf("error waiting for Route53 Hosted Zone (%s) association to VPC (%s): %s", zoneID, aws.StringValue(vpc.VPCId), err)
	}

	return nil
}

func route53HostedZoneVPCDisassociate(conn *route53.Route53, zoneID string, vpc *route53.VPC) error {
	input := &route53.DisassociateVPCFromHostedZoneInput{
		HostedZoneId: aws.String(zoneID),
		VPC:          vpc,
	}

	log.Printf("[DEBUG] Disassociating Route53 Hosted Zone with VPC: %s", input)
	output, err := conn.DisassociateVPCFromHostedZone(input)

	if err != nil {
		return fmt.Errorf("error disassociating Route53 Hosted Zone (%s) from VPC (%s): %s", zoneID, aws.StringValue(vpc.VPCId), err)
	}

	if err := route53WaitForChangeSynchronization(conn, cleanChangeID(aws.StringValue(output.ChangeInfo.Id))); err != nil {
		return fmt.Errorf("error waiting for Route53 Hosted Zone (%s) disassociation from VPC (%s): %s", zoneID, aws.StringValue(vpc.VPCId), err)
	}

	return nil
}

func route53HostedZoneVPCHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["vpc_id"].(string)))

	return hashcode.String(buf.String())
}

func route53WaitForChangeSynchronization(conn *route53.Route53, changeID string) error {
	conf := resource.StateChangeConf{
		Delay:      30 * time.Second,
		Pending:    []string{route53.ChangeStatusPending},
		Target:     []string{route53.ChangeStatusInsync},
		Timeout:    15 * time.Minute,
		MinTimeout: 2 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			input := &route53.GetChangeInput{
				Id: aws.String(changeID),
			}

			log.Printf("[DEBUG] Getting Route53 Change status: %s", input)
			output, err := conn.GetChange(input)

			if err != nil {
				return nil, "UNKNOWN", err
			}

			if output == nil || output.ChangeInfo == nil {
				return nil, "UNKNOWN", fmt.Errorf("Route53 GetChange response empty for ID: %s", changeID)
			}

			return true, aws.StringValue(output.ChangeInfo.Status), nil
		},
	}

	_, err := conf.WaitForState()

	return err
}
