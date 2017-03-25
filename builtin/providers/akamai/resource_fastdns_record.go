package akamai

import (
	"errors"
	"fmt"
	"github.com/akamai-open/AkamaiOPEN-edgegrid-golang"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
	"strings"
	"time"
)

func resourceFastDnsRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceFastDnsRecordCreate,
		Read:   resourceFastDnsRecordRead,
		Update: resourceFastDnsRecordCreate,
		Delete: resourceFastDnsRecordDelete,
		Exists: resourceFastDnsRecordExists,
		Importer: &schema.ResourceImporter{
			State: importRecord,
		},
		Schema: map[string]*schema.Schema{
			// Terraform-only Params
			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			// Special to allow multiple targets:
			"targets": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			// DNS Record attributes
			"active": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"algorithm": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"contact": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"digest": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"digesttype": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"expiration": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"expire": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"fingerprinttype": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"flags": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"hardware": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"inception": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"iterations": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"key": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"keytag": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"labels": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"mailbox": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"minimum": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"nexthashedownername": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"order": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"originalttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"originserver": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"preference": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"priority": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"protocol": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"refresh": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"regexp": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"replacement": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"retry": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"salt": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"serial": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"service": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"signature": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"signer": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"software": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"subtype": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"txt": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"typebitmaps": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"typecovered": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"weight": &schema.Schema{
				Type:     schema.TypeInt, // Should be uint
				Optional: true,
			},
		},
	}
}

func resourceFastDnsRecordCreate(d *schema.ResourceData, meta interface{}) error {
	recordType := strings.ToUpper(d.Get("type").(string))

	if recordType == "SOA" {
		log.Printf("[INFO] Creating %s Record on %s", recordType, d.Get("hostname"))
	} else {
		log.Printf("[INFO] Creating %s Record \"%s\" on %s", recordType, d.Get("name"), d.Get("hostname"))
	}

	config := meta.(*Config)

	zone := config.ClientFastDns
	error := zone.GetZone(d.Get("hostname").(string))

	if error != nil {
		return error
	}

	records := DnsRecordSet{}
	error = records.unmarshalResourceData(d)

	if error != nil {
		return error
	}

	var name string
	if recordType == "SOA" {
		zone.Zone.Records[recordType] = append(zone.Zone.Records[recordType], records[0])
		name = recordType
	} else {
		name = d.Get("name").(string)

		// Add existing records unless they have the same name
		for _, v := range zone.Zone.Records[recordType] {
			if v.Name != name {
				records = append(records, v)
			}
		}

		zone.Zone.Records[recordType] = records
		zone.fixupCnames(records[0])
	}

	error = zone.Save()
	if error != nil {
		return error
	}

	d.SetId(fmt.Sprintf("%s-%s-%s-%s", zone.Token, zone.Zone.Name, recordType, name))

	return nil
}

func resourceFastDnsRecordRead(d *schema.ResourceData, meta interface{}) error {
	log.Println("[INFO] resourceFastDnsRecordRead")
	config := meta.(*Config)

	zone := config.ClientFastDns
	zone.GetZone(d.Get("hostname").(string))
	log.Printf("[INFO] Resource Data:\n\n%#v\n\n", &d)

	token, _, recordType, name := zone.getRecordId(d.Id())

	if zone.Token != token {
		log.Println("[WARN] Resource has been modified, aborting")
		return errors.New("Resource has been modified, aborting")
	}

	recordSet := DnsRecordSet{}
	for _, record := range zone.Zone.Records[recordType] {
		if record.Name == name {
			recordSet = append(recordSet, record)
		}
	}

	err := recordSet.marshalResourceData(d)
	if err != nil {
		return err
	}

	id := fmt.Sprintf("%s-%s-%s-%s", zone.Token, zone.Zone.Name, recordType, name)
	log.Println("[INFO] Read ID: " + id)
	d.SetId(id)

	return nil
}

func resourceFastDnsRecordDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	zone := config.ClientFastDns
	zone.GetZone(d.Get("hostname").(string))

	recordType := strings.ToUpper(d.Get("type").(string))

	records := DnsRecordSet{}
	error := records.unmarshalResourceData(d)

	if error != nil {
		return error
	}

	if recordType == "SOA" {
		zone.Zone.Records[recordType] = nil
	} else {
		name := d.Get("name").(string)

		newRecords := DnsRecordSet{}
		for _, v := range zone.Zone.Records[recordType] {
			if v.Name != name {
				newRecords = append(newRecords, v)
			}
		}

		zone.Zone.Records[recordType] = records
	}

	error = zone.Save()
	if error != nil {
		return error
	}

	d.SetId("")

	return nil
}

func resourceFastDnsRecordExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	config := meta.(*Config)

	zone := config.ClientFastDns
	error := zone.GetZone(d.Get("hostname").(string))

	if error != nil {
		log.Println("[WARN] Error checking if record exists: " + error.Error())
		return false, error
	}

	token, _, recordType, name := zone.getRecordId(d.Id())
	name += "."

	if zone.Token != token {
		log.Println("[WARN] Token mismatch")
		return false, nil
	}

	zone.unmarshalRecords()

	var found_record bool
	for _, record := range zone.Zone.Records[recordType] {
		if record.Name == name {
			found_record = true
			break
		}
	}

	if found_record {
		log.Println("[INFO] Record found")
	} else {
		log.Println("[INFO] Record not found")
	}
	return found_record, nil
}

type DnsZone struct {
	client *edgegrid.Client
	Token  string `json:"token"`
	Zone   struct {
		Name       string                  `json:"name,omitempty"`
		A          []*DnsRecord            `json:"a,omitempty"`
		AAAA       []*DnsRecord            `json:"aaaa,omitempty"`
		Afsdb      []*DnsRecord            `json:"afsdb,omitempty"`
		Cname      []*DnsRecord            `json:"cname,omitempty"`
		Dnskey     []*DnsRecord            `json:"dnskey,omitempty"`
		Ds         []*DnsRecord            `json:"ds,omitempty"`
		Hinfo      []*DnsRecord            `json:"hinfo,omitempty"`
		Loc        []*DnsRecord            `json:"loc,omitempty"`
		Mx         []*DnsRecord            `json:"mx,omitempty"`
		Naptr      []*DnsRecord            `json:"naptr,omitempty"`
		Ns         []*DnsRecord            `json:"ns,omitempty"`
		Nsec3      []*DnsRecord            `json:"nsec3,omitempty"`
		Nsec3param []*DnsRecord            `json:"nsec3param,omitempty"`
		Ptr        []*DnsRecord            `json:"ptr,omitempty"`
		Rp         []*DnsRecord            `json:"rp,omitempty"`
		Rrsig      []*DnsRecord            `json:"rrsig,omitempty"`
		Soa        *DnsRecord              `json:"soa,omitempty"`
		Spf        []*DnsRecord            `json:"spf,omitempty"`
		Srv        []*DnsRecord            `json:"srv,omitempty"`
		Sshfp      []*DnsRecord            `json:"sshfp,omitempty"`
		Txt        []*DnsRecord            `json:"txt,omitempty"`
		Records    map[string][]*DnsRecord `json:"-"`
	} `json:"zone"`
}

func NewZone(client *edgegrid.Client, hostname string) DnsZone {
	zone := DnsZone{client: client, Token: "new"}
	zone.Zone.Name = hostname
	return zone
}

func (zone *DnsZone) getRecordId(id string) (token string, hostname string, recordType string, name string) {
	parts := strings.Split(id, "-")
	return parts[0], parts[1], parts[2], parts[3]
}

