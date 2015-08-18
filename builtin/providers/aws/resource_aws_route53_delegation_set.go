package aws

import (
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

func resourceAwsRoute53DelegationSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRoute53DelegationSetCreate,
		Read:   resourceAwsRoute53DelegationSetRead,
		Delete: resourceAwsRoute53DelegationSetDelete,

		Schema: map[string]*schema.Schema{
			"reference_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"name_servers": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
		},
	}
}

func resourceAwsRoute53DelegationSetCreate(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	callerRef := resource.UniqueId()
	if v, ok := d.GetOk("reference_name"); ok {
		callerRef = strings.Join([]string{
			v.(string), "-", callerRef,
		}, "")
	}
	input := &route53.CreateReusableDelegationSetInput{
		CallerReference: aws.String(callerRef),
	}

	log.Printf("[DEBUG] Creating Route53 reusable delegation set: %#v", input)
	out, err := r53.CreateReusableDelegationSet(input)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Route53 reusable delegation set created: %#v", out)

	set := out.DelegationSet
	d.SetId(cleanDelegationSetId(*set.Id))
	d.Set("name_servers", expandNameServers(set.NameServers))
	return nil
}

func resourceAwsRoute53DelegationSetRead(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	input := &route53.GetReusableDelegationSetInput{
		Id: aws.String(cleanDelegationSetId(d.Id())),
	}
	log.Printf("[DEBUG] Reading Route53 reusable delegation set: %#v", input)
	out, err := r53.GetReusableDelegationSet(input)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Route53 reusable delegation set received: %#v", out)

	set := out.DelegationSet

	d.SetId(cleanDelegationSetId(*set.Id))
	d.Set("name_servers", expandNameServers(set.NameServers))

	return nil
}

func resourceAwsRoute53DelegationSetDelete(d *schema.ResourceData, meta interface{}) error {
	r53 := meta.(*AWSClient).r53conn

	input := &route53.DeleteReusableDelegationSetInput{
		Id: aws.String(cleanDelegationSetId(d.Id())),
	}
	log.Printf("[DEBUG] Deleting Route53 reusable delegation set: %#v", input)
	_, err := r53.DeleteReusableDelegationSet(input)
	return err
}

func expandNameServers(name_servers []*string) []string {
	log.Printf("[DEBUG] Processing %d name servers: %#v...", len(name_servers), name_servers)
	ns := make([]string, len(name_servers))
	for i, server := range name_servers {
		ns[i] = *server
	}
	sort.Strings(ns)
	log.Printf("[DEBUG] Returning processed name servers: %#v", ns)
	return ns
}

func cleanDelegationSetId(id string) string {
	return strings.TrimPrefix(id, "/delegationset/")
}
