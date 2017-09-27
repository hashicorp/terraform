package akamai

import (
	"fmt"
	"log"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
				Required: true,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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
				Type:     schema.TypeList,
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

// Unmarshal the config data from the terraform config file to our local types
func unmarshalResourceData(d *schema.ResourceData, zone *dns.Zone) {
	a, ok := d.GetOk("a")
	if ok {
		for _, val := range a.([]interface{}) {
			record := dns.NewARecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	aaaa, ok := d.GetOk("aaaa")
	if ok {
		for _, val := range aaaa.([]interface{}) {
			record := dns.NewAaaaRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	afsdb, ok := d.GetOk("afsdb")
	if ok {
		for _, val := range afsdb.([]interface{}) {
			record := dns.NewAfsdbRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	cname, ok := d.GetOk("cname")
	if ok {
		for _, val := range cname.([]interface{}) {
			record := dns.NewCnameRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	dnskey, ok := d.GetOk("dnskey")
	if ok {
		for _, val := range dnskey.([]interface{}) {
			record := dns.NewDnskeyRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	ds, ok := d.GetOk("ds")
	if ok {
		for _, val := range ds.([]interface{}) {
			record := dns.NewDsRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	hinfo, ok := d.GetOk("hinfo")
	if ok {
		for _, val := range hinfo.([]interface{}) {
			record := dns.NewHinfoRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	loc, ok := d.GetOk("loc")
	if ok {
		for _, val := range loc.([]interface{}) {
			record := dns.NewLocRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	mx, ok := d.GetOk("mx")
	if ok {
		for _, val := range mx.([]interface{}) {
			record := dns.NewMxRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	naptr, ok := d.GetOk("naptr")
	if ok {
		for _, val := range naptr.([]interface{}) {
			record := dns.NewNaptrRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	ns, ok := d.GetOk("ns")
	if ok {
		for _, val := range ns.([]interface{}) {
			record := dns.NewNsRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	nsec3, ok := d.GetOk("nsec3")
	if ok {
		for _, val := range nsec3.([]interface{}) {
			record := dns.NewNsec3Record()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	nsec3param, ok := d.GetOk("nsec3param")
	if ok {
		for _, val := range nsec3param.([]interface{}) {
			record := dns.NewNsec3paramRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	ptr, ok := d.GetOk("ptr")
	if ok {
		for _, val := range ptr.([]interface{}) {
			record := dns.NewPtrRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	rp, ok := d.GetOk("rp")
	if ok {
		for _, val := range rp.([]interface{}) {
			record := dns.NewRpRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	rrsig, ok := d.GetOk("rrsig")
	if ok {
		for _, val := range rrsig.([]interface{}) {
			record := dns.NewRrsigRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	soa, ok := d.GetOk("soa")
	if ok {
		for _, val := range soa.([]interface{}) {
			record := dns.NewSoaRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	spf, ok := d.GetOk("spf")
	if ok {
		for _, val := range spf.([]interface{}) {
			record := dns.NewSpfRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	srv, ok := d.GetOk("srv")
	if ok {
		for _, val := range srv.([]interface{}) {
			record := dns.NewSrvRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	sshfp, ok := d.GetOk("sshfp")
	if ok {
		for _, val := range sshfp.([]interface{}) {
			record := dns.NewSshfpRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}

	txt, ok := d.GetOk("txt")
	if ok {
		for _, val := range txt.([]interface{}) {
			record := dns.NewTxtRecord()
			assignFields(record, val.(map[string]interface{}))
			zone.AddRecord(record)
		}
	}
}

func resourceFastDNSZoneRead(d *schema.ResourceData, meta interface{}) error {
	hostname := d.Get("hostname").(string)

	// find the zone first
	log.Printf("[INFO] [Akamai FastDNS] Searching for zone [%s]", hostname)
	zone, err := dns.GetZone(hostname)
	if err != nil {
		return err
	}

	// assign each of the record sets to the resource data
	marshalResourceData(d, zone)

	// Give terraform the ID
	d.SetId(fmt.Sprintf("%s-%s-%s", zone.Token, zone.Zone.Name, hostname))

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

	d.Set("soa", zone.Zone.Soa.ToMap())

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
