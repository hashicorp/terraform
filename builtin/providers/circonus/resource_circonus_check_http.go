package circonus

import (
	"bytes"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.http.* resource attribute names
	_CheckHTTPAuthMethodAttr   _SchemaAttr = "auth_method"
	_CheckHTTPAuthPasswordAttr _SchemaAttr = "auth_password"
	_CheckHTTPAuthUserAttr     _SchemaAttr = "auth_user"
	_CheckHTTPBodyRegexpAttr   _SchemaAttr = "body_regexp"
	_CheckHTTPCAChainAttr      _SchemaAttr = "ca_chain"
	_CheckHTTPCertFileAttr     _SchemaAttr = "certificate_file"
	_CheckHTTPCiphersAttr      _SchemaAttr = "ciphers"
	_CheckHTTPCodeRegexpAttr   _SchemaAttr = "code"
	_CheckHTTPExtractAttr      _SchemaAttr = "extract"
	_CheckHTTPHeadersAttr      _SchemaAttr = "headers"
	_CheckHTTPKeyFileAttr      _SchemaAttr = "key_file"
	_CheckHTTPMethodAttr       _SchemaAttr = "method"
	_CheckHTTPPayloadAttr      _SchemaAttr = "payload"
	_CheckHTTPReadLimitAttr    _SchemaAttr = "read_limit"
	_CheckHTTPURLAttr          _SchemaAttr = "url"
	_CheckHTTPVersionAttr      _SchemaAttr = "version"
)

var _CheckHTTPDescriptions = _AttrDescrs{
	_CheckHTTPAuthMethodAttr:   "The HTTP Authentication method",
	_CheckHTTPAuthPasswordAttr: "The HTTP Authentication user password",
	_CheckHTTPAuthUserAttr:     "The HTTP Authentication user name",
	_CheckHTTPBodyRegexpAttr:   `This regular expression is matched against the body of the response. If a match is not found, the check will be marked as "bad.`,
	_CheckHTTPCAChainAttr:      "A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for TLS checks)",
	_CheckHTTPCodeRegexpAttr:   `The HTTP code that is expected. If the code received does not match this regular expression, the check is marked as "bad."`,
	_CheckHTTPCiphersAttr:      "A list of ciphers to be used in the TLS protocol (for HTTPS checks)",
	_CheckHTTPCertFileAttr:     "A path to a file containing the client certificate that will be presented to the remote server (for TLS-enabled checks)",
	_CheckHTTPExtractAttr:      "This regular expression is matched against the body of the response globally. The first capturing match is the key and the second capturing match is the value. Each key/value extracted is registered as a metric for the check.",
	_CheckHTTPHeadersAttr:      "Map of HTTP Headers to send along with HTTP Requests",
	_CheckHTTPKeyFileAttr:      "A path to a file containing key to be used in conjunction with the cilent certificate (for TLS checks)",
	_CheckHTTPMethodAttr:       "The HTTP method to use",
	_CheckHTTPPayloadAttr:      "The information transferred as the payload of an HTTP request",
	_CheckHTTPReadLimitAttr:    "Sets an approximate limit on the data read (0 means no limit)",
	_CheckHTTPURLAttr:          "The URL to use as the target of the check",
	_CheckHTTPVersionAttr:      "Sets the HTTP version for the check to use",
}

