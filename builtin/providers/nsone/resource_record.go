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
	"strconv"
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
										//ConflictsWith: []string{"value"},
									},
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
										//ConflictsWith: []string{"feed"},
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
						"up": &schema.Schema{
							Type:     schema.TypeBool,
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
	buf.WriteString(fmt.Sprintf("%t-", r["up"].(bool)))
	return hashcode.String(buf.String())
}

func answersToHash(v interface{}) int {
	var buf bytes.Buffer
	a := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", a["answer"].(string)))
	if a["region"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", a["region"].(string)))
	}
	var metas []int
	switch t := a["meta"].(type) {
	default:
		panic(fmt.Sprintf("unexpected type %T", t))
	case *schema.Set:
		for _, meta := range t.List() {
			metas = append(metas, metaToHash(meta))
		}
	case []map[string]interface{}:
		for _, meta := range t {
			metas = append(metas, metaToHash(meta))
		}
	}
	sort.Ints(metas)
	for _, metahash := range metas {
		buf.WriteString(fmt.Sprintf("%d-", metahash))
	}
	hash := hashcode.String(buf.String())
	log.Printf("Generated answersToHash %d from %+v\n", hash, a)
	return hash
}

func metaToHash(v interface{}) int {
	var buf bytes.Buffer
	s := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", s["field"].(string)))
	if v, ok := s["feed"]; ok && v.(string) != "" {
		buf.WriteString(fmt.Sprintf("feed%s-", v.(string)))
	}
	if v, ok := s["value"]; ok && v.(string) != "" {
		buf.WriteString(fmt.Sprintf("value%s-", v.(string)))
	}

	hash := hashcode.String(buf.String())
	log.Printf("Generated metaToHash %d from %+v\n", hash, s)
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
	if len(r.Filters) > 0 {
		filters := make([]map[string]interface{}, len(r.Filters))
		for i, f := range r.Filters {
			m := make(map[string]interface{})
			m["filter"] = f.Filter
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
		for regionName, region := range r.Regions {
			newRegion := make(map[string]interface{})
			newRegion["name"] = regionName
			if len(region.Meta.GeoRegion) > 0 {
				newRegion["georegion"] = region.Meta.GeoRegion[0]
			}
			if len(region.Meta.Country) > 0 {
				newRegion["country"] = region.Meta.Country[0]
			}
			if len(region.Meta.USState) > 0 {
				newRegion["us_state"] = region.Meta.USState[0]
			}
			if region.Meta.Up {
				newRegion["up"] = region.Meta.Up
			} else {
				newRegion["up"] = false
			}
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
			switch t := v.(type) {
			case map[string]interface{}:
				meta["feed"] = t["feed"].(string)
			case string:
				meta["value"] = t
			case []interface{}:
				var valArray []string
				for _, pref := range t {
					valArray = append(valArray, pref.(string))
				}
				sort.Strings(valArray)
				stringVal := strings.Join(valArray, ",")
				meta["value"] = stringVal
			case bool:
				intVal := btoi(t)
				meta["value"] = strconv.Itoa(intVal)
			case float64:
				intVal := int(t)
				meta["value"] = strconv.Itoa(intVal)
			}
			metas.Add(meta)
		}
		m["meta"] = metas
	}
	return m
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func resourceDataToRecord(r *nsone.Record, d *schema.ResourceData) error {
	r.Id = d.Id()
	if answers := d.Get("answers").(*schema.Set); answers.Len() > 0 {
		al := make([]nsone.Answer, answers.Len())
		for i, answerRaw := range answers.List() {
			answer := answerRaw.(map[string]interface{})
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
				for _, metaRaw := range metas.List() {
					meta := metaRaw.(map[string]interface{})
					key := meta["field"].(string)
					if value, ok := meta["feed"]; ok && value.(string) != "" {
						a.Meta[key] = nsone.NewMetaFeed(value.(string))
					}
					if value, ok := meta["value"]; ok && value.(string) != "" {
						metaArray := strings.Split(value.(string), ",")
						if len(metaArray) > 1 {
							sort.Strings(metaArray)
							a.Meta[key] = metaArray
						} else {
							a.Meta[key] = value.(string)
						}
					}
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
	useClientSubnetVal := d.Get("use_client_subnet").(bool)
	if v := strconv.FormatBool(useClientSubnetVal); v != "" {
		r.UseClientSubnet = useClientSubnetVal
	}

	if rawFilters := d.Get("filters").([]interface{}); len(rawFilters) > 0 {
		f := make([]nsone.Filter, len(rawFilters))
		for i, filterRaw := range rawFilters {
			fi := filterRaw.(map[string]interface{})
			config := make(map[string]interface{})
			filter := nsone.Filter{
				Filter: fi["filter"].(string),
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
			f[i] = filter
		}
		r.Filters = f
	}
	if regions := d.Get("regions").(*schema.Set); regions.Len() > 0 {
		rm := make(map[string]nsone.Region)
		for _, regionRaw := range regions.List() {
			region := regionRaw.(map[string]interface{})
			nsoneR := nsone.Region{
				Meta: nsone.RegionMeta{},
			}
			if g := region["georegion"].(string); g != "" {
				nsoneR.Meta.GeoRegion = []string{g}
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

// RecordCreate creates DNS record in ns1
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

// RecordRead reads the DNS record from ns1
func RecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	r, err := client.GetRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err != nil {
		return err
	}
	recordToResourceData(d, r)
	return nil
}

// RecordDelete deltes the DNS record from ns1
func RecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteRecord(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	d.SetId("")
	return err
}

// RecordUpdate updates the given dns record in ns1
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
