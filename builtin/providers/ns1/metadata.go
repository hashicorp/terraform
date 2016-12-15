package ns1

import (
	"fmt"
	"log"
	"math"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
)

func metadataSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"up": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"connections": &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validatePositiveInt,
				},
				"requests": &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validatePositiveInt,
				},
				"loadavg": &schema.Schema{
					Type:         schema.TypeFloat,
					Optional:     true,
					ValidateFunc: validatePositiveFloat,
				},
				"pulsar": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"latitude": &schema.Schema{
					Type:         schema.TypeFloat,
					Optional:     true,
					ValidateFunc: validateCoordinate,
				},
				"longitude": &schema.Schema{
					Type:         schema.TypeFloat,
					Optional:     true,
					ValidateFunc: validateCoordinate,
				},
				"georegion": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validateGeoregion,
					},
				},
				"country": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validateCountry,
					},
				},
				"us_state": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validateUSState,
					},
				},
				"ca_province": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validateCAProvince,
					},
				},
				"note": &schema.Schema{
					Type:         schema.TypeString,
					Optional:     true,
					ValidateFunc: validateNote,
				},
				"ip_prefixes": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Schema{
						Type:         schema.TypeString,
						ValidateFunc: validateIPPrefix,
					},
				},
				"asn": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeInt},
				},
				"priority": &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validatePositiveInt,
				},
				"weight": &schema.Schema{
					Type:         schema.TypeFloat,
					Optional:     true,
					ValidateFunc: validatePositiveFloat,
				},
				"low_watermark": &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validatePositiveInt,
				},
				"high_watermark": &schema.Schema{
					Type:         schema.TypeInt,
					Optional:     true,
					ValidateFunc: validatePositiveInt,
				},

				// Dynamic(feed) metadata
				"up_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"connections_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"requests_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"loadavg_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"pulsar_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"latitude_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"longitude_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"georegion_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"country_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"us_state_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"ca_province_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"note_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"ip_prefixes_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"asn_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"priority_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"weight_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"low_watermark_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"high_watermark_feed": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func expandNS1Metadata(d *schema.ResourceData, prefix string) *data.Meta {
	meta := &data.Meta{}

	len, ok := d.GetOk(prefix + ".#")
	if len == "0" || !ok {
		return nil
	}

	expandNS1StaticMetadata(d, prefix, meta)
	expandNS1DynamicMetadata(d, prefix, meta)

	return meta
}

func expandNS1StaticMetadata(d *schema.ResourceData, prefix string, meta *data.Meta) {
	if v, ok := d.GetOk(prefix + ".0.up"); ok {
		meta.Up = v.(bool)
	}

	if v, ok := d.GetOk(prefix + ".0.connections"); ok {
		meta.Connections = v.(int)
	}

	if v, ok := d.GetOk(prefix + ".0.requests"); ok {
		meta.Requests = v.(int)
	}

	if v, ok := d.GetOk(prefix + ".0.loadavg"); ok {
		if !math.IsNaN(v.(float64)) {
			meta.LoadAvg = v.(float64)
		}
	}

	if v, ok := d.GetOk(prefix + ".0.pulsar"); ok {
		meta.Pulsar = v.(string)
	}

	if v, ok := d.GetOk(prefix + ".0.latitude"); ok {
		if !math.IsNaN(v.(float64)) {
			meta.Latitude = v.(float64)
		}
	}

	if v, ok := d.GetOk(prefix + ".0.longitude"); ok {
		if !math.IsNaN(v.(float64)) {
			meta.Longitude = v.(float64)
		}
	}

	if len, ok := d.GetOk(prefix + ".0.georegion.#"); ok {
		regions := make([]string, len.(int))
		for i := 0; i < len.(int); i++ {
			key := fmt.Sprintf("%s.0.georegion.%d", prefix, i)
			regions[i] = d.Get(key).(string)
		}
		meta.Georegion = regions
	}

	if len, ok := d.GetOk(prefix + ".0.country.#"); ok {
		countries := make([]string, len.(int))
		for i := 0; i < len.(int); i++ {
			key := fmt.Sprintf("%s.0.country.%d", prefix, i)
			countries[i] = d.Get(key).(string)
		}
		meta.Country = countries
	}

	if len, ok := d.GetOk(prefix + ".0.us_state.#"); ok {
		states := make([]string, len.(int))
		for i := 0; i < len.(int); i++ {
			key := fmt.Sprintf("%s.0.us_state.%d", prefix, i)
			states[i] = d.Get(key).(string)
		}
		meta.USState = states
	}

	if len, ok := d.GetOk(prefix + ".0.ca_province.#"); ok {
		provinces := make([]string, len.(int))
		for i := 0; i < len.(int); i++ {
			key := fmt.Sprintf("%s.0.ca_province.%d", prefix, i)
			provinces[i] = d.Get(key).(string)
		}
		meta.CAProvince = provinces
	}

	if v, ok := d.GetOk(prefix + ".0.note"); ok {
		meta.Note = v.(string)
	}

	if len, ok := d.GetOk(prefix + ".0.ip_prefixes.#"); ok {
		ip_prefixes := make([]string, len.(int))
		for i := 0; i < len.(int); i++ {
			key := fmt.Sprintf("%s.0.ip_prefixes.%d", prefix, i)
			ip_prefixes[i] = d.Get(key).(string)
		}
		meta.IPPrefixes = ip_prefixes
	}

	if len, ok := d.GetOk(prefix + ".0.asn.#"); ok {
		asns := make([]int, len.(int))
		for i := 0; i < len.(int); i++ {
			key := fmt.Sprintf("%s.0.asn.%d", prefix, i)
			asns[i] = d.Get(key).(int)
		}
		meta.ASN = asns
	}

	if v, ok := d.GetOk(prefix + ".0.priority"); ok {
		meta.Priority = v.(int)
	}

	if v, ok := d.GetOk(prefix + ".0.weight"); ok {
		if !math.IsNaN(v.(float64)) {
			meta.Weight = v.(float64)
		}
	}

	if v, ok := d.GetOk(prefix + ".0.low_watermark"); ok {
		meta.LowWatermark = v.(int)
	}

	if v, ok := d.GetOk(prefix + ".0.high_watermark"); ok {
		meta.HighWatermark = v.(int)
	}
}

