package circonus

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.consul.* resource attribute names
	checkConsulACLTokenAttr             = "acl_token"
	checkConsulAllowStaleAttr           = "allow_stale"
	checkConsulCAChainAttr              = "ca_chain"
	checkConsulCertFileAttr             = "certificate_file"
	checkConsulCheckNameBlacklistAttr   = "check_blacklist"
	checkConsulCiphersAttr              = "ciphers"
	checkConsulDatacenterAttr           = "dc"
	checkConsulHTTPAddrAttr             = "http_addr"
	checkConsulHeadersAttr              = "headers"
	checkConsulKeyFileAttr              = "key_file"
	checkConsulNodeAttr                 = "node"
	checkConsulNodeBlacklistAttr        = "node_blacklist"
	checkConsulServiceAttr              = "service"
	checkConsulServiceNameBlacklistAttr = "service_blacklist"
	checkConsulStateAttr                = "state"
)

var checkConsulDescriptions = attrDescrs{
	checkConsulACLTokenAttr:             "A Consul ACL token",
	checkConsulAllowStaleAttr:           "Allow Consul to read from a non-leader system",
	checkConsulCAChainAttr:              "A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for TLS checks)",
	checkConsulCertFileAttr:             "A path to a file containing the client certificate that will be presented to the remote server (for TLS-enabled checks)",
	checkConsulCheckNameBlacklistAttr:   "A blacklist of check names to exclude from metric results",
	checkConsulCiphersAttr:              "A list of ciphers to be used in the TLS protocol (for HTTPS checks)",
	checkConsulDatacenterAttr:           "The Consul datacenter to extract health information from",
	checkConsulHeadersAttr:              "Map of HTTP Headers to send along with HTTP Requests",
	checkConsulHTTPAddrAttr:             "The HTTP Address of a Consul agent to query",
	checkConsulKeyFileAttr:              "A path to a file containing key to be used in conjunction with the cilent certificate (for TLS checks)",
	checkConsulNodeAttr:                 "Node Name or NodeID of a Consul agent",
	checkConsulNodeBlacklistAttr:        "A blacklist of node names or IDs to exclude from metric results",
	checkConsulServiceAttr:              "Name of the Consul service to check",
	checkConsulServiceNameBlacklistAttr: "A blacklist of service names to exclude from metric results",
	checkConsulStateAttr:                "Check for Consul services in this particular state",
}

var consulHealthCheckRE = regexp.MustCompile(fmt.Sprintf(`^%s/(%s|%s|%s)/(.+)`, checkConsulV1Prefix, checkConsulV1NodePrefix, checkConsulV1ServicePrefix, checkConsulV1StatePrefix))

var schemaCheckConsul = &schema.Schema{
	Type:     schema.TypeList,
	Optional: true,
	MaxItems: 1,
	Elem: &schema.Resource{
		Schema: convertToHelperSchema(checkConsulDescriptions, map[schemaAttr]*schema.Schema{
			checkConsulACLTokenAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkConsulACLTokenAttr, `^[a-zA-Z0-9\-]+$`),
			},
			checkConsulAllowStaleAttr: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			checkConsulCAChainAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkConsulCAChainAttr, `.+`),
			},
			checkConsulCertFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkConsulCertFileAttr, `.+`),
			},
			checkConsulCheckNameBlacklistAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateRegexp(checkConsulCheckNameBlacklistAttr, `^[A-Za-z0-9_-]+$`),
				},
			},
			checkConsulCiphersAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkConsulCiphersAttr, `.+`),
			},
			checkConsulDatacenterAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkConsulCertFileAttr, `^[a-zA-Z0-9]+$`),
			},
			checkConsulHTTPAddrAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckConsulHTTPAddr,
				ValidateFunc: validateHTTPURL(checkConsulHTTPAddrAttr, urlIsAbs|urlWithoutPath),
			},
			checkConsulHeadersAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Elem:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateHTTPHeaders,
			},
			checkConsulKeyFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkConsulKeyFileAttr, `.+`),
			},
			checkConsulNodeAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkConsulNodeAttr, `^[a-zA-Z0-9_\-]+$`),
				ConflictsWith: []string{
					checkConsulAttr + "." + checkConsulServiceAttr,
					checkConsulAttr + "." + checkConsulStateAttr,
				},
			},
			checkConsulNodeBlacklistAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateRegexp(checkConsulNodeBlacklistAttr, `^[A-Za-z0-9_-]+$`),
				},
			},
			checkConsulServiceAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkConsulServiceAttr, `^[a-zA-Z0-9_\-]+$`),
				ConflictsWith: []string{
					checkConsulAttr + "." + checkConsulNodeAttr,
					checkConsulAttr + "." + checkConsulStateAttr,
				},
			},
			checkConsulServiceNameBlacklistAttr: &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validateRegexp(checkConsulServiceNameBlacklistAttr, `^[A-Za-z0-9_-]+$`),
				},
			},
			checkConsulStateAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkConsulStateAttr, `^(any|passing|warning|critical)$`),
				ConflictsWith: []string{
					checkConsulAttr + "." + checkConsulNodeAttr,
					checkConsulAttr + "." + checkConsulServiceAttr,
				},
			},
		}),
	},
}

