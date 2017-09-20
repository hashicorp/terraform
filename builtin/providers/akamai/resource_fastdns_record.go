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

// get the zone, fetch the resource (just one) from the .tf file, update the zone, send to api
func resourceFastDNSRecordCreate(d *schema.ResourceData, meta interface{}) error {
	recordType := strings.ToUpper(d.Get("type").(string))

	if recordType == "SOA" {
		log.Printf("[INFO] [Akamai FastDNS]: Creating %s Record on %s", recordType, d.Get("hostname"))
	} else {
		log.Printf("[INFO] [Akamai FastDNS] Creating %s Record \"%s\" on %s", recordType, d.Get("name"), d.Get("hostname"))
	}

	zone, e := dns.GetZone(d.Get("hostname").(string))

	if e != nil {
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

	records := dns.RecordSet{}
	unmarshalResourceData(d, records)

	var name string
	if recordType == "SOA" {
		zone.Zone.Soa = records[0].(dns.SoaRecord)
		name = recordType
	} else {
		name = d.Get("name").(string)
		for _, record := range records {
			err := zone.AddRecord(record)
			if err != nil {
				return err
			}
		}
	}

	e = zone.Save()
	// if e.(client.APIError).Status == 409 {
	//tempZone, err := config.ConfigDNSV1Service.GetZone(d.Get("hostname").(string))
	// }

	if e != nil {
		return e
	}

	d.SetId(fmt.Sprintf("%s-%s-%s-%s", zone.Token, zone.Zone.Name, recordType, name))

	return nil
}

/*
func resourceFastDNSRecordRead(d *schema.ResourceData, meta interface{}) error {
	log.Println("[INFO] [Akamai FastDNS] resourcePropertyRead")

	zone, err := dns.GetZone(d.Get("hostname").(string))
	if err != nil {
		return err
	}

	log.Printf("[INFO] [Akamai FastDNS] Resource Data:\n\n%#v\n\n", &d)

	token, _, recordType, name := getDNSRecordId(d.Id())

	if zone.Token != token {
		log.Println("[WARN] [Akamai FastDNS] Resource has been modified, aborting")
		return errors.New("Resource has been modified, aborting")
	}

	recordSet := zone.GetRecordType(recordType).(dns.RecordSet)
	err = marshalResourceData(d, &recordSet)
	if err != nil {
		return err
	}

	id := fmt.Sprintf("%s-%s-%s-%s", zone.Token, zone.Zone.Name, recordType, name)
	log.Println("[INFO] [Akamai FastDNS] Read ID: " + id)
	d.SetId(id)

	return nil
}

func resourceFastDNSRecordDelete(d *schema.ResourceData, meta interface{}) error {
	zone, err := dns.GetZone(d.Get("hostname").(string))
	if err != nil {
		return err
	}

	recordType := strings.ToUpper(d.Get("type").(string))

	records := dns.RecordSet{}
	error := unmarshalResourceData(d, &records)

	if error != nil {
		return error
	}

	if recordType == "SOA" {
		zone.Zone.Soa = nil
	} else {
		name := d.Get("name").(string)

		zone.RemoveRecordsByName(name, []string{recordType})
	}

	error = zone.Save()
	if error != nil {
		return error
	}

	d.SetId("")

	return nil
}

func resourceFastDNSRecordExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	log.Println("[INFO] [Akamai FastDNS] resourcePropertyExists")

	zone, err := dns.GetZone(d.Get("hostname").(string))
	if err != nil {
		log.Println("[WARN] [Akamai FastDNS] Error checking if record exists: " + err.Error())
		return false, err
	}

	token, _, recordType, name := getDNSRecordId(d.Id())
	name = strings.TrimSuffix(name, ".")

	if zone.Token != token {
		log.Printf("[WARN] [Akamai FastDNS] Token mismatch: Remote: %s, Local: %s", zone.Token, token)
		return false, nil
	}

	var found_record bool
	for _, record := range zone.GetRecordType(recordType).(dns.RecordSet) {
		if strings.TrimSuffix(record.Name, ".") == name {
			found_record = true
			break
		} else {
			log.Printf("[TRACE] [Akamai FastDNS] record.Name: %s != name: %s", record.Name, name)
		}
	}

	if found_record {
		log.Println("[INFO] [Akamai FastDNS] Record found")
	} else {
		log.Println("[INFO] [Akamai FastDNS] Record not found")
	}
	return found_record, nil
}

func getDNSRecordId(id string) (token string, hostname string, recordType string, name string) {
	parts := strings.Split(id, "-")
	return parts[0], parts[1], parts[2], parts[3]
}
*/

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
func unmarshalResourceData(d *schema.ResourceData, records dns.RecordSet) {
	// We will get 1 record at a time from terraform
	// For example an MX record
	// Any record can have 1 or more targets
	// We'll make a record for each target and add them to the record set
	recordType := strings.ToUpper(d.Get("type").(string))
	targets := d.Get("targets").(*schema.Set).Len() //unsafe
	log.Printf("[DEBUG] [Akamai FastDNS] Record type is %s", recordType)
	// for each target listed, create a record in the record set
	for i := 0; i < targets; i++ {
		switch recordType {
		case "A":
			record := dns.NewARecord()
			log.Printf("[DEBUG] [Akamai FastDNS] Creating A Record")
			assignFields(record, d, i)
			log.Printf("[DEBUG] [Akamai FastDNS] A Record is: %s", record)
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
}

/*
func marshalResourceData(d *schema.ResourceData, records *dns.RecordSet) error {
	if len(*records) == 0 {
		return nil
	}

	for _, record := range *records {
		if val, exists := d.GetOk("targets"); exists != false {
			val.(*schema.Set).Add(record.Target)
			d.Set("targets", val)
		} else {
			set := &schema.Set{}
			set.Add(record.Target)
			d.Set("targets", set)
		}

		d.Set("ttl", record.TTL)
		d.Set("name", record.Name)
		d.Set("active", record.Active)
		d.Set("subtype", record.Subtype)
		d.Set("flags", record.Flags)
		d.Set("protocol", record.Protocol)
		d.Set("algorithm", record.Algorithm)
		d.Set("key", record.Key)
		d.Set("keytag", record.Keytag)
		d.Set("digesttype", record.DigestType)
		d.Set("digest", record.Digest)
		d.Set("hardware", record.Hardware)
		d.Set("software", record.Software)
		d.Set("priority", record.Priority)
		d.Set("order", record.Order)
		d.Set("preference", record.Preference)
		d.Set("service", record.Service)
		d.Set("regexp", record.Regexp)
		d.Set("replacement", record.Replacement)
		d.Set("iterations", record.Iterations)
		d.Set("salt", record.Salt)
		d.Set("nexthashedownername", record.NextHashedOwnerName)
		d.Set("typebitmaps", record.TypeBitmaps)
		d.Set("mailbox", record.Mailbox)
		d.Set("txt", record.Txt)
		d.Set("typecovered", record.TypeCovered)
		d.Set("originalttl", record.OriginalTTL)
		d.Set("expiration", record.Expiration)
		d.Set("inception", record.Inception)
		d.Set("signer", record.Signer)
		d.Set("signature", record.Signature)
		d.Set("labels", record.Labels)
		d.Set("originserver", record.Originserver)
		d.Set("contact", record.Contact)
		d.Set("serial", record.Serial)
		d.Set("refresh", record.Refresh)
		d.Set("retry", record.Retry)
		d.Set("expire", record.Expire)
		d.Set("minimum", record.Minimum)
		d.Set("weight", record.Weight)
		d.Set("port", record.Port)
		d.Set("fingerprinttype", record.FingerprintType)
		d.Set("fingerprint", record.Fingerprint)
	}

	return nil
}

func importRecord(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	zone, err := dns.GetZone(d.Get("hostname").(string))
	if err != nil {
		return nil, err
	}

	_, hostname, recordType, name := getDNSRecordId("_." + d.Id())

	var exists bool
	for _, record := range zone.GetRecordType(recordType).(dns.RecordSet) {
		if strings.ToLower(record.Name) == name {
			exists = true
		}
	}

	if exists == true {
		d.SetId(fmt.Sprintf("%s-%s-%s-%s", zone.Token, hostname, recordType, name))
		return []*schema.ResourceData{d}, nil
	}

	return nil, errors.New(fmt.Sprintf("Resource \"%s\" not found", d.Id()))
}
*/
