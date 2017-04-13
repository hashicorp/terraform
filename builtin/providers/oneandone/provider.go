package oneandone

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/1and1/oneandone-cloudserver-sdk-go"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ONEANDONE_TOKEN", nil),
				Description: "1&1 token for API operations.",
			},
			"retries": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  50,
			},
			"endpoint": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  oneandone.BaseUrl,
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"oneandone_server":            resourceOneandOneServer(),
			"oneandone_firewall_policy":   resourceOneandOneFirewallPolicy(),
			"oneandone_private_network":   resourceOneandOnePrivateNetwork(),
			"oneandone_public_ip":         resourceOneandOnePublicIp(),
			"oneandone_shared_storage":    resourceOneandOneSharedStorage(),
			"oneandone_monitoring_policy": resourceOneandOneMonitoringPolicy(),
			"oneandone_loadbalancer":      resourceOneandOneLoadbalancer(),
			"oneandone_vpn":               resourceOneandOneVPN(),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	var endpoint string
	if d.Get("endpoint").(string) != oneandone.BaseUrl {
		endpoint = d.Get("endpoint").(string)
	}
	config := Config{
		Token:    d.Get("token").(string),
		Retries:  d.Get("retries").(int),
		Endpoint: endpoint,
	}
	return config.Client()
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
