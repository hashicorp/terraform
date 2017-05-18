package circonus

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.json.* resource attribute names
	checkJSONAuthMethodAttr   = "auth_method"
	checkJSONAuthPasswordAttr = "auth_password"
	checkJSONAuthUserAttr     = "auth_user"
	checkJSONCAChainAttr      = "ca_chain"
	checkJSONCertFileAttr     = "certificate_file"
	checkJSONCiphersAttr      = "ciphers"
	checkJSONHeadersAttr      = "headers"
	checkJSONKeyFileAttr      = "key_file"
	checkJSONMethodAttr       = "method"
	checkJSONPayloadAttr      = "payload"
	checkJSONPortAttr         = "port"
	checkJSONReadLimitAttr    = "read_limit"
	checkJSONURLAttr          = "url"
	checkJSONVersionAttr      = "version"
)

var checkJSONDescriptions = attrDescrs{
	checkJSONAuthMethodAttr:   "The HTTP Authentication method",
	checkJSONAuthPasswordAttr: "The HTTP Authentication user password",
	checkJSONAuthUserAttr:     "The HTTP Authentication user name",
	checkJSONCAChainAttr:      "A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for TLS checks)",
	checkJSONCertFileAttr:     "A path to a file containing the client certificate that will be presented to the remote server (for TLS-enabled checks)",
	checkJSONCiphersAttr:      "A list of ciphers to be used in the TLS protocol (for HTTPS checks)",
	checkJSONHeadersAttr:      "Map of HTTP Headers to send along with HTTP Requests",
	checkJSONKeyFileAttr:      "A path to a file containing key to be used in conjunction with the cilent certificate (for TLS checks)",
	checkJSONMethodAttr:       "The HTTP method to use",
	checkJSONPayloadAttr:      "The information transferred as the payload of an HTTP request",
	checkJSONPortAttr:         "Specifies the port on which the management interface can be reached",
	checkJSONReadLimitAttr:    "Sets an approximate limit on the data read (0 means no limit)",
	checkJSONURLAttr:          "The URL to use as the target of the check",
	checkJSONVersionAttr:      "Sets the HTTP version for the check to use",
}

var schemaCheckJSON = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      checkJSONConfigChecksum,
	Elem: &schema.Resource{
		Schema: convertToHelperSchema(checkJSONDescriptions, map[schemaAttr]*schema.Schema{
			checkJSONAuthMethodAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkJSONAuthMethodAttr, `^(?:Basic|Digest|Auto)$`),
			},
			checkJSONAuthPasswordAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validateRegexp(checkJSONAuthPasswordAttr, `^.*`),
			},
			checkJSONAuthUserAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkJSONAuthUserAttr, `[^:]+`),
			},
			checkJSONCAChainAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkJSONCAChainAttr, `.+`),
			},
			checkJSONCertFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkJSONCertFileAttr, `.+`),
			},
			checkJSONCiphersAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkJSONCiphersAttr, `.+`),
			},
			checkJSONHeadersAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Elem:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateHTTPHeaders,
			},
			checkJSONKeyFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkJSONKeyFileAttr, `.+`),
			},
			checkJSONMethodAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckJSONMethod,
				ValidateFunc: validateRegexp(checkJSONMethodAttr, `\S+`),
			},
			checkJSONPayloadAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkJSONPayloadAttr, `\S+`),
			},
			checkJSONPortAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Default:  defaultCheckJSONPort,
				Optional: true,
				ValidateFunc: validateFuncs(
					validateIntMin(checkJSONPortAttr, 0),
					validateIntMax(checkJSONPortAttr, 65535),
				),
			},
			checkJSONReadLimitAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: validateFuncs(
					validateIntMin(checkJSONReadLimitAttr, 0),
				),
			},
			checkJSONURLAttr: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validateFuncs(
					validateHTTPURL(checkJSONURLAttr, urlIsAbs),
				),
			},
			checkJSONVersionAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckJSONVersion,
				ValidateFunc: validateStringIn(checkJSONVersionAttr, supportedHTTPVersions),
			},
		}),
	},
}

// checkAPIToStateJSON reads the Config data out of circonusCheck.CheckBundle into
// the statefile.
func checkAPIToStateJSON(c *circonusCheck, d *schema.ResourceData) error {
	jsonConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, s := range c.Config {
		swamp[k] = s
	}

	saveStringConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if s, ok := c.Config[apiKey]; ok && s != "" {
			jsonConfig[string(attrName)] = s
		}

		delete(swamp, apiKey)
	}

	saveIntConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if s, ok := c.Config[apiKey]; ok && s != "0" {
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				log.Printf("[ERROR]: Unable to convert %s to an integer: %v", apiKey, err)
				return
			}
			jsonConfig[string(attrName)] = int(i)
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState(config.AuthMethod, checkJSONAuthMethodAttr)
	saveStringConfigToState(config.AuthPassword, checkJSONAuthPasswordAttr)
	saveStringConfigToState(config.AuthUser, checkJSONAuthUserAttr)
	saveStringConfigToState(config.CAChain, checkJSONCAChainAttr)
	saveStringConfigToState(config.CertFile, checkJSONCertFileAttr)
	saveStringConfigToState(config.Ciphers, checkJSONCiphersAttr)

	headers := make(map[string]interface{}, len(c.Config))
	headerPrefixLen := len(config.HeaderPrefix)
	for k, v := range c.Config {
		if len(k) <= headerPrefixLen {
			continue
		}

		if strings.Compare(string(k[:headerPrefixLen]), string(config.HeaderPrefix)) == 0 {
			key := k[headerPrefixLen:]
			headers[string(key)] = v
		}
		delete(swamp, k)
	}
	jsonConfig[string(checkJSONHeadersAttr)] = headers

	saveStringConfigToState(config.KeyFile, checkJSONKeyFileAttr)
	saveStringConfigToState(config.Method, checkJSONMethodAttr)
	saveStringConfigToState(config.Payload, checkJSONPayloadAttr)
	saveIntConfigToState(config.Port, checkJSONPortAttr)
	saveIntConfigToState(config.ReadLimit, checkJSONReadLimitAttr)
	saveStringConfigToState(config.URL, checkJSONURLAttr)
	saveStringConfigToState(config.HTTPVersion, checkJSONVersionAttr)

	whitelistedConfigKeys := map[config.Key]struct{}{
		config.ReverseSecretKey: struct{}{},
		config.SubmissionURL:    struct{}{},
	}

	for k := range swamp {
		if _, ok := whitelistedConfigKeys[k]; ok {
			delete(c.Config, k)
		}

		if _, ok := whitelistedConfigKeys[k]; !ok {
			return fmt.Errorf("PROVIDER BUG: API Config not empty: %#v", swamp)
		}
	}

	if err := d.Set(checkJSONAttr, schema.NewSet(checkJSONConfigChecksum, []interface{}{jsonConfig})); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkJSONAttr), err)
	}

	return nil
}

