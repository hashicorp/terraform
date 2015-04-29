package aws

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/route53"
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

			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"name_servers": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRoute53ZoneCreate(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	comment := &route53.HostedZoneConfig{Comment: aws.String("Managed by Terraform")}
	req := &route53.CreateHostedZoneInput{
		Name:             aws.String(d.Get("name").(string)),
		HostedZoneConfig: comment,
		CallerReference:  aws.String(time.Now().Format(time.RFC3339Nano)),
	}

	log.Printf("[DEBUG] Creating Route53 hosted zone: %s", *req.Name)
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
		if r53err, ok := err.(aws.APIError); ok && r53err.Code == "NoSuchHostedZone" {
			d.SetId("")
			return nil
		}
		return err
	}

	ns := make([]string, len(zone.DelegationSet.NameServers))
	for i := range zone.DelegationSet.NameServers {
		ns[i] = *zone.DelegationSet.NameServers[i]
	}
	sort.Strings(ns)
	if err := d.Set("name_servers", ns); err != nil {
		return fmt.Errorf("[DEBUG] Error setting name servers for: %s, error: %#v", d.Id(), err)
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

	if err := setTagsR53(conn, d); err != nil {
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
	_, err := r53.DeleteHostedZone(&route53.DeleteHostedZoneInput{ID: aws.String(d.Id())})
	if err != nil {
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
