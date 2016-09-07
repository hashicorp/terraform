package profitbricks

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"strconv"
	"errors"
	"strings"
)

func resourceProfitBricksServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksServerCreate,
		Read:   resourceProfitBricksServerRead,
		Update: resourceProfitBricksServerUpdate,
		Delete: resourceProfitBricksServerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
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
							Type:     schema.TypeString,
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
	username, password, _ := getCredentials(meta)

	profitbricks.SetAuth(username, password)
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
			var imagePassword, sshkey_path string
			var image, licenceType string

			if rawMap["image_name"] != nil {
				image = getImageId(d.Get("datacenter_id").(string), rawMap["image_name"].(string), rawMap["disk_type"].(string))
				if image == "" {
					dc := profitbricks.GetDatacenter(d.Get("datacenter_id").(string))
					return fmt.Errorf("Image '%s' doesn't exist. in location %s", rawMap["image_name"], dc.Properties.Location)

				}
			}
			if rawMap["licence_type"] != nil {
				licenceType = rawMap["licence_type"].(string)
			}

			if rawMap["image_password"] != nil {
				imagePassword = rawMap["image_password"].(string)
			}
			if rawMap["ssh_key_path"] != nil {
				sshkey_path = rawMap["ssh_key_path"].(string)
			}
			if rawMap["image_name"] != nil {
				if imagePassword == "" && sshkey_path == "" {
					return fmt.Errorf("'image_password' and 'ssh_key_path' are not provided.")
				}
			}
			var publicKey string
			var err error
			if sshkey_path != "" {
				log.Println("[DEBUG] GETTING THE KEY")
				_, publicKey, err = getSshKey(d, sshkey_path)
				if err != nil {
					return fmt.Errorf("Error fetching sshkeys (%s)", err)
				}
				d.Set("sshkey", publicKey)
			}

			if image == "" && licenceType == "" {
				return fmt.Errorf("Either 'image', or 'licenceType' must be set.")
			}

			request.Entities = &profitbricks.ServerEntities{
				Volumes: &profitbricks.Volumes{
					Items: []profitbricks.Volume{
						profitbricks.Volume{
							Properties: profitbricks.VolumeProperties{
								Name:          rawMap["name"].(string),
								Size:          rawMap["size"].(int),
								Type:          rawMap["disk_type"].(string),
								ImagePassword: imagePassword,
								Image:         image,
								Bus:           rawMap["bus"].(string),
								LicenceType:   licenceType,
							},
						},
					},
				},
			}

			log.Printf("[DEBUG] PUBLIC KEY %s", publicKey)

			if publicKey == "" {
				request.Entities.Volumes.Items[0].Properties.SshKeys = nil
			} else {
				request.Entities.Volumes.Items[0].Properties.SshKeys = []string{publicKey}
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
						&profitbricks.FirewallRules{
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
	d.Set("primary_nic", server.Entities.Nics.Items[0])
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
	username, password, _ := getCredentials(meta)

	profitbricks.SetAuth(username, password)
	dcId := d.Get("datacenter_id").(string)

	server := profitbricks.GetServer(dcId, d.Id())

	primarynic := ""

	if server.Entities != nil && server.Entities.Nics != nil && len(server.Entities.Nics.Items) > 0 {
		for _, n := range server.Entities.Nics.Items {
			if n.Properties.Lan != 0 {
				lan := profitbricks.GetLan(dcId, strconv.Itoa(n.Properties.Lan))
				if lan.StatusCode > 299 {
					return fmt.Errorf("Error while fetching a lan %s", lan.Response)
				}
				if lan.Properties.Public.(interface{}) == true {
					primarynic = n.Id
					break
				}
			}
		}
	}

	d.Set("name", server.Properties.Name)
	d.Set("cores", server.Properties.Cores)
	d.Set("ram", server.Properties.Ram)
	d.Set("availability_zone", server.Properties.AvailabilityZone)
	d.Set("primary_nic", primarynic)

	if server.Properties.BootVolume != nil {
		d.Set("boot_volume", server.Properties.BootVolume.Id)
	}
	if server.Properties.BootCdrom != nil {
		d.Set("boot_cdrom", server.Properties.BootCdrom.Id)
	}
	return nil
}

func resourceProfitBricksServerUpdate(d *schema.ResourceData, meta interface{}) error {
	username, password, _ := getCredentials(meta)

	profitbricks.SetAuth(username, password)
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
	log.Println("[INFO] hlab hlab", request)

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
		log.Println("[INFO] blah blah", properties)

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
		}

		nic = profitbricks.PatchNic(d.Get("datacenter_id").(string), server.Id, server.Entities.Nics.Items[0].Id, properties)
		log.Println("[INFO] blah blah", properties)

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
	username, password, _ := getCredentials(meta)

	profitbricks.SetAuth(username, password)
	dcId := d.Get("datacenter_id").(string)

	server := profitbricks.GetServer(dcId, d.Id())

	if (server.Properties.BootVolume != nil) {
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

func getSshKey(d *schema.ResourceData, path string) (privatekey string, publickey string, err error) {
	pemBytes, err := ioutil.ReadFile(path)

	if err != nil {
		return "", "", err
	}

	block, _ := pem.Decode(pemBytes)

	if (block == nil) {
		return "", "", errors.New("File " + path + " contains nothing")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)

	if err != nil {
		return "", "", err
	}

	priv_blk := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(priv),
	}

	pub, err := ssh.NewPublicKey(&priv.PublicKey)
	if err != nil {
		return "", "", err
	}
	publickey = string(ssh.MarshalAuthorizedKey(pub))
	privatekey = string(pem.EncodeToMemory(&priv_blk))

	return privatekey, publickey, nil
}
