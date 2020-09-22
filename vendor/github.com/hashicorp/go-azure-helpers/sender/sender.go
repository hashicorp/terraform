package sender

import (
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/Azure/go-autorest/autorest"
)

func BuildSender(providerName string) autorest.Sender {
	return autorest.DecorateSender(&http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}, withRequestLogging(providerName))
}

func withRequestLogging(providerName string) autorest.SendDecorator {
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			// strip the authorization header prior to printing
			authHeaderName := "Authorization"
			auth := r.Header.Get(authHeaderName)
			if auth != "" {
				r.Header.Del(authHeaderName)
			}

			// dump request to wire format
			if dump, err := httputil.DumpRequestOut(r, true); err == nil {
				log.Printf("[DEBUG] %s Request: \n%s\n", providerName, dump)
			} else {
				// fallback to basic message
				log.Printf("[DEBUG] %s Request: %s to %s\n", providerName, r.Method, r.URL)
			}

			// add the auth header back
			if auth != "" {
				r.Header.Add(authHeaderName, auth)
			}

			resp, err := s.Do(r)
			if resp != nil {
				// dump response to wire format
				if dump, err2 := httputil.DumpResponse(resp, true); err2 == nil {
					log.Printf("[DEBUG] %s Response for %s: \n%s\n", providerName, r.URL, dump)
				} else {
					// fallback to basic message
					log.Printf("[DEBUG] %s Response: %s for %s\n", providerName, resp.Status, r.URL)
				}
			} else {
				log.Printf("[DEBUG] Request to %s completed with no response", r.URL)
			}
			return resp, err
		})
	}
}
