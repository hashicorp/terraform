// (C) Copyright 2016 Hewlett Packard Enterprise Development LP
//
// Licensed under the Apache License, Version 2.0 (the "License");
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied. See the License for the
// specific language governing permissions and limitations under the License.

package oneview

import (
	"fmt"
	"github.com/HewlettPackard/oneview-golang/ov"
	"github.com/HewlettPackard/oneview-golang/utils"
	"github.com/hashicorp/terraform/helper/schema"
	"reflect"
	"strconv"
)

func resourceLogicalInterconnectGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceLogicalInterconnectGroupCreate,
		Read:   resourceLogicalInterconnectGroupRead,
		Update: resourceLogicalInterconnectGroupUpdate,
		Delete: resourceLogicalInterconnectGroupDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "logical-interconnect-groupV3",
			},
			"interconnect_map_entry_template": {
				Optional: true,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bay_number": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"interconnect_type_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"enclosure_index": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  1,
						},
					},
				},
			},
			"uplink_set": {
				Optional: true,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"network_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Ethernet",
						},
						"ethernet_network_type": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"logical_port_config": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"desired_speed": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  "Auto",
									},
									"port_num": {
										Type:     schema.TypeSet,
										Required: true,
										Elem:     &schema.Schema{Type: schema.TypeInt},
										Set: func(a interface{}) int {
											return a.(int)
										},
									},
									"bay_num": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"enclosure_num": {
										Type:     schema.TypeInt,
										Optional: true,
										Default:  1,
									},
									"primary_port": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
								},
							},
						},
						"mode": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Auto",
						},
						"network_uris": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"lacp_timer": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Short",
						},
						"native_network_uri": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"internal_network_uris": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"telemetry_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "telemetry-configuration",
						},
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"sample_count": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  12,
						},
						"sample_interval": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  300,
						},
					},
				},
			},
			"snmp_configuration": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "snmp-configuration",
						},
						"read_community": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "public",
						},
						"system_contact": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"snmp_access": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},
						"trap_destination": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"community_string": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"enet_trap_categories": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Set:      schema.HashString,
									},
									"fc_trap_categories": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Set:      schema.HashString,
									},
									"vcm_trap_categories": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Set:      schema.HashString,
									},
									"trap_destination": {
										Type:     schema.TypeString,
										Required: true,
									},
									"trap_format": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  "SNMPv1",
									},
									"trap_severities": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Set:      schema.HashString,
									},
								},
							},
						},
					},
				},
			},
			"interconnect_settings": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "EthernetInterconnectSettingsV3",
						},
						"fast_mac_cache_failover": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"igmp_snooping": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"network_loop_protection": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"pause_flood_protection": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
						},
						"rich_tlv": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"igmp_timeout_interval": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  260,
						},
						"mac_refresh_interval": {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  5,
						},
					},
				},
			},
			"quality_of_service": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "qos-aggregated-configuration",
						},
						"active_qos_config_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "QosConfiguration",
						},
						"config_type": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "Passthrough",
						},
						"uplink_classification_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"downlink_classification_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"qos_traffic_class": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"enabled": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  true,
									},
									"egress_dot1p_value": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"real_time": {
										Type:     schema.TypeBool,
										Optional: true,
										Default:  false,
									},
									"bandwidth_share": {
										Type:     schema.TypeString,
										Required: true,
									},
									"max_bandwidth": {
										Type:     schema.TypeInt,
										Required: true,
									},
									"qos_classification_map": {
										Type:     schema.TypeList,
										Required: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"dot1p_class_map": {
													Type:     schema.TypeSet,
													Optional: true,
													Elem:     &schema.Schema{Type: schema.TypeInt},
													Set: func(a interface{}) int {
														return a.(int)
													},
												},
												"dscp_class_map": {
													Type:     schema.TypeSet,
													Optional: true,
													Elem:     &schema.Schema{Type: schema.TypeString},
													Set:      schema.HashString,
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
			"created": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"modified": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"category": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"fabric_uri": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"eTag": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceLogicalInterconnectGroupCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	lig := ov.LogicalInterconnectGroup{
		Name: d.Get("name").(string),
		Type: d.Get("type").(string),
	}

	interconnectMapEntryTemplateCount := d.Get("interconnect_map_entry_template.#").(int)
	interconnectMapEntryTemplates := make([]ov.InterconnectMapEntryTemplate, 0)
	for i := 0; i < interconnectMapEntryTemplateCount; i++ {
		interconnectMapEntryTemplatePrefix := fmt.Sprintf("interconnect_map_entry_template.%d", i)
		interconnectTypeName := d.Get(interconnectMapEntryTemplatePrefix + ".interconnect_type_name").(string)
		interconnectType, err := config.ovClient.GetInterconnectTypeByName(interconnectTypeName)
		if err != nil {
			return err
		}
		if interconnectType.URI == "" {
			return fmt.Errorf("Could not find Interconnect Type from name: %s", interconnectTypeName)
		}

		enclosureLocation := ov.LocationEntry{
			RelativeValue: d.Get(interconnectMapEntryTemplatePrefix + ".enclosure_index").(int),
			Type:          "Enclosure",
		}
		locationEntries := make([]ov.LocationEntry, 0)
		locationEntries = append(locationEntries, enclosureLocation)

		bayLocation := ov.LocationEntry{
			RelativeValue: d.Get(interconnectMapEntryTemplatePrefix + ".bay_number").(int),
			Type:          "Bay",
		}
		locationEntries = append(locationEntries, bayLocation)
		logicalLocation := ov.LogicalLocation{
			LocationEntries: locationEntries,
		}
		interconnectMapEntryTemplates = append(interconnectMapEntryTemplates, ov.InterconnectMapEntryTemplate{
			LogicalLocation:              logicalLocation,
			EnclosureIndex:               d.Get(interconnectMapEntryTemplatePrefix + ".enclosure_index").(int),
			PermittedInterconnectTypeUri: interconnectType.URI,
		})
	}
	interconnectMapTemplate := ov.InterconnectMapTemplate{
		InterconnectMapEntryTemplates: interconnectMapEntryTemplates,
	}
	lig.InterconnectMapTemplate = &interconnectMapTemplate

	uplinkSetCount := d.Get("uplink_set.#").(int)
	uplinkSets := make([]ov.UplinkSet, 0)
	for i := 0; i < uplinkSetCount; i++ {
		uplinkSetPrefix := fmt.Sprintf("uplink_set.%d", i)
		uplinkSet := ov.UplinkSet{}
		if val, ok := d.GetOk(uplinkSetPrefix + ".name"); ok {
			uplinkSet.Name = val.(string)
		}
		if val, ok := d.GetOk(uplinkSetPrefix + ".network_type"); ok {
			uplinkSet.NetworkType = val.(string)
		}
		if val, ok := d.GetOk(uplinkSetPrefix + ".ethernet_network_type"); ok {
			uplinkSet.EthernetNetworkType = val.(string)
		}
		if val, ok := d.GetOk(uplinkSetPrefix + ".mode"); ok {
			uplinkSet.Mode = val.(string)
		}
		if val, ok := d.GetOk(uplinkSetPrefix + ".lacp_timer"); ok {
			uplinkSet.LacpTimer = val.(string)
		}
		if val, ok := d.GetOk(uplinkSetPrefix + ".native_network_uri"); ok {
			uplinkSet.NativeNetworkUri = utils.NewNstring(val.(string))
		}

		logicalPortCount := d.Get(uplinkSetPrefix + ".logical_port_config.#").(int)
		logicalPorts := make([]ov.LogicalPortConfigInfo, 0)
		for i := 0; i < logicalPortCount; i++ {
			logicalPortPrefix := fmt.Sprintf(uplinkSetPrefix+".logical_port_config.%d", i)
			rawPortLocations := d.Get(logicalPortPrefix + ".port_num").(*schema.Set).List()
			for _, raw := range rawPortLocations {
				logicalPort := ov.LogicalPortConfigInfo{}

				if val, ok := d.GetOk(logicalPortPrefix + ".desired_speed"); ok {
					logicalPort.DesiredSpeed = val.(string)
				}

				locationEntries := make([]ov.LocationEntry, 0)
				enclosureLocation := ov.LocationEntry{
					RelativeValue: d.Get(logicalPortPrefix + ".enclosure_num").(int),
					Type:          "Enclosure",
				}
				locationEntries = append(locationEntries, enclosureLocation)

				bayLocation := ov.LocationEntry{
					RelativeValue: d.Get(logicalPortPrefix + ".bay_num").(int),
					Type:          "Bay",
				}
				locationEntries = append(locationEntries, bayLocation)

				portLocation := ov.LocationEntry{
					RelativeValue: raw.(int),
					Type:          "Port",
				}
				locationEntries = append(locationEntries, portLocation)

				logicalLocation := ov.LogicalLocation{
					LocationEntries: locationEntries,
				}

				logicalPort.LogicalLocation = logicalLocation
				if _, ok := d.GetOk(logicalPortPrefix + ".primary_port"); ok {
					if uplinkSet.PrimaryPort == nil {
						uplinkSet.PrimaryPort = &logicalLocation
					}
				}

				logicalPorts = append(logicalPorts, logicalPort)
			}

		}
		uplinkSet.LogicalPortConfigInfos = logicalPorts

		rawNetUris := d.Get(uplinkSetPrefix + ".network_uris").(*schema.Set).List()
		netUris := make([]utils.Nstring, 0)
		for _, raw := range rawNetUris {
			netUris = append(netUris, utils.NewNstring(raw.(string)))
		}
		uplinkSet.NetworkUris = netUris

		uplinkSets = append(uplinkSets, uplinkSet)
	}

	lig.UplinkSets = uplinkSets

	rawInternalNetUris := d.Get("internal_network_uris").(*schema.Set).List()
	internalNetUris := make([]utils.Nstring, len(rawInternalNetUris))
	for i, raw := range rawInternalNetUris {
		internalNetUris[i] = utils.NewNstring(raw.(string))
	}
	lig.InternalNetworkUris = internalNetUris

	telemetryConfigPrefix := fmt.Sprintf("telemetry_configuration.0")
	telemetryConfiguration := ov.TelemetryConfiguration{}
	if val, ok := d.GetOk(telemetryConfigPrefix + ".sample_count"); ok {
		telemetryConfiguration.SampleCount = val.(int)
	}
	if val, ok := d.GetOk(telemetryConfigPrefix + ".sample_interval"); ok {
		telemetryConfiguration.SampleInterval = val.(int)
	}
	if val, ok := d.GetOk(telemetryConfigPrefix + ".enabled"); ok {
		enabled := val.(bool)
		telemetryConfiguration.EnableTelemetry = &enabled
	}
	if telemetryConfiguration != (ov.TelemetryConfiguration{}) {
		telemetryConfiguration.Type = d.Get(telemetryConfigPrefix + ".type").(string)
		lig.TelemetryConfiguration = &telemetryConfiguration
	}

	snmpConfigPrefix := fmt.Sprintf("snmp_configuration.0")
	snmpConfiguration := ov.SnmpConfiguration{}
	if val, ok := d.GetOk(snmpConfigPrefix + ".enabled"); ok {
		enabled := val.(bool)
		snmpConfiguration.Enabled = &enabled
	}
	if val, ok := d.GetOk(snmpConfigPrefix + ".read_community"); ok {
		snmpConfiguration.ReadCommunity = val.(string)
	}
	if val, ok := d.GetOk(snmpConfigPrefix + ".system_contact"); ok {
		snmpConfiguration.SystemContact = val.(string)
	}
	rawSnmpAccess := d.Get(snmpConfigPrefix + ".snmp_access").(*schema.Set).List()
	snmpAccess := make([]string, len(rawSnmpAccess))
	for i, raw := range rawSnmpAccess {
		snmpAccess[i] = raw.(string)
	}
	snmpConfiguration.SnmpAccess = snmpAccess

	trapDestinationCount := d.Get(snmpConfigPrefix + ".trap_destination.#").(int)
	trapDestinations := make([]ov.TrapDestination, 0, trapDestinationCount)
	for i := 0; i < trapDestinationCount; i++ {
		trapDestinationPrefix := fmt.Sprintf(snmpConfigPrefix+".trap_destination.%d", i)

		rawEnetTrapCategories := d.Get(trapDestinationPrefix + ".enet_trap_categories").(*schema.Set).List()
		enetTrapCategories := make([]string, len(rawEnetTrapCategories))
		for i, raw := range rawEnetTrapCategories {
			enetTrapCategories[i] = raw.(string)
		}

		rawFcTrapCategories := d.Get(trapDestinationPrefix + ".fc_trap_categories").(*schema.Set).List()
		fcTrapCategories := make([]string, len(rawFcTrapCategories))
		for i, raw := range rawFcTrapCategories {
			fcTrapCategories[i] = raw.(string)
		}

		rawVcmTrapCategories := d.Get(trapDestinationPrefix + ".vcm_trap_categories").(*schema.Set).List()
		vcmTrapCategories := make([]string, len(rawVcmTrapCategories))
		for i, raw := range rawVcmTrapCategories {
			vcmTrapCategories[i] = raw.(string)
		}

		rawTrapSeverities := d.Get(trapDestinationPrefix + ".trap_severities").(*schema.Set).List()
		trapSeverities := make([]string, len(rawTrapSeverities))
		for i, raw := range rawTrapSeverities {
			trapSeverities[i] = raw.(string)
		}

		trapDestination := ov.TrapDestination{
			TrapDestination:    d.Get(trapDestinationPrefix + ".trap_destination").(string),
			CommunityString:    d.Get(trapDestinationPrefix + ".community_string").(string),
			TrapFormat:         d.Get(trapDestinationPrefix + ".trap_format").(string),
			EnetTrapCategories: enetTrapCategories,
			FcTrapCategories:   fcTrapCategories,
			VcmTrapCategories:  vcmTrapCategories,
			TrapSeverities:     trapSeverities,
		}
		trapDestinations = append(trapDestinations, trapDestination)
	}
	if trapDestinationCount > 0 {
		snmpConfiguration.TrapDestinations = trapDestinations
	}

	if val, ok := d.GetOk(snmpConfigPrefix + ".type"); ok {
		snmpConfiguration.Type = val.(string)
		lig.SnmpConfiguration = &snmpConfiguration
	}

	interconnectSettingsPrefix := fmt.Sprintf("interconnect_settings.0")
	if val, ok := d.GetOk(interconnectSettingsPrefix + ".type"); ok {
		interconnectSettings := ov.EthernetSettings{}

		macFailoverEnabled := d.Get(interconnectSettingsPrefix + ".fast_mac_cache_failover").(bool)
		interconnectSettings.EnableFastMacCacheFailover = &macFailoverEnabled

		networkLoopProtectionEnabled := d.Get(interconnectSettingsPrefix + ".network_loop_protection").(bool)
		interconnectSettings.EnableNetworkLoopProtection = &networkLoopProtectionEnabled

		pauseFloodProtectionEnabled := d.Get(interconnectSettingsPrefix + ".pause_flood_protection").(bool)
		interconnectSettings.EnablePauseFloodProtection = &pauseFloodProtectionEnabled

		if val1, ok := d.GetOk(interconnectSettingsPrefix + ".rich_tlv"); ok {
			enabled := val1.(bool)
			interconnectSettings.EnableRichTLV = &enabled
		}

		if val1, ok := d.GetOk(interconnectSettingsPrefix + ".igmp_snooping"); ok {
			enabled := val1.(bool)
			interconnectSettings.EnableIgmpSnooping = &enabled
		}

		if val1, ok := d.GetOk(interconnectSettingsPrefix + ".igmp_timeout_interval"); ok {
			interconnectSettings.IgmpIdleTimeoutInterval = val1.(int)
		}

		if val1, ok := d.GetOk(interconnectSettingsPrefix + ".mac_refresh_interval"); ok {
			interconnectSettings.MacRefreshInterval = val1.(int)
		}

		interconnectSettings.Type = val.(string)
		lig.EthernetSettings = &interconnectSettings
	}

	qualityOfServicePrefix := fmt.Sprintf("quality_of_service.0")
	activeQosConfig := ov.ActiveQosConfig{}

	if val, ok := d.GetOk(qualityOfServicePrefix + ".config_type"); ok {
		activeQosConfig.ConfigType = val.(string)
	}

	if val, ok := d.GetOk(qualityOfServicePrefix + ".uplink_classification_type"); ok {
		activeQosConfig.UplinkClassificationType = val.(string)
	}

	if val, ok := d.GetOk(qualityOfServicePrefix + ".downlink_classification_type"); ok {
		activeQosConfig.DownlinkClassificationType = val.(string)
	}

	qosTrafficClassCount := d.Get(qualityOfServicePrefix + ".qos_traffic_class.#").(int)
	qosTrafficClassifiers := make([]ov.QosTrafficClassifier, 0, 1)
	for i := 0; i < qosTrafficClassCount; i++ {
		qosTrafficClassPrefix := fmt.Sprintf(qualityOfServicePrefix+".qos_traffic_class.%d", i)
		qosTrafficClassifier := ov.QosTrafficClassifier{}
		qosClassMap := ov.QosClassificationMap{}
		qosTrafficClass := ov.QosTrafficClass{}

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".name"); ok {
			qosTrafficClass.ClassName = val.(string)
		}
		classEnabled := d.Get(qosTrafficClassPrefix + ".enabled").(bool)
		qosTrafficClass.Enabled = &classEnabled

		realTimeEnabled := d.Get(qosTrafficClassPrefix + ".real_time").(bool)
		qosTrafficClass.RealTime = &realTimeEnabled

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".egress_dot1p_value"); ok {
			qosTrafficClass.EgressDot1pValue = val.(int)
		}

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".bandwidth_share"); ok {
			qosTrafficClass.BandwidthShare = val.(string)
		}

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".max_bandwidth"); ok {
			qosTrafficClass.MaxBandwidth = val.(int)
		}

		qosTrafficClassifier.QosTrafficClass = qosTrafficClass

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".qos_classification_map.0.dscp_class_map"); ok {
			rawDscpClassMapping := val.(*schema.Set).List()
			dscpClassMapping := make([]string, len(rawDscpClassMapping))
			for i, raw := range rawDscpClassMapping {
				dscpClassMapping[i] = raw.(string)
			}
			qosClassMap.DscpClassMapping = dscpClassMapping
		}

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".qos_classification_map.0.dot1p_class_map"); ok {
			rawDot1pClassMap := val.(*schema.Set).List()
			dot1pClassMap := make([]int, len(rawDot1pClassMap))
			for i, raw := range rawDot1pClassMap {
				dot1pClassMap[i] = raw.(int)
			}
			qosClassMap.Dot1pClassMapping = dot1pClassMap
		}

		qosTrafficClassifier.QosClassificationMapping = &qosClassMap

		qosTrafficClassifiers = append(qosTrafficClassifiers, qosTrafficClassifier)
	}
	activeQosConfig.QosTrafficClassifiers = qosTrafficClassifiers

	if val, ok := d.GetOk(qualityOfServicePrefix + ".active_qos_config_type"); ok {
		activeQosConfig.Type = val.(string)

		qualityOfService := ov.QosConfiguration{
			Type:            d.Get(qualityOfServicePrefix + ".type").(string),
			ActiveQosConfig: activeQosConfig,
		}

		lig.QosConfiguration = &qualityOfService
	}

	ligError := config.ovClient.CreateLogicalInterconnectGroup(lig)
	d.SetId(d.Get("name").(string))
	if ligError != nil {
		d.SetId("")
		return ligError
	}
	return resourceLogicalInterconnectGroupRead(d, meta)
}

func resourceLogicalInterconnectGroupRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	logicalInterconnectGroup, err := config.ovClient.GetLogicalInterconnectGroupByName(d.Id())
	if err != nil || logicalInterconnectGroup.URI.IsNil() {
		d.SetId("")
		return nil
	}

	d.Set("name", logicalInterconnectGroup.Name)
	d.Set("type", logicalInterconnectGroup.Type)
	d.Set("created", logicalInterconnectGroup.Created)
	d.Set("modified", logicalInterconnectGroup.Modified)
	d.Set("uri", logicalInterconnectGroup.URI.String())
	d.Set("status", logicalInterconnectGroup.Status)
	d.Set("category", logicalInterconnectGroup.Category)
	d.Set("state", logicalInterconnectGroup.State)
	d.Set("fabric_uri", logicalInterconnectGroup.FabricUri.String())
	d.Set("eTag", logicalInterconnectGroup.ETAG)
	d.Set("description", logicalInterconnectGroup.Description)
	d.Set("interconnect_settings.0.igmp_snooping", logicalInterconnectGroup.EthernetSettings.EnableIgmpSnooping)

	interconnectMapEntryTemplates := make([]map[string]interface{}, 0, len(logicalInterconnectGroup.InterconnectMapTemplate.InterconnectMapEntryTemplates))
	for _, interconnectMapEntryTemplate := range logicalInterconnectGroup.InterconnectMapTemplate.InterconnectMapEntryTemplates {
		interconnectType, err := config.ovClient.GetInterconnectTypeByUri(interconnectMapEntryTemplate.PermittedInterconnectTypeUri)
		if err != nil {
			return err
		}
		if interconnectType.Name == "" {
			return fmt.Errorf("Could not find interconnectType with URI %s", interconnectMapEntryTemplate.PermittedInterconnectTypeUri.String())
		}
		var bayNum int
		var enclosureIndex int
		if interconnectMapEntryTemplate.LogicalLocation.LocationEntries[0].Type == "Bay" {
			bayNum = interconnectMapEntryTemplate.LogicalLocation.LocationEntries[0].RelativeValue
			enclosureIndex = interconnectMapEntryTemplate.LogicalLocation.LocationEntries[1].RelativeValue
		} else {
			bayNum = interconnectMapEntryTemplate.LogicalLocation.LocationEntries[1].RelativeValue
			enclosureIndex = interconnectMapEntryTemplate.LogicalLocation.LocationEntries[0].RelativeValue
		}

		interconnectMapEntryTemplates = append(interconnectMapEntryTemplates, map[string]interface{}{
			"interconnect_type_name": interconnectType.Name,
			"bay_number":             bayNum,
			"enclosure_index":        enclosureIndex,
		})
	}

	interconnectMapEntryTemplateCount := d.Get("interconnect_map_entry_template.#").(int)
	for i := 0; i < interconnectMapEntryTemplateCount; i++ {
		currBayNum := d.Get("interconnect_map_entry_template." + strconv.Itoa(i) + ".bay_number")
		for j := 0; j < len(logicalInterconnectGroup.InterconnectMapTemplate.InterconnectMapEntryTemplates); j++ {
			if currBayNum == interconnectMapEntryTemplates[j]["bay_number"] {
				interconnectMapEntryTemplates[i], interconnectMapEntryTemplates[j] = interconnectMapEntryTemplates[j], interconnectMapEntryTemplates[i]
			}
		}
	}
	d.Set("interconnect_map_entry_template", interconnectMapEntryTemplates)

	uplinkSets := make([]map[string]interface{}, 0, len(logicalInterconnectGroup.UplinkSets))
	for i, uplinkSet := range logicalInterconnectGroup.UplinkSets {

		primaryPortEnclosure := 0
		primaryPortBay := 0
		primaryPortPort := 0

		if uplinkSet.PrimaryPort != nil {
			for _, primaryPortLocation := range uplinkSet.PrimaryPort.LocationEntries {
				if primaryPortLocation.Type == "Bay" {
					primaryPortBay = primaryPortLocation.RelativeValue
				}
				if primaryPortLocation.Type == "Enclosure" {
					primaryPortEnclosure = primaryPortLocation.RelativeValue
				}
				if primaryPortLocation.Type == "Port" {
					primaryPortPort = primaryPortLocation.RelativeValue
				}
			}
		}

		logicalPortConfigs := make([]map[string]interface{}, 0, len(uplinkSet.LogicalPortConfigInfos))
		for _, logicalPortConfigInfo := range uplinkSet.LogicalPortConfigInfos {
			portEnclosure := 0
			portBay := 0
			portPort := 0
			primaryPort := false
			for _, portLocation := range logicalPortConfigInfo.LogicalLocation.LocationEntries {
				if portLocation.Type == "Bay" {
					portBay = portLocation.RelativeValue
				}
				if portLocation.Type == "Enclosure" {
					portEnclosure = portLocation.RelativeValue
				}
				if portLocation.Type == "Port" {
					portPort = portLocation.RelativeValue
				}
			}
			if primaryPortEnclosure == portEnclosure && primaryPortBay == portBay && primaryPortPort == portPort {
				primaryPort = true
			}

			portPorts := make([]interface{}, 0)
			portPorts = append(portPorts, portPort)

			included := false
			for j, portConfig := range logicalPortConfigs {
				if portConfig["bay_num"] == portBay && portConfig["enclosure_num"] == portEnclosure {
					included = true
					portSet := logicalPortConfigs[j]["port_num"].(*schema.Set)
					portSet.Add(portPort)
				}
			}

			if included == false {
				logicalPortConfigs = append(logicalPortConfigs, map[string]interface{}{
					"desired_speed": logicalPortConfigInfo.DesiredSpeed,
					"primary_port":  primaryPort,
					"port_num":      schema.NewSet(func(a interface{}) int { return a.(int) }, portPorts),
					"bay_num":       portBay,
					"enclosure_num": portEnclosure,
				})
			}
		}

		//Oneview returns an unordered list so order it to match the configuration file
		logicalPortCount := d.Get("uplink_set." + strconv.Itoa(i) + ".logical_port_config.#").(int)
		oneviewLogicalPortCount := len(logicalPortConfigs)
		for j := 0; j < logicalPortCount; j++ {
			currBay := d.Get("uplink_set." + strconv.Itoa(i) + ".logical_port_config." + strconv.Itoa(j) + ".bay_num").(int)
			for k := 0; k < oneviewLogicalPortCount; k++ {
				if currBay == logicalPortConfigs[k]["bay_num"] && j <= k {
					logicalPortConfigs[j], logicalPortConfigs[k] = logicalPortConfigs[k], logicalPortConfigs[j]
				}
			}
		}

		networkUris := make([]interface{}, len(uplinkSet.NetworkUris))
		for i, networkUri := range uplinkSet.NetworkUris {
			networkUris[i] = networkUri.String()
		}

		uplinkSets = append(uplinkSets, map[string]interface{}{
			"network_type":          uplinkSet.NetworkType,
			"ethernet_network_type": uplinkSet.EthernetNetworkType,
			"name":                  uplinkSet.Name,
			"mode":                  uplinkSet.Mode,
			"lacp_timer":            uplinkSet.LacpTimer,
			"native_network_uri":    uplinkSet.NativeNetworkUri,
			"logical_port_config":   logicalPortConfigs,
			"network_uris":          schema.NewSet(schema.HashString, networkUris),
		})
	}
	uplinkCount := d.Get("uplink_set.#").(int)
	oneviewUplinkCount := len(uplinkSets)
	for i := 0; i < uplinkCount; i++ {
		currUplinkName := d.Get("uplink_set." + strconv.Itoa(i) + ".name").(string)
		for j := 0; j < oneviewUplinkCount; j++ {
			if currUplinkName == uplinkSets[j]["name"] && i <= j {
				uplinkSets[i], uplinkSets[j] = uplinkSets[j], uplinkSets[i]
			}
		}
	}
	d.Set("uplink_set", uplinkSets)

	internalNetworkUris := make([]interface{}, len(logicalInterconnectGroup.InternalNetworkUris))
	for i, internalNetworkUri := range logicalInterconnectGroup.InternalNetworkUris {
		internalNetworkUris[i] = internalNetworkUri
	}
	d.Set("internal_network_uris", internalNetworkUris)

	telemetryConfigurations := make([]map[string]interface{}, 0, 1)
	telemetryConfigurations = append(telemetryConfigurations, map[string]interface{}{
		"enabled":         *logicalInterconnectGroup.TelemetryConfiguration.EnableTelemetry,
		"sample_count":    logicalInterconnectGroup.TelemetryConfiguration.SampleCount,
		"sample_interval": logicalInterconnectGroup.TelemetryConfiguration.SampleInterval,
		"type":            logicalInterconnectGroup.TelemetryConfiguration.Type,
	})
	d.Set("telemetry_configuration", telemetryConfigurations)

	trapDestinations := make([]map[string]interface{}, 0, 1)
	for _, trapDestination := range logicalInterconnectGroup.SnmpConfiguration.TrapDestinations {

		enetTrapCategories := make([]interface{}, len(trapDestination.EnetTrapCategories))
		for i, enetTrapCategory := range trapDestination.EnetTrapCategories {
			enetTrapCategories[i] = enetTrapCategory
		}

		fcTrapCategories := make([]interface{}, len(trapDestination.FcTrapCategories))
		for i, fcTrapCategory := range trapDestination.FcTrapCategories {
			fcTrapCategories[i] = fcTrapCategory
		}

		vcmTrapCategories := make([]interface{}, len(trapDestination.VcmTrapCategories))
		for i, vcmTrapCategory := range trapDestination.VcmTrapCategories {
			vcmTrapCategories[i] = vcmTrapCategory
		}

		trapSeverities := make([]interface{}, len(trapDestination.TrapSeverities))
		for i, trapSeverity := range trapDestination.TrapSeverities {
			trapSeverities[i] = trapSeverity
		}

		trapDestinations = append(trapDestinations, map[string]interface{}{
			"trap_destination":     trapDestination.TrapDestination,
			"community_string":     trapDestination.CommunityString,
			"trap_format":          trapDestination.TrapFormat,
			"enet_trap_categories": schema.NewSet(schema.HashString, enetTrapCategories),
			"fc_trap_categories":   schema.NewSet(schema.HashString, fcTrapCategories),
			"vcm_trap_categories":  schema.NewSet(schema.HashString, vcmTrapCategories),
			"trap_severities":      schema.NewSet(schema.HashString, trapSeverities),
		})
	}

	//Oneview returns an unordered list so order it to match the configuration file
	trapDestinationCount := d.Get("snmp_configuration.0.trap_destination.#").(int)
	oneviewTrapDestinationCount := len(trapDestinations)
	for i := 0; i < trapDestinationCount; i++ {
		currDest := d.Get("snmp_configuration.0.trap_destination." + strconv.Itoa(i) + ".trap_destination").(string)
		for j := 0; j < oneviewTrapDestinationCount; j++ {
			if currDest == trapDestinations[j]["trap_destination"] && i <= j {
				trapDestinations[i], trapDestinations[j] = trapDestinations[j], trapDestinations[i]
			}
		}
	}

	snmpAccess := make([]interface{}, len(logicalInterconnectGroup.SnmpConfiguration.SnmpAccess))
	for i, snmpAccessIP := range logicalInterconnectGroup.SnmpConfiguration.SnmpAccess {
		snmpAccess[i] = snmpAccessIP
	}

	snmpConfiguration := make([]map[string]interface{}, 0, 1)
	snmpConfiguration = append(snmpConfiguration, map[string]interface{}{
		"enabled":          *logicalInterconnectGroup.SnmpConfiguration.Enabled,
		"read_community":   logicalInterconnectGroup.SnmpConfiguration.ReadCommunity,
		"snmp_access":      schema.NewSet(schema.HashString, snmpAccess),
		"system_contact":   logicalInterconnectGroup.SnmpConfiguration.SystemContact,
		"type":             logicalInterconnectGroup.SnmpConfiguration.Type,
		"trap_destination": trapDestinations,
	})
	d.Set("snmp_configuration", snmpConfiguration)

	interconnectSettings := make([]map[string]interface{}, 0, 1)
	interconnectSettings = append(interconnectSettings, map[string]interface{}{
		"type": logicalInterconnectGroup.EthernetSettings.Type,
		"fast_mac_cache_failover": *logicalInterconnectGroup.EthernetSettings.EnableFastMacCacheFailover,
		"igmp_snooping":           *logicalInterconnectGroup.EthernetSettings.EnableIgmpSnooping,
		"network_loop_protection": *logicalInterconnectGroup.EthernetSettings.EnableNetworkLoopProtection,
		"pause_flood_protection":  *logicalInterconnectGroup.EthernetSettings.EnablePauseFloodProtection,
		"rich_tlv":                *logicalInterconnectGroup.EthernetSettings.EnableRichTLV,
		"igmp_timeout_interval":   logicalInterconnectGroup.EthernetSettings.IgmpIdleTimeoutInterval,
		"mac_refresh_interval":    logicalInterconnectGroup.EthernetSettings.MacRefreshInterval,
	})
	d.Set("interconnect_settings", interconnectSettings)

	qosTrafficClasses := make([]map[string]interface{}, 0, 1)
	for _, qosTrafficClass := range logicalInterconnectGroup.QosConfiguration.ActiveQosConfig.QosTrafficClassifiers {

		dscpClassMap := make([]interface{}, len(qosTrafficClass.QosClassificationMapping.DscpClassMapping))
		for i, dscpValue := range qosTrafficClass.QosClassificationMapping.DscpClassMapping {
			dscpClassMap[i] = dscpValue
		}

		dot1pClassMap := make([]interface{}, len(qosTrafficClass.QosClassificationMapping.Dot1pClassMapping))
		for i, dot1pValue := range qosTrafficClass.QosClassificationMapping.Dot1pClassMapping {
			dot1pClassMap[i] = dot1pValue
		}
		qosClassificationMap := make([]map[string]interface{}, 0, 1)
		qosClassificationMap = append(qosClassificationMap, map[string]interface{}{
			"dot1p_class_map": schema.NewSet(func(a interface{}) int { return a.(int) }, dot1pClassMap),
			"dscp_class_map":  schema.NewSet(schema.HashString, dscpClassMap),
		})

		qosTrafficClasses = append(qosTrafficClasses, map[string]interface{}{
			"name":                   qosTrafficClass.QosTrafficClass.ClassName,
			"enabled":                *qosTrafficClass.QosTrafficClass.Enabled,
			"egress_dot1p_value":     qosTrafficClass.QosTrafficClass.EgressDot1pValue,
			"real_time":              *qosTrafficClass.QosTrafficClass.RealTime,
			"bandwidth_share":        qosTrafficClass.QosTrafficClass.BandwidthShare,
			"max_bandwidth":          qosTrafficClass.QosTrafficClass.MaxBandwidth,
			"qos_classification_map": qosClassificationMap,
		})
	}
	qosTrafficClassCount := d.Get("quality_of_service.0.qos_traffic_class.#").(int)
	oneviewTrafficClassCount := len(qosTrafficClasses)
	for i := 0; i < qosTrafficClassCount; i++ {
		currName := d.Get("quality_of_service.0.qos_traffic_class." + strconv.Itoa(i) + ".name").(string)
		for j := 0; j < oneviewTrafficClassCount; j++ {
			if currName == qosTrafficClasses[j]["name"] && i <= j {
				qosTrafficClasses[i], qosTrafficClasses[j] = qosTrafficClasses[j], qosTrafficClasses[i]
			}
		}
	}

	qualityOfService := make([]map[string]interface{}, 0, 1)
	qualityOfService = append(qualityOfService, map[string]interface{}{
		"type": logicalInterconnectGroup.QosConfiguration.Type,
		"active_qos_config_type":       logicalInterconnectGroup.QosConfiguration.ActiveQosConfig.Type,
		"config_type":                  logicalInterconnectGroup.QosConfiguration.ActiveQosConfig.ConfigType,
		"uplink_classification_type":   logicalInterconnectGroup.QosConfiguration.ActiveQosConfig.UplinkClassificationType,
		"downlink_classification_type": logicalInterconnectGroup.QosConfiguration.ActiveQosConfig.DownlinkClassificationType,
		"qos_traffic_class":            qosTrafficClasses,
	})

	d.Set("quality_of_service", qualityOfService)

	return nil
}

func resourceLogicalInterconnectGroupDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	error := config.ovClient.DeleteLogicalInterconnectGroup(d.Get("name").(string))
	if error != nil {
		return error
	}
	return nil
}

func resourceLogicalInterconnectGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	lig := ov.LogicalInterconnectGroup{
		Name: d.Get("name").(string),
		Type: d.Get("type").(string),
		URI:  utils.NewNstring(d.Get("uri").(string)),
	}

	interconnectMapEntryTemplateCount := d.Get("interconnect_map_entry_template.#").(int)
	interconnectMapEntryTemplates := make([]ov.InterconnectMapEntryTemplate, 0)
	for i := 0; i < interconnectMapEntryTemplateCount; i++ {
		interconnectMapEntryTemplatePrefix := fmt.Sprintf("interconnect_map_entry_template.%d", i)
		interconnectTypeName := d.Get(interconnectMapEntryTemplatePrefix + ".interconnect_type_name").(string)
		interconnectType, err := config.ovClient.GetInterconnectTypeByName(interconnectTypeName)
		if err != nil {
			return err
		}
		if interconnectType.URI == "" {
			return fmt.Errorf("Could not find Interconnect Type from name: %s", interconnectTypeName)
		}

		enclosureLocation := ov.LocationEntry{
			RelativeValue: d.Get(interconnectMapEntryTemplatePrefix + ".enclosure_index").(int),
			Type:          "Enclosure",
		}
		locationEntries := make([]ov.LocationEntry, 0)
		locationEntries = append(locationEntries, enclosureLocation)

		bayLocation := ov.LocationEntry{
			RelativeValue: d.Get(interconnectMapEntryTemplatePrefix + ".bay_number").(int),
			Type:          "Bay",
		}
		locationEntries = append(locationEntries, bayLocation)
		logicalLocation := ov.LogicalLocation{
			LocationEntries: locationEntries,
		}
		interconnectMapEntryTemplates = append(interconnectMapEntryTemplates, ov.InterconnectMapEntryTemplate{
			LogicalLocation:              logicalLocation,
			EnclosureIndex:               d.Get(interconnectMapEntryTemplatePrefix + ".enclosure_index").(int),
			PermittedInterconnectTypeUri: interconnectType.URI,
		})
	}

	interconnectMapTemplate := ov.InterconnectMapTemplate{
		InterconnectMapEntryTemplates: interconnectMapEntryTemplates,
	}
	lig.InterconnectMapTemplate = &interconnectMapTemplate

	uplinkSetCount := d.Get("uplink_set.#").(int)
	uplinkSets := make([]ov.UplinkSet, 0)
	for i := 0; i < uplinkSetCount; i++ {
		uplinkSetPrefix := fmt.Sprintf("uplink_set.%d", i)
		uplinkSet := ov.UplinkSet{}
		if val, ok := d.GetOk(uplinkSetPrefix + ".name"); ok {
			uplinkSet.Name = val.(string)
		}
		if val, ok := d.GetOk(uplinkSetPrefix + ".network_type"); ok {
			uplinkSet.NetworkType = val.(string)
		}
		if val, ok := d.GetOk(uplinkSetPrefix + ".ethernet_network_type"); ok {
			uplinkSet.EthernetNetworkType = val.(string)
		}
		if val, ok := d.GetOk(uplinkSetPrefix + ".mode"); ok {
			uplinkSet.Mode = val.(string)
		}
		if val, ok := d.GetOk(uplinkSetPrefix + ".lacp_timer"); ok {
			uplinkSet.LacpTimer = val.(string)
		}
		if val, ok := d.GetOk(uplinkSetPrefix + ".native_network_uri"); ok {
			uplinkSet.NativeNetworkUri = utils.NewNstring(val.(string))
		}

		logicalPortCount := d.Get(uplinkSetPrefix + ".logical_port_config.#").(int)
		logicalPorts := make([]ov.LogicalPortConfigInfo, 0)
		for i := 0; i < logicalPortCount; i++ {
			logicalPortPrefix := fmt.Sprintf(uplinkSetPrefix+".logical_port_config.%d", i)
			rawPortLocations := d.Get(logicalPortPrefix + ".port_num").(*schema.Set).List()
			for _, raw := range rawPortLocations {
				logicalPort := ov.LogicalPortConfigInfo{}

				if val, ok := d.GetOk(logicalPortPrefix + ".desired_speed"); ok {
					logicalPort.DesiredSpeed = val.(string)
				}

				locationEntries := make([]ov.LocationEntry, 0)
				enclosureLocation := ov.LocationEntry{
					RelativeValue: d.Get(logicalPortPrefix + ".enclosure_num").(int),
					Type:          "Enclosure",
				}
				locationEntries = append(locationEntries, enclosureLocation)

				bayLocation := ov.LocationEntry{
					RelativeValue: d.Get(logicalPortPrefix + ".bay_num").(int),
					Type:          "Bay",
				}
				locationEntries = append(locationEntries, bayLocation)

				portLocation := ov.LocationEntry{
					RelativeValue: raw.(int),
					Type:          "Port",
				}
				locationEntries = append(locationEntries, portLocation)

				logicalLocation := ov.LogicalLocation{
					LocationEntries: locationEntries,
				}

				logicalPort.LogicalLocation = logicalLocation
				if _, ok := d.GetOk(logicalPortPrefix + ".primary_port"); ok {
					if uplinkSet.PrimaryPort == nil {
						uplinkSet.PrimaryPort = &logicalLocation
					}
				}

				logicalPorts = append(logicalPorts, logicalPort)
			}

		}
		uplinkSet.LogicalPortConfigInfos = logicalPorts

		rawNetUris := d.Get(uplinkSetPrefix + ".network_uris").(*schema.Set).List()
		netUris := make([]utils.Nstring, 0)
		for _, raw := range rawNetUris {
			netUris = append(netUris, utils.NewNstring(raw.(string)))
		}
		uplinkSet.NetworkUris = netUris

		uplinkSets = append(uplinkSets, uplinkSet)
	}
	lig.UplinkSets = uplinkSets

	rawInternalNetUris := d.Get("internal_network_uris").(*schema.Set).List()
	internalNetUris := make([]utils.Nstring, len(rawInternalNetUris))
	for i, raw := range rawInternalNetUris {
		internalNetUris[i] = utils.NewNstring(raw.(string))
	}
	lig.InternalNetworkUris = internalNetUris

	telemetryConfigPrefix := fmt.Sprintf("telemetry_configuration.0")
	telemetryConfiguration := ov.TelemetryConfiguration{}
	if val, ok := d.GetOk(telemetryConfigPrefix + ".sample_count"); ok {
		telemetryConfiguration.SampleCount = val.(int)
	}
	if val, ok := d.GetOk(telemetryConfigPrefix + ".sample_interval"); ok {
		telemetryConfiguration.SampleInterval = val.(int)
	}
	if val, ok := d.GetOk(telemetryConfigPrefix + ".enabled"); ok {
		enabled := val.(bool)
		telemetryConfiguration.EnableTelemetry = &enabled
	}
	if telemetryConfiguration != (ov.TelemetryConfiguration{}) {
		telemetryConfiguration.Type = d.Get(telemetryConfigPrefix + ".type").(string)
		lig.TelemetryConfiguration = &telemetryConfiguration
	}

	snmpConfigPrefix := fmt.Sprintf("snmp_configuration.0")
	snmpConfiguration := ov.SnmpConfiguration{}
	if val, ok := d.GetOk(snmpConfigPrefix + ".enabled"); ok {
		enabled := val.(bool)
		snmpConfiguration.Enabled = &enabled
	}
	if val, ok := d.GetOk(snmpConfigPrefix + ".read_community"); ok {
		snmpConfiguration.ReadCommunity = val.(string)
	}
	if val, ok := d.GetOk(snmpConfigPrefix + ".system_contact"); ok {
		snmpConfiguration.SystemContact = val.(string)
	}
	rawSnmpAccess := d.Get(snmpConfigPrefix + ".snmp_access").(*schema.Set).List()
	snmpAccess := make([]string, len(rawSnmpAccess))
	for i, raw := range rawSnmpAccess {
		snmpAccess[i] = raw.(string)
	}

	trapDestinationCount := d.Get(snmpConfigPrefix + ".trap_destination.#").(int)
	trapDestinations := make([]ov.TrapDestination, 0, trapDestinationCount)
	for i := 0; i < trapDestinationCount; i++ {
		trapDestinationPrefix := fmt.Sprintf(snmpConfigPrefix+".trap_destination.%d", i)

		rawEnetTrapCategories := d.Get(trapDestinationPrefix + ".enet_trap_categories").(*schema.Set).List()
		enetTrapCategories := make([]string, len(rawEnetTrapCategories))
		for i, raw := range rawEnetTrapCategories {
			enetTrapCategories[i] = raw.(string)
		}

		rawFcTrapCategories := d.Get(trapDestinationPrefix + ".fc_trap_categories").(*schema.Set).List()
		fcTrapCategories := make([]string, len(rawFcTrapCategories))
		for i, raw := range rawFcTrapCategories {
			fcTrapCategories[i] = raw.(string)
		}

		rawVcmTrapCategories := d.Get(trapDestinationPrefix + ".vcm_trap_categories").(*schema.Set).List()
		vcmTrapCategories := make([]string, len(rawVcmTrapCategories))
		for i, raw := range rawVcmTrapCategories {
			vcmTrapCategories[i] = raw.(string)
		}

		rawTrapSeverities := d.Get(trapDestinationPrefix + ".trap_severities").(*schema.Set).List()
		trapSeverities := make([]string, len(rawTrapSeverities))
		for i, raw := range rawTrapSeverities {
			trapSeverities[i] = raw.(string)
		}

		trapDestination := ov.TrapDestination{
			TrapDestination:    d.Get(trapDestinationPrefix + ".trap_destination").(string),
			CommunityString:    d.Get(trapDestinationPrefix + ".community_string").(string),
			TrapFormat:         d.Get(trapDestinationPrefix + ".trap_format").(string),
			EnetTrapCategories: enetTrapCategories,
			FcTrapCategories:   fcTrapCategories,
			VcmTrapCategories:  vcmTrapCategories,
			TrapSeverities:     trapSeverities,
		}
		trapDestinations = append(trapDestinations, trapDestination)
	}
	if trapDestinationCount > 0 {
		snmpConfiguration.TrapDestinations = trapDestinations
	}

	snmpConfiguration.SnmpAccess = snmpAccess
	if val, ok := d.GetOk(snmpConfigPrefix + ".type"); ok {
		snmpConfiguration.Type = val.(string)
		lig.SnmpConfiguration = &snmpConfiguration
	}

	interconnectSettingsPrefix := fmt.Sprintf("interconnect_settings.0")
	if val, ok := d.GetOk(interconnectSettingsPrefix + ".type"); ok {
		interconnectSettings := ov.EthernetSettings{}

		macFailoverEnabled := d.Get(interconnectSettingsPrefix + ".fast_mac_cache_failover").(bool)
		interconnectSettings.EnableFastMacCacheFailover = &macFailoverEnabled

		networkLoopProtectionEnabled := d.Get(interconnectSettingsPrefix + ".network_loop_protection").(bool)
		interconnectSettings.EnableNetworkLoopProtection = &networkLoopProtectionEnabled

		pauseFloodProtectionEnabled := d.Get(interconnectSettingsPrefix + ".pause_flood_protection").(bool)
		interconnectSettings.EnablePauseFloodProtection = &pauseFloodProtectionEnabled

		if val1, ok := d.GetOk(interconnectSettingsPrefix + ".rich_tlv"); ok {
			enabled := val1.(bool)
			interconnectSettings.EnableRichTLV = &enabled
		}

		if val1, ok := d.GetOk(interconnectSettingsPrefix + ".igmp_snooping"); ok {
			enabled := val1.(bool)
			interconnectSettings.EnableIgmpSnooping = &enabled
		}

		if val1, ok := d.GetOk(interconnectSettingsPrefix + ".igmp_timeout_interval"); ok {
			interconnectSettings.IgmpIdleTimeoutInterval = val1.(int)
		}

		if val1, ok := d.GetOk(interconnectSettingsPrefix + ".mac_refresh_interval"); ok {
			interconnectSettings.MacRefreshInterval = val1.(int)
		}

		interconnectSettings.Type = val.(string)
		lig.EthernetSettings = &interconnectSettings
	}

	qualityOfServicePrefix := fmt.Sprintf("quality_of_service.0")
	activeQosConfig := ov.ActiveQosConfig{}

	if val, ok := d.GetOk(qualityOfServicePrefix + ".config_type"); ok {
		activeQosConfig.ConfigType = val.(string)
	}

	if val, ok := d.GetOk(qualityOfServicePrefix + ".uplink_classification_type"); ok {
		activeQosConfig.UplinkClassificationType = val.(string)
	}

	if val, ok := d.GetOk(qualityOfServicePrefix + ".downlink_classification_type"); ok {
		activeQosConfig.DownlinkClassificationType = val.(string)
	}

	qosTrafficClassCount := d.Get(qualityOfServicePrefix + ".qos_traffic_class.#").(int)
	qosTrafficClassifiers := make([]ov.QosTrafficClassifier, 0, 1)
	for i := 0; i < qosTrafficClassCount; i++ {
		qosTrafficClassPrefix := fmt.Sprintf(qualityOfServicePrefix+".qos_traffic_class.%d", i)
		qosTrafficClassifier := ov.QosTrafficClassifier{}
		qosClassMap := ov.QosClassificationMap{}
		qosTrafficClass := ov.QosTrafficClass{}

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".name"); ok {
			qosTrafficClass.ClassName = val.(string)
		}
		classEnabled := d.Get(qosTrafficClassPrefix + ".enabled").(bool)
		qosTrafficClass.Enabled = &classEnabled

		realTimeEnabled := d.Get(qosTrafficClassPrefix + ".real_time").(bool)
		qosTrafficClass.RealTime = &realTimeEnabled

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".egress_dot1p_value"); ok {
			qosTrafficClass.EgressDot1pValue = val.(int)
		}

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".bandwidth_share"); ok {
			qosTrafficClass.BandwidthShare = val.(string)
		}

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".max_bandwidth"); ok {
			qosTrafficClass.MaxBandwidth = val.(int)
		}

		qosTrafficClassifier.QosTrafficClass = qosTrafficClass

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".qos_classification_map.0.dscp_class_map"); ok {
			rawDscpClassMapping := val.(*schema.Set).List()
			dscpClassMapping := make([]string, len(rawDscpClassMapping))
			for i, raw := range rawDscpClassMapping {
				dscpClassMapping[i] = raw.(string)
			}
			qosClassMap.DscpClassMapping = dscpClassMapping
		}

		if val, ok := d.GetOk(qosTrafficClassPrefix + ".qos_classification_map.0.dot1p_class_map"); ok {
			rawDot1pClassMap := val.(*schema.Set).List()
			dot1pClassMap := make([]int, len(rawDot1pClassMap))
			for i, raw := range rawDot1pClassMap {
				dot1pClassMap[i] = raw.(int)
			}
			qosClassMap.Dot1pClassMapping = dot1pClassMap
		}

		qosTrafficClassifier.QosClassificationMapping = &qosClassMap

		qosTrafficClassifiers = append(qosTrafficClassifiers, qosTrafficClassifier)
	}
	activeQosConfig.QosTrafficClassifiers = qosTrafficClassifiers

	if !reflect.DeepEqual(activeQosConfig, (ov.ActiveQosConfig{})) {
		activeQosConfig.Type = d.Get(qualityOfServicePrefix + ".active_qos_config_type").(string)

		qualityOfService := ov.QosConfiguration{
			Type:            d.Get(qualityOfServicePrefix + ".type").(string),
			ActiveQosConfig: activeQosConfig,
		}

		lig.QosConfiguration = &qualityOfService
	}

	err := config.ovClient.UpdateLogicalInterconnectGroup(lig)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))

	return resourceLogicalInterconnectGroupRead(d, meta)
}
