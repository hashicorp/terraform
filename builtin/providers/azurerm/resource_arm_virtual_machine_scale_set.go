package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmVirtualMachineScaleSet() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmVirtualMachineScaleSetCreate,
		Read:   resourceArmVirtualMachineScaleSetRead,
		Update: resourceArmVirtualMachineScaleSetCreate,
		Delete: resourceArmVirtualMachineScaleSetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"sku": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"tier": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceArmVirtualMachineScaleSetSkuHash,
			},

			"upgrade_policy_mode": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"os_profile": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"computer_name_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"admin_username": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"admin_password": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"custom_data": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceArmVirtualMachineScaleSetsOsProfileHash,
			},

			"os_profile_secrets": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source_vault_id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"vault_certificates": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"certificate_url": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"certificate_store": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},

			"os_profile_windows_config": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"provision_vm_agent": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"enable_automatic_upgrades": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"winrm": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"protocol": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"certificate_url": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"additional_unattend_config": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"pass": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"component": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"setting_name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"content": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
				Set: resourceArmVirtualMachineScaleSetOsProfileLWindowsConfigHash,
			},

			"os_profile_linux_config": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"disable_password_authentication": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
							ForceNew: true,
						},
						"ssh_keys": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"path": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},
									"key_data": &schema.Schema{
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
				Set: resourceArmVirtualMachineScaleSetOsProfileLinuxConfigHash,
			},

			"network_profile": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"primary": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
						},

						"ip_configuration": &schema.Schema{
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},

									"subnet_id": &schema.Schema{
										Type:     schema.TypeString,
										Required: true,
									},

									"load_balancer_backend_address_pool_ids": &schema.Schema{
										Type:     schema.TypeSet,
										Optional: true,
										Computed: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Set:      schema.HashString,
									},
								},
							},
						},
					},
				},
				Set: resourceArmVirtualMachineScaleSetNetworkConfigurationHash,
			},

			"storage_profile_os_disk": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"image": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"vhd_containers": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
						},

						"caching": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"os_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"create_option": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceArmVirtualMachineScaleSetStorageProfileOsDiskHash,
			},

			"storage_profile_image_reference": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"publisher": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"offer": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"sku": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"version": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceArmVirtualMachineScaleSetStorageProfileImageReferenceHash,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmVirtualMachineScaleSetCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	vmScaleSetClient := client.vmScaleSetClient

	log.Printf("[INFO] preparing arguments for Azure ARM Virtual Machine Scale Set creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	sku, err := expandVirtualMachineScaleSetSku(d)
	if err != nil {
		return err
	}

	storageProfile := compute.VirtualMachineScaleSetStorageProfile{}
	osDisk, err := expandAzureRMVirtualMachineScaleSetsStorageProfileOsDisk(d)
	if err != nil {
		return err
	}
	storageProfile.OsDisk = osDisk
	if _, ok := d.GetOk("storage_profile_image_reference"); ok {
		imageRef, err := expandAzureRmVirtualMachineScaleSetStorageProfileImageReference(d)
		if err != nil {
			return err
		}
		storageProfile.ImageReference = imageRef
	}

	osProfile, err := expandAzureRMVirtualMachineScaleSetsOsProfile(d)
	if err != nil {
		return err
	}

	updatePolicy := d.Get("upgrade_policy_mode").(string)
	scaleSetProps := compute.VirtualMachineScaleSetProperties{
		UpgradePolicy: &compute.UpgradePolicy{
			Mode: compute.UpgradeMode(updatePolicy),
		},
		VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
			NetworkProfile: expandAzureRmVirtualMachineScaleSetNetworkProfile(d),
			StorageProfile: &storageProfile,
			OsProfile:      osProfile,
		},
	}

	scaleSetParams := compute.VirtualMachineScaleSet{
		Name:     &name,
		Location: &location,
		Tags:     expandTags(tags),
		Sku:      sku,
		VirtualMachineScaleSetProperties: &scaleSetProps,
	}
	_, vmErr := vmScaleSetClient.CreateOrUpdate(resGroup, name, scaleSetParams, make(chan struct{}))
	if vmErr != nil {
		return vmErr
	}

	read, err := vmScaleSetClient.Get(resGroup, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Virtual Machine Scale Set %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmVirtualMachineScaleSetRead(d, meta)
}

