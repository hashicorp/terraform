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

func resourceKubernetesPod() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesPodCreate,
		Read:   resourceKubernetesPodRead,
		Update: resourceKubernetesPodUpdate,
		Delete: resourceKubernetesPodDelete,

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

			"volume": genVolume(),

			"container": genContainer(),

			"restart_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"termination_grace_period_seconds": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"active_deadline_seconds": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"dns_policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"node_selector": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
				Elem:     schema.TypeString,
			},

			"service_account_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"node_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"security_context": genSecurityContext(),

			"image_pull_secret": genLocalObjectReference(),
		},
	}
}

func resourceKubernetesPodCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	_name := d.Get("name").(string)

	spec := &api.PodSpec{}

	if res, err := createVolumes(d.Get("volume")); err == nil {
		spec.Volumes = res
	} else {
		return err
	}

	if res, err := createContainers(d.Get("container").([]interface{}); err == nil {
		spec.Containers = res
	} else {
		return err
	}

	if res, err := createDnsPolicy(d.Get("dns_policy")); err == nil {
		spec.DNSPolicy = res
	} else {
		return err
	}

	if res, err := createNodeSelector(d.Get("node_selector")); err == nil {
		spec.NodeSelector = res
	} else {
		return err
	}

	if v, ok := d.GetOk("restart_policy") {
		spec.RestartPolicy = v.(string)
	}

	if v, ok := d.GetOk("termination_grace_period_seconds") {
		spec.TeriminationGracePeriodSecons = &(int64(v.(int)))
	}

	if v, ok := d.GetOk("active_deadline_seconds") {
		spec.ActiveDeadlineSeconds = &(int64(v.(int)))
	}

	if v, ok := d.GetOk("service_account_name") {
		spec.ServiceAccountName = v.(string)
	}

	if v, ok := d.GetOk("node_name") {
		spec.NodeName = v.(string)
	}

	if v, ok := d.GetOk("security_context") {
		if res, err := createSecurityContext(v); err == nil {
			spec.SecurityContext = res
		} else {
			return err
		}
	}

	if v, ok := d.GetOk("image_pull_secret") {
		if res, err := createImagePullSecret(v); err == nil {
			sepc.ImagePullSecret = res
		} else {
			return err
		}
	}


	_labels := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(_labels))
	for k, v := range _labels{
		labels[k] = v.(string)
	}

	req := api.Pod{
		ObjectMeta: api.ObjectMeta{
			Name:   _name,
			Labels: labels,
		},
		Spec: spec,
	}

	_namespace := d.Get("namespace").(string)

	pod, err := c.Pods(_namespace).Create(&req)
	if err != nil {
		return err
	}

	d.SetId(string(pod.UID))

	return resourceKubernetesPodRead(d, meta)

	return nil
}

func resourceKubernetesPodRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	pod, err := c.Pods(d.Get("namespace").(string)).Get(d.Get("name").(string))
	if err != nil {
		return err
	}

	spec, err := flattenPodSpec(pod.Spec)
	if err != nil {
		return err
	}
	d.Set("spec", spec)
	d.Set("labels", pod.Labels)
	d.Set("spec", pod.Spec)

	return nil
}

func resourceKubernetesPodUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	spec, err := expandPodSpec(d.Get("spec").(string))
	if err != nil {
		return err
	}

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.Pod{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		Spec: spec,
	}

	_, err = c.Pods(d.Get("namespace").(string)).Update(&req)
	if err != nil {
		return err
	}

	return resourceKubernetesPodRead(d, meta)
}

func resourceKubernetesPodDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	err := c.Pods(d.Get("namespace").(string)).Delete(d.Get("name").(string), nil)
	return err
}
