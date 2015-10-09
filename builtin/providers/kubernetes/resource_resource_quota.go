package kubernetes

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/util/yaml"
)

func resourceKubernetesResourceQuota() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesResourceQuotaCreate,
		Read:   resourceKubernetesResourceQuotaRead,
		Update: resourceKubernetesResourceQuotaUpdate,
		Delete: resourceKubernetesResourceQuotaDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"labels": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},

			"spec": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(input interface{}) string {
					s, err := normalizeResourceQuotaSpec(input.(string))
					if err != nil {
						log.Printf("[ERROR] Normalising spec failed: %q", err.Error())
					}
					return s
				},
			},
		},
	}
}

func resourceKubernetesResourceQuotaCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	spec, err := expandResourceQuotaSpec(d.Get("spec").(string))
	if err != nil {
		return err
	}

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.ResourceQuota{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		Spec: spec,
	}

	rq, err := c.ResourceQuotas(d.Get("namespace").(string)).Create(&req)
	if err != nil {
		return err
	}

	d.SetId(string(rq.UID))

	return resourceKubernetesResourceQuotaRead(d, meta)
}

func resourceKubernetesResourceQuotaRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	rq, err := c.ResourceQuotas(d.Get("namespace").(string)).Get(d.Get("name").(string))
	if err != nil {
		return err
	}

	spec, err := flattenResourceQuotaSpec(rq.Spec)
	if err != nil {
		return err
	}
	d.Set("spec", spec)
	d.Set("labels", rq.Labels)
	d.Set("spec", rq.Spec)

	return nil
}

func resourceKubernetesResourceQuotaUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	spec, err := expandResourceQuotaSpec(d.Get("spec").(string))
	if err != nil {
		return err
	}

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.ResourceQuota{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		Spec: spec,
	}

	_, err = c.ResourceQuotas(d.Get("namespace").(string)).Update(&req)
	if err != nil {
		return err
	}

	return resourceKubernetesResourceQuotaRead(d, meta)
}

func resourceKubernetesResourceQuotaDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	return c.ResourceQuotas(d.Get("namespace").(string)).Delete(d.Get("name").(string))
}

func expandResourceQuotaSpec(input string) (spec api.ResourceQuotaSpec, err error) {
	r := strings.NewReader(input)
	y := yaml.NewYAMLOrJSONDecoder(r, 4096)

	err = y.Decode(&spec)
	if err != nil {
		return
	}
	return
}

func flattenResourceQuotaSpec(spec api.ResourceQuotaSpec) (string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func normalizeResourceQuotaSpec(input string) (string, error) {
	r := strings.NewReader(input)
	y := yaml.NewYAMLOrJSONDecoder(r, 4096)
	spec := api.ResourceQuotaSpec{}

	err := y.Decode(&spec)
	if err != nil {
		return "", err
	}

	// TODO: Add/ignore default structures, e.g.

	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
