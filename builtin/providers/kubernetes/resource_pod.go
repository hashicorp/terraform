package kubernetes

import (
	"log"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
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
				Default:  "Always",
			},

			"termination_grace_period_seconds": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  30,
			},

			"active_deadline_seconds": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},

			"dns_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "Default",
			},

			"node_selector": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     schema.TypeString,
			},

			"service_account_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
			},

			"node_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"security_context": genSecurityContext(),

			"image_pull_secret": genLocalObjectReference(),
		},
	}
}

func resourceKubernetesPodCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	log.Printf("[DEBUG] preparing to create pod");

	_name := d.Get("name").(string)

	spec := api.PodSpec{}

	spec.Volumes = createVolumes(d.Get("volume").([]interface{}))

	spec.Containers = createContainers(d.Get("container").([]interface{}))

	if v, ok := d.GetOk("dns_policy"); ok {
		spec.DNSPolicy = api.DNSPolicy(v.(string))
	}

	if v, ok := d.GetOk("node_selector"); ok {
		spec.NodeSelector = createNodeSelector(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("restart_policy"); ok {
		spec.RestartPolicy = api.RestartPolicy(v.(string))
	}

	if v, ok := d.GetOk("termination_grace_period_seconds"); ok {
		val := int64(v.(int))
		spec.TerminationGracePeriodSeconds = &val
	}

	if v, ok := d.GetOk("active_deadline_seconds"); ok {
		val := int64(v.(int))
		spec.ActiveDeadlineSeconds = &val
	}

	if v, ok := d.GetOk("service_account_name"); ok {
		spec.ServiceAccountName = v.(string)
	}

	if v, ok := d.GetOk("security_context"); ok {
		spec.SecurityContext = createPodSecurityContext(v.([]interface{}))
	}

	if v, ok := d.GetOk("image_pull_secret"); ok {
		spec.ImagePullSecrets = createImagePullSecrets(v.([]interface{}))
	}

	_labels := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(_labels))
	for k, v := range _labels {
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

	_, err := c.Pods(_namespace).Create(&req)
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to create pod %s: %s", _name, err)
	}

	return resourceKubernetesPodRead(d, meta)
}

func resourceKubernetesPodRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	pod, err := c.Pods(d.Get("namespace").(string)).Get(d.Get("name").(string))
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to read pod %s: %s", d.Get("name").(string), err)
	}

	spec := pod.Spec

	d.Set("volume", readVolumes(spec.Volumes))
	d.Set("containers", readContainers(spec.Containers))
	d.Set("dns_policy", spec.DNSPolicy)
	d.Set("node_selector", readNodeSelector(spec.NodeSelector))
	d.Set("restart_policy", spec.RestartPolicy)
	v := spec.TerminationGracePeriodSeconds
	if v != nil {
		d.Set("termination_grace_period_seconds", *v)
	}
	v = spec.ActiveDeadlineSeconds
	if v != nil {
		d.Set("active_deadline_seconds", *v)
	}
	d.Set("service_account_name", spec.ServiceAccountName)
	d.Set("node_name", spec.NodeName)
	d.Set("security_context", readPodSecurityContext(spec.SecurityContext))
	d.Set("image_pull_secret", readImagePullSecrets(spec.ImagePullSecrets))

	labels := pod.ObjectMeta.Labels
	_labels := make(map[string]interface{}, len(labels))
	for k, v := range labels {
		_labels[k] = v
	}

	d.SetId(string(pod.UID))

	return nil
}

func resourceKubernetesPodUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	_name := d.Get("name").(string)

	spec := api.PodSpec{}

	spec.Volumes = createVolumes(d.Get("volume").([]interface{}))

	spec.Containers = createContainers(d.Get("container").([]interface{}))

	if v, ok := d.GetOk("dns_policy"); ok {
		spec.DNSPolicy = api.DNSPolicy(v.(string))
	}

	if v, ok := d.GetOk("node_selector"); ok {
		spec.NodeSelector = createNodeSelector(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("restart_policy"); ok {
		spec.RestartPolicy = api.RestartPolicy(v.(string))
	}

	if v, ok := d.GetOk("termination_grace_period_seconds"); ok {
		val := int64(v.(int))
		spec.TerminationGracePeriodSeconds = &val
	}

	if v, ok := d.GetOk("active_deadline_seconds"); ok {
		val := int64(v.(int))
		spec.ActiveDeadlineSeconds = &val
	}

	if v, ok := d.GetOk("service_account_name"); ok {
		spec.ServiceAccountName = v.(string)
	}

	if v, ok := d.GetOk("node_name"); ok {
		spec.NodeName = v.(string)
	}

	if v, ok := d.GetOk("security_context"); ok {
		spec.SecurityContext = createPodSecurityContext(v.([]interface{}))
	}

	if v, ok := d.GetOk("image_pull_secret"); ok {
		spec.ImagePullSecrets = createImagePullSecrets(v.([]interface{}))
	}

	_labels := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(_labels))
	for k, v := range _labels {
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

	_, err := c.Pods(_namespace).Update(&req)
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
