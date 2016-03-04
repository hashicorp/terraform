package dynect

// ZonesResponse is used for holding the data returned by a call to
// "https://api.dynect.net/REST/Zone/".
type ZonesResponse struct {
	ResponseBlock
	Data []string `json:"data"`
}

// ZoneResponse is used for holding the data returned by a call to
// "https://api.dynect.net/REST/Zone/ZONE_NAME".
type ZoneResponse struct {
	ResponseBlock
	Data ZoneDataBlock `json:"data"`
}

// Type ZoneDataBlock is used as a nested struct, which holds the data for a
// zone returned by a call to "https://api.dynect.net/REST/Zone/ZONE_NAME".
type ZoneDataBlock struct {
	Serial      int    `json:"serial"`
	SerialStyle string `json:"serial_style"`
	Zone        string `json:"zone"`
	ZoneType    string `json:"zone_type"`
}
