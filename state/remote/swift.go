package remote

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/containers"
	"github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	tf_openstack "github.com/hashicorp/terraform/builtin/providers/openstack"
)

const TFSTATE_NAME = "tfstate.tf"

// SwiftClient implements the Client interface for an Openstack Swift server.
type SwiftClient struct {
	client      *gophercloud.ServiceClient
	authurl     string
	cacert      string
	cert        string
	domainid    string
	domainname  string
	insecure    bool
	key         string
	password    string
	path        string
	region      string
	tenantid    string
	tenantname  string
	userid      string
	username    string
	token       string
	archive     bool
	archivepath string
	expireSecs  int
}

func swiftFactory(conf map[string]string) (Client, error) {
	client := &SwiftClient{}

	if err := client.validateConfig(conf); err != nil {
		return nil, err
	}

	return client, nil
}

func (c *SwiftClient) validateConfig(conf map[string]string) (err error) {
	authUrl, ok := conf["auth_url"]
	if !ok {
		authUrl = os.Getenv("OS_AUTH_URL")
		if authUrl == "" {
			return fmt.Errorf("missing 'auth_url' configuration or OS_AUTH_URL environment variable")
		}
	}
	c.authurl = authUrl

	username, ok := conf["user_name"]
	if !ok {
		username = os.Getenv("OS_USERNAME")
	}
	c.username = username

	userID, ok := conf["user_id"]
	if !ok {
		userID = os.Getenv("OS_USER_ID")
	}
	c.userid = userID

	token, ok := conf["token"]
	if !ok {
		token = os.Getenv("OS_AUTH_TOKEN")
	}
	c.token = token

	password, ok := conf["password"]
	if !ok {
		password = os.Getenv("OS_PASSWORD")

	}
	c.password = password
	if password == "" && token == "" {
		return fmt.Errorf("missing either password or token configuration or OS_PASSWORD or OS_AUTH_TOKEN environment variable")
	}

	region, ok := conf["region_name"]
	if !ok {
		region = os.Getenv("OS_REGION_NAME")
	}
	c.region = region

	tenantID, ok := conf["tenant_id"]
	if !ok {
		tenantID = multiEnv([]string{
			"OS_TENANT_ID",
			"OS_PROJECT_ID",
		})
	}
	c.tenantid = tenantID

	tenantName, ok := conf["tenant_name"]
	if !ok {
		tenantName = multiEnv([]string{
			"OS_TENANT_NAME",
			"OS_PROJECT_NAME",
		})
	}
	c.tenantname = tenantName

	domainID, ok := conf["domain_id"]
	if !ok {
		domainID = multiEnv([]string{
			"OS_USER_DOMAIN_ID",
			"OS_PROJECT_DOMAIN_ID",
			"OS_DOMAIN_ID",
		})
	}
	c.domainid = domainID

	domainName, ok := conf["domain_name"]
	if !ok {
		domainName = multiEnv([]string{
			"OS_USER_DOMAIN_NAME",
			"OS_PROJECT_DOMAIN_NAME",
			"OS_DOMAIN_NAME",
			"DEFAULT_DOMAIN",
		})
	}
	c.domainname = domainName

	path, ok := conf["path"]
	if !ok || path == "" {
		return fmt.Errorf("missing 'path' configuration")
	}
	c.path = path

	if archivepath, ok := conf["archive_path"]; ok {
		log.Printf("[DEBUG] Archivepath set, enabling object versioning")
		c.archive = true
		c.archivepath = archivepath
	}

	if expire, ok := conf["expire_after"]; ok {
		log.Printf("[DEBUG] Requested that remote state expires after %s", expire)

		if strings.HasSuffix(expire, "d") {
			log.Printf("[DEBUG] Got a days expire after duration. Converting to hours")
			days, err := strconv.Atoi(expire[:len(expire)-1])
			if err != nil {
				return fmt.Errorf("Error converting expire_after value %s to int: %s", expire, err)
			}

			expire = fmt.Sprintf("%dh", days*24)
			log.Printf("[DEBUG] Expire after %s hours", expire)
		}

		expireDur, err := time.ParseDuration(expire)
		if err != nil {
			log.Printf("[DEBUG] Error parsing duration %s: %s", expire, err)
			return fmt.Errorf("Error parsing expire_after duration '%s': %s", expire, err)
		}
		log.Printf("[DEBUG] Seconds duration = %d", int(expireDur.Seconds()))
		c.expireSecs = int(expireDur.Seconds())
	}

	c.insecure = false
	raw, ok := conf["insecure"]
	if !ok {
		raw = os.Getenv("OS_INSECURE")
	}
	if raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("'insecure' and 'OS_INSECURE' could not be parsed as bool: %s", err)
		}
		c.insecure = v
	}

	cacertFile, ok := conf["cacert_file"]
	if !ok {
		cacertFile = os.Getenv("OS_CACERT")
	}
	c.cacert = cacertFile

	cert, ok := conf["cert"]
	if !ok {
		cert = os.Getenv("OS_CERT")
	}
	c.cert = cert

	key, ok := conf["key"]
	if !ok {
		key = os.Getenv("OS_KEY")
	}
	c.key = key

	ao := gophercloud.AuthOptions{
		IdentityEndpoint: c.authurl,
		UserID:           c.userid,
		Username:         c.username,
		TenantID:         c.tenantid,
		TenantName:       c.tenantname,
		Password:         c.password,
		TokenID:          c.token,
		DomainID:         c.domainid,
		DomainName:       c.domainname,
	}

	provider, err := openstack.NewClient(ao.IdentityEndpoint)
	if err != nil {
		return err
	}

	config := &tls.Config{}

	if c.cacert != "" {
		caCert, err := ioutil.ReadFile(c.cacert)
		if err != nil {
			return err
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		config.RootCAs = caCertPool
	}

	if c.insecure {
		log.Printf("[DEBUG] Insecure mode set")
		config.InsecureSkipVerify = true
	}

	if c.cert != "" && c.key != "" {
		cert, err := tls.LoadX509KeyPair(c.cert, c.key)
		if err != nil {
			return err
		}

		config.Certificates = []tls.Certificate{cert}
		config.BuildNameToCertificate()
	}

	// if OS_DEBUG is set, log the requests and responses
	var osDebug bool
	if os.Getenv("OS_DEBUG") != "" {
		osDebug = true
	}

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment, TLSClientConfig: config}
	provider.HTTPClient = http.Client{
		Transport: &tf_openstack.LogRoundTripper{
			Rt:      transport,
			OsDebug: osDebug,
		},
	}

	err = openstack.Authenticate(provider, ao)
	if err != nil {
		return err
	}

	c.client, err = openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{
		Region: c.region,
	})

	return err
}