func resourceArmVirtualMachineScaleSetRead(d *schema.ResourceData, meta interface{}) error {
	vmScaleSetClient := meta.(*ArmClient).vmScaleSetClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["virtualMachineScaleSets"]

	resp, err := vmScaleSetClient.Get(resGroup, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			log.Printf("[INFO] AzureRM Virtual Machine Scale Set (%s) Not Found. Removing from State", name)
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure Virtual Machine Scale Set %s: %s", name, err)
	}

	d.Set("location", resp.Location)
	d.Set("name", resp.Name)

	if err := d.Set("sku", flattenAzureRmVirtualMachineScaleSetSku(resp.Sku)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting Virtual Machine Scale Set Sku error: %#v", err)
	}

	properties := resp.VirtualMachineScaleSetProperties

	d.Set("upgrade_policy_mode", properties.UpgradePolicy.Mode)

	if err := d.Set("os_profile", flattenAzureRMVirtualMachineScaleSetOsProfile(properties.VirtualMachineProfile.OsProfile)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting Virtual Machine Scale Set OS Profile error: %#v", err)
	}

	if properties.VirtualMachineProfile.OsProfile.Secrets != nil {
		if err := d.Set("os_profile_secrets", flattenAzureRmVirtualMachineScaleSetOsProfileSecrets(properties.VirtualMachineProfile.OsProfile.Secrets)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Scale Set OS Profile Secrets error: %#v", err)
		}
	}

	if properties.VirtualMachineProfile.OsProfile.WindowsConfiguration != nil {
		if err := d.Set("os_profile_windows_config", flattenAzureRmVirtualMachineScaleSetOsProfileWindowsConfig(properties.VirtualMachineProfile.OsProfile.WindowsConfiguration)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Scale Set OS Profile Windows config error: %#v", err)
		}
	}

	if properties.VirtualMachineProfile.OsProfile.LinuxConfiguration != nil {
		if err := d.Set("os_profile_linux_config", flattenAzureRmVirtualMachineScaleSetOsProfileLinuxConfig(properties.VirtualMachineProfile.OsProfile.LinuxConfiguration)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Scale Set OS Profile Windows config error: %#v", err)
		}
	}

	if err := d.Set("network_profile", flattenAzureRmVirtualMachineScaleSetNetworkProfile(properties.VirtualMachineProfile.NetworkProfile)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting Virtual Machine Scale Set Network Profile error: %#v", err)
	}

	if properties.VirtualMachineProfile.StorageProfile.ImageReference != nil {
		if err := d.Set("storage_profile_image_reference", flattenAzureRmVirtualMachineScaleSetStorageProfileImageReference(properties.VirtualMachineProfile.StorageProfile.ImageReference)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Scale Set Storage Profile Image Reference error: %#v", err)
		}
	}

	if err := d.Set("storage_profile_os_disk", flattenAzureRmVirtualMachineScaleSetStorageProfileOSDisk(properties.VirtualMachineProfile.StorageProfile.OsDisk)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting Virtual Machine Scale Set Storage Profile OS Disk error: %#v", err)
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmVirtualMachineScaleSetDelete(d *schema.ResourceData, meta interface{}) error {
	vmScaleSetClient := meta.(*ArmClient).vmScaleSetClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["virtualMachineScaleSets"]

	_, err = vmScaleSetClient.Delete(resGroup, name, make(chan struct{}))

	return err
}

func flattenAzureRmVirtualMachineScaleSetOsProfileLinuxConfig(config *compute.LinuxConfiguration) []interface{} {
	result := make(map[string]interface{})
	result["disable_password_authentication"] = *config.DisablePasswordAuthentication

	if config.SSH != nil && len(*config.SSH.PublicKeys) > 0 {
		ssh_keys := make([]map[string]interface{}, len(*config.SSH.PublicKeys))
		for _, i := range *config.SSH.PublicKeys {
			key := make(map[string]interface{})
			key["path"] = *i.Path

			if i.KeyData != nil {
				key["key_data"] = *i.KeyData
			}

			ssh_keys = append(ssh_keys, key)
		}

		result["ssh_keys"] = ssh_keys
	}

	return []interface{}{result}
}

func flattenAzureRmVirtualMachineScaleSetOsProfileWindowsConfig(config *compute.WindowsConfiguration) []interface{} {
	result := make(map[string]interface{})

	if config.ProvisionVMAgent != nil {
		result["provision_vm_agent"] = *config.ProvisionVMAgent
	}

	if config.EnableAutomaticUpdates != nil {
		result["enable_automatic_upgrades"] = *config.EnableAutomaticUpdates
	}

	if config.WinRM != nil {
		listeners := make([]map[string]interface{}, 0, len(*config.WinRM.Listeners))
		for _, i := range *config.WinRM.Listeners {
			listener := make(map[string]interface{})
			listener["protocol"] = i.Protocol

			if i.CertificateURL != nil {
				listener["certificate_url"] = *i.CertificateURL
			}

			listeners = append(listeners, listener)
		}

		result["winrm"] = listeners
	}

	if config.AdditionalUnattendContent != nil {
		content := make([]map[string]interface{}, 0, len(*config.AdditionalUnattendContent))
		for _, i := range *config.AdditionalUnattendContent {
			c := make(map[string]interface{})
			c["pass"] = i.PassName
			c["component"] = i.ComponentName
			c["setting_name"] = i.SettingName
			c["content"] = *i.Content

			content = append(content, c)
		}

		result["additional_unattend_config"] = content
	}

	return []interface{}{result}
}

func flattenAzureRmVirtualMachineScaleSetOsProfileSecrets(secrets *[]compute.VaultSecretGroup) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(*secrets))
	for _, secret := range *secrets {
		s := map[string]interface{}{
			"source_vault_id": *secret.SourceVault.ID,
		}

		if secret.VaultCertificates != nil {
			certs := make([]map[string]interface{}, 0, len(*secret.VaultCertificates))
			for _, cert := range *secret.VaultCertificates {
				vaultCert := make(map[string]interface{})
				vaultCert["certificate_url"] = *cert.CertificateURL

				if cert.CertificateStore != nil {
					vaultCert["certificate_store"] = *cert.CertificateStore
				}

				certs = append(certs, vaultCert)
			}

			s["vault_certificates"] = certs
		}

		result = append(result, s)
	}
	return result
}

