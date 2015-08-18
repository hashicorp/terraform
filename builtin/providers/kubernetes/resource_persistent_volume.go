package kubernetes

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client"
	"k8s.io/kubernetes/pkg/util/yaml"
)

func resourceKubernetesPersistentVolume() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesPersistentVolumeCreate,
		Read:   resourceKubernetesPersistentVolumeRead,
		Update: resourceKubernetesPersistentVolumeUpdate,
		Delete: resourceKubernetesPersistentVolumeDelete,

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
					s, err := normalizePersistentVolumeSpec(input.(string))
					if err != nil {
						log.Printf("[ERROR] Normalising spec failed: %q", err.Error())
					}
					return s
				},
			},
		},
	}
}

func resourceKubernetesPersistentVolumeCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	spec, err := expandPersistentVolumeSpec(d.Get("spec").(string))
	if err != nil {
		return err
	}

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.PersistentVolume{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		Spec: spec,
	}

	vol, err := c.PersistentVolumes().Create(&req)
	if err != nil {
		return err
	}

	d.SetId(string(vol.UID))

	return resourceKubernetesPersistentVolumeRead(d, meta)
}

func resourceKubernetesPersistentVolumeRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	vol, err := c.PersistentVolumes().Get(d.Get("name").(string))
	if err != nil {
		return err
	}

	spec, err := flattenPersistentVolumeSpec(vol.Spec)
	if err != nil {
		return err
	}
	d.Set("spec", spec)
	d.Set("labels", vol.Labels)
	d.Set("spec", vol.Spec)

	return nil
}

func resourceKubernetesPersistentVolumeUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	spec, err := expandPersistentVolumeSpec(d.Get("spec").(string))
	if err != nil {
		return err
	}

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.PersistentVolume{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		Spec: spec,
	}

	_, err = c.PersistentVolumes().Update(&req)
	if err != nil {
		return err
	}

	return resourceKubernetesPersistentVolumeRead(d, meta)
}

func resourceKubernetesPersistentVolumeDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	return c.PersistentVolumes().Delete(d.Get("name").(string))
}

func expandPersistentVolumeSpec(input string) (spec api.PersistentVolumeSpec, err error) {
	r := strings.NewReader(input)
	y := yaml.NewYAMLOrJSONDecoder(r, 4096)

	err = y.Decode(&spec)
	if err != nil {
		return
	}

	spec = setDefaultVolumeSpecValues(&spec)
	return
}

func flattenPersistentVolumeSpec(spec api.PersistentVolumeSpec) (string, error) {
	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func normalizePersistentVolumeSpec(input string) (string, error) {
	r := strings.NewReader(input)
	y := yaml.NewYAMLOrJSONDecoder(r, 4096)
	spec := api.PersistentVolumeSpec{}

	err := y.Decode(&spec)
	if err != nil {
		return "", err
	}

	spec = setDefaultVolumeSpecValues(&spec)

	b, err := json.Marshal(spec)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func setDefaultVolumeSpecValues(spec *api.PersistentVolumeSpec) api.PersistentVolumeSpec {
	if spec.PersistentVolumeReclaimPolicy == "" {
		spec.PersistentVolumeReclaimPolicy = "Retain"
	}
	return *spec
}
