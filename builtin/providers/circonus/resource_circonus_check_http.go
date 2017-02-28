package circonus

import (
	"bytes"
	"fmt"
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
	// circonus_check.http.* resource attribute names
	checkHTTPAuthMethodAttr   schemaAttr = "auth_method"
	checkHTTPAuthPasswordAttr schemaAttr = "auth_password"
	checkHTTPAuthUserAttr     schemaAttr = "auth_user"
	checkHTTPBodyRegexpAttr   schemaAttr = "body_regexp"
	checkHTTPCAChainAttr      schemaAttr = "ca_chain"
	checkHTTPCertFileAttr     schemaAttr = "certificate_file"
	checkHTTPCiphersAttr      schemaAttr = "ciphers"
	checkHTTPCodeRegexpAttr   schemaAttr = "code"
	checkHTTPExtractAttr      schemaAttr = "extract"
	checkHTTPHeadersAttr      schemaAttr = "headers"
	checkHTTPKeyFileAttr      schemaAttr = "key_file"
	checkHTTPMethodAttr       schemaAttr = "method"
	checkHTTPPayloadAttr      schemaAttr = "payload"
	checkHTTPReadLimitAttr    schemaAttr = "read_limit"
	checkHTTPURLAttr          schemaAttr = "url"
	checkHTTPVersionAttr      schemaAttr = "version"
)

var checkHTTPDescriptions = attrDescrs{
	checkHTTPAuthMethodAttr:   "The HTTP Authentication method",
	checkHTTPAuthPasswordAttr: "The HTTP Authentication user password",
	checkHTTPAuthUserAttr:     "The HTTP Authentication user name",
	checkHTTPBodyRegexpAttr:   `This regular expression is matched against the body of the response. If a match is not found, the check will be marked as "bad.`,
	checkHTTPCAChainAttr:      "A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for TLS checks)",
	checkHTTPCodeRegexpAttr:   `The HTTP code that is expected. If the code received does not match this regular expression, the check is marked as "bad."`,
	checkHTTPCiphersAttr:      "A list of ciphers to be used in the TLS protocol (for HTTPS checks)",
	checkHTTPCertFileAttr:     "A path to a file containing the client certificate that will be presented to the remote server (for TLS-enabled checks)",
	checkHTTPExtractAttr:      "This regular expression is matched against the body of the response globally. The first capturing match is the key and the second capturing match is the value. Each key/value extracted is registered as a metric for the check.",
	checkHTTPHeadersAttr:      "Map of HTTP Headers to send along with HTTP Requests",
	checkHTTPKeyFileAttr:      "A path to a file containing key to be used in conjunction with the cilent certificate (for TLS checks)",
	checkHTTPMethodAttr:       "The HTTP method to use",
	checkHTTPPayloadAttr:      "The information transferred as the payload of an HTTP request",
	checkHTTPReadLimitAttr:    "Sets an approximate limit on the data read (0 means no limit)",
	checkHTTPURLAttr:          "The URL to use as the target of the check",
	checkHTTPVersionAttr:      "Sets the HTTP version for the check to use",
}

var schemaCheckHTTP = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckHTTP,
	Elem: &schema.Resource{
		Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
			checkHTTPAuthMethodAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkHTTPAuthMethodAttr, `^(?:Basic|Digest|Auto)$`),
			},
			checkHTTPAuthPasswordAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validateRegexp(checkHTTPAuthPasswordAttr, `^.*`),
			},
			checkHTTPAuthUserAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkHTTPAuthUserAttr, `[^:]+`),
			},
			checkHTTPBodyRegexpAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkHTTPBodyRegexpAttr, `.+`),
			},
			checkHTTPCAChainAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkHTTPCAChainAttr, `.+`),
			},
			checkHTTPCertFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkHTTPCertFileAttr, `.+`),
			},
			checkHTTPCiphersAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkHTTPCiphersAttr, `.+`),
			},
			checkHTTPCodeRegexpAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckHTTPCodeRegexp,
				ValidateFunc: validateRegexp(checkHTTPCodeRegexpAttr, `.+`),
			},
			checkHTTPExtractAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkHTTPExtractAttr, `.+`),
			},
			checkHTTPHeadersAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Elem:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateHTTPHeaders,
			},
			checkHTTPKeyFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkHTTPKeyFileAttr, `.+`),
			},
			checkHTTPMethodAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckHTTPMethod,
				ValidateFunc: validateRegexp(checkHTTPMethodAttr, `\S+`),
			},
			checkHTTPPayloadAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkHTTPPayloadAttr, `\S+`),
			},
			checkHTTPReadLimitAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: validateFuncs(
					validateIntMin(checkHTTPReadLimitAttr, 0),
				),
			},
			checkHTTPURLAttr: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validateFuncs(
					validateHTTPURL(checkHTTPURLAttr, urlIsAbs),
				),
			},
			checkHTTPVersionAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckHTTPVersion,
				ValidateFunc: validateStringIn(checkHTTPVersionAttr, supportedHTTPVersions),
			},
		}, checkHTTPDescriptions),
	},
}

