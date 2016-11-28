package ov

import (
	"encoding/json"
	"fmt"
	"github.com/HewlettPackard/oneview-golang/rest"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/docker/machine/libmachine/log"
)

type LogicalInterconnectGroup struct {
	Category                string                   `json:"category,omitempty"`               // "category": "logical-interconnect-groups",
	Created                 string                   `json:"created,omitempty"`                // "created": "20150831T154835.250Z",
	Description             utils.Nstring            `json:"description,omitempty"`            // "description": "Logical Interconnect Group 1",
	ETAG                    string                   `json:"eTag,omitempty"`                   // "eTag": "1441036118675/8",
	EnclosureIndexes        []int                    `json:"enclosureIndexes,omitempty"`       // "enclosureIndexes": [1],
	EnclosureType           string                   `json:"enclosureType,omitempty"`          // "enclosureType": "C7000",
	EthernetSettings        *EthernetSettings        `json:"ethernetSettings,omitempty"`       // "ethernetSettings": {...},
	FabricUri               utils.Nstring            `json:"fabricUri,omitempty"`              // "fabricUri": "/rest/fabrics/9b8f7ec0-52b3-475e-84f4-c4eac51c2c20",
	InterconnectMapTemplate *InterconnectMapTemplate `json:"interconnectMapTemplate"`          // "interconnectMapTemplate": {...},
	InternalNetworkUris     []utils.Nstring          `json:"internalNetworkUris,omitempty"`    // "internalNetworkUris": []
	Modified                string                   `json:"modified,omitempty"`               // "modified": "20150831T154835.250Z",
	Name                    string                   `json:"name"`                             // "name": "Logical Interconnect Group1",
	QosConfiguration        *QosConfiguration        `json:"qosConfiguration,omitempty"`       // "qosConfiguration": {},
	RedundancyType          string                   `json:"redundancyType,omitempty"`         // "redundancyType": "HighlyAvailable"
	SnmpConfiguration       *SnmpConfiguration       `json:"snmpConfiguration,omitempty"`      // "snmpConfiguration": {...}
	StackingHealth          string                   `json:"stackingHealth,omitempty"`         //"stackingHealth": "Connected",
	StackingMode            string                   `json:"stackingMode,omitempty"`           //"stackingMode": "Enclosure",
	State                   string                   `json:"state,omitempty"`                  // "state": "Normal",
	Status                  string                   `json:"status,omitempty"`                 // "status": "Critical",
	TelemetryConfiguration  *TelemetryConfiguration  `json:"telemetryConfiguration,omitempty"` // "telemetryConfiguration": {...},
	Type                    string                   `json:"type"`                             // "type": "logical-interconnect-groupsV3",
	UplinkSets              []UplinkSet              `json:"uplinkSets,omitempty"`             // "uplinkSets": {...},
	URI                     utils.Nstring            `json:"uri,omitempty"`                    // "uri": "/rest/logical-interconnect-groups/e2f0031b-52bd-4223-9ac1-d91cb519d548",
}

