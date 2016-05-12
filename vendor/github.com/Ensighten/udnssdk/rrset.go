package udnssdk

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// RRSetsService provides access to RRSet resources
type RRSetsService struct {
	client *Client
}

// Here is the big 'Profile' mess that should get refactored to a more managable place

// ProfileSchema are the schema URIs for RRSet Profiles
type ProfileSchema string

const (
	// DirPoolSchema is the schema URI for a Directional pool profile
	DirPoolSchema ProfileSchema = "http://schemas.ultradns.com/DirPool.jsonschema"
	// RDPoolSchema is the schema URI for a Resource Distribution pool profile
	RDPoolSchema = "http://schemas.ultradns.com/RDPool.jsonschema"
	// SBPoolSchema is the schema URI for a SiteBacker pool profile
	SBPoolSchema = "http://schemas.ultradns.com/SBPool.jsonschema"
	// TCPoolSchema is the schema URI for a Traffic Controller pool profile
	TCPoolSchema = "http://schemas.ultradns.com/TCPool.jsonschema"
)

// RawProfile represents the naive interface to an RRSet Profile
type RawProfile map[string]interface{}

// Context extracts the schema context from a RawProfile
func (rp RawProfile) Context() ProfileSchema {
	return ProfileSchema(rp["@context"].(string))
}

// GetProfileObject extracts the full Profile by its schema type
func (rp RawProfile) GetProfileObject() (interface{}, error) {
	c := rp.Context()
	switch c {
	case DirPoolSchema:
		return rp.DirPoolProfile()
	case RDPoolSchema:
		return rp.RDPoolProfile()
	case SBPoolSchema:
		return rp.SBPoolProfile()
	case TCPoolSchema:
		return rp.TCPoolProfile()
	default:
		return nil, fmt.Errorf("Fallthrough on GetProfileObject type %s\n", c)
	}
}

// remarshalJSON takes a structure and marshals it into another type
func remarshalJSON(in, result interface{}) error {
	bs, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(bs, &result)
}

// DirPoolProfile extracts the full Profile as a DirPoolProfile or returns an error
func (rp RawProfile) DirPoolProfile() (DirPoolProfile, error) {
	var result DirPoolProfile
	c := rp.Context()
	if c != DirPoolSchema {
		return result, fmt.Errorf("RDPoolProfile has incorrect JSON-LD @context %s\n", c)
	}
	err := remarshalJSON(rp, &result)
	return result, err
}

// RDPoolProfile extracts the full Profile as a RDPoolProfile or returns an error
func (rp RawProfile) RDPoolProfile() (RDPoolProfile, error) {
	var result RDPoolProfile
	c := rp.Context()
	if c != RDPoolSchema {
		return result, fmt.Errorf("RDPoolProfile has incorrect JSON-LD @context %s\n", c)
	}
	err := remarshalJSON(rp, &result)
	return result, err
}

// SBPoolProfile extracts the full Profile as a SBPoolProfile or returns an error
func (rp RawProfile) SBPoolProfile() (SBPoolProfile, error) {
	var result SBPoolProfile
	c := rp.Context()
	if c != SBPoolSchema {
		return result, fmt.Errorf("SBPoolProfile has incorrect JSON-LD @context %s\n", c)
	}
	err := remarshalJSON(rp, &result)
	return result, err
}

// TCPoolProfile extracts the full Profile as a TCPoolProfile or returns an error
func (rp RawProfile) TCPoolProfile() (TCPoolProfile, error) {
	var result TCPoolProfile
	c := rp.Context()
	if c != TCPoolSchema {
		return result, fmt.Errorf("TCPoolProfile has incorrect JSON-LD @context %s\n", c)
	}
	err := remarshalJSON(rp, &result)
	return result, err
}

// RawProfile converts to a naive RawProfile
func (p DirPoolProfile) RawProfile() (RawProfile, error) {
	var rp RawProfile
	err := remarshalJSON(p, &rp)
	return rp, err
}

// RawProfile converts to a naive RawProfile
func (p RDPoolProfile) RawProfile() (RawProfile, error) {
	var rp RawProfile
	err := remarshalJSON(p, &rp)
	return rp, err
}

// RawProfile converts to a naive RawProfile
func (p SBPoolProfile) RawProfile() (RawProfile, error) {
	var rp RawProfile
	err := remarshalJSON(p, &rp)
	return rp, err
}

// RawProfile converts to a naive RawProfile
func (p TCPoolProfile) RawProfile() (RawProfile, error) {
	var rp RawProfile
	err := remarshalJSON(p, &rp)
	return rp, err
}