func (c *SwiftClient) Get() (*Payload, error) {
	result := objects.Download(c.client, c.path, TFSTATE_NAME, nil)

	// Extract any errors from result
	_, err := result.Extract()

	// 404 response is to be expected if the object doesn't already exist!
	if _, ok := err.(gophercloud.ErrDefault404); ok {
		log.Printf("[DEBUG] Container doesn't exist to download.")
		return nil, nil
	}

	bytes, err := result.ExtractContent()
	if err != nil {
		return nil, err
	}

	hash := md5.Sum(bytes)
	payload := &Payload{
		Data: bytes,
		MD5:  hash[:md5.Size],
	}

	return payload, nil
}

func (c *SwiftClient) Put(data []byte) error {
	if err := c.ensureContainerExists(); err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating object %s at path %s", TFSTATE_NAME, c.path)
	reader := bytes.NewReader(data)
	createOpts := objects.CreateOpts{
		Content: reader,
	}

	if c.expireSecs != 0 {
		log.Printf("[DEBUG] ExpireSecs = %d", c.expireSecs)
		createOpts.DeleteAfter = c.expireSecs
	}

	result := objects.Create(c.client, c.path, TFSTATE_NAME, createOpts)

	return result.Err
}

func (c *SwiftClient) Delete() error {
	result := objects.Delete(c.client, c.path, TFSTATE_NAME, nil)
	return result.Err
}

func (c *SwiftClient) ensureContainerExists() error {
	containerOpts := &containers.CreateOpts{}

	if c.archive {
		log.Printf("[DEBUG] Creating container %s", c.archivepath)
		result := containers.Create(c.client, c.archivepath, nil)
		if result.Err != nil {
			log.Printf("[DEBUG] Error creating container %s: %s", c.archivepath, result.Err)
			return result.Err
		}

		log.Printf("[DEBUG] Enabling Versioning on container %s", c.path)
		containerOpts.VersionsLocation = c.archivepath
	}

	log.Printf("[DEBUG] Creating container %s", c.path)
	result := containers.Create(c.client, c.path, containerOpts)
	if result.Err != nil {
		return result.Err
	}

	return nil
}

func multiEnv(ks []string) string {
	for _, k := range ks {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}