type EthernetSettings struct {
	Category                    utils.Nstring `json:"category,omitempty"`                    // "category": null,
	Created                     string        `json:"created,omitempty"`                     // "created": "20150831T154835.250Z",
	DependentResourceUri        utils.Nstring `json:"dependentResourceUri,omitempty"`        // dependentResourceUri": "/rest/logical-interconnect-groups/b7b144e9-1f5e-4d52-8534-2e39280f9e86",
	Description                 utils.Nstring `json:"description,omitempty,omitempty"`       // "description": "Ethernet Settings",
	ETAG                        string        `json:"eTag,omitempty"`                        // "eTag": "1441036118675/8",
	EnableFastMacCacheFailover  *bool         `json:"enableFastMacCacheFailover,omitempty"`  //"enableFastMacCacheFailover": false,
	EnableIgmpSnooping          *bool         `json:"enableIgmpSnooping,omitempty"`          // "enableIgmpSnooping": false,
	EnableNetworkLoopProtection *bool         `json:"enableNetworkLoopProtection,omitempty"` // "enableNetworkLoopProtection": false,
	EnablePauseFloodProtection  *bool         `json:"enablePauseFloodProtection,omitempty"`  // "enablePauseFloodProtection": false,
	EnableRichTLV               *bool         `json:"enableRichTLV,omitempty"`               // "enableRichTLV": false,
	ID                          string        `json:"id,omitempty"`                          //"id": "0c398238-2d35-48eb-9eb5-7560d59f94b3",
	IgmpIdleTimeoutInterval     int           `json:"igmpIdleTimeoutInterval,omitempty"`     // "igmpIdleTimeoutInterval": 260,
	InterconnectType            string        `json:"interconnectType,omitempty"`            // "interconnectType": "Ethernet",
	MacRefreshInterval          int           `json:"macRefreshInterval,omitempty"`          // "macRefreshInterval": 5,
	Modified                    string        `json:"modified,omitempty"`                    // "modified": "20150831T154835.250Z",
	Name                        string        `json:"name,omitempty"`                        // "name": "ethernetSettings 1",
	State                       string        `json:"state,omitempty"`                       // "state": "Normal",
	Status                      string        `json:"status,omitempty"`                      // "status": "Critical",
	Type                        string        `json:"type,omitempty"`                        // "EthernetInterconnectSettingsV3",
	URI                         utils.Nstring `json:"uri,omitempty"`                         // "uri": "/rest/logical-interconnect-groups/b7b144e9-1f5e-4d52-8534-2e39280f9e86/ethernetSettings"
}

type InterconnectMapTemplate struct {
	InterconnectMapEntryTemplates []InterconnectMapEntryTemplate `json:"interconnectMapEntryTemplates"` // "interconnectMapEntryTemplates": {...},
}

type InterconnectMapEntryTemplate struct {
	EnclosureIndex               int             `json:"enclosureIndex,omitempty"`               // "enclosureIndex": 1,
	LogicalDownlinkUri           utils.Nstring   `json:"logicalDownlinkUri,omitempty"`           // "logicalDownlinkUri": "/rest/logical-downlinks/5b33fec1-63e8-40e1-9e3d-3af928917b2f",
	LogicalLocation              LogicalLocation `json:"logicalLocation,omitempty"`              // "logicalLocation": {...},
	PermittedInterconnectTypeUri utils.Nstring   `json:"permittedInterconnectTypeUri,omitempty"` //"permittedSwitchTypeUri": "/rest/switch-types/a2bc8f42-8bb8-4560-b80f-6c3c0e0d66e0",
}

type LogicalLocation struct {
	LocationEntries []LocationEntry `json:"locationEntries,omitempty"` // "locationEntries": {...}
}

type LocationEntry struct {
	RelativeValue int    `json:"relativeValue,omitempty"` //"relativeValue": 2,
	Type          string `json:"type,omitempty"`          //"type": "StackingMemberId",
}

type QosConfiguration struct {
	ActiveQosConfig          ActiveQosConfig           `json:"activeQosConfig,omitempty"`          //"activeQosConfig": {...},
	Category                 string                    `json:"category,omitempty"`                 // "category": "qos-aggregated-configuration",
	Created                  string                    `json:"created,omitempty"`                  // "created": "20150831T154835.250Z",
	Description              utils.Nstring             `json:"description,omitempty,omitempty"`    // "description": null,
	ETAG                     string                    `json:"eTag,omitempty"`                     // "eTag": "1441036118675/8",
	InactiveFCoEQosConfig    *InactiveFCoEQosConfig    `json:"inactiveFCoEQosConfig,omitempty"`    // "inactiveFCoEQosConfig": {...},
	InactiveNonFCoEQosConfig *InactiveNonFCoEQosConfig `json:"inactiveNonFCoEQosConfig,omitempty"` // "inactiveNonFCoEQosConfig": {...},
	Modified                 string                    `json:"modified,omitempty"`                 // "modified": "20150831T154835.250Z",
	Name                     string                    `json:"name,omitempty"`                     // "name": "Qos Config 1",
	State                    string                    `json:"state,omitempty"`                    // "state": "Normal",
	Status                   string                    `json:"status,omitempty"`                   // "status": "Critical",
	Type                     string                    `json:"type,omitempty"`                     // "qos-aggregated-configuration",
	URI                      utils.Nstring             `json:"uri,omitempty"`                      // "uri": null
}

