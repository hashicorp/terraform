package ns1

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"

	nsone "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
	"gopkg.in/ns1/ns1-go.v2/rest/model/dns"
	"gopkg.in/ns1/ns1-go.v2/rest/model/filter"
)

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
			// Optional
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
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
			// Computed
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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

func recordToResourceData(d *schema.ResourceData, r *dns.Record) error {
	d.SetId(r.ID)
	d.Set("domain", r.Domain)
	d.Set("zone", r.Zone)
	d.Set("type", r.Type)
	d.Set("ttl", r.TTL)
	if r.Link != "" {
		d.Set("link", r.Link)
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
			F: answersToHash,
		}
		log.Printf("Got back from ns1 answers: %+v", r.Answers)
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

			meta := region.Meta
			// TODO: support as FeedPtr
			switch meta.Georegion.(type) {
			case nil:
				break
			case []interface{}:
				georegion := region.Meta.Georegion.([]interface{})
				if len(georegion) > 0 {
					newRegion["georegion"] = georegion[0]
				}
			default:
				panic(fmt.Sprintf("meta.Georegion in unexpected type: %#v", meta.Georegion))
			}
			// TODO: support as FeedPtr
			switch meta.Country.(type) {
			case nil:
				break
			case []interface{}:
				country := region.Meta.Country.([]interface{})
				if len(country) > 0 {
					newRegion["country"] = country[0]
				}
			default:
				panic(fmt.Sprintf("meta.Country in unexpected type: %#v", meta.Country))
			}
			// TODO: support as FeedPtr
			switch meta.USState.(type) {
			case nil:
				break
			case []interface{}:
				usState := region.Meta.USState.([]interface{})
				if len(usState) > 0 && usState != nil {
					newRegion["us_state"] = usState[0]
				}
			default:
				panic(fmt.Sprintf("meta.USState in unexpected type: %#v", meta.USState))
			}
			// TODO: support as FeedPtr
			switch meta.Up.(type) {
			case nil:
				break
			case bool:
				newRegion["up"] = region.Meta.Up.(bool)
			default:
				panic(fmt.Sprintf("meta.Up in unexpected type: %#v", meta.Up))
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

func answerToMap(a dns.Answer) map[string]interface{} {
	m := make(map[string]interface{})
	m["meta"] = make([]map[string]interface{}, 0)
	m["answer"] = strings.Join(a.Rdata, " ")
	if a.RegionName != "" {
		m["region"] = a.RegionName
	}
	if a.Meta != nil {
		metas := &schema.Set{
			F: metaToHash,
		}
		meta := a.Meta
		// TODO: set things up to use FeedPtr
		switch meta.Up.(type) {
		case nil:
			break
		case bool:
			up := meta.Up.(bool)
			if up {
				metas.Add(map[string]interface{}{
					"field": "up",
					"value": strconv.Itoa(btoi(up)),
				})
			}
		default:
			panic(fmt.Sprintf("meta.Up in unexpected type: %#v", meta.Up))
		}
		switch meta.Connections.(type) {
		case nil:
			break
		case int:
			connections := meta.Connections.(int)
			if connections != 0 {
				metas.Add(map[string]interface{}{
					"field": "connections",
					"value": strconv.Itoa(connections),
				})
			}
		default:
			panic(fmt.Sprintf("meta.Connections in unexpected type: %#v", meta.Connections))
		}
		switch meta.Requests.(type) {
		case nil:
			break
		case int:
			requests := meta.Requests.(int)
			if requests != 0 {
				metas.Add(map[string]interface{}{
					"field": "requests",
					"value": strconv.Itoa(requests),
				})
			}
		default:
			panic(fmt.Sprintf("meta.Requests in unexpected type: %#v", meta.Requests))
		}
		switch meta.LoadAvg.(type) {
		case nil:
			break
		case float64:
			loadavg := meta.LoadAvg.(float64)
			if loadavg != 0 {
				metas.Add(map[string]interface{}{
					"field": "loadavg",
					"value": strconv.Itoa(int(loadavg)),
				})
			}
		default:
			panic(fmt.Sprintf("meta.LoadAvg in unexpected type: %#v", meta.LoadAvg))
		}
		switch meta.Pulsar.(type) {
		case nil:
			break
		case string:
			pulsar := meta.Pulsar.(string)
			if pulsar != "" {
				metas.Add(map[string]interface{}{
					"field": "pulsar",
					"value": pulsar,
				})
			}
		default:
			panic(fmt.Sprintf("meta.Pulsar in unexpected type: %#v", meta.Pulsar))
		}
		switch meta.Latitude.(type) {
		case nil:
			break
		case float64:
			latitude := meta.Latitude.(float64)
			if latitude != 0 {
				metas.Add(map[string]interface{}{
					"field": "latitude",
					"value": strconv.Itoa(int(latitude)),
				})
			}
		default:
			panic(fmt.Sprintf("meta.Latitude in unexpected type: %#v", meta.Latitude))
		}
		switch meta.Longitude.(type) {
		case nil:
			break
		case float64:
			longitude := meta.Longitude.(float64)
			if longitude != 0 {
				metas.Add(map[string]interface{}{
					"field": "longitude",
					"value": strconv.Itoa(int(longitude)),
				})
			}
		default:
			panic(fmt.Sprintf("meta.Longitude in unexpected type: %#v", meta.Longitude))
		}
		switch meta.Georegion.(type) {
		case nil:
			break
		case []string:
			georegion := meta.Georegion.([]string)
			if len(georegion) != 0 {
				sort.Strings(georegion)
				metas.Add(map[string]interface{}{
					"field": "georegion",
					"value": strings.Join(georegion, ","),
				})
			}
		default:
			panic(fmt.Sprintf("meta.Georegion in unexpected type: %#v", meta.Georegion))
		}
		switch meta.Country.(type) {
		case nil:
			break
		case []string:
			country := meta.Country.([]string)
			if len(country) != 0 {
				sort.Strings(country)
				metas.Add(map[string]interface{}{
					"field": "country",
					"value": strings.Join(country, ","),
				})
			}
		default:
			panic(fmt.Sprintf("meta.Country in unexpected type: %#v", meta.Country))
		}
		switch meta.USState.(type) {
		case nil:
			break
		case []string:
			usState := meta.USState.([]string)
			if len(usState) != 0 {
				sort.Strings(usState)
				metas.Add(map[string]interface{}{
					"field": "us_state",
					"value": strings.Join(usState, ","),
				})
			}
		default:
			panic(fmt.Sprintf("meta.USState in unexpected type: %#v", meta.USState))
		}
		switch meta.CAProvince.(type) {
		case nil:
			break
		case []string:
			caProvince := meta.CAProvince.([]string)
			if len(caProvince) != 0 {
				sort.Strings(caProvince)
				metas.Add(map[string]interface{}{
					"field": "ca_province",
					"value": strings.Join(caProvince, ","),
				})
			}
		default:
			panic(fmt.Sprintf("meta.CAProvince in unexpected type: %#v", meta.CAProvince))
		}
		switch meta.Note.(type) {
		case nil:
			break
		case string:
			note := meta.Note.(string)
			if note != "" {
				metas.Add(map[string]interface{}{
					"field": "note",
					"value": note,
				})
			}
		default:
			panic(fmt.Sprintf("meta.Note in unexpected type: %#v", meta.Note))
		}
		switch meta.IPPrefixes.(type) {
		case nil:
			break
		case []string:
			ipPrefixes := meta.IPPrefixes.([]string)
			if len(ipPrefixes) != 0 {
				sort.Strings(ipPrefixes)
				metas.Add(map[string]interface{}{
					"field": "ip_prefixes",
					"value": strings.Join(ipPrefixes, ","),
				})
			}
		default:
			panic(fmt.Sprintf("meta.IPPrefixes in unexpected type: %#v", meta.IPPrefixes))
		}
		switch meta.ASN.(type) {
		case nil:
			break
		case []string:
			asn := meta.ASN.([]string)
			if len(asn) != 0 {
				sort.Strings(asn)
				metas.Add(map[string]interface{}{
					"field": "asn",
					"value": strings.Join(asn, ","),
				})
			}
		default:
			panic(fmt.Sprintf("meta.ASN in unexpected type: %#v", meta.ASN))
		}
		switch meta.Priority.(type) {
		case nil:
			break
		case int:
			priority := meta.Priority.(int)
			if priority != 0 {
				metas.Add(map[string]interface{}{
					"field": "priority",
					"value": strconv.Itoa(priority),
				})
			}
		default:
			panic(fmt.Sprintf("meta.Priority in unexpected type: %#v", meta.Priority))
		}
		switch meta.Weight.(type) {
		case nil:
			break
		case float64:
			weight := meta.Weight.(float64)
			if weight != 0 {
				metas.Add(map[string]interface{}{
					"field": "weight",
					"value": strconv.Itoa(int(weight)),
				})
			}
		default:
			panic(fmt.Sprintf("meta.Weight in unexpected type: %#v", meta.Weight))
		}
		switch meta.LowWatermark.(type) {
		case nil:
			break
		case int:
			lowWatermark := meta.LowWatermark.(int)
			if lowWatermark != 0 {
				metas.Add(map[string]interface{}{
					"field": "low_watermark",
					"value": strconv.Itoa(lowWatermark),
				})
			}
		default:
			panic(fmt.Sprintf("meta.LowWatermark in unexpected type: %#v", meta.LowWatermark))
		}
		switch meta.HighWatermark.(type) {
		case nil:
			break
		case int:
			highWatermark := meta.HighWatermark.(int)
			if highWatermark != 0 {
				metas.Add(map[string]interface{}{
					"field": "high_watermark",
					"value": strconv.Itoa(highWatermark),
				})
			}
		default:
			panic(fmt.Sprintf("meta.HighWatermark in unexpected type: %#v", meta.HighWatermark))
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
			if metas := answer["meta"].(*schema.Set); metas.Len() > 0 {
				for _, metaRaw := range metas.List() {
					meta := metaRaw.(map[string]interface{})
					key := meta["field"].(string)
					if value, ok := meta["feed"]; ok && value.(string) != "" {
						switch key {
						case "up": // bool
							a.Meta.Up = data.NewFeed(value.(string), data.Config{})
						case "connections": // int
							a.Meta.Connections = data.NewFeed(value.(string), data.Config{})
						case "requests": //int
							a.Meta.Requests = data.NewFeed(value.(string), data.Config{})
						case "loadavg": // float64
							a.Meta.LoadAvg = data.NewFeed(value.(string), data.Config{})
						case "pulsar": //string
							a.Meta.Pulsar = data.NewFeed(value.(string), data.Config{})
						case "latitude": // float64
							a.Meta.Latitude = data.NewFeed(value.(string), data.Config{})
						case "longitude": // float64
							a.Meta.Longitude = data.NewFeed(value.(string), data.Config{})
						case "georegion": // []string
							a.Meta.Georegion = data.NewFeed(value.(string), data.Config{})
						case "country": // []string
							a.Meta.Country = data.NewFeed(value.(string), data.Config{})
						case "us_state": // []string
							a.Meta.USState = data.NewFeed(value.(string), data.Config{})
						case "ca_province": // []string
							a.Meta.CAProvince = data.NewFeed(value.(string), data.Config{})
						case "note": // string
							a.Meta.Note = data.NewFeed(value.(string), data.Config{})
						case "ip_prefixes": // []string
							a.Meta.IPPrefixes = data.NewFeed(value.(string), data.Config{})
						case "asn": // []string
							a.Meta.ASN = data.NewFeed(value.(string), data.Config{})
						case "priority": // int
							a.Meta.Priority = data.NewFeed(value.(string), data.Config{})
						case "weight": // float64
							a.Meta.Weight = data.NewFeed(value.(string), data.Config{})
						case "low_watermark": // int
							a.Meta.LowWatermark = data.NewFeed(value.(string), data.Config{})
						case "high_watermark": // int
							a.Meta.HighWatermark = data.NewFeed(value.(string), data.Config{})
						}
					}
					if value, ok := meta["value"]; ok && value.(string) != "" {
						switch key {
						case "up": // bool
							a.Meta.Up = value.(string)
						case "connections": // int
							a.Meta.Connections = value.(string)
						case "requests": //int
							a.Meta.Requests = value.(string)
						case "loadavg": // float64
							a.Meta.LoadAvg = value.(string)
						case "pulsar": //string
							a.Meta.Pulsar = value.(string)
						case "latitude": // float64
							a.Meta.Latitude = value.(string)
						case "longitude": // float64
							a.Meta.Longitude = value.(string)
						case "georegion": // []string
							metaArray := strings.Split(value.(string), ",")
							sort.Strings(metaArray)
							a.Meta.Georegion = metaArray
						case "country": // []string
							metaArray := strings.Split(value.(string), ",")
							sort.Strings(metaArray)
							a.Meta.Country = metaArray
						case "us_state": // []string
							metaArray := strings.Split(value.(string), ",")
							sort.Strings(metaArray)
							a.Meta.USState = metaArray
						case "ca_province": // []string
							metaArray := strings.Split(value.(string), ",")
							sort.Strings(metaArray)
							a.Meta.CAProvince = metaArray
						case "note": // string
							a.Meta.Note = data.NewFeed(value.(string), data.Config{})
						case "ip_prefixes": // []string
							metaArray := strings.Split(value.(string), ",")
							sort.Strings(metaArray)
							a.Meta.IPPrefixes = metaArray
						case "asn": // []string

							metaArray := strings.Split(value.(string), ",")
							sort.Strings(metaArray)
							a.Meta.ASN = metaArray
						case "priority": // int
							a.Meta.Priority = value.(string)
						case "weight": // float64
							a.Meta.Weight = value.(string)
						case "low_watermark": // int
							a.Meta.LowWatermark = value.(string)
						case "high_watermark": // int
							a.Meta.HighWatermark = value.(string)
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
		r.TTL = v.(int)
	}
	if v, ok := d.GetOk("link"); ok {
		r.LinkTo(v.(string))
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
	client := meta.(*nsone.Client)
	r, _, err := client.Records.Get(d.Get("zone").(string), d.Get("domain").(string), d.Get("type").(string))
	if err != nil {
		return err
	}
	return recordToResourceData(d, r)
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
