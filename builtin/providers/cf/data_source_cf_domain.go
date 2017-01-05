package cloudfoundry

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/builtin/providers/cf/cfapi"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceDomain() *schema.Resource {

	return &schema.Resource{

		Read: dataSourceDomainRead,

		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"sub_domain": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
			"domain": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"name"},
			},
			"org": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceDomainRead(d *schema.ResourceData, meta interface{}) (err error) {

	session := meta.(*cfapi.Session)
	if session == nil {
		return fmt.Errorf("client is nil")
	}

	dm := session.DomainManager()

	var (
		name, prefix                           string
		sharedDomains, privateDomains, domains []cfapi.CCDomain
		domain                                 *cfapi.CCDomain
	)

	if sharedDomains, err = dm.GetSharedDomains(); err != nil {
		return
	}
	if privateDomains, err = dm.GetPrivateDomains(); err != nil {
		return
	}
	domains = append(sharedDomains, privateDomains[:0]...)

	if v, ok := d.GetOk("sub_domain"); ok {
		prefix = v.(string) + "."
		if v, ok = d.GetOk("domain"); ok {
			name = prefix + v.(string)
		}
	} else if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else {
		return fmt.Errorf("neither a full name or sub-domain was provided to do an effective domain search")
	}

	if len(name) == 0 {
		for _, d := range domains {
			if strings.HasPrefix(d.Name, prefix) {
				domain = &d
				break
			}
		}
		if domain == nil {
			return fmt.Errorf("no domain found with sub-domain '%s'", prefix)
		}
	} else {
		for _, d := range domains {
			if name == d.Name {
				domain = &d
				break
			}
		}
		if domain == nil {
			return fmt.Errorf("no domain found with name '%s'", name)
		}
	}

	domainParts := strings.Split(domain.Name, ".")

	d.Set("name", domain.Name)
	d.Set("sub_domain", domainParts[0])
	d.Set("domain", strings.Join(domainParts[1:], "."))
	d.Set("org", domain.OwningOrganizationGUID)
	d.SetId(domain.ID)
	return
}
