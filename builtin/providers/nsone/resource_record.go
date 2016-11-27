package nsone

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"encoding/json"
	"github.com/mitchellh/hashstructure"
	nsone "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
	"gopkg.in/ns1/ns1-go.v2/rest/model/filter"
)

var recordTypeStringEnum *StringEnum = NewStringEnum([]string{
	"A",
	"AAAA",
	"ALIAS",
	"AFSDB",
	"CNAME",
	"DNAME",
	"HINFO",
	"MX",
	"NAPTR",
	"NS",
	"PTR",
	"RP",
	"SPF",
	"SRV",
	"TXT",
})

func recordResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			// Required
			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: recordTypeStringEnum.ValidateFunc,
			},
			// Optional
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"meta": metaSchema,
			"link": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"use_client_subnet": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"answers": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"answer": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"region": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"meta": metaSchema,
					},
				},
				Set: genericHasher,
			},
			"regions": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"meta": metaSchema,
					},
				},
				Set: genericHasher,
			},
			"filters": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"filter": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"disabled": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"config": &schema.Schema{
							Type:     schema.TypeMap,
							Optional: true,
						},
					},
				},
			},
			// Computed
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		Create:   RecordCreate,
		Read:     RecordRead,
		Update:   RecordUpdate,
		Delete:   RecordDelete,
		Importer: &schema.ResourceImporter{RecordStateFunc},
	}
}

func genericHasher(v interface{}) int {
	hash, err := hashstructure.Hash(v, nil)
	if err != nil {
		panic(fmt.Sprintf("error computing hash code for %#v: %s", v, err.Error()))
	}
	return int(hash)
}

func recordToResourceData(d *schema.ResourceData, r *dns.Record) error {
	d.SetId(r.ID)
	d.Set("domain", r.Domain)
	d.Set("zone", r.Zone)
	d.Set("type", r.Type)
	d.Set("ttl", r.TTL)
	if r.Link != "" {
		d.Set("link", r.Link)
	}
	if r.Meta != nil {
		d.State()
		t := metaStructToDynamic(r.Meta)
		d.Set("meta", t)
	}
	if len(r.Filters) > 0 {
		filters := make([]map[string]interface{}, len(r.Filters))
		for i, f := range r.Filters {
			m := make(map[string]interface{})
			m["filter"] = f.Type
			if f.Disabled {
				m["disabled"] = true
			}
			if f.Config != nil {
				m["config"] = f.Config
			}
			filters[i] = m
		}
		d.Set("filters", filters)
	}
	if len(r.Answers) > 0 {
		ans := &schema.Set{
			F: genericHasher,
		}
		log.Printf("Got back from nsone answers: %+v", r.Answers)
		for _, answer := range r.Answers {
			ans.Add(answerToMap(*answer))
		}
		log.Printf("Setting answers %+v", ans)
		err := d.Set("answers", ans)
		if err != nil {
			return fmt.Errorf("[DEBUG] Error setting answers for: %s, error: %#v", r.Domain, err)
		}
	}
	if len(r.Regions) > 0 {
		regions := make([]map[string]interface{}, 0, len(r.Regions))
		for regionName, region := range r.Regions {
			newRegion := make(map[string]interface{})
			newRegion["name"] = regionName
			newRegion["meta"] = metaStructToDynamic(&region.Meta)
			regions = append(regions, newRegion)
		}
		log.Printf("Setting regions %+v", regions)
		err := d.Set("regions", regions)
		if err != nil {
			return fmt.Errorf("[DEBUG] Error setting regions for: %s, error: %#v", r.Domain, err)
		}
	}
	return nil
}

