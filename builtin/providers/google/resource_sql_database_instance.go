package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/sqladmin/v1beta4"
)

func resourceSqlDatabaseInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceSqlDatabaseInstanceCreate,
		Read:   resourceSqlDatabaseInstanceRead,
		Update: resourceSqlDatabaseInstanceUpdate,
		Delete: resourceSqlDatabaseInstanceDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"settings": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"version": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"tier": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"activation_policy": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"authorized_gae_applications": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"backup_configuration": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"binary_log_enabled": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
									},
									"enabled": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
									},
									"start_time": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"crash_safe_replication": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"database_flags": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"value": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"ip_configuration": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"authorized_networks": &schema.Schema{
										Type:     schema.TypeList,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"expiration_time": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"name": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
												"value": &schema.Schema{
													Type:     schema.TypeString,
													Optional: true,
												},
											},
										},
									},
									"ipv4_enabled": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
									},
									"require_ssl": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
									},
								},
							},
						},
						"location_preference": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"follow_gae_application": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
									"zone": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"pricing_plan": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"replication_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"database_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "MYSQL_5_5",
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip_address": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"time_to_retire": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"master_instance_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"replica_configuration": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ca_certificate": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"client_certificate": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"client_key": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"connect_retry_interval": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"dump_file_path": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"master_heartbeat_period": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"password": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"ssl_cipher": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"username": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"verify_server_certificate": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceSqlDatabaseInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region := d.Get("region").(string)
	databaseVersion := d.Get("database_version").(string)

	_settingsList := d.Get("settings").([]interface{})
	if len(_settingsList) > 1 {
		return fmt.Errorf("At most one settings block is allowed")
	}

	_settings := _settingsList[0].(map[string]interface{})
	settings := &sqladmin.Settings{
		Tier: _settings["tier"].(string),
	}

	if v, ok := _settings["activation_policy"]; ok {
		settings.ActivationPolicy = v.(string)
	}

	if v, ok := _settings["authorized_gae_applications"]; ok {
		settings.AuthorizedGaeApplications = make([]string, 0)
		for _, app := range v.([]interface{}) {
			settings.AuthorizedGaeApplications = append(settings.AuthorizedGaeApplications,
				app.(string))
		}
	}

	if v, ok := _settings["backup_configuration"]; ok {
		_backupConfigurationList := v.([]interface{})
		if len(_backupConfigurationList) > 1 {
			return fmt.Errorf("At most one backup_configuration block is allowed")
		}

		if len(_backupConfigurationList) == 1 && _backupConfigurationList[0] != nil {
			settings.BackupConfiguration = &sqladmin.BackupConfiguration{}
			_backupConfiguration := _backupConfigurationList[0].(map[string]interface{})

			if vp, okp := _backupConfiguration["binary_log_enabled"]; okp {
				settings.BackupConfiguration.BinaryLogEnabled = vp.(bool)
			}

			if vp, okp := _backupConfiguration["enabled"]; okp {
				settings.BackupConfiguration.Enabled = vp.(bool)
			}

			if vp, okp := _backupConfiguration["start_time"]; okp {
				settings.BackupConfiguration.StartTime = vp.(string)
			}
		}
	}

	if v, ok := _settings["crash_safe_replication"]; ok {
		settings.CrashSafeReplicationEnabled = v.(bool)
	}

	if v, ok := _settings["database_flags"]; ok {
		settings.DatabaseFlags = make([]*sqladmin.DatabaseFlags, 0)
		_databaseFlagsList := v.([]interface{})
		for _, _flag := range _databaseFlagsList {
			_entry := _flag.(map[string]interface{})
			flag := &sqladmin.DatabaseFlags{}
			if vp, okp := _entry["name"]; okp {
				flag.Name = vp.(string)
			}

			if vp, okp := _entry["value"]; okp {
				flag.Value = vp.(string)
			}

			settings.DatabaseFlags = append(settings.DatabaseFlags, flag)
		}
	}

	if v, ok := _settings["ip_configuration"]; ok {
		_ipConfigurationList := v.([]interface{})
		if len(_ipConfigurationList) > 1 {
			return fmt.Errorf("At most one ip_configuration block is allowed")
		}

		if len(_ipConfigurationList) == 1 && _ipConfigurationList[0] != nil {
			settings.IpConfiguration = &sqladmin.IpConfiguration{}
			_ipConfiguration := _ipConfigurationList[0].(map[string]interface{})

			if vp, okp := _ipConfiguration["ipv4_enabled"]; okp {
				settings.IpConfiguration.Ipv4Enabled = vp.(bool)
			}

			if vp, okp := _ipConfiguration["require_ssl"]; okp {
				settings.IpConfiguration.RequireSsl = vp.(bool)
			}

			if vp, okp := _ipConfiguration["authorized_networks"]; okp {
				settings.IpConfiguration.AuthorizedNetworks = make([]*sqladmin.AclEntry, 0)
				_authorizedNetworksList := vp.([]interface{})
				for _, _acl := range _authorizedNetworksList {
					_entry := _acl.(map[string]interface{})
					entry := &sqladmin.AclEntry{}

					if vpp, okpp := _entry["expiration_time"]; okpp {
						entry.ExpirationTime = vpp.(string)
					}

					if vpp, okpp := _entry["name"]; okpp {
						entry.Name = vpp.(string)
					}

					if vpp, okpp := _entry["value"]; okpp {
						entry.Value = vpp.(string)
					}

					settings.IpConfiguration.AuthorizedNetworks = append(
						settings.IpConfiguration.AuthorizedNetworks, entry)
				}
			}
		}
	}

	if v, ok := _settings["location_preference"]; ok {
		_locationPreferenceList := v.([]interface{})
		if len(_locationPreferenceList) > 1 {
			return fmt.Errorf("At most one location_preference block is allowed")
		}

		if len(_locationPreferenceList) == 1 && _locationPreferenceList[0] != nil {
			settings.LocationPreference = &sqladmin.LocationPreference{}
			_locationPreference := _locationPreferenceList[0].(map[string]interface{})

			if vp, okp := _locationPreference["follow_gae_application"]; okp {
				settings.LocationPreference.FollowGaeApplication = vp.(string)
			}

			if vp, okp := _locationPreference["zone"]; okp {
				settings.LocationPreference.Zone = vp.(string)
			}
		}
	}

	if v, ok := _settings["pricing_plan"]; ok {
		settings.PricingPlan = v.(string)
	}

	if v, ok := _settings["replication_type"]; ok {
		settings.ReplicationType = v.(string)
	}

	instance := &sqladmin.DatabaseInstance{
		Region:          region,
		Settings:        settings,
		DatabaseVersion: databaseVersion,
	}

	if v, ok := d.GetOk("name"); ok {
		instance.Name = v.(string)
	} else {
		instance.Name = resource.UniqueId()
		d.Set("name", instance.Name)
	}

	if v, ok := d.GetOk("replica_configuration"); ok {
		_replicaConfigurationList := v.([]interface{})
		if len(_replicaConfigurationList) > 1 {
			return fmt.Errorf("Only one replica_configuration block may be defined")
		}

		if len(_replicaConfigurationList) == 1 && _replicaConfigurationList[0] != nil {
			replicaConfiguration := &sqladmin.ReplicaConfiguration{}
			mySqlReplicaConfiguration := &sqladmin.MySqlReplicaConfiguration{}
			_replicaConfiguration := _replicaConfigurationList[0].(map[string]interface{})

			if vp, okp := _replicaConfiguration["ca_certificate"]; okp {
				mySqlReplicaConfiguration.CaCertificate = vp.(string)
			}

			if vp, okp := _replicaConfiguration["client_certificate"]; okp {
				mySqlReplicaConfiguration.ClientCertificate = vp.(string)
			}

			if vp, okp := _replicaConfiguration["client_key"]; okp {
				mySqlReplicaConfiguration.ClientKey = vp.(string)
			}

			if vp, okp := _replicaConfiguration["connect_retry_interval"]; okp {
				mySqlReplicaConfiguration.ConnectRetryInterval = int64(vp.(int))
			}

			if vp, okp := _replicaConfiguration["dump_file_path"]; okp {
				mySqlReplicaConfiguration.DumpFilePath = vp.(string)
			}

			if vp, okp := _replicaConfiguration["master_heartbeat_period"]; okp {
				mySqlReplicaConfiguration.MasterHeartbeatPeriod = int64(vp.(int))
			}

			if vp, okp := _replicaConfiguration["password"]; okp {
				mySqlReplicaConfiguration.Password = vp.(string)
			}

			if vp, okp := _replicaConfiguration["ssl_cipher"]; okp {
				mySqlReplicaConfiguration.SslCipher = vp.(string)
			}

			if vp, okp := _replicaConfiguration["username"]; okp {
				mySqlReplicaConfiguration.Username = vp.(string)
			}

			if vp, okp := _replicaConfiguration["verify_server_certificate"]; okp {
				mySqlReplicaConfiguration.VerifyServerCertificate = vp.(bool)
			}

			replicaConfiguration.MysqlReplicaConfiguration = mySqlReplicaConfiguration
			instance.ReplicaConfiguration = replicaConfiguration
		}
	}

	if v, ok := d.GetOk("master_instance_name"); ok {
		instance.MasterInstanceName = v.(string)
	}

	op, err := config.clientSqlAdmin.Instances.Insert(project, instance).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 409 {
			return fmt.Errorf("Error, the name %s is unavailable because it was used recently", instance.Name)
		} else {
			return fmt.Errorf("Error, failed to create instance %s: %s", instance.Name, err)
		}
	}

	err = sqladminOperationWait(config, op, "Create Instance")
	if err != nil {
		return err
	}

	return resourceSqlDatabaseInstanceRead(d, meta)
}

func resourceSqlDatabaseInstanceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	instance, err := config.clientSqlAdmin.Instances.Get(project,
		d.Get("name").(string)).Do()

	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing SQL Database %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error retrieving instance %s: %s",
			d.Get("name").(string), err)
	}

	_settingsList := d.Get("settings").([]interface{})
	_settings := _settingsList[0].(map[string]interface{})

	settings := instance.Settings
	_settings["version"] = settings.SettingsVersion
	_settings["tier"] = settings.Tier

	// Take care to only update attributes that the user has defined explicitly
	if v, ok := _settings["activation_policy"]; ok && len(v.(string)) > 0 {
		_settings["activation_policy"] = settings.ActivationPolicy
	}

	if v, ok := _settings["authorized_gae_applications"]; ok && len(v.([]interface{})) > 0 {
		_authorized_gae_applications := make([]interface{}, 0)
		for _, app := range settings.AuthorizedGaeApplications {
			_authorized_gae_applications = append(_authorized_gae_applications, app)
		}
		_settings["authorized_gae_applications"] = _authorized_gae_applications
	}

	if v, ok := _settings["backup_configuration"]; ok {
		_backupConfigurationList := v.([]interface{})
		if len(_backupConfigurationList) > 1 {
			return fmt.Errorf("At most one backup_configuration block is allowed")
		}

		if len(_backupConfigurationList) == 1 && _backupConfigurationList[0] != nil {
			_backupConfiguration := _backupConfigurationList[0].(map[string]interface{})

			if vp, okp := _backupConfiguration["binary_log_enabled"]; okp && vp != nil {
				_backupConfiguration["binary_log_enabled"] = settings.BackupConfiguration.BinaryLogEnabled
			}

			if vp, okp := _backupConfiguration["enabled"]; okp && vp != nil {
				_backupConfiguration["enabled"] = settings.BackupConfiguration.Enabled
			}

			if vp, okp := _backupConfiguration["start_time"]; okp && vp != nil {
				_backupConfiguration["start_time"] = settings.BackupConfiguration.StartTime
			}

			_backupConfigurationList[0] = _backupConfiguration
			_settings["backup_configuration"] = _backupConfigurationList
		}
	}

	if v, ok := _settings["crash_safe_replication"]; ok && v != nil {
		_settings["crash_safe_replication"] = settings.CrashSafeReplicationEnabled
	}

	if v, ok := _settings["database_flags"]; ok && len(v.([]interface{})) > 0 {
		_flag_map := make(map[string]string)
		// First keep track of localy defined flag pairs
		for _, _flag := range _settings["database_flags"].([]interface{}) {
			_entry := _flag.(map[string]interface{})
			_flag_map[_entry["name"].(string)] = _entry["value"].(string)
		}

		_database_flags := make([]interface{}, 0)
		// Next read the flag pairs from the server, and reinsert those that
		// correspond to ones defined locally
		for _, entry := range settings.DatabaseFlags {
			if _, okp := _flag_map[entry.Name]; okp {
				_entry := make(map[string]interface{})
				_entry["name"] = entry.Name
				_entry["value"] = entry.Value
				_database_flags = append(_database_flags, _entry)
			}
		}
		_settings["database_flags"] = _database_flags
	}

	if v, ok := _settings["ip_configuration"]; ok {
		_ipConfigurationList := v.([]interface{})
		if len(_ipConfigurationList) > 1 {
			return fmt.Errorf("At most one ip_configuration block is allowed")
		}

		if len(_ipConfigurationList) == 1 && _ipConfigurationList[0] != nil {
			_ipConfiguration := _ipConfigurationList[0].(map[string]interface{})

			if vp, okp := _ipConfiguration["ipv4_enabled"]; okp && vp != nil {
				_ipConfiguration["ipv4_enabled"] = settings.IpConfiguration.Ipv4Enabled
			}

			if vp, okp := _ipConfiguration["require_ssl"]; okp && vp != nil {
				_ipConfiguration["require_ssl"] = settings.IpConfiguration.RequireSsl
			}

			if vp, okp := _ipConfiguration["authorized_networks"]; okp && vp != nil {
				_ipc_map := make(map[string]interface{})
				// First keep track of localy defined ip configurations
				for _, _ipc := range _ipConfigurationList {
					_entry := _ipc.(map[string]interface{})
					if _entry["value"] == nil {
						continue
					}
					_value := make(map[string]interface{})
					_value["name"] = _entry["name"]
					_value["expiration_time"] = _entry["expiration_time"]
					// We key on value, since that is the only required part of
					// this 3-tuple
					_ipc_map[_entry["value"].(string)] = _value
				}
				_authorized_networks := make([]interface{}, 0)
				// Next read the network tuples from the server, and reinsert those that
				// correspond to ones defined locally
				for _, entry := range settings.IpConfiguration.AuthorizedNetworks {
					if _, okp := _ipc_map[entry.Value]; okp {
						_entry := make(map[string]interface{})
						_entry["value"] = entry.Value
						_entry["name"] = entry.Name
						_entry["expiration_time"] = entry.ExpirationTime
						_authorized_networks = append(_authorized_networks, _entry)
					}
				}
				_ipConfiguration["authorized_networks"] = _authorized_networks
			}
			_ipConfigurationList[0] = _ipConfiguration
			_settings["ip_configuration"] = _ipConfigurationList
		}
	}

	if v, ok := _settings["location_preference"]; ok && len(v.([]interface{})) > 0 {
		_locationPreferenceList := v.([]interface{})
		if len(_locationPreferenceList) > 1 {
			return fmt.Errorf("At most one location_preference block is allowed")
		}

		if len(_locationPreferenceList) == 1 && _locationPreferenceList[0] != nil &&
			settings.LocationPreference != nil {
			_locationPreference := _locationPreferenceList[0].(map[string]interface{})

			if vp, okp := _locationPreference["follow_gae_application"]; okp && vp != nil {
				_locationPreference["follow_gae_application"] =
					settings.LocationPreference.FollowGaeApplication
			}

			if vp, okp := _locationPreference["zone"]; okp && vp != nil {
				_locationPreference["zone"] = settings.LocationPreference.Zone
			}

			_locationPreferenceList[0] = _locationPreference
			_settings["location_preference"] = _locationPreferenceList[0]
		}
	}

	if v, ok := _settings["pricing_plan"]; ok && len(v.(string)) > 0 {
		_settings["pricing_plan"] = settings.PricingPlan
	}

	if v, ok := _settings["replication_type"]; ok && len(v.(string)) > 0 {
		_settings["replication_type"] = settings.ReplicationType
	}

	_settingsList[0] = _settings
	d.Set("settings", _settingsList)

	if v, ok := d.GetOk("replica_configuration"); ok && v != nil {
		_replicaConfigurationList := v.([]interface{})
		if len(_replicaConfigurationList) > 1 {
			return fmt.Errorf("Only one replica_configuration block may be defined")
		}

		if len(_replicaConfigurationList) == 1 && _replicaConfigurationList[0] != nil {
			mySqlReplicaConfiguration := instance.ReplicaConfiguration.MysqlReplicaConfiguration
			_replicaConfiguration := _replicaConfigurationList[0].(map[string]interface{})

			if vp, okp := _replicaConfiguration["ca_certificate"]; okp && vp != nil {
				_replicaConfiguration["ca_certificate"] = mySqlReplicaConfiguration.CaCertificate
			}

			if vp, okp := _replicaConfiguration["client_certificate"]; okp && vp != nil {
				_replicaConfiguration["client_certificate"] = mySqlReplicaConfiguration.ClientCertificate
			}

			if vp, okp := _replicaConfiguration["client_key"]; okp && vp != nil {
				_replicaConfiguration["client_key"] = mySqlReplicaConfiguration.ClientKey
			}

			if vp, okp := _replicaConfiguration["connect_retry_interval"]; okp && vp != nil {
				_replicaConfiguration["connect_retry_interval"] = mySqlReplicaConfiguration.ConnectRetryInterval
			}

			if vp, okp := _replicaConfiguration["dump_file_path"]; okp && vp != nil {
				_replicaConfiguration["dump_file_path"] = mySqlReplicaConfiguration.DumpFilePath
			}

			if vp, okp := _replicaConfiguration["master_heartbeat_period"]; okp && vp != nil {
				_replicaConfiguration["master_heartbeat_period"] = mySqlReplicaConfiguration.MasterHeartbeatPeriod
			}

			if vp, okp := _replicaConfiguration["password"]; okp && vp != nil {
				_replicaConfiguration["password"] = mySqlReplicaConfiguration.Password
			}

			if vp, okp := _replicaConfiguration["ssl_cipher"]; okp && vp != nil {
				_replicaConfiguration["ssl_cipher"] = mySqlReplicaConfiguration.SslCipher
			}

			if vp, okp := _replicaConfiguration["username"]; okp && vp != nil {
				_replicaConfiguration["username"] = mySqlReplicaConfiguration.Username
			}

			if vp, okp := _replicaConfiguration["verify_server_certificate"]; okp && vp != nil {
				_replicaConfiguration["verify_server_certificate"] = mySqlReplicaConfiguration.VerifyServerCertificate
			}

			_replicaConfigurationList[0] = _replicaConfiguration
			d.Set("replica_configuration", _replicaConfigurationList)
		}
	}

	_ipAddresses := make([]interface{}, len(instance.IpAddresses))

	for i, ip := range instance.IpAddresses {
		_ipAddress := make(map[string]interface{})

		_ipAddress["ip_address"] = ip.IpAddress
		_ipAddress["time_to_retire"] = ip.TimeToRetire

		_ipAddresses[i] = _ipAddress
	}

	d.Set("ip_address", _ipAddresses)

	if v, ok := d.GetOk("master_instance_name"); ok && v != nil {
		d.Set("master_instance_name", instance.MasterInstanceName)
	}

	d.Set("self_link", instance.SelfLink)
	d.SetId(instance.Name)

	return nil
}

func resourceSqlDatabaseInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	d.Partial(true)

	instance, err := config.clientSqlAdmin.Instances.Get(project,
		d.Get("name").(string)).Do()

	if err != nil {
		return fmt.Errorf("Error retrieving instance %s: %s",
			d.Get("name").(string), err)
	}

	if d.HasChange("settings") {
		_oListCast, _settingsListCast := d.GetChange("settings")
		_oList := _oListCast.([]interface{})
		_o := _oList[0].(map[string]interface{})
		_settingsList := _settingsListCast.([]interface{})
		if len(_settingsList) > 1 {
			return fmt.Errorf("At most one settings block is allowed")
		}

		_settings := _settingsList[0].(map[string]interface{})
		settings := &sqladmin.Settings{
			Tier:            _settings["tier"].(string),
			SettingsVersion: instance.Settings.SettingsVersion,
		}

		if v, ok := _settings["activation_policy"]; ok {
			settings.ActivationPolicy = v.(string)
		}

		if v, ok := _settings["authorized_gae_applications"]; ok {
			settings.AuthorizedGaeApplications = make([]string, 0)
			for _, app := range v.([]interface{}) {
				settings.AuthorizedGaeApplications = append(settings.AuthorizedGaeApplications,
					app.(string))
			}
		}

		if v, ok := _settings["backup_configuration"]; ok {
			_backupConfigurationList := v.([]interface{})
			if len(_backupConfigurationList) > 1 {
				return fmt.Errorf("At most one backup_configuration block is allowed")
			}

			if len(_backupConfigurationList) == 1 && _backupConfigurationList[0] != nil {
				settings.BackupConfiguration = &sqladmin.BackupConfiguration{}
				_backupConfiguration := _backupConfigurationList[0].(map[string]interface{})

				if vp, okp := _backupConfiguration["binary_log_enabled"]; okp {
					settings.BackupConfiguration.BinaryLogEnabled = vp.(bool)
				}

				if vp, okp := _backupConfiguration["enabled"]; okp {
					settings.BackupConfiguration.Enabled = vp.(bool)
				}

				if vp, okp := _backupConfiguration["start_time"]; okp {
					settings.BackupConfiguration.StartTime = vp.(string)
				}
			}
		}

		if v, ok := _settings["crash_safe_replication"]; ok {
			settings.CrashSafeReplicationEnabled = v.(bool)
		}

		_oldDatabaseFlags := make([]interface{}, 0)
		if ov, ook := _o["database_flags"]; ook {
			_oldDatabaseFlags = ov.([]interface{})
		}

		if v, ok := _settings["database_flags"]; ok || len(_oldDatabaseFlags) > 0 {
			oldDatabaseFlags := settings.DatabaseFlags
			settings.DatabaseFlags = make([]*sqladmin.DatabaseFlags, 0)
			_databaseFlagsList := make([]interface{}, 0)
			if v != nil {
				_databaseFlagsList = v.([]interface{})
			}

			_odbf_map := make(map[string]interface{})
			for _, _dbf := range _oldDatabaseFlags {
				_entry := _dbf.(map[string]interface{})
				_odbf_map[_entry["name"].(string)] = true
			}

			// First read the flags from the server, and reinsert those that
			// were not previously defined
			for _, entry := range oldDatabaseFlags {
				_, ok_old := _odbf_map[entry.Name]
				if !ok_old {
					settings.DatabaseFlags = append(
						settings.DatabaseFlags, entry)
				}
			}
			// finally, insert only those that were previously defined
			// and are still defined.
			for _, _flag := range _databaseFlagsList {
				_entry := _flag.(map[string]interface{})
				flag := &sqladmin.DatabaseFlags{}
				if vp, okp := _entry["name"]; okp {
					flag.Name = vp.(string)
				}

				if vp, okp := _entry["value"]; okp {
					flag.Value = vp.(string)
				}

				settings.DatabaseFlags = append(settings.DatabaseFlags, flag)
			}
		}

		if v, ok := _settings["ip_configuration"]; ok {
			_ipConfigurationList := v.([]interface{})
			if len(_ipConfigurationList) > 1 {
				return fmt.Errorf("At most one ip_configuration block is allowed")
			}

			if len(_ipConfigurationList) == 1 && _ipConfigurationList[0] != nil {
				settings.IpConfiguration = &sqladmin.IpConfiguration{}
				_ipConfiguration := _ipConfigurationList[0].(map[string]interface{})

				if vp, okp := _ipConfiguration["ipv4_enabled"]; okp {
					settings.IpConfiguration.Ipv4Enabled = vp.(bool)
				}

				if vp, okp := _ipConfiguration["require_ssl"]; okp {
					settings.IpConfiguration.RequireSsl = vp.(bool)
				}

				_oldAuthorizedNetworkList := make([]interface{}, 0)
				if ov, ook := _o["ip_configuration"]; ook {
					_oldIpConfList := ov.([]interface{})
					if len(_oldIpConfList) > 0 {
						_oldIpConf := _oldIpConfList[0].(map[string]interface{})
						if ovp, ookp := _oldIpConf["authorized_networks"]; ookp {
							_oldAuthorizedNetworkList = ovp.([]interface{})
						}
					}
				}

				if vp, okp := _ipConfiguration["authorized_networks"]; okp || len(_oldAuthorizedNetworkList) > 0 {
					oldAuthorizedNetworks := settings.IpConfiguration.AuthorizedNetworks
					settings.IpConfiguration.AuthorizedNetworks = make([]*sqladmin.AclEntry, 0)

					_authorizedNetworksList := make([]interface{}, 0)
					if vp != nil {
						_authorizedNetworksList = vp.([]interface{})
					}
					_oipc_map := make(map[string]interface{})
					for _, _ipc := range _oldAuthorizedNetworkList {
						_entry := _ipc.(map[string]interface{})
						_oipc_map[_entry["value"].(string)] = true
					}
					// Next read the network tuples from the server, and reinsert those that
					// were not previously defined
					for _, entry := range oldAuthorizedNetworks {
						_, ok_old := _oipc_map[entry.Value]
						if !ok_old {
							settings.IpConfiguration.AuthorizedNetworks = append(
								settings.IpConfiguration.AuthorizedNetworks, entry)
						}
					}
					// finally, insert only those that were previously defined
					// and are still defined.
					for _, _ipc := range _authorizedNetworksList {
						_entry := _ipc.(map[string]interface{})
						if _, ok_old := _oipc_map[_entry["value"].(string)]; ok_old {
							entry := &sqladmin.AclEntry{}

							if vpp, okpp := _entry["expiration_time"]; okpp {
								entry.ExpirationTime = vpp.(string)
							}

							if vpp, okpp := _entry["name"]; okpp {
								entry.Name = vpp.(string)
							}

							if vpp, okpp := _entry["value"]; okpp {
								entry.Value = vpp.(string)
							}

							settings.IpConfiguration.AuthorizedNetworks = append(
								settings.IpConfiguration.AuthorizedNetworks, entry)
						}
					}
				}
			}
		}

		if v, ok := _settings["location_preference"]; ok {
			_locationPreferenceList := v.([]interface{})
			if len(_locationPreferenceList) > 1 {
				return fmt.Errorf("At most one location_preference block is allowed")
			}

			if len(_locationPreferenceList) == 1 && _locationPreferenceList[0] != nil {
				settings.LocationPreference = &sqladmin.LocationPreference{}
				_locationPreference := _locationPreferenceList[0].(map[string]interface{})

				if vp, okp := _locationPreference["follow_gae_application"]; okp {
					settings.LocationPreference.FollowGaeApplication = vp.(string)
				}

				if vp, okp := _locationPreference["zone"]; okp {
					settings.LocationPreference.Zone = vp.(string)
				}
			}
		}

		if v, ok := _settings["pricing_plan"]; ok {
			settings.PricingPlan = v.(string)
		}

		if v, ok := _settings["replication_type"]; ok {
			settings.ReplicationType = v.(string)
		}

		instance.Settings = settings
	}

	d.Partial(false)

	op, err := config.clientSqlAdmin.Instances.Update(project, instance.Name, instance).Do()
	if err != nil {
		return fmt.Errorf("Error, failed to update instance %s: %s", instance.Name, err)
	}

	err = sqladminOperationWait(config, op, "Create Instance")
	if err != nil {
		return err
	}

	return resourceSqlDatabaseInstanceRead(d, meta)
}

func resourceSqlDatabaseInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	op, err := config.clientSqlAdmin.Instances.Delete(project, d.Get("name").(string)).Do()

	if err != nil {
		return fmt.Errorf("Error, failed to delete instance %s: %s", d.Get("name").(string), err)
	}

	err = sqladminOperationWait(config, op, "Delete Instance")
	if err != nil {
		return err
	}

	return nil
}