// DirPoolProfile wraps a Profile for a Directional Pool
type DirPoolProfile struct {
	Context         ProfileSchema `json:"@context"`
	Description     string        `json:"description"`
	ConflictResolve string        `json:"conflictResolve,omitempty"`
	RDataInfo       []DPRDataInfo `json:"rdataInfo"`
	NoResponse      DPRDataInfo   `json:"noResponse"`
}

// DPRDataInfo wraps the rdataInfo object of a DirPoolProfile response
type DPRDataInfo struct {
	AllNonConfigured bool    `json:"allNonConfigured,omitempty"`
	IPInfo           IPInfo  `json:"ipInfo,omitempty"`
	GeoInfo          GeoInfo `json:"geoInfo,omitempty"`
}

// IPInfo wraps the ipInfo object of a DPRDataInfo
type IPInfo struct {
	Name           string      `json:"name"`
	IsAccountLevel bool        `json:"isAccountLevel,omitempty"`
	Ips            []IPAddrDTO `json:"ips"`
}

// GeoInfo wraps the geoInfo object of a DPRDataInfo
type GeoInfo struct {
	Name           string   `json:"name"`
	IsAccountLevel bool     `json:"isAccountLevel,omitempty"`
	Codes          []string `json:"codes"`
}

// RDPoolProfile wraps a Profile for a Resource Distribution pool
type RDPoolProfile struct {
	Context     ProfileSchema `json:"@context"`
	Order       string        `json:"order"`
	Description string        `json:"description"`
}

// SBPoolProfile wraps a Profile for a SiteBacker pool
type SBPoolProfile struct {
	Context       ProfileSchema  `json:"@context"`
	Description   string         `json:"description"`
	RunProbes     bool           `json:"runProbes"`
	ActOnProbes   bool           `json:"actOnProbes"`
	Order         string         `json:"order,omitempty"`
	MaxActive     int            `json:"maxActive,omitempty"`
	MaxServed     int            `json:"maxServed,omitempty"`
	RDataInfo     []SBRDataInfo  `json:"rdataInfo"`
	BackupRecords []BackupRecord `json:"backupRecords"`
}

// SBRDataInfo wraps the rdataInfo object of a SBPoolProfile
type SBRDataInfo struct {
	State         string `json:"state"`
	RunProbes     bool   `json:"runProbes"`
	Priority      int    `json:"priority"`
	FailoverDelay int    `json:"failoverDelay,omitempty"`
	Threshold     int    `json:"threshold"`
	Weight        int    `json:"weight"`
}

// BackupRecord wraps the backupRecord objects of an SBPoolProfile response
type BackupRecord struct {
	RData         string `json:"rdata,omitempty"`
	FailoverDelay int    `json:"failoverDelay,omitempty"`
}

// TCPoolProfile wraps a Profile for a Traffic Controller pool
type TCPoolProfile struct {
	Context      ProfileSchema `json:"@context"`
	Description  string        `json:"description"`
	RunProbes    bool          `json:"runProbes"`
	ActOnProbes  bool          `json:"actOnProbes"`
	MaxToLB      int           `json:"maxToLB,omitempty"`
	RDataInfo    []SBRDataInfo `json:"rdataInfo"`
	BackupRecord *BackupRecord `json:"backupRecord,omitempty"`
}

// RRSet wraps an RRSet resource
type RRSet struct {
	OwnerName string     `json:"ownerName"`
	RRType    string     `json:"rrtype"`
	TTL       int        `json:"ttl"`
	RData     []string   `json:"rdata"`
	Profile   RawProfile `json:"profile,omitempty"`
}

// RRSetListDTO wraps a list of RRSet resources
type RRSetListDTO struct {
	ZoneName   string     `json:"zoneName"`
	Rrsets     []RRSet    `json:"rrsets"`
	Queryinfo  QueryInfo  `json:"queryInfo"`
	Resultinfo ResultInfo `json:"resultInfo"`
}

// RRSetKey collects the identifiers of a Zone
type RRSetKey struct {
	Zone string
	Type string
	Name string
}

// URI generates the URI for an RRSet
func (k RRSetKey) URI() string {
	uri := fmt.Sprintf("zones/%s/rrsets", k.Zone)
	if k.Type != "" {
		uri += fmt.Sprintf("/%v", k.Type)
		if k.Name != "" {
			uri += fmt.Sprintf("/%v", k.Name)
		}
	}
	return uri
}

