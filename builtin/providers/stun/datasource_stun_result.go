package stun

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pixelbender/go-stun/stun"
)

func dataSourceStun() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceStunRead,

		Schema: map[string]*schema.Schema{
			"server": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "STUN server",
			},
			"ip_address": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "IP address",
			},
			"ip_family": &schema.Schema{
				Type:        schema.TypeString,
				Computed:    true,
				Description: "IP family (IPv4 or IPv6)",
			},
		},
	}
}

func dataSourceStunRead(d *schema.ResourceData, meta interface{}) error {
	addr, err := getResult(d)
	if err != nil {
		return err
	}
	d.Set("ip_address", addr.IP.String())
	d.Set("ip_family", "ipv4")
	if addr.IP.To4() == nil {
		d.Set("ip_family", "ipv6")
	}
	d.SetId(hash(addr.IP.String()))
	return nil
}

func getResult(d *schema.ResourceData) (*stun.Addr, error) {
	server := d.Get("server").(string)
	return stun.Lookup("stun:"+server, "", "")
}

func hash(s string) string {
	sha := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sha[:])
}
