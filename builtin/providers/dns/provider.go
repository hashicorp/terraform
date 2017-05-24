package dns

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/miekg/dns"
)

// Provider returns a schema.Provider for DNS dynamic updates.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"update": &schema.Schema{
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"server": &schema.Schema{
							Type:        schema.TypeString,
							Required:    true,
							DefaultFunc: schema.EnvDefaultFunc("DNS_UPDATE_SERVER", nil),
						},
						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  53,
						},
						"key_name": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							DefaultFunc: schema.EnvDefaultFunc("DNS_UPDATE_KEYNAME", nil),
						},
						"key_algorithm": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							DefaultFunc: schema.EnvDefaultFunc("DNS_UPDATE_KEYALGORITHM", nil),
						},
						"key_secret": &schema.Schema{
							Type:        schema.TypeString,
							Optional:    true,
							DefaultFunc: schema.EnvDefaultFunc("DNS_UPDATE_KEYSECRET", nil),
						},
					},
				},
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"dns_a_record_set":     dataSourceDnsARecordSet(),
			"dns_cname_record_set": dataSourceDnsCnameRecordSet(),
			"dns_txt_record_set":   dataSourceDnsTxtRecordSet(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"dns_a_record_set":    resourceDnsARecordSet(),
			"dns_aaaa_record_set": resourceDnsAAAARecordSet(),
			"dns_cname_record":    resourceDnsCnameRecord(),
			"dns_ptr_record":      resourceDnsPtrRecord(),
		},

		ConfigureFunc: configureProvider,
	}
}

func configureProvider(d *schema.ResourceData) (interface{}, error) {

	var server, keyname, keyalgo, keysecret string
	var port int

	// if the update block is missing, schema.EnvDefaultFunc is not called
	if v, ok := d.GetOk("update"); ok {
		update := v.([]interface{})[0].(map[string]interface{})
		if val, ok := update["port"]; ok {
			port = int(val.(int))
		}
		if val, ok := update["server"]; ok {
			server = val.(string)
		}
		if val, ok := update["key_name"]; ok {
			keyname = val.(string)
		}
		if val, ok := update["key_algorithm"]; ok {
			keyalgo = val.(string)
		}
		if val, ok := update["key_secret"]; ok {
			keysecret = val.(string)
		}
	} else {
		if len(os.Getenv("DNS_UPDATE_SERVER")) > 0 {
			server = os.Getenv("DNS_UPDATE_SERVER")
		} else {
			return nil, nil
		}
		port = 53
		if len(os.Getenv("DNS_UPDATE_KEYNAME")) > 0 {
			keyname = os.Getenv("DNS_UPDATE_KEYNAME")
		}
		if len(os.Getenv("DNS_UPDATE_KEYALGORITHM")) > 0 {
			keyalgo = os.Getenv("DNS_UPDATE_KEYALGORITHM")
		}
		if len(os.Getenv("DNS_UPDATE_KEYSECRET")) > 0 {
			keysecret = os.Getenv("DNS_UPDATE_KEYSECRET")
		}
	}

	config := Config{
		server:    server,
		port:      port,
		keyname:   keyname,
		keyalgo:   keyalgo,
		keysecret: keysecret,
	}

	return config.Client()
}

func getAVal(record interface{}) (string, error) {

	recstr := record.(*dns.A).String()
	var name, ttl, class, typ, addr string

	_, err := fmt.Sscanf(recstr, "%s\t%s\t%s\t%s\t%s", &name, &ttl, &class, &typ, &addr)
	if err != nil {
		return "", fmt.Errorf("Error parsing record: %s", err)
	}

	return addr, nil
}

func getAAAAVal(record interface{}) (string, error) {

	recstr := record.(*dns.AAAA).String()
	var name, ttl, class, typ, addr string

	_, err := fmt.Sscanf(recstr, "%s\t%s\t%s\t%s\t%s", &name, &ttl, &class, &typ, &addr)
	if err != nil {
		return "", fmt.Errorf("Error parsing record: %s", err)
	}

	return addr, nil
}

func getCnameVal(record interface{}) (string, error) {

	recstr := record.(*dns.CNAME).String()
	var name, ttl, class, typ, cname string

	_, err := fmt.Sscanf(recstr, "%s\t%s\t%s\t%s\t%s", &name, &ttl, &class, &typ, &cname)
	if err != nil {
		return "", fmt.Errorf("Error parsing record: %s", err)
	}

	return cname, nil
}

func getPtrVal(record interface{}) (string, error) {

	recstr := record.(*dns.PTR).String()
	var name, ttl, class, typ, ptr string

	_, err := fmt.Sscanf(recstr, "%s\t%s\t%s\t%s\t%s", &name, &ttl, &class, &typ, &ptr)
	if err != nil {
		return "", fmt.Errorf("Error parsing record: %s", err)
	}

	return ptr, nil
}
