package ultradns

import (
	"fmt"
	"log"
	"strings"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceUltradnsTcpool() *schema.Resource {
	return &schema.Resource{
		Create: resourceUltradnsTcpoolCreate,
		Read:   resourceUltradnsTcpoolRead,
		Update: resourceUltradnsTcpoolUpdate,
		Delete: resourceUltradnsTcpoolDelete,

		Schema: map[string]*schema.Schema{
			// Required
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
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				// 0-255 char
			},
			"rdata": &schema.Schema{
				Type:     schema.TypeSet,
				Set:      hashRdatas,
				Required: true,
				// Valid: len(rdataInfo) == len(rdata)
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// Required
						"host": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						// Optional
						"failover_delay": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  0,
							// Valid: 0-30
							// Units: Minutes
						},
						"priority": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  1,
						},
						"run_probes": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"state": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "NORMAL",
						},
						"threshold": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  1,
						},
						"weight": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  2,
							// Valid: i%2 == 0 && 2 <= i <= 100
						},
					},
				},
			},
			// Optional
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  3600,
			},
			"run_probes": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"act_on_probes": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"max_to_lb": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				// Valid: 0 <= i <= len(rdata)
			},
			"backup_record_rdata": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				// Valid: IPv4 address or CNAME
			},
			"backup_record_failover_delay": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				// Valid: 0-30
				// Units: Minutes
			},
			// Computed
			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// CRUD Operations

func resourceUltradnsTcpoolCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := newRRSetResourceFromTcpool(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_tcpool create: %#v", r)
	_, err = client.RRSets.Create(r.RRSetKey(), r.RRSet())
	if err != nil {
		return fmt.Errorf("create failed: %#v -> %v", r, err)
	}

	d.SetId(r.ID())
	log.Printf("[INFO] ultradns_tcpool.id: %v", d.Id())

	return resourceUltradnsTcpoolRead(d, meta)
}

func resourceUltradnsTcpoolRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	rr, err := newRRSetResourceFromTcpool(d)
	if err != nil {
		return err
	}

	rrsets, err := client.RRSets.Select(rr.RRSetKey())
	if err != nil {
		uderr, ok := err.(*udnssdk.ErrorResponseList)
		if ok {
			for _, resps := range uderr.Responses {
				// 70002 means Records Not Found
				if resps.ErrorCode == 70002 {
					d.SetId("")
					return nil
				}
				return fmt.Errorf("resource not found: %v", err)
			}
		}
		return fmt.Errorf("resource not found: %v", err)
	}

	r := rrsets[0]

	zone := d.Get("zone")
	// ttl
	d.Set("ttl", r.TTL)
	// hostname
	if r.OwnerName == "" {
		d.Set("hostname", zone)
	} else {
		if strings.HasSuffix(r.OwnerName, ".") {
			d.Set("hostname", r.OwnerName)
		} else {
			d.Set("hostname", fmt.Sprintf("%s.%s", r.OwnerName, zone))
		}
	}

	// And now... the Profile!
	if r.Profile == nil {
		return fmt.Errorf("RRSet.profile missing: invalid TCPool schema in: %#v", r)
	}
	p, err := r.Profile.TCPoolProfile()
	if err != nil {
		return fmt.Errorf("RRSet.profile could not be unmarshalled: %v\n", err)
	}

	// Set simple values
	d.Set("description", p.Description)
	d.Set("run_probes", p.RunProbes)
	d.Set("act_on_probes", p.ActOnProbes)
	d.Set("max_to_lb", p.MaxToLB)
	if p.BackupRecord != nil {
		d.Set("backup_record_rdata", p.BackupRecord.RData)
		d.Set("backup_record_failover_delay", p.BackupRecord.FailoverDelay)
	}

	// TODO: rigorously test this to see if we can remove the error handling
	err = d.Set("rdata", makeSetFromRdata(r.RData, p.RDataInfo))
	if err != nil {
		return fmt.Errorf("rdata set failed: %#v", err)
	}
	return nil
}

func resourceUltradnsTcpoolUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := newRRSetResourceFromTcpool(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_tcpool update: %+v", r)
	_, err = client.RRSets.Update(r.RRSetKey(), r.RRSet())
	if err != nil {
		return fmt.Errorf("resource update failed: %v", err)
	}

	return resourceUltradnsTcpoolRead(d, meta)
}

func resourceUltradnsTcpoolDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := newRRSetResourceFromTcpool(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_tcpool delete: %+v", r)
	_, err = client.RRSets.Delete(r.RRSetKey())
	if err != nil {
		return fmt.Errorf("resource delete failed: %v", err)
	}

	return nil
}

// Resource Helpers

func newRRSetResourceFromTcpool(d *schema.ResourceData) (rRSetResource, error) {
	rDataRaw := d.Get("rdata").(*schema.Set).List()
	r := rRSetResource{
		// "The only valid rrtype value for SiteBacker or Traffic Controller pools is A"
		// per https://portal.ultradns.com/static/docs/REST-API_User_Guide.pdf
		RRType:    "A",
		Zone:      d.Get("zone").(string),
		OwnerName: d.Get("name").(string),
		TTL:       d.Get("ttl").(int),
		RData:     unzipRdataHosts(rDataRaw),
	}

	profile := udnssdk.TCPoolProfile{
		Context:     udnssdk.TCPoolSchema,
		ActOnProbes: d.Get("act_on_probes").(bool),
		Description: d.Get("description").(string),
		MaxToLB:     d.Get("max_to_lb").(int),
		RunProbes:   d.Get("run_probes").(bool),
		RDataInfo:   unzipRdataInfos(rDataRaw),
	}

	// Only send BackupRecord if present
	br := d.Get("backup_record_rdata").(string)
	if br != "" {
		profile.BackupRecord = &udnssdk.BackupRecord{
			RData:         d.Get("backup_record_rdata").(string),
			FailoverDelay: d.Get("backup_record_failover_delay").(int),
		}
	}

	rp := profile.RawProfile()
	r.Profile = rp

	return r, nil
}

func unzipRdataInfos(configured []interface{}) []udnssdk.SBRDataInfo {
	rdataInfos := make([]udnssdk.SBRDataInfo, 0, len(configured))
	for _, rRaw := range configured {
		data := rRaw.(map[string]interface{})
		r := udnssdk.SBRDataInfo{
			FailoverDelay: data["failover_delay"].(int),
			Priority:      data["priority"].(int),
			RunProbes:     data["run_probes"].(bool),
			State:         data["state"].(string),
			Threshold:     data["threshold"].(int),
			Weight:        data["weight"].(int),
		}
		rdataInfos = append(rdataInfos, r)
	}
	return rdataInfos
}

// collate and zip RData and RDataInfo into []map[string]interface{}
func zipRData(rds []string, rdis []udnssdk.SBRDataInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(rds))
	for i, rdi := range rdis {
		r := map[string]interface{}{
			"host":           rds[i],
			"failover_delay": rdi.FailoverDelay,
			"priority":       rdi.Priority,
			"run_probes":     rdi.RunProbes,
			"state":          rdi.State,
			"threshold":      rdi.Threshold,
			"weight":         rdi.Weight,
		}
		result = append(result, r)
	}
	return result
}

// makeSetFromRdatas encodes an array of Rdata into a
// *schema.Set in the appropriate structure for the schema
func makeSetFromRdata(rds []string, rdis []udnssdk.SBRDataInfo) *schema.Set {
	s := &schema.Set{F: hashRdatas}
	rs := zipRData(rds, rdis)
	for _, r := range rs {
		s.Add(r)
	}
	return s
}
