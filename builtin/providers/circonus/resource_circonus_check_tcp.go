package circonus

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.tcp.* resource attribute names
	_CheckTCPBannerRegexpAttr _SchemaAttr = "banner_regexp"
	_CheckTCPCAChainAttr      _SchemaAttr = "ca_chain"
	_CheckTCPCertFileAttr     _SchemaAttr = "certificate_file"
	_CheckTCPCiphersAttr      _SchemaAttr = "ciphers"
	_CheckTCPKeyFileAttr      _SchemaAttr = "key_file"
	_CheckTCPPortAttr         _SchemaAttr = "port"
	_CheckTCPTLSAttr          _SchemaAttr = "tls"
)

var _CheckTCPDescriptions = _AttrDescrs{
	_CheckTCPBannerRegexpAttr: `This regular expression is matched against the response banner. If a match is not found, the check will be marked as bad.`,
	_CheckTCPCAChainAttr:      "A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for TLS checks).",
	_CheckTCPCertFileAttr:     "A path to a file containing the client certificate that will be presented to the remote server (for TLS checks).",
	_CheckTCPCiphersAttr:      "A list of ciphers to be used in the TLS protocol (for TCPS checks)",
	_CheckTCPKeyFileAttr:      "A path to a file containing key to be used in conjunction with the cilent certificate (for TLS checks)",
	_CheckTCPPortAttr:         "Specifies the port on which the management interface can be reached.",
	_CheckTCPTLSAttr:          "Upgrade TCP connection to use TLS.",
}

var _SchemaCheckTCP = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckTCP,
	Elem: &schema.Resource{
		Schema: _CastSchemaToTF(map[_SchemaAttr]*schema.Schema{
			_CheckTCPBannerRegexpAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckTCPBannerRegexpAttr, `.+`),
			},
			_CheckTCPCAChainAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckTCPCAChainAttr, `.+`),
			},
			_CheckTCPCertFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckTCPCertFileAttr, `.+`),
			},
			_CheckTCPCiphersAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckTCPCiphersAttr, `.+`),
			},
			_CheckTCPKeyFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: _ValidateRegexp(_CheckTCPKeyFileAttr, `.+`),
			},
			_CheckTCPPortAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ValidateFunc: _ValidateFuncs(
					_ValidateIntMin(_CheckTCPPortAttr, 0),
					_ValidateIntMax(_CheckTCPPortAttr, 65535),
				),
			},
			_CheckTCPTLSAttr: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		}, _CheckTCPDescriptions),
	},
}

// _CheckAPIToStateTCP reads the Config data out of _Check.CheckBundle into the
// statefile.
func _CheckAPIToStateTCP(c *_Check, d *schema.ResourceData) error {
	tcpConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, v := range c.Config {
		swamp[k] = v
	}

	saveStringConfigToState := func(apiKey config.Key, attrName _SchemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			tcpConfig[string(attrName)] = v
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
			tcpConfig[string(attrName)] = int(i)
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState(config.BannerMatch, _CheckTCPBannerRegexpAttr)
	saveStringConfigToState(config.CAChain, _CheckTCPCAChainAttr)
	saveStringConfigToState(config.CertFile, _CheckTCPCertFileAttr)
	saveStringConfigToState(config.Ciphers, _CheckTCPCiphersAttr)
	saveStringConfigToState(config.KeyFile, _CheckTCPKeyFileAttr)
	saveIntConfigToState(config.Port, _CheckTCPPortAttr)
	saveStringConfigToState(config.UseSSL, _CheckTCPTLSAttr)

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

	_StateSet(d, _CheckTCPAttr, schema.NewSet(hashCheckTCP, []interface{}{tcpConfig}))

	return nil
}

// hashCheckTCP creates a stable hash of the normalized values
func hashCheckTCP(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeBool := func(attrName _SchemaAttr) {
		if v, ok := m[string(attrName)]; ok {
			fmt.Fprintf(b, "%t", v.(bool))
		}
	}

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
	writeString(_CheckTCPBannerRegexpAttr)
	writeString(_CheckTCPCAChainAttr)
	writeString(_CheckTCPCertFileAttr)
	writeString(_CheckTCPCiphersAttr)
	writeString(_CheckTCPKeyFileAttr)
	writeInt(_CheckTCPPortAttr)
	writeBool(_CheckTCPTLSAttr)

	s := b.String()
	return hashcode.String(s)
}

func _CheckConfigToAPITCP(c *_Check, ctxt *_ProviderContext, l _InterfaceList) error {
	c.Type = string(_APICheckTypeTCP)

	// Iterate over all `tcp` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		tcpConfig := _NewInterfaceMap(mapRaw)
		ar := _NewMapReader(ctxt, tcpConfig)

		if s, ok := ar.GetStringOK(_CheckTCPBannerRegexpAttr); ok {
			c.Config[config.BannerMatch] = s
		}

		if s, ok := ar.GetStringOK(_CheckTCPCAChainAttr); ok {
			c.Config[config.CAChain] = s
		}

		if s, ok := ar.GetStringOK(_CheckTCPCertFileAttr); ok {
			c.Config[config.CertFile] = s
		}

		if s, ok := ar.GetStringOK(_CheckTCPCiphersAttr); ok {
			c.Config[config.Ciphers] = s
		}

		if s, ok := ar.GetStringOK(_CheckTCPKeyFileAttr); ok {
			c.Config[config.KeyFile] = s
		}

		if i, ok := ar.GetIntOK(_CheckTCPPortAttr); ok {
			c.Config[config.Port] = fmt.Sprintf("%d", i)
		}

		if b, ok := ar.GetBoolOK(_CheckTCPTLSAttr); ok {
			c.Config[config.UseSSL] = fmt.Sprintf("%t", b)
		}
	}

	return nil
}
