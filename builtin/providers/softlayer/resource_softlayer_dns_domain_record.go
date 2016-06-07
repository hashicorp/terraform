package softlayer

import (
	"fmt"
	"log"
	"strconv"

	datatypes "github.com/TheWeatherCompany/softlayer-go/data_types"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceSoftLayerDnsDomainResourceRecord() *schema.Resource {
	return &schema.Resource{
		Exists: resourceSoftLayerDnsDomainResourceRecordExists,
		Create: resourceSoftLayerDnsDomainResourceRecordCreate,
		Read:   resourceSoftLayerDnsDomainResourceRecordRead,
		Update: resourceSoftLayerDnsDomainResourceRecordUpdate,
		Delete: resourceSoftLayerDnsDomainResourceRecordDelete,
		Schema: map[string]*schema.Schema{
			"record_data": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"domain_id": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"expire": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"host": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"minimum_ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"mx_priority": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"refresh": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"contact_email": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"retry": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"record_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"service": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"priority": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"weight": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}
}

//  Creates DNS Domain Resource Record
//  https://sldn.softlayer.com/reference/services/SoftLayer_Dns_Domain_ResourceRecord/createObject
func resourceSoftLayerDnsDomainResourceRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).dnsDomainResourceRecordService

	if client == nil {
		return fmt.Errorf("The client was nil.")
	}

	opts := datatypes.SoftLayer_Dns_Domain_ResourceRecord_Template{
		Data:              d.Get("record_data").(string),
		DomainId:          d.Get("domain_id").(int),
		Expire:            d.Get("expire").(int),
		Host:              d.Get("host").(string),
		Minimum:           d.Get("minimum_ttl").(int),
		MxPriority:        d.Get("mx_priority").(int),
		Refresh:           d.Get("refresh").(int),
		ResponsiblePerson: d.Get("contact_email").(string),
		Retry:             d.Get("retry").(int),
		Ttl:               d.Get("ttl").(int),
		Type:              d.Get("record_type").(string),
		Service:           d.Get("service").(string),
		Protocol:          d.Get("protocol").(string),
		Priority:          d.Get("priority").(int),
		Weight:            d.Get("weight").(int),
		Port:              d.Get("port").(int),
	}

	log.Printf("[INFO] Creating DNS Resource Record for '%d' dns domain", d.Get("id"))

	record, err := client.CreateObject(opts)

	if err != nil {
		return fmt.Errorf("Error creating DNS Resource Record: %s", err)
	}

	d.SetId(fmt.Sprintf("%d", record.Id))

	log.Printf("[INFO] Dns Resource Record ID: %s", d.Id())

	return resourceSoftLayerDnsDomainResourceRecordRead(d, meta)
}

//  Reads DNS Domain Resource Record from SL system
//  https://sldn.softlayer.com/reference/services/SoftLayer_Dns_Domain_ResourceRecord/getObject
func resourceSoftLayerDnsDomainResourceRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).dnsDomainResourceRecordService

	if client == nil {
		return fmt.Errorf("The client was nil.")
	}

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}
	result, err := client.GetObject(id)
	if err != nil {
		return fmt.Errorf("Error retrieving DNS Resource Record: %s", err)
	}

	d.Set("data", result.Data)
	d.Set("domainId", result.DomainId)
	d.Set("expire", result.Expire)
	d.Set("host", result.Host)
	d.Set("id", result.Id)
	d.Set("minimum", result.Minimum)
	d.Set("mxPriority", result.MxPriority)
	d.Set("refresh", result.Refresh)
	d.Set("responsiblePerson", result.ResponsiblePerson)
	d.Set("retry", result.Retry)
	d.Set("ttl", result.Ttl)
	d.Set("type", result.Type)
	d.Set("service", result.Service)
	d.Set("protocol", result.Protocol)
	d.Set("port", result.Port)
	d.Set("priority", result.Priority)
	d.Set("weight", result.Weight)

	return nil
}

//  Updates DNS Domain Resource Record in SL system
//  https://sldn.softlayer.com/reference/services/SoftLayer_Dns_Domain_ResourceRecord/editObject
func resourceSoftLayerDnsDomainResourceRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).dnsDomainResourceRecordService

	if client == nil {
		return fmt.Errorf("The client was nil.")
	}

	recordId, _ := strconv.Atoi(d.Id())

	record, err := client.GetObject(recordId)
	if err != nil {
		return fmt.Errorf("Error retrieving DNS Resource Record: %s", err)
	}

	if data, ok := d.GetOk("record_data"); ok {
		record.Data = data.(string)
	}
	if domain_id, ok := d.GetOk("domain_id"); ok {
		record.DomainId = domain_id.(int)
	}
	if expire, ok := d.GetOk("expire"); ok {
		record.Expire = expire.(int)
	}
	if host, ok := d.GetOk("host"); ok {
		record.Host = host.(string)
	}
	if minimum_ttl, ok := d.GetOk("minimum_ttl"); ok {
		record.Minimum = minimum_ttl.(int)
	}
	if mx_priority, ok := d.GetOk("mx_priority"); ok {
		record.MxPriority = mx_priority.(int)
	}
	if refresh, ok := d.GetOk("refresh"); ok {
		record.Refresh = refresh.(int)
	}
	if contact_email, ok := d.GetOk("contact_email"); ok {
		record.ResponsiblePerson = contact_email.(string)
	}
	if retry, ok := d.GetOk("retry"); ok {
		record.Retry = retry.(int)
	}
	if ttl, ok := d.GetOk("ttl"); ok {
		record.Ttl = ttl.(int)
	}
	if record_type, ok := d.GetOk("record_type"); ok {
		record.Type = record_type.(string)
	}
	if service, ok := d.GetOk("service"); ok {
		record.Service = service.(string)
	}
	if priority, ok := d.GetOk("priority"); ok {
		record.Priority = priority.(int)
	}
	if protocol, ok := d.GetOk("protocol"); ok {
		record.Protocol = protocol.(string)
	}
	if port, ok := d.GetOk("port"); ok {
		record.Port = port.(int)
	}
	if weight, ok := d.GetOk("weight"); ok {
		record.Weight = weight.(int)
	}

	_, err = client.EditObject(recordId, record)
	if err != nil {
		return fmt.Errorf("Error editing DNS Resoource Record: %s", err)
	}
	return nil
}

//  Deletes DNS Domain Resource Record in SL system
//  https://sldn.softlayer.com/reference/services/SoftLayer_Dns_Domain_ResourceRecord/deleteObject
func resourceSoftLayerDnsDomainResourceRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client).dnsDomainResourceRecordService

	if client == nil {
		return fmt.Errorf("The client was nil.")
	}

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	_, err = client.DeleteObject(id)

	if err != nil {
		return fmt.Errorf("Error deleting DNS Resource Record: %s", err)
	}

	return nil
}

// Exists function is called by refresh
// if the entity is absent - it is deleted from the .tfstate file
func resourceSoftLayerDnsDomainResourceRecordExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*Client).dnsDomainResourceRecordService

	if client == nil {
		return false, fmt.Errorf("The client was nil.")
	}

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return false, fmt.Errorf("Not a valid ID, must be an integer: %s", err)
	}

	record, err := client.GetObject(id)

	return record.Id == id && err == nil, nil
}
