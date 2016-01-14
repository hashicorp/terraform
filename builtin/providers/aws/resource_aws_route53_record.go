package aws

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/route53"
)

func resourceAwsRoute53Record() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53RecordCreate,
		Read:   resourceAwsRoute53RecordRead,
		Update: resourceAwsRoute53RecordUpdate,
		Delete: resourceAwsRoute53RecordDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToLower(value)
				},
			},

			"fqdn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if value == "" {
						es = append(es, fmt.Errorf("Cannot have empty zone_id"))
					}
					return
				},
			},

			"ttl": &schema.Schema{
				Type:          schema.TypeInt,
				Optional:      true,
				ConflictsWith: []string{"alias"},
			},

			// Weight uses a special sentinel value to indicate it's presense.
			// Because 0 is a valid value for Weight, we default to -1 so that any
			// inclusion of a weight (zero or not) will be a usable value
			"weight": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  -1,
			},

			"set_identifier": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"alias": &schema.Schema{
				Type:          schema.TypeSet,
				Optional:      true,
				ConflictsWith: []string{"records", "ttl"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"zone_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"evaluate_target_health": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},
					},
				},
				Set: resourceAwsRoute53AliasRecordHash,
			},

			"failover": &schema.Schema{ // PRIMARY | SECONDARY
				Type:     schema.TypeString,
				Optional: true,
			},

			"health_check_id": &schema.Schema{ // ID of health check
				Type:     schema.TypeString,
				Optional: true,
			},

			"records": &schema.Schema{
				Type:          schema.TypeSet,
				ConflictsWith: []string{"alias"},
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				Set:           schema.HashString,
			},
		},
	}
}

