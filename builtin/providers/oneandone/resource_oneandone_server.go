package oneandone

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/schema"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"strings"

	"errors"
)

func resourceOneandOneServer() *schema.Resource {
	return &schema.Resource{
		Create: resourceOneandOneServerCreate,
		Read:   resourceOneandOneServerRead,
		Update: resourceOneandOneServerUpdate,
		Delete: resourceOneandOneServerDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"image": {
				Type:     schema.TypeString,
				Required: true,
			},
			"vcores": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"cores_per_processor": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"ram": {
				Type:     schema.TypeFloat,
				Required: true,
			},
			"ssh_key_path": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"password": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			"datacenter": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ip": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ips": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ip": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"firewall_policy_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Computed: true,
			},
			"hdds": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"disk_size": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"is_main": {
							Type:     schema.TypeBool,
							Optional: true,
						},
					},
				},
				Required: true,
			},
			"firewall_policy_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"monitoring_policy_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"loadbalancer_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceOneandOneServerCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	saps, _ := config.API.ListServerAppliances()

	var sa oneandone.ServerAppliance
	for _, a := range saps {

		if a.Type == "IMAGE" && strings.Contains(strings.ToLower(a.Name), strings.ToLower(d.Get("image").(string))) {
			sa = a
			break
		}
	}

	var hdds []oneandone.Hdd
	if raw, ok := d.GetOk("hdds"); ok {
		rawhdds := raw.([]interface{})

		var istheremain bool
		for _, raw := range rawhdds {
			hd := raw.(map[string]interface{})
			hdd := oneandone.Hdd{
				Size:   hd["disk_size"].(int),
				IsMain: hd["is_main"].(bool),
			}

			if hdd.IsMain {
				if hdd.Size < sa.MinHddSize {
					return fmt.Errorf(fmt.Sprintf("Minimum required disk size %d", sa.MinHddSize))
				}
				istheremain = true
			}

			hdds = append(hdds, hdd)
		}

		if !istheremain {
			return fmt.Errorf("At least one HDD has to be %s", "`is_main`")
		}
	}

	req := oneandone.ServerRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		ApplianceId: sa.Id,
		PowerOn:     true,
		Hardware: oneandone.Hardware{
			Vcores:            d.Get("vcores").(int),
			CoresPerProcessor: d.Get("cores_per_processor").(int),
			Ram:               float32(d.Get("ram").(float64)),
			Hdds:              hdds,
		},
	}

	if raw, ok := d.GetOk("ip"); ok {

		new_ip := raw.(string)

		ips, err := config.API.ListPublicIps()
		if err != nil {
			return err
		}

		for _, ip := range ips {
			if ip.IpAddress == new_ip {
				req.IpId = ip.Id
				break
			}
		}

		log.Println("[DEBUG] req.IP", req.IpId)
	}

	if raw, ok := d.GetOk("datacenter"); ok {

		dcs, err := config.API.ListDatacenters()

		if err != nil {
			return fmt.Errorf("An error occured while fetching list of datacenters %s", err)

		}

		decenter := raw.(string)
		for _, dc := range dcs {
			if strings.ToLower(dc.CountryCode) == strings.ToLower(decenter) {
				req.DatacenterId = dc.Id
				break
			}
		}
	}

	if fwp_id, ok := d.GetOk("firewall_policy_id"); ok {
		req.FirewallPolicyId = fwp_id.(string)
	}

	if mp_id, ok := d.GetOk("monitoring_policy_id"); ok {
		req.MonitoringPolicyId = mp_id.(string)
	}

	if mp_id, ok := d.GetOk("loadbalancer_id"); ok {
		req.LoadBalancerId = mp_id.(string)
	}

	var privateKey string
	if raw, ok := d.GetOk("ssh_key_path"); ok {
		rawpath := raw.(string)

		priv, publicKey, err := getSshKey(rawpath)
		privateKey = priv
		if err != nil {
			return err
		}

		req.SSHKey = publicKey
	}

	var password string
	if raw, ok := d.GetOk("password"); ok {
		req.Password = raw.(string)
		password = req.Password
	}

	server_id, server, err := config.API.CreateServer(&req)
	if err != nil {
		return err
	}

	err = config.API.WaitForState(server, "POWERED_ON", 10, config.Retries)

	d.SetId(server_id)
	server, err = config.API.GetServer(d.Id())
	if err != nil {
		return err
	}

	if password == "" {
		password = server.FirstPassword
	}
	d.SetConnInfo(map[string]string{
		"type":        "ssh",
		"host":        server.Ips[0].Ip,
		"password":    password,
		"private_key": privateKey,
	})

	return resourceOneandOneServerRead(d, meta)
}

func resourceOneandOneServerRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	server, err := config.API.GetServer(d.Id())

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", server.Name)
	d.Set("datacenter", server.Datacenter.CountryCode)

	d.Set("hdds", readHdds(server.Hardware))

	d.Set("ips", readIps(server.Ips))

	if len(server.FirstPassword) > 0 {
		d.Set("password", server.FirstPassword)
	}

	return nil
}

func resourceOneandOneServerUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if d.HasChange("name") || d.HasChange("description") {
		_, name := d.GetChange("name")
		_, description := d.GetChange("description")
		server, err := config.API.RenameServer(d.Id(), name.(string), description.(string))
		if err != nil {
			return err
		}

		err = config.API.WaitForState(server, "POWERED_ON", 10, config.Retries)

	}

	if d.HasChange("hdds") {
		oldV, newV := d.GetChange("hdds")
		newValues := newV.([]interface{})
		oldValues := oldV.([]interface{})

		if len(oldValues) > len(newValues) {
			diff := difference(oldValues, newValues)
			for _, old := range diff {
				o := old.(map[string]interface{})
				old_id := o["id"].(string)
				server, err := config.API.DeleteServerHdd(d.Id(), old_id)
				if err != nil {
					return err
				}

				err = config.API.WaitForState(server, "POWERED_ON", 10, config.Retries)
				if err != nil {
					return err
				}
			}
		} else {
			for _, newHdd := range newValues {
				n := newHdd.(map[string]interface{})
				//old := oldHdd.(map[string]interface{})

				if n["id"].(string) == "" {
					hdds := oneandone.ServerHdds{
						Hdds: []oneandone.Hdd{
							{
								Size:   n["disk_size"].(int),
								IsMain: n["is_main"].(bool),
							},
						},
					}

					server, err := config.API.AddServerHdds(d.Id(), &hdds)

					if err != nil {
						return err
					}
					err = config.API.WaitForState(server, "POWERED_ON", 10, config.Retries)
					if err != nil {
						return err
					}
				} else {
					id := n["id"].(string)
					isMain := n["is_main"].(bool)

					if id != "" && !isMain {
						log.Println("[DEBUG] Resizing existing HDD")
						config.API.ResizeServerHdd(d.Id(), id, n["disk_size"].(int))
					}
				}

			}
		}
	}

	if d.HasChange("monitoring_policy_id") {
		o, n := d.GetChange("monitoring_policy_id")

		if n == nil {
			mp, err := config.API.RemoveMonitoringPolicyServer(o.(string), d.Id())

			if err != nil {
				return err
			}

			err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
			if err != nil {
				return err
			}
		} else {
			mp, err := config.API.AttachMonitoringPolicyServers(n.(string), []string{d.Id()})
			if err != nil {
				return err
			}

			err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
			if err != nil {
				return err
			}
		}
	}

	if d.HasChange("loadbalancer_id") {
		o, n := d.GetChange("loadbalancer_id")
		server, err := config.API.GetServer(d.Id())
		if err != nil {
			return err
		}

		if n == nil || n.(string) == "" {
			log.Println("[DEBUG] Removing")
			log.Println("[DEBUG] IPS:", server.Ips)

			for _, ip := range server.Ips {
				mp, err := config.API.DeleteLoadBalancerServerIp(o.(string), ip.Id)

				if err != nil {
					return err
				}

				err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
				if err != nil {
					return err
				}
			}
		} else {
			log.Println("[DEBUG] Adding")
			ip_ids := []string{}
			for _, ip := range server.Ips {
				ip_ids = append(ip_ids, ip.Id)
			}
			mp, err := config.API.AddLoadBalancerServerIps(n.(string), ip_ids)
			if err != nil {
				return err
			}

			err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
			if err != nil {
				return err
			}

		}
	}

	if d.HasChange("firewall_policy_id") {
		server, err := config.API.GetServer(d.Id())
		if err != nil {
			return err
		}

		o, n := d.GetChange("firewall_policy_id")
		if n == nil {
			for _, ip := range server.Ips {
				mp, err := config.API.DeleteFirewallPolicyServerIp(o.(string), ip.Id)
				if err != nil {
					return err
				}

				err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
				if err != nil {
					return err
				}
			}
		} else {
			ip_ids := []string{}
			for _, ip := range server.Ips {
				ip_ids = append(ip_ids, ip.Id)
			}

			mp, err := config.API.AddFirewallPolicyServerIps(n.(string), ip_ids)
			if err != nil {
				return err
			}

			err = config.API.WaitForState(mp, "ACTIVE", 30, config.Retries)
			if err != nil {
				return err
			}
		}
	}

	return resourceOneandOneServerRead(d, meta)
}

func resourceOneandOneServerDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	_, ok := d.GetOk("ip")

	server, err := config.API.DeleteServer(d.Id(), ok)
	if err != nil {
		return err
	}

	err = config.API.WaitUntilDeleted(server)

	if err != nil {
		log.Println("[DEBUG] ************ ERROR While waiting ************")
		return err
	}
	return nil
}

func readHdds(hardware *oneandone.Hardware) []map[string]interface{} {
	hdds := make([]map[string]interface{}, 0, len(hardware.Hdds))

	for _, hd := range hardware.Hdds {
		hdds = append(hdds, map[string]interface{}{
			"id":        hd.Id,
			"disk_size": hd.Size,
			"is_main":   hd.IsMain,
		})
	}

	return hdds
}

func readIps(ips []oneandone.ServerIp) []map[string]interface{} {
	raw := make([]map[string]interface{}, 0, len(ips))
	for _, ip := range ips {

		toadd := map[string]interface{}{
			"ip": ip.Ip,
			"id": ip.Id,
		}

		if ip.Firewall != nil {
			toadd["firewall_policy_id"] = ip.Firewall.Id
		}
		raw = append(raw, toadd)
	}

	return raw
}

func getSshKey(path string) (privatekey string, publickey string, err error) {
	pemBytes, err := ioutil.ReadFile(path)

	if err != nil {
		return "", "", err
	}

	block, _ := pem.Decode(pemBytes)

	if block == nil {
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
