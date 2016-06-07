package swift

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/ncw/swift"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SWIFT_USERNAME", nil),
				Description: "The user name to use for Swift API operations.",
			},
			"api_key": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SWIFT_API_KEY", nil),
				Description: "The API key to use for Swift API operations.",
			},
			"auth_url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SWIFT_AUTH_URL", nil),
				Description: "The swifth object storage url to use for authentication.",
			},
			"storage_url": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SWIFT_STORAGE_URL", ""),
				Description: "Alternate object storage url to access containers in (defaults to storage url returned by auth api)",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"swift_container": resourceSwiftContainer(),
			"swift_object":    resourceSwiftObject(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	c := swift.Connection{
		UserName: d.Get("username").(string),
		ApiKey:   d.Get("api_key").(string),
		AuthUrl:  d.Get("auth_url").(string),
	}

	storage_url := d.Get("storage_url").(string)
	if storage_url != "" {
		c.StorageUrl = storage_url
	}

	err := c.Authenticate()

	return &c, err
}

func obtainConnection(meta interface{}) *swift.Connection {
	c := meta.(*swift.Connection)
	if c == nil {
		panic("swift container resource creation: The connection object was nil.")
	}

	return c
}