func (zone *DnsZone) fixupCnames(record *DnsRecord) {
	if record.recordType == "CNAME" {
		names := make(map[string]string, len(zone.Zone.Records["CNAME"]))
		for _, record := range zone.Zone.Records["CNAME"] {
			names[strings.ToUpper(record.Name)] = record.Name
		}

		for recordType, records := range zone.Zone.Records {
			if recordType == "CNAME" {
				continue
			}

			newRecords := DnsRecordSet{}
			for _, record := range records {
				if _, ok := names[record.Name]; ok == false {
					newRecords = append(newRecords, record)
				} else {
					log.Printf(
						"[WARN] %s Record conflicts with CNAME \"%s\", %[1]s Record ignored.",
						recordType,
						names[strings.ToUpper(record.Name)],
					)
				}
			}
			zone.Zone.Records[recordType] = newRecords
		}
	} else if record.Name != "" {
		name := strings.ToLower(record.Name)

		newRecords := DnsRecordSet{}
		for _, cname := range zone.Zone.Records["CNAME"] {
			if strings.ToLower(cname.Name) != name {
				newRecords = append(newRecords, cname)
			} else {
				log.Printf(
					"[WARN] %s Record \"%s\" conflicts with existing CNAME \"%s\", removing CNAME",
					record.recordType,
					record.Name,
					cname.Name,
				)
			}
		}

		zone.Zone.Records["CNAME"] = newRecords
	}
}

func (zone *DnsZone) GetZone(hostname string) error {
	res, err := zone.client.Get("/config-dns/v1/zones/" + hostname)
	if err != nil {
		return err
	}

	if res.IsError() == true && res.StatusCode != 404 {
		return NewApiError(res)
	} else if res.StatusCode == 404 {
		log.Printf("[DEBUG] Zone \"%s\" not found, creating new zone.", hostname)
		newZone := NewZone(zone.client, hostname)
		zone = &newZone
		return nil
	} else {
		err = res.BodyJson(&zone)
		if err != nil {
			return err
		}

		zone.marshalRecords()

		return nil
	}
}

func (zone *DnsZone) Save() error {
	zone.unmarshalRecords()

	zone.Zone.Soa.Serial = int(time.Now().Unix())

	res, err := zone.client.PostJson("/config-dns/v1/zones/"+zone.Zone.Name, zone)
	if err != nil {
		return err
	}

	if res.IsError() == true {
		err := NewApiError(res)
		return errors.New("Unable to save record (" + err.Error() + ")")
	}

	err = zone.GetZone(zone.Zone.Name)

	if err != nil {
		return errors.New("Unable to save record (" + err.Error() + ")")
	}

	log.Printf("[INFO] Zone Saved")

	return nil
}

func (zone *DnsZone) marshalRecords() {
	zone.Zone.Records = make(map[string][]*DnsRecord)
	zone.Zone.Records["A"] = zone.Zone.A
	zone.Zone.Records["AAAA"] = zone.Zone.AAAA
	zone.Zone.Records["AFSDB"] = zone.Zone.Afsdb
	zone.Zone.Records["CNAME"] = zone.Zone.Cname
	zone.Zone.Records["DNSKEY"] = zone.Zone.Dnskey
	zone.Zone.Records["DS"] = zone.Zone.Ds
	zone.Zone.Records["HINFO"] = zone.Zone.Hinfo
	zone.Zone.Records["LOC"] = zone.Zone.Loc
	zone.Zone.Records["MX"] = zone.Zone.Mx
	zone.Zone.Records["NAPTR"] = zone.Zone.Naptr
	zone.Zone.Records["NS"] = zone.Zone.Ns
	zone.Zone.Records["NSEC3"] = zone.Zone.Nsec3
	zone.Zone.Records["NSEC3PARAM"] = zone.Zone.Nsec3param
	zone.Zone.Records["PTR"] = zone.Zone.Ptr
	zone.Zone.Records["RP"] = zone.Zone.Rp
	zone.Zone.Records["RRSIG"] = zone.Zone.Rrsig
	zone.Zone.Records["SOA"] = []*DnsRecord{zone.Zone.Soa}
	zone.Zone.Records["SPF"] = zone.Zone.Spf
	zone.Zone.Records["SRV"] = zone.Zone.Srv
	zone.Zone.Records["SSHFP"] = zone.Zone.Sshfp
	zone.Zone.Records["TXT"] = zone.Zone.Txt
}

