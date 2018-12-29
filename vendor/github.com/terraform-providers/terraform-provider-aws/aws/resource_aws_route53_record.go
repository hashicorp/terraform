package aws

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/route53"
)

var r53NoRecordsFound = errors.New("No matching records found")
var r53NoHostedZoneFound = errors.New("No matching Hosted Zone found")
var r53ValidRecordTypes = regexp.MustCompile("^(A|AAAA|CAA|CNAME|MX|NAPTR|NS|PTR|SOA|SPF|SRV|TXT)$")

func resourceAwsRoute53Record() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53RecordCreate,
		Read:   resourceAwsRoute53RecordRead,
		Update: resourceAwsRoute53RecordUpdate,
		Delete: resourceAwsRoute53RecordDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		SchemaVersion: 2,
		MigrateState:  resourceAwsRoute53RecordMigrateState,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					value := strings.TrimSuffix(v.(string), ".")
					return strings.ToLower(value)
				},
			},

			"fqdn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					route53.RRTypeSoa,
					route53.RRTypeA,
					route53.RRTypeTxt,
					route53.RRTypeNs,
					route53.RRTypeCname,
					route53.RRTypeMx,
					route53.RRTypeNaptr,
					route53.RRTypePtr,
					route53.RRTypeSrv,
					route53.RRTypeSpf,
					route53.RRTypeAaaa,
					route53.RRTypeCaa,
				}, false),
			},

			"zone_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},

			"ttl": {
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"alias"},
			},

			"weight": {
				Type:     schema.TypeInt,
				Optional: true,
				Removed:  "Now implemented as weighted_routing_policy; Please see https://www.terraform.io/docs/providers/aws/r/route53_record.html",
			},

			"set_identifier": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"alias": {
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"records", "ttl"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"zone_id": {
							Type:     schema.TypeString,
							Required: true,
						},

						"name": {
							Type:      schema.TypeString,
							Required:  true,
							StateFunc: normalizeAwsAliasName,
							DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
								return strings.ToLower(old) == strings.ToLower(new)
							},
						},

						"evaluate_target_health": {
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
				Set: resourceAwsRoute53AliasRecordHash,
			},

			"failover": { // PRIMARY | SECONDARY
				Type:     schema.TypeString,
				Optional: true,
				Removed:  "Now implemented as failover_routing_policy; see docs",
			},

			"failover_routing_policy": {
				Type:     schema.TypeList,
				Optional: true,
				ConflictsWith: []string{
					"geolocation_routing_policy",
					"latency_routing_policy",
					"weighted_routing_policy",
					"multivalue_answer_routing_policy",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
								value := v.(string)
								if value != "PRIMARY" && value != "SECONDARY" {
									es = append(es, fmt.Errorf("Failover policy type must be PRIMARY or SECONDARY"))
								}
								return
							},
						},
					},
				},
			},

			"latency_routing_policy": {
				Type:     schema.TypeList,
				Optional: true,
				ConflictsWith: []string{
					"failover_routing_policy",
					"geolocation_routing_policy",
					"weighted_routing_policy",
					"multivalue_answer_routing_policy",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"region": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"geolocation_routing_policy": { // AWS Geolocation
				Type:     schema.TypeList,
				Optional: true,
				ConflictsWith: []string{
					"failover_routing_policy",
					"latency_routing_policy",
					"weighted_routing_policy",
					"multivalue_answer_routing_policy",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"continent": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"country": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"subdivision": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"weighted_routing_policy": {
				Type:     schema.TypeList,
				Optional: true,
				ConflictsWith: []string{
					"failover_routing_policy",
					"geolocation_routing_policy",
					"latency_routing_policy",
					"multivalue_answer_routing_policy",
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"weight": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"multivalue_answer_routing_policy": {
				Type:     schema.TypeBool,
				Optional: true,
				ConflictsWith: []string{
					"failover_routing_policy",
					"geolocation_routing_policy",
					"latency_routing_policy",
					"weighted_routing_policy",
				},
			},

			"health_check_id": { // ID of health check
				Type:     schema.TypeString,
				Optional: true,
			},

			"records": {
				Type:          schema.TypeSet,
				ConflictsWith: []string{"alias"},
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				Set:           schema.HashString,
			},

			"allow_overwrite": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceAwsRoute53RecordUpdate(d *schema.ResourceData, meta interface{}) error {
	// Route 53 supports CREATE, DELETE, and UPSERT actions. We use UPSERT, and
	// AWS dynamically determines if a record should be created or updated.
	// Amazon Route 53 can update an existing resource record set only when all
	// of the following values match: Name, Type and SetIdentifier
	// See http://docs.aws.amazon.com/Route53/latest/APIReference/API_ChangeResourceRecordSets.html

	if !d.HasChange("type") && !d.HasChange("set_identifier") {
		// If neither type nor set_identifier changed we use UPSERT,
		// for resouce update here we simply fall through to
		// our resource create function.
		return resourceAwsRoute53RecordCreate(d, meta)
	}

	// Otherwise we delete the existing record and create a new record within
	// a transactional change
	conn := meta.(*AWSClient).r53conn
	zone := cleanZoneID(d.Get("zone_id").(string))

	var err error
	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneInput{Id: aws.String(zone)})
	if err != nil {
		return err
	}
	if zoneRecord.HostedZone == nil {
		return fmt.Errorf("No Route53 Zone found for id (%s)", zone)
	}

	// Build the to be deleted record
	en := expandRecordName(d.Get("name").(string), *zoneRecord.HostedZone.Name)
	typeo, _ := d.GetChange("type")

	oldRec := &route53.ResourceRecordSet{
		Name: aws.String(en),
		Type: aws.String(typeo.(string)),
	}

	if v, _ := d.GetChange("ttl"); v.(int) != 0 {
		oldRec.TTL = aws.Int64(int64(v.(int)))
	}

	// Resource records
	if v, _ := d.GetChange("records"); v != nil {
		recs := v.(*schema.Set).List()
		if len(recs) > 0 {
			oldRec.ResourceRecords = expandResourceRecords(recs, typeo.(string))
		}
	}

	// Alias record
	if v, _ := d.GetChange("alias"); v != nil {
		aliases := v.(*schema.Set).List()
		if len(aliases) == 1 {
			alias := aliases[0].(map[string]interface{})
			oldRec.AliasTarget = &route53.AliasTarget{
				DNSName:              aws.String(alias["name"].(string)),
				EvaluateTargetHealth: aws.Bool(alias["evaluate_target_health"].(bool)),
				HostedZoneId:         aws.String(alias["zone_id"].(string)),
			}
		}
	}

	if v, _ := d.GetChange("set_identifier"); v.(string) != "" {
		oldRec.SetIdentifier = aws.String(v.(string))
	}

	// Build the to be created record
	rec, err := resourceAwsRoute53RecordBuildSet(d, *zoneRecord.HostedZone.Name)
	if err != nil {
		return err
	}

	// Delete the old and create the new records in a single batch. We abuse
	// StateChangeConf for this to retry for us since Route53 sometimes returns
	// errors about another operation happening at the same time.
	changeBatch := &route53.ChangeBatch{
		Comment: aws.String("Managed by Terraform"),
		Changes: []*route53.Change{
			{
				Action:            aws.String("DELETE"),
				ResourceRecordSet: oldRec,
			},
			{
				Action:            aws.String("CREATE"),
				ResourceRecordSet: rec,
			},
		},
	}

	req := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(cleanZoneID(*zoneRecord.HostedZone.Id)),
		ChangeBatch:  changeBatch,
	}

	log.Printf("[DEBUG] Updating resource records for zone: %s, name: %s\n\n%s",
		zone, *rec.Name, req)

	respRaw, err := changeRoute53RecordSet(conn, req)
	if err != nil {
		return fmt.Errorf("[ERR]: Error building changeset: %s", err)
	}

	changeInfo := respRaw.(*route53.ChangeResourceRecordSetsOutput).ChangeInfo

	// Generate an ID
	vars := []string{
		zone,
		strings.ToLower(d.Get("name").(string)),
		d.Get("type").(string),
	}
	if v, ok := d.GetOk("set_identifier"); ok {
		vars = append(vars, v.(string))
	}

	d.SetId(strings.Join(vars, "_"))

	err = waitForRoute53RecordSetToSync(conn, cleanChangeID(*changeInfo.Id))
	if err != nil {
		return err
	}

	_, err = findRecord(d, meta)
	return err
}

func resourceAwsRoute53RecordCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn
	zone := cleanZoneID(d.Get("zone_id").(string))

	var err error
	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneInput{Id: aws.String(zone)})
	if err != nil {
		return err
	}
	if zoneRecord.HostedZone == nil {
		return fmt.Errorf("No Route53 Zone found for id (%s)", zone)
	}

	// Build the record
	rec, err := resourceAwsRoute53RecordBuildSet(d, *zoneRecord.HostedZone.Name)
	if err != nil {
		return err
	}

	// Protect existing DNS records which might be managed in another way
	// Use UPSERT only if the overwrite flag is true or if the current action is an update
	// Else CREATE is used and fail if the same record exists
	var action string
	if d.Get("allow_overwrite").(bool) || !d.IsNewResource() {
		action = "UPSERT"
	} else {
		action = "CREATE"
	}

	// Create the new records. We abuse StateChangeConf for this to
	// retry for us since Route53 sometimes returns errors about another
	// operation happening at the same time.
	changeBatch := &route53.ChangeBatch{
		Comment: aws.String("Managed by Terraform"),
		Changes: []*route53.Change{
			{
				Action:            aws.String(action),
				ResourceRecordSet: rec,
			},
		},
	}

	req := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(cleanZoneID(*zoneRecord.HostedZone.Id)),
		ChangeBatch:  changeBatch,
	}

	log.Printf("[DEBUG] Creating resource records for zone: %s, name: %s\n\n%s",
		zone, *rec.Name, req)

	respRaw, err := changeRoute53RecordSet(conn, req)
	if err != nil {
		return fmt.Errorf("[ERR]: Error building changeset: %s", err)
	}

	changeInfo := respRaw.(*route53.ChangeResourceRecordSetsOutput).ChangeInfo

	// Generate an ID
	vars := []string{
		zone,
		strings.ToLower(d.Get("name").(string)),
		d.Get("type").(string),
	}
	if v, ok := d.GetOk("set_identifier"); ok {
		vars = append(vars, v.(string))
	}

	d.SetId(strings.Join(vars, "_"))

	err = waitForRoute53RecordSetToSync(conn, cleanChangeID(*changeInfo.Id))
	if err != nil {
		return err
	}

	_, err = findRecord(d, meta)
	return err
}

func changeRoute53RecordSet(conn *route53.Route53, input *route53.ChangeResourceRecordSetsInput) (interface{}, error) {
	wait := resource.StateChangeConf{
		Pending:    []string{"rejected"},
		Target:     []string{"accepted"},
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.ChangeResourceRecordSets(input)
			if err != nil {
				if r53err, ok := err.(awserr.Error); ok {
					if r53err.Code() == "PriorRequestNotComplete" {
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

	return wait.WaitForState()
}

func waitForRoute53RecordSetToSync(conn *route53.Route53, requestId string) error {
	wait := resource.StateChangeConf{
		Delay:      30 * time.Second,
		Pending:    []string{"PENDING"},
		Target:     []string{"INSYNC"},
		Timeout:    30 * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			changeRequest := &route53.GetChangeInput{
				Id: aws.String(requestId),
			}
			return resourceAwsGoRoute53Wait(conn, changeRequest)
		},
	}
	_, err := wait.WaitForState()
	return err
}

func resourceAwsRoute53RecordRead(d *schema.ResourceData, meta interface{}) error {
	// If we don't have a zone ID we're doing an import. Parse it from the ID.
	if _, ok := d.GetOk("zone_id"); !ok {
		parts := parseRecordId(d.Id())
		//we check that we have parsed the id into the correct number of segments
		//we need at least 3 segments!
		if parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return fmt.Errorf("Error Importing aws_route_53 record. Please make sure the record ID is in the form ZONEID_RECORDNAME_TYPE (i.e. Z4KAPRWWNC7JR_dev_A")
		}

		d.Set("zone_id", parts[0])
		d.Set("name", parts[1])
		d.Set("type", parts[2])
		if parts[3] != "" {
			d.Set("set_identifier", parts[3])
		}
	}

	record, err := findRecord(d, meta)
	if err != nil {
		switch err {
		case r53NoHostedZoneFound, r53NoRecordsFound:
			log.Printf("[DEBUG] %s for: %s, removing from state file", err, d.Id())
			d.SetId("")
			return nil
		default:
			return err
		}
	}

	err = d.Set("records", flattenResourceRecords(record.ResourceRecords, *record.Type))
	if err != nil {
		return fmt.Errorf("Error setting records for: %s, error: %#v", d.Id(), err)
	}

	if alias := record.AliasTarget; alias != nil {
		name := normalizeAwsAliasName(*alias.DNSName)
		d.Set("alias", []interface{}{
			map[string]interface{}{
				"zone_id":                *alias.HostedZoneId,
				"name":                   name,
				"evaluate_target_health": *alias.EvaluateTargetHealth,
			},
		})
	}

	d.Set("ttl", record.TTL)

	if record.Failover != nil {
		v := []map[string]interface{}{{
			"type": aws.StringValue(record.Failover),
		}}
		if err := d.Set("failover_routing_policy", v); err != nil {
			return fmt.Errorf("Error setting failover records for: %s, error: %#v", d.Id(), err)
		}
	}

	if record.GeoLocation != nil {
		v := []map[string]interface{}{{
			"continent":   aws.StringValue(record.GeoLocation.ContinentCode),
			"country":     aws.StringValue(record.GeoLocation.CountryCode),
			"subdivision": aws.StringValue(record.GeoLocation.SubdivisionCode),
		}}
		if err := d.Set("geolocation_routing_policy", v); err != nil {
			return fmt.Errorf("Error setting gelocation records for: %s, error: %#v", d.Id(), err)
		}
	}

	if record.Region != nil {
		v := []map[string]interface{}{{
			"region": aws.StringValue(record.Region),
		}}
		if err := d.Set("latency_routing_policy", v); err != nil {
			return fmt.Errorf("Error setting latency records for: %s, error: %#v", d.Id(), err)
		}
	}

	if record.Weight != nil {
		v := []map[string]interface{}{{
			"weight": aws.Int64Value((record.Weight)),
		}}
		if err := d.Set("weighted_routing_policy", v); err != nil {
			return fmt.Errorf("Error setting weighted records for: %s, error: %#v", d.Id(), err)
		}
	}

	if record.MultiValueAnswer != nil {
		if err := d.Set("multivalue_answer_routing_policy", *record.MultiValueAnswer); err != nil {
			return fmt.Errorf("Error setting multivalue answer records for: %s, error: %#v", d.Id(), err)
		}
	}

	d.Set("set_identifier", record.SetIdentifier)
	d.Set("health_check_id", record.HealthCheckId)

	return nil
}

// findRecord takes a ResourceData struct for aws_resource_route53_record. It
// uses the referenced zone_id to query Route53 and find information on it's
// records.
//
// If records are found, it returns the matching
// route53.ResourceRecordSet and nil for the error.
//
// If no hosted zone is found, it returns a nil recordset and r53NoHostedZoneFound
// error.
//
// If no matching recordset is found, it returns nil and a r53NoRecordsFound
// error
//
// If there are other errors, it returns nil a nil recordset and passes on the
// error.
func findRecord(d *schema.ResourceData, meta interface{}) (*route53.ResourceRecordSet, error) {
	conn := meta.(*AWSClient).r53conn
	// Scan for a
	zone := cleanZoneID(d.Get("zone_id").(string))

	// get expanded name
	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneInput{Id: aws.String(zone)})
	if err != nil {
		if r53err, ok := err.(awserr.Error); ok && r53err.Code() == "NoSuchHostedZone" {
			return nil, r53NoHostedZoneFound
		}
		return nil, err
	}

	en := expandRecordName(d.Get("name").(string), *zoneRecord.HostedZone.Name)
	log.Printf("[DEBUG] Expanded record name: %s", en)
	d.Set("fqdn", en)

	recordName := FQDN(strings.ToLower(en))
	recordType := d.Get("type").(string)
	recordSetIdentifier := d.Get("set_identifier")

	// If this isn't a Weighted, Latency, Geo, or Failover resource with
	// a SetIdentifier we only need to look at the first record in the response since there can be
	// only one
	maxItems := "1"
	if recordSetIdentifier != "" {
		maxItems = "100"
	}

	lopts := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cleanZoneID(zone)),
		StartRecordName: aws.String(recordName),
		StartRecordType: aws.String(recordType),
		MaxItems:        aws.String(maxItems),
	}

	log.Printf("[DEBUG] List resource records sets for zone: %s, opts: %s",
		zone, lopts)

	var record *route53.ResourceRecordSet

	// We need to loop over all records starting from the record we are looking for because
	// Weighted, Latency, Geo, and Failover resource record sets have a special option
	// called SetIdentifier which allows multiple entries with the same name and type but
	// a different SetIdentifier
	// For all other records we are setting the maxItems to 1 so that we don't return extra
	// unneeded records
	err = conn.ListResourceRecordSetsPages(lopts, func(resp *route53.ListResourceRecordSetsOutput, lastPage bool) bool {
		for _, recordSet := range resp.ResourceRecordSets {

			responseName := strings.ToLower(cleanRecordName(*recordSet.Name))
			responseType := strings.ToUpper(*recordSet.Type)

			if recordName != responseName {
				continue
			}
			if recordType != responseType {
				continue
			}
			if recordSet.SetIdentifier != nil && *recordSet.SetIdentifier != recordSetIdentifier {
				continue
			}

			record = recordSet
			return false
		}

		nextRecordName := strings.ToLower(cleanRecordName(aws.StringValue(resp.NextRecordName)))
		nextRecordType := strings.ToUpper(aws.StringValue(resp.NextRecordType))

		if nextRecordName != recordName {
			return false
		}

		if nextRecordType != recordType {
			return false
		}

		return !lastPage
	})

	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, r53NoRecordsFound
	}
	return record, nil
}