func answerToMap(a dns.Answer) map[string]interface{} {
	m := make(map[string]interface{})
	m["answer"] = strings.Join(a.Rdata, " ")
	if a.RegionName != "" {
		m["region"] = a.RegionName
	}
	if a.Meta != nil {
		m["meta"] = metaStructToDynamic(a.Meta)
	}
	return m
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func resourceDataToRecord(r *dns.Record, d *schema.ResourceData) error {
	r.ID = d.Id()
	if answers := d.Get("answers").(*schema.Set); answers.Len() > 0 {
		al := make([]*dns.Answer, answers.Len())
		for i, answerRaw := range answers.List() {
			answer := answerRaw.(map[string]interface{})
			var a *dns.Answer
			v := answer["answer"].(string)
			switch d.Get("type") {
			case "TXT":
				a = dns.NewTXTAnswer(v)
			default:
				a = dns.NewAnswer(strings.Split(v, " "))
			}
			if v, ok := answer["region"]; ok {
				a.RegionName = v.(string)
			}

			if v, ok := answer["meta"]; ok {
				metaDynamicToStruct(a.Meta, v)
			}
			al[i] = a
		}
		r.Answers = al
		if _, ok := d.GetOk("link"); ok {
			return errors.New("Cannot have both link and answers in a record")
		}
	}
	if v, ok := d.GetOk("ttl"); ok {
		r.TTL = v.(int)
	}
	if v, ok := d.GetOk("link"); ok {
		r.LinkTo(v.(string))
	}
	if v, ok := d.GetOk("meta"); ok {
		metaDynamicToStruct(r.Meta, v)
	}
	useClientSubnetVal := d.Get("use_client_subnet").(bool)
	if v := strconv.FormatBool(useClientSubnetVal); v != "" {
		r.UseClientSubnet = &useClientSubnetVal
	}

	if rawFilters := d.Get("filters").([]interface{}); len(rawFilters) > 0 {
		f := make([]*filter.Filter, len(rawFilters))
		for i, filterRaw := range rawFilters {
			fi := filterRaw.(map[string]interface{})
			config := make(map[string]interface{})
			filter := filter.Filter{
				Type:   fi["filter"].(string),
				Config: config,
			}
			if disabled, ok := fi["disabled"]; ok {
				filter.Disabled = disabled.(bool)
			}
			if rawConfig, ok := fi["config"]; ok {
				for k, v := range rawConfig.(map[string]interface{}) {
					if i, err := strconv.Atoi(v.(string)); err == nil {
						filter.Config[k] = i
					} else {
						filter.Config[k] = v
					}
				}
			}
			f[i] = &filter
		}
		r.Filters = f
	}
	if regions := d.Get("regions").(*schema.Set); regions.Len() > 0 {
		rm := make(map[string]data.Region)
		for _, regionRaw := range regions.List() {
			region := regionRaw.(map[string]interface{})
			nsoneR := data.Region{
				Meta: data.Meta{},
			}
			if g := region["georegion"].(string); g != "" {
				nsoneR.Meta.Georegion = []string{g}
			}
			if g := region["country"].(string); g != "" {
				nsoneR.Meta.Country = []string{g}
			}
			if g := region["us_state"].(string); g != "" {
				nsoneR.Meta.USState = []string{g}
			}
			if g := region["up"].(bool); g {
				nsoneR.Meta.Up = g
			}

			rm[region["name"].(string)] = nsoneR

			if v, ok := region["meta"]; ok {
				metaDynamicToStruct(&nsoneR.Meta, v)
			}
		}
		r.Regions = rm
	}
	return nil
}

// RecordCreate creates DNS record in ns1
func RecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	r := dns.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err := resourceDataToRecord(r, d); err != nil {
		return err
	}
	if _, err := client.Records.Create(r); err != nil {
		return err
	}
	return recordToResourceData(d, r)
}

// RecordRead reads the DNS record from ns1
func RecordRead(d *schema.ResourceData, meta interface{}) error {
	//client := meta.(*nsone.Client)

	/*
		r, _, err := client.Records.Get(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
		if err != nil {
			return err
		}
	*/

	var r dns.Record
	responseBody := []byte(`{"domain":"block-api-dfw.dropbox-dns.com","zone":"dropbox-dns.com","use_client_subnet":true,"answers":[{"region":"dfw3a","meta":{"up":{"feed":"565494fa2db15678d7ddbfba"}},"answer":["45.58.75.4"],"feeds":[{"feed":"565494fa2db15678d7ddbfba","source":"3050efa1809ded58bba11547735b7fbd"}],"id":"57e2dac15927240001277046"},{"region":"dfw3a","meta":{"up":{"feed":"565494fa2db15678d7ddbfbc"}},"answer":["45.58.75.36"],"feeds":[{"feed":"565494fa2db15678d7ddbfbc","source":"3050efa1809ded58bba11547735b7fbd"}],"id":"57e2dac15927240001277047"},{"region":"dfw3b","meta":{"up":{"feed":"565494fa2db15678d7ddbfbe"}},"answer":["45.58.75.132"],"feeds":[{"feed":"565494fa2db15678d7ddbfbe","source":"3050efa1809ded58bba11547735b7fbd"}],"id":"57e2dac15927240001277048"},{"region":"dfw3b","meta":{"up":{"feed":"565494fa2db15678d7ddbfc0"}},"answer":["45.58.75.164"],"feeds":[{"feed":"565494fa2db15678d7ddbfc0","source":"3050efa1809ded58bba11547735b7fbd"}],"id":"57e2dac15927240001277049"}],"id":"57e2dac1592724000127704a","regions":{"dfw3a":{"meta":{"weight":13.0}},"dfw3b":{"meta":{"weight":54.0}}},"meta":{"low_watermark":1.0,"high_watermark":9999.0},"link":null,"filters":[{"filter":"up","config":{}},{"filter":"shed_load","config":{"metric":"loadavg"}},{"filter":"weighted_shuffle","config":{}},{"filter":"select_first_n","config":{"N":1}}],"ttl":30,"tier":3,"type":"A","networks":[0]}`)
	json.NewDecoder(bytes.NewReader(responseBody)).Decode(&r)

	return recordToResourceData(d, &r)
}

// RecordDelete deltes the DNS record from ns1
func RecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	_, err := client.Records.Delete(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	d.SetId("")
	return err
}

// RecordUpdate updates the given dns record in ns1
func RecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.Client)
	r := dns.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err := resourceDataToRecord(r, d); err != nil {
		return err
	}
	if _, err := client.Records.Update(r); err != nil {
		return err
	}
	return recordToResourceData(d, r)
}

func RecordStateFunc(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("Invalid record specifier.  Expecting 2 slashes (\"zone/domain/type\"), got %d.", len(parts)-1)
	}

	d.Set("zone", parts[0])
	d.Set("domain", parts[1])
	d.Set("type", parts[2])

	return []*schema.ResourceData{d}, nil
}
