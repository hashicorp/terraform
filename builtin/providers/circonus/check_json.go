package circonus

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/circonus-labs/circonus-gometrics/api/config"
)

func (c *_Check) parseJSONCheck(ctxt *_ProviderContext, l _InterfaceList) error {
	c.Type = string(_CheckTypeJSON)

	// Iterate over all `json` attributes, even though we have a max of 1 in the
	// schema.
	for _, mapRaw := range l {
		jsonConfig := _NewInterfaceMap(mapRaw)
		ar := _NewMapReader(ctxt, jsonConfig)

		if s, ok := ar.GetStringOk(_CheckJSONAuthMethodAttr); ok {
			c.Config[config.AuthMethod] = s
		}

		if s, ok := ar.GetStringOk(_CheckJSONAuthPasswordAttr); ok {
			c.Config[config.AuthPassword] = s
		}

		if s, ok := ar.GetStringOk(_CheckJSONAuthUserAttr); ok {
			c.Config[config.AuthUser] = s
		}

		if s, ok := ar.GetStringOk(_CheckJSONCAChainAttr); ok {
			c.Config[config.CAChain] = s
		}

		if s, ok := ar.GetStringOk(_CheckJSONCertFileAttr); ok {
			c.Config[config.CertFile] = s
		}

		if s, ok := ar.GetStringOk(_CheckJSONCiphersAttr); ok {
			c.Config[config.Ciphers] = s
		}

		if headers := jsonConfig.CollectMap(_CheckJSONHeadersAttr); headers != nil {
			for k, v := range headers {
				h := config.HeaderPrefix + config.Key(k)
				c.Config[h] = v
			}
		}

		if s, ok := ar.GetStringOk(_CheckJSONKeyFileAttr); ok {
			c.Config[config.KeyFile] = s
		}

		if s, ok := ar.GetStringOk(_CheckJSONMethodAttr); ok {
			c.Config[config.Method] = s
		}

		if s, ok := ar.GetStringOk(_CheckJSONPayloadAttr); ok {
			c.Config[config.Payload] = s
		}

		if s, ok := ar.GetStringOk(_CheckJSONPortAttr); ok {
			c.Config[config.Port] = s
		}

		if i, ok := ar.GetIntOk(_CheckJSONReadLimitAttr); ok {
			c.Config[config.ReadLimit] = fmt.Sprintf("%d", i)
		}

		if s, ok := ar.GetStringOk(_CheckJSONURLAttr); ok {
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

		if s, ok := ar.GetStringOk(_CheckJSONVersionAttr); ok {
			c.Config[config.HTTPVersion] = s
		}
	}

	return nil
}