type ActiveQosConfig struct {
	Category                   utils.Nstring          `json:"category,omitempty"`                   // "category": "null",
	ConfigType                 string                 `json:"configType,omitempty"`                 // "configType": "CustomWithFCoE",
	Created                    string                 `json:"created,omitempty"`                    // "created": "20150831T154835.250Z",
	Description                utils.Nstring          `json:"description,omitempty,omitempty"`      // "description": "Ethernet Settings",
	DownlinkClassificationType string                 `json:"downlinkClassificationType,omitempty"` //"downlinkClassifcationType": "DOT1P_AND_DSCP",
	ETAG                       string                 `json:"eTag,omitempty"`                       // "eTag": "1441036118675/8",
	Modified                   string                 `json:"modified,omitempty"`                   // "modified": "20150831T154835.250Z",
	Name                       string                 `json:"name,omitempty"`                       // "name": "active QOS Config 1",
	QosTrafficClassifiers      []QosTrafficClassifier `json:"qosTrafficClassifiers"`                // "qosTrafficClassifiers": {...},
	State                      string                 `json:"state,omitempty"`                      // "state": "Normal",
	Status                     string                 `json:"status,omitempty"`                     // "status": "Critical",
	Type                       string                 `json:"type,omitempty"`                       // "type": "QosConfiguration",
	UplinkClassificationType   string                 `json:"uplinkClassificationType,omitempty"`   // "uplinkClassificationType": "DOT1P"
	URI                        utils.Nstring          `json:"uri,omitempty"`                        // "uri": null
}

type InactiveFCoEQosConfig struct {
	Category                   utils.Nstring          `json:"category,omitempty"`                   // "category": "null",
	ConfigType                 string                 `json:"configType,omitempty"`                 // "configType": "CustomWithFCoE",
	Created                    string                 `json:"created,omitempty"`                    // "created": "20150831T154835.250Z",
	Description                utils.Nstring          `json:"description,omitempty,omitempty"`      // "description": "Ethernet Settings",
	DownlinkClassificationType string                 `json:"downlinkClassificationType,omitempty"` //"downlinkClassifcationType": "DOT1P_AND_DSCP",
	ETAG                       string                 `json:"eTag,omitempty"`                       // "eTag": "1441036118675/8",
	Modified                   string                 `json:"modified,omitempty"`                   // "modified": "20150831T154835.250Z",
	Name                       string                 `json:"name,omitempty"`                       // "name": "active QOS Config 1",
	QosTrafficClassifiers      []QosTrafficClassifier `json:"qosTrafficClassifiers,omitempty"`      // "qosTrafficClassifiers": {...},
	State                      string                 `json:"state,omitempty"`                      // "state": "Normal",
	Status                     string                 `json:"status,omitempty"`                     // "status": "Critical",
	Type                       string                 `json:"type,omitempty"`                       // "type": "QosConfiguration",
	UplinkClassificationType   string                 `json:"uplinkClassificationType,omitempty"`   // "uplinkClassificationType": "DOT1P"
	URI                        utils.Nstring          `json:"uri,omitempty"`                        // "uri": null
}

type InactiveNonFCoEQosConfig struct {
	Category                   utils.Nstring          `json:"category,omitempty"`                   // "category": "null",
	ConfigType                 string                 `json:"configType,omitempty"`                 // "configType": "CustomWithFCoE",
	Created                    string                 `json:"created,omitempty"`                    // "created": "20150831T154835.250Z",
	Description                utils.Nstring          `json:"description,omitempty,omitempty"`      // "description": "Ethernet Settings",
	DownlinkClassificationType string                 `json:"downlinkClassificationType,omitempty"` //"downlinkClassifcationType": "DOT1P_AND_DSCP",
	ETAG                       string                 `json:"eTag,omitempty"`                       // "eTag": "1441036118675/8",
	Modified                   string                 `json:"modified,omitempty"`                   // "modified": "20150831T154835.250Z",
	Name                       string                 `json:"name,omitempty"`                       // "name": "active QOS Config 1",
	QosTrafficClassifiers      []QosTrafficClassifier `json:"qosTrafficClassifiers,omitempty"`      // "qosTrafficClassifiers": {...},
	State                      string                 `json:"state,omitempty"`                      // "state": "Normal",
	Status                     string                 `json:"status,omitempty"`                     // "status": "Critical",
	Type                       string                 `json:"type,omitempty"`                       // "type": "QosConfiguration",
	UplinkClassificationType   string                 `json:"uplinkClassificationType,omitempty"`   // "uplinkClassificationType": "DOT1P"
	URI                        utils.Nstring          `json:"uri,omitempty"`                        // "uri": null
}