func flattenAzureRmVirtualMachineScaleSetNetworkProfile(profile *compute.VirtualMachineScaleSetNetworkProfile) []map[string]interface{} {
	networkConfigurations := profile.NetworkInterfaceConfigurations
	result := make([]map[string]interface{}, 0, len(*networkConfigurations))
	for _, netConfig := range *networkConfigurations {
		s := map[string]interface{}{
			"name":    *netConfig.Name,
			"primary": *netConfig.VirtualMachineScaleSetNetworkConfigurationProperties.Primary,
		}

		if netConfig.VirtualMachineScaleSetNetworkConfigurationProperties.IPConfigurations != nil {
			ipConfigs := make([]map[string]interface{}, 0, len(*netConfig.VirtualMachineScaleSetNetworkConfigurationProperties.IPConfigurations))
			for _, ipConfig := range *netConfig.VirtualMachineScaleSetNetworkConfigurationProperties.IPConfigurations {
				config := make(map[string]interface{})
				config["name"] = *ipConfig.Name

				properties := ipConfig.VirtualMachineScaleSetIPConfigurationProperties

				if ipConfig.VirtualMachineScaleSetIPConfigurationProperties.Subnet != nil {
					config["subnet_id"] = *properties.Subnet.ID
				}

				if properties.LoadBalancerBackendAddressPools != nil {
					addressPools := make([]string, 0, len(*properties.LoadBalancerBackendAddressPools))
					for _, pool := range *properties.LoadBalancerBackendAddressPools {
						addressPools = append(addressPools, *pool.ID)
					}
					config["load_balancer_backend_address_pool_ids"] = addressPools
				}
			}

			s["ip_configuration"] = ipConfigs
		}

		result = append(result, s)
	}

	return result
}

func flattenAzureRMVirtualMachineScaleSetOsProfile(profile *compute.VirtualMachineScaleSetOSProfile) []interface{} {
	result := make(map[string]interface{})

	result["computer_name_prefix"] = *profile.ComputerNamePrefix
	result["admin_username"] = *profile.AdminUsername

	if profile.CustomData != nil {
		result["custom_data"] = *profile.CustomData
	}

	return []interface{}{result}
}

func flattenAzureRmVirtualMachineScaleSetStorageProfileOSDisk(profile *compute.VirtualMachineScaleSetOSDisk) []interface{} {
	result := make(map[string]interface{})
	result["name"] = *profile.Name
	if profile.Image != nil {
		result["image"] = *profile.Image.URI
	}

	containers := make([]interface{}, 0, len(*profile.VhdContainers))
	for _, container := range *profile.VhdContainers {
		containers = append(containers, container)
	}
	result["vhd_containers"] = schema.NewSet(schema.HashString, containers)

	result["caching"] = profile.Caching
	result["create_option"] = profile.CreateOption

	return []interface{}{result}
}

