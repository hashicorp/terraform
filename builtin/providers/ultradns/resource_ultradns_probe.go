package ultradns

import (
	"encoding/json"
	"fmt"
	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

/*
func schemaTransaction() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: false,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"method": &schema.Schema{
					Type:     schema.TypeString,
					Optional: false,
				},
				"url": &schema.Schema{
					Type:     schema.TypeString,
					Optional: false,
				},
				"transmittedData": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"followRedirects": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"limits": &schema.Schema{
					Type:     schema.TypeSet,
					Optional: false,
					Elem:     resourceLimits(),
				},
			},
		},
	}
}

/*
func schemaPingProbeLimits() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"lossPercent": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Elem:     resourceLimits(),
				},
				"total": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Elem:     resourceLimits(),
				},
				"average": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Elem:     resourceLimits(),
				},
				"run": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Elem:     resourceLimits(),
				},
				"avgRun": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Elem:     resourceLimits(),
				},
			},
		},
	}
}
func resourcePingProbeLimits() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"lossPercent": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     resourceLimits(),
			},
			"total": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     resourceLimits(),
			},
			"average": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     resourceLimits(),
			},
			"run": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     resourceLimits(),
			},
			"avgRun": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     resourceLimits(),
			},
		},
	}
}
*/
func resourceLimits() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"warning": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"critical": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"fail": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
		},
	}
}

/*
func schemaHTTPProbe() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"transactions": schemaTransaction(),
				//"totalLimits":  schemaLimits(),
			},
		},
	}
}
func schemaSMTPProbe() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"port": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},
				"limits": &schema.Schema{
					Type:     schema.TypeMap,
					Required: true,
					Elem:     resourceLimits(),
				},
			},
		},
	}
}
func schemaSMTPSENDProbe() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"port": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},
				"from": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				"to": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},

				"message": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				"limits": &schema.Schema{
					Type:     schema.TypeMap,
					Required: true,
					Elem:     resourceLimits(),
				},
			},
		},
	}
}

func schemaTCPProbe() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"port": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},
				"controlIP": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"limits": &schema.Schema{
					Type:     schema.TypeMap,
					Required: true,
					Elem:     resourceLimits(),
				},
			},
		},
	}
}

func schemaFTPProbe() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"port": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},
				"passiveMode": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"username": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"password": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"path": &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				"limits": &schema.Schema{
					Type:     schema.TypeMap,
					Required: true,
					Elem:     resourceLimits(),
				},
			},
		},
	}
}
*/
func schemaPingProbe() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"packets": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"packetSize": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"limits": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     resourceLimits(),
			},
		},
	}

}

/*
func schemaDNSProbe() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"port": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},
				"tcpOnly": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
				"type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"ownerName": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
				"limits": &schema.Schema{
					Type:     schema.TypeMap,
					Optional: true,
					Elem:     resourceLimits(),
				},
			},
		},
	}
}
*/
func resourceUltraDNSProbe() *schema.Resource {
	return &schema.Resource{
		Create: resourceUltraDNSProbeCreate,
		Read:   resourceUltraDNSProbeRead,
		Update: resourceUltraDNSProbeUpdate,
		Delete: resourceUltraDNSProbeDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"ownerName": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ownerType": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"zoneName": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"poolRecord": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"interval": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"agents": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			}, /*
				"http_probe": &schema.Schema{
					Type:          schema.TypeSet,
					Optional:      true,
					ConflictsWith: []string{"dns_probe", "ping_probe", "smtp_probe", "smtpsend_probe"},
					Elem:          schemaHTTPProbe(),
				},
				"dns_probe": &schema.Schema{
					Type:          schema.TypeSet,
					Optional:      true,
					ConflictsWith: []string{"http_probe", "ping_probe", "smtp_probe", "smtpsend_probe"},
					Elem:          schemaDNSProbe(),
				},
				"smtpsend_probe": &schema.Schema{
					Type:          schema.TypeSet,
					Optional:      true,
					ConflictsWith: []string{"http_probe", "ping_probe", "smtp_probe", "dns_probe"},
					Elem:          schemaDNSProbe(),
				},
				"smtp_probe": &schema.Schema{
					Type:          schema.TypeSet,
					Optional:      true,
					ConflictsWith: []string{"http_probe", "ping_probe", "dns_probe", "smtpsend_probe"},
					Elem:          schemaDNSProbe(),
				},*/
			"ping_probe": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				//ConflictsWith: []string{"http_probe", "smtp_probe", "dns_probe", "smtpsend_probe"},
				Elem: schemaPingProbe(),
			},
		},
	}

}

func resourceUltraDNSProbeCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)
	newProbe := udnssdk.ProbeInfoDTO{
		ID:         d.Id(),
		PoolRecord: d.Get("poolRecord").(string),
		ProbeType:  d.Get("type").(string),
		Interval:   d.Get("interval").(string),
		Threshold:  d.Get("threshold").(int),
	}
	typeToTypeMap := map[string]string{
		"HTTP":      "http_probe",
		"PING":      "ping_probe",
		"FTP":       "ftp_probe",
		"SMTP":      "smtp_probe",
		"SMTP_SEND": "smtpsend_probe",
		"DNS":       "dns_probe",
	}
	// Transmute format of 'agents' field
	oldagents, ok := d.GetOk("agents")
	if !ok {
		return fmt.Errorf("Can not get agents for probetype %s.", newProbe.ProbeType)
	}
	var newagents []string
	for _, e := range oldagents.([]interface{}) {
		newagents = append(newagents, e.(string))
	}
	newProbe.Agents = newagents
	// Find probe type
	probeset, ok := d.GetOk(typeToTypeMap[newProbe.ProbeType])
	if !ok {
		return fmt.Errorf("Can not get appropriate details for probetype %s.", newProbe.ProbeType)
	}
	var probedetails map[string]interface{}
	probedetails = probeset.(*schema.Set).List()[0].(map[string]interface{})
	// Convert limits from flattened set format to mapping.
	newlimits := map[string]interface{}{}
	for _, el := range probedetails["limits"].([]interface{}) {
		element := el.(map[string]interface{})
		newlimits[element["name"].(string)] = map[string]interface{}{"warning": element["warning"], "critical": element["critical"], "fail": element["fail"]}
	}
	probedetails["limits"] = newlimits
	newdetails := &udnssdk.ProbeDetailsDTO{
		Detail: probedetails,
	}
	newProbe.Details = newdetails
	log.Printf("[DEBUG] UltraDNS Probe create configuration: %#v", newProbe)
	name := d.Get("ownerName").(string)
	//typ := newProbe.ProbeType
	typ := d.Get("ownerType").(string)
	zone := d.Get("zoneName").(string)
	guid, locale, _, err := client.SBTCService.CreateProbe(name, typ, zone, newProbe)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to create UltraDNS Probe: %s", err)
	}
	d.Set("uri", locale)
	d.SetId(guid)
	log.Printf("[INFO] Probe ID: %s", d.Id())
	return resourceUltraDNSProbeRead(d, meta)
}

