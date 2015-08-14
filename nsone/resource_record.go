package nsone

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"regexp"
	"sort"
	"strings"
)

func recordResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
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
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^(A|AAAA|ALIAS|AFSDB|CNAME|DNAME|HINFO|MX|NAPTR|NS|PTR|RP|SPF|SRV|TXT)$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"only A, AAAA, ALIAS, AFSDB, CNAME, DNAME, HINFO, MX, NAPTR, NS, PTR, RP, SPF, SRV, TXT allowed in %q", k))
					}
					return
				},
			},
			"meta": metaSchema(),
			"link": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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
						"meta": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"field": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"feed": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
							Set: metaToHash,
						},
					},
				},
				Set: answersToHash,
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
						"georegion": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
								value := v.(string)
								if !regexp.MustCompile(`^(US-WEST|US-EAST|US-CENTRAL|EUROPE|AFRICA|ASIAPAC|SOUTH-AMERICA)$`).MatchString(value) {
									es = append(es, fmt.Errorf(
										"only US-WEST, US-EAST, US-CENTRAL, EUROPE, AFRICA, ASIAPAC, SOUTH-AMERICA allowed in %q", k))
								}
								return
							},
						},
						"country": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"us_state": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: regionsToHash,
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
					},
				},
			},
		},
		Create: RecordCreate,
		Read:   RecordRead,
		Update: RecordUpdate,
		Delete: RecordDelete,
	}
}

func regionsToHash(v interface{}) int {
	var buf bytes.Buffer
	r := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", r["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", r["georegion"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", r["country"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", r["us_state"].(string)))
	return hashcode.String(buf.String())
}

func answersToHash(v interface{}) int {
	var buf bytes.Buffer
	a := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", a["answer"].(string)))
	if a["region"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", a["region"].(string)))
	}
	ms := a["meta"].(*schema.Set)
	metas := make([]int, ms.Len())
	for _, meta := range ms.List() {
		metas = append(metas, metaToHash(meta))
	}
	sort.Ints(metas)
	for _, metahash := range metas {
		buf.WriteString(fmt.Sprintf("%d-", metahash))
	}
	hash := hashcode.String(buf.String())
	log.Println("Generated answersToHash %d from %+v", hash, ms)
	return hash
}

func metaToHash(v interface{}) int {
	var buf bytes.Buffer
	s := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", s["field"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", s["feed"].(string)))

	hash := hashcode.String(buf.String())
	log.Println("Generated metaToHash %d from %+v", hash, s)
	return hash
}

func recordToResourceData(d *schema.ResourceData, r *nsone.Record) error {
	d.SetId(r.Id)
	d.Set("domain", r.Domain)
	d.Set("zone", r.Zone)
	d.Set("type", r.Type)
	d.Set("ttl", r.Ttl)
	if r.Link != "" {
		d.Set("link", r.Link)
	}
	if len(r.Answers) > 0 {
		ans := &schema.Set{
			F: answersToHash,
		}
		log.Printf("Got back from nsone answers: %+v", r.Answers)
		for _, answer := range r.Answers {
			ans.Add(answerToMap(answer))
		}
		log.Printf("Setting answers %+v", ans)
		err := d.Set("answers", ans)
		if err != nil {
			return fmt.Errorf("[DEBUG] Error setting answers for: %s, error: %#v", r.Domain, err)
		}
	}
	if len(r.Regions) > 0 {
		regions := make([]map[string]interface{}, 0, len(r.Regions))
		for region_name, region := range r.Regions {
			new_region := make(map[string]interface{})
			new_region["name"] = region_name
			if len(region.Meta.GeoRegion) > 0 {
				new_region["georegion"] = region.Meta.GeoRegion[0]
			}
			if len(region.Meta.Country) > 0 {
				new_region["country"] = region.Meta.Country[0]
			}
			if len(region.Meta.USState) > 0 {
				new_region["us_state"] = region.Meta.USState[0]
			}
			regions = append(regions, new_region)
		}
		log.Printf("Setting regions %+v", regions)
		err := d.Set("regions", regions)
		if err != nil {
			return fmt.Errorf("[DEBUG] Error setting regions for: %s, error: %#v", r.Domain, err)
		}
	}
	return nil
}