func flattenAzureRmVirtualMachineScaleSetStorageProfileImageReference(profile *compute.ImageReference) []interface{} {
	result := make(map[string]interface{})
	result["publisher"] = *profile.Publisher
	result["offer"] = *profile.Offer
	result["sku"] = *profile.Sku
	result["version"] = *profile.Version

	return []interface{}{result}
}

func flattenAzureRmVirtualMachineScaleSetSku(sku *compute.Sku) []interface{} {
	result := make(map[string]interface{})
	result["name"] = *sku.Name
	result["capacity"] = *sku.Capacity

	if *sku.Tier != "" {
		result["tier"] = *sku.Tier
	}

	return []interface{}{result}
}

func resourceArmVirtualMachineScaleSetStorageProfileImageReferenceHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["publisher"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["offer"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["sku"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["version"].(string)))

	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineScaleSetSkuHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	if m["tier"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", m["tier"].(string)))
	}
	buf.WriteString(fmt.Sprintf("%d-", m["capacity"].(int)))

	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineScaleSetStorageProfileOsDiskHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))

	if m["image"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", m["image"].(string)))
	}
	if m["os_type"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", m["os_type"].(string)))
	}

	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineScaleSetNetworkConfigurationHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%t-", m["primary"].(bool)))
	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineScaleSetsOsProfileHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["computer_name_prefix"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["admin_username"].(string)))
	if m["custom_data"] != nil {
		buf.WriteString(fmt.Sprintf("%s-", m["custom_data"].(string)))
	}
	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineScaleSetOsProfileLinuxConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%t-", m["disable_password_authentication"].(bool)))

	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineScaleSetOsProfileLWindowsConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if m["provision_vm_agent"] != nil {
		buf.WriteString(fmt.Sprintf("%t-", m["provision_vm_agent"].(bool)))
	}
	if m["enable_automatic_upgrades"] != nil {
		buf.WriteString(fmt.Sprintf("%t-", m["enable_automatic_upgrades"].(bool)))
	}
	return hashcode.String(buf.String())
}

func expandVirtualMachineScaleSetSku(d *schema.ResourceData) (*compute.Sku, error) {
	skuConfig := d.Get("sku").(*schema.Set).List()

	config := skuConfig[0].(map[string]interface{})

	name := config["name"].(string)
	tier := config["tier"].(string)
	capacity := int64(config["capacity"].(int))

	sku := &compute.Sku{
		Name:     &name,
		Capacity: &capacity,
	}

	if tier != "" {
		sku.Tier = &tier
	}

	return sku, nil
}

func expandAzureRmVirtualMachineScaleSetNetworkProfile(d *schema.ResourceData) *compute.VirtualMachineScaleSetNetworkProfile {
	scaleSetNetworkProfileConfigs := d.Get("network_profile").(*schema.Set).List()
	networkProfileConfig := make([]compute.VirtualMachineScaleSetNetworkConfiguration, 0, len(scaleSetNetworkProfileConfigs))

	for _, npProfileConfig := range scaleSetNetworkProfileConfigs {
		config := npProfileConfig.(map[string]interface{})

		name := config["name"].(string)
		primary := config["primary"].(bool)

		ipConfigurationConfigs := config["ip_configuration"].([]interface{})
		ipConfigurations := make([]compute.VirtualMachineScaleSetIPConfiguration, 0, len(ipConfigurationConfigs))
		for _, ipConfigConfig := range ipConfigurationConfigs {
			ipconfig := ipConfigConfig.(map[string]interface{})
			name := ipconfig["name"].(string)
			subnetId := ipconfig["subnet_id"].(string)

			ipConfiguration := compute.VirtualMachineScaleSetIPConfiguration{
				Name: &name,
				VirtualMachineScaleSetIPConfigurationProperties: &compute.VirtualMachineScaleSetIPConfigurationProperties{
					Subnet: &compute.APIEntityReference{
						ID: &subnetId,
					},
				},
			}
			//TODO: Add the support for the load balancers when it drops
			//if v := ipconfig["load_balancer_backend_address_pool_ids"]; v != nil {
			//
			//}

			ipConfigurations = append(ipConfigurations, ipConfiguration)
		}

		nProfile := compute.VirtualMachineScaleSetNetworkConfiguration{
			Name: &name,
			VirtualMachineScaleSetNetworkConfigurationProperties: &compute.VirtualMachineScaleSetNetworkConfigurationProperties{
				Primary:          &primary,
				IPConfigurations: &ipConfigurations,
			},
		}

		networkProfileConfig = append(networkProfileConfig, nProfile)
	}

	return &compute.VirtualMachineScaleSetNetworkProfile{
		NetworkInterfaceConfigurations: &networkProfileConfig,
	}
}