func resourceAwsRoute53RecordDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn
	// Get the records
	rec, err := findRecord(d, meta)
	if err != nil {
		switch err {
		case r53NoHostedZoneFound, r53NoRecordsFound:
			return nil
		default:
			return err
		}
	}

	// Change batch for deleting
	changeBatch := &route53.ChangeBatch{
		Comment: aws.String("Deleted by Terraform"),
		Changes: []*route53.Change{
			{
				Action:            aws.String("DELETE"),
				ResourceRecordSet: rec,
			},
		},
	}

	zone := cleanZoneID(d.Get("zone_id").(string))

	req := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(cleanZoneID(zone)),
		ChangeBatch:  changeBatch,
	}

	respRaw, err := deleteRoute53RecordSet(conn, req)
	if err != nil {
		return fmt.Errorf("[ERR]: Error building changeset: %s", err)
	}

	changeInfo := respRaw.(*route53.ChangeResourceRecordSetsOutput).ChangeInfo
	if changeInfo == nil {
		log.Printf("[INFO] No ChangeInfo Found. Waiting for Sync not required")
		return nil
	}

	err = waitForRoute53RecordSetToSync(conn, cleanChangeID(*changeInfo.Id))
	if err != nil {
		return err
	}

	return err
}

func deleteRoute53RecordSet(conn *route53.Route53, input *route53.ChangeResourceRecordSetsInput) (interface{}, error) {
	wait := resource.StateChangeConf{
		Pending:    []string{"rejected"},
		Target:     []string{"accepted"},
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.ChangeResourceRecordSets(input)
			if err != nil {
				if r53err, ok := err.(awserr.Error); ok {
					if r53err.Code() == "PriorRequestNotComplete" {
						// There is some pending operation, so just retry
						// in a bit.
						return 42, "rejected", nil
					}

					if r53err.Code() == "InvalidChangeBatch" {
						// This means that the record is already gone.
						return resp, "accepted", nil
					}
				}

				return 42, "failure", err
			}

			return resp, "accepted", nil
		},
	}

	return wait.WaitForState()
}