type QosTrafficClassifier struct {
	QosClassificationMapping *QosClassificationMap `json:"qosClassificationMapping"`  // "qosClassificationMapping": {...},
	QosTrafficClass          QosTrafficClass       `json:"qosTrafficClass,omitempty"` // "qosTrafficClass": {...},
}

type QosClassificationMap struct {
	Dot1pClassMapping []int    `json:"dot1pClassMapping"` // "dot1pClassMapping": [3],
	DscpClassMapping  []string `json:"dscpClassMapping"`  // "dscpClassMapping": [],
}

type QosTrafficClass struct {
	BandwidthShare   string `json:"bandwidthShare,omitempty"` // "bandwidthShare": "fcoe",
	ClassName        string `json:"className"`                // "className": "FCoE lossless",
	EgressDot1pValue int    `json:"egressDot1pValue"`         // "egressDot1pValue": 3,
	Enabled          *bool  `json:"enabled,omitempty"`        // "enabled": true,
	MaxBandwidth     int    `json:"maxBandwidth"`             // "maxBandwidth": 100,
	RealTime         *bool  `json:"realTime,omitempty"`       // "realTime": true,
}

//TODO SNMPConfiguration
type SnmpConfiguration struct {
	Category         utils.Nstring     `json:"category,omitempty"`         // "category": "snmp-configuration",
	Created          string            `json:"created,omitempty"`          // "created": "20150831T154835.250Z",
	Description      utils.Nstring     `json:"description,omitempty"`      // "description": null,
	ETAG             string            `json:"eTag,omitempty"`             // "eTag": "1441036118675/8",
	Enabled          *bool             `json:"enabled,omitempty"`          // "enabled": true,
	Modified         string            `json:"modified,omitempty"`         // "modified": "20150831T154835.250Z",
	Name             string            `json:"name,omitempty"`             // "name": "Snmp Config",
	ReadCommunity    string            `json:"readCommunity,omitempty"`    // "readCommunity": "public",
	SnmpAccess       []string          `json:"snmpAccess,omitempty"`       // "snmpAccess": [],
	State            string            `json:"state,omitempty"`            // "state": "Normal",
	Status           string            `json:"status,omitempty"`           // "status": "Critical",
	SystemContact    string            `json:"systemContact,omitempty"`    // "systemContact": "",
	TrapDestinations []TrapDestination `json:"trapDestinations,omitempty"` // "trapDestinations": {...}
	Type             string            `json:"type,omitempty"`             // "type": "snmp-configuration",
	URI              utils.Nstring     `json:"uri,omitempty"`              // "uri": null
}

type TrapDestination struct {
	CommunityString    string   `json:"communityString,omitempty"`    //"communityString": "public",
	EnetTrapCategories []string `json:"enetTrapCategories,omitempty"` //"enetTrapCategories": ["PortStatus", "Other"],
	FcTrapCategories   []string `json:"fcTrapCategories,omitempty"`   //"fcTrapCategories": ["PortStatus", "Other"]
	TrapDestination    string   `json:"trapDestination,omitempty"`    //"trapDestination": "127.0.0.1",
	TrapFormat         string   `json:"trapFormat,omitempty"`         //"trapFormat", "SNMPv1",
	TrapSeverities     []string `json:"trapSeverities,omitempty"`     //"trapSeverities": "Info",
	VcmTrapCategories  []string `json:"vcmTrapCategories,omitempty"`  // "vcmTrapCategories": ["Legacy"],
}