func expandAzureRMVirtualMachineScaleSetsOsProfile(d *schema.ResourceData) (*compute.VirtualMachineScaleSetOSProfile, error) {
	osProfileConfigs := d.Get("os_profile").(*schema.Set).List()

	osProfileConfig := osProfileConfigs[0].(map[string]interface{})
	namePrefix := osProfileConfig["computer_name_prefix"].(string)
	username := osProfileConfig["admin_username"].(string)
	password := osProfileConfig["admin_password"].(string)
	customData := osProfileConfig["custom_data"].(string)

	osProfile := &compute.VirtualMachineScaleSetOSProfile{
		ComputerNamePrefix: &namePrefix,
		AdminUsername:      &username,
	}

	if password != "" {
		osProfile.AdminPassword = &password
	}

	if customData != "" {
		osProfile.CustomData = &customData
	}

	if _, ok := d.GetOk("os_profile_secrets"); ok {
		secrets := expandAzureRmVirtualMachineScaleSetOsProfileSecrets(d)
		if secrets != nil {
			osProfile.Secrets = secrets
		}
	}

	if _, ok := d.GetOk("os_profile_linux_config"); ok {
		linuxConfig, err := expandAzureRmVirtualMachineScaleSetOsProfileLinuxConfig(d)
		if err != nil {
			return nil, err
		}
		osProfile.LinuxConfiguration = linuxConfig
	}

	if _, ok := d.GetOk("os_profile_windows_config"); ok {
		winConfig, err := expandAzureRmVirtualMachineScaleSetOsProfileWindowsConfig(d)
		if err != nil {
			return nil, err
		}
		if winConfig != nil {
			osProfile.WindowsConfiguration = winConfig
		}
	}

	return osProfile, nil
}

func expandAzureRMVirtualMachineScaleSetsStorageProfileOsDisk(d *schema.ResourceData) (*compute.VirtualMachineScaleSetOSDisk, error) {
	osDiskConfigs := d.Get("storage_profile_os_disk").(*schema.Set).List()

	osDiskConfig := osDiskConfigs[0].(map[string]interface{})
	name := osDiskConfig["name"].(string)
	image := osDiskConfig["image"].(string)
	caching := osDiskConfig["caching"].(string)
	osType := osDiskConfig["os_type"].(string)
	createOption := osDiskConfig["create_option"].(string)

	var vhdContainers []string
	containers := osDiskConfig["vhd_containers"].(*schema.Set).List()
	for _, v := range containers {
		str := v.(string)
		vhdContainers = append(vhdContainers, str)
	}

	osDisk := &compute.VirtualMachineScaleSetOSDisk{
		Name:          &name,
		Caching:       compute.CachingTypes(caching),
		OsType:        compute.OperatingSystemTypes(osType),
		CreateOption:  compute.DiskCreateOptionTypes(createOption),
		VhdContainers: &vhdContainers,
	}

	if image != "" {
		osDisk.Image = &compute.VirtualHardDisk{
			URI: &image,
		}
	}

	return osDisk, nil

}

func expandAzureRmVirtualMachineScaleSetStorageProfileImageReference(d *schema.ResourceData) (*compute.ImageReference, error) {
	storageImageRefs := d.Get("storage_profile_image_reference").(*schema.Set).List()

	storageImageRef := storageImageRefs[0].(map[string]interface{})

	publisher := storageImageRef["publisher"].(string)
	offer := storageImageRef["offer"].(string)
	sku := storageImageRef["sku"].(string)
	version := storageImageRef["version"].(string)

	return &compute.ImageReference{
		Publisher: &publisher,
		Offer:     &offer,
		Sku:       &sku,
		Version:   &version,
	}, nil
}

