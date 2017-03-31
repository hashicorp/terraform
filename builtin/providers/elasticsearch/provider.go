package elasticsearch

import (
	"net/http"
	"net/url"
	"regexp"

	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	awssigv4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/deoxxa/aws_signing_client"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	elastic "gopkg.in/olivere/elastic.v5"
)

var awsUrlRegexp = regexp.MustCompile(`([a-z0-9-]+).es.amazonaws.com$`)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("ELASTICSEARCH_URL", nil),
				Description: "Elasticsearch URL",
			},

			"aws_access_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The access key for use with AWS Elasticsearch Service domains",
			},

			"aws_secret_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The secret key for use with AWS Elasticsearch Service domains",
			},

			"aws_token": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The session token for use with AWS Elasticsearch Service domains",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"elasticsearch_index_template":      resourceElasticsearchIndexTemplate(),
			"elasticsearch_snapshot_repository": resourceElasticsearchSnapshotRepository(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	rawUrl := d.Get("url").(string)
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	opts := []elastic.ClientOptionFunc{
		elastic.SetURL(rawUrl),
		elastic.SetScheme(parsedUrl.Scheme),
	}

	if m := awsUrlRegexp.FindStringSubmatch(parsedUrl.Hostname()); m != nil {
		opts = append(opts, elastic.SetHttpClient(awsHttpClient(m[1], d)), elastic.SetSniff(false))
	}

	return elastic.NewClient(opts...)
}

func awsHttpClient(region string, d *schema.ResourceData) *http.Client {
	creds := awscredentials.NewChainCredentials([]awscredentials.Provider{
		&awscredentials.StaticProvider{
			Value: awscredentials.Value{
				AccessKeyID:     d.Get("aws_access_key").(string),
				SecretAccessKey: d.Get("aws_secret_key").(string),
				SessionToken:    d.Get("aws_token").(string),
			},
		},
		&awscredentials.EnvProvider{},
	})
	signer := awssigv4.NewSigner(creds)
	client, _ := aws_signing_client.New(signer, nil, "es", region)

	return client
}