var _SchemaCheckHTTP = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckHTTP,
	Elem: &schema.Resource{
		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckHTTPAuthMethodAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPAuthMethodAttr, `^(?:Basic|Digest|Auto)$`),
			},
			_CheckHTTPAuthPasswordAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPAuthPasswordAttr, `^.*`),
			},
			_CheckHTTPAuthUserAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPAuthUserAttr, `[^:]+`),
			},
			_CheckHTTPBodyRegexpAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPBodyRegexpAttr, `.+`),
			},
			_CheckHTTPCAChainAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPCAChainAttr, `.+`),
			},
			_CheckHTTPCertFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPCertFileAttr, `.+`),
			},
			_CheckHTTPCiphersAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPCiphersAttr, `.+`),
			},
			_CheckHTTPCodeRegexpAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckHTTPCodeRegexp,
				ValidateFunc: _ValidateRegexp(_CheckHTTPCodeRegexpAttr, `.+`),
			},
			_CheckHTTPExtractAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPExtractAttr, `.+`),
			},
			_CheckHTTPHeadersAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Elem:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateHTTPHeaders,
			},
			_CheckHTTPKeyFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPKeyFileAttr, `.+`),
			},
			_CheckHTTPMethodAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckHTTPMethod,
				ValidateFunc: _ValidateRegexp(_CheckHTTPMethodAttr, `\S+`),
			},
			_CheckHTTPPayloadAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckHTTPPayloadAttr, `\S+`),
			},
			_CheckHTTPReadLimitAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: _ValidateFuncs(
					_ValidateIntMin(_CheckHTTPReadLimitAttr, 0),
				),
			},
			_CheckHTTPURLAttr: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: _ValidateFuncs(
					_ValidateHTTPURL(_CheckHTTPURLAttr, _URLIsAbs),
				),
			},
			_CheckHTTPVersionAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckHTTPVersion,
				ValidateFunc: _ValidateStringIn(_CheckHTTPVersionAttr, _SupportedHTTPVersions),
			},
		}, _CheckHTTPDescriptions),
	},
}

// _CheckAPIToStateHTTP reads the Config data out of _Check.CheckBundle into the
// statefile.
func _CheckAPIToStateHTTP(c *_Check, d *schema.ResourceData) error {
	httpConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, v := range c.Config {
		swamp[k] = v
	}

	saveStringConfigToState := func(apiKey config.Key, attrName _SchemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			httpConfig[string(attrName)] = v
		}

		delete(swamp, apiKey)
	}

	saveIntConfigToState := func(apiKey config.Key, attrName _SchemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				panic(fmt.Sprintf("Unable to convert %s to an integer: %v", apiKey, err))
			}

			httpConfig[string(attrName)] = int(i)
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState(config.AuthMethod, _CheckHTTPAuthMethodAttr)
	saveStringConfigToState(config.AuthPassword, _CheckHTTPAuthPasswordAttr)
	saveStringConfigToState(config.AuthUser, _CheckHTTPAuthUserAttr)
	saveStringConfigToState(config.Body, _CheckHTTPBodyRegexpAttr)
	saveStringConfigToState(config.CAChain, _CheckHTTPCAChainAttr)
	saveStringConfigToState(config.CertFile, _CheckHTTPCertFileAttr)
	saveStringConfigToState(config.Ciphers, _CheckHTTPCiphersAttr)
	saveStringConfigToState(config.Code, _CheckHTTPCodeRegexpAttr)
	saveStringConfigToState(config.Extract, _CheckHTTPExtractAttr)

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
	httpConfig[string(_CheckHTTPHeadersAttr)] = headers

	saveStringConfigToState(config.KeyFile, _CheckHTTPKeyFileAttr)
	saveStringConfigToState(config.Method, _CheckHTTPMethodAttr)
	saveStringConfigToState(config.Payload, _CheckHTTPPayloadAttr)
	saveIntConfigToState(config.ReadLimit, _CheckHTTPReadLimitAttr)
	saveStringConfigToState(config.URL, _CheckHTTPURLAttr)
	saveStringConfigToState(config.HTTPVersion, _CheckHTTPVersionAttr)

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

	_StateSet(d, _CheckHTTPAttr, schema.NewSet(hashCheckHTTP, []interface{}{httpConfig}))

	return nil
}

