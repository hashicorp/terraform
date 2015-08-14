package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client"
)

func resourceKubernetesNamespace() *schema.Resource {
	return &schema.Resource{
		Create: resourceKubernetesNamespaceCreate,
		Read:   resourceKubernetesNamespaceRead,
		Update: resourceKubernetesNamespaceUpdate,
		Delete: resourceKubernetesNamespaceDelete,

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
		},
	}
}

func resourceKubernetesNamespaceCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.Namespace{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		// TODO: Spec.Finalizers[] ?
	}

	ns, err := c.Namespaces().Create(&req)
	if err != nil {
		return err
	}

	d.SetId(string(ns.UID))

	return resourceKubernetesNamespaceRead(d, meta)
}

func resourceKubernetesNamespaceRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	ns, err := c.Namespaces().Get(d.Get("name").(string))
	if err != nil {
		return err
	}

	d.Set("labels", ns.Labels)
	// TODO: d.Set("spec", ns.Spec)

	return nil
}

func resourceKubernetesNamespaceUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)

	l := d.Get("labels").(map[string]interface{})
	labels := make(map[string]string, len(l))
	for k, v := range l {
		labels[k] = v.(string)
	}

	req := api.Namespace{
		ObjectMeta: api.ObjectMeta{
			Name:   d.Get("name").(string),
			Labels: labels,
		},
		// TODO: Spec.Finalizers[] ?
	}

	_, err := c.Namespaces().Update(&req)
	if err != nil {
		return err
	}

	return resourceKubernetesNamespaceRead(d, meta)
}

func resourceKubernetesNamespaceDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*client.Client)
	err := c.Namespaces().Delete(d.Get("name").(string))
	return err
}
