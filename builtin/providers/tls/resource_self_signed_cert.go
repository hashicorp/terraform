package tls

import (
	"crypto/x509"
	"fmt"
	"net"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceSelfSignedCert() *schema.Resource {
	s := resourceCertificateCommonSchema()

	s["subject"] = &schema.Schema{
		Type:     schema.TypeList,
		Required: true,
		Elem:     nameSchema,
		ForceNew: true,
	}

	s["dns_names"] = &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "List of DNS names to use as subjects of the certificate",
		ForceNew:    true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}

	s["ip_addresses"] = &schema.Schema{
		Type:        schema.TypeList,
		Optional:    true,
		Description: "List of IP addresses to use as subjects of the certificate",
		ForceNew:    true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
		},
	}

	s["key_algorithm"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "Name of the algorithm to use to generate the certificate's private key",
		ForceNew:    true,
	}

	s["private_key_pem"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "PEM-encoded private key that the certificate will belong to",
		ForceNew:    true,
		StateFunc: func(v interface{}) string {
			return hashForState(v.(string))
		},
	}

	return &schema.Resource{
		Create: CreateSelfSignedCert,
		Delete: DeleteCertificate,
		Read:   ReadCertificate,
		Schema: s,
	}
}

func CreateSelfSignedCert(d *schema.ResourceData, meta interface{}) error {
	key, err := parsePrivateKey(d, "private_key_pem", "key_algorithm")
	if err != nil {
		return err
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

	cert := x509.Certificate{
		Subject:               *subject,
		BasicConstraintsValid: true,
	}

	dnsNamesI := d.Get("dns_names").([]interface{})
	for _, nameI := range dnsNamesI {
		cert.DNSNames = append(cert.DNSNames, nameI.(string))
	}
	ipAddressesI := d.Get("ip_addresses").([]interface{})
	for _, ipStrI := range ipAddressesI {
		ip := net.ParseIP(ipStrI.(string))
		if ip == nil {
			return fmt.Errorf("invalid IP address %#v", ipStrI.(string))
		}
		cert.IPAddresses = append(cert.IPAddresses, ip)
	}

	return createCertificate(d, &cert, &cert, publicKey(key), key)
}
