package cassandra

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gocql/gocql"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"time"
)

var quoteReplacer = strings.NewReplacer(`"`, `\"`)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"cassandra_keyspace": ResourceKeyspace(),
		},

		Schema: map[string]*schema.Schema{
			"hostport": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.EnvDefaultFunc(
					"CASSANDRA_HOSTPORT", "localhost:9042",
				),
			},
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CASSANDRA_USERNAME", ""),
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CASSANDRA_PASSWORD", ""),
			},
			"proto_version": &schema.Schema{
				Type:        schema.TypeInt,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CASSANDRA_PROTO_VERSION", 3),
			},
		},

		ConfigureFunc: Configure,
	}
}

func Configure(d *schema.ResourceData) (interface{}, error) {
	hostPortRegex, err := regexp.Compile(".*:\\d*")
	if err != nil {
		return nil, fmt.Errorf("Invalid regex: %s", err)
	}

	hostPort := d.Get("hostport").(string)
	if !hostPortRegex.MatchString(hostPort) {
		return nil, fmt.Errorf("invalid Cassandra Hostport: %s", err)
	}

	cluster := gocql.NewCluster(hostPort)
	cluster.ProtoVersion = d.Get("proto_version").(int)
	cluster.Keyspace = "system"
	cluster.Timeout = time.Second * time.Duration(3)

	username := d.Get("username").(string)
	if username != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: username,
			Password: d.Get("password").(string),
		}
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, err
	}

	return session, nil
}

func quoteIdentifier(ident string) string {
	return fmt.Sprintf(`"%s"`, quoteReplacer.Replace(ident))
}
