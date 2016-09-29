package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	riviera "github.com/jen20/riviera/azure"
)

func resourceArmVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmVirtualMachineCreate,
		Read:   resourceArmVirtualMachineRead,
		Update: resourceArmVirtualMachineCreate,
		Delete: resourceArmVirtualMachineDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": {
				Type:      schema.TypeString,
				Required:  true,
				ForceNew:  true,
				StateFunc: azureRMNormalizeLocation,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"plan": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"publisher": {
							Type:     schema.TypeString,
							Required: true,
						},

						"product": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceArmVirtualMachinePlanHash,
			},

			"availability_set_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				StateFunc: func(id interface{}) string {
					return strings.ToLower(id.(string))
				},
			},

			"license_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"vm_size": {
				Type:     schema.TypeString,
				Required: true,
			},

			"storage_image_reference": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"publisher": {
							Type:     schema.TypeString,
							Required: true,
						},

						"offer": {
							Type:     schema.TypeString,
							Required: true,
						},

						"sku": {
							Type:     schema.TypeString,
							Required: true,
						},

						"version": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceArmVirtualMachineStorageImageReferenceHash,
			},

			"storage_os_disk": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"os_type": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"vhd_uri": {
							Type:     schema.TypeString,
							Required: true,
						},

						"image_uri": {
							Type:     schema.TypeString,
							Optional: true,
						},

						"caching": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},

						"create_option": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceArmVirtualMachineStorageOsDiskHash,
			},

			"delete_os_disk_on_termination": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"storage_data_disk": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},

						"vhd_uri": {
							Type:     schema.TypeString,
							Required: true,
						},

						"create_option": {
							Type:     schema.TypeString,
							Required: true,
						},

						"disk_size_gb": {
							Type:     schema.TypeInt,
							Required: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
								value := v.(int)
								if value < 1 || value > 1023 {
									errors = append(errors, fmt.Errorf(
										"The `disk_size_gb` can only be between 1 and 1023"))
								}
								return
							},
						},

						"lun": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},

			"delete_data_disks_on_termination": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"diagnostics_profile": {
				Type:          schema.TypeSet,
				Optional:      true,
				MaxItems:      1,
				ConflictsWith: []string{"boot_diagnostics"},
				Deprecated:    "Use field boot_diagnostics instead",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"boot_diagnostics": {
							Type:     schema.TypeSet,
							Required: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"enabled": {
										Type:     schema.TypeBool,
										Required: true,
									},

									"storage_uri": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},

			"boot_diagnostics": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Required: true,
						},

						"storage_uri": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"os_profile": {
				Type:     schema.TypeSet,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"computer_name": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},

						"admin_username": {
							Type:     schema.TypeString,
							Required: true,
						},

						"admin_password": {
							Type:     schema.TypeString,
							Required: true,
						},

						"custom_data": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
						},
					},
				},
				Set: resourceArmVirtualMachineStorageOsProfileHash,
			},

			"os_profile_windows_config": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"provision_vm_agent": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"enable_automatic_upgrades": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"winrm": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"protocol": {
										Type:     schema.TypeString,
										Required: true,
									},
									"certificate_url": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"additional_unattend_config": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"pass": {
										Type:     schema.TypeString,
										Required: true,
									},
									"component": {
										Type:     schema.TypeString,
										Required: true,
									},
									"setting_name": {
										Type:     schema.TypeString,
										Required: true,
									},
									"content": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
				Set: resourceArmVirtualMachineStorageOsProfileWindowsConfigHash,
			},

			"os_profile_linux_config": {
				Type:     schema.TypeSet,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"disable_password_authentication": {
							Type:     schema.TypeBool,
							Required: true,
						},
						"ssh_keys": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"path": {
										Type:     schema.TypeString,
										Required: true,
									},
									"key_data": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
				Set: resourceArmVirtualMachineStorageOsProfileLinuxConfigHash,
			},

			"os_profile_secrets": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"source_vault_id": {
							Type:     schema.TypeString,
							Required: true,
						},

						"vault_certificates": {
							Type:     schema.TypeList,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"certificate_url": {
										Type:     schema.TypeString,
										Required: true,
									},
									"certificate_store": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},

			"network_interface_ids": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},

			"tags": tagsSchema(),
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
	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

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

	if _, ok := d.GetOk("boot_diagnostics"); ok {
		diagnosticsProfile := expandAzureRmVirtualMachineDiagnosticsProfile(d)
		if diagnosticsProfile != nil {
			properties.DiagnosticsProfile = diagnosticsProfile
		}
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
		Tags:       expandedTags,
	}

	if _, ok := d.GetOk("plan"); ok {
		plan, err := expandAzureRmVirtualMachinePlan(d)
		if err != nil {
			return err
		}

		vm.Plan = plan
	}

	_, vmErr := vmClient.CreateOrUpdate(resGroup, name, vm, make(chan struct{}))
	if vmErr != nil {
		return vmErr
	}

	read, err := vmClient.Get(resGroup, name, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Virtual Machine %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

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

	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Virtual Machine %s: %s", name, err)
	}
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}

	if resp.Plan != nil {
		if err := d.Set("plan", flattenAzureRmVirtualMachinePlan(resp.Plan)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Plan error: %#v", err)
		}
	}

	if resp.Properties.AvailabilitySet != nil {
		d.Set("availability_set_id", strings.ToLower(*resp.Properties.AvailabilitySet.ID))
	}

	d.Set("vm_size", resp.Properties.HardwareProfile.VMSize)

	if resp.Properties.StorageProfile.ImageReference != nil {
		if err := d.Set("storage_image_reference", schema.NewSet(resourceArmVirtualMachineStorageImageReferenceHash, flattenAzureRmVirtualMachineImageReference(resp.Properties.StorageProfile.ImageReference))); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Storage Image Reference error: %#v", err)
		}
	}

	if err := d.Set("storage_os_disk", schema.NewSet(resourceArmVirtualMachineStorageOsDiskHash, flattenAzureRmVirtualMachineOsDisk(resp.Properties.StorageProfile.OsDisk))); err != nil {
		return fmt.Errorf("[DEBUG] Error setting Virtual Machine Storage OS Disk error: %#v", err)
	}

	if resp.Properties.StorageProfile.DataDisks != nil {
		if err := d.Set("storage_data_disk", flattenAzureRmVirtualMachineDataDisk(resp.Properties.StorageProfile.DataDisks)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Storage Data Disks error: %#v", err)
		}
	}

	if err := d.Set("os_profile", schema.NewSet(resourceArmVirtualMachineStorageOsProfileHash, flattenAzureRmVirtualMachineOsProfile(resp.Properties.OsProfile))); err != nil {
		return fmt.Errorf("[DEBUG] Error setting Virtual Machine Storage OS Profile: %#v", err)
	}

	if resp.Properties.OsProfile.WindowsConfiguration != nil {
		if err := d.Set("os_profile_windows_config", flattenAzureRmVirtualMachineOsProfileWindowsConfiguration(resp.Properties.OsProfile.WindowsConfiguration)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Storage OS Profile Windows Configuration: %#v", err)
		}
	}

	if resp.Properties.OsProfile.LinuxConfiguration != nil {
		if err := d.Set("os_profile_linux_config", flattenAzureRmVirtualMachineOsProfileLinuxConfiguration(resp.Properties.OsProfile.LinuxConfiguration)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Storage OS Profile Linux Configuration: %#v", err)
		}
	}

	if resp.Properties.OsProfile.Secrets != nil {
		if err := d.Set("os_profile_secrets", flattenAzureRmVirtualMachineOsProfileSecrets(resp.Properties.OsProfile.Secrets)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Storage OS Profile Secrets: %#v", err)
		}
	}

	if resp.Properties.DiagnosticsProfile != nil && resp.Properties.DiagnosticsProfile.BootDiagnostics != nil {
		if err := d.Set("boot_diagnostics", flattenAzureRmVirtualMachineDiagnosticsProfile(resp.Properties.DiagnosticsProfile.BootDiagnostics)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Diagnostics Profile: %#v", err)
		}
	}

	if resp.Properties.NetworkProfile != nil {
		if err := d.Set("network_interface_ids", flattenAzureRmVirtualMachineNetworkInterfaces(resp.Properties.NetworkProfile)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting Virtual Machine Storage Network Interfaces: %#v", err)
		}
	}

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmVirtualMachineDelete(d *schema.ResourceData, meta interface{}) error {
	vmClient := meta.(*ArmClient).vmClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["virtualMachines"]

	if _, err = vmClient.Delete(resGroup, name, make(chan struct{})); err != nil {
		return err
	}

	// delete OS Disk if opted in
	if deleteOsDisk := d.Get("delete_os_disk_on_termination").(bool); deleteOsDisk {
		log.Printf("[INFO] delete_os_disk_on_termination is enabled, deleting")

		osDisk, err := expandAzureRmVirtualMachineOsDisk(d)
		if err != nil {
			return fmt.Errorf("Error expanding OS Disk: %s", err)
		}

		if err = resourceArmVirtualMachineDeleteVhd(*osDisk.Vhd.URI, resGroup, meta); err != nil {
			return fmt.Errorf("Error deleting OS Disk VHD: %s", err)
		}
	}

	// delete Data disks if opted in
	if deleteDataDisks := d.Get("delete_data_disks_on_termination").(bool); deleteDataDisks {
		log.Printf("[INFO] delete_data_disks_on_termination is enabled, deleting each data disk")

		disks, err := expandAzureRmVirtualMachineDataDisk(d)
		if err != nil {
			return fmt.Errorf("Error expanding Data Disks: %s", err)
		}

		for _, disk := range disks {
			if err = resourceArmVirtualMachineDeleteVhd(*disk.Vhd.URI, resGroup, meta); err != nil {
				return fmt.Errorf("Error deleting Data Disk VHD: %s", err)
			}
		}
	}

	return nil
}

