package edgegrid

import (
	"strings"
)

// Domains is a representation of the Akamai GTM
// domains response available at:
// http://apibase.com/config-gtm/v1/domains
type Domains struct {
	Domains []DomainSummary `json:"items"`
}

// DomainSummary is a representation of the Akamai GTM
// domain summary associated with each domain returned by
// the domains response at:
// http://apibase.com/config-gtm/v1/domains
type DomainSummary struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	LastModified string `json:"lastModified"`
}

// DomainResponse is a representation of the Akamai GTM
// response from a domain CREATEs and UPDATEs at:
// http://apibase.com/config-gtm/v1/domains/domain
type DomainResponse struct {
	Domain *Domain         `json:"resource"`
	Status *ResourceStatus `json:"status"`
}

// Domain is a representation of an Akamai GTM domain.
type Domain struct {
	Name                 string          `json:"name"`
	Status               *ResourceStatus `json:"status,omitempty"`
	Type                 string          `json:"type"`
	LastModified         string          `json:"lastModified,omitempty"`
	LastModifiedBy       string          `json:"lastModifiedBy,omitempty"`
	ModificationComments string          `json:"modificationComments,omitempty"`
	CIDRMaps             []interface{}   `json:"cidrMaps,omitempty"`
	Datacenters          []DataCenter    `json:"datacenters,omitempty"`
	Properties           []Property      `json:"properties,omitempty"`
	Links                []Link          `json:"links,omitempty"`
	GeographicMaps       []interface{}   `json:"geographicMaps,omitempty"`
	Resources            []interface{}   `json:"resources,omitempty"`
}

// ResourceStatus is a representation of an Akamai GTM status for
// a given resource, such as a domain.
type ResourceStatus struct {
	Message               string `json:"message"`
	ChangeID              string `json:"changeId"`
	PropagationStatus     string `json:"propagationStatus"`
	PropagationStatusDate string `json:"propagationStatusDate"`
	PassingValidation     bool   `json:"passingValidation"`
	Links                 []Link `json:"links"`
}

// DataCenters is a representation of an Akamai GTM datacenters
// response returned from:
// http://apibase.com/config-gtm/v1/domains/domain/datacenters
type DataCenters struct {
	Items []DataCenter `json:"items"`
}

// DataCenterResponse is a representation of an Akamai GTM datacenter
// response returned from datacenter CREATEs and UPDATEs at:
// http://apibase.com/config-gtm/v1/domains/domain/datacenters/4
type DataCenterResponse struct {
	DataCenter *DataCenter     `json:"resource"`
	Status     *ResourceStatus `json:"status"`
}

// DataCenter represents an Akamai GTM datacenter.
type DataCenter struct {
	City                 string             `json:"city"`
	CloneOf              int                `json:"cloneOf,omitempty"`
	CloudServerTargeting bool               `json:"cloudServerTargeting"`
	Continent            string             `json:"continent"`
	Country              string             `json:"country"`
	DataCenterID         int                `json:"datacenterId,omitempty"`
	DefaultLoadObject    *DefaultLoadObject `json:"defaultLoadObject,omitempty"`
	Latitude             float64            `json:"latitude"`
	Links                []Link             `json:"links,omitempty"`
	Longitude            float64            `json:"longitude"`
	Nickname             string             `json:"nickname"`
	StateOrProvince      string             `json:"stateOrProvince"`
	Virtual              bool               `json:"virtual"`
}

// Link represents the link objects embedded in Akamai GTM API
// response JSON.
type Link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

// DefaultLoadObject represents the default load object associated
// with an Akamai GTM datacenter.
type DefaultLoadObject struct {
	LoadObject     interface{} `json:"loadObject"`
	LoadObjectPort int64       `json:"loadObjectPort"`
	LoadServers    interface{} `json:"loadServers"`
}

// LoadObject represents the load object associated with an Akamai
// GTM datacenter.
type LoadObject struct {
	LoadObject           string `json:"loadObject"`
	LoadObjectPort       string `json:"loadObjectPort"`
	LoadServers          string `json:"loadServers"`
	Continent            string `json:"continent"`
	CloudServerTargeting bool   `json:"cloudServerTargeting"`
}

// PropertyResponse represents the Akamai GTM response returned
// by Akamai GTM API CREATEs and DELETEs against:
// http://apibase.com/config-gtm/v1/domains/domain/properties/property
type PropertyResponse struct {
	Property *Property       `json:"resource"`
	Status   *ResourceStatus `json:"status"`
}

// Properties represents the properties returned from:
// http://apibase.com/config-gtm/v1/domains/domain/properties/property
type Properties struct {
	Properties []Property `json:"items"`
}

