package akamai

import "github.com/akamai-open/AkamaiOPEN-edgegrid-golang/edgegrid"

type Config struct {
	ConfigDNSV1Service *edgegrid.ConfigDNSV1Service
	PapiV0Service      *edgegrid.PapiV0Service
}
