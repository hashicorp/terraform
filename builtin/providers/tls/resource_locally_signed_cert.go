package tls

import (
	"crypto/x509"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceLocallySignedCert() *schema.Resource {
	s := resourceCertificateCommonSchema()

	s["cert_request_pem"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "PEM-encoded certificate request",
		ForceNew:    true,
		StateFunc: func(v interface{}) string {
			return hashForState(v.(string))
		},
	}

	s["ca_key_algorithm"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "Name of the algorithm used to generate the certificate's private key",
		ForceNew:    true,
	}

	s["ca_private_key_pem"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "PEM-encoded CA private key used to sign the certificate",
		ForceNew:    true,
		StateFunc: func(v interface{}) string {
			return hashForState(v.(string))
		},
	}

	s["ca_cert_pem"] = &schema.Schema{
		Type:        schema.TypeString,
		Required:    true,
		Description: "PEM-encoded CA certificate",
		ForceNew:    true,
		StateFunc: func(v interface{}) string {
			return hashForState(v.(string))
		},
	}

	return &schema.Resource{
		Create: CreateLocallySignedCert,
		Delete: DeleteCertificate,
		Read:   ReadCertificate,
		Schema: s,
	}
}

func CreateLocallySignedCert(d *schema.ResourceData, meta interface{}) error {
	certReq, err := parseCertificateRequest(d, "cert_request_pem")
	if err != nil {
		return err
	}
	caKey, err := parsePrivateKey(d, "ca_private_key_pem", "ca_key_algorithm")
	if err != nil {
		return err
	}
	caCert, err := parseCertificate(d, "ca_cert_pem")
	if err != nil {
		return err
	}

	cert := x509.Certificate{
		Subject:               certReq.Subject,
		DNSNames:              certReq.DNSNames,
		IPAddresses:           certReq.IPAddresses,
		BasicConstraintsValid: true,
	}

	return createCertificate(d, &cert, caCert, certReq.PublicKey, caKey)
}