func expandAzureRmVirtualMachineScaleSetOsProfileLinuxConfig(d *schema.ResourceData) (*compute.LinuxConfiguration, error) {
	osProfilesLinuxConfig := d.Get("os_profile_linux_config").(*schema.Set).List()

	linuxConfig := osProfilesLinuxConfig[0].(map[string]interface{})
	disablePasswordAuth := linuxConfig["disable_password_authentication"].(bool)

	config := &compute.LinuxConfiguration{
		DisablePasswordAuthentication: &disablePasswordAuth,
	}
	linuxKeys := linuxConfig["ssh_keys"].([]interface{})
	sshPublicKeys := make([]compute.SSHPublicKey, 0, len(linuxKeys))
	for _, key := range linuxKeys {
		sshKey := key.(map[string]interface{})
		path := sshKey["path"].(string)
		keyData := sshKey["key_data"].(string)

		sshPublicKey := compute.SSHPublicKey{
			Path:    &path,
			KeyData: &keyData,
		}

		sshPublicKeys = append(sshPublicKeys, sshPublicKey)
	}

	config.SSH = &compute.SSHConfiguration{
		PublicKeys: &sshPublicKeys,
	}

	return config, nil
}

func expandAzureRmVirtualMachineScaleSetOsProfileWindowsConfig(d *schema.ResourceData) (*compute.WindowsConfiguration, error) {
	osProfilesWindowsConfig := d.Get("os_profile_windows_config").(*schema.Set).List()

	osProfileConfig := osProfilesWindowsConfig[0].(map[string]interface{})
	config := &compute.WindowsConfiguration{}

	if v := osProfileConfig["provision_vm_agent"]; v != nil {
		provision := v.(bool)
		config.ProvisionVMAgent = &provision
	}

	if v := osProfileConfig["enable_automatic_upgrades"]; v != nil {
		update := v.(bool)
		config.EnableAutomaticUpdates = &update
	}

	if v := osProfileConfig["winrm"]; v != nil {
		winRm := v.(*schema.Set).List()
		if len(winRm) > 0 {
			winRmListners := make([]compute.WinRMListener, 0, len(winRm))
			for _, winRmConfig := range winRm {
				config := winRmConfig.(map[string]interface{})

				protocol := config["protocol"].(string)
				winRmListner := compute.WinRMListener{
					Protocol: compute.ProtocolTypes(protocol),
				}
				if v := config["certificate_url"].(string); v != "" {
					winRmListner.CertificateURL = &v
				}

				winRmListners = append(winRmListners, winRmListner)
			}
			config.WinRM = &compute.WinRMConfiguration{
				Listeners: &winRmListners,
			}
		}
	}
	if v := osProfileConfig["additional_unattend_config"]; v != nil {
		additionalConfig := v.(*schema.Set).List()
		if len(additionalConfig) > 0 {
			additionalConfigContent := make([]compute.AdditionalUnattendContent, 0, len(additionalConfig))
			for _, addConfig := range additionalConfig {
				config := addConfig.(map[string]interface{})
				pass := config["pass"].(string)
				component := config["component"].(string)
				settingName := config["setting_name"].(string)
				content := config["content"].(string)

				addContent := compute.AdditionalUnattendContent{
					PassName:      compute.PassNames(pass),
					ComponentName: compute.ComponentNames(component),
					SettingName:   compute.SettingNames(settingName),
					Content:       &content,
				}

				additionalConfigContent = append(additionalConfigContent, addContent)
			}
			config.AdditionalUnattendContent = &additionalConfigContent
		}
	}
	return config, nil
}

func expandAzureRmVirtualMachineScaleSetOsProfileSecrets(d *schema.ResourceData) *[]compute.VaultSecretGroup {
	secretsConfig := d.Get("os_profile_secrets").(*schema.Set).List()
	secrets := make([]compute.VaultSecretGroup, 0, len(secretsConfig))

	for _, secretConfig := range secretsConfig {
		config := secretConfig.(map[string]interface{})
		sourceVaultId := config["source_vault_id"].(string)

		vaultSecretGroup := compute.VaultSecretGroup{
			SourceVault: &compute.SubResource{
				ID: &sourceVaultId,
			},
		}

		if v := config["vault_certificates"]; v != nil {
			certsConfig := v.([]interface{})
			certs := make([]compute.VaultCertificate, 0, len(certsConfig))
			for _, certConfig := range certsConfig {
				config := certConfig.(map[string]interface{})

				certUrl := config["certificate_url"].(string)
				cert := compute.VaultCertificate{
					CertificateURL: &certUrl,
				}
				if v := config["certificate_store"].(string); v != "" {
					cert.CertificateStore = &v
				}

				certs = append(certs, cert)
			}
			vaultSecretGroup.VaultCertificates = &certs
		}

		secrets = append(secrets, vaultSecretGroup)
	}

	return &secrets
}