// checkAPIToStateConsul reads the Config data out of circonusCheck.CheckBundle into
// the statefile.
func checkAPIToStateConsul(c *circonusCheck, d *schema.ResourceData) error {
	consulConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, s := range c.Config {
		swamp[k] = s
	}

	saveStringConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if s, ok := c.Config[apiKey]; ok && s != "" {
			consulConfig[string(attrName)] = s
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState(config.CAChain, checkConsulCAChainAttr)
	saveStringConfigToState(config.CertFile, checkConsulCertFileAttr)
	saveStringConfigToState(config.Ciphers, checkConsulCiphersAttr)

	// httpAddrURL is used to compose the http_addr value using multiple c.Config
	// values.
	var httpAddrURL url.URL

	headers := make(map[string]interface{}, len(c.Config)+1) // +1 is for the ACLToken
	headerPrefixLen := len(config.HeaderPrefix)

	// Explicitly handle several config parameters in sequence: URL, then port,
	// then everything else.
	if v, found := c.Config[config.URL]; found {
		u, err := url.Parse(v)
		if err != nil {
			return errwrap.Wrapf(fmt.Sprintf("unable to parse %q from config: {{err}}", config.URL), err)
		}

		queryArgs := u.Query()
		if vals, found := queryArgs[apiConsulStaleAttr]; found && len(vals) > 0 {
			consulConfig[string(checkConsulAllowStaleAttr)] = true
		}

		if dc := queryArgs.Get(apiConsulDatacenterAttr); dc != "" {
			consulConfig[string(checkConsulDatacenterAttr)] = dc
		}

		httpAddrURL.Host = u.Host
		httpAddrURL.Scheme = u.Scheme

		md := consulHealthCheckRE.FindStringSubmatch(u.EscapedPath())
		if md == nil {
			return fmt.Errorf("config %q failed to match the health regexp", config.URL)
		}

		checkMode := md[1]
		checkArg := md[2]
		switch checkMode {
		case checkConsulV1NodePrefix:
			consulConfig[string(checkConsulNodeAttr)] = checkArg
		case checkConsulV1ServicePrefix:
			consulConfig[string(checkConsulServiceAttr)] = checkArg
		case checkConsulV1StatePrefix:
			consulConfig[string(checkConsulStateAttr)] = checkArg
		default:
			return fmt.Errorf("PROVIDER BUG: unsupported check mode %q from %q", checkMode, u.EscapedPath())
		}

		delete(swamp, config.URL)
	}

	if v, found := c.Config[config.Port]; found {
		hostInfo := strings.SplitN(httpAddrURL.Host, ":", 2)
		switch {
		case len(hostInfo) == 1 && v != defaultCheckConsulPort, len(hostInfo) > 1:
			httpAddrURL.Host = net.JoinHostPort(hostInfo[0], v)
		}

		delete(swamp, config.Port)
	}

	if v, found := c.Config[apiConsulCheckBlacklist]; found {
		consulConfig[checkConsulCheckNameBlacklistAttr] = strings.Split(v, ",")
	}

	if v, found := c.Config[apiConsulNodeBlacklist]; found {
		consulConfig[checkConsulNodeBlacklistAttr] = strings.Split(v, ",")
	}

	if v, found := c.Config[apiConsulServiceBlacklist]; found {
		consulConfig[checkConsulServiceNameBlacklistAttr] = strings.Split(v, ",")
	}

	// NOTE(sean@): headers attribute processed last.  See below.

	consulConfig[string(checkConsulHTTPAddrAttr)] = httpAddrURL.String()

	saveStringConfigToState(config.KeyFile, checkConsulKeyFileAttr)

	// Process the headers last in order to provide an escape hatch capible of
	// overriding any other derived value above.
	for k, v := range c.Config {
		if len(k) <= headerPrefixLen {
			continue
		}

		// Handle all of the prefix variable headers, like `header_`
		if strings.Compare(string(k[:headerPrefixLen]), string(config.HeaderPrefix)) == 0 {
			key := k[headerPrefixLen:]
			switch key {
			case checkConsulTokenHeader:
				consulConfig[checkConsulACLTokenAttr] = v
			default:
				headers[string(key)] = v
			}
		}

		delete(swamp, k)
	}
	consulConfig[string(checkConsulHeadersAttr)] = headers

	whitelistedConfigKeys := map[config.Key]struct{}{
		config.Port:             struct{}{},
		config.ReverseSecretKey: struct{}{},
		config.SubmissionURL:    struct{}{},
		config.URL:              struct{}{},
	}

	for k := range swamp {
		if _, ok := whitelistedConfigKeys[k]; ok {
			delete(c.Config, k)
		}

		if _, ok := whitelistedConfigKeys[k]; !ok {
			return fmt.Errorf("PROVIDER BUG: API Config not empty: %#v", swamp)
		}
	}

	if err := d.Set(checkConsulAttr, []interface{}{consulConfig}); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkConsulAttr), err)
	}

	return nil
}