func resourceArmVirtualMachineDeleteVhd(uri, resGroup string, meta interface{}) error {
	vhdURL, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("Cannot parse Disk VHD URI: %s", err)
	}

	// VHD URI is in the form: https://storageAccountName.blob.core.windows.net/containerName/blobName
	storageAccountName := strings.Split(vhdURL.Host, ".")[0]
	path := strings.Split(strings.TrimPrefix(vhdURL.Path, "/"), "/")
	containerName := path[0]
	blobName := path[1]

	blobClient, saExists, err := meta.(*ArmClient).getBlobStorageClientForStorageAccount(resGroup, storageAccountName)
	if err != nil {
		return fmt.Errorf("Error creating blob store client for VHD deletion: %s", err)
	}

	if !saExists {
		log.Printf("[INFO] Storage Account %q doesn't exist so the VHD blob won't exist", storageAccountName)
		return nil
	}

	log.Printf("[INFO] Deleting VHD blob %s", blobName)
	_, err = blobClient.DeleteBlobIfExists(containerName, blobName, nil)
	if err != nil {
		return fmt.Errorf("Error deleting VHD blob: %s", err)
	}

	return nil
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

	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineStorageOsProfileHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["admin_username"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["computer_name"].(string)))
	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineStorageOsDiskHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["vhd_uri"].(string)))

	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineStorageOsProfileLinuxConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%t-", m["disable_password_authentication"].(bool)))

	return hashcode.String(buf.String())
}

