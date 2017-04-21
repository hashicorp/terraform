package scaleway

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/scaleway/scaleway-cli/pkg/api"
)

func dataSourceScalewayBootscript() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceScalewayBootscriptRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"name_filter": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"architecture": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
			// Computed values.
			"organization": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"public": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"boot_cmd_args": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"dtb": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"initrd": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kernel": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func bootscriptDescriptionAttributes(d *schema.ResourceData, script api.ScalewayBootscript) error {
	d.Set("architecture", script.Arch)
	d.Set("organization", script.Organization)
	d.Set("public", script.Public)
	d.Set("boot_cmd_args", script.Bootcmdargs)
	d.Set("dtb", script.Dtb)
	d.Set("initrd", script.Initrd)
	d.Set("kernel", script.Kernel)
	d.SetId(script.Identifier)

	return nil
}

func dataSourceScalewayBootscriptRead(d *schema.ResourceData, meta interface{}) error {
	scaleway := meta.(*Client).scaleway

	scripts, err := scaleway.GetBootscripts()
	if err != nil {
		return err
	}

	var isMatch func(api.ScalewayBootscript) bool

	architecture := d.Get("architecture")
	if name, ok := d.GetOk("name"); ok {
		isMatch = func(s api.ScalewayBootscript) bool {
			architectureMatch := true
			if architecture != "" {
				architectureMatch = architecture == s.Arch
			}
			return s.Title == name.(string) && architectureMatch
		}
	} else if nameFilter, ok := d.GetOk("name_filter"); ok {
		exp, err := regexp.Compile(nameFilter.(string))
		if err != nil {
			return err
		}

		isMatch = func(s api.ScalewayBootscript) bool {
			nameMatch := exp.MatchString(s.Title)
			architectureMatch := true
			if architecture != "" {
				architectureMatch = architecture == s.Arch
			}
			return nameMatch && architectureMatch
		}
	}

	var matches []api.ScalewayBootscript
	for _, script := range *scripts {
		if isMatch(script) {
			matches = append(matches, script)
		}
	}

	if len(matches) > 1 {
		return fmt.Errorf("The query returned more than one result. Please refine your query.")
	}
	if len(matches) == 0 {
		return fmt.Errorf("The query returned no result. Please refine your query.")
	}

	return bootscriptDescriptionAttributes(d, matches[0])
}