// checkJSONConfigChecksum creates a stable hash of the normalized values found
// in a user's Terraform config.
func checkJSONConfigChecksum(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeInt := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(int) != 0 {
			fmt.Fprintf(b, "%x", v.(int))
		}
	}

	writeString := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			fmt.Fprint(b, strings.TrimSpace(v.(string)))
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeString(checkJSONAuthMethodAttr)
	writeString(checkJSONAuthPasswordAttr)
	writeString(checkJSONAuthUserAttr)
	writeString(checkJSONCAChainAttr)
	writeString(checkJSONCertFileAttr)
	writeString(checkJSONCiphersAttr)

	if headersRaw, ok := m[string(checkJSONHeadersAttr)]; ok {
		headerMap := headersRaw.(map[string]interface{})
		headers := make([]string, 0, len(headerMap))
		for k := range headerMap {
			headers = append(headers, k)
		}

		sort.Strings(headers)
		for i := range headers {
			fmt.Fprint(b, headers[i])
			fmt.Fprint(b, headerMap[headers[i]].(string))
		}
	}

	writeString(checkJSONKeyFileAttr)
	writeString(checkJSONMethodAttr)
	writeString(checkJSONPayloadAttr)
	writeInt(checkJSONPortAttr)
	writeInt(checkJSONReadLimitAttr)
	writeString(checkJSONURLAttr)
	writeString(checkJSONVersionAttr)

	s := b.String()
	return hashcode.String(s)
}

func checkConfigToAPIJSON(c *circonusCheck, l interfaceList) error {
	c.Type = string(apiCheckTypeJSON)

	// Iterate over all `json` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		jsonConfig := newInterfaceMap(mapRaw)

		if v, found := jsonConfig[checkJSONAuthMethodAttr]; found {
			c.Config[config.AuthMethod] = v.(string)
		}

		if v, found := jsonConfig[checkJSONAuthPasswordAttr]; found {
			c.Config[config.AuthPassword] = v.(string)
		}

		if v, found := jsonConfig[checkJSONAuthUserAttr]; found {
			c.Config[config.AuthUser] = v.(string)
		}

		if v, found := jsonConfig[checkJSONCAChainAttr]; found {
			c.Config[config.CAChain] = v.(string)
		}

		if v, found := jsonConfig[checkJSONCertFileAttr]; found {
			c.Config[config.CertFile] = v.(string)
		}

		if v, found := jsonConfig[checkJSONCiphersAttr]; found {
			c.Config[config.Ciphers] = v.(string)
		}

		if headers := jsonConfig.CollectMap(checkJSONHeadersAttr); headers != nil {
			for k, v := range headers {
				h := config.HeaderPrefix + config.Key(k)
				c.Config[h] = v
			}
		}

		if v, found := jsonConfig[checkJSONKeyFileAttr]; found {
			c.Config[config.KeyFile] = v.(string)
		}

		if v, found := jsonConfig[checkJSONMethodAttr]; found {
			c.Config[config.Method] = v.(string)
		}

		if v, found := jsonConfig[checkJSONPayloadAttr]; found {
			c.Config[config.Payload] = v.(string)
		}

		if v, found := jsonConfig[checkJSONPortAttr]; found {
			i := v.(int)
			if i != 0 {
				c.Config[config.Port] = fmt.Sprintf("%d", i)
			}
		}

		if v, found := jsonConfig[checkJSONReadLimitAttr]; found {
			i := v.(int)
			if i != 0 {
				c.Config[config.ReadLimit] = fmt.Sprintf("%d", i)
			}
		}

		if v, found := jsonConfig[checkJSONURLAttr]; found {
			c.Config[config.URL] = v.(string)

			u, _ := url.Parse(v.(string))
			hostInfo := strings.SplitN(u.Host, ":", 2)
			if len(c.Target) == 0 {
				c.Target = hostInfo[0]
			}

			if len(hostInfo) > 1 && c.Config[config.Port] == "" {
				c.Config[config.Port] = hostInfo[1]
			}
		}

		if v, found := jsonConfig[checkJSONVersionAttr]; found {
			c.Config[config.HTTPVersion] = v.(string)
		}
	}

	return nil
}