func resourceArmVirtualMachineStorageOsProfileWindowsConfigHash(v interface{}) int {
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

func flattenAzureRmVirtualMachinePlan(plan *compute.Plan) map[string]interface{} {
	result := make(map[string]interface{})
	result["name"] = *plan.Name
	result["publisher"] = *plan.Publisher
	result["product"] = *plan.Product

	return result
}

func flattenAzureRmVirtualMachineImageReference(image *compute.ImageReference) []interface{} {
	result := make(map[string]interface{})
	result["offer"] = *image.Offer
	result["publisher"] = *image.Publisher
	result["sku"] = *image.Sku

	if image.Version != nil {
		result["version"] = *image.Version
	}

	return []interface{}{result}
}

func flattenAzureRmVirtualMachineDiagnosticsProfile(profile *compute.BootDiagnostics) []interface{} {
	result := make(map[string]interface{})

	result["enabled"] = *profile.Enabled
	result["storage_uri"] = *profile.StorageURI

	return []interface{}{result}
}

func flattenAzureRmVirtualMachineNetworkInterfaces(profile *compute.NetworkProfile) []string {
	result := make([]string, 0, len(*profile.NetworkInterfaces))
	for _, nic := range *profile.NetworkInterfaces {
		result = append(result, *nic.ID)
	}
	return result
}

func flattenAzureRmVirtualMachineOsProfileSecrets(secrets *[]compute.VaultSecretGroup) []map[string]interface{} {
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

func flattenAzureRmVirtualMachineDataDisk(disks *[]compute.DataDisk) interface{} {
	result := make([]interface{}, len(*disks))
	for i, disk := range *disks {
		l := make(map[string]interface{})
		l["name"] = *disk.Name
		l["vhd_uri"] = *disk.Vhd.URI
		l["create_option"] = disk.CreateOption
		l["disk_size_gb"] = *disk.DiskSizeGB
		l["lun"] = *disk.Lun

		result[i] = l
	}
	return result
}

func flattenAzureRmVirtualMachineOsProfile(osProfile *compute.OSProfile) []interface{} {
	result := make(map[string]interface{})
	result["computer_name"] = *osProfile.ComputerName
	result["admin_username"] = *osProfile.AdminUsername
	if osProfile.CustomData != nil {
		result["custom_data"] = *osProfile.CustomData
	}

	return []interface{}{result}
}

func flattenAzureRmVirtualMachineOsProfileWindowsConfiguration(config *compute.WindowsConfiguration) []interface{} {
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

			if i.Content != nil {
				c["content"] = *i.Content
			}

			content = append(content, c)
		}

		result["additional_unattend_config"] = content
	}

	return []interface{}{result}
}

