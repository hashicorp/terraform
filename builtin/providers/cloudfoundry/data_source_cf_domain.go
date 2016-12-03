package cloudfoundry

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/builtin/providers/cloudfoundry/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceDomain() *schema.Resource {

	return &schema.Resource{

		Read: dataSourceDomainRead,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"sub_domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDomainRead(d *schema.ResourceData, meta interface{}) error {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	dm := session.DomainManager()
	sharedDomains, err := dm.GetSharedDomains()
	if err != nil {
		return err
	}

	subDomain := d.Get("sub_domain").(string)

	var domain *cfapi.CCDomain
	prefix := subDomain + "."
	for _, d := range sharedDomains {
		if strings.HasPrefix(d.Name, prefix) {
			domain = &d
			break
		}
	}
	if domain == nil {
		return fmt.Errorf("no domain found with sub-domain '%s'", subDomain)
	}

	d.Set("name", domain.Name)
	d.Set("domain", domain.Name[len(prefix):])
	d.SetId(domain.ID)

	return nil
}
