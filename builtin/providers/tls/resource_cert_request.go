package tls

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceCertRequest() *schema.Resource {
	return &schema.Resource{
		Create: CreateCertRequest,
		Delete: DeleteCertRequest,
		Read:   ReadCertRequest,

		Schema: map[string]*schema.Schema{

			"dns_names": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of DNS names to use as subjects of the certificate",
				ForceNew:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"ip_addresses": &schema.Schema{
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of IP addresses to use as subjects of the certificate",
				ForceNew:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"key_algorithm": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the algorithm to use to generate the certificate's private key",
				ForceNew:    true,
			},

			"private_key_pem": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "PEM-encoded private key that the certificate will belong to",
				ForceNew:    true,
				StateFunc: func(v interface{}) string {
					return hashForState(v.(string))
				},
			},

			"subject": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     nameSchema,
				ForceNew: true,
			},

			"cert_request_pem": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func CreateCertRequest(d *schema.ResourceData, meta interface{}) error {
	keyAlgoName := d.Get("key_algorithm").(string)
	var keyFunc keyParser
	var ok bool
	if keyFunc, ok = keyParsers[keyAlgoName]; !ok {
		return fmt.Errorf("invalid key_algorithm %#v", keyAlgoName)
	}
	keyBlock, _ := pem.Decode([]byte(d.Get("private_key_pem").(string)))
	if keyBlock == nil {
		return fmt.Errorf("no PEM block found in private_key_pem")
	}
	key, err := keyFunc(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to decode private_key_pem: %s", err)
	}

	subjectConfs := d.Get("subject").([]interface{})
	if len(subjectConfs) != 1 {
		return fmt.Errorf("must have exactly one 'subject' block")
	}
	subjectConf := subjectConfs[0].(map[string]interface{})
	subject, err := nameFromResourceData(subjectConf)
	if err != nil {
		return fmt.Errorf("invalid subject block: %s", err)
	}

	certReq := x509.CertificateRequest{
		Subject: *subject,
	}

	dnsNamesI := d.Get("dns_names").([]interface{})
	for _, nameI := range dnsNamesI {
		certReq.DNSNames = append(certReq.DNSNames, nameI.(string))
	}
	ipAddressesI := d.Get("ip_addresses").([]interface{})
	for _, ipStrI := range ipAddressesI {
		ip := net.ParseIP(ipStrI.(string))
		if ip == nil {
			return fmt.Errorf("invalid IP address %#v", ipStrI.(string))
		}
		certReq.IPAddresses = append(certReq.IPAddresses, ip)
	}

	certReqBytes, err := x509.CreateCertificateRequest(rand.Reader, &certReq, key)
	if err != nil {
		fmt.Errorf("Error creating certificate request: %s", err)
	}
	certReqPem := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: certReqBytes}))

	d.SetId(hashForState(string(certReqBytes)))
	d.Set("cert_request_pem", certReqPem)

	return nil
}

func DeleteCertRequest(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func ReadCertRequest(d *schema.ResourceData, meta interface{}) error {
	return nil
}
