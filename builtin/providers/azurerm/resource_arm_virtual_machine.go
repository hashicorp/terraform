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

func resourceArmVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmVirtualMachineCreate,
		Read:   resourceArmVirtualMachineRead,
		Update: resourceArmVirtualMachineUpdate,
		Delete: resourceArmVirtualMachineDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": &schema.Schema{
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"plan": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"publisher": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"product": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceArmVirtualMachinePlanHash,
			},

			"availability_set_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"license_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"vm_size": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"storage_image_reference": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
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
				Set: resourceArmVirtualMachineStorageImageReferenceHash,
			},

			"storage_os_disk": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"vhd_uri": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"caching": &schema.Schema{
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
				Set: resourceArmVirtualMachineStorageOsDiskHash,
			},

			"storage_data_disk": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"vhd_uri": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"create_option": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"disk_size_gb": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"lun": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
				Set: resourceArmVirtualMachineStorageDataDiskHash,
			},

			"os_profile": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"computer_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
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
							Computed: true,
						},
					},
				},
				Set: resourceArmVirtualMachineStorageOsProfileHash,
			},

			"os_profile_windows_config": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
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
			},

			"os_profile_linux_config": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"disable_password_authentication": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"ssh_keys": &schema.Schema{
							Type:     schema.TypeSet,
							Required: true,
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
							Type:     schema.TypeSet,
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

			"network_interface_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceArmVirtualMachineCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	vmClient := client.vmClient

	log.Printf("[INFO] preparing arguments for Azure ARM Virtual Machine creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)

	osDisk, err := expandAzureRmVirtualMachineOsDisk(d)
	if err != nil {
		return err
	}
	storageProfile := compute.StorageProfile{
		OsDisk: osDisk,
	}

	if _, ok := d.GetOk("storage_image_reference"); ok {
		imageRef, err := expandAzureRmVirtualMachineImageReference(d)
		if err != nil {
			return err
		}
		storageProfile.ImageReference = imageRef
	}

	if _, ok := d.GetOk("storage_data_disk"); ok {
		dataDisks, err := expandAzureRmVirtualMachineDataDisk(d)
		if err != nil {
			return err
		}
		storageProfile.DataDisks = &dataDisks
	}

	networkProfile := expandAzureRmVirtualMachineNetworkProfile(d)
	vmSize := d.Get("vm_size").(string)
	properties := compute.VirtualMachineProperties{
		NetworkProfile: &networkProfile,
		HardwareProfile: &compute.HardwareProfile{
			VMSize: compute.VirtualMachineSizeTypes(vmSize),
		},
		StorageProfile: &storageProfile,
	}

	osProfile, err := expandAzureRmVirtualMachineOsProfile(d)
	if err != nil {
		return err
	}
	properties.OsProfile = osProfile

	if v, ok := d.GetOk("availability_set_id"); ok {
		availabilitySet := v.(string)
		availSet := compute.SubResource{
			ID: &availabilitySet,
		}

		properties.AvailabilitySet = &availSet
	}

	vm := compute.VirtualMachine{
		Name:       &name,
		Location:   &location,
		Properties: &properties,
	}

	if _, ok := d.GetOk("plan"); ok {
		plan, err := expandAzureRmVirtualMachinePlan(d)
		if err != nil {
			return err
		}

		vm.Plan = plan
	}

	_, err = vmClient.CreateOrUpdate(resGroup, name, vm)
	if err != nil {
		return err
	}

	return resourceArmVirtualMachineRead(d, meta)
}

func resourceArmVirtualMachineRead(d *schema.ResourceData, meta interface{}) error {
	vmClient := meta.(*ArmClient).vmClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["virtualMachines"]

	resp, err := vmClient.Get(resGroup, name, "")
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Virtual Machine %s: %s", name, err)
	}
	return nil
}

func resourceArmVirtualMachineUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceArmVirtualMachineRead(d, meta)
}

func resourceArmVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	vmClient := meta.(*ArmClient).vmClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["virtualMachines"]

	_, err = vmClient.Delete(resGroup, name)

	return err
}

func resourceArmVirtualMachinePlanHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["publisher"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["product"].(string)))

	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineStorageImageReferenceHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["publisher"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["offer"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["sku"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["version"].(string)))

	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineStorageOsProfileHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["admin_username"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["admin_password"].(string)))
	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineStorageDataDiskHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["vhd_uri"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["create_option"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["disk_size_gb"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["lun"].(int)))

	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineStorageOsDiskHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["vhd_uri"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["create_option"].(string)))

	return hashcode.String(buf.String())
}

func expandAzureRmVirtualMachinePlan(d *schema.ResourceData) (*compute.Plan, error) {
	planConfigs := d.Get("plan").(*schema.Set).List()

	if len(planConfigs) != 1 {
		return nil, fmt.Errorf("Cannot specify more than one plan.")
	}

	planConfig := planConfigs[0].(map[string]interface{})

	publisher := planConfig["publisher"].(string)
	name := planConfig["name"].(string)
	product := planConfig["product"].(string)

	return &compute.Plan{
		Publisher: &publisher,
		Name:      &name,
		Product:   &product,
	}, nil
}

