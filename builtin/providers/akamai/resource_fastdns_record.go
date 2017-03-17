package akamai

import (
	"errors"
	"github.com/akamai-open/AkamaiOPEN-edgegrid-golang"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func resourceFastDnsRecord() *schema.Resource {
	return &schema.Resource{
		Create:   resourceFastDnsRecordCreate,
		Read:     resourceFastDnsRecordRead,
		Update:   resourceFastDnsRecordCreate,
		Delete:   resourceFastDnsRecordDelete,
		Importer: nil,
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
	config := meta.(Config)

	zone := config.ClientFastDns
	zone.GetZone(d.Get("hostname").(string))

	recordType := d.Get("type").(string)

	records := DnsRecordSet{}
	error := records.unmarshalResourceData(d)

	if error != nil {
		return error
	}

	var name string
	if strings.ToUpper(recordType) != "SOA" {
		zone.Zone.Records[recordType] = append(zone.Zone.Records[recordType], records[0])
		name = "soa"
	} else {
		name = d.Get("name").(string)

		// Only overwrite records with the same name
		for _, v := range zone.Zone.Records[recordType] {
			if v.Name != name {
				records = append(records, v)
			}
		}

		zone.Zone.Records[recordType] = records
	}

	error = zone.Save()
	if error != nil {
		return error
	}

	d.SetId(zone.Token + "-" + recordType + "-" + name)

	return nil
}

func resourceFastDnsRecordRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(Config)

	zone := config.ClientFastDns
	zone.GetZone(d.Get("hostname").(string))

	token, recordType, name := zone.getRecordId(d)

	if zone.Token != token {
		return errors.New("Resource has been modified, aborting")
	}

	var record *DnsRecord
	for _, record := range zone.Zone.Records[recordType] {
		if record.Name == name {
			break
		}
	}

	record.marshalResourceData(d)

	return nil
}

func resourceFastDnsRecordDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(Config)

	zone := config.ClientFastDns
	zone.GetZone(d.Get("hostname").(string))

	recordType := d.Get("type").(string)

	records := DnsRecordSet{}
	error := records.unmarshalResourceData(d)

	if error != nil {
		return error
	}

	if strings.ToUpper(recordType) != "SOA" {
		zone.Zone.Records[recordType] = nil
	} else {
		name := d.Get("name").(string)

		for _, v := range zone.Zone.Records[recordType] {
			if v.Name != name {
				records = append(records, v)
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

func (zone *DnsZone) getRecordId(d *schema.ResourceData) (token string, recordType string, name string) {
	parts := strings.Split(d.Id(), "-")
	return parts[0], parts[1], parts[2]
}

type DnsZone struct {
	client *edgegrid.Client
	Token  string `json:"token"`
	Zone   struct {
		Version    float64      `json:"version,omitempty"`
		Name       string       `json:"name"`
		Publisher  string       `json:"publisher,omitempty"`
		Instance   string       `json:"instance,omitempty"`
		Id         int          `json:id,omitempty`
		Time       int          `json:"time,omitempty"`
		A          []*DnsRecord `json:"a,omitempty"`
		AAAA       []*DnsRecord `json:"aaaa,omitempty"`
		Afsdb      []*DnsRecord `json:"afsdb,omitempty"`
		Cname      []*DnsRecord `json:"cname,omitempty"`
		Dnskey     []*DnsRecord `json:"dnskey,omitempty"`
		Ds         []*DnsRecord `json:"ds,omitempty"`
		Hinfo      []*DnsRecord `json:"hinfo,omitempty"`
		Loc        []*DnsRecord `json:"loc,omitempty"`
		Mx         []*DnsRecord `json:"mx,omitempty"`
		Naptr      []*DnsRecord `json:"naptr,omitempty"`
		Ns         []*DnsRecord `json:"ns,omitempty"`
		Nsec3      []*DnsRecord `json:"nsec3,omitempty"`
		Nsec3param []*DnsRecord `json:"nsec3param,omitempty"`
		Ptr        []*DnsRecord `json:"ptr,omitempty"`
		Rp         []*DnsRecord `json:"rp,omitempty"`
		Rrsig      []*DnsRecord `json:"rrsig,omitempty"`
		Soa        *DnsRecord   `json:"soa,omitempty"`
		Spf        []*DnsRecord `json:"spf,omitempty"`
		Srv        []*DnsRecord `json:"srv,omitempty"`
		Sshfp      []*DnsRecord `json:"sshfp,omitempty"`
		Txt        []*DnsRecord `json:"txt,omitempty"`
		Records    map[string][]*DnsRecord
	} `json:"zone"`
}

func NewZone(client *edgegrid.Client) DnsZone {
	return DnsZone{client: client, Token: "new"}
}

func (zone *DnsZone) GetZone(hostname string) error {
	res, err := zone.client.Get("/config-dns/v1/zones/" + hostname)
	if err != nil {
		return err
	}

	err = res.BodyJson(&zone)
	if err != nil {
		return err
	}

	zone.marshalRecords()

	return nil
}

func (zone *DnsZone) Save() error {
	zone.unmarshalRecords()

	zone.Zone.Soa.Serial++

	ret, err := zone.client.PostJson("/config-dns/v1/zones/"+zone.Zone.Name, zone)
	if err != nil {
		return err
	}

	if err = ret.BodyJson(&zone); err != nil {
		return errors.New("Unable to create record (" + err.Error() + ")")
	}

	return nil
}

func (zone *DnsZone) marshalRecords() {
	zone.Zone.Records["A"] = zone.Zone.A
	zone.Zone.Records["AAAA"] = zone.Zone.AAAA
	zone.Zone.Records["Afsdb"] = zone.Zone.Afsdb
	zone.Zone.Records["Cname"] = zone.Zone.Cname
	zone.Zone.Records["Dnskey"] = zone.Zone.Dnskey
	zone.Zone.Records["Ds"] = zone.Zone.Ds
	zone.Zone.Records["Hinfo"] = zone.Zone.Hinfo
	zone.Zone.Records["Loc"] = zone.Zone.Loc
	zone.Zone.Records["Mx"] = zone.Zone.Mx
	zone.Zone.Records["Naptr"] = zone.Zone.Naptr
	zone.Zone.Records["Ns"] = zone.Zone.Ns
	zone.Zone.Records["Nsec3"] = zone.Zone.Nsec3
	zone.Zone.Records["Nsec3param"] = zone.Zone.Nsec3param
	zone.Zone.Records["Ptr"] = zone.Zone.Ptr
	zone.Zone.Records["Rp"] = zone.Zone.Rp
	zone.Zone.Records["Rrsig"] = zone.Zone.Rrsig
	zone.Zone.Records["Soa"] = []*DnsRecord{zone.Zone.Soa}
	zone.Zone.Records["Spf"] = zone.Zone.Spf
	zone.Zone.Records["Srv"] = zone.Zone.Srv
	zone.Zone.Records["Sshfp"] = zone.Zone.Sshfp
	zone.Zone.Records["Txt"] = zone.Zone.Txt
}

func (zone *DnsZone) unmarshalRecords() {
	zone.Zone.A = zone.Zone.Records["A"]
	zone.Zone.AAAA = zone.Zone.Records["AAAA"]
	zone.Zone.Afsdb = zone.Zone.Records["Afsdb"]
	zone.Zone.Cname = zone.Zone.Records["Cname"]
	zone.Zone.Dnskey = zone.Zone.Records["Dnskey"]
	zone.Zone.Ds = zone.Zone.Records["Ds"]
	zone.Zone.Hinfo = zone.Zone.Records["Hinfo"]
	zone.Zone.Loc = zone.Zone.Records["Loc"]
	zone.Zone.Mx = zone.Zone.Records["Mx"]
	zone.Zone.Naptr = zone.Zone.Records["Naptr"]
	zone.Zone.Ns = zone.Zone.Records["Ns"]
	zone.Zone.Nsec3 = zone.Zone.Records["Nsec3"]
	zone.Zone.Nsec3param = zone.Zone.Records["Nsec3param"]
	zone.Zone.Ptr = zone.Zone.Records["Ptr"]
	zone.Zone.Rp = zone.Zone.Records["Rp"]
	zone.Zone.Rrsig = zone.Zone.Records["Rrsig"]
	zone.Zone.Soa = zone.Zone.Records["Soa"][0]
	zone.Zone.Spf = zone.Zone.Records["Spf"]
	zone.Zone.Srv = zone.Zone.Records["Srv"]
	zone.Zone.Sshfp = zone.Zone.Records["Sshfp"]
	zone.Zone.Txt = zone.Zone.Records["Txt"]
}

type DnsRecordSet []*DnsRecord

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
	Name                string `json:"name"`
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

func (records *DnsRecordSet) unmarshalResourceData(d *schema.ResourceData) error {
	recordType := d.Get("type").(string)
	record := DnsRecord{recordType: recordType}

	var targets int

	if record.allows("target") {
		targets = d.Get("target").(*schema.Set).Len()
	} else {
		targets = 1
	}

	for i := 0; i <= targets; i++ {
		if val, exists := d.GetOk("ttl"); exists != false && !record.allows("ttl") {
			return errors.New("Attribute \"ttl\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Ttl = val.(int)
		} else {
			record.Ttl = 0
		}

		if val, exists := d.GetOk("name"); exists != false && !record.allows("name") {
			return errors.New("Attribute \"name\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Name = val.(string)
		} else {
			record.Name = ""
		}

		if val, exists := d.GetOk("active"); exists != false && !record.allows("active") {
			return errors.New("Attribute \"active\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Active = val.(bool)
		} else {
			record.Active = false
		}

		if val, exists := d.GetOk("target"); exists != false && !record.allows("target") {
			return errors.New("Attribute \"target\" not allowed for record type " + record.recordType)
		} else if exists != false {
			record.Target = val.(string)
		} else {
			record.Target = ""
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
		*records = append(*records, &record)
	}

	return nil
}

func (record *DnsRecord) marshalResourceData(d *schema.ResourceData) {
	if record.Ttl != 0 {
		d.Set("ttl", record.Ttl)
	}

	if record.Name != "" {
		d.Set("name", record.Name)
	}

	if record.Active != false {
		d.Set("active", record.Active)
	}

	if record.Target != "" {
		d.Set("target", record.Target)
	}

	if record.Subtype != 0 {
		d.Set("subtype", record.Subtype)
	}

	if record.Flags != 0 {
		d.Set("flags", record.Flags)
	}

	if record.Protocol != 0 {
		d.Set("protocol", record.Protocol)
	}

	if record.Algorithm != 0 {
		d.Set("algorithm", record.Algorithm)
	}

	if record.Key != "" {
		d.Set("key", record.Key)
	}

	if record.Keytag != 0 {
		d.Set("keytag", record.Keytag)
	}

	if record.DigestType != 0 {
		d.Set("digesttype", record.DigestType)
	}

	if record.Digest != "" {
		d.Set("digest", record.Digest)
	}

	if record.Hardware != "" {
		d.Set("hardware", record.Hardware)
	}

	if record.Software != "" {
		d.Set("software", record.Software)
	}

	if record.Priority != 0 {
		d.Set("priority", record.Priority)
	}

	if record.Order != 0 {
		d.Set("order", record.Order)
	}

	if record.Preference != 0 {
		d.Set("preference", record.Preference)
	}

	if record.Service != "" {
		d.Set("service", record.Service)
	}

	if record.Regexp != "" {
		d.Set("regexp", record.Regexp)
	}

	if record.Replacement != "" {
		d.Set("replacement", record.Replacement)
	}

	if record.Iterations != 0 {
		d.Set("iterations", record.Iterations)
	}

	if record.Salt != "" {
		d.Set("salt", record.Salt)
	}

	if record.NextHashedOwnerName != "" {
		d.Set("nexthashedownername", record.NextHashedOwnerName)
	}

	if record.TypeBitmaps != "" {
		d.Set("typebitmaps", record.TypeBitmaps)
	}

	if record.Mailbox != "" {
		d.Set("mailbox", record.Mailbox)
	}

	if record.Txt != "" {
		d.Set("txt", record.Txt)
	}

	if record.TypeCovered != "" {
		d.Set("typecovered", record.TypeCovered)
	}

	if record.OriginalTtl != 0 {
		d.Set("originalttl", record.OriginalTtl)
	}

	if record.Expiration != "" {
		d.Set("expiration", record.Expiration)
	}

	if record.Inception != "" {
		d.Set("inception", record.Inception)
	}

	if record.Signer != "" {
		d.Set("signer", record.Signer)
	}

	if record.Signature != "" {
		d.Set("signature", record.Signature)
	}

	if record.Labels != 0 {
		d.Set("labels", record.Labels)
	}

	if record.Originserver != "" {
		d.Set("originserver", record.Originserver)
	}

	if record.Contact != "" {
		d.Set("contact", record.Contact)
	}

	if record.Serial != 0 {
		d.Set("serial", record.Serial)
	}

	if record.Refresh != 0 {
		d.Set("refresh", record.Refresh)
	}

	if record.Retry != 0 {
		d.Set("retry", record.Retry)
	}

	if record.Expire != 0 {
		d.Set("expire", record.Expire)
	}

	if record.Minimum != 0 {
		d.Set("minimum", record.Minimum)
	}

	if record.Weight != 0 {
		d.Set("weight", record.Weight)
	}

	if record.Port != 0 {
		d.Set("port", record.Port)
	}

	if record.FingerprintType != 0 {
		d.Set("fingerprinttype", record.FingerprintType)
	}

	if record.Fingerprint != "" {
		d.Set("fingerprint", record.Fingerprint)
	}
}

func (record *DnsRecord) allows(field string) bool {
	field = strings.ToLower(field)

	field_map := map[string]map[string]struct{}{
		"active": {
			"a":          {},
			"aaaa":       {},
			"afsdb":      {},
			"cname":      {},
			"dnskey":     {},
			"ds":         {},
			"hinfo":      {},
			"loc":        {},
			"mx":         {},
			"naptr":      {},
			"ns":         {},
			"nsec3":      {},
			"nsec3param": {},
			"ptr":        {},
			"rp":         {},
			"rrsig":      {},
			"spf":        {},
			"srv":        {},
			"sshfp":      {},
			"txt":        {},
		},
		"algorithm": {
			"dnskey":     {},
			"ds":         {},
			"nsec3":      {},
			"nsec3param": {},
			"rrsig":      {},
			"sshfp":      {},
		},
		"contact":         {"soa": {}},
		"digest":          {"ds": {}},
		"digesttype":      {"ds": {}},
		"expiration":      {"rrsig": {}},
		"expire":          {"soa": {}},
		"fingerprint":     {"sshfp": {}},
		"fingerprinttype": {"sshfp": {}},
		"flags": {
			"dnskey":     {},
			"naptr":      {},
			"nsec3":      {},
			"nsec3param": {},
		},
		"hardware":  {"hinfo": {}},
		"inception": {"rrsig": {}},
		"iterations": {
			"nsec3":       {},
			"nsec3params": {},
		},
		"key": {
			"dnskey": {},
			"ds":     {},
		},
		"keytag":  {"rrsig": {}},
		"labels":  {"rrsig": {}},
		"mailbox": {"rp": {}},
		"minimum": {"soa": {}},
		"name": {
			"a":          {},
			"aaaa":       {},
			"afsdb":      {},
			"cname":      {},
			"dnskey":     {},
			"ds":         {},
			"hinfo":      {},
			"loc":        {},
			"mx":         {},
			"naptr":      {},
			"ns":         {},
			"nsec3":      {},
			"nsec3param": {},
			"ptr":        {},
			"rp":         {},
			"rrsig":      {},
			"spf":        {},
			"srv":        {},
			"sshfp":      {},
			"txt":        {},
		},
		"nexthashedownername": {"nsec3": {}},
		"order":               {"naptr": {}},
		"originalttl":         {"rrsig": {}},
		"originserver":        {"soa": {}},
		"port":                {"srv": {}},
		"preference":          {"naptr": {}},
		"priority": {
			"srv": {},
			"mx":  {},
		},
		"protocol":    {"dnskey": {}},
		"refresh":     {"soa": {}},
		"regexp":      {"naptr": {}},
		"replacement": {"naptr": {}},
		"retry":       {"soa": {}},
		"salt": {
			"nsec3":      {},
			"nsec3param": {},
		},
		"serial":    {"soa": {}},
		"service":   {"naptr": {}},
		"signature": {"rrsig": {}},
		"signer":    {"rrsig": {}},
		"software":  {"hinfo": {}},
		"subtype":   {"afsdb": {}},
		"target": {
			"a":          {},
			"aaaa":       {},
			"afsdb":      {},
			"cname":      {},
			"dnskey":     {},
			"ds":         {},
			"hinfo":      {},
			"loc":        {},
			"mx":         {},
			"naptr":      {},
			"ns":         {},
			"nsec3":      {},
			"nsec3param": {},
			"ptr":        {},
			"rp":         {},
			"rrsig":      {},
			"soa":        {},
			"spf":        {},
			"srv":        {},
			"sshfp":      {},
			"txt":        {},
		},
		"ttl": {
			"a":     {},
			"aaaa":  {},
			"afsdb": {},
			"cname": {},
			"loc":   {},
			"mx":    {},
			"ns":    {},
			"ptr":   {},
			"spf":   {},
			"srv":   {},
			"txt":   {},
		},
		"txt":         {"rp": {}},
		"typebitmaps": {"nsec3": {}},
		"typecovered": {"rrsig": {}},
		"weight":      {"srv": {}},
	}

	_, ok := field_map[field][record.recordType]

	return ok
}
