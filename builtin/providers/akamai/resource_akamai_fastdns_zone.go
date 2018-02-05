package akamai

import (
	"bytes"
	"fmt"
	"log"
	"sync"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v1"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

var dnsWriteLock sync.Mutex

func resourceFastDNSZone() *schema.Resource {
	return &schema.Resource{
		Create: resourceFastDNSZoneCreate,
		Read:   resourceFastDNSZoneRead,
		Update: resourceFastDNSZoneCreate,
		Delete: resourceFastDNSZoneDelete,
		Exists: resourceFastDNSZoneExists,
		Importer: &schema.ResourceImporter{
			State: resourceFastDNSZoneImport,
		},
		Schema: map[string]*schema.Schema{
			"hostname": {
				Type:     schema.TypeString,
				Required: true,
			},
			"a": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"aaaa": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"afsdb": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
						"subtype": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"cname": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"dnskey": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"flags": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"protocol": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"algorithm": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"ds": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"keytag": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"algorithm": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"digest_type": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"digest": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"hinfo": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"hardware": {
							Type:     schema.TypeString,
							Required: true,
						},
						"software": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"loc": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"mx": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
						"priority": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"naptr": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"order": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"preference": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"flags": {
							Type:     schema.TypeString,
							Required: true,
						},
						"service": {
							Type:     schema.TypeString,
							Required: true,
						},
						"regexp": {
							Type:     schema.TypeString,
							Required: true,
						},
						"replacement": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"ns": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"nsec3": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"algorithm": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"flags": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"iterations": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"salt": {
							Type:     schema.TypeString,
							Required: true,
						},
						"next_hashed_owner_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type_bitmaps": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"nsec3param": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"algorithm": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"flags": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"iterations": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"salt": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"ptr": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"rp": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"mailbox": {
							Type:     schema.TypeString,
							Required: true,
						},
						"txt": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"rrsig": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"type_covered": {
							Type:     schema.TypeString,
							Required: true,
						},
						"algorithm": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"original_ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"expiration": {
							Type:     schema.TypeString,
							Required: true,
						},
						"inception": {
							Type:     schema.TypeString,
							Required: true,
						},
						"keytag": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"signer": {
							Type:     schema.TypeString,
							Required: true,
						},
						"signature": {
							Type:     schema.TypeString,
							Required: true,
						},
						"labels": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"soa": {
				Type:     schema.TypeSet,
				Optional: true,
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf(
						"%s-%s-%s-%s-%s-%s-%s",
						m["ttl"],
						m["originserver"],
						m["contact"],
						m["refresh"],
						m["retry"],
						m["expire"],
						m["minimum"],
					))
					return hashcode.String(buf.String())
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ttl": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"originserver": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"contact": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"serial": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"refresh": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"retry": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"expire": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"minimum": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"spf": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"srv": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
						"priority": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"weight": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"sshfp": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"algorithm": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"fingerprint_type": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"fingerprint": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"txt": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"ttl": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"active": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"target": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

// Create a new DNS Record
func resourceFastDNSZoneCreate(d *schema.ResourceData, meta interface{}) error {
	// only allow one record to be created at a time
	// this prevents lost data if you are using a counter/dynamic variables
	// in your config.tf which might overwrite each other
	dnsWriteLock.Lock()
	defer dnsWriteLock.Unlock()

	hostname := d.Get("hostname").(string)

	// First try to get the zone from the API
	log.Printf("[INFO] [Akamai FastDNS] Searching for zone [%s]", hostname)
	zone, e := dns.GetZone(hostname)

	if e != nil {
		// If there's no existing zone we'll create a blank one
		if dns.IsConfigDNSError(e) && e.(dns.ConfigDNSError).NotFound() == true {
			// if the zone is not found/404 we will create a new
			// blank zone for the records to be added to and continue
			log.Printf("[DEBUG] [Akamai FastDNS] [ERROR] %s", e.Error())
			log.Printf("[DEBUG] [Akamai FastDNS] Creating new zone")
			zone = dns.NewZone(hostname)
			e = nil
		} else {
			return e
		}
	}

	// Transform the record data from the terraform config to a local type
	log.Printf("[DEBUG] [Akamai FastDNS] Adding records to zone")
	unmarshalResourceData(d, zone)

	// Save the zone to the API
	log.Printf("[DEBUG] [Akamai FastDNS] Saving zone")
	e = zone.Save()
	if e != nil {
		return e
	}

	// Give terraform the ID
	d.SetId(fmt.Sprintf("%s-%s-%s", zone.Token, zone.Zone.Name, hostname))

	return nil
}

// Helper function for unmarshalResourceData() below
func assignFields(record dns.DNSRecord, d map[string]interface{}) {
	f := record.GetAllowedFields()
	for _, field := range f {
		val, ok := d[field]
		if ok {
			e := record.SetField(field, val)
			if e != nil {
				log.Printf("[WARN] [Akamai FastDNS] Couldn't add field to record: %s", e.Error())
			}
		}
	}
}

// Sometimes records exist in the API but not in tf config
// In that case we will merge our records from the config with the API records
// But those API records don't ever get saved in the tf config
// This is on purpose because the Akamai API will inject several
// Default records to a given zone and we don't want those to show up
// In diffs or in acceptance tests
func mergeConfigs(recordType string, records []interface{}, s *schema.Resource, d *schema.ResourceData) *schema.Set {
	recordsInStateFile, recordsInConfigFile := d.GetChange(recordType)
	recordsInAPI := schema.NewSet(
		schema.HashResource(s.Schema[recordType].Elem.(*schema.Resource)),
		records,
	)
	recordsInAPIButNotInStateFile := recordsInAPI.Difference(recordsInStateFile.(*schema.Set))
	mergedRecordsToBeSaved := recordsInConfigFile.(*schema.Set).Union(recordsInAPIButNotInStateFile)

	return mergedRecordsToBeSaved
}

// Unmarshal the config data from the terraform config file to our local types so it can be saved
func unmarshalResourceData(d *schema.ResourceData, zone *dns.Zone) {
	s := resourceFastDNSZone()

	_, ok := d.GetOk("a")
	if ok {
		if d.HasChange("a") {
			zoneARecords := make([]interface{}, len(zone.Zone.A))
			for k, v := range zone.Zone.A {
				zoneARecords[k] = v.ToMap()
			}
			mergedARecords := mergeConfigs("a", zoneARecords, s, d)
			zone.Zone.A = nil
			for _, val := range mergedARecords.List() {
				record := dns.NewARecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("aaaa")
	if ok {
		if d.HasChange("aaaa") {
			zoneAaaaRecords := make([]interface{}, len(zone.Zone.Aaaa))
			for k, v := range zone.Zone.Aaaa {
				zoneAaaaRecords[k] = v.ToMap()
			}
			mergedAaaaRecords := mergeConfigs("aaaa", zoneAaaaRecords, s, d)
			zone.Zone.Aaaa = nil
			for _, val := range mergedAaaaRecords.List() {
				record := dns.NewAaaaRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("afsdb")
	if ok {
		if d.HasChange("afsdb") {
			zoneAfsdbRecords := make([]interface{}, len(zone.Zone.Afsdb))
			for k, v := range zone.Zone.Afsdb {
				zoneAfsdbRecords[k] = v.ToMap()
			}
			mergedAfsdbRecords := mergeConfigs("afsdb", zoneAfsdbRecords, s, d)
			zone.Zone.Afsdb = nil
			for _, val := range mergedAfsdbRecords.List() {
				record := dns.NewAfsdbRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("cname")
	if ok {
		if d.HasChange("cname") {
			zoneCnameRecords := make([]interface{}, len(zone.Zone.Cname))
			for k, v := range zone.Zone.Cname {
				zoneCnameRecords[k] = v.ToMap()
			}
			mergedCnameRecords := mergeConfigs("cname", zoneCnameRecords, s, d)
			zone.Zone.Cname = nil
			for _, val := range mergedCnameRecords.List() {
				record := dns.NewCnameRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("dnskey")
	if ok {
		if d.HasChange("dnskey") {
			zoneDnskeyRecords := make([]interface{}, len(zone.Zone.Dnskey))
			for k, v := range zone.Zone.Dnskey {
				zoneDnskeyRecords[k] = v.ToMap()
			}
			mergedDnskeyRecords := mergeConfigs("dnskey", zoneDnskeyRecords, s, d)
			zone.Zone.Dnskey = nil
			for _, val := range mergedDnskeyRecords.List() {
				record := dns.NewDnskeyRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("ds")
	if ok {
		if d.HasChange("ds") {
			zoneDsRecords := make([]interface{}, len(zone.Zone.Ds))
			for k, v := range zone.Zone.Ds {
				zoneDsRecords[k] = v.ToMap()
			}
			mergedDsRecords := mergeConfigs("ds", zoneDsRecords, s, d)
			zone.Zone.Ds = nil
			for _, val := range mergedDsRecords.List() {
				record := dns.NewDsRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("hinfo")
	if ok {
		if d.HasChange("hinfo") {
			zoneHinfoRecords := make([]interface{}, len(zone.Zone.Hinfo))
			for k, v := range zone.Zone.Hinfo {
				zoneHinfoRecords[k] = v.ToMap()
			}
			mergedHinfoRecords := mergeConfigs("hinfo", zoneHinfoRecords, s, d)
			zone.Zone.Hinfo = nil
			for _, val := range mergedHinfoRecords.List() {
				record := dns.NewHinfoRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("loc")
	if ok {
		if d.HasChange("loc") {
			zoneLocRecords := make([]interface{}, len(zone.Zone.Loc))
			for k, v := range zone.Zone.Loc {
				zoneLocRecords[k] = v.ToMap()
			}
			mergedLocRecords := mergeConfigs("loc", zoneLocRecords, s, d)
			zone.Zone.Loc = nil
			for _, val := range mergedLocRecords.List() {
				record := dns.NewLocRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("mx")
	if ok {
		if d.HasChange("mx") {
			zoneMxRecords := make([]interface{}, len(zone.Zone.Mx))
			for k, v := range zone.Zone.Mx {
				zoneMxRecords[k] = v.ToMap()
			}
			mergedMxRecords := mergeConfigs("mx", zoneMxRecords, s, d)
			zone.Zone.Mx = nil
			for _, val := range mergedMxRecords.List() {
				record := dns.NewMxRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("naptr")
	if ok {
		if d.HasChange("naptr") {
			zoneNaptrRecords := make([]interface{}, len(zone.Zone.Naptr))
			for k, v := range zone.Zone.Naptr {
				zoneNaptrRecords[k] = v.ToMap()
			}
			mergedNaptrRecords := mergeConfigs("naptr", zoneNaptrRecords, s, d)
			zone.Zone.Naptr = nil
			for _, val := range mergedNaptrRecords.List() {
				record := dns.NewNaptrRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("ns")
	if ok {
		if d.HasChange("ns") {
			zoneNsRecords := make([]interface{}, len(zone.Zone.Ns))
			for k, v := range zone.Zone.Ns {
				zoneNsRecords[k] = v.ToMap()
			}
			mergedNsRecords := mergeConfigs("ns", zoneNsRecords, s, d)
			zone.Zone.Ns = nil
			for _, val := range mergedNsRecords.List() {
				record := dns.NewNsRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("nsec3")
	if ok {
		if d.HasChange("nsec3") {
			zoneNsec3Records := make([]interface{}, len(zone.Zone.Nsec3))
			for k, v := range zone.Zone.Nsec3 {
				zoneNsec3Records[k] = v.ToMap()
			}
			mergedNsec3Records := mergeConfigs("nsec3", zoneNsec3Records, s, d)
			zone.Zone.Nsec3 = nil
			for _, val := range mergedNsec3Records.List() {
				record := dns.NewNsec3Record()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("nsec3param")
	if ok {
		if d.HasChange("nsec3param") {
			zoneNsec3paramRecords := make([]interface{}, len(zone.Zone.Nsec3param))
			for k, v := range zone.Zone.Nsec3param {
				zoneNsec3paramRecords[k] = v.ToMap()
			}
			mergedNsec3paramRecords := mergeConfigs("nsec3param", zoneNsec3paramRecords, s, d)
			zone.Zone.Nsec3param = nil
			for _, val := range mergedNsec3paramRecords.List() {
				record := dns.NewNsec3paramRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("ptr")
	if ok {
		if d.HasChange("ptr") {
			zonePtrRecords := make([]interface{}, len(zone.Zone.Ptr))
			for k, v := range zone.Zone.Ptr {
				zonePtrRecords[k] = v.ToMap()
			}
			mergedPtrRecords := mergeConfigs("Ptr", zonePtrRecords, s, d)
			zone.Zone.Ptr = nil
			for _, val := range mergedPtrRecords.List() {
				record := dns.NewPtrRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("rp")
	if ok {
		if d.HasChange("rp") {
			zoneRpRecords := make([]interface{}, len(zone.Zone.Rp))
			for k, v := range zone.Zone.Rp {
				zoneRpRecords[k] = v.ToMap()
			}
			mergedRpRecords := mergeConfigs("rp", zoneRpRecords, s, d)
			zone.Zone.Rp = nil
			for _, val := range mergedRpRecords.List() {
				record := dns.NewRpRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("rrsig")
	if ok {
		if d.HasChange("rrsig") {
			zoneRrsigRecords := make([]interface{}, len(zone.Zone.Rrsig))
			for k, v := range zone.Zone.Rrsig {
				zoneRrsigRecords[k] = v.ToMap()
			}
			mergedRrsigRecords := mergeConfigs("Rrsig", zoneRrsigRecords, s, d)
			zone.Zone.Rrsig = nil
			for _, val := range mergedRrsigRecords.List() {
				record := dns.NewRrsigRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("soa")
	if ok {
		if d.HasChange("soa") {
			zoneSoaRecords := make([]interface{}, 1)
			zoneSoaRecords[0] = zone.Zone.Soa.ToMap()
			mergedSoaRecords := mergeConfigs("soa", zoneSoaRecords, s, d)
			zone.Zone.Soa = nil
			for _, val := range mergedSoaRecords.List() {
				record := dns.NewSoaRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("spf")
	if ok {
		if d.HasChange("spf") {
			zoneSpfRecords := make([]interface{}, len(zone.Zone.Spf))
			for k, v := range zone.Zone.Spf {
				zoneSpfRecords[k] = v.ToMap()
			}
			mergedSpfRecords := mergeConfigs("spf", zoneSpfRecords, s, d)
			zone.Zone.Spf = nil
			for _, val := range mergedSpfRecords.List() {
				record := dns.NewSpfRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("srv")
	if ok {
		if d.HasChange("srv") {
			zoneSrvRecords := make([]interface{}, len(zone.Zone.Srv))
			for k, v := range zone.Zone.Srv {
				zoneSrvRecords[k] = v.ToMap()
			}
			mergedSrvRecords := mergeConfigs("srv", zoneSrvRecords, s, d)
			zone.Zone.Srv = nil
			for _, val := range mergedSrvRecords.List() {
				record := dns.NewSrvRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("sshfp")
	if ok {
		if d.HasChange("sshfp") {
			zoneSshfpRecords := make([]interface{}, len(zone.Zone.Sshfp))
			for k, v := range zone.Zone.Sshfp {
				zoneSshfpRecords[k] = v.ToMap()
			}
			mergedSshfpRecords := mergeConfigs("sshfp", zoneSshfpRecords, s, d)
			zone.Zone.Sshfp = nil
			for _, val := range mergedSshfpRecords.List() {
				record := dns.NewSshfpRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}

	_, ok = d.GetOk("txt")
	if ok {
		if d.HasChange("txt") {
			zoneTxtRecords := make([]interface{}, len(zone.Zone.Txt))
			for k, v := range zone.Zone.Txt {
				zoneTxtRecords[k] = v.ToMap()
			}
			mergedTxtRecords := mergeConfigs("txt", zoneTxtRecords, s, d)
			zone.Zone.Txt = nil
			for _, val := range mergedTxtRecords.List() {
				record := dns.NewTxtRecord()
				assignFields(record, val.(map[string]interface{}))
				zone.AddRecord(record)
			}
		}
	}
}

// Only ever save data from the tf config in the tf state file, to help with
// api issues. See func unmarshalResourceData for more info.
func resourceFastDNSZoneRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceFastDNSZoneImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	hostname := d.Id()

	// find the zone first
	log.Printf("[INFO] [Akamai FastDNS] Searching for zone [%s]", hostname)
	zone, err := dns.GetZone(hostname)
	if err != nil {
		return nil, err
	}

	// assign each of the record sets to the resource data
	marshalResourceData(d, zone)

	// Give terraform the ID
	d.SetId(fmt.Sprintf("%s-%s-%s", zone.Token, zone.Zone.Name, hostname))

	return []*schema.ResourceData{d}, nil
}

func marshalResourceData(d *schema.ResourceData, zone *dns.Zone) {
	d.Set("hostname", zone.Zone.Name)

	a := make([]map[string]interface{}, len(zone.Zone.A))
	for i, v := range zone.Zone.A {
		a[i] = v.ToMap()
	}
	d.Set("a", a)

	aaaa := make([]map[string]interface{}, len(zone.Zone.Aaaa))
	for i, v := range zone.Zone.Aaaa {
		aaaa[i] = v.ToMap()
	}
	d.Set("aaaa", aaaa)

	afsdb := make([]map[string]interface{}, len(zone.Zone.Afsdb))
	for i, v := range zone.Zone.Afsdb {
		afsdb[i] = v.ToMap()
	}
	d.Set("afsdb", afsdb)

	cname := make([]map[string]interface{}, len(zone.Zone.Cname))
	for i, v := range zone.Zone.Cname {
		cname[i] = v.ToMap()
	}
	d.Set("cname", cname)

	dnskey := make([]map[string]interface{}, len(zone.Zone.Dnskey))
	for i, v := range zone.Zone.Dnskey {
		dnskey[i] = v.ToMap()
	}
	d.Set("dnskey", dnskey)

	ds := make([]map[string]interface{}, len(zone.Zone.Ds))
	for i, v := range zone.Zone.Ds {
		ds[i] = v.ToMap()
	}
	d.Set("ds", ds)

	hinfo := make([]map[string]interface{}, len(zone.Zone.Hinfo))
	for i, v := range zone.Zone.Hinfo {
		hinfo[i] = v.ToMap()
	}
	d.Set("hinfo", hinfo)

	loc := make([]map[string]interface{}, len(zone.Zone.Loc))
	for i, v := range zone.Zone.Loc {
		loc[i] = v.ToMap()
	}
	d.Set("loc", loc)

	mx := make([]map[string]interface{}, len(zone.Zone.Mx))
	for i, v := range zone.Zone.Mx {
		mx[i] = v.ToMap()
	}
	d.Set("mx", mx)

	naptr := make([]map[string]interface{}, len(zone.Zone.Naptr))
	for i, v := range zone.Zone.Naptr {
		naptr[i] = v.ToMap()
	}
	d.Set("naptr", naptr)

	ns := make([]map[string]interface{}, len(zone.Zone.Ns))
	for i, v := range zone.Zone.Ns {
		ns[i] = v.ToMap()
	}
	d.Set("ns", ns)

	nsec3 := make([]map[string]interface{}, len(zone.Zone.Nsec3))
	for i, v := range zone.Zone.Nsec3 {
		nsec3[i] = v.ToMap()
	}
	d.Set("nsec3", nsec3)

	nsec3param := make([]map[string]interface{}, len(zone.Zone.Nsec3param))
	for i, v := range zone.Zone.Nsec3param {
		nsec3param[i] = v.ToMap()
	}
	d.Set("nsec3param", nsec3param)

	ptr := make([]map[string]interface{}, len(zone.Zone.Ptr))
	for i, v := range zone.Zone.Ptr {
		ptr[i] = v.ToMap()
	}
	d.Set("ptr", ptr)

	rp := make([]map[string]interface{}, len(zone.Zone.Rp))
	for i, v := range zone.Zone.Rp {
		rp[i] = v.ToMap()
	}
	d.Set("rp", rp)

	rrsig := make([]map[string]interface{}, len(zone.Zone.Rrsig))
	for i, v := range zone.Zone.Rrsig {
		rrsig[i] = v.ToMap()
	}
	d.Set("rrsig", rrsig)

	soa := make([]map[string]interface{}, 1)
	soa[0] = zone.Zone.Soa.ToMap()
	d.Set("soa", soa)

	spf := make([]map[string]interface{}, len(zone.Zone.Spf))
	for i, v := range zone.Zone.Spf {
		spf[i] = v.ToMap()
	}
	d.Set("spf", spf)

	srv := make([]map[string]interface{}, len(zone.Zone.Srv))
	for i, v := range zone.Zone.Srv {
		srv[i] = v.ToMap()
	}
	d.Set("srv", srv)

	sshfp := make([]map[string]interface{}, len(zone.Zone.Sshfp))
	for i, v := range zone.Zone.Sshfp {
		sshfp[i] = v.ToMap()
	}
	d.Set("sshfp", sshfp)

	txt := make([]map[string]interface{}, len(zone.Zone.Txt))
	for i, v := range zone.Zone.Txt {
		txt[i] = v.ToMap()
	}
	d.Set("txt", txt)
}

func resourceFastDNSZoneDelete(d *schema.ResourceData, meta interface{}) error {
	dnsWriteLock.Lock()
	defer dnsWriteLock.Unlock()

	hostname := d.Get("hostname").(string)

	// find the zone first
	log.Printf("[INFO] [Akamai FastDNS] Searching for zone [%s]", hostname)
	zone, err := dns.GetZone(hostname)
	if err != nil {
		return err
	}

	// 'delete' the zone - this is a soft delete which
	// will just remove the non required records
	err = zone.Delete()
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourceFastDNSZoneExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	hostname := d.Get("hostname").(string)

	// try to get the zone from the API
	log.Printf("[INFO] [Akamai FastDNS] Searching for zone [%s]", hostname)
	zone, err := dns.GetZone(hostname)
	return zone != nil, err
}