func expandNS1DynamicMetadata(d *schema.ResourceData, prefix string, meta *data.Meta) {
	if v, ok := d.GetOk(prefix + ".0.up_feed"); ok {
		meta.Up = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.connections_feed"); ok {
		meta.Connections = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.requests_feed"); ok {
		meta.Requests = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.loadavg_feed"); ok {
		meta.LoadAvg = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.pulsar_feed"); ok {
		meta.Pulsar = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.latitude_feed"); ok {
		meta.Latitude = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.longitude_feed"); ok {
		meta.Longitude = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.georegion_feed"); ok {
		meta.Georegion = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.country_feed"); ok {
		meta.Country = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.us_state_feed"); ok {
		meta.USState = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.ca_province_feed"); ok {
		meta.CAProvince = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.note_feed"); ok {
		meta.Note = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.ip_prefixes_feed"); ok {
		meta.IPPrefixes = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.asn_feed"); ok {
		meta.ASN = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.priority_feed"); ok {
		meta.Priority = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.weight_feed"); ok {
		meta.Weight = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.low_watermark_feed"); ok {
		meta.LowWatermark = data.FeedPtr{v.(string)}
	}

	if v, ok := d.GetOk(prefix + ".0.high_watermark_feed"); ok {
		meta.HighWatermark = data.FeedPtr{v.(string)}
	}
}

func flattenNS1Metadata(meta *data.Meta) []map[string]interface{} {
	if meta == nil {
		return []map[string]interface{}{}
	}
	m := setMap(make(map[string]interface{}))

	m.Set("up", parseFeedBool(meta.Up))
	m.Set("connections", parseFeedInt(meta.Connections))
	m.Set("requests", parseFeedInt(meta.Requests))
	m.Set("loadavg", parseFeedFloat(meta.LoadAvg))
	m.Set("pulsar", parseFeedString(meta.Pulsar))
	m.Set("latitude", parseFeedFloat(meta.Latitude))
	m.Set("longitude", parseFeedFloat(meta.Longitude))

	return m.MapList()
}

func getFloat(v interface{}, field string, precision int) float64 {
	f, err := strconv.ParseFloat(v.(string), 64)
	if err != nil {
		log.Printf("Expected float for %s, instead got: %#v", field, v)
		return 0
	}

	return f
}

func getBool(v interface{}, field string) bool {
	b, err := strconv.ParseBool(v.(string))
	if err != nil {
		log.Printf("Expected bool for %s, instead got: %#v \n", field, v)
	}
	return b
}

func getInt(v interface{}, field string) int {
	i, err := strconv.Atoi(v.(string))
	if err != nil {
		log.Printf("Expected int for %s, instead got: %#v \n", field, v)
		return 0
	}
	return i
}

func getString(v interface{}, field string) string {
	s, ok := v.(string)
	if !ok {
		log.Printf("Expected string for %s, instead got: %#v \n", field, v)
		return ""
	}
	return s
}

func parseFeedBool(v interface{}) *string {
	switch v.(type) {
	case nil:
		return nil
	case bool:
		s := strconv.FormatBool(v.(bool))
		return &s
	case map[string]interface{}:
		s := v.(map[string]interface{})["feed"].(string)
		return &s
	case data.FeedPtr:
		s := v.(data.FeedPtr).FeedID
		return &s
	default:
		fmt.Printf("[DEBUG] parseFeedBool unknown type for v: %#v \n", v)
	}
	return nil
}

func parseFeedInt(v interface{}) *string {
	switch v.(type) {
	case nil:
		return nil
	case int:
		s := strconv.Itoa(v.(int))
		return &s
	case float64:
		s := strconv.Itoa(int(v.(float64)))
		return &s
	case map[string]interface{}:
		s := v.(map[string]interface{})["feed"].(string)
		return &s
	default:
		fmt.Printf("[DEBUG] parseFeedInt unknown type for v: %#v \n", v)
	}
	return nil
}

func parseFeedFloat(v interface{}) *string {
	switch v.(type) {
	case nil:
		return nil
	case int:
		s := strconv.FormatFloat(float64(v.(int)), 'f', 2, 64)
		return &s
	case float64:
		s := strconv.FormatFloat(v.(float64), 'f', 2, 64)
		fmt.Printf("[DEBUG] parseFeedFloat v: %#v \n", s)
		return &s
	case map[string]interface{}:
		s := v.(map[string]interface{})["feed"].(string)
		return &s
	default:
		fmt.Printf("[DEBUG] parseFeedFloat unknown type for v: %#v \n", v)
	}
	return nil
}

func parseFeedString(v interface{}) *string {
	switch v.(type) {
	case nil:
		return nil
	case string:
		s := v.(string)
		return &s
	case map[string]interface{}:
		s := v.(map[string]interface{})["feed"].(string)
		return &s
	default:
		fmt.Printf("[DEBUG] parseFeedString unknown type for v: %#v \n", v)
	}
	return nil
}
