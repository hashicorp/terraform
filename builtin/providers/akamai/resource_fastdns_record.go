package akamai

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/akamai/AkamaiOPEN-edgegrid-golang/client-v1"
	"github.com/akamai/AkamaiOPEN-edgegrid-golang/configdns-v1"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceFastDNSRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceFastDNSRecordCreate,
		Read:   resourceFastDNSRecordRead,
		Update: resourceFastDNSRecordCreate,
		Delete: resourceFastDNSRecordDelete,
		Exists: resourceFastDNSRecordExists,
		Importer: &schema.ResourceImporter{
			State: importRecord,
		},
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

func resourceFastDNSRecordCreate(d *schema.ResourceData, meta interface{}) error {
	recordType := strings.ToUpper(d.Get("type").(string))

	if recordType == "SOA" {
		log.Printf("[INFO] [Akamai FastDNS]: Creating %s Record on %s", recordType, d.Get("hostname"))
	} else {
		log.Printf("[INFO] [Akamai FastDNS] Creating %s Record \"%s\" on %s", recordType, d.Get("name"), d.Get("hostname"))
	}

	zone, error := dns.GetZone(d.Get("hostname").(string))

	if error != nil {
		return error
	}

	records := dns.RecordSet{}
	error = unmarshalResourceData(d, &records)

	if error != nil {
		return error
	}

	var name string
	if recordType == "SOA" {
		zone.Zone.Soa = records[0]
		name = recordType
	} else {
		name = d.Get("name").(string)

		for _, record := range records {
			err := zone.SetRecord(record)
			if err != nil {
				return err
			}
		}
	}

	error = zone.Save()
	if error.(client.APIError).Status == 409 {
		//tempZone, err := config.ConfigDNSV1Service.GetZone(d.Get("hostname").(string))
	}

	if error != nil {
		return error
	}

	d.SetId(fmt.Sprintf("%s-%s-%s-%s", zone.Token, zone.Zone.Name, recordType, name))

	return nil
}

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