func resourceAwsRoute53RecordBuildSet(d *schema.ResourceData, zoneName string) (*route53.ResourceRecordSet, error) {
	// get expanded name
	en := expandRecordName(d.Get("name").(string), zoneName)

	// Create the RecordSet request with the fully expanded name, e.g.
	// sub.domain.com. Route 53 requires a fully qualified domain name, but does
	// not require the trailing ".", which it will itself, so we don't call FQDN
	// here.
	rec := &route53.ResourceRecordSet{
		Name: aws.String(en),
		Type: aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("ttl"); ok {
		rec.TTL = aws.Int64(int64(v.(int)))
	}

	// Resource records
	if v, ok := d.GetOk("records"); ok {
		recs := v.(*schema.Set).List()
		rec.ResourceRecords = expandResourceRecords(recs, d.Get("type").(string))
	}

	// Alias record
	if v, ok := d.GetOk("alias"); ok {
		aliases := v.(*schema.Set).List()
		if len(aliases) > 1 {
			return nil, fmt.Errorf("You can only define a single alias target per record")
		}
		alias := aliases[0].(map[string]interface{})
		rec.AliasTarget = &route53.AliasTarget{
			DNSName:              aws.String(alias["name"].(string)),
			EvaluateTargetHealth: aws.Bool(alias["evaluate_target_health"].(bool)),
			HostedZoneId:         aws.String(alias["zone_id"].(string)),
		}
		log.Printf("[DEBUG] Creating alias: %#v", alias)
	} else {
		if _, ok := d.GetOk("ttl"); !ok {
			return nil, fmt.Errorf(`provider.aws: aws_route53_record: %s: "ttl": required field is not set`, d.Get("name").(string))
		}

		if _, ok := d.GetOk("records"); !ok {
			return nil, fmt.Errorf(`provider.aws: aws_route53_record: %s: "records": required field is not set`, d.Get("name").(string))
		}
	}

	if v, ok := d.GetOk("failover_routing_policy"); ok {
		if _, ok := d.GetOk("set_identifier"); !ok {
			return nil, fmt.Errorf(`provider.aws: aws_route53_record: %s: "set_identifier": required field is not set when "failover_routing_policy" is set`, d.Get("name").(string))
		}
		records := v.([]interface{})
		if len(records) > 1 {
			return nil, fmt.Errorf("You can only define a single failover_routing_policy per record")
		}
		failover := records[0].(map[string]interface{})

		rec.Failover = aws.String(failover["type"].(string))
	}

	if v, ok := d.GetOk("health_check_id"); ok {
		rec.HealthCheckId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("weighted_routing_policy"); ok {
		if _, ok := d.GetOk("set_identifier"); !ok {
			return nil, fmt.Errorf(`provider.aws: aws_route53_record: %s: "set_identifier": required field is not set when "weighted_routing_policy" is set`, d.Get("name").(string))
		}
		records := v.([]interface{})
		if len(records) > 1 {
			return nil, fmt.Errorf("You can only define a single weighted_routing_policy per record")
		}
		weight := records[0].(map[string]interface{})

		rec.Weight = aws.Int64(int64(weight["weight"].(int)))
	}

	if v, ok := d.GetOk("set_identifier"); ok {
		rec.SetIdentifier = aws.String(v.(string))
	}

	if v, ok := d.GetOk("latency_routing_policy"); ok {
		if _, ok := d.GetOk("set_identifier"); !ok {
			return nil, fmt.Errorf(`provider.aws: aws_route53_record: %s: "set_identifier": required field is not set when "latency_routing_policy" is set`, d.Get("name").(string))
		}
		records := v.([]interface{})
		if len(records) > 1 {
			return nil, fmt.Errorf("You can only define a single latency_routing_policy per record")
		}
		latency := records[0].(map[string]interface{})

		rec.Region = aws.String(latency["region"].(string))
	}

	if v, ok := d.GetOk("geolocation_routing_policy"); ok {
		if _, ok := d.GetOk("set_identifier"); !ok {
			return nil, fmt.Errorf(`provider.aws: aws_route53_record: %s: "set_identifier": required field is not set when "geolocation_routing_policy" is set`, d.Get("name").(string))
		}
		geolocations := v.([]interface{})
		if len(geolocations) > 1 {
			return nil, fmt.Errorf("You can only define a single geolocation_routing_policy per record")
		}
		geolocation := geolocations[0].(map[string]interface{})

		rec.GeoLocation = &route53.GeoLocation{
			ContinentCode:   nilString(geolocation["continent"].(string)),
			CountryCode:     nilString(geolocation["country"].(string)),
			SubdivisionCode: nilString(geolocation["subdivision"].(string)),
		}
		log.Printf("[DEBUG] Creating geolocation: %#v", geolocation)
	}

	if v, ok := d.GetOk("multivalue_answer_routing_policy"); ok {
		if _, ok := d.GetOk("set_identifier"); !ok {
			return nil, fmt.Errorf(`provider.aws: aws_route53_record: %s: "set_identifier": required field is not set when "multivalue_answer_routing_policy" is set`, d.Get("name").(string))
		}
		rec.MultiValueAnswer = aws.Bool(v.(bool))
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

// Route 53 stores certain characters with the octal equivalent in ASCII format.
// This function converts all of these characters back into the original character
// E.g. "*" is stored as "\\052" and "@" as "\\100"

func cleanRecordName(name string) string {
	str := name
	s, err := strconv.Unquote(`"` + str + `"`)
	if err != nil {
		return str
	}
	return s
}

// Check if the current record name contains the zone suffix.
// If it does not, add the zone name to form a fully qualified name
// and keep AWS happy.
func expandRecordName(name, zone string) string {
	rn := strings.ToLower(strings.TrimSuffix(name, "."))
	zone = strings.TrimSuffix(zone, ".")
	if !strings.HasSuffix(rn, zone) {
		if len(name) == 0 {
			rn = zone
		} else {
			rn = strings.Join([]string{rn, zone}, ".")
		}
	}
	return rn
}

func resourceAwsRoute53AliasRecordHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", normalizeAwsAliasName(m["name"].(string))))
	buf.WriteString(fmt.Sprintf("%s-", m["zone_id"].(string)))
	buf.WriteString(fmt.Sprintf("%t-", m["evaluate_target_health"].(bool)))

	return hashcode.String(buf.String())
}

// nilString takes a string as an argument and returns a string
// pointer. The returned pointer is nil if the string argument is
// empty, otherwise it is a pointer to a copy of the string.
func nilString(s string) *string {
	if s == "" {
		return nil
	}
	return aws.String(s)
}

func normalizeAwsAliasName(alias interface{}) string {
	input := strings.ToLower(alias.(string))
	output := strings.TrimPrefix(input, "dualstack.")
	return strings.TrimSuffix(output, ".")
}

func parseRecordId(id string) [4]string {
	var recZone, recType, recName, recSet string
	parts := strings.SplitN(id, "_", 2)
	if len(parts) == 2 {
		recZone = parts[0]
		lastUnderscore := strings.LastIndex(parts[1], "_")
		if lastUnderscore != -1 {
			recName, recType = parts[1][0:lastUnderscore], parts[1][lastUnderscore+1:]
			if !r53ValidRecordTypes.MatchString(recType) {
				recSet, recType = recType, ""
				lastUnderscore = strings.LastIndex(recName, "_")
				if lastUnderscore != -1 {
					recName, recType = recName[0:lastUnderscore], recName[lastUnderscore+1:]
				}
			}
		}
	}
	recName = strings.TrimSuffix(recName, ".")
	return [4]string{recZone, recName, recType, recSet}
}