// QueryURI generates the query URI for an RRSet and offset
func (k RRSetKey) QueryURI(offset int) string {
	// TODO: find a more appropriate place to set "" to "ANY"
	if k.Type == "" {
		k.Type = "ANY"
	}
	return fmt.Sprintf("%s?offset=%d", k.URI(), offset)
}

// AlertsURI generates the URI for an RRSet
func (k RRSetKey) AlertsURI() string {
	return fmt.Sprintf("%s/alerts", k.URI())
}

// AlertsQueryURI generates the alerts query URI for an RRSet with query
func (k RRSetKey) AlertsQueryURI(offset int) string {
	uri := k.AlertsURI()
	if offset != 0 {
		uri = fmt.Sprintf("%s?offset=%d", uri, offset)
	}
	return uri
}

// EventsURI generates the URI for an RRSet
func (k RRSetKey) EventsURI() string {
	return fmt.Sprintf("%s/events", k.URI())
}

// EventsQueryURI generates the events query URI for an RRSet with query
func (k RRSetKey) EventsQueryURI(query string, offset int) string {
	uri := k.EventsURI()
	if query != "" {
		return fmt.Sprintf("%s?sort=NAME&query=%s&offset=%d", uri, query, offset)
	}
	if offset != 0 {
		return fmt.Sprintf("%s?offset=%d", uri, offset)
	}
	return uri
}

// NotificationsURI generates the notifications URI for an RRSet
func (k RRSetKey) NotificationsURI() string {
	return fmt.Sprintf("%s/notifications", k.URI())
}

// NotificationsQueryURI generates the notifications query URI for an RRSet with query
func (k RRSetKey) NotificationsQueryURI(query string, offset int) string {
	uri := k.NotificationsURI()
	if query != "" {
		uri = fmt.Sprintf("%s?sort=NAME&query=%s&offset=%d", uri, query, offset)
	} else {
		uri = fmt.Sprintf("%s?offset=%d", uri, offset)
	}
	return uri
}

// ProbesURI generates the probes URI for an RRSet
func (k RRSetKey) ProbesURI() string {
	return fmt.Sprintf("%s/probes", k.URI())
}

// ProbesQueryURI generates the probes query URI for an RRSet with query
func (k RRSetKey) ProbesQueryURI(query string) string {
	uri := k.ProbesURI()
	if query != "" {
		uri = fmt.Sprintf("%s?sort=NAME&query=%s", uri, query)
	}
	return uri
}

// Select will list the zone rrsets, paginating through all available results
func (s *RRSetsService) Select(k RRSetKey) ([]RRSet, error) {
	// TODO: Sane Configuration for timeouts / retries
	maxerrs := 5
	waittime := 5 * time.Second

	rrsets := []RRSet{}
	errcnt := 0
	offset := 0

	for {
		reqRrsets, ri, res, err := s.SelectWithOffset(k, offset)
		if err != nil {
			if res.StatusCode >= 500 {
				errcnt = errcnt + 1
				if errcnt < maxerrs {
					time.Sleep(waittime)
					continue
				}
			}
			return rrsets, err
		}

		log.Printf("ResultInfo: %+v\n", ri)
		for _, rrset := range reqRrsets {
			rrsets = append(rrsets, rrset)
		}
		if ri.ReturnedCount+ri.Offset >= ri.TotalCount {
			return rrsets, nil
		}
		offset = ri.ReturnedCount + ri.Offset
		continue
	}
}

// SelectWithOffset requests zone rrsets by RRSetKey & optional offset
func (s *RRSetsService) SelectWithOffset(k RRSetKey, offset int) ([]RRSet, ResultInfo, *Response, error) {
	var rrsld RRSetListDTO

	uri := k.QueryURI(offset)
	res, err := s.client.get(uri, &rrsld)

	rrsets := []RRSet{}
	for _, rrset := range rrsld.Rrsets {
		rrsets = append(rrsets, rrset)
	}
	return rrsets, rrsld.Resultinfo, res, err
}

// Create creates an rrset with val
func (s *RRSetsService) Create(k RRSetKey, rrset RRSet) (*Response, error) {
	var ignored interface{}
	return s.client.post(k.URI(), rrset, &ignored)
}

// Update updates a RRSet with the provided val
func (s *RRSetsService) Update(k RRSetKey, val RRSet) (*Response, error) {
	var ignored interface{}
	return s.client.put(k.URI(), val, &ignored)
}

// Delete deletes an RRSet
func (s *RRSetsService) Delete(k RRSetKey) (*Response, error) {
	return s.client.delete(k.URI(), nil)
}
