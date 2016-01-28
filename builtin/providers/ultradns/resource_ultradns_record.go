package ultradns

import (
	"encoding/json"
	"fmt"
	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strconv"
	"strings"
)

func schemaSBPoolProfile() *schema.Schema {
	return &schema.Schema{
		Type:          schema.TypeMap,
		Optional:      true,
		ConflictsWith: []string{"dirpool_profile", "rdpool_profile", "tcpool_profile", "string_profile", "map_profile"},
	}
}
func schemaDirPoolRDataInfo() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"allNonConfigured": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  false,
				},
				"geoInfo": &schema.Schema{
					Type:     schema.TypeSet,
					Optional: true,
					Elem:     schemaDirPoolGeoInfo(),
				},
				"ipInfo": &schema.Schema{
					Type:     schema.TypeSet,
					Optional: true,
					Elem:     schemaDirPoolIPInfo(),
				},
			},
		},
	}
}

func schemaDirPoolGeoInfo() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: false,
				},
				"isAccountLevel": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  false,
				},
				"codes": &schema.Schema{
					Type:     schema.TypeSet,
					Optional: true,
					Elem:     schema.TypeString,
				},
			},
		},
	}
}
func schemaDirPoolIPInfo() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: false,
				},
				"isAccountLevel": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  false,
				},
				"ips": &schema.Schema{
					Type:     schema.TypeSet,
					Optional: true,
					Elem:     schemaIPAddrDTO(),
				},
			},
		},
	}
}
func schemaIPAddrDTO() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"start": &schema.Schema{
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"cidr", "address"},
				},
				"end": &schema.Schema{
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"cidr", "address"},
				},
				"cidr": &schema.Schema{
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"start", "end", "address"},
				},
				"address": &schema.Schema{
					Type:          schema.TypeString,
					Optional:      true,
					ConflictsWith: []string{"start", "end", "cidr"},
				},
			},
		},
	}
}

func schemaDirPoolProfile() *schema.Schema {
	return &schema.Schema{
		Type:          schema.TypeMap,
		Optional:      true,
		ConflictsWith: []string{"rdpool_profile", "sbpool_profile", "tcpool_profile", "string_profile", "map_profile"},
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"description": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "RD Pool Profile created by Terraform",
				},
				"conflictResolve": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "GEO",
				},
				"rdataInfo": &schema.Schema{
					Type:     schema.TypeSet,
					Optional: true,
					Elem:     schemaDirPoolRDataInfo(),
				},
				"noResponse": schemaDirPoolRDataInfo(),
			},
		},
	}
}
func schemaTCPoolProfile() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,

		ConflictsWith: []string{"dirpool_profile", "sbpool_profile", "rdpool_profile", "string_profile", "map_profile"},
	}
}
func schemaRDPoolProfile() *schema.Schema {
	return &schema.Schema{
		Type:          schema.TypeMap,
		Optional:      true,
		ConflictsWith: []string{"dirpool_profile", "sbpool_profile", "tcpool_profile", "string_profile", "map_profile"},
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"@context": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "http://schemas.ultradns.com/RDPool.jsonschema",
				},
				"order": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "ROUND_ROBIN",
				},
				"description": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
					Default:  "RD Pool Profile created by Terraform",
				},
			},
		},
	}
}
func resourceUltraDNSRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceUltraDNSRecordCreate,
		Read:   resourceUltraDNSRecordRead,
		Update: resourceUltraDNSRecordUpdate,
		Delete: resourceUltraDNSRecordDelete,

		Schema: map[string]*schema.Schema{
			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"rdata": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "3600",
			},
			"rdpool_profile":  schemaRDPoolProfile(),
			"dirpool_profile": schemaDirPoolProfile(),
			"sbpool_profile":  schemaSBPoolProfile(),
			"tcpool_profile":  schemaTCPoolProfile(),
			"map_profile": &schema.Schema{
				Type:          schema.TypeMap,
				ConflictsWith: []string{"dirpool_profile", "sbpool_profile", "tcpool_profile", "rdpool_profile", "string_profile"},
				Optional:      true,
			},
			"string_profile": &schema.Schema{
				Type:          schema.TypeString,
				ConflictsWith: []string{"dirpool_profile", "sbpool_profile", "tcpool_profile", "rdpool_profile", "map_profile"},
				Optional:      true,
			},
		},
	}
}

func resourceUltraDNSRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)
	newRecord := &udnssdk.RRSet{
		OwnerName: d.Get("name").(string),
		RRType:    d.Get("type").(string),
	}
	rdata := d.Get("rdata").([]interface{})
	rdatas := make([]string, len(rdata))
	for i, j := range rdata {
		rdatas[i] = j.(string)
	}
	newRecord.RData = rdatas
	ttl := d.Get("ttl").(string)
	newRecord.TTL, _ = strconv.Atoi(ttl)
	newProfileStr := d.Get("string_profile").(string)
	if newProfileStr != "" {
		newProfile := &udnssdk.StringProfile{Profile: newProfileStr}
		newRecord.Profile = newProfile
	}
	profilelist := map[string]string{"rdpool_profile": "http://schemas.ultradns.com/RDPool.jsonschema", "sbpool_profile": "http://schemas.ultradns.com/SBPool.jsonschema", "tcpool_profile": "http://schemas.ultradns.com/TCPool.jsonschema", "dirpool_profile": "http://schemas.ultradns.com/DirPool.jsonschema"}
	for key, value := range profilelist {
		firstValidation := d.Get(key)
		if firstValidation == nil {
			continue
		}
		poolProfile := firstValidation.(map[string]interface{})
		log.Printf("[DEBUG] - Create - %s = %+v\n", key, poolProfile)
		if len(poolProfile) != 0 {
			poolProfile["@context"] = value
			x, e := json.Marshal(poolProfile)
			if e != nil {
				return fmt.Errorf("[ERROR] poolProfile Marshalling error: %+v", e)
			}
			newProfile := &udnssdk.StringProfile{Profile: string(x)}
			newRecord.Profile = newProfile
			break
		}
	}
	log.Printf("[DEBUG] UltraDNS RRSet create configuration: %#v", newRecord)

	_, err := client.RRSets.CreateRRSet(d.Get("zone").(string), *newRecord)
	recId := fmt.Sprintf("%s.%s", d.Get("name").(string), d.Get("zone").(string))
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to create UltraDNS RRSet: %s", err)
	}

	d.SetId(recId)
	log.Printf("[INFO] record ID: %s", d.Id())

	return resourceUltraDNSRecordRead(d, meta)
}

func resourceUltraDNSRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	rrsets, err := client.RRSets.ListAllRRSets(d.Get("zone").(string), d.Get("name").(string), d.Get("type").(string))
	if err != nil {
		uderr, ok := err.(*udnssdk.ErrorResponseList)
		if ok {
			for _, r := range uderr.Responses {
				// 70002 means Records Not Found
				if r.ErrorCode == 70002 {
					d.SetId("")
					return nil
				} else {
					return fmt.Errorf("[ERROR] Couldn't find UltraDNS RRSet: %s", err)
				}
			}
		} else {
			return fmt.Errorf("[ERROR] Couldn't find UltraDNS RRSet: %s", err)
		}
	}
	rec := rrsets[0]
	err = d.Set("rdata", rec.RData)
	if err != nil {
		return fmt.Errorf("[DEBUG] Error setting records: %#v", err)
	}
	d.Set("ttl", rec.TTL)

	if rec.OwnerName == "" {
		d.Set("hostname", d.Get("zone").(string))
	} else {
		if strings.HasSuffix(rec.OwnerName, ".") {
			d.Set("hostname", rec.OwnerName)
		} else {
			d.Set("hostname", fmt.Sprintf("%s.%s", rec.OwnerName, d.Get("zone").(string)))
		}
	}
	if rec.Profile != nil {
		t := rec.Profile.GetType()
		d.Set("string_profile", rec.Profile.Profile)
		var dp map[string]interface{}
		err = json.Unmarshal([]byte(rec.Profile.Profile), &dp)
		if err != nil {
			return err
		}
		tmpvar := strings.Split(t, "/")
		switch tmpvar[len(tmpvar)-1] {
		case "DirPool.jsonschema":
			d.Set("dirpool_profile", dp)
		case "RDPool.jsonschema":
			d.Set("rdpool_profile", dp)
		case "TCPool.jsonschema":
			d.Set("tcpool_profile", dp)
		case "SBPool.jsonschema":
			d.Set("sbpool_profile", dp)
		default:
			return fmt.Errorf("[DEBUG] Unknown Type %s\n", t)
		}
	}
	return nil
}

func resourceUltraDNSRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	updateRecord := &udnssdk.RRSet{}

	if attr, ok := d.GetOk("name"); ok {
		updateRecord.OwnerName = attr.(string)
	}

	if attr, ok := d.GetOk("type"); ok {
		updateRecord.RRType = attr.(string)
	}

	if attr, ok := d.GetOk("rdata"); ok {
		rdata := attr.([]interface{})
		rdatas := make([]string, len(rdata))
		for i, j := range rdata {
			rdatas[i] = j.(string)
		}
		updateRecord.RData = rdatas
	}

	if attr, ok := d.GetOk("ttl"); ok {
		updateRecord.TTL, _ = strconv.Atoi(attr.(string))
	}
	newProfileStr := d.Get("string_profile").(string)
	if newProfileStr != "" {
		newProfile := &udnssdk.StringProfile{Profile: newProfileStr}
		updateRecord.Profile = newProfile
	}
	profilelist := map[string]string{"rdpool_profile": "http://schemas.ultradns.com/RDPool.jsonschema", "sbpool_profile": "http://schemas.ultradns.com/SBPool.jsonschema", "tcpool_profile": "http://schemas.ultradns.com/TCPool.jsonschema", "dirpool_profile": "http://schemas.ultradns.com/DirPool.jsonschema"}
	for key, value := range profilelist {
		firstValidation := d.Get(key)
		if firstValidation == nil {
			continue
		}
		poolProfile := firstValidation.(map[string]interface{})
		if len(poolProfile) != 0 {
			poolProfile["@context"] = value
			x, e := json.Marshal(poolProfile)
			if e != nil {
				return fmt.Errorf("[ERROR] poolProfile Marshal error: %+v", e)
			}
			newProfile := &udnssdk.StringProfile{Profile: string(x)}
			updateRecord.Profile = newProfile
			break
		}
	}
	log.Printf("[DEBUG] UltraDNS RRSet update configuration: %#v", updateRecord)

	_, err := client.RRSets.UpdateRRSet(d.Get("zone").(string), *updateRecord)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to update UltraDNS RRSet: %s", err)
	}

	return resourceUltraDNSRecordRead(d, meta)
}

func resourceUltraDNSRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	log.Printf("[INFO] Deleting UltraDNS RRSet: %s, %s", d.Get("zone").(string), d.Id())
	deleteRecord := &udnssdk.RRSet{}

	if attr, ok := d.GetOk("name"); ok {
		deleteRecord.OwnerName = attr.(string)
	}

	if attr, ok := d.GetOk("type"); ok {
		deleteRecord.RRType = attr.(string)
	}

	_, err := client.RRSets.DeleteRRSet(d.Get("zone").(string), *deleteRecord)

	if err != nil {
		return fmt.Errorf("[ERROR] Error deleting UltraDNS RRSet: %s", err)
	}

	return nil
}
