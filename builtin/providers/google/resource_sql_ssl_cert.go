package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/api/googleapi"
)

func resourceSqlSslCert() *schema.Resource {
	return &schema.Resource{
		Create: resourceSqlSslCertCreate,
		Read:   resourceSqlSslCertRead,
		Delete: resourceSqlSslCertDelete,

		Schema: map[string]*schema.Schema{
			"server_ca_cert": sslCertInfo(false),

			"client_cert": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cert_private_key": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
							ForceNew: true,
						},

						"cert_info": sslCertInfo(true),
					},
				},
			},
		},
	}
}

func sslCertInfo(client bool) *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: client,
		ForceNew: true,
		Required: !client,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"cert": &schema.Schema{
					Type:     schema.TypeString,
					Computed: true,
					ForceNew: true,
				},

				"cert_serial_number": &schema.Schema{
					Type:     schema.TypeString,
					Computed: true,
					ForceNew: true,
				},

				"common_name": &schema.Schema{
					Type:     schema.TypeString,
					Computed: client,
					Required: !client,
					ForceNew: true,
				},

				"create_time": &schema.Schema{
					Type:     schema.TypeString,
					Computed: true,
					ForceNew: true,
				},

				"expiration_time": &schema.Schema{
					Type:     schema.TypeString,
					Computed: true,
					ForceNew: true,
				},

				"instance": &schema.Schema{
					Type:     schema.TypeString,
					Computed: client,
					Required: !client,
					ForceNew: true,
				},

				"self_link": &schema.Schema{
					Type:     schema.TypeString,
					Computed: true,
					ForceNew: true,
				},

				"sha1_fingerprint": &schema.Schema{
					Type:     schema.TypeString,
					Computed: true,
					ForceNew: true,
				},
			},
		},
	}
}

func readSslCertInfo(sslCertInfo *sqladmin.SslCert) []interface{} {
	_sslCertInfo := make(map[string]interface{})

	_sslCertInfo["cert"] = sslCertInfo.Cert
	_sslCertInfo["cert_serial_number"] = sslCertInfo.CertSerialNumber
	_sslCertInfo["common_name"] = sslCertInfo.CommonName
	_sslCertInfo["create_time"] = sslCertInfo.CreateTime
	_sslCertInfo["expiration_time"] = sslCertInfo.ExpirationTime
	_sslCertInfo["instance"] = sslCertInfo.Instance
	_sslCertInfo["self_link"] = sslCertInfo.SelfLink
	_sslCertInfo["sha1_fingerprint"] = sslCertInfo.Sha1Fingerprint

	_sslCertInfos := make([]interface{}, 1)
	_sslCertInfos[0] = sslCertInfo
	return _sslCertInfos
}

func resourceSqlSslCertCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if len(d.Get("server_ca_cert").([]interface{})) > 1 {
		return fmt.Errorf("At most one \"server_ca_cert\" block is allowed");
	}

	project := config.Project
	_serverCaCerts := d.Get("server_ca_cert").([]interface{})
	_serverCaCert := _serverCaCerts[0].(map[string]interface{})
	instance := _serverCaCert["instance"].(string)
	name := _serverCaCert["common_name"].(string)

	sslCert := &sqladmin.SslCertsInsertRequest{
		CommonName: name,
	}

	certResponse, err := config.clientSqlAdmin.SslCerts.Insert(project, instance, sslCert).Do()

	if err != nil {
		return fmt.Errorf("Failed to insert ssl cert %s: %s", name, err)
	}

	_clientCerts := make([]interface{}, 1);
	_clientCert := make(map[string]interface{})
	_clientCert["cert_private_key"] = certResponse.ClientCert.CertPrivateKey
	_clientCert["cert_info"] = readSslCertInfo(certResponse.ClientCert.CertInfo)
	_clientCerts[0] = _clientCert

	d.Set("client_cert", _clientCerts)
	d.Set("server_ca_cert", readSslCertInfo(certResponse.ServerCaCert))
	d.SetId(name)

	return nil
}

func resourceSqlSslCertRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project := config.Project
	_serverCaCerts := d.Get("server_ca_cert").([]interface{})
	_serverCaCert := _serverCaCerts[0].(map[string]interface{})
	instance := _serverCaCert["instance"].(string)
	name := _serverCaCert["common_name"].(string)
	sha1 := _serverCaCert["sha1_fingerprint"].(string)

	sslCert, err := config.clientSqlAdmin.SslCerts.Get(project, instance, sha1).Do()

	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			d.SetId("")

			log.Printf("[WARN] Sql Cert %s no longer exists", name)
			return nil
		}

		return fmt.Errorf("Error trying to retrieve SSL certificate from instance %s: %s", instance, err)
	}

	_clientCerts := make([]interface{}, 1);
	_clientCert := make(map[string]interface{})
	_clientCert["cert_info"] = readSslCertInfo(sslCert)
	_clientCerts[0] = _clientCert

	d.Set("client_cert", _clientCerts)

	return nil
}

func resourceSqlSslCertDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project := config.Project
	_serverCaCerts := d.Get("server_ca_cert").([]interface{})
	_serverCaCert := _serverCaCerts[0].(map[string]interface{})
	instance := _serverCaCert["instance"].(string)
	sha1 := _serverCaCert["sha1_fingerprint"].(string)

	_, err := config.clientSqlAdmin.SslCerts.Delete(project, instance, sha1).Do()

	if err != nil {
		return fmt.Errorf("Error trying to delete SSL certificate on instance %s: %s", instance, err)
	}

	return nil
}
