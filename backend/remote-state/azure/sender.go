package azure

import (
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/terraform/helper/logging"
)

func buildSender() autorest.Sender {
	return autorest.DecorateSender(&http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}, withRequestLogging())
}

func withRequestLogging() autorest.SendDecorator {
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			// only log if logging's enabled
			logLevel := logging.CurrentLogLevel()
			if logLevel == "" {
				return s.Do(r)
			}

			// strip the authorization header prior to printing
			authHeaderName := "Authorization"
			auth := r.Header.Get(authHeaderName)
			if auth != "" {
				r.Header.Del(authHeaderName)
			}

			// dump request to wire format
			if dump, err := httputil.DumpRequestOut(r, true); err == nil {
				log.Printf("[DEBUG] Azure Backend Request: \n%s\n", dump)
			} else {
				// fallback to basic message
				log.Printf("[DEBUG] Azure Backend Request: %s to %s\n", r.Method, r.URL)
			}

			// add the auth header back
			if auth != "" {
				r.Header.Add(authHeaderName, auth)
			}

			resp, err := s.Do(r)
			if resp != nil {
				// dump response to wire format
				if dump, err2 := httputil.DumpResponse(resp, true); err2 == nil {
					log.Printf("[DEBUG] Azure Backend Response for %s: \n%s\n", r.URL, dump)
				} else {
					// fallback to basic message
					log.Printf("[DEBUG] Azure Backend Response: %s for %s\n", resp.Status, r.URL)
				}
			} else {
				log.Printf("[DEBUG] Request to %s completed with no response", r.URL)
			}
			return resp, err
		})
	}
}