type TelemetryConfiguration struct {
	Category        string        `json:"category,omitempty"`        // "category": "telemetry-configuration",
	Created         string        `json:"created,omitempty"`         // "created": "20150831T154835.250Z",
	Description     utils.Nstring `json:"description,omitempty"`     // "description": null,
	ETAG            string        `json:"eTag,omitempty"`            // "eTag": "1441036118675/8",
	EnableTelemetry *bool         `json:"enableTelemetry,omitempty"` // "enableTelemetry": false,
	Modified        string        `json:"modified,omitempty"`        // "modified": "20150831T154835.250Z",
	Name            string        `json:"name,omitempty"`            // "name": "telemetry configuration",
	SampleCount     int           `json:"sampleCount,omitempty"`     // "sampleCount": 12
	SampleInterval  int           `json:"sampleInterval,omitempty"`  // "sampleInterval": 300,
	State           string        `json:"state,omitempty"`           // "state": "Normal",
	Status          string        `json:"status,omitempty"`          // "status": "Critical",
	Type            string        `json:"type,omitempty"`            // "type": "telemetry-configuration",
	URI             utils.Nstring `json:"uri,omitempty"`             // "uri": null
}

type UplinkSet struct {
	EthernetNetworkType    string                  `json:"ethernetNetworkType,omitempty"` // "ethernetNetworkType": "Tagged",
	LacpTimer              string                  `json:"lacpTimer,omitempty"`           // "lacpTimer": "Long",
	LogicalPortConfigInfos []LogicalPortConfigInfo `json:"logicalPortConfigInfos"`        // "logicalPortConfigInfos": {...},
	Mode                   string                  `json:"mode,omitempty"`                // "mode": "Auto",
	Name                   string                  `json:"name,omitempty"`                // "name": "Uplink 1",
	NativeNetworkUri       utils.Nstring           `json:"nativeNetworkUri,omitempty"`    // "nativeNetworkUri": null,
	NetworkType            string                  `json:"networkType,omitempty"`         // "networkType": "Ethernet",
	NetworkUris            []utils.Nstring         `json:"networkUris"`                   // "networkUris": ["/rest/ethernet-networks/f1e38895-721b-4204-8395-ae0caba5e163"]
	PrimaryPort            *LogicalLocation        `json:"primaryPort,omitempty"`         // "primaryPort": {...},
	Reachability           string                  `json:"reachability,omitempty"`        // "reachability": "Reachable",
}

type LogicalPortConfigInfo struct {
	DesiredSpeed    string          `json:"desiredSpeed,omitempty"`    // "desiredSpeed": "Auto",
	LogicalLocation LogicalLocation `json:"logicalLocation,omitempty"` // "logicalLocation": {...},
}

type LogicalInterconnectGroupList struct {
	Total       int                        `json:"total,omitempty"`       // "total": 1,
	Count       int                        `json:"count,omitempty"`       // "count": 1,
	Start       int                        `json:"start,omitempty"`       // "start": 0,
	PrevPageURI utils.Nstring              `json:"prevPageUri,omitempty"` // "prevPageUri": null,
	NextPageURI utils.Nstring              `json:"nextPageUri,omitempty"` // "nextPageUri": null,
	URI         utils.Nstring              `json:"uri,omitempty"`         // "uri": "/rest/server-profiles?filter=connectionTemplateUri%20matches%7769cae0-b680-435b-9b87-9b864c81657fsort=name:asc"
	Members     []LogicalInterconnectGroup `json:"members,omitempty"`     // "members":[]
}

func (c *OVClient) GetLogicalInterconnectGroupByName(name string) (LogicalInterconnectGroup, error) {
	var (
		logicalInterconnectGroup LogicalInterconnectGroup
	)
	logicalInterconnectGroups, err := c.GetLogicalInterconnectGroups(fmt.Sprintf("name matches '%s'", name), "name:asc")
	if logicalInterconnectGroups.Total > 0 {
		return logicalInterconnectGroups.Members[0], err
	} else {
		return logicalInterconnectGroup, err
	}
}

