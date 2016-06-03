package random

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceId() *schema.Resource {
	return &schema.Resource{
		Create: CreateID,
		Read:   stubRead,
		Delete: stubDelete,

		Schema: map[string]*schema.Schema{
			"keepers": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"byte_length": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"b64": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"hex": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"dec": &schema.Schema{
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
		return fmt.Errorf("generated insufficient random bytes")
	}
	if err != nil {
		return fmt.Errorf("error generating random bytes: %s", err)
	}

	b64Str := base64.RawURLEncoding.EncodeToString(bytes)
	hexStr := hex.EncodeToString(bytes)

	int := big.Int{}
	int.SetBytes(bytes)
	decStr := int.String()

	d.SetId(b64Str)
	d.Set("b64", b64Str)
	d.Set("hex", hexStr)
	d.Set("dec", decStr)

	return nil
}
