package random

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceId() *schema.Resource {
	return &schema.Resource{
		Create: CreateID,
		Read:   RepopulateEncodings,
		Delete: schema.RemoveFromState,

		Schema: map[string]*schema.Schema{
			"keepers": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"byte_length": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"prefix": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"b64": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Use b64_url for old behavior, or b64_std for standard base64 encoding",
			},

			"b64_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"b64_std": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"hex": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"dec": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func CreateID(d *schema.ResourceData, meta interface{}) error {
	byteLength := d.Get("byte_length").(int)
	bytes := make([]byte, byteLength)

	n, err := rand.Reader.Read(bytes)
	if n != byteLength {
		return errors.New("generated insufficient random bytes")
	}
	if err != nil {
		return errwrap.Wrapf("error generating random bytes: {{err}}", err)
	}

	b64Str := base64.RawURLEncoding.EncodeToString(bytes)
	d.SetId(b64Str)

	return RepopulateEncodings(d, meta)
}

func RepopulateEncodings(d *schema.ResourceData, _ interface{}) error {
	prefix := d.Get("prefix").(string)
	base64Str := d.Id()

	bytes, err := base64.RawURLEncoding.DecodeString(base64Str)
	if err != nil {
		return errwrap.Wrapf("Error decoding ID: {{err}}", err)
	}

	b64StdStr := base64.StdEncoding.EncodeToString(bytes)
	hexStr := hex.EncodeToString(bytes)

	bigInt := big.Int{}
	bigInt.SetBytes(bytes)
	decStr := bigInt.String()

	d.Set("b64", prefix+base64Str)
	d.Set("b64_url", prefix+base64Str)
	d.Set("b64_std", prefix+b64StdStr)

	d.Set("hex", prefix+hexStr)
	d.Set("dec", prefix+decStr)

	return nil
}
