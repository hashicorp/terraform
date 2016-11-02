package ultradns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Ensighten/udnssdk"
	"github.com/fatih/structs"
	"github.com/hashicorp/terraform/helper/hashcode"
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
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if len(value) > 255 {
						errors = append(errors, fmt.Errorf(
							"'description' too long, must be less than 255 characters"))
					}
					return
				},
			},
			"rdata": &schema.Schema{
				// UltraDNS API does not respect rdata ordering
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
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Set:      schema.HashString,
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
										Type:     schema.TypeSet,
										Optional: true,
										Set:      hashIPInfoIPs,
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
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					if value != "GEO" && value != "IP" {
						errors = append(errors, fmt.Errorf(
							"only 'GEO', and 'IP' are supported values for 'conflict_resolve'"))
					}
					return
				},
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
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Set:      schema.HashString,
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
										Type:     schema.TypeSet,
										Optional: true,
										Set:      hashIPInfoIPs,
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

// makeDirpoolRRSetResource converts ResourceData into an rRSetResource
// ready for use in any CRUD operation
func makeDirpoolRRSetResource(d *schema.ResourceData) (rRSetResource, error) {
	rDataRaw := d.Get("rdata").(*schema.Set).List()
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

	ri, err := makeDirpoolRdataInfos(rDataRaw)
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

// populateResourceFromDirpool takes an RRSet and populates the ResourceData
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

	// Ensure default looks like "GEO", even when nothing is returned
	if p.ConflictResolve == "" {
		d.Set("conflict_resolve", "GEO")
	} else {
		d.Set("conflict_resolve", p.ConflictResolve)
	}

	rd := makeSetFromDirpoolRdata(r.RData, p.RDataInfo)
	err = d.Set("rdata", rd)
	if err != nil {
		return fmt.Errorf("rdata set failed: %v, from %#v", err, rd)
	}
	return nil
}

// makeDirpoolRdataInfos converts []map[string]interface{} from rdata
// blocks into []DPRDataInfo
func makeDirpoolRdataInfos(configured []interface{}) ([]udnssdk.DPRDataInfo, error) {
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

// makeDirpoolRdataInfo converts a map[string]interface{} from
// an rdata or no_response block into an DPRDataInfo
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
		ii, err := makeIPInfo(ipInfo[0])
		if err != nil {
			return res, fmt.Errorf("%v ip_info: %#v", err, ii)
		}
		res.IPInfo = &ii
	}
	// GeoInfo
	geoInfo := data["geo_info"].([]interface{})
	if len(geoInfo) >= 1 {
		if len(geoInfo) > 1 {
			return res, fmt.Errorf("geo_info: only 0 or 1 blocks alowed, got: %#v", len(geoInfo))
		}
		gi, err := makeGeoInfo(geoInfo[0])
		if err != nil {
			return res, fmt.Errorf("%v geo_info: %#v GeoInfo: %#v", err, geoInfo[0], gi)
		}
		res.GeoInfo = &gi
	}
	return res, nil
}

// makeGeoInfo converts a map[string]interface{} from an geo_info block
// into an GeoInfo
func makeGeoInfo(configured interface{}) (udnssdk.GeoInfo, error) {
	var res udnssdk.GeoInfo
	c := configured.(map[string]interface{})
	err := mapDecode(c, &res)
	if err != nil {
		return res, err
	}

	rawCodes := c["codes"].(*schema.Set).List()
	res.Codes = make([]string, 0, len(rawCodes))
	for _, i := range rawCodes {
		res.Codes = append(res.Codes, i.(string))
	}
	return res, err
}

// makeIPInfo converts a map[string]interface{} from an ip_info block
// into an IPInfo
func makeIPInfo(configured interface{}) (udnssdk.IPInfo, error) {
	var res udnssdk.IPInfo
	c := configured.(map[string]interface{})
	err := mapDecode(c, &res)
	if err != nil {
		return res, err
	}

	rawIps := c["ips"].(*schema.Set).List()
	res.Ips = make([]udnssdk.IPAddrDTO, 0, len(rawIps))
	for _, rawIa := range rawIps {
		var i udnssdk.IPAddrDTO
		err = mapDecode(rawIa, &i)
		if err != nil {
			return res, err
		}
		res.Ips = append(res.Ips, i)
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
			"ip_info":            mapFromIPInfos(rdi.IPInfo),
			"geo_info":           mapFromGeoInfos(rdi.GeoInfo),
		}
		result = append(result, r)
	}
	return result
}

// makeSetFromDirpoolRdata encodes an array of Rdata into a
// *schema.Set in the appropriate structure for the schema
func makeSetFromDirpoolRdata(rds []string, rdis []udnssdk.DPRDataInfo) *schema.Set {
	s := &schema.Set{F: hashRdatas}
	rs := zipDirpoolRData(rds, rdis)
	for _, r := range rs {
		s.Add(r)
	}
	return s
}

// mapFromIPInfos encodes 0 or 1 IPInfos into a []map[string]interface{}
// in the appropriate structure for the schema
func mapFromIPInfos(rdi *udnssdk.IPInfo) []map[string]interface{} {
	res := make([]map[string]interface{}, 0, 1)
	if rdi != nil {
		m := map[string]interface{}{
			"name":             rdi.Name,
			"is_account_level": rdi.IsAccountLevel,
			"ips":              makeSetFromIPAddrDTOs(rdi.Ips),
		}
		res = append(res, m)
	}
	return res
}

// makeSetFromIPAddrDTOs encodes an array of IPAddrDTO into a
// *schema.Set in the appropriate structure for the schema
func makeSetFromIPAddrDTOs(ias []udnssdk.IPAddrDTO) *schema.Set {
	s := &schema.Set{F: hashIPInfoIPs}
	for _, ia := range ias {
		s.Add(mapEncode(ia))
	}
	return s
}

// mapFromGeoInfos encodes 0 or 1 GeoInfos into a []map[string]interface{}
// in the appropriate structure for the schema
func mapFromGeoInfos(gi *udnssdk.GeoInfo) []map[string]interface{} {
	res := make([]map[string]interface{}, 0, 1)
	if gi != nil {
		m := mapEncode(gi)
		m["codes"] = makeSetFromStrings(gi.Codes)
		res = append(res, m)
	}
	return res
}

// hashIPInfoIPs generates a hashcode for an ip_info.ips block
func hashIPInfoIPs(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["start"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["end"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["cidr"].(string)))
	buf.WriteString(fmt.Sprintf("%s", m["address"].(string)))

	h := hashcode.String(buf.String())
	log.Printf("[DEBUG] hashIPInfoIPs(): %v -> %v", buf.String(), h)
	return h
}

// Map <-> Struct transcoding
// Ideally, we sould be able to handle almost all the type conversion
// in this resource using the following helpers. Unfortunately, some
// issues remain:
// - schema.Set values cannot be naively assigned, and must be
//   manually converted
// - ip_info and geo_info come in as []map[string]interface{}, but are
//   in DPRDataInfo as singluar.

// mapDecode takes a map[string]interface{} and uses reflection to
// convert it into the given Go native structure. val must be a pointer
// to a struct. This is identical to mapstructure.Decode, but uses the
// `terraform:` tag instead of `mapstructure:`
func mapDecode(m interface{}, rawVal interface{}) error {
	config := &mapstructure.DecoderConfig{
		Metadata:         nil,
		TagName:          "terraform",
		Result:           rawVal,
		WeaklyTypedInput: true,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	return decoder.Decode(m)
}

func mapEncode(rawVal interface{}) map[string]interface{} {
	s := structs.New(rawVal)
	s.TagName = "terraform"
	return s.Map()
}