func answerToMap(a nsone.Answer) map[string]interface{} {
	m := make(map[string]interface{})
	m["meta"] = make([]map[string]interface{}, 0)
	m["answer"] = strings.Join(a.Answer, " ")
	if a.Region != "" {
		m["region"] = a.Region
	}
	if a.Meta != nil {
		metas := &schema.Set{
			F: metaToHash,
		}
		for k, v := range a.Meta {
			meta := make(map[string]interface{})
			meta["field"] = k
			meta["feed"] = v.Feed
			metas.Add(meta)
		}
		m["meta"] = metas
	}
	return m
}

func resourceDataToRecord(r *nsone.Record, d *schema.ResourceData) error {
	r.Id = d.Id()
	if answers := d.Get("answers").(*schema.Set); answers.Len() > 0 {
		al := make([]nsone.Answer, answers.Len())
		for i, answer_raw := range answers.List() {
			answer := answer_raw.(map[string]interface{})
			a := nsone.NewAnswer()
			v := answer["answer"].(string)
			if d.Get("type") != "TXT" {
				a.Answer = strings.Split(v, " ")
			} else {
				a.Answer = []string{v}
			}
			if v, ok := answer["region"]; ok {
				a.Region = v.(string)
			}
			if metas := answer["meta"].(*schema.Set); metas.Len() > 0 {
				for _, meta_raw := range metas.List() {
					meta := meta_raw.(map[string]interface{})
					key := meta["field"].(string)
					value := meta["feed"].(string)
					a.Meta[key] = nsone.NewMetaFeed(value)
				}
			}
			al[i] = a
		}
		r.Answers = al
		if _, ok := d.GetOk("link"); ok {
			return errors.New("Cannot have both link and answers in a record")
		}
	}
	if v, ok := d.GetOk("ttl"); ok {
		r.Ttl = v.(int)
	}
	if v, ok := d.GetOk("link"); ok {
		r.LinkTo(v.(string))
	}
	if regions := d.Get("regions").(*schema.Set); regions.Len() > 0 {
		rm := make(map[string]nsone.Region)
		for _, region_raw := range regions.List() {
			region := region_raw.(map[string]interface{})
			nsone_r := nsone.Region{
				Meta: nsone.RegionMeta{},
			}
			if g := region["georegion"].(string); g != "" {
				nsone_r.Meta.GeoRegion = []string{g}
			}
			if g := region["country"].(string); g != "" {
				nsone_r.Meta.Country = []string{g}
			}
			if g := region["us_state"].(string); g != "" {
				nsone_r.Meta.USState = []string{g}
			}
			rm[region["name"].(string)] = nsone_r
		}
		r.Regions = rm
	}
	return nil
}

func setToMapByKey(s *schema.Set, key string) map[string]interface{} {
	result := make(map[string]interface{})
	for _, rawData := range s.List() {
		data := rawData.(map[string]interface{})
		result[data[key].(string)] = data
	}

	return result
}

func RecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r := nsone.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err := resourceDataToRecord(r, d); err != nil {
		return err
	}
	if err := client.CreateRecord(r); err != nil {
		return err
	}
	return recordToResourceData(d, r)
}

func RecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r, err := client.GetRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err != nil {
		return err
	}
	recordToResourceData(d, r)
	return nil
}

func RecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	d.SetId("")
	return err
}

func RecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r := nsone.NewRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err := resourceDataToRecord(r, d); err != nil {
		return err
	}
	if err := client.UpdateRecord(r); err != nil {
		return err
	}
	recordToResourceData(d, r)
	return nil
}
