package tls

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

var keyUsages map[string]x509.KeyUsage = map[string]x509.KeyUsage{
	"digital_signature":  x509.KeyUsageDigitalSignature,
	"content_commitment": x509.KeyUsageContentCommitment,
	"key_encipherment":   x509.KeyUsageKeyEncipherment,
	"data_encipherment":  x509.KeyUsageDataEncipherment,
	"key_agreement":      x509.KeyUsageKeyAgreement,
	"cert_signing":       x509.KeyUsageCertSign,
	"crl_signing":        x509.KeyUsageCRLSign,
	"encipher_only":      x509.KeyUsageEncipherOnly,
	"decipher_only":      x509.KeyUsageDecipherOnly,
}

var extKeyUsages map[string]x509.ExtKeyUsage = map[string]x509.ExtKeyUsage{
	"any_extended":                  x509.ExtKeyUsageAny,
	"server_auth":                   x509.ExtKeyUsageServerAuth,
	"client_auth":                   x509.ExtKeyUsageClientAuth,
	"code_signing":                  x509.ExtKeyUsageCodeSigning,
	"email_protection":              x509.ExtKeyUsageEmailProtection,
	"ipsec_end_system":              x509.ExtKeyUsageIPSECEndSystem,
	"ipsec_tunnel":                  x509.ExtKeyUsageIPSECTunnel,
	"ipsec_user":                    x509.ExtKeyUsageIPSECUser,
	"timestamping":                  x509.ExtKeyUsageTimeStamping,
	"ocsp_signing":                  x509.ExtKeyUsageOCSPSigning,
	"microsoft_server_gated_crypto": x509.ExtKeyUsageMicrosoftServerGatedCrypto,
	"netscape_server_gated_crypto":  x509.ExtKeyUsageNetscapeServerGatedCrypto,
}

func resourceSelfSignedCert() *schema.Resource {
	return &schema.Resource{
		Create: CreateSelfSignedCert,
		Delete: DeleteSelfSignedCert,
		Read:   ReadSelfSignedCert,

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

			"validity_period_hours": &schema.Schema{
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Number of hours that the certificate will remain valid for",
				ForceNew:    true,
			},

			"early_renewal_hours": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Number of hours before the certificates expiry when a new certificate will be generated",
				ForceNew:    true,
			},

			"is_ca_certificate": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether the generated certificate will be usable as a CA certificate",
				ForceNew:    true,
			},

			"allowed_uses": &schema.Schema{
				Type:        schema.TypeList,
				Required:    true,
				Description: "Uses that are allowed for the certificate",
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

			"cert_pem": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"validity_start_time": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"validity_end_time": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func CreateSelfSignedCert(d *schema.ResourceData, meta interface{}) error {
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

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Duration(d.Get("validity_period_hours").(int)) * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %s", err)
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
		SerialNumber:          serialNumber,
		Subject:               *subject,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		BasicConstraintsValid: true,
	}

	keyUsesI := d.Get("allowed_uses").([]interface{})
	for _, keyUseI := range keyUsesI {
		keyUse := keyUseI.(string)
		if usage, ok := keyUsages[keyUse]; ok {
			cert.KeyUsage |= usage
		}
		if usage, ok := extKeyUsages[keyUse]; ok {
			cert.ExtKeyUsage = append(cert.ExtKeyUsage, usage)
		}
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

	if d.Get("is_ca_certificate").(bool) {
		cert.IsCA = true
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &cert, &cert, publicKey(key), key)
	if err != nil {
		fmt.Errorf("Error creating certificate: %s", err)
	}
	certPem := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}))

	validFromBytes, err := notBefore.MarshalText()
	if err != nil {
		return fmt.Errorf("error serializing validity_start_time: %s", err)
	}
	validToBytes, err := notAfter.MarshalText()
	if err != nil {
		return fmt.Errorf("error serializing validity_end_time: %s", err)
	}

	d.SetId(serialNumber.String())
	d.Set("cert_pem", certPem)
	d.Set("validity_start_time", string(validFromBytes))
	d.Set("validity_end_time", string(validToBytes))

	return nil
}

func DeleteSelfSignedCert(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func ReadSelfSignedCert(d *schema.ResourceData, meta interface{}) error {

	endTimeStr := d.Get("validity_end_time").(string)
	endTime := time.Now()
	err := endTime.UnmarshalText([]byte(endTimeStr))
	if err != nil {
		// If end time is invalid then we'll just throw away the whole
		// thing so we can generate a new one.
		d.SetId("")
		return nil
	}

	earlyRenewalPeriod := time.Duration(-d.Get("early_renewal_hours").(int)) * time.Hour
	endTime = endTime.Add(earlyRenewalPeriod)

	if time.Now().After(endTime) {
		// Treat an expired certificate as not existing, so we'll generate
		// a new one with the next plan.
		d.SetId("")
	}

	return nil
}