func (zone *DnsZone) unmarshalRecords() {
	zone.Zone.A = zone.Zone.Records["A"]
	zone.Zone.AAAA = zone.Zone.Records["AAAA"]
	zone.Zone.Afsdb = zone.Zone.Records["AFSDB"]
	zone.Zone.Cname = zone.Zone.Records["CNAME"]
	zone.Zone.Dnskey = zone.Zone.Records["DNSKEY"]
	zone.Zone.Ds = zone.Zone.Records["DS"]
	zone.Zone.Hinfo = zone.Zone.Records["HINFO"]
	zone.Zone.Loc = zone.Zone.Records["LOC"]
	zone.Zone.Mx = zone.Zone.Records["MX"]
	zone.Zone.Naptr = zone.Zone.Records["NAPTR"]
	zone.Zone.Ns = zone.Zone.Records["NS"]
	zone.Zone.Nsec3 = zone.Zone.Records["NSEC3"]
	zone.Zone.Nsec3param = zone.Zone.Records["NSEC3PARAM"]
	zone.Zone.Ptr = zone.Zone.Records["PTR"]
	zone.Zone.Rp = zone.Zone.Records["RP"]
	zone.Zone.Rrsig = zone.Zone.Records["RRSIG"]
	zone.Zone.Soa = zone.Zone.Records["SOA"][0]
	zone.Zone.Spf = zone.Zone.Records["SPF"]
	zone.Zone.Srv = zone.Zone.Records["SRV"]
	zone.Zone.Sshfp = zone.Zone.Records["SSHFP"]
	zone.Zone.Txt = zone.Zone.Records["TXT"]
}

type DnsRecordSet []*DnsRecord