func expandAzureRmVirtualMachineOsProfile(d *schema.ResourceData) (*compute.OSProfile, error) {
	osProfiles := d.Get("os_profile").(*schema.Set).List()

	if len(osProfiles) != 1 {
		return nil, fmt.Errorf("[ERROR] Only 1 OS Profile Can be specified for an Azure RM Virtual Machine")
	}

	osProfile := osProfiles[0].(map[string]interface{})

	adminUsername := osProfile["admin_username"].(string)
	adminPassword := osProfile["admin_password"].(string)

	profile := &compute.OSProfile{
		AdminUsername:        &adminUsername,
		AdminPassword:        &adminPassword,
		WindowsConfiguration: &compute.WindowsConfiguration{},
		LinuxConfiguration:   &compute.LinuxConfiguration{},
	}

	if _, ok := d.GetOk("os_profile_windows_config"); ok {
		winConfig, err := expandAzureRmVirtualMachineOsProfileWindowsConfig(d)
		if err != nil {
			return nil, err
		}
		if winConfig != nil {
			profile.WindowsConfiguration = winConfig
		}
	}

	if _, ok := d.GetOk("os_profile_linux_config"); ok {
		linuxConfig, err := expandAzureRmVirtualMachineOsProfileLinuxConfig(d)
		if err != nil {
			return nil, err
		}
		if linuxConfig != nil {
			profile.LinuxConfiguration = linuxConfig
		}
	}

	if _, ok := d.GetOk("os_profile_secrets"); ok {
		secrets := expandAzureRmVirtualMachineOsProfileSecrets(d)
		if secrets != nil {
			profile.Secrets = secrets
		}
	}

	if v := osProfile["computer_name"].(string); v != "" {
		profile.ComputerName = &v
	}
	if v := osProfile["custom_data"].(string); v != "" {
		profile.CustomData = &v
	}

	return profile, nil
}

func expandAzureRmVirtualMachineOsProfileSecrets(d *schema.ResourceData) *[]compute.VaultSecretGroup {
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
			certsConfig := v.(*schema.Set).List()
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

func expandAzureRmVirtualMachineOsProfileLinuxConfig(d *schema.ResourceData) (*compute.LinuxConfiguration, error) {
	osProfilesLinuxConfig := d.Get("os_profile_linux_config").(*schema.Set).List()

	if len(osProfilesLinuxConfig) != 1 {
		return nil, fmt.Errorf("[ERROR] Only 1 OS Profile Linux Config Can be specified for an Azure RM Virtual Machine")
	}

	linuxConfig := osProfilesLinuxConfig[0].(map[string]interface{})
	disablePasswordAuth := linuxConfig["disable_password_authentication"].(bool)

	config := &compute.LinuxConfiguration{
		DisablePasswordAuthentication: &disablePasswordAuth,
	}

	linuxKeys := linuxConfig["ssh_keys"].(*schema.Set).List()
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

func expandAzureRmVirtualMachineOsProfileWindowsConfig(d *schema.ResourceData) (*compute.WindowsConfiguration, error) {
	osProfilesWindowsConfig := d.Get("os_profile_windows_config").(*schema.Set).List()

	if len(osProfilesWindowsConfig) != 1 {
		return nil, fmt.Errorf("[ERROR] Only 1 OS Profile Windows Config Can be specified for an Azure RM Virtual Machine")
	}

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
	if v := osProfileConfig["additional_unattend_config"]; v != nil {
		additionalConfig := v.(*schema.Set).List()
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
	return config, nil
}

func expandAzureRmVirtualMachineDataDisk(d *schema.ResourceData) ([]compute.DataDisk, error) {
	disks := d.Get("storage_data_disk").([]interface{})
	data_disks := make([]compute.DataDisk, 0, len(disks))
	for _, disk_config := range disks {
		config := disk_config.(map[string]interface{})

		name := config["name"].(string)
		vhd := config["vhd_uri"].(string)
		createOption := config["create_option"].(string)
		lun := config["lun"].(int)
		disk_size := config["disk_size_gb"].(int)

		data_disk := compute.DataDisk{
			Name: &name,
			Vhd: &compute.VirtualHardDisk{
				URI: &vhd,
			},
			Lun:          &lun,
			DiskSizeGB:   &disk_size,
			CreateOption: compute.DiskCreateOptionTypes(createOption),
		}

		data_disks = append(data_disks, data_disk)
	}

	return data_disks, nil
}

func expandAzureRmVirtualMachineImageReference(d *schema.ResourceData) (*compute.ImageReference, error) {
	storageImageRefs := d.Get("storage_image_reference").(*schema.Set).List()

	if len(storageImageRefs) != 1 {
		return nil, fmt.Errorf("Cannot specify more than one storage_image_reference.")
	}

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

func expandAzureRmVirtualMachineNetworkProfile(d *schema.ResourceData) compute.NetworkProfile {
	nicIds := d.Get("network_interface_ids").(*schema.Set).List()
	network_interfaces := make([]compute.NetworkInterfaceReference, 0, len(nicIds))

	network_profile := compute.NetworkProfile{}

	for _, nic := range nicIds {
		id := nic.(string)
		network_interface := compute.NetworkInterfaceReference{
			ID: &id,
		}
		network_interfaces = append(network_interfaces, network_interface)
	}

	network_profile.NetworkInterfaces = &network_interfaces

	return network_profile
}

func expandAzureRmVirtualMachineOsDisk(d *schema.ResourceData) (*compute.OSDisk, error) {
	disks := d.Get("storage_os_disk").(*schema.Set).List()

	if len(disks) != 1 {
		return nil, fmt.Errorf("[ERROR] Only 1 OS Disk Can be specified for an Azure RM Virtual Machine")
	}

	disk := disks[0].(map[string]interface{})

	name := disk["name"].(string)
	vhdURI := disk["vhd_uri"].(string)
	createOption := disk["create_option"].(string)

	osDisk := &compute.OSDisk{
		Name: &name,
		Vhd: &compute.VirtualHardDisk{
			URI: &vhdURI,
		},
		CreateOption: compute.DiskCreateOptionTypes(createOption),
	}

	if v := disk["caching"].(string); v != "" {
		osDisk.Caching = compute.CachingTypes(v)
	}

	return osDisk, nil
}