func resourceAwsRoute53RecordUpdate(d *schema.ResourceData, meta interface{}) error {
	// Route 53 supports CREATE, DELETE, and UPSERT actions. We use UPSERT, and
	// AWS dynamically determines if a record should be created or updated.
	// Amazon Route 53 can update an existing resource record set only when all
	// of the following values match: Name, Type
	// (and SetIdentifier, which we don't use yet).
	// See http://docs.aws.amazon.com/Route53/latest/APIReference/API_ChangeResourceRecordSets_Requests.html#change-rrsets-request-action
	//
	// Because we use UPSERT, for resouce update here we simply fall through to
	// our resource create function.
	return resourceAwsRoute53RecordCreate(d, meta)
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
		return fmt.Errorf("[WARN] No Route53 Zone found for id (%s)", zone)
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
		Changes: []*route53.Change{
			&route53.Change{
				Action:            aws.String("UPSERT"),
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

	wait := resource.StateChangeConf{
		Pending:    []string{"rejected"},
		Target:     "accepted",
		Timeout:    5 * time.Minute,
		MinTimeout: 1 * time.Second,
		Refresh: func() (interface{}, string, error) {
			resp, err := conn.ChangeResourceRecordSets(req)
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

	respRaw, err := wait.WaitForState()
	if err != nil {
		return err
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

	// Wait until we are done
	wait = resource.StateChangeConf{
		Delay:      30 * time.Second,
		Pending:    []string{"PENDING"},
		Target:     "INSYNC",
		Timeout:    30 * time.Minute,
		MinTimeout: 5 * time.Second,
		Refresh: func() (result interface{}, state string, err error) {
			changeRequest := &route53.GetChangeInput{
				Id: aws.String(cleanChangeID(*changeInfo.Id)),
			}
			return resourceAwsGoRoute53Wait(conn, changeRequest)
		},
	}
	_, err = wait.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsRoute53RecordRead(d, meta)
}

func resourceAwsRoute53RecordRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	zone := cleanZoneID(d.Get("zone_id").(string))

	// get expanded name
	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneInput{Id: aws.String(zone)})
	if err != nil {
		if r53err, ok := err.(awserr.Error); ok && r53err.Code() == "NoSuchHostedZone" {
			log.Printf("[DEBUG] No matching Route 53 Record found for: %s, removing from state file", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	en := expandRecordName(d.Get("name").(string), *zoneRecord.HostedZone.Name)
	log.Printf("[DEBUG] Expanded record name: %s", en)
	d.Set("fqdn", en)

	lopts := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(cleanZoneID(zone)),
		StartRecordName: aws.String(en),
		StartRecordType: aws.String(d.Get("type").(string)),
	}

	log.Printf("[DEBUG] List resource records sets for zone: %s, opts: %s",
		zone, lopts)
	resp, err := conn.ListResourceRecordSets(lopts)
	if err != nil {
		return err
	}

	// Scan for a matching record
	found := false
	for _, record := range resp.ResourceRecordSets {
		name := cleanRecordName(*record.Name)
		if FQDN(strings.ToLower(name)) != FQDN(strings.ToLower(*lopts.StartRecordName)) {
			continue
		}
		if strings.ToUpper(*record.Type) != strings.ToUpper(*lopts.StartRecordType) {
			continue
		}

		if record.SetIdentifier != nil && *record.SetIdentifier != d.Get("set_identifier") {
			continue
		}

		found = true

		err := d.Set("records", flattenResourceRecords(record.ResourceRecords))
		if err != nil {
			return fmt.Errorf("[DEBUG] Error setting records for: %s, error: %#v", en, err)
		}

		d.Set("ttl", record.TTL)
		// Only set the weight if it's non-nil, otherwise we end up with a 0 weight
		// which has actual contextual meaning with Route 53 records
		//   See http://docs.aws.amazon.com/fr_fr/Route53/latest/APIReference/API_ChangeResourceRecordSets_Examples.html
		if record.Weight != nil {
			d.Set("weight", record.Weight)
		}
		d.Set("set_identifier", record.SetIdentifier)
		d.Set("failover", record.Failover)
		d.Set("health_check_id", record.HealthCheckId)

		break
	}

	if !found {
		log.Printf("[DEBUG] No matching record found for: %s, removing from state file", en)
		d.SetId("")
	}

	return nil
}

func resourceAwsRoute53RecordDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).r53conn

	zone := cleanZoneID(d.Get("zone_id").(string))
	log.Printf("[DEBUG] Deleting resource records for zone: %s, name: %s",
		zone, d.Get("name").(string))
	var err error
	zoneRecord, err := conn.GetHostedZone(&route53.GetHostedZoneInput{Id: aws.String(zone)})
	if err != nil {
		if r53err, ok := err.(awserr.Error); ok && r53err.Code() == "NoSuchHostedZone" {
			log.Printf("[DEBUG] No matching Route 53 Record found for: %s, removing from state file", d.Id())
			d.SetId("")
			return nil
		}
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
		Changes: []*route53.Change{
			&route53.Change{
				Action:            aws.String("DELETE"),
				ResourceRecordSet: rec,
			},
		},
	}

	req := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(cleanZoneID(zone)),
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

	if v, ok := d.GetOk("failover"); ok {
		rec.Failover = aws.String(v.(string))
	}

	if v, ok := d.GetOk("health_check_id"); ok {
		rec.HealthCheckId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("set_identifier"); ok {
		rec.SetIdentifier = aws.String(v.(string))
	}

	w := d.Get("weight").(int)
	if w > -1 {
		rec.Weight = aws.Int64(int64(w))
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
	rn := strings.ToLower(strings.TrimSuffix(name, "."))
	zone = strings.TrimSuffix(zone, ".")
	if !strings.HasSuffix(rn, zone) {
		rn = strings.Join([]string{name, zone}, ".")
	}
	return rn
}

func resourceAwsRoute53AliasRecordHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["zone_id"].(string)))
	buf.WriteString(fmt.Sprintf("%t-", m["evaluate_target_health"].(bool)))

	return hashcode.String(buf.String())
}