func (props Properties) Len() int {
	return len(props.Properties)
}

func (props Properties) Less(i, j int) bool {
	return props.Properties[i].Name < props.Properties[j].Name
}

func (props Properties) Swap(i, j int) {
	props.Properties[i], props.Properties[j] = props.Properties[j], props.Properties[i]
}

// Property represents an Akamai GTM property.
type Property struct {
	BackupCname               string          `json:"backupCName,omitempty"`
	BackupIP                  string          `json:"backupIp,omitempty"`
	BalanceByDownloadScore    bool            `json:"balanceByDownloadScore,omitempty"`
	Cname                     string          `json:"cname,omitempty"`
	Comments                  string          `json:"comments,omitempty"`
	DynamicTTL                int             `json:"dynamicTTL,omitempty"`
	FailbackDelay             int             `json:"failbackDelay"`
	FailoverDelay             int             `json:"failoverDelay"`
	HandoutMode               string          `json:"handoutMode,omitempty"`
	HealthMax                 float64         `json:"healthMax,omitempty"`
	HealthMultiplier          float64         `json:"healthMultiplier,omitempty"`
	HealthThreshold           float64         `json:"healthThreshold,omitempty"`
	Ipv6                      bool            `json:"ipv6,omitempty"`
	LastModified              string          `json:"lastModified,omitempty"`
	Links                     []Link          `json:"links,omitempty"`
	LivenessTests             []LivenessTest  `json:"livenessTests,omitempty"`
	LoadImbalancePercentage   float64         `json:"loadImbalancePercentage,omitempty"`
	MapName                   interface{}     `json:"mapName,omitempty"`
	MaxUnreachablePenalty     interface{}     `json:"maxUnreachablePenalty,omitempty"`
	MxRecords                 []interface{}   `json:"mxRecords,omitempty"`
	Name                      string          `json:"name"`
	ScoreAggregationType      string          `json:"scoreAggregationType"`
	StaticTTL                 interface{}     `json:"staticTTL,omitempty"`
	StickinessBonusConstant   interface{}     `json:"stickinessBonusConstant,omitempty"`
	StickinessBonusPercentage interface{}     `json:"stickinessBonusPercentage,omitempty"`
	TrafficTargets            []TrafficTarget `json:"trafficTargets"`
	Type                      string          `json:"type"`
	UnreachableThreshold      interface{}     `json:"unreachableThreshold,omitempty"`
	UseComputedTargets        bool            `json:"useComputedTargets,omitempty"`
}

// LivenessTest represents a liveness test associated with an Akamai
// GTM property.
type LivenessTest struct {
	Name                          string  `json:"name"`
	HTTPError3xx                  bool    `json:"httpError3xx,omitempty"`
	HTTPError4xx                  bool    `json:"httpError4xx,omitempty"`
	HTTPError5xx                  bool    `json:"httpError5xx,omitempty"`
	TestInterval                  int64   `json:"testInterval,omitempty"`
	TestObject                    string  `json:"testObject,omitempty"`
	TestObjectPort                int64   `json:"testObjectPort,omitempty"`
	TestObjectProtocol            string  `json:"testObjectProtocol,omitempty"`
	TestObjectUsername            string  `json:"testObjectUsername,omitempty"`
	TestObjectPassword            string  `json:"testObjectPassword,omitempty"`
	TestTimeout                   float64 `json:"testTimeout,omitempty"`
	DisableNonstandardPortWarning bool    `json:"disableNonstandardPortWarning,omitempty"`
	RequestString                 string  `json:"requestString,omitempty"`
	ResponseString                string  `json:"responseString,omitempty"`
	SSLClientPrivateKey           string  `json:"sslClientPrivateKey,omitempty"`
	SSLCertificate                string  `json:"sslClientCertificate,omitempty"`
	HostHeader                    string  `json:"hostHeader,omitempty"`
}

// TrafficTarget represents a traffic target associated with an Akamai
// GTM property.
type TrafficTarget struct {
	DataCenterID int         `json:"datacenterId"`
	Enabled      bool        `json:"enabled"`
	HandoutCname interface{} `json:"handoutCName"`
	Name         interface{} `json:"name"`
	Servers      []string    `json:"servers"`
	Weight       float64     `json:"weight"`
}

// AkamaiError represents a non-successful HTTP response from the
// Akamai API.
type AkamaiError struct {
	Type         string `json:"type"`
	Title        string `json:"title"`
	Detail       string `json:"detail"`
	RequestBody  string `json:"-"`
	ResponseBody string `json:"-"`
}

func (a AkamaiError) Error() string {
	components := []string{a.Title, a.Detail, a.RequestBody, a.ResponseBody}
	return strings.Join(components, "\n")
}
