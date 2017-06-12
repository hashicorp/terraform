package google

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
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
		cert, err := canonicalizeCertUrl(v.(string))
		if err != nil {
			return err
		}
		sslCertificates[i] = cert
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

	err = computeOperationWaitGlobal(config, op, project, "Creating Target Https Proxy")
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

		err = computeOperationWaitGlobal(config, op, project, "Updating Target Https Proxy URL Map")
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
			cert, err := canonicalizeCertUrl(v.(string))
			if err != nil {
				return err
			}
			_oldMap[cert] = true
		}

		for _, v := range _newCerts {
			cert, err := canonicalizeCertUrl(v.(string))
			if err != nil {
				return err
			}
			_newMap[cert] = true
		}

		sslCertificates := make([]string, 0)
		// Only modify certificates in one of our old or new states
		for _, v := range current {
			cert, err := canonicalizeCertUrl(v)
			if err != nil {
				return err
			}
			_, okOld := _oldMap[cert]
			_, okNew := _newMap[cert]

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

		err = computeOperationWaitGlobal(config, op, project, "Updating Target Https Proxy SSL certificates")
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
		return handleNotFoundError(err, d, fmt.Sprintf("Target HTTPS proxy %q", d.Get("name").(string)))
	}

	userSpecifiedCerts := d.Get("ssl_certificates").([]interface{})
	actualCerts := proxy.SslCertificates
	certMap := make(map[string]bool)
	certs := make([]interface{}, 0)

	for _, v := range actualCerts {
		cert, err := canonicalizeCertUrl(v)
		if err != nil {
			return err
		}
		certMap[cert] = true
	}

	// Store intersection of server certificates and user defined certificates
	for _, v := range userSpecifiedCerts {
		cert, err := canonicalizeCertUrl(v.(string))
		if err != nil {
			return err
		}
		if _, ok := certMap[cert]; ok {
			certs = append(certs, v.(string))
		}
	}

	d.Set("ssl_certificates", certs)
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

	err = computeOperationWaitGlobal(config, op, project, "Deleting Target Https Proxy")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
