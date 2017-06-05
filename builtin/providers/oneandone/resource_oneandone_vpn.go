package oneandone

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/schema"
	"io"
	"os"
	fp "path/filepath"
	"strings"
)

func resourceOneandOneVPN() *schema.Resource {
	return &schema.Resource{
		Create: resourceOneandOneVPNCreate,
		Read:   resourceOneandOneVPNRead,
		Update: resourceOneandOneVPNUpdate,
		Delete: resourceOneandOneVPNDelete,
		Schema: map[string]*schema.Schema{

			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"download_path": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"datacenter": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"file_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOneandOneVPNCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	var datacenter string

	if raw, ok := d.GetOk("datacenter"); ok {
		dcs, err := config.API.ListDatacenters()
		if err != nil {
			return fmt.Errorf("An error occured while fetching list of datacenters %s", err)
		}

		decenter := raw.(string)
		for _, dc := range dcs {
			if strings.ToLower(dc.CountryCode) == strings.ToLower(decenter) {
				datacenter = dc.Id
				break
			}
		}
	}

	var description string
	if raw, ok := d.GetOk("description"); ok {
		description = raw.(string)
	}

	vpn_id, vpn, err := config.API.CreateVPN(d.Get("name").(string), description, datacenter)
	if err != nil {
		return err
	}

	err = config.API.WaitForState(vpn, "ACTIVE", 10, config.Retries)
	if err != nil {
		return err
	}

	d.SetId(vpn_id)

	return resourceOneandOneVPNRead(d, meta)
}

func resourceOneandOneVPNUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if d.HasChange("name") || d.HasChange("description") {

		vpn, err := config.API.ModifyVPN(d.Id(), d.Get("name").(string), d.Get("description").(string))
		if err != nil {
			return err
		}

		err = config.API.WaitForState(vpn, "ACTIVE", 10, config.Retries)
		if err != nil {
			return err
		}
	}

	return resourceOneandOneVPNRead(d, meta)
}

func resourceOneandOneVPNRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	vpn, err := config.API.GetVPN(d.Id())

	base64_str, err := config.API.GetVPNConfigFile(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return err
	}

	var download_path string
	if raw, ok := d.GetOk("download_path"); ok {
		download_path = raw.(string)
	}

	path, fileName, err := writeCofnig(vpn, download_path, base64_str)
	if err != nil {
		return err
	}

	d.Set("name", vpn.Name)
	d.Set("description", vpn.Description)
	d.Set("download_path", path)
	d.Set("file_name", fileName)
	d.Set("datacenter", vpn.Datacenter.CountryCode)

	return nil
}

func writeCofnig(vpn *oneandone.VPN, path, base64config string) (string, string, error) {
	data, err := base64.StdEncoding.DecodeString(base64config)
	if err != nil {
		return "", "", err
	}

	var fileName string
	if vpn.CloudPanelId != "" {
		fileName = vpn.CloudPanelId + ".zip"
	} else {
		fileName = "vpn_" + fmt.Sprintf("%x", md5.Sum(data)) + ".zip"
	}

	if path == "" {
		path, err = os.Getwd()
		if err != nil {
			return "", "", err
		}
	}

	if !fp.IsAbs(path) {
		path, err = fp.Abs(path)
		if err != nil {
			return "", "", err
		}
	}

	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// make all dirs
			os.MkdirAll(path, 0666)
		} else {
			return "", "", err
		}
	}

	fpath := fp.Join(path, fileName)

	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0666)
	defer f.Close()

	if err != nil {
		return "", "", err
	}

	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}

	if err != nil {
		return "", "", err
	}

	return path, fileName, nil

}

func resourceOneandOneVPNDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	vpn, err := config.API.DeleteVPN(d.Id())
	if err != nil {
		return err
	}

	err = config.API.WaitUntilDeleted(vpn)
	if err != nil {
		return err
	}

	fullPath := fp.Join(d.Get("download_path").(string), d.Get("file_name").(string))
	if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
		os.Remove(fullPath)
	}

	return nil
}
