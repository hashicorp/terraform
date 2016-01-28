package ultradns

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Ensighten/udnssdk"
	"github.com/fatih/structs"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/mapstructure"
)

func resourceUltradnsDirpool() *schema.Resource {
	return &schema.Resource{
		Create: resourceUltradnsDirpoolCreate,
		Read:   resourceUltradnsDirpoolRead,
		Update: resourceUltradnsDirpoolUpdate,
		Delete: resourceUltradnsDirpoolDelete,

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
			"type": &schema.Schema{
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
				Type:     schema.TypeList,
				Required: true,
				// Valid: len(rdataInfo) == len(rdata)
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// Required
						"host": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"all_non_configured": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"geo_info": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"is_account_level": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"codes": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"ip_info": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"is_account_level": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"ips": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"start": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
													// ConflictsWith: []string{"cidr", "address"},
												},
												"end": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
													// ConflictsWith: []string{"cidr", "address"},
												},
												"cidr": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
													// ConflictsWith: []string{"start", "end", "address"},
												},
												"address": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
													// ConflictsWith: []string{"start", "end", "cidr"},
												},
											},
										},
									},
								},
							},
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
			"conflict_resolve": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "GEO",
				// Valid: "GEO", "IP"
			},
			"no_response": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"all_non_configured": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"geo_info": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"is_account_level": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"codes": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"ip_info": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"is_account_level": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"ips": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"start": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
													// ConflictsWith: []string{"cidr", "address"},
												},
												"end": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
													// ConflictsWith: []string{"cidr", "address"},
												},
												"cidr": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
													// ConflictsWith: []string{"start", "end", "address"},
												},
												"address": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
													// ConflictsWith: []string{"start", "end", "cidr"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
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

func resourceUltradnsDirpoolCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makeDirpoolRRSetResource(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_dirpool create: %#v", r)
	_, err = client.RRSets.Create(r.RRSetKey(), r.RRSet())
	if err != nil {
		// FIXME: remove the json from log
		marshalled, _ := json.Marshal(r)
		ms := string(marshalled)
		return fmt.Errorf("create failed: %#v [[[[ %v ]]]] -> %v", r, ms, err)
	}

	d.SetId(r.ID())
	log.Printf("[INFO] ultradns_dirpool.id: %v", d.Id())

	return resourceUltradnsDirpoolRead(d, meta)
}

func resourceUltradnsDirpoolRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	rr, err := makeDirpoolRRSetResource(d)
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

	return populateResourceFromDirpool(d, &r)
}

func resourceUltradnsDirpoolUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makeDirpoolRRSetResource(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_dirpool update: %+v", r)
	_, err = client.RRSets.Update(r.RRSetKey(), r.RRSet())
	if err != nil {
		return fmt.Errorf("resource update failed: %v", err)
	}

	return resourceUltradnsDirpoolRead(d, meta)
}

func resourceUltradnsDirpoolDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	r, err := makeDirpoolRRSetResource(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] ultradns_dirpool delete: %+v", r)
	_, err = client.RRSets.Delete(r.RRSetKey())
	if err != nil {
		return fmt.Errorf("resource delete failed: %v", err)
	}

	return nil
}

// Resource Helpers

func makeDirpoolRRSetResource(d *schema.ResourceData) (rRSetResource, error) {
	rDataRaw := d.Get("rdata").([]interface{})
	res := rRSetResource{
		RRType:    d.Get("type").(string),
		Zone:      d.Get("zone").(string),
		OwnerName: d.Get("name").(string),
		TTL:       d.Get("ttl").(int),
		RData:     unzipRdataHosts(rDataRaw),
	}

	profile := udnssdk.DirPoolProfile{
		Context:         udnssdk.DirPoolSchema,
		Description:     d.Get("description").(string),
		ConflictResolve: d.Get("conflict_resolve").(string),
	}

	ri, err := unzipDirpoolRdataInfos(rDataRaw)
	if err != nil {
		return res, err
	}
	profile.RDataInfo = ri

	noResponseRaw := d.Get("no_response").([]interface{})
	if len(noResponseRaw) >= 1 {
		if len(noResponseRaw) > 1 {
			return res, fmt.Errorf("no_response: only 0 or 1 blocks alowed, got: %#v", len(noResponseRaw))
		}
		nr, err := makeDirpoolRdataInfo(noResponseRaw[0])
		if err != nil {
			return res, err
		}
		profile.NoResponse = nr
	}

	res.Profile = profile.RawProfile()

	return res, nil
}

