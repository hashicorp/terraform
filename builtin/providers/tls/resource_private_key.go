package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/hashicorp/terraform/helper/schema"
)

type keyAlgo func(d *schema.ResourceData) (interface{}, error)
type keyParser func([]byte) (interface{}, error)

var keyAlgos map[string]keyAlgo = map[string]keyAlgo{
	"RSA": func(d *schema.ResourceData) (interface{}, error) {
		rsaBits := d.Get("rsa_bits").(int)
		return rsa.GenerateKey(rand.Reader, rsaBits)
	},
	"ECDSA": func(d *schema.ResourceData) (interface{}, error) {
		curve := d.Get("ecdsa_curve").(string)
		switch curve {
		case "P224":
			return ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
		case "P256":
			return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		case "P384":
			return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		case "P521":
			return ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		default:
			return nil, fmt.Errorf("invalid ecdsa_curve; must be P224, P256, P384 or P521")
		}
	},
}

var keyParsers map[string]keyParser = map[string]keyParser{
	"RSA": func(der []byte) (interface{}, error) {
		return x509.ParsePKCS1PrivateKey(der)
	},
	"ECDSA": func(der []byte) (interface{}, error) {
		return x509.ParseECPrivateKey(der)
	},
}

func resourcePrivateKey() *schema.Resource {
	return &schema.Resource{
		Create: CreatePrivateKey,
		Delete: DeletePrivateKey,
		Read:   ReadPrivateKey,

		Schema: map[string]*schema.Schema{
			"algorithm": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the algorithm to use to generate the private key",
				ForceNew:    true,
			},

			"rsa_bits": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Number of bits to use when generating an RSA key",
				ForceNew:    true,
				Default:     2048,
			},

			"ecdsa_curve": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "ECDSA curve to use when generating a key",
				ForceNew:    true,
				Default:     "P224",
			},

			"private_key_pem": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"public_key_pem": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"public_key_openssh": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func CreatePrivateKey(d *schema.ResourceData, meta interface{}) error {
	keyAlgoName := d.Get("algorithm").(string)
	var keyFunc keyAlgo
	var ok bool
	if keyFunc, ok = keyAlgos[keyAlgoName]; !ok {
		return fmt.Errorf("invalid key_algorithm %#v", keyAlgoName)
	}

	key, err := keyFunc(d)
	if err != nil {
		return err
	}

	var keyPemBlock *pem.Block
	switch k := key.(type) {
	case *rsa.PrivateKey:
		keyPemBlock = &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(k),
		}
	case *ecdsa.PrivateKey:
		keyBytes, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return fmt.Errorf("error encoding key to PEM: %s", err)
		}
		keyPemBlock = &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyBytes,
		}
	default:
		return fmt.Errorf("unsupported private key type")
	}
	keyPem := string(pem.EncodeToMemory(keyPemBlock))

	pubKey := publicKey(key)
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %s", err)
	}
	pubKeyPemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}

	d.SetId(hashForState(string((pubKeyBytes))))
	d.Set("private_key_pem", keyPem)
	d.Set("public_key_pem", string(pem.EncodeToMemory(pubKeyPemBlock)))

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err == nil {
		// Not all EC types can be SSH keys, so we'll produce this only
		// if an appropriate type was selected.
		sshPubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)
		d.Set("public_key_openssh", string(sshPubKeyBytes))
	} else {
		d.Set("public_key_openssh", "")
	}

	return nil
}

func DeletePrivateKey(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func ReadPrivateKey(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}