func flattenAzureRmVirtualMachineOsProfileLinuxConfiguration(config *compute.LinuxConfiguration) []interface{} {

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

func flattenAzureRmVirtualMachineOsDisk(disk *compute.OSDisk) []interface{} {
	result := make(map[string]interface{})
	result["name"] = *disk.Name
	result["vhd_uri"] = *disk.Vhd.URI
	result["create_option"] = disk.CreateOption
	result["caching"] = disk.Caching

	return []interface{}{result}
}

func expandAzureRmVirtualMachinePlan(d *schema.ResourceData) (*compute.Plan, error) {
	planConfigs := d.Get("plan").(*schema.Set).List()

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

	osProfile := osProfiles[0].(map[string]interface{})

	adminUsername := osProfile["admin_username"].(string)
	adminPassword := osProfile["admin_password"].(string)
	computerName := osProfile["computer_name"].(string)

	profile := &compute.OSProfile{
		AdminUsername: &adminUsername,
		ComputerName:  &computerName,
	}

	if adminPassword != "" {
		profile.AdminPassword = &adminPassword
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

func expandAzureRmVirtualMachineOsProfileLinuxConfig(d *schema.ResourceData) (*compute.LinuxConfiguration, error) {
	osProfilesLinuxConfig := d.Get("os_profile_linux_config").(*schema.Set).List()

	linuxConfig := osProfilesLinuxConfig[0].(map[string]interface{})
	disablePasswordAuth := linuxConfig["disable_password_authentication"].(bool)

	config := &compute.LinuxConfiguration{
		DisablePasswordAuthentication: &disablePasswordAuth,
	}

	linuxKeys := linuxConfig["ssh_keys"].([]interface{})
	sshPublicKeys := []compute.SSHPublicKey{}
	for _, key := range linuxKeys {

		sshKey, ok := key.(map[string]interface{})
		if !ok {
			continue
		}
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
		winRm := v.([]interface{})
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
		additionalConfig := v.([]interface{})
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

func expandAzureRmVirtualMachineDataDisk(d *schema.ResourceData) ([]compute.DataDisk, error) {
	disks := d.Get("storage_data_disk").([]interface{})
	data_disks := make([]compute.DataDisk, 0, len(disks))
	for _, disk_config := range disks {
		config := disk_config.(map[string]interface{})

		name := config["name"].(string)
		vhd := config["vhd_uri"].(string)
		createOption := config["create_option"].(string)
		lun := int32(config["lun"].(int))
		disk_size := int32(config["disk_size_gb"].(int))

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

func expandAzureRmVirtualMachineDiagnosticsProfile(d *schema.ResourceData) *compute.DiagnosticsProfile {
	bootDiagnostics := d.Get("boot_diagnostics").([]interface{})

	diagnosticsProfile := &compute.DiagnosticsProfile{}
	if len(bootDiagnostics) > 0 {
		bootDiagnostic := bootDiagnostics[0].(map[string]interface{})

		diagnostic := &compute.BootDiagnostics{
			Enabled:    riviera.Bool(bootDiagnostic["enabled"].(bool)),
			StorageURI: riviera.String(bootDiagnostic["storage_uri"].(string)),
		}

		diagnosticsProfile.BootDiagnostics = diagnostic

		return diagnosticsProfile
	}

	return nil
}

func expandAzureRmVirtualMachineImageReference(d *schema.ResourceData) (*compute.ImageReference, error) {
	storageImageRefs := d.Get("storage_image_reference").(*schema.Set).List()

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

	disk := disks[0].(map[string]interface{})

	name := disk["name"].(string)
	vhdURI := disk["vhd_uri"].(string)
	imageURI := disk["image_uri"].(string)
	createOption := disk["create_option"].(string)

	osDisk := &compute.OSDisk{
		Name: &name,
		Vhd: &compute.VirtualHardDisk{
			URI: &vhdURI,
		},
		CreateOption: compute.DiskCreateOptionTypes(createOption),
	}

	if v := disk["image_uri"].(string); v != "" {
		osDisk.Image = &compute.VirtualHardDisk{
			URI: &imageURI,
		}
	}

	if v := disk["os_type"].(string); v != "" {
		if v == "linux" {
			osDisk.OsType = compute.Linux
		} else if v == "windows" {
			osDisk.OsType = compute.Windows
		} else {
			return nil, fmt.Errorf("[ERROR] os_type must be 'linux' or 'windows'")
		}
	}

	if v := disk["caching"].(string); v != "" {
		osDisk.Caching = compute.CachingTypes(v)
	}

	return osDisk, nil
}