func checkConfigToAPIConsul(c *circonusCheck, l interfaceList) error {
	c.Type = string(apiCheckTypeConsul)

	// Iterate over all `consul` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		consulConfig := newInterfaceMap(mapRaw)
		if v, found := consulConfig[checkConsulCAChainAttr]; found {
			c.Config[config.CAChain] = v.(string)
		}

		if v, found := consulConfig[checkConsulCertFileAttr]; found {
			c.Config[config.CertFile] = v.(string)
		}

		if v, found := consulConfig[checkConsulCheckNameBlacklistAttr]; found {
			listRaw := v.([]interface{})
			checks := make([]string, 0, len(listRaw))
			for _, v := range listRaw {
				checks = append(checks, v.(string))
			}
			c.Config[apiConsulCheckBlacklist] = strings.Join(checks, ",")
		}

		if v, found := consulConfig[checkConsulCiphersAttr]; found {
			c.Config[config.Ciphers] = v.(string)
		}

		if headers := consulConfig.CollectMap(checkConsulHeadersAttr); headers != nil {
			for k, v := range headers {
				h := config.HeaderPrefix + config.Key(k)
				c.Config[h] = v
			}
		}

		if v, found := consulConfig[checkConsulKeyFileAttr]; found {
			c.Config[config.KeyFile] = v.(string)
		}

		{
			// Extract all of the input attributes necessary to construct the
			// Consul agent's URL.

			httpAddr := consulConfig[checkConsulHTTPAddrAttr].(string)
			checkURL, err := url.Parse(httpAddr)
			if err != nil {
				return errwrap.Wrapf(fmt.Sprintf("Unable to parse %s's attribute %q: {{err}}", checkConsulAttr, httpAddr), err)
			}

			hostInfo := strings.SplitN(checkURL.Host, ":", 2)
			if len(c.Target) == 0 {
				c.Target = hostInfo[0]
			}

			if len(hostInfo) > 1 {
				c.Config[config.Port] = hostInfo[1]
			}

			if v, found := consulConfig[checkConsulNodeAttr]; found && v.(string) != "" {
				checkURL.Path = strings.Join([]string{checkConsulV1Prefix, checkConsulV1NodePrefix, v.(string)}, "/")
			}

			if v, found := consulConfig[checkConsulServiceAttr]; found && v.(string) != "" {
				checkURL.Path = strings.Join([]string{checkConsulV1Prefix, checkConsulV1ServicePrefix, v.(string)}, "/")
			}

			if v, found := consulConfig[checkConsulStateAttr]; found && v.(string) != "" {
				checkURL.Path = strings.Join([]string{checkConsulV1Prefix, checkConsulV1StatePrefix, v.(string)}, "/")
			}

			q := checkURL.Query()

			if v, found := consulConfig[checkConsulAllowStaleAttr]; found && v.(bool) {
				q.Set(apiConsulStaleAttr, "")
			}

			if v, found := consulConfig[checkConsulDatacenterAttr]; found && v.(string) != "" {
				q.Set(apiConsulDatacenterAttr, v.(string))
			}

			checkURL.RawQuery = q.Encode()

			c.Config[config.URL] = checkURL.String()
		}

		if v, found := consulConfig[checkConsulNodeBlacklistAttr]; found {
			listRaw := v.([]interface{})
			checks := make([]string, 0, len(listRaw))
			for _, v := range listRaw {
				checks = append(checks, v.(string))
			}
			c.Config[apiConsulNodeBlacklist] = strings.Join(checks, ",")
		}

		if v, found := consulConfig[checkConsulServiceNameBlacklistAttr]; found {
			listRaw := v.([]interface{})
			checks := make([]string, 0, len(listRaw))
			for _, v := range listRaw {
				checks = append(checks, v.(string))
			}
			c.Config[apiConsulServiceBlacklist] = strings.Join(checks, ",")
		}
	}

	return nil
}
