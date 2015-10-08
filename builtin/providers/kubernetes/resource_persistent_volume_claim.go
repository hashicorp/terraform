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

func resourceKubernetesPersistentVolumeClaim() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesPersistentVolumeClaimCreate,
		Read:   resourceKubernetesPersistentVolumeClaimRead,
		Update: resourceKubernetesPersistentVolumeClaimUpdate,
		Delete: resourceKubernetesPersistentVolumeClaimDelete,

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
					s, err := normalizeVolumeClaimSpec(input.(string))
					if err != nil {
						log.Printf("[ERROR] Normalising spec failed: %q", err.Error())
					}
					return s
				},
			},
		},
	}
}

func resourceKubernetesPersistentVolumeClaimCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	spec, err := expandVolumeClaimSpec(d.Get("spec").(string))
	if err != nil {
		return err
	}

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.PersistentVolumeClaim{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		Spec: spec,
	}

	ns := d.Get("namespace").(string)

	pvc, err := c.PersistentVolumeClaims(ns).Create(&req)
	if err != nil {
		return err
	}

	d.SetId(string(pvc.UID))

	return resourceKubernetesPersistentVolumeClaimRead(d, meta)
}

func resourceKubernetesPersistentVolumeClaimRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	pvc, err := c.PersistentVolumeClaims(d.Get("namespace").(string)).Get(d.Get("name").(string))
	if err != nil {
		return err
	}

	spec, err := flattenVolumeClaimSpec(pvc.Spec)
	if err != nil {
		return err
	}
	d.Set("spec", spec)
	d.Set("labels", pvc.Labels)
	d.Set("spec", pvc.Spec)

	return nil
}

func resourceKubernetesPersistentVolumeClaimUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	spec, err := expandVolumeClaimSpec(d.Get("spec").(string))
	if err != nil {
		return err
	}

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.PersistentVolumeClaim{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		Spec: spec,
	}

	_, err = c.PersistentVolumeClaims(d.Get("namespace").(string)).Update(&req)
	if err != nil {
		return err
	}

	return resourceKubernetesPersistentVolumeClaimRead(d, meta)
}

func resourceKubernetesPersistentVolumeClaimDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	err := c.PersistentVolumeClaims(d.Get("namespace").(string)).Delete(d.Get("name").(string))
	return err
}

func expandVolumeClaimSpec(input string) (spec api.PersistentVolumeClaimSpec, err error) {
	r := strings.NewReader(input)
	y := yaml.NewYAMLOrJSONDecoder(r, 4096)

	err = y.Decode(&spec)
	if err != nil {
		return
	}

	return
}

func flattenVolumeClaimSpec(spec api.PersistentVolumeClaimSpec) (string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func normalizeVolumeClaimSpec(input string) (string, error) {
	r := strings.NewReader(input)
	y := yaml.NewYAMLOrJSONDecoder(r, 4096)
	spec := api.PersistentVolumeClaimSpec{}

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
