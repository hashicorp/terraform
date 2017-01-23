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
	// circonus_check.json.* resource attribute names
	_CheckJSONAuthMethodAttr   _SchemaAttr = "auth_method"
	_CheckJSONAuthPasswordAttr _SchemaAttr = "auth_password"
	_CheckJSONAuthUserAttr     _SchemaAttr = "auth_user"
	_CheckJSONCAChainAttr      _SchemaAttr = "ca_chain"
	_CheckJSONCertFileAttr     _SchemaAttr = "certificate_file"
	_CheckJSONCiphersAttr      _SchemaAttr = "ciphers"
	_CheckJSONHeadersAttr      _SchemaAttr = "headers"
	_CheckJSONKeyFileAttr      _SchemaAttr = "key_file"
	_CheckJSONMethodAttr       _SchemaAttr = "method"
	_CheckJSONPayloadAttr      _SchemaAttr = "payload"
	_CheckJSONPortAttr         _SchemaAttr = "port"
	_CheckJSONReadLimitAttr    _SchemaAttr = "read_limit"
	_CheckJSONURLAttr          _SchemaAttr = "url"
	_CheckJSONVersionAttr      _SchemaAttr = "version"
)

var _CheckJSONDescriptions = _AttrDescrs{
	_CheckJSONAuthMethodAttr:   "The HTTP Authentication method",
	_CheckJSONAuthPasswordAttr: "The HTTP Authentication user password",
	_CheckJSONAuthUserAttr:     "The HTTP Authentication user name",
	_CheckJSONCAChainAttr:      "A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for TLS checks)",
	_CheckJSONCertFileAttr:     "A path to a file containing the client certificate that will be presented to the remote server (for TLS-enabled checks)",
	_CheckJSONCiphersAttr:      "A list of ciphers to be used in the TLS protocol (for HTTPS checks)",
	_CheckJSONHeadersAttr:      "Map of HTTP Headers to send along with HTTP Requests",
	_CheckJSONKeyFileAttr:      "A path to a file containing key to be used in conjunction with the cilent certificate (for TLS checks)",
	_CheckJSONMethodAttr:       "The HTTP method to use",
	_CheckJSONPayloadAttr:      "The information transferred as the payload of an HTTP request",
	_CheckJSONPortAttr:         "Specifies the port on which the management interface can be reached",
	_CheckJSONReadLimitAttr:    "Sets an approximate limit on the data read (0 means no limit)",
	_CheckJSONURLAttr:          "The URL to use as the target of the check",
	_CheckJSONVersionAttr:      "Sets the HTTP version for the check to use",
}

var _SchemaCheckJSON = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      _CheckJSONConfigChecksum,
	Elem: &schema.Resource{
		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckJSONAuthMethodAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckJSONAuthMethodAttr, `^(?:Basic|Digest|Auto)$`),
			},
			_CheckJSONAuthPasswordAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: _ValidateRegexp(_CheckJSONAuthPasswordAttr, `^.*`),
			},
			_CheckJSONAuthUserAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckJSONAuthUserAttr, `[^:]+`),
			},
			_CheckJSONCAChainAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckJSONCAChainAttr, `.+`),
			},
			_CheckJSONCertFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckJSONCertFileAttr, `.+`),
			},
			_CheckJSONCiphersAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckJSONCiphersAttr, `.+`),
			},
			_CheckJSONHeadersAttr: &schema.Schema{
				Type:         schema.TypeMap,
				Elem:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateHTTPHeaders,
			},
			_CheckJSONKeyFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckJSONKeyFileAttr, `.+`),
			},
			_CheckJSONMethodAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckJSONMethod,
				ValidateFunc: _ValidateRegexp(_CheckJSONMethodAttr, `\S+`),
			},
			_CheckJSONPayloadAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckJSONPayloadAttr, `\S+`),
			},
			_CheckJSONPortAttr: &schema.Schema{
				Type:     schema.TypeString, // NOTE(sean@): Why isn't this an Int on Circonus's side?  Are they doing an /etc/services lookup?  TODO: convert this to a TypeInt and force users in TF to do a map lookup.
				Default:  defaultCheckJSONPort,
				Optional: true,
			},
			_CheckJSONReadLimitAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: _ValidateFuncs(
					_ValidateIntMin(_CheckJSONReadLimitAttr, 0),
				),
			},
			_CheckJSONURLAttr: &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: _ValidateFuncs(
					_ValidateHTTPURL(_CheckJSONURLAttr, _URLIsAbs),
				),
			},
			_CheckJSONVersionAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      defaultCheckJSONVersion,
				ValidateFunc: _ValidateStringIn(_CheckJSONVersionAttr, _SupportedHTTPVersions),
			},
		}, _CheckJSONDescriptions),
	},
}

