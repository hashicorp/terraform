package ultradns

import (
	"fmt"
	"log"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceUltradnsProbeHTTP() *schema.Resource {
	return &schema.Resource{
		Create: resourceUltradnsProbeHTTPCreate,
		Read:   resourceUltradnsProbeHTTPRead,
		Update: resourceUltradnsProbeHTTPUpdate,
		Delete: resourceUltradnsProbeHTTPDelete,

		Schema: map[string]*schema.Schema{
			// Key
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
			"pool_record": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			// Required
			"agents": &schema.Schema{
				Type:     schema.TypeSet,
				Set:      schema.HashString,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			// Optional
			"interval": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "FIVE_MINUTES",
			},
			"http_probe": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     schemaHTTPProbe(),
			},
			// Computed
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func schemaHTTPProbe() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"transaction": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"method": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"url": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"transmitted_data": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"follow_redirects": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"limit": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Set:      hashLimits,
							Elem:     resourceProbeLimits(),
						},
					},
				},
			},
			"total_limits": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"warning": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"critical": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"fail": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceUltradnsProbeHTTPCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makeHTTPProbeResource(d)
	if err != nil {
		return fmt.Errorf("Could not load ultradns_probe_http configuration: %v", err)
	}

	log.Printf("[INFO] ultradns_probe_http create: %#v, detail: %#v", r, r.Details.Detail)
	resp, err := client.Probes.Create(r.Key().RRSetKey(), r.ProbeInfoDTO())
	if err != nil {
		return fmt.Errorf("create failed: %v", err)
	}

	uri := resp.Header.Get("Location")
	d.Set("uri", uri)
	d.SetId(uri)
	log.Printf("[INFO] ultradns_probe_http.id: %v", d.Id())

	return resourceUltradnsProbeHTTPRead(d, meta)
}

func resourceUltradnsProbeHTTPRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makeHTTPProbeResource(d)
	if err != nil {
		return fmt.Errorf("Could not load ultradns_probe_http configuration: %v", err)
	}

	log.Printf("[DEBUG] ultradns_probe_http read: %#v", r)
	probe, _, err := client.Probes.Find(r.Key())
	log.Printf("[DEBUG] ultradns_probe_http response: %#v", probe)

	if err != nil {
		uderr, ok := err.(*udnssdk.ErrorResponseList)
		if ok {
			for _, r := range uderr.Responses {
				// 70002 means Probes Not Found
				if r.ErrorCode == 70002 {
					d.SetId("")
					return nil
				}
				return fmt.Errorf("not found: %s", err)
			}
		}
		return fmt.Errorf("not found: %s", err)
	}

	return populateResourceDataFromHTTPProbe(probe, d)
}

func resourceUltradnsProbeHTTPUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makeHTTPProbeResource(d)
	if err != nil {
		return fmt.Errorf("Could not load ultradns_probe_http configuration: %v", err)
	}

	log.Printf("[INFO] ultradns_probe_http update: %+v", r)
	_, err = client.Probes.Update(r.Key(), r.ProbeInfoDTO())
	if err != nil {
		return fmt.Errorf("update failed: %s", err)
	}

	return resourceUltradnsProbeHTTPRead(d, meta)
}

func resourceUltradnsProbeHTTPDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makeHTTPProbeResource(d)
	if err != nil {
		return fmt.Errorf("Could not load ultradns_probe_http configuration: %s", err)
	}

	log.Printf("[INFO] ultradns_probe_http delete: %+v", r)
	_, err = client.Probes.Delete(r.Key())
	if err != nil {
		return fmt.Errorf("delete failed: %s", err)
	}

	return nil
}

// Resource Helpers

func makeHTTPProbeResource(d *schema.ResourceData) (probeResource, error) {
	p := probeResource{}
	p.Zone = d.Get("zone").(string)
	p.Name = d.Get("name").(string)
	p.ID = d.Id()
	p.Interval = d.Get("interval").(string)
	p.PoolRecord = d.Get("pool_record").(string)
	p.Threshold = d.Get("threshold").(int)
	for _, a := range d.Get("agents").(*schema.Set).List() {
		p.Agents = append(p.Agents, a.(string))
	}

	p.Type = udnssdk.HTTPProbeType
	hps := d.Get("http_probe").([]interface{})
	if len(hps) >= 1 {
		if len(hps) > 1 {
			return p, fmt.Errorf("http_probe: only 0 or 1 blocks alowed, got: %#v", len(hps))
		}
		p.Details = makeHTTPProbeDetails(hps[0])
	}

	return p, nil
}

func makeHTTPProbeDetails(configured interface{}) *udnssdk.ProbeDetailsDTO {
	data := configured.(map[string]interface{})
	// Convert limits from flattened set format to mapping.
	d := udnssdk.HTTPProbeDetailsDTO{}

	ts := []udnssdk.Transaction{}
	for _, rt := range data["transaction"].([]interface{}) {
		mt := rt.(map[string]interface{})
		ls := make(map[string]udnssdk.ProbeDetailsLimitDTO)
		for _, limit := range mt["limit"].(*schema.Set).List() {
			l := limit.(map[string]interface{})
			name := l["name"].(string)
			ls[name] = *makeProbeDetailsLimit(l)
		}
		t := udnssdk.Transaction{
			Method:          mt["method"].(string),
			URL:             mt["url"].(string),
			TransmittedData: mt["transmitted_data"].(string),
			FollowRedirects: mt["follow_redirects"].(bool),
			Limits:          ls,
		}
		ts = append(ts, t)
	}
	d.Transactions = ts
	rawLims := data["total_limits"].([]interface{})
	if len(rawLims) >= 1 {
		// TODO: validate 0 or 1 total_limits
		// if len(rawLims) > 1 {
		// 	return nil, fmt.Errorf("total_limits: only 0 or 1 blocks alowed, got: %#v", len(rawLims))
		// }
		d.TotalLimits = makeProbeDetailsLimit(rawLims[0])
	}
	res := udnssdk.ProbeDetailsDTO{
		Detail: d,
	}
	return &res
}

func populateResourceDataFromHTTPProbe(p udnssdk.ProbeInfoDTO, d *schema.ResourceData) error {
	d.SetId(p.ID)
	d.Set("pool_record", p.PoolRecord)
	d.Set("interval", p.Interval)
	d.Set("agents", makeSetFromStrings(p.Agents))
	d.Set("threshold", p.Threshold)

	hp := map[string]interface{}{}
	hd, err := p.Details.HTTPProbeDetails()
	if err != nil {
		return fmt.Errorf("ProbeInfo.details could not be unmarshalled: %v, Details: %#v", err, p.Details)
	}
	ts := make([]map[string]interface{}, 0, len(hd.Transactions))
	for _, rt := range hd.Transactions {
		t := map[string]interface{}{
			"method":           rt.Method,
			"url":              rt.URL,
			"transmitted_data": rt.TransmittedData,
			"follow_redirects": rt.FollowRedirects,
			"limit":            makeSetFromLimits(rt.Limits),
		}
		ts = append(ts, t)
	}
	hp["transaction"] = ts

	tls := []map[string]interface{}{}
	rawtl := hd.TotalLimits
	if rawtl != nil {
		tl := map[string]interface{}{
			"warning":  rawtl.Warning,
			"critical": rawtl.Critical,
			"fail":     rawtl.Fail,
		}
		tls = append(tls, tl)
	}
	hp["total_limits"] = tls

	err = d.Set("http_probe", []map[string]interface{}{hp})
	if err != nil {
		return fmt.Errorf("http_probe set failed: %v, from %#v", err, hp)
	}
	return nil
}