func unmarshalResourceData(d *schema.ResourceData, records *dns.RecordSet) error {
	recordType := strings.ToUpper(d.Get("type").(string))
	tester := dns.Record{RecordType: recordType}
	recordSet := dns.RecordSet{}

	targets := 1
	if tester.Allows("targets") {
		targets = d.Get("targets").(*schema.Set).Len()
	}

	for i := 0; i < targets; i++ {
		record := dns.Record{RecordType: recordType}

		if val, exists := d.GetOk("targets"); exists != false && !record.Allows("targets") {
			return errors.New("Attribute \"targets\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Target = val.(*schema.Set).List()[i].(string)
		} else {
			record.Target = ""
		}

		if val, exists := d.GetOk("ttl"); exists != false && !record.Allows("ttl") {
			return errors.New("Attribute \"ttl\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.TTL = val.(int)
		} else {
			return errors.New("Attribute \"ttl\" is required for record type " + record.RecordType)
		}

		if val, exists := d.GetOk("name"); exists != false && !record.Allows("name") {
			return errors.New("Attribute \"name\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Name = val.(string)
		} else {
			return errors.New("Attribute \"name\" is required for record type " + record.RecordType)
		}

		if val, exists := d.GetOk("active"); exists != false && !record.Allows("active") {
			return errors.New("Attribute \"active\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Active = val.(bool)
		} else {
			record.Active = true
		}

		if val, exists := d.GetOk("subtype"); exists != false && !record.Allows("subtype") {
			return errors.New("Attribute \"subtype\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Subtype = val.(int)
		} else {
			record.Subtype = 0
		}

		if val, exists := d.GetOk("flags"); exists != false && !record.Allows("flags") {
			return errors.New("Attribute \"flags\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Flags = val.(int)
		} else {
			record.Flags = 0
		}

		if val, exists := d.GetOk("protocol"); exists != false && !record.Allows("protocol") {
			return errors.New("Attribute \"protocol\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Protocol = val.(int)
		} else {
			record.Protocol = 0
		}

		if val, exists := d.GetOk("algorithm"); exists != false && !record.Allows("algorithm") {
			return errors.New("Attribute \"algorithm\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Algorithm = val.(int)
		} else {
			record.Algorithm = 0
		}

		if val, exists := d.GetOk("key"); exists != false && !record.Allows("key") {
			return errors.New("Attribute \"key\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Key = val.(string)
		} else {
			record.Key = ""
		}

		if val, exists := d.GetOk("keytag"); exists != false && !record.Allows("keytag") {
			return errors.New("Attribute \"keytag\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Keytag = val.(int)
		} else {
			record.Keytag = 0
		}

		if val, exists := d.GetOk("digesttype"); exists != false && !record.Allows("digesttype") {
			return errors.New("Attribute \"digesttype\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.DigestType = val.(int)
		} else {
			record.DigestType = 0
		}

		if val, exists := d.GetOk("digest"); exists != false && !record.Allows("digest") {
			return errors.New("Attribute \"digest\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Digest = val.(string)
		} else {
			record.Digest = ""
		}

		if val, exists := d.GetOk("hardware"); exists != false && !record.Allows("hardware") {
			return errors.New("Attribute \"hardware\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Hardware = val.(string)
		} else {
			record.Hardware = ""
		}

		if val, exists := d.GetOk("software"); exists != false && !record.Allows("software") {
			return errors.New("Attribute \"software\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Software = val.(string)
		} else {
			record.Software = ""
		}

		if val, exists := d.GetOk("priority"); exists != false && !record.Allows("priority") {
			return errors.New("Attribute \"priority\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Priority = val.(int)
		} else {
			record.Priority = 0
		}

		if val, exists := d.GetOk("order"); exists != false && !record.Allows("order") {
			return errors.New("Attribute \"order\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Order = val.(int)
		} else {
			record.Order = 0
		}

		if val, exists := d.GetOk("preference"); exists != false && !record.Allows("preference") {
			return errors.New("Attribute \"preference\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Preference = val.(int)
		} else {
			record.Preference = 0
		}

		if val, exists := d.GetOk("service"); exists != false && !record.Allows("service") {
			return errors.New("Attribute \"service\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Service = val.(string)
		} else {
			record.Service = ""
		}

		if val, exists := d.GetOk("regexp"); exists != false && !record.Allows("regexp") {
			return errors.New("Attribute \"regexp\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Regexp = val.(string)
		} else {
			record.Regexp = ""
		}

		if val, exists := d.GetOk("replacement"); exists != false && !record.Allows("replacement") {
			return errors.New("Attribute \"replacement\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Replacement = val.(string)
		} else {
			record.Replacement = ""
		}

		if val, exists := d.GetOk("iterations"); exists != false && !record.Allows("iterations") {
			return errors.New("Attribute \"iterations\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Iterations = val.(int)
		} else {
			record.Iterations = 0
		}

		if val, exists := d.GetOk("salt"); exists != false && !record.Allows("salt") {
			return errors.New("Attribute \"salt\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Salt = val.(string)
		} else {
			record.Salt = ""
		}

		if val, exists := d.GetOk("nexthashedownername"); exists != false && !record.Allows("nexthashedownername") {
			return errors.New("Attribute \"nexthashedownername\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.NextHashedOwnerName = val.(string)
		} else {
			record.NextHashedOwnerName = ""
		}

		if val, exists := d.GetOk("typebitmaps"); exists != false && !record.Allows("typebitmaps") {
			return errors.New("Attribute \"typebitmaps\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.TypeBitmaps = val.(string)
		} else {
			record.TypeBitmaps = ""
		}

		if val, exists := d.GetOk("mailbox"); exists != false && !record.Allows("mailbox") {
			return errors.New("Attribute \"mailbox\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Mailbox = val.(string)
		} else {
			record.Mailbox = ""
		}

		if val, exists := d.GetOk("txt"); exists != false && !record.Allows("txt") {
			return errors.New("Attribute \"txt\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Txt = val.(string)
		} else {
			record.Txt = ""
		}

		if val, exists := d.GetOk("typecovered"); exists != false && !record.Allows("typecovered") {
			return errors.New("Attribute \"typecovered\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.TypeCovered = val.(string)
		} else {
			record.TypeCovered = ""
		}

		if val, exists := d.GetOk("originalttl"); exists != false && !record.Allows("originalttl") {
			return errors.New("Attribute \"originalttl\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.OriginalTTL = val.(int)
		} else {
			record.OriginalTTL = 0
		}

		if val, exists := d.GetOk("expiration"); exists != false && !record.Allows("expiration") {
			return errors.New("Attribute \"expiration\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Expiration = val.(string)
		} else {
			record.Expiration = ""
		}

		if val, exists := d.GetOk("inception"); exists != false && !record.Allows("inception") {
			return errors.New("Attribute \"inception\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Inception = val.(string)
		} else {
			record.Inception = ""
		}

		if val, exists := d.GetOk("signer"); exists != false && !record.Allows("signer") {
			return errors.New("Attribute \"signer\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Signer = val.(string)
		} else {
			record.Signer = ""
		}

		if val, exists := d.GetOk("signature"); exists != false && !record.Allows("signature") {
			return errors.New("Attribute \"signature\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Signature = val.(string)
		} else {
			record.Signature = ""
		}

		if val, exists := d.GetOk("labels"); exists != false && !record.Allows("labels") {
			return errors.New("Attribute \"labels\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Labels = val.(int)
		} else {
			record.Labels = 0
		}

		if val, exists := d.GetOk("originserver"); exists != false && !record.Allows("originserver") {
			return errors.New("Attribute \"originserver\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Originserver = val.(string)
		} else {
			record.Originserver = ""
		}

		if val, exists := d.GetOk("contact"); exists != false && !record.Allows("contact") {
			return errors.New("Attribute \"contact\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Contact = val.(string)
		} else {
			record.Contact = ""
		}

		if val, exists := d.GetOk("serial"); exists != false && !record.Allows("serial") {
			return errors.New("Attribute \"serial\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Serial = val.(int)
		} else {
			record.Serial = 0
		}

		if val, exists := d.GetOk("refresh"); exists != false && !record.Allows("refresh") {
			return errors.New("Attribute \"refresh\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Refresh = val.(int)
		} else {
			record.Refresh = 0
		}

		if val, exists := d.GetOk("retry"); exists != false && !record.Allows("retry") {
			return errors.New("Attribute \"retry\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Retry = val.(int)
		} else {
			record.Retry = 0
		}

		if val, exists := d.GetOk("expire"); exists != false && !record.Allows("expire") {
			return errors.New("Attribute \"expire\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Expire = val.(int)
		} else {
			record.Expire = 0
		}

		if val, exists := d.GetOk("minimum"); exists != false && !record.Allows("minimum") {
			return errors.New("Attribute \"minimum\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Minimum = val.(int)
		} else {
			record.Minimum = 0
		}

		if val, exists := d.GetOk("weight"); exists != false && !record.Allows("weight") {
			return errors.New("Attribute \"weight\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Weight = val.(uint)
		} else {
			record.Weight = 0
		}

		if val, exists := d.GetOk("port"); exists != false && !record.Allows("port") {
			return errors.New("Attribute \"port\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Port = val.(int)
		} else {
			record.Port = 0
		}

		if val, exists := d.GetOk("fingerprinttype"); exists != false && !record.Allows("fingerprinttype") {
			return errors.New("Attribute \"fingerprinttype\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.FingerprintType = val.(int)
		} else {
			record.FingerprintType = 0
		}

		if val, exists := d.GetOk("fingerprint"); exists != false && !record.Allows("fingerprint") {
			return errors.New("Attribute \"fingerprint\" not allowed for record type " + record.RecordType)
		} else if exists != false {
			record.Fingerprint = val.(string)
		} else {
			record.Fingerprint = ""
		}

		recordSet = append(recordSet, &record)
	}

	*records = recordSet

	return nil
}

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