// _CheckAPIToStateJSON reads the Config data out of _Check.CheckBundle into
// the statefile.
func _CheckAPIToStateJSON(c *_Check, d *schema.ResourceData) error {
	jsonConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, v := range c.Config {
		swamp[k] = v
	}

	saveStringConfigToState := func(apiKey config.Key, attrName _SchemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			jsonConfig[string(attrName)] = v
		}

		delete(swamp, apiKey)
	}

	saveIntConfigToState := func(apiKey config.Key, attrName _SchemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				panic(fmt.Sprintf("Unable to convert %s to an integer: %v", err))
				return
			}
			jsonConfig[string(attrName)] = int(i)
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState(config.AuthMethod, _CheckJSONAuthMethodAttr)
	saveStringConfigToState(config.AuthPassword, _CheckJSONAuthPasswordAttr)
	saveStringConfigToState(config.AuthUser, _CheckJSONAuthUserAttr)
	saveStringConfigToState(config.CAChain, _CheckJSONCAChainAttr)
	saveStringConfigToState(config.CertFile, _CheckJSONCertFileAttr)
	saveStringConfigToState(config.Ciphers, _CheckJSONCiphersAttr)

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
	jsonConfig[string(_CheckJSONHeadersAttr)] = headers

	saveStringConfigToState(config.KeyFile, _CheckJSONKeyFileAttr)
	saveStringConfigToState(config.Method, _CheckJSONMethodAttr)
	saveStringConfigToState(config.Payload, _CheckJSONPayloadAttr)
	saveStringConfigToState(config.Port, _CheckJSONPortAttr)
	saveIntConfigToState(config.ReadLimit, _CheckJSONReadLimitAttr)
	saveStringConfigToState(config.URL, _CheckJSONURLAttr)
	saveStringConfigToState(config.HTTPVersion, _CheckJSONVersionAttr)

	whitelistedConfigKeys := map[config.Key]struct{}{
		config.ReverseSecretKey: struct{}{},
		config.SubmissionURL:    struct{}{},
	}

	for k, _ := range swamp {
		if _, ok := whitelistedConfigKeys[k]; ok {
			delete(c.Config, k)
		}

		if _, ok := whitelistedConfigKeys[k]; !ok {
			panic(fmt.Sprintf("PROVIDER BUG: API Config not empty: %#v", swamp))
		}
	}

	_StateSet(d, _CheckJSONAttr, schema.NewSet(_CheckJSONConfigChecksum, []interface{}{jsonConfig}))

	return nil
}

// _CheckJSONConfigChecksum creates a stable hash of the normalized values found
// in a user's Terraform config.
func _CheckJSONConfigChecksum(v interface{}) int {
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
	writeString(_CheckJSONAuthMethodAttr)
	writeString(_CheckJSONAuthPasswordAttr)
	writeString(_CheckJSONAuthUserAttr)
	writeString(_CheckJSONCAChainAttr)
	writeString(_CheckJSONCertFileAttr)
	writeString(_CheckJSONCiphersAttr)

	if headersRaw, ok := m[string(_CheckJSONHeadersAttr)]; ok {
		headerMap := headersRaw.(map[string]interface{})
		headers := make([]string, 0, len(headerMap))
		for k, _ := range headerMap {
			headers = append(headers, k)
		}

		sort.Strings(headers)
		for i, _ := range headers {
			fmt.Fprint(b, headers[i])
			fmt.Fprint(b, headerMap[headers[i]].(string))
		}
	}

	writeString(_CheckJSONKeyFileAttr)
	writeString(_CheckJSONMethodAttr)
	writeString(_CheckJSONPayloadAttr)
	writeString(_CheckJSONPortAttr)
	writeInt(_CheckJSONReadLimitAttr)
	writeString(_CheckJSONURLAttr)
	writeString(_CheckJSONVersionAttr)

	s := b.String()
	return hashcode.String(s)
}

func _CheckConfigToAPIJSON(c *_Check, ctxt *_ProviderContext, l _InterfaceList) error {
	c.Type = string(_APICheckTypeJSON)

	// Iterate over all `json` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		jsonConfig := _NewInterfaceMap(mapRaw)
		ar := _NewMapReader(ctxt, jsonConfig)

		if s, ok := ar.GetStringOK(_CheckJSONAuthMethodAttr); ok {
			c.Config[config.AuthMethod] = s
		}

		if s, ok := ar.GetStringOK(_CheckJSONAuthPasswordAttr); ok {
			c.Config[config.AuthPassword] = s
		}

		if s, ok := ar.GetStringOK(_CheckJSONAuthUserAttr); ok {
			c.Config[config.AuthUser] = s
		}

		if s, ok := ar.GetStringOK(_CheckJSONCAChainAttr); ok {
			c.Config[config.CAChain] = s
		}

		if s, ok := ar.GetStringOK(_CheckJSONCertFileAttr); ok {
			c.Config[config.CertFile] = s
		}

		if s, ok := ar.GetStringOK(_CheckJSONCiphersAttr); ok {
			c.Config[config.Ciphers] = s
		}

		if headers := jsonConfig.CollectMap(_CheckJSONHeadersAttr); headers != nil {
			for k, v := range headers {
				h := config.HeaderPrefix + config.Key(k)
				c.Config[h] = v
			}
		}

		if s, ok := ar.GetStringOK(_CheckJSONKeyFileAttr); ok {
			c.Config[config.KeyFile] = s
		}

		if s, ok := ar.GetStringOK(_CheckJSONMethodAttr); ok {
			c.Config[config.Method] = s
		}

		if s, ok := ar.GetStringOK(_CheckJSONPayloadAttr); ok {
			c.Config[config.Payload] = s
		}

		if s, ok := ar.GetStringOK(_CheckJSONPortAttr); ok {
			c.Config[config.Port] = s
		}

		if i, ok := ar.GetIntOK(_CheckJSONReadLimitAttr); ok {
			c.Config[config.ReadLimit] = fmt.Sprintf("%d", i)
		}

		if s, ok := ar.GetStringOK(_CheckJSONURLAttr); ok {
			c.Config[config.URL] = s

			u, _ := url.Parse(s)
			hostInfo := strings.SplitN(u.Host, ":", 2)
			if len(c.Target) == 0 {
				c.Target = hostInfo[0]
			}

			if len(hostInfo) == 2 {
				// Only override the port if no port has been set.  The config option
				// `port` takes precidence.
				if _, ok := c.Config[config.Port]; !ok {
					c.Config[config.Port] = hostInfo[1]
				}
			}
		}

		if s, ok := ar.GetStringOK(_CheckJSONVersionAttr); ok {
			c.Config[config.HTTPVersion] = s
		}
	}

	return nil
}
