package ultradns

import (
	"fmt"
	"log"

	"github.com/Ensighten/udnssdk"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

// Conversion helper functions
type rRSetResource struct {
	OwnerName string
	RRType    string
	RData     []string
	TTL       int
	Profile   udnssdk.RawProfile
	Zone      string
}

// profileAttrSchemaMap is a map from each ultradns_tcpool attribute name onto its respective ProfileSchema URI
var profileAttrSchemaMap = map[string]udnssdk.ProfileSchema{
	"dirpool_profile": udnssdk.DirPoolSchema,
	"rdpool_profile":  udnssdk.RDPoolSchema,
	"sbpool_profile":  udnssdk.SBPoolSchema,
	"tcpool_profile":  udnssdk.TCPoolSchema,
}

func (r rRSetResource) RRSetKey() udnssdk.RRSetKey {
	return udnssdk.RRSetKey{
		Zone: r.Zone,
		Type: r.RRType,
		Name: r.OwnerName,
	}
}

func (r rRSetResource) RRSet() udnssdk.RRSet {
	return udnssdk.RRSet{
		OwnerName: r.OwnerName,
		RRType:    r.RRType,
		RData:     r.RData,
		TTL:       r.TTL,
		Profile:   r.Profile,
	}
}

func (r rRSetResource) ID() string {
	return fmt.Sprintf("%s.%s", r.OwnerName, r.Zone)
}

func unzipRdataHosts(configured []interface{}) []string {
	hs := make([]string, 0, len(configured))
	for _, rRaw := range configured {
		data := rRaw.(map[string]interface{})
		h := data["host"].(string)
		hs = append(hs, h)
	}
	return hs
}

func schemaPingProbe() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"packets": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  3,
			},
			"packet_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  56,
			},
			"limit": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Set:      hashLimits,
				Elem:     resourceProbeLimits(),
			},
		},
	}
}

func resourceProbeLimits() *schema.Resource {
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

type probeResource struct {
	Name string
	Zone string
	ID   string

	Agents     []string
	Interval   string
	PoolRecord string
	Threshold  int
	Type       udnssdk.ProbeType

	Details *udnssdk.ProbeDetailsDTO
}

func (p probeResource) RRSetKey() udnssdk.RRSetKey {
	return p.Key().RRSetKey()
}

func (p probeResource) ProbeInfoDTO() udnssdk.ProbeInfoDTO {
	return udnssdk.ProbeInfoDTO{
		ID:         p.ID,
		PoolRecord: p.PoolRecord,
		ProbeType:  p.Type,
		Interval:   p.Interval,
		Agents:     p.Agents,
		Threshold:  p.Threshold,
		Details:    p.Details,
	}
}

func (p probeResource) Key() udnssdk.ProbeKey {
	return udnssdk.ProbeKey{
		Zone: p.Zone,
		Name: p.Name,
		ID:   p.ID,
	}
}

func mapFromLimit(name string, l udnssdk.ProbeDetailsLimitDTO) map[string]interface{} {
	return map[string]interface{}{
		"name":     name,
		"warning":  l.Warning,
		"critical": l.Critical,
		"fail":     l.Fail,
	}
}

// hashLimits generates a hashcode for a limits block
func hashLimits(v interface{}) int {
	m := v.(map[string]interface{})
	h := hashcode.String(m["name"].(string))
	log.Printf("[INFO] hashLimits(): %v -> %v", m["name"].(string), h)
	return h
}

// makeSetFromLimits encodes an array of Limits into a
// *schema.Set in the appropriate structure for the schema
func makeSetFromLimits(ls map[string]udnssdk.ProbeDetailsLimitDTO) *schema.Set {
	s := &schema.Set{F: hashLimits}
	for name, l := range ls {
		s.Add(mapFromLimit(name, l))
	}
	return s
}

func makeProbeDetailsLimit(configured interface{}) *udnssdk.ProbeDetailsLimitDTO {
	l := configured.(map[string]interface{})
	return &udnssdk.ProbeDetailsLimitDTO{
		Warning:  l["warning"].(int),
		Critical: l["critical"].(int),
		Fail:     l["fail"].(int),
	}
}

// makeSetFromStrings encodes an []string into a
// *schema.Set in the appropriate structure for the schema
func makeSetFromStrings(ss []string) *schema.Set {
	st := &schema.Set{F: schema.HashString}
	for _, s := range ss {
		st.Add(s)
	}
	return st
}

// hashRdata generates a hashcode for an Rdata block
func hashRdatas(v interface{}) int {
	m := v.(map[string]interface{})
	h := hashcode.String(m["host"].(string))
	log.Printf("[DEBUG] hashRdatas(): %v -> %v", m["host"].(string), h)
	return h
}
