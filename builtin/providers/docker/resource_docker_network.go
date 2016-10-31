package docker

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerNetworkCreate,
		Read:   resourceDockerNetworkRead,
		Delete: resourceDockerNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"check_duplicate": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"options": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"ipam_driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ipam_config": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     getIpamConfigElem(),
				Set:      resourceDockerIpamConfigHash,
			},

			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"scope": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func getIpamConfigElem() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"subnet": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ip_range": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"gateway": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"aux_address": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDockerIpamConfigHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["subnet"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["ip_range"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["gateway"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["aux_address"]; ok {
		auxAddress := v.(map[string]interface{})

		keys := make([]string, len(auxAddress))
		i := 0
		for k, _ := range auxAddress {
			keys[i] = k
			i++
		}
		sort.Strings(keys)

		for _, k := range keys {
			buf.WriteString(fmt.Sprintf("%v-%v-", k, auxAddress[k].(string)))
		}
	}

	return hashcode.String(buf.String())
}
