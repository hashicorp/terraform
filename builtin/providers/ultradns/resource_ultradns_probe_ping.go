package ultradns

import (
	"fmt"
	"log"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceUltradnsProbePing() *schema.Resource {
	return &schema.Resource{
		Create: resourceUltradnsProbePingCreate,
		Read:   resourceUltradnsProbePingRead,
		Update: resourceUltradnsProbePingUpdate,
		Delete: resourceUltradnsProbePingDelete,

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
				Type:     schema.TypeList,
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
			"ping_probe": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     schemaPingProbe(),
			},
			// Computed
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceUltradnsProbePingCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makePingProbeResource(d)
	if err != nil {
		return fmt.Errorf("Could not load ultradns_probe_ping configuration: %v", err)
	}

	log.Printf("[INFO] ultradns_probe_ping create: %#v, detail: %#v", r, r.Details.Detail)
	resp, err := client.Probes.Create(r.Key().RRSetKey(), r.ProbeInfoDTO())
	if err != nil {
		return fmt.Errorf("create failed: %v", err)
	}

	uri := resp.Header.Get("Location")
	d.Set("uri", uri)
	d.SetId(uri)
	log.Printf("[INFO] ultradns_probe_ping.id: %v", d.Id())

	return resourceUltradnsProbePingRead(d, meta)
}

func resourceUltradnsProbePingRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makePingProbeResource(d)
	if err != nil {
		return fmt.Errorf("Could not load ultradns_probe_ping configuration: %v", err)
	}

	log.Printf("[DEBUG] ultradns_probe_ping read: %#v", r)
	probe, _, err := client.Probes.Find(r.Key())
	log.Printf("[DEBUG] ultradns_probe_ping response: %#v", probe)

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

	return populateResourceDataFromPingProbe(probe, d)
}

func resourceUltradnsProbePingUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makePingProbeResource(d)
	if err != nil {
		return fmt.Errorf("Could not load ultradns_probe_ping configuration: %v", err)
	}

	log.Printf("[INFO] ultradns_probe_ping update: %+v", r)
	_, err = client.Probes.Update(r.Key(), r.ProbeInfoDTO())
	if err != nil {
		return fmt.Errorf("update failed: %s", err)
	}

	return resourceUltradnsProbePingRead(d, meta)
}

func resourceUltradnsProbePingDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makePingProbeResource(d)
	if err != nil {
		return fmt.Errorf("Could not load ultradns_probe_ping configuration: %s", err)
	}

	log.Printf("[INFO] ultradns_probe_ping delete: %+v", r)
	_, err = client.Probes.Delete(r.Key())
	if err != nil {
		return fmt.Errorf("delete failed: %s", err)
	}

	return nil
}

// Resource Helpers

func makePingProbeResource(d *schema.ResourceData) (probeResource, error) {
	p := probeResource{}
	p.Zone = d.Get("zone").(string)
	p.Name = d.Get("name").(string)
	p.ID = d.Id()
	p.Interval = d.Get("interval").(string)
	p.PoolRecord = d.Get("pool_record").(string)
	p.Threshold = d.Get("threshold").(int)
	for _, a := range d.Get("agents").([]interface{}) {
		p.Agents = append(p.Agents, a.(string))
	}

	p.Type = udnssdk.PingProbeType
	pps := d.Get("ping_probe").([]interface{})
	if len(pps) >= 1 {
		if len(pps) > 1 {
			return p, fmt.Errorf("ping_probe: only 0 or 1 blocks alowed, got: %#v", len(pps))
		}
		p.Details = makePingProbeDetails(pps[0])
	}

	return p, nil
}

func makePingProbeDetails(configured interface{}) *udnssdk.ProbeDetailsDTO {
	data := configured.(map[string]interface{})
	// Convert limits from flattened set format to mapping.
	ls := make(map[string]udnssdk.ProbeDetailsLimitDTO)
	for _, limit := range data["limit"].(*schema.Set).List() {
		l := limit.(map[string]interface{})
		name := l["name"].(string)
		ls[name] = *makeProbeDetailsLimit(l)
	}
	res := udnssdk.ProbeDetailsDTO{
		Detail: udnssdk.PingProbeDetailsDTO{
			Limits:     ls,
			PacketSize: data["packet_size"].(int),
			Packets:    data["packets"].(int),
		},
	}
	return &res
}

func populateResourceDataFromPingProbe(p udnssdk.ProbeInfoDTO, d *schema.ResourceData) error {
	d.SetId(p.ID)
	d.Set("pool_record", p.PoolRecord)
	d.Set("interval", p.Interval)
	d.Set("agents", p.Agents)
	d.Set("threshold", p.Threshold)

	pd, err := p.Details.PingProbeDetails()
	if err != nil {
		return fmt.Errorf("ProbeInfo.details could not be unmarshalled: %v, Details: %#v", err, p.Details)
	}
	pp := map[string]interface{}{
		"packets":     pd.Packets,
		"packet_size": pd.PacketSize,
		"limit":       makeSetFromLimits(pd.Limits),
	}

	err = d.Set("ping_probe", []map[string]interface{}{pp})
	if err != nil {
		return fmt.Errorf("ping_probe set failed: %v, from %#v", err, pp)
	}
	return nil
}
