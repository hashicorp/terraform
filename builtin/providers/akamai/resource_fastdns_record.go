package akamai

import (
	"fmt"
	"log"
	"strings"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceFastDNSRecordRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}
func resourceFastDNSRecordDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
func resourceFastDNSRecordExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	return true, nil
}

func resourceFastDNSRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceFastDNSRecordCreate,
		Read:   resourceFastDNSRecordRead,
		Update: resourceFastDNSRecordCreate,
		Delete: resourceFastDNSRecordDelete,
		Exists: resourceFastDNSRecordExists,
		// Importer: &schema.ResourceImporter{
		// 	State: importRecord,
		// },
		Schema: map[string]*schema.Schema{
			// Terraform-only Params
			"hostname": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
			},

			// Special to allow multiple targets:
			"targets": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			// DNS Record attributes
			"active": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"algorithm": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"contact": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"digest": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"digesttype": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"expiration": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"expire": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"fingerprint": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"fingerprinttype": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"flags": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"hardware": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"inception": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"iterations": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"key": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"keytag": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"labels": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"mailbox": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"minimum": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"nexthashedownername": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"order": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"originalttl": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"originserver": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"preference": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"priority": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"protocol": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"refresh": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"regexp": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"replacement": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"retry": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"salt": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"serial": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"service": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"signature": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"signer": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"software": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"subtype": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"ttl": {
				Type:     schema.TypeInt,
				Optional: true,
			},
			"txt": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"typebitmaps": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"typecovered": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"weight": {
				Type:     schema.TypeInt, // Should be uint
				Optional: true,
			},
		},
	}
}

// Create a new DNS Record
func resourceFastDNSRecordCreate(d *schema.ResourceData, meta interface{}) error {
	recordType := strings.ToUpper(d.Get("type").(string))
	name := d.Get("name").(string)

	if recordType == "SOA" {
		name = recordType
		log.Printf("[INFO] [Akamai FastDNS]: Creating %s Record on %s", recordType, d.Get("hostname"))
	} else {
		log.Printf("[INFO] [Akamai FastDNS] Creating %s Record \"%s\" on %s", recordType, d.Get("name"), d.Get("hostname"))
	}

	// First try to get the zone from the API
	zone, e := dns.GetZone(d.Get("hostname").(string))

	if e != nil {
		// If there's no existing zone we'll create a blank one
		if dns.IsConfigDNSError(e) && e.(dns.ConfigDNSError).NotFound() == true {
			// if the zone is not found/404 we will create a new
			// blank zone for the records to be added to and continue
			log.Printf("[DEBUG] [Akamai FastDNS] [ERROR] %s", e.Error())
			log.Printf("[DEBUG] [Akamai FastDNS] Creating new zone")
			zone = dns.NewZone(d.Get("hostname").(string))
			e = nil
		} else {
			return e
		}
	}

	// Transform the record data from the terraform config to local types
	// Then add each record to the zone
	records := unmarshalResourceData(d)
	for _, record := range records {
		err := zone.AddRecord(record)
		if err != nil {
			return err
		}
	}

	// Save the zone to the API
	e = zone.Save()
	if e != nil {
		return e
	}

	// Give terraform the ID
	d.SetId(fmt.Sprintf("%s-%s-%s-%s", zone.Token, zone.Zone.Name, recordType, name))

	return nil
}

// Helper function for unmarshalResourceData() below
func assignFields(record dns.DNSRecord, d *schema.ResourceData, i int) {
	f := record.GetAllowedFields()
	for _, field := range f {
		val, exists := d.GetOk(field)
		if !exists {
			// TODO: maybe not all fields are required?
			log.Printf("[WARN] [Akamai FastDNS] Field [%s] is missing from your terraform config", field)
		} else {
			if field == "targets" {
				val = val.(*schema.Set).List()[i].(string) //unsafe?
			}
			e := record.SetField(field, val)
			if e != nil {
				log.Printf("[WARN] [Akamai FastDNS] Couldn't add field to record: %s", e.Error())
			}
		}
	}
}

// Unmarshal the config data from the terraform config file to our local types
func unmarshalResourceData(d *schema.ResourceData) dns.RecordSet {
	// We will get 1 record at a time from terraform
	// For example an MX record
	// Any record can have 1 or more targets
	// We'll make a record for each target and add them to the record set
	records := dns.RecordSet{}
	recordType := strings.ToUpper(d.Get("type").(string))
	targets := d.Get("targets").(*schema.Set).Len() //unsafe
	// for each target listed, create a record in the record set
	for i := 0; i < targets; i++ {
		switch recordType {
		case "A":
			record := dns.NewARecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "AAAA":
			record := dns.NewAaaaRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "AFSDB":
			record := dns.NewAfsdbRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "CNAME":
			record := dns.NewCnameRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "DNSKEY":
			record := dns.NewDnskeyRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "DS":
			record := dns.NewDsRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "HINFO":
			record := dns.NewHinfoRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "LOC":
			record := dns.NewLocRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "MX":
			record := dns.NewMxRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "NAPTR":
			record := dns.NewNaptrRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "NS":
			record := dns.NewNsRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "NSEC3":
			record := dns.NewNsec3Record()
			assignFields(record, d, i)
			records = append(records, record)
		case "NSEC3PARAM":
			record := dns.NewNsec3paramRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "PTR":
			record := dns.NewPtrRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "RP":
			record := dns.NewRpRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "RRSIG":
			record := dns.NewRrsigRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "SOA":
			record := dns.NewSoaRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "SPF":
			record := dns.NewSpfRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "SRV":
			record := dns.NewSrvRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "SSHFP":
			record := dns.NewSshfpRecord()
			assignFields(record, d, i)
			records = append(records, record)
		case "TXT":
			record := dns.NewTxtRecord()
			assignFields(record, d, i)
			records = append(records, record)
		}
	}

	return records
}
