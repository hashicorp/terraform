package dynect

// Type AllRecordsResponse is a struct for holding a list of all URIs returned
// from an HTTP GET call to either https://api.dynect.net/REST/AllRecord/<zone>
// or https://api/dynect.net/REST/AllRecord/<zone>/<FQDN>/.
type AllRecordsResponse struct {
	ResponseBlock
	Data []string `json:"data"`
}

// Type RecordResponse is used to hold the information for a single DNS record
// returned from Dyn's DynECT API.
type RecordResponse struct {
	ResponseBlock
	Data BaseRecord `json:"data"`
}

/*
The base struct for record data returned from the Dyn REST API.

It should never be directly passed to the *Client.Do() function for marshaling
response data to. Instead, it should aid in the composition of a more-specific
response struct.
*/
type BaseRecord struct {
	FQDN       string    `json:"fqdn"`
	RecordId   int       `json:"record_id"`
	RecordType string    `json:"record_type"`
	TTL        int       `json:"ttl"`
	Zone       string    `json:"zone"`
	RData      DataBlock `json:"rdata"`
}

// Type DataBlock is nested within the BaseRecord struct, and is used for
// holding record information.
//
// The comment above each field indicates which record types you can expect
// the information to be provided.
type DataBlock struct {
	// A, AAAA
	Address string `json:"address,omitempty" bson:"address,omitempty"`

	// CERT, DNSKEY, DS, IPSECKEY, KEY, SSHFP
	Algorithm string `json:"algorithm,omitempty" bson:"algorithm,omitempty"`

	// LOC
	Altitude string `json:"altitude,omitempty" bson:"altitude,omitempty"`

	// CNAME
	CName string `json:"cname,omitempty" bson:"cname,omitempty"`

	// CERT
	Certificate string `json:"certificate,omitempty" bson:"algorithm,omitempty"`

	// DNAME
	DName string `json:"dname,omitempty" bson:"dname,omitempty"`

	// DHCID, DS
	Digest string `json:"digest,omitempty" bson:"digest,omitempty"`

	// DS
	DigestType string `json:"digtype,omitempty" bson:"digest_type,omitempty"`

	// KX, MX
	Exchange string `json:"exchange,omitempty" bson:"exchange,omitempty"`

	// SSHFP
	FPType string `json:"fptype,omitempty" bson:"fp_type,omitempty"`

	// SSHFP
	Fingerprint string `json:"fingerprint,omitempty" bson:"fingerprint,omitempty"`

	// DNSKEY, KEY, NAPTR
	Flags string `json:"flags,omitempty" bson:"flags,omitempty"`

	// CERT
	Format string `json:"format,omitempty" bson:"format,omitempty"`

	// IPSECKEY
	GatewayType string `json:"gatetype,omitempty" bson:"gateway_type,omitempty"`

	// LOC
	HorizPre string `json:"horiz_pre,omitempty" bson:"horiz_pre,omitempty"`

	// DS
	KeyTag string `json:"keytag,omitempty" bson:"keytag,omitempty"`

	// LOC
	Latitude string `json:"latitude,omitempty" bson:"latitude,omitempty"`

	// LOC
	Longitude string `json:"longitude,omitempty" bson:"longitude,omitempty"`

	// PX
	Map822 string `json:"map822,omitempty" bson:"map_822,omitempty"`

	// PX
	MapX400 string `json:"mapx400,omitempty" bson:"map_x400,omitempty"`

	// RP
	Mbox string `json:"mbox,omitempty" bson:"mbox,omitempty"`

	// NS
	NSDName string `json:"nsdname,omitempty" bson:"nsdname,omitempty"`

	// NSAP
	NSAP string `json:"nsap,omitempty" bson:"nsap,omitempty"`

	// NAPTR
	Order string `json:"order,omitempty" bson:"order,omitempty"`

	// SRV
	Port string `json:"port,omitempty" bson:"port,omitempty"`

	// IPSECKEY
	Precendence string `json:"precendence,omitempty" bson:"precendence,omitempty"`

	// KX, MX, NAPTR, PX
	Preference int `json:"preference,omitempty" bson:"preference,omitempty"`

	// SRV
	Priority int `json:"priority,omitempty" bson:"priority,omitempty"`

	// DNSKEY, KEY
	Protocol string `json:"protocol,omitempty" bson:"protocol,omitempty"`

	// PTR
	PTRDname string `json:"ptrdname,omitempty" bson:"ptrdname,omitempty"`

	// DNSKEY, IPSECKEY, KEY
	PublicKey string `json:"public_key,omitempty" bson:"public_key,omitempty"`

	// NAPTR
	Regexp string `json:"regexp,omitempty" bson:"regexp,omitempty"`

	// NAPTR
	Replacement string `json:"replacement,omitempty" bson:"replacement,omitempty"`

	// SOA
	RName string `json:"rname,omitempty" bson:"rname,omitempty"`

	// NAPTR
	Services string `json:"services,omitempty" bson:"services,omitempty"`

	// LOC
	Size string `json:"size,omitempty" bson:"size,omitempty"`

	// CERT
	Tag string `json:"tag,omitempty" bson:"tag,omitempty"`

	// SRV
	Target string `json:"target,omitempty" bson:"target,omitempty"`

	// RP
	TxtDName string `json:"txtdname,omitempty" bson:"txtdname,omitempty"`

	// SPF, TXT
	TxtData string `json:"txtdata,omitempty" bson:"txtdata,omitempty"`

	// LOC
	Version string `json:"version,omitempty" bson:"version,omitempty"`

	// LOC
	VertPre string `json:"vert_pre,omitempty" bson:"vert_pre,omitempty"`

	// SRV
	Weight string `json:"weight,omitempty" bson:"weight,omitempty"`
}