func resourceUltraDNSProbeRead(d *schema.ResourceData, meta interface{}) error {

	typeToTypeMap := map[string]string{
		"HTTP":      "http_probe",
		"PING":      "ping_probe",
		"FTP":       "ftp_probe",
		"SMTP":      "smtp_probe",
		"SMTP_SEND": "smtpsend_probe",
		"DNS":       "dns_probe",
	}
	log.Printf("[DEBUG] Entering resourceUltraDNSProbeRead\n")
	client := meta.(*udnssdk.Client)
	probe, _, err := client.SBTCService.GetProbe(d.Get("name").(string), d.Get("type").(string), d.Get("zone").(string), d.Id())
	if err != nil {
		uderr, ok := err.(*udnssdk.ErrorResponseList)
		if ok {
			for _, r := range uderr.Responses {
				// 70002 means Probes Not Found
				if r.ErrorCode == 70002 {
					d.SetId("")
					return nil
				} else {
					return fmt.Errorf("[ERROR] Couldn't find UltraDNS Probe: %s", err)
				}
			}
		} else {
			return fmt.Errorf("[ERROR] Couldn't find UltraDNS Probe: %s", err)
		}
	}
	err = probe.Details.Populate(probe.ProbeType)
	if err != nil {
		return fmt.Errorf("[DEBUG] Could not populate probe details: %#v", err)
	}
	err = d.Set("poolRecord", probe.PoolRecord)
	if err != nil {
		return fmt.Errorf("[DEBUG] Error setting poolRecord: %#v", err)
	}
	err = d.Set("interval", probe.Interval)
	if err != nil {
		return fmt.Errorf("[DEBUG] Error setting interval: %#v", err)
	}
	err = d.Set("type", probe.ProbeType)
	if err != nil {
		return fmt.Errorf("[DEBUG] Error setting type: %#v", err)
	}

	err = d.Set("agents", probe.Agents)
	if err != nil {
		return fmt.Errorf("[DEBUG] Error setting agents: %#v", err)
	}
	err = d.Set("threshold", probe.Threshold)
	if err != nil {
		return fmt.Errorf("[DEBUG] Error setting threshold: %#v", err)
	}
	d.SetId(probe.ID)

	if probe.Details != nil {

		var dp map[string]interface{}
		err = json.Unmarshal(probe.Details.GetData(), &dp)
		if err != nil {
			return err
		}

		err = d.Set(typeToTypeMap[probe.ProbeType], dp)
		if err != nil {
			return fmt.Errorf("[DEBUG] Error setting details: %#v", err)
		}

	}
	return nil
}

func resourceUltraDNSProbeUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	typeToTypeMap := map[string]string{
		"HTTP":      "http_probe",
		"PING":      "ping_probe",
		"FTP":       "ftp_probe",
		"SMTP":      "smtp_probe",
		"SMTP_SEND": "smtpsend_probe",
		"DNS":       "dns_probe",
	}
	updateProbe := &udnssdk.ProbeInfoDTO{}
	newProbe := udnssdk.ProbeInfoDTO{
		ID:         d.Id(),
		PoolRecord: d.Get("poolRecord").(string),
		ProbeType:  d.Get("type").(string),
		Interval:   d.Get("interval").(string),
		Agents:     d.Get("agents").([]string),
		Threshold:  d.Get("threshold").(int),
	}

	deets2, ok := d.GetOk(typeToTypeMap[newProbe.ProbeType])
	if !ok {
		return fmt.Errorf("Can not get appropriate details for probetype %s.", newProbe.ProbeType)
	}
	_, err := json.Marshal(deets2)
	if err != nil {
		return fmt.Errorf("Could not marshal data details.  %+v", err)
	}
	newdetails := &udnssdk.ProbeDetailsDTO{
		Detail: deets2,
	}
	newProbe.Details = newdetails

	log.Printf("[DEBUG] UltraDNS Probe create configuration: %#v", newProbe)
	name := d.Get("ownerName").(string)
	typ := newProbe.ProbeType
	zone := d.Get("zoneName").(string)
	guid := d.Id()
	_, err = client.SBTCService.UpdateProbe(name, typ, zone, guid, newProbe)
	if err != nil {
		return fmt.Errorf("[ERROR] Failed to update UltraDNS Probe: %s", err)
	}

	log.Printf("[DEBUG] UltraDNS Probe update configuration: %#v", updateProbe)

	return resourceUltraDNSProbeRead(d, meta)
}

func resourceUltraDNSProbeDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)
	zone := d.Get("zoneName").(string)
	guid := d.Id()
	name := d.Get("ownerName").(string)
	typ := d.Get("type").(string)
	log.Printf("[INFO] Deleting UltraDNS Probe: %s, %s", d.Get("zone").(string), d.Id())

	_, err := client.SBTCService.DeleteProbe(name, typ, zone, guid)

	if err != nil {
		return fmt.Errorf("[ERROR] Error deleting UltraDNS Probe: %s", err)
	}

	return nil
}
