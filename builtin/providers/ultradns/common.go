package ultradns

import (
	"fmt"

	"github.com/Ensighten/udnssdk"
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