// hashCheckHTTP creates a stable hash of the normalized values
func hashCheckHTTP(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeInt := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok {
			fmt.Fprintf(b, "%x", v.(int))
		}
	}

	writeString := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok && v.(string) != "" {
			fmt.Fprint(b, strings.TrimSpace(v.(string)))
		}
	}

	// Order writes to the buffer using lexically sorted list for easy visual
	// reconciliation with other lists.
	writeString(_CheckHTTPAuthMethodAttr)
	writeString(_CheckHTTPAuthPasswordAttr)
	writeString(_CheckHTTPAuthUserAttr)
	writeString(_CheckHTTPBodyRegexpAttr)
	writeString(_CheckHTTPCAChainAttr)
	writeString(_CheckHTTPCertFileAttr)
	writeString(_CheckHTTPCiphersAttr)
	writeString(_CheckHTTPCodeRegexpAttr)
	writeString(_CheckHTTPExtractAttr)

	if headersRaw, ok := m[string(_CheckHTTPHeadersAttr)]; ok {
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

	writeString(_CheckHTTPKeyFileAttr)
	writeString(_CheckHTTPMethodAttr)
	writeString(_CheckHTTPPayloadAttr)
	writeInt(_CheckHTTPReadLimitAttr)
	writeString(_CheckHTTPURLAttr)
	writeString(_CheckHTTPVersionAttr)

	s := b.String()
	return hashcode.String(s)
}

func _CheckConfigToAPIHTTP(c *_Check, ctxt *_ProviderContext, l _InterfaceList) error {
	c.Type = string(_APICheckTypeHTTP)

	// Iterate over all `http` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		httpConfig := _NewInterfaceMap(mapRaw)
		ar := _NewMapReader(ctxt, httpConfig)

		if s, ok := ar.GetStringOK(_CheckHTTPAuthMethodAttr); ok {
			c.Config[config.AuthMethod] = s
		}

		if s, ok := ar.GetStringOK(_CheckHTTPAuthPasswordAttr); ok {
			c.Config[config.AuthPassword] = s
		}

		if s, ok := ar.GetStringOK(_CheckHTTPAuthUserAttr); ok {
			c.Config[config.AuthUser] = s
		}

		if s, ok := ar.GetStringOK(_CheckHTTPBodyRegexpAttr); ok {
			c.Config[config.Body] = s
		}

		if s, ok := ar.GetStringOK(_CheckHTTPCAChainAttr); ok {
			c.Config[config.CAChain] = s
		}

		if s, ok := ar.GetStringOK(_CheckHTTPCertFileAttr); ok {
			c.Config[config.CertFile] = s
		}

		if s, ok := ar.GetStringOK(_CheckHTTPCiphersAttr); ok {
			c.Config[config.Ciphers] = s
		}

		if s, ok := ar.GetStringOK(_CheckHTTPCodeRegexpAttr); ok {
			c.Config[config.Code] = s
		}

		if s, ok := ar.GetStringOK(_CheckHTTPExtractAttr); ok {
			c.Config[config.Extract] = s
		}

		if headers := httpConfig.CollectMap(_CheckHTTPHeadersAttr); headers != nil {
			for k, v := range headers {
				h := config.HeaderPrefix + config.Key(k)
				c.Config[h] = v
			}
		}

		if s, ok := ar.GetStringOK(_CheckHTTPKeyFileAttr); ok {
			c.Config[config.KeyFile] = s
		}

		if s, ok := ar.GetStringOK(_CheckHTTPMethodAttr); ok {
			c.Config[config.Method] = s
		}

		if s, ok := ar.GetStringOK(_CheckHTTPPayloadAttr); ok {
			c.Config[config.Payload] = s
		}

		if i, ok := ar.GetIntOK(_CheckHTTPReadLimitAttr); ok {
			c.Config[config.ReadLimit] = fmt.Sprintf("%d", i)
		}

		if s, ok := ar.GetStringOK(_CheckHTTPURLAttr); ok {
			c.Config[config.URL] = s

			u, _ := url.Parse(s)
			hostInfo := strings.SplitN(u.Host, ":", 2)
			if len(c.Target) == 0 {
				c.Target = hostInfo[0]
			}
		}

		if s, ok := ar.GetStringOK(_CheckHTTPVersionAttr); ok {
			c.Config[config.HTTPVersion] = s
		}
	}

	return nil
}
