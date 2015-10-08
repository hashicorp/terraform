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

func resourceKubernetesLimitRange() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesLimitRangeCreate,
		Read:   resourceKubernetesLimitRangeRead,
		Update: resourceKubernetesLimitRangeUpdate,
		Delete: resourceKubernetesLimitRangeDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"namespace": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  api.NamespaceDefault,
			},

			"labels": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},

			"spec": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				StateFunc: func(input interface{}) string {
					s, err := normalizeLimitRangeSpec(input.(string))
					if err != nil {
						log.Printf("[ERROR] Normalising spec failed: %q", err.Error())
					}
					return s
				},
			},
		},
	}
}

func resourceKubernetesLimitRangeCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	spec, err := expandLimitRangeSpec(d.Get("spec").(string))
	if err != nil {
		return err
	}

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.LimitRange{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		Spec: spec,
	}

	lr, err := c.LimitRanges(d.Get("namespace").(string)).Create(&req)
	if err != nil {
		return err
	}

	d.SetId(string(lr.UID))

	return resourceKubernetesLimitRangeRead(d, meta)
}

func resourceKubernetesLimitRangeRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	lr, err := c.LimitRanges(d.Get("namespace").(string)).Get(d.Get("name").(string))
	if err != nil {
		return err
	}

	spec, err := flattenLimitRangeSpec(lr.Spec)
	if err != nil {
		return err
	}
	d.Set("spec", spec)
	d.Set("labels", lr.Labels)
	d.Set("spec", lr.Spec)

	return nil
}

func resourceKubernetesLimitRangeUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	spec, err := expandLimitRangeSpec(d.Get("spec").(string))
	if err != nil {
		return err
	}

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.LimitRange{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		Spec: spec,
	}

	_, err = c.LimitRanges(d.Get("namespace").(string)).Update(&req)
	if err != nil {
		return err
	}

	return resourceKubernetesLimitRangeRead(d, meta)
}

func resourceKubernetesLimitRangeDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	return c.LimitRanges(d.Get("namespace").(string)).Delete(d.Get("name").(string))
}

func expandLimitRangeSpec(input string) (spec api.LimitRangeSpec, err error) {
	r := strings.NewReader(input)
	y := yaml.NewYAMLOrJSONDecoder(r, 4096)

	err = y.Decode(&spec)
	if err != nil {
		return
	}
	return
}

func flattenLimitRangeSpec(spec api.LimitRangeSpec) (string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func normalizeLimitRangeSpec(input string) (string, error) {
	r := strings.NewReader(input)
	y := yaml.NewYAMLOrJSONDecoder(r, 4096)
	spec := api.LimitRangeSpec{}

	err := y.Decode(&spec)
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
