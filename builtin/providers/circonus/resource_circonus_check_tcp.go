package circonus

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

const (
	// circonus_check.tcp.* resource attribute names
	checkTCPBannerRegexpAttr schemaAttr = "banner_regexp"
	checkTCPCAChainAttr      schemaAttr = "ca_chain"
	checkTCPCertFileAttr     schemaAttr = "certificate_file"
	checkTCPCiphersAttr      schemaAttr = "ciphers"
	checkTCPHostAttr         schemaAttr = "host"
	checkTCPKeyFileAttr      schemaAttr = "key_file"
	checkTCPPortAttr         schemaAttr = "port"
	checkTCPTLSAttr          schemaAttr = "tls"
)

var checkTCPDescriptions = attrDescrs{
	checkTCPBannerRegexpAttr: `This regular expression is matched against the response banner. If a match is not found, the check will be marked as bad.`,
	checkTCPCAChainAttr:      "A path to a file containing all the certificate authorities that should be loaded to validate the remote certificate (for TLS checks).",
	checkTCPCertFileAttr:     "A path to a file containing the client certificate that will be presented to the remote server (for TLS checks).",
	checkTCPCiphersAttr:      "A list of ciphers to be used when establishing a TLS connection",
	checkTCPHostAttr:         "Specifies the host name or IP address to connect to for this TCP check",
	checkTCPKeyFileAttr:      "A path to a file containing key to be used in conjunction with the cilent certificate (for TLS checks)",
	checkTCPPortAttr:         "Specifies the port on which the management interface can be reached.",
	checkTCPTLSAttr:          "Upgrade TCP connection to use TLS.",
}

var schemaCheckTCP = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	MaxItems: 1,
	MinItems: 1,
	Set:      hashCheckTCP,
	Elem: &schema.Resource{
		Schema: castSchemaToTF(map[schemaAttr]*schema.Schema{
			checkTCPBannerRegexpAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkTCPBannerRegexpAttr, `.+`),
			},
			checkTCPCAChainAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkTCPCAChainAttr, `.+`),
			},
			checkTCPCertFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkTCPCertFileAttr, `.+`),
			},
			checkTCPCiphersAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkTCPCiphersAttr, `.+`),
			},
			checkTCPHostAttr: &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRegexp(checkTCPHostAttr, `.+`),
			},
			checkTCPKeyFileAttr: &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateRegexp(checkTCPKeyFileAttr, `.+`),
			},
			checkTCPPortAttr: &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ValidateFunc: validateFuncs(
					validateIntMin(checkTCPPortAttr, 0),
					validateIntMax(checkTCPPortAttr, 65535),
				),
			},
			checkTCPTLSAttr: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		}, checkTCPDescriptions),
	},
}

// checkAPIToStateTCP reads the Config data out of circonusCheck.CheckBundle into the
// statefile.
func checkAPIToStateTCP(c *circonusCheck, d *schema.ResourceData) error {
	tcpConfig := make(map[string]interface{}, len(c.Config))

	// swamp is a sanity check: it must be empty by the time this method returns
	swamp := make(map[config.Key]string, len(c.Config))
	for k, v := range c.Config {
		swamp[k] = v
	}

	saveBoolConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if s, ok := c.Config[apiKey]; ok {
			switch strings.ToLower(s) {
			case "1", "true", "t", "yes", "y":
				tcpConfig[string(attrName)] = true
			case "0", "false", "f", "no", "n":
				tcpConfig[string(attrName)] = false
			default:
				panic(fmt.Sprintf("PROVIDER BUG: unsupported boolean: %q for API Config Key %q", s, string(apiKey)))
			}
		}

		delete(swamp, apiKey)
	}

	saveIntConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				panic(fmt.Sprintf("Unable to convert %s to an integer: %v", apiKey, err))
			}
			tcpConfig[string(attrName)] = int(i)
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState := func(apiKey config.Key, attrName schemaAttr) {
		if v, ok := c.Config[apiKey]; ok {
			tcpConfig[string(attrName)] = v
		}

		delete(swamp, apiKey)
	}

	saveStringConfigToState(config.BannerMatch, checkTCPBannerRegexpAttr)
	saveStringConfigToState(config.CAChain, checkTCPCAChainAttr)
	saveStringConfigToState(config.CertFile, checkTCPCertFileAttr)
	saveStringConfigToState(config.Ciphers, checkTCPCiphersAttr)
	tcpConfig[string(checkTCPHostAttr)] = c.Target
	saveStringConfigToState(config.KeyFile, checkTCPKeyFileAttr)
	saveIntConfigToState(config.Port, checkTCPPortAttr)
	saveBoolConfigToState(config.UseSSL, checkTCPTLSAttr)

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

	if err := d.Set(checkTCPAttr, schema.NewSet(hashCheckTCP, []interface{}{tcpConfig})); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Unable to store check %q attribute: {{err}}", checkTCPAttr), err)
	}

	return nil
}

// hashCheckTCP creates a stable hash of the normalized values
func hashCheckTCP(v interface{}) int {
	m := v.(map[string]interface{})
	b := &bytes.Buffer{}
	b.Grow(defaultHashBufSize)

	writeBool := func(attrName schemaAttr) {
		if v, ok := m[string(attrName)]; ok {
			fmt.Fprintf(b, "%t", v.(bool))
		}
	}

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
	writeString(checkTCPBannerRegexpAttr)
	writeString(checkTCPCAChainAttr)
	writeString(checkTCPCertFileAttr)
	writeString(checkTCPCiphersAttr)
	writeString(checkTCPHostAttr)
	writeString(checkTCPKeyFileAttr)
	writeInt(checkTCPPortAttr)
	writeBool(checkTCPTLSAttr)

	s := b.String()
	return hashcode.String(s)
}

func checkConfigToAPITCP(c *circonusCheck, ctxt *providerContext, l interfaceList) error {
	c.Type = string(apiCheckTypeTCP)

	// Iterate over all `tcp` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		tcpConfig := newInterfaceMap(mapRaw)
		ar := newMapReader(ctxt, tcpConfig)

		if s, ok := ar.GetStringOK(checkTCPBannerRegexpAttr); ok {
			c.Config[config.BannerMatch] = s
		}

		if s, ok := ar.GetStringOK(checkTCPCAChainAttr); ok {
			c.Config[config.CAChain] = s
		}

		if s, ok := ar.GetStringOK(checkTCPCertFileAttr); ok {
			c.Config[config.CertFile] = s
		}

		if s, ok := ar.GetStringOK(checkTCPCiphersAttr); ok {
			c.Config[config.Ciphers] = s
		}

		if s, ok := ar.GetStringOK(checkTCPHostAttr); ok {
			c.Target = s
		}

		if s, ok := ar.GetStringOK(checkTCPKeyFileAttr); ok {
			c.Config[config.KeyFile] = s
		}

		if i, ok := ar.GetIntOK(checkTCPPortAttr); ok {
			c.Config[config.Port] = fmt.Sprintf("%d", i)
		}

		if b, ok := ar.GetBoolOK(checkTCPTLSAttr); ok {
			c.Config[config.UseSSL] = fmt.Sprintf("%t", b)
		}
	}

	return nil
}