func populateResourceFromDirpool(d *schema.ResourceData, r *udnssdk.RRSet) error {
	// TODO: fix from tcpool to dirpool
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
		return fmt.Errorf("RRSet.profile missing: invalid DirPool schema in: %#v", r)
	}
	p, err := r.Profile.DirPoolProfile()
	if err != nil {
		return fmt.Errorf("RRSet.profile could not be unmarshalled: %v\n", err)
	}

	// Set simple values
	d.Set("description", p.Description)
	d.Set("conflict_resolve", p.ConflictResolve)

	// TODO: rigorously test this to see if we can remove the error handling
	rd := zipDirpoolRData(r.RData, p.RDataInfo)
	err = d.Set("rdata", rd)
	if err != nil {
		return fmt.Errorf("rdata set failed: %v, from %#v", err, rd)
	}
	return nil
}

func unzipDirpoolRdataInfos(configured []interface{}) ([]udnssdk.DPRDataInfo, error) {
	res := make([]udnssdk.DPRDataInfo, 0, len(configured))
	for _, r := range configured {
		ri, err := makeDirpoolRdataInfo(r)
		if err != nil {
			return res, err
		}
		res = append(res, ri)
	}
	return res, nil
}

func makeDirpoolRdataInfo(configured interface{}) (udnssdk.DPRDataInfo, error) {
	data := configured.(map[string]interface{})
	res := udnssdk.DPRDataInfo{
		AllNonConfigured: data["all_non_configured"].(bool),
	}
	// IPInfo
	ipInfo := data["ip_info"].([]interface{})
	if len(ipInfo) >= 1 {
		if len(ipInfo) > 1 {
			return res, fmt.Errorf("ip_info: only 0 or 1 blocks alowed, got: %#v", len(ipInfo))
		}
		mapstructure.Decode(ipInfo[0], &res.IPInfo)
	}
	// GeoInfo
	geoInfo := data["geo_info"].([]interface{})
	if len(geoInfo) >= 1 {
		if len(geoInfo) > 1 {
			return res, fmt.Errorf("geo_info: only 0 or 1 blocks alowed, got: %#v", len(geoInfo))
		}
		mapstructure.Decode(geoInfo[0], &res.GeoInfo)
	}
	return res, nil
}

// collate and zip RData and RDataInfo into []map[string]interface{}
func zipDirpoolRData(rds []string, rdis []udnssdk.DPRDataInfo) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(rds))
	for i, rdi := range rdis {
		r := map[string]interface{}{
			"host":               rds[i],
			"all_non_configured": rdi.AllNonConfigured,
			"ip_info":            flattenIPInfos(rdi.IPInfo),
			"geo_info":           flattenGeoInfos(rdi.GeoInfo),
		}
		result = append(result, r)
	}
	return result
}

func flattenIPInfos(rdi *udnssdk.IPInfo) []map[string]interface{} {
	res := make([]map[string]interface{}, 0, 1)
	if rdi != nil {
		res = append(res, map[string]interface{}{
			"name":             rdi.Name,
			"is_account_level": rdi.IsAccountLevel,
			// "ips": flattenIPAddrDTOs(rdi.Ips),
		})
	}
	return res
}

func flattenGeoInfos(gi *udnssdk.GeoInfo) []map[string]interface{} {
	res := make([]map[string]interface{}, 0, 1)
	if gi != nil {
		res = append(res, map[string]interface{}{
			"name":             gi.Name,
			"is_account_level": gi.IsAccountLevel,
			"codes":            gi.Codes,
		})
	}
	return res
}

func mapstructureEncode(rawVal interface{}) map[string]interface{} {
	s := structs.New(rawVal)
	s.TagName = "mapstructure"
	// s.TagName = "json"
	return s.Map()
}