func (c *OVClient) GetLogicalInterconnectGroupByUri(uri utils.Nstring) (LogicalInterconnectGroup, error) {
	var (
		lig LogicalInterconnectGroup
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	data, err := c.RestAPICall(rest.GET, uri.String(), nil)
	if err != nil {
		return lig, err
	}
	log.Debugf("GetLogicalInterconnectGroup %s", data)
	if err := json.Unmarshal([]byte(data), &lig); err != nil {
		return lig, err
	}
	return lig, nil
}

func (c *OVClient) GetLogicalInterconnectGroups(filter string, sort string) (LogicalInterconnectGroupList, error) {
	var (
		uri                       = "/rest/logical-interconnect-groups"
		q                         map[string]interface{}
		logicalInterconnectGroups LogicalInterconnectGroupList
	)
	q = make(map[string]interface{})
	if len(filter) > 0 {
		q["filter"] = filter
	}

	if sort != "" {
		q["sort"] = sort
	}

	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())
	// Setup query
	if len(q) > 0 {
		c.SetQueryString(q)
	}
	data, err := c.RestAPICall(rest.GET, uri, nil)
	if err != nil {
		return logicalInterconnectGroups, err
	}

	log.Debugf("GetLogicalInterconnectGroups %s", data)
	if err := json.Unmarshal([]byte(data), &logicalInterconnectGroups); err != nil {
		return logicalInterconnectGroups, err
	}
	return logicalInterconnectGroups, nil
}

func (c *OVClient) CreateLogicalInterconnectGroup(logicalInterconnectGroup LogicalInterconnectGroup) error {
	log.Infof("Initializing creation of logicalInterconnectGroup for %s.", logicalInterconnectGroup.Name)
	var (
		uri = "/rest/logical-interconnect-groups"
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()

	log.Debugf("REST : %s \n %+v\n", uri, logicalInterconnectGroup)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.POST, uri, logicalInterconnectGroup)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting new logical interconnect group request: %s", err)
		return err
	}

	log.Debugf("Response New LogicalInterconnectGroup %s", data)
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		t.TaskIsDone = true
		log.Errorf("Error with task un-marshal: %s", err)
		return err
	}

	err = t.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (c *OVClient) DeleteLogicalInterconnectGroup(name string) error {
	var (
		logicalInterconnectGroup LogicalInterconnectGroup
		err                      error
		t                        *Task
		uri                      string
	)

	logicalInterconnectGroup, err = c.GetLogicalInterconnectGroupByName(name)
	if err != nil {
		return err
	}
	if logicalInterconnectGroup.Name != "" {
		t = t.NewProfileTask(c)
		t.ResetTask()
		log.Debugf("REST : %s \n %+v\n", logicalInterconnectGroup.URI, logicalInterconnectGroup)
		log.Debugf("task -> %+v", t)
		uri = logicalInterconnectGroup.URI.String()
		if uri == "" {
			log.Warn("Unable to post delete, no uri found.")
			t.TaskIsDone = true
			return err
		}
		data, err := c.RestAPICall(rest.DELETE, uri, nil)
		if err != nil {
			log.Errorf("Error submitting delete logicalInterconnectGroup request: %s", err)
			t.TaskIsDone = true
			return err
		}

		log.Debugf("Response delete logicalInterconnectGroup %s", data)
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			t.TaskIsDone = true
			log.Errorf("Error with task un-marshal: %s", err)
			return err
		}
		err = t.Wait()
		if err != nil {
			return err
		}
		return nil
	} else {
		log.Infof("LogicalInterconnectGroup could not be found to delete, %s, skipping delete ...", name)
	}
	return nil
}

func (c *OVClient) UpdateLogicalInterconnectGroup(logicalInterconnectGroup LogicalInterconnectGroup) error {
	log.Infof("Initializing update of logicalInterConnectGroup for %s.", logicalInterconnectGroup.Name)
	var (
		uri = logicalInterconnectGroup.URI.String()
		t   *Task
	)
	// refresh login
	c.RefreshLogin()
	c.SetAuthHeaderOptions(c.GetAuthHeaderMap())

	t = t.NewProfileTask(c)
	t.ResetTask()
	log.Debugf("REST : %s \n %+v\n", uri, logicalInterconnectGroup)
	log.Debugf("task -> %+v", t)
	data, err := c.RestAPICall(rest.PUT, uri, logicalInterconnectGroup)
	if err != nil {
		t.TaskIsDone = true
		log.Errorf("Error submitting update logicalInterConnectGroup request: %s", err)
		return err
	}

	log.Debugf("Response update LogicalInterConnectGroup %s", data)
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		t.TaskIsDone = true
		log.Errorf("Error with task un-marshal: %s", err)
		return err
	}

	err = t.Wait()
	if err != nil {
		return err
	}

	return nil
}