func (records *DnsRecordSet) unmarshalResourceData(d *schema.ResourceData) error {
	recordType := strings.ToUpper(d.Get("type").(string))
	tester := DnsRecord{recordType: recordType}
	recordSet := DnsRecordSet{}

	targets := 1
	if tester.allows("targets") {
		targets = d.Get("targets").(*schema.Set).Len()
	}

	for i := 0; i < targets; i++ {
		record := DnsRecord{recordType: recordType}

		if val, exists := d.GetOk("targets"); exists != false && !record.allows("targets") {
			return errors.New("Attribute \"targets\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Target = val.(*schema.Set).List()[i].(string)
		} else {
			record.Target = ""
		}

		if val, exists := d.GetOk("ttl"); exists != false && !record.allows("ttl") {
			return errors.New("Attribute \"ttl\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Ttl = val.(int)
		} else {
			return errors.New("Attribute \"ttl\" is required for record type " + record.recordType)
		}

		if val, exists := d.GetOk("name"); exists != false && !record.allows("name") {
			return errors.New("Attribute \"name\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Name = val.(string)
		} else {
			return errors.New("Attribute \"name\" is required for record type " + record.recordType)
		}

		if val, exists := d.GetOk("active"); exists != false && !record.allows("active") {
			return errors.New("Attribute \"active\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Active = val.(bool)
		} else {
			record.Active = true
		}

		if val, exists := d.GetOk("subtype"); exists != false && !record.allows("subtype") {
			return errors.New("Attribute \"subtype\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Subtype = val.(int)
		} else {
			record.Subtype = 0
		}

		if val, exists := d.GetOk("flags"); exists != false && !record.allows("flags") {
			return errors.New("Attribute \"flags\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Flags = val.(int)
		} else {
			record.Flags = 0
		}

		if val, exists := d.GetOk("protocol"); exists != false && !record.allows("protocol") {
			return errors.New("Attribute \"protocol\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Protocol = val.(int)
		} else {
			record.Protocol = 0
		}

		if val, exists := d.GetOk("algorithm"); exists != false && !record.allows("algorithm") {
			return errors.New("Attribute \"algorithm\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Algorithm = val.(int)
		} else {
			record.Algorithm = 0
		}

		if val, exists := d.GetOk("key"); exists != false && !record.allows("key") {
			return errors.New("Attribute \"key\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Key = val.(string)
		} else {
			record.Key = ""
		}

		if val, exists := d.GetOk("keytag"); exists != false && !record.allows("keytag") {
			return errors.New("Attribute \"keytag\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Keytag = val.(int)
		} else {
			record.Keytag = 0
		}

		if val, exists := d.GetOk("digesttype"); exists != false && !record.allows("digesttype") {
			return errors.New("Attribute \"digesttype\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.DigestType = val.(int)
		} else {
			record.DigestType = 0
		}

		if val, exists := d.GetOk("digest"); exists != false && !record.allows("digest") {
			return errors.New("Attribute \"digest\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Digest = val.(string)
		} else {
			record.Digest = ""
		}

		if val, exists := d.GetOk("hardware"); exists != false && !record.allows("hardware") {
			return errors.New("Attribute \"hardware\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Hardware = val.(string)
		} else {
			record.Hardware = ""
		}

		if val, exists := d.GetOk("software"); exists != false && !record.allows("software") {
			return errors.New("Attribute \"software\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Software = val.(string)
		} else {
			record.Software = ""
		}

		if val, exists := d.GetOk("priority"); exists != false && !record.allows("priority") {
			return errors.New("Attribute \"priority\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Priority = val.(int)
		} else {
			record.Priority = 0
		}

		if val, exists := d.GetOk("order"); exists != false && !record.allows("order") {
			return errors.New("Attribute \"order\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Order = val.(int)
		} else {
			record.Order = 0
		}

		if val, exists := d.GetOk("preference"); exists != false && !record.allows("preference") {
			return errors.New("Attribute \"preference\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Preference = val.(int)
		} else {
			record.Preference = 0
		}

		if val, exists := d.GetOk("service"); exists != false && !record.allows("service") {
			return errors.New("Attribute \"service\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Service = val.(string)
		} else {
			record.Service = ""
		}

		if val, exists := d.GetOk("regexp"); exists != false && !record.allows("regexp") {
			return errors.New("Attribute \"regexp\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Regexp = val.(string)
		} else {
			record.Regexp = ""
		}

		if val, exists := d.GetOk("replacement"); exists != false && !record.allows("replacement") {
			return errors.New("Attribute \"replacement\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Replacement = val.(string)
		} else {
			record.Replacement = ""
		}

		if val, exists := d.GetOk("iterations"); exists != false && !record.allows("iterations") {
			return errors.New("Attribute \"iterations\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Iterations = val.(int)
		} else {
			record.Iterations = 0
		}

		if val, exists := d.GetOk("salt"); exists != false && !record.allows("salt") {
			return errors.New("Attribute \"salt\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Salt = val.(string)
		} else {
			record.Salt = ""
		}

		if val, exists := d.GetOk("nexthashedownername"); exists != false && !record.allows("nexthashedownername") {
			return errors.New("Attribute \"nexthashedownername\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.NextHashedOwnerName = val.(string)
		} else {
			record.NextHashedOwnerName = ""
		}

		if val, exists := d.GetOk("typebitmaps"); exists != false && !record.allows("typebitmaps") {
			return errors.New("Attribute \"typebitmaps\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.TypeBitmaps = val.(string)
		} else {
			record.TypeBitmaps = ""
		}

		if val, exists := d.GetOk("mailbox"); exists != false && !record.allows("mailbox") {
			return errors.New("Attribute \"mailbox\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Mailbox = val.(string)
		} else {
			record.Mailbox = ""
		}

		if val, exists := d.GetOk("txt"); exists != false && !record.allows("txt") {
			return errors.New("Attribute \"txt\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Txt = val.(string)
		} else {
			record.Txt = ""
		}

		if val, exists := d.GetOk("typecovered"); exists != false && !record.allows("typecovered") {
			return errors.New("Attribute \"typecovered\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.TypeCovered = val.(string)
		} else {
			record.TypeCovered = ""
		}

		if val, exists := d.GetOk("originalttl"); exists != false && !record.allows("originalttl") {
			return errors.New("Attribute \"originalttl\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.OriginalTtl = val.(int)
		} else {
			record.OriginalTtl = 0
		}

		if val, exists := d.GetOk("expiration"); exists != false && !record.allows("expiration") {
			return errors.New("Attribute \"expiration\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Expiration = val.(string)
		} else {
			record.Expiration = ""
		}

		if val, exists := d.GetOk("inception"); exists != false && !record.allows("inception") {
			return errors.New("Attribute \"inception\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Inception = val.(string)
		} else {
			record.Inception = ""
		}

		if val, exists := d.GetOk("signer"); exists != false && !record.allows("signer") {
			return errors.New("Attribute \"signer\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Signer = val.(string)
		} else {
			record.Signer = ""
		}

		if val, exists := d.GetOk("signature"); exists != false && !record.allows("signature") {
			return errors.New("Attribute \"signature\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Signature = val.(string)
		} else {
			record.Signature = ""
		}

		if val, exists := d.GetOk("labels"); exists != false && !record.allows("labels") {
			return errors.New("Attribute \"labels\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Labels = val.(int)
		} else {
			record.Labels = 0
		}

		if val, exists := d.GetOk("originserver"); exists != false && !record.allows("originserver") {
			return errors.New("Attribute \"originserver\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Originserver = val.(string)
		} else {
			record.Originserver = ""
		}

		if val, exists := d.GetOk("contact"); exists != false && !record.allows("contact") {
			return errors.New("Attribute \"contact\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Contact = val.(string)
		} else {
			record.Contact = ""
		}

		if val, exists := d.GetOk("serial"); exists != false && !record.allows("serial") {
			return errors.New("Attribute \"serial\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Serial = val.(int)
		} else {
			record.Serial = 0
		}

		if val, exists := d.GetOk("refresh"); exists != false && !record.allows("refresh") {
			return errors.New("Attribute \"refresh\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Refresh = val.(int)
		} else {
			record.Refresh = 0
		}

		if val, exists := d.GetOk("retry"); exists != false && !record.allows("retry") {
			return errors.New("Attribute \"retry\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Retry = val.(int)
		} else {
			record.Retry = 0
		}

		if val, exists := d.GetOk("expire"); exists != false && !record.allows("expire") {
			return errors.New("Attribute \"expire\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Expire = val.(int)
		} else {
			record.Expire = 0
		}

		if val, exists := d.GetOk("minimum"); exists != false && !record.allows("minimum") {
			return errors.New("Attribute \"minimum\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Minimum = val.(int)
		} else {
			record.Minimum = 0
		}

		if val, exists := d.GetOk("weight"); exists != false && !record.allows("weight") {
			return errors.New("Attribute \"weight\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Weight = val.(uint)
		} else {
			record.Weight = 0
		}

		if val, exists := d.GetOk("port"); exists != false && !record.allows("port") {
			return errors.New("Attribute \"port\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Port = val.(int)
		} else {
			record.Port = 0
		}

		if val, exists := d.GetOk("fingerprinttype"); exists != false && !record.allows("fingerprinttype") {
			return errors.New("Attribute \"fingerprinttype\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.FingerprintType = val.(int)
		} else {
			record.FingerprintType = 0
		}

		if val, exists := d.GetOk("fingerprint"); exists != false && !record.allows("fingerprint") {
			return errors.New("Attribute \"fingerprint\" not allowed for record type " + record.recordType)
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

func (recordSet *DnsRecordSet) marshalResourceData(d *schema.ResourceData) error {
	if len(*recordSet) == 0 {
		return nil
	}

	for _, record := range *recordSet {
		if val, exists := d.GetOk("targets"); exists != false {
			val.(*schema.Set).Add(record.Target)
			d.Set("targets", val)
		} else {
			set := &schema.Set{}
			set.Add(record.Target)
			d.Set("targets", set)
		}

		d.Set("ttl", record.Ttl)
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
		d.Set("originalttl", record.OriginalTtl)
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

type DnsRecord struct {
	recordType          string
	Active              bool   `json:"active,omitempty"`
	Algorithm           int    `json:"algorithm,omitempty"`
	Contact             string `json:"contact,omitempty"`
	Digest              string `json:"digest,omitempty"`
	DigestType          int    `json:"digest_type,omitempty"`
	Expiration          string `json:"expiration,omitempty"`
	Expire              int    `json:"expire,omitempty"`
	Fingerprint         string `json:"fingerprint,omitempty"`
	FingerprintType     int    `json:"fingerprint_type,omitempty"`
	Flags               int    `json:"flags,omitempty"`
	Hardware            string `json:"hardware,omitempty"`
	Inception           string `json:"inception,omitempty"`
	Iterations          int    `json:"iterations,omitempty"`
	Key                 string `json:"key,omitempty"`
	Keytag              int    `json:"keytag,omitempty"`
	Labels              int    `json:"labels,omitempty"`
	Mailbox             string `json:"mailbox,omitempty"`
	Minimum             int    `json:"minimum,omitempty"`
	Name                string `json:"name,omitempty"`
	NextHashedOwnerName string `json:"next_hashed_owner_name,omitempty"`
	Order               int    `json:"order,omitempty"`
	OriginalTtl         int    `json:"original_ttl,omitempty"`
	Originserver        string `json:"originserver,omitempty"`
	Port                int    `json:"port,omitempty"`
	Preference          int    `json:"preference,omitempty"`
	Priority            int    `json:"priority,omitempty"`
	Protocol            int    `json:"protocol,omitempty"`
	Refresh             int    `json:"refresh,omitempty"`
	Regexp              string `json:"regexp,omitempty"`
	Replacement         string `json:"replacement,omitempty"`
	Retry               int    `json:"retry,omitempty"`
	Salt                string `json:"salt,omitempty"`
	Serial              int    `json:"serial,omitempty"`
	Service             string `json:"service,omitempty"`
	Signature           string `json:"signature,omitempty"`
	Signer              string `json:"signer,omitempty"`
	Software            string `json:"software,omitempty"`
	Subtype             int    `json:"subtype,omitempty"`
	Target              string `json:"target,omitempty"`
	Ttl                 int    `json:"ttl,omitempty"`
	Txt                 string `json:"txt,omitempty"`
	TypeBitmaps         string `json:"type_bitmaps,omitempty"`
	TypeCovered         string `json:"type_coverered,omitempty"`
	Weight              uint   `json:"weight,omitempty"`
}

func (record *DnsRecord) allows(field string) bool {
	field = strings.ToLower(field)

	field_map := map[string]map[string]struct{}{
		"active": {
			"A":          {},
			"AAAA":       {},
			"AFSDB":      {},
			"CNAME":      {},
			"DNSKEY":     {},
			"DS":         {},
			"HINFO":      {},
			"LOC":        {},
			"MX":         {},
			"NAPTR":      {},
			"NS":         {},
			"NSEC3":      {},
			"NSEC3PARAM": {},
			"PTR":        {},
			"RP":         {},
			"RRSIG":      {},
			"SPF":        {},
			"SRV":        {},
			"SSHFP":      {},
			"TXT":        {},
		},
		"algorithm": {
			"DNSKEY":     {},
			"DS":         {},
			"NSEC3":      {},
			"NSEC3PARAM": {},
			"RRSIG":      {},
			"SSHFP":      {},
		},
		"contact":         {"SOA": {}},
		"digest":          {"DS": {}},
		"digesttype":      {"DS": {}},
		"expiration":      {"RRSIG": {}},
		"expire":          {"SOA": {}},
		"fingerprint":     {"SSHFP": {}},
		"fingerprinttype": {"SSHFP": {}},
		"flags": {
			"DNSKEY":     {},
			"NAPTR":      {},
			"NSEC3":      {},
			"NSEC3PARAM": {},
		},
		"hardware":  {"HINFO": {}},
		"inception": {"RRSIG": {}},
		"iterations": {
			"NSEC3":       {},
			"NSEC3PARAMS": {},
		},
		"key": {
			"DNSKEY": {},
			"DS":     {},
		},
		"keytag":  {"RRSIG": {}},
		"labels":  {"RRSIG": {}},
		"mailbox": {"RP": {}},
		"minimum": {"SOA": {}},
		"name": {
			"A":          {},
			"AAAA":       {},
			"AFSDB":      {},
			"CNAME":      {},
			"DNSKEY":     {},
			"DS":         {},
			"HINFO":      {},
			"LOC":        {},
			"MX":         {},
			"NAPTR":      {},
			"NS":         {},
			"NSEC3":      {},
			"NSEC3PARAM": {},
			"PTR":        {},
			"RP":         {},
			"RRSIG":      {},
			"SPF":        {},
			"SRV":        {},
			"SSHFP":      {},
			"TXT":        {},
		},
		"nexthashedownername": {"NSEC3": {}},
		"order":               {"NAPTR": {}},
		"originalttl":         {"RRSIG": {}},
		"originserver":        {"SOA": {}},
		"port":                {"SRV": {}},
		"preference":          {"NAPTR": {}},
		"priority": {
			"SRV": {},
			"MX":  {},
		},
		"protocol":    {"DNSKEY": {}},
		"refresh":     {"SOA": {}},
		"regexp":      {"NAPTR": {}},
		"replacement": {"NAPTR": {}},
		"retry":       {"SOA": {}},
		"salt": {
			"NSEC3":      {},
			"NSEC3PARAM": {},
		},
		"serial":    {"SOA": {}},
		"service":   {"NAPTR": {}},
		"signature": {"RRSIG": {}},
		"signer":    {"RRSIG": {}},
		"software":  {"HINFO": {}},
		"subtype":   {"AFSDB": {}},
		"targets": {
			"A":          {},
			"AAAA":       {},
			"AFSDB":      {},
			"CNAME":      {},
			"DNSKEY":     {},
			"DS":         {},
			"HINFO":      {},
			"LOC":        {},
			"MX":         {},
			"NAPTR":      {},
			"NS":         {},
			"NSEC3":      {},
			"NSEC3PARAM": {},
			"PTR":        {},
			"RP":         {},
			"RRSIG":      {},
			"SOA":        {},
			"SPF":        {},
			"SRV":        {},
			"SSHFP":      {},
			"TXT":        {},
		},
		"ttl": {
			"A":     {},
			"AAAA":  {},
			"AFSDB": {},
			"CNAME": {},
			"LOC":   {},
			"MX":    {},
			"NS":    {},
			"PTR":   {},
			"SPF":   {},
			"SRV":   {},
			"TXT":   {},
		},
		"txt":         {"RP": {}},
		"typebitmaps": {"NSEC3": {}},
		"typecovered": {"RRSIG": {}},
		"weight":      {"SRV": {}},
	}

	_, ok := field_map[field][strings.ToUpper(record.recordType)]

	return ok
}

func importRecord(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	config := meta.(*Config)

	zone := config.ClientFastDns
	_, hostname, recordType, name := zone.getRecordId("_." + d.Id())
	zone.GetZone(hostname)

	var exists bool
	for _, record := range zone.Zone.Records[recordType] {
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

type ApiError struct {
	error
	Type        string `json:"type"`
	Title       string `json:"title"`
	Status      int    `json:"status"`
	Detail      string `json:"detail"`
	Instance    string `json:"instance"`
	Method      string `json:"method"`
	ServerIP    string `json:"serverIp"`
	ClientIP    string `json:"clientIp"`
	requestId   string `json:"requestId"`
	requestTime string `json:"requestTime"`
}

func (error ApiError) Error() string {
	return strings.TrimSpace(fmt.Sprintf("API Error: %d %s %s", error.Status, error.Title, error.Detail))
}

func NewApiError(response *edgegrid.Response) ApiError {
	error := ApiError{}
	if err := response.BodyJson(&error); err != nil {
		error.Status = response.StatusCode
		error.Title = response.Status
	}

	return error
}
