package google

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeTargetHttpsProxy() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeTargetHttpsProxyCreate,
		Read:   resourceComputeTargetHttpsProxyRead,
		Delete: resourceComputeTargetHttpsProxyDelete,
		Update: resourceComputeTargetHttpsProxyUpdate,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ssl_certificates": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"url_map": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceComputeTargetHttpsProxyCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	_sslCertificates := d.Get("ssl_certificates").([]interface{})
	sslCertificates := make([]string, len(_sslCertificates))

	for i, v := range _sslCertificates {
		sslCertificates[i] = v.(string)
	}

	proxy := &compute.TargetHttpsProxy{
		Name:            d.Get("name").(string),
		UrlMap:          d.Get("url_map").(string),
		SslCertificates: sslCertificates,
	}

	if v, ok := d.GetOk("description"); ok {
		proxy.Description = v.(string)
	}

	log.Printf("[DEBUG] TargetHttpsProxy insert request: %#v", proxy)
	op, err := config.clientCompute.TargetHttpsProxies.Insert(
		project, proxy).Do()
	if err != nil {
		return fmt.Errorf("Error creating TargetHttpsProxy: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, "Creating Target Https Proxy")
	if err != nil {
		return err
	}

	d.SetId(proxy.Name)

	return resourceComputeTargetHttpsProxyRead(d, meta)
}

func resourceComputeTargetHttpsProxyUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	d.Partial(true)

	if d.HasChange("url_map") {
		url_map := d.Get("url_map").(string)
		url_map_ref := &compute.UrlMapReference{UrlMap: url_map}
		op, err := config.clientCompute.TargetHttpsProxies.SetUrlMap(
			project, d.Id(), url_map_ref).Do()
		if err != nil {
			return fmt.Errorf("Error updating Target HTTPS proxy URL map: %s", err)
		}

		err = computeOperationWaitGlobal(config, op, "Updating Target Https Proxy URL Map")
		if err != nil {
			return err
		}

		d.SetPartial("url_map")
	}

	if d.HasChange("ssl_certificates") {
		proxy, err := config.clientCompute.TargetHttpsProxies.Get(
			project, d.Id()).Do()

		_old, _new := d.GetChange("ssl_certificates")
		_oldCerts := _old.([]interface{})
		_newCerts := _new.([]interface{})
		current := proxy.SslCertificates

		_oldMap := make(map[string]bool)
		_newMap := make(map[string]bool)

		for _, v := range _oldCerts {
			_oldMap[v.(string)] = true
		}

		for _, v := range _newCerts {
			_newMap[v.(string)] = true
		}

		sslCertificates := make([]string, 0)
		// Only modify certificates in one of our old or new states
		for _, v := range current {
			_, okOld := _oldMap[v]
			_, okNew := _newMap[v]

			// we deleted the certificate
			if okOld && !okNew {
				continue
			}

			sslCertificates = append(sslCertificates, v)

			// Keep track of the fact that we have added this certificate
			if okNew {
				delete(_newMap, v)
			}
		}

		// Add fresh certificates
		for k, _ := range _newMap {
			sslCertificates = append(sslCertificates, k)
		}

		cert_ref := &compute.TargetHttpsProxiesSetSslCertificatesRequest{
			SslCertificates: sslCertificates,
		}
		op, err := config.clientCompute.TargetHttpsProxies.SetSslCertificates(
			project, d.Id(), cert_ref).Do()
		if err != nil {
			return fmt.Errorf("Error updating Target Https Proxy SSL Certificates: %s", err)
		}

		err = computeOperationWaitGlobal(config, op, "Updating Target Https Proxy SSL certificates")
		if err != nil {
			return err
		}

		d.SetPartial("ssl_certificate")
	}

	d.Partial(false)

	return resourceComputeTargetHttpsProxyRead(d, meta)
}

func resourceComputeTargetHttpsProxyRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	proxy, err := config.clientCompute.TargetHttpsProxies.Get(
		project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing Target HTTPS Proxy %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading TargetHttpsProxy: %s", err)
	}

	_certs := d.Get("ssl_certificates").([]interface{})
	current := proxy.SslCertificates

	_certMap := make(map[string]bool)
	_newCerts := make([]interface{}, 0)

	for _, v := range _certs {
		_certMap[v.(string)] = true
	}

	// Store intersection of server certificates and user defined certificates
	for _, v := range current {
		if _, ok := _certMap[v]; ok {
			_newCerts = append(_newCerts, v)
		}
	}

	d.Set("ssl_certificates", _newCerts)
	d.Set("self_link", proxy.SelfLink)
	d.Set("id", strconv.FormatUint(proxy.Id, 10))

	return nil
}

func resourceComputeTargetHttpsProxyDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the TargetHttpsProxy
	log.Printf("[DEBUG] TargetHttpsProxy delete request")
	op, err := config.clientCompute.TargetHttpsProxies.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting TargetHttpsProxy: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, "Deleting Target Https Proxy")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
