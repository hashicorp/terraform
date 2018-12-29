package profitbricks

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"strings"
)

func resourceProfitBricksServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksServerCreate,
		Read:   resourceProfitBricksServerRead,
		Update: resourceProfitBricksServerUpdate,
		Delete: resourceProfitBricksServerDelete,
		Schema: map[string]*schema.Schema{

			//Server parameters
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cores": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"ram": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"licence_type": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"boot_volume": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"boot_cdrom": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cpu_family": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"boot_image": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"primary_nic": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"primary_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"volume": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"image_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"disk_type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"image_password": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"licence_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ssh_key_path": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Optional: true,
						},
						"bus": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"availability_zone": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"nic": {
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"lan": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"name": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"dhcp": {
							Type:     schema.TypeBool,
							Optional: true,
						},

						"ip": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ips": {
							Type:     schema.TypeList,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Computed: true,
						},
						"nat": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"firewall_active": {
							Type:     schema.TypeBool,
							Optional: true,
						},
						"firewall": {
							Type:     schema.TypeSet,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"name": {
										Type:     schema.TypeString,
										Optional: true,
									},

									"protocol": {
										Type:     schema.TypeString,
										Required: true,
									},
									"source_mac": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"source_ip": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"target_ip": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"ip": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"ips": {
										Type:     schema.TypeList,
										Elem:     &schema.Schema{Type: schema.TypeString},
										Optional: true,
									},
									"port_range_start": {
										Type:     schema.TypeInt,
										Optional: true,
										ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
											if v.(int) < 1 && v.(int) > 65534 {
												errors = append(errors, fmt.Errorf("Port start range must be between 1 and 65534"))
											}
											return
										},
									},

									"port_range_end": {
										Type:     schema.TypeInt,
										Optional: true,
										ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
											if v.(int) < 1 && v.(int) > 65534 {
												errors = append(errors, fmt.Errorf("Port end range must be between 1 and 65534"))
											}
											return
										},
									},
									"icmp_type": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"icmp_code": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceProfitBricksServerCreate(d *schema.ResourceData, meta interface{}) error {
	request := profitbricks.Server{
		Properties: profitbricks.ServerProperties{
			Name:  d.Get("name").(string),
			Cores: d.Get("cores").(int),
			Ram:   d.Get("ram").(int),
		},
	}

	if v, ok := d.GetOk("availability_zone"); ok {
		request.Properties.AvailabilityZone = v.(string)
	}

	if v, ok := d.GetOk("cpu_family"); ok {
		if v.(string) != "" {
			request.Properties.CpuFamily = v.(string)
		}
	}
	if vRaw, ok := d.GetOk("volume"); ok {

		volumeRaw := vRaw.(*schema.Set).List()

		for _, raw := range volumeRaw {
			rawMap := raw.(map[string]interface{})
			var imagePassword string
			//Can be one file or a list of files
			var sshkey_path []interface{}
			var image, licenceType, availabilityZone string

			if !IsValidUUID(rawMap["image_name"].(string)) {
				if rawMap["image_name"] != nil {
					image = getImageId(d.Get("datacenter_id").(string), rawMap["image_name"].(string), rawMap["disk_type"].(string))
					if image == "" {
						dc := profitbricks.GetDatacenter(d.Get("datacenter_id").(string))
						return fmt.Errorf("Image '%s' doesn't exist. in location %s", rawMap["image_name"], dc.Properties.Location)

					}
				}
			} else {
				image = rawMap["image_name"].(string)
			}

			if rawMap["licence_type"] != nil {
				licenceType = rawMap["licence_type"].(string)
			}

			if rawMap["image_password"] != nil {
				imagePassword = rawMap["image_password"].(string)
			}
			if rawMap["ssh_key_path"] != nil {
				sshkey_path = rawMap["ssh_key_path"].([]interface{})
			}
			if rawMap["image_name"] != nil {
				if imagePassword == "" && len(sshkey_path) == 0 {
					return fmt.Errorf("Either 'image_password' or 'ssh_key_path' must be provided.")
				}
			}
			var publicKeys []string
			if len(sshkey_path) != 0 {
				for _, path := range sshkey_path {
					log.Printf("[DEBUG] Reading file %s", path)
					publicKey, err := readPublicKey(path.(string))
					if err != nil {
						return fmt.Errorf("Error fetching sshkey from file (%s) %s", path, err.Error())
					}
					publicKeys = append(publicKeys, publicKey)
				}
			}
			if rawMap["availability_zone"] != nil {
				availabilityZone = rawMap["availability_zone"].(string)
			}
			if image == "" && licenceType == "" {
				return fmt.Errorf("Either 'image', or 'licenceType' must be set.")
			}

			request.Entities = &profitbricks.ServerEntities{
				Volumes: &profitbricks.Volumes{
					Items: []profitbricks.Volume{
						{
							Properties: profitbricks.VolumeProperties{
								Name:             rawMap["name"].(string),
								Size:             rawMap["size"].(int),
								Type:             rawMap["disk_type"].(string),
								ImagePassword:    imagePassword,
								Image:            image,
								Bus:              rawMap["bus"].(string),
								LicenceType:      licenceType,
								AvailabilityZone: availabilityZone,
							},
						},
					},
				},
			}

			if len(publicKeys) == 0 {
				request.Entities.Volumes.Items[0].Properties.SshKeys = nil
			} else {
				request.Entities.Volumes.Items[0].Properties.SshKeys = publicKeys
			}
		}

	}

	if nRaw, ok := d.GetOk("nic"); ok {
		nicRaw := nRaw.(*schema.Set).List()

		for _, raw := range nicRaw {
			rawMap := raw.(map[string]interface{})
			nic := profitbricks.Nic{Properties: profitbricks.NicProperties{}}
			if rawMap["lan"] != nil {
				nic.Properties.Lan = rawMap["lan"].(int)
			}
			if rawMap["name"] != nil {
				nic.Properties.Name = rawMap["name"].(string)
			}
			if rawMap["dhcp"] != nil {
				nic.Properties.Dhcp = rawMap["dhcp"].(bool)
			}
			if rawMap["firewall_active"] != nil {
				nic.Properties.FirewallActive = rawMap["firewall_active"].(bool)
			}
			if rawMap["ip"] != nil {
				rawIps := rawMap["ip"].(string)
				ips := strings.Split(rawIps, ",")
				if rawIps != "" {
					nic.Properties.Ips = ips
				}
			}
			if rawMap["nat"] != nil {
				nic.Properties.Nat = rawMap["nat"].(bool)
			}
			request.Entities.Nics = &profitbricks.Nics{
				Items: []profitbricks.Nic{
					nic,
				},
			}

			if rawMap["firewall"] != nil {
				rawFw := rawMap["firewall"].(*schema.Set).List()
				for _, rraw := range rawFw {
					fwRaw := rraw.(map[string]interface{})
					log.Println("[DEBUG] fwRaw", fwRaw["protocol"])

					firewall := profitbricks.FirewallRule{
						Properties: profitbricks.FirewallruleProperties{
							Protocol: fwRaw["protocol"].(string),
						},
					}

					if fwRaw["name"] != nil {
						firewall.Properties.Name = fwRaw["name"].(string)
					}
					if fwRaw["source_mac"] != nil {
						firewall.Properties.SourceMac = fwRaw["source_mac"].(string)
					}
					if fwRaw["source_ip"] != nil {
						firewall.Properties.SourceIp = fwRaw["source_ip"].(string)
					}
					if fwRaw["target_ip"] != nil {
						firewall.Properties.TargetIp = fwRaw["target_ip"].(string)
					}
					if fwRaw["port_range_start"] != nil {
						firewall.Properties.PortRangeStart = fwRaw["port_range_start"].(int)
					}
					if fwRaw["port_range_end"] != nil {
						firewall.Properties.PortRangeEnd = fwRaw["port_range_end"].(int)
					}
					if fwRaw["icmp_type"] != nil {
						firewall.Properties.IcmpType = fwRaw["icmp_type"].(string)
					}
					if fwRaw["icmp_code"] != nil {
						firewall.Properties.IcmpCode = fwRaw["icmp_code"].(string)
					}

					request.Entities.Nics.Items[0].Entities = &profitbricks.NicEntities{
						Firewallrules: &profitbricks.FirewallRules{
							Items: []profitbricks.FirewallRule{
								firewall,
							},
						},
					}
				}

			}
		}
	}

	if len(request.Entities.Nics.Items[0].Properties.Ips) == 0 {
		request.Entities.Nics.Items[0].Properties.Ips = nil
	}
	server := profitbricks.CreateServer(d.Get("datacenter_id").(string), request)

	jsn, _ := json.Marshal(request)
	log.Println("[DEBUG] Server request", string(jsn))
	log.Println("[DEBUG] Server response", server.Response)

	if server.StatusCode > 299 {
		return fmt.Errorf(
			"Error creating server: (%s)", server.Response)
	}

	err := waitTillProvisioned(meta, server.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId(server.Id)
	server = profitbricks.GetServer(d.Get("datacenter_id").(string), server.Id)

	d.Set("primary_nic", server.Entities.Nics.Items[0].Id)
	if len(server.Entities.Nics.Items[0].Properties.Ips) > 0 {
		d.SetConnInfo(map[string]string{
			"type":     "ssh",
			"host":     server.Entities.Nics.Items[0].Properties.Ips[0],
			"password": request.Entities.Volumes.Items[0].Properties.ImagePassword,
		})
	}
	return resourceProfitBricksServerRead(d, meta)
}

func resourceProfitBricksServerRead(d *schema.ResourceData, meta interface{}) error {
	dcId := d.Get("datacenter_id").(string)
	serverId := d.Id()

	server := profitbricks.GetServer(dcId, serverId)
	if server.StatusCode > 299 {
		if server.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error occured while fetching a server ID %s %s", d.Id(), server.Response)
	}
	d.Set("name", server.Properties.Name)
	d.Set("cores", server.Properties.Cores)
	d.Set("ram", server.Properties.Ram)
	d.Set("availability_zone", server.Properties.AvailabilityZone)

	if primarynic, ok := d.GetOk("primary_nic"); ok {
		d.Set("primary_nic", primarynic.(string))

		nic := profitbricks.GetNic(dcId, serverId, primarynic.(string))

		if len(nic.Properties.Ips) > 0 {
			d.Set("primary_ip", nic.Properties.Ips[0])
		}

		if nRaw, ok := d.GetOk("nic"); ok {
			log.Printf("[DEBUG] parsing nic")

			nicRaw := nRaw.(*schema.Set).List()

			for _, raw := range nicRaw {

				rawMap := raw.(map[string]interface{})

				rawMap["lan"] = nic.Properties.Lan
				rawMap["name"] = nic.Properties.Name
				rawMap["dhcp"] = nic.Properties.Dhcp
				rawMap["nat"] = nic.Properties.Nat
				rawMap["firewall_active"] = nic.Properties.FirewallActive
				rawMap["ips"] = nic.Properties.Ips
			}
			d.Set("nic", nicRaw)
		}
	}

	if server.Properties.BootVolume != nil {
		d.Set("boot_volume", server.Properties.BootVolume.Id)
	}
	if server.Properties.BootCdrom != nil {
		d.Set("boot_cdrom", server.Properties.BootCdrom.Id)
	}
	return nil
}

func resourceProfitBricksServerUpdate(d *schema.ResourceData, meta interface{}) error {
	dcId := d.Get("datacenter_id").(string)

	request := profitbricks.ServerProperties{}

	if d.HasChange("name") {
		_, n := d.GetChange("name")
		request.Name = n.(string)
	}
	if d.HasChange("cores") {
		_, n := d.GetChange("cores")
		request.Cores = n.(int)
	}
	if d.HasChange("ram") {
		_, n := d.GetChange("ram")
		request.Ram = n.(int)
	}
	if d.HasChange("availability_zone") {
		_, n := d.GetChange("availability_zone")
		request.AvailabilityZone = n.(string)
	}
	if d.HasChange("cpu_family") {
		_, n := d.GetChange("cpu_family")
		request.CpuFamily = n.(string)
	}
	server := profitbricks.PatchServer(dcId, d.Id(), request)

	//Volume stuff
	if d.HasChange("volume") {
		volume := server.Entities.Volumes.Items[0]
		_, new := d.GetChange("volume")

		newVolume := new.(*schema.Set).List()
		properties := profitbricks.VolumeProperties{}

		for _, raw := range newVolume {
			rawMap := raw.(map[string]interface{})
			if rawMap["name"] != nil {
				properties.Name = rawMap["name"].(string)
			}
			if rawMap["size"] != nil {
				properties.Size = rawMap["size"].(int)
			}
			if rawMap["bus"] != nil {
				properties.Bus = rawMap["bus"].(string)
			}
		}

		volume = profitbricks.PatchVolume(d.Get("datacenter_id").(string), server.Entities.Volumes.Items[0].Id, properties)

		if volume.StatusCode > 299 {
			return fmt.Errorf("Error patching volume (%s) (%s)", d.Id(), volume.Response)
		}

		err := waitTillProvisioned(meta, volume.Headers.Get("Location"))
		if err != nil {
			return err
		}
	}

	//Nic stuff
	if d.HasChange("nic") {
		nic := profitbricks.Nic{}
		for _, n := range server.Entities.Nics.Items {
			if n.Id == d.Get("primary_nic").(string) {
				nic = n
				break
			}
		}
		_, new := d.GetChange("nic")

		newNic := new.(*schema.Set).List()
		properties := profitbricks.NicProperties{}

		for _, raw := range newNic {
			rawMap := raw.(map[string]interface{})
			if rawMap["name"] != nil {
				properties.Name = rawMap["name"].(string)
			}
			if rawMap["ip"] != nil {
				rawIps := rawMap["ip"].(string)
				ips := strings.Split(rawIps, ",")

				if rawIps != "" {
					nic.Properties.Ips = ips
				}
			}
			if rawMap["lan"] != nil {
				properties.Lan = rawMap["lan"].(int)
			}
			if rawMap["dhcp"] != nil {
				properties.Dhcp = rawMap["dhcp"].(bool)
			}
			if rawMap["nat"] != nil {
				properties.Nat = rawMap["nat"].(bool)
			}
		}

		nic = profitbricks.PatchNic(d.Get("datacenter_id").(string), server.Id, server.Entities.Nics.Items[0].Id, properties)

		if nic.StatusCode > 299 {
			return fmt.Errorf(
				"Error patching nic (%s)", nic.Response)
		}

		err := waitTillProvisioned(meta, nic.Headers.Get("Location"))
		if err != nil {
			return err
		}
	}

	if server.StatusCode > 299 {
		return fmt.Errorf(
			"Error patching server (%s) (%s)", d.Id(), server.Response)
	}
	return resourceProfitBricksServerRead(d, meta)
}

func resourceProfitBricksServerDelete(d *schema.ResourceData, meta interface{}) error {
	dcId := d.Get("datacenter_id").(string)

	server := profitbricks.GetServer(dcId, d.Id())

	if server.Properties.BootVolume != nil {
		resp := profitbricks.DeleteVolume(dcId, server.Properties.BootVolume.Id)
		err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
		if err != nil {
			return err
		}
	}

	resp := profitbricks.DeleteServer(dcId, d.Id())
	if resp.StatusCode > 299 {
		return fmt.Errorf("An error occured while deleting a server ID %s %s", d.Id(), string(resp.Body))

	}
	err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

//Reads public key from file and returns key string iff valid
func readPublicKey(path string) (key string, err error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(bytes)
	if err != nil {
		return "", err
	}
	return string(ssh.MarshalAuthorizedKey(pubKey)[:]), nil
}