// checkAPIToStateHTTP reads the Config data out of circonusCheck.CheckBundle into the
// statefile.
func checkAPIToStateHTTP(c *circonusCheck, d *schema.ResourceData) error {
	httpConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, v := range c.Config {
		swamp[k] = v
	}

	saveStringConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			httpConfig[string(attrName)] = v
		}

		delete(swamp, apiKey)
	}

	saveIntConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				panic(fmt.Sprintf("Unable to convert %s to an integer: %v", apiKey, err))
			}

			httpConfig[string(attrName)] = int(i)
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState(config.AuthMethod, checkHTTPAuthMethodAttr)
	saveStringConfigToState(config.AuthPassword, checkHTTPAuthPasswordAttr)
	saveStringConfigToState(config.AuthUser, checkHTTPAuthUserAttr)
	saveStringConfigToState(config.Body, checkHTTPBodyRegexpAttr)
	saveStringConfigToState(config.CAChain, checkHTTPCAChainAttr)
	saveStringConfigToState(config.CertFile, checkHTTPCertFileAttr)
	saveStringConfigToState(config.Ciphers, checkHTTPCiphersAttr)
	saveStringConfigToState(config.Code, checkHTTPCodeRegexpAttr)
	saveStringConfigToState(config.Extract, checkHTTPExtractAttr)

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
	httpConfig[string(checkHTTPHeadersAttr)] = headers

	saveStringConfigToState(config.KeyFile, checkHTTPKeyFileAttr)
	saveStringConfigToState(config.Method, checkHTTPMethodAttr)
	saveStringConfigToState(config.Payload, checkHTTPPayloadAttr)
	saveIntConfigToState(config.ReadLimit, checkHTTPReadLimitAttr)
	saveStringConfigToState(config.URL, checkHTTPURLAttr)
	saveStringConfigToState(config.HTTPVersion, checkHTTPVersionAttr)

	whitelistedConfigKeys := map[config.Key]struct{}{
		config.ReverseSecretKey: struct{}{},
		config.SubmissionURL:    struct{}{},
	}

	for k := range swamp {
		if _, ok := whitelistedConfigKeys[k]; ok {
			delete(c.Config, k)
		}

		if _, ok := whitelistedConfigKeys[k]; !ok {
			panic(fmt.Sprintf("PROVIDER BUG: API Config not empty: %#v", swamp))
		}
	}

	if err := d.Set(checkHTTPAttr, schema.NewSet(hashCheckHTTP, []interface{}{httpConfig})); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkHTTPAttr), err)
	}

	return nil
}

// hashCheckHTTP creates a stable hash of the normalized values
func hashCheckHTTP(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeInt := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok {
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
	writeString(checkHTTPAuthMethodAttr)
	writeString(checkHTTPAuthPasswordAttr)
	writeString(checkHTTPAuthUserAttr)
	writeString(checkHTTPBodyRegexpAttr)
	writeString(checkHTTPCAChainAttr)
	writeString(checkHTTPCertFileAttr)
	writeString(checkHTTPCiphersAttr)
	writeString(checkHTTPCodeRegexpAttr)
	writeString(checkHTTPExtractAttr)

	if headersRaw, ok := m[string(checkHTTPHeadersAttr)]; ok {
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

	writeString(checkHTTPKeyFileAttr)
	writeString(checkHTTPMethodAttr)
	writeString(checkHTTPPayloadAttr)
	writeInt(checkHTTPReadLimitAttr)
	writeString(checkHTTPURLAttr)
	writeString(checkHTTPVersionAttr)

	s := b.String()
	return hashcode.String(s)
}

func checkConfigToAPIHTTP(c *circonusCheck, ctxt *providerContext, l interfaceList) error {
	c.Type = string(apiCheckTypeHTTP)

	// Iterate over all `http` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		httpConfig := newInterfaceMap(mapRaw)
		ar := newMapReader(ctxt, httpConfig)

		if s, ok := ar.GetStringOK(checkHTTPAuthMethodAttr); ok {
			c.Config[config.AuthMethod] = s
		}

		if s, ok := ar.GetStringOK(checkHTTPAuthPasswordAttr); ok {
			c.Config[config.AuthPassword] = s
		}

		if s, ok := ar.GetStringOK(checkHTTPAuthUserAttr); ok {
			c.Config[config.AuthUser] = s
		}

		if s, ok := ar.GetStringOK(checkHTTPBodyRegexpAttr); ok {
			c.Config[config.Body] = s
		}

		if s, ok := ar.GetStringOK(checkHTTPCAChainAttr); ok {
			c.Config[config.CAChain] = s
		}

		if s, ok := ar.GetStringOK(checkHTTPCertFileAttr); ok {
			c.Config[config.CertFile] = s
		}

		if s, ok := ar.GetStringOK(checkHTTPCiphersAttr); ok {
			c.Config[config.Ciphers] = s
		}

		if s, ok := ar.GetStringOK(checkHTTPCodeRegexpAttr); ok {
			c.Config[config.Code] = s
		}

		if s, ok := ar.GetStringOK(checkHTTPExtractAttr); ok {
			c.Config[config.Extract] = s
		}

		if headers := httpConfig.CollectMap(checkHTTPHeadersAttr); headers != nil {
			for k, v := range headers {
				h := config.HeaderPrefix + config.Key(k)
				c.Config[h] = v
			}
		}

		if s, ok := ar.GetStringOK(checkHTTPKeyFileAttr); ok {
			c.Config[config.KeyFile] = s
		}

		if s, ok := ar.GetStringOK(checkHTTPMethodAttr); ok {
			c.Config[config.Method] = s
		}

		if s, ok := ar.GetStringOK(checkHTTPPayloadAttr); ok {
			c.Config[config.Payload] = s
		}

		if i, ok := ar.GetIntOK(checkHTTPReadLimitAttr); ok {
			c.Config[config.ReadLimit] = fmt.Sprintf("%d", i)
		}

		if s, ok := ar.GetStringOK(checkHTTPURLAttr); ok {
			c.Config[config.URL] = s

			u, _ := url.Parse(s)
			hostInfo := strings.SplitN(u.Host, ":", 2)
			if len(c.Target) == 0 {
				c.Target = hostInfo[0]
			}
		}

		if s, ok := ar.GetStringOK(checkHTTPVersionAttr); ok {
			c.Config[config.HTTPVersion] = s
		}
	}

	return nil
}
