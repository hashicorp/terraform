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
		Read:   schema.Noop,
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

func CreateID(d *schema.ResourceData, _ interface{}) error {

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
	b64StdStr := base64.StdEncoding.EncodeToString(bytes)
	hexStr := hex.EncodeToString(bytes)

	bigInt := big.Int{}
	bigInt.SetBytes(bytes)
	decStr := bigInt.String()

	d.SetId(b64Str)

	d.Set("b64", b64Str)
	d.Set("b64_url", b64Str)
	d.Set("b64_std", b64StdStr)

	d.Set("hex", hexStr)
	d.Set("dec", decStr)

	return nil
}
