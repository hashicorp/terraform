// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azure

import (
	"log"
	"net/http"

	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/terraform/internal/logging"
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

			log.Printf("[DEBUG] Azure Backend Request: %s to %s\n", r.Method, r.URL)

			// add the auth header back
			if auth != "" {
				r.Header.Add(authHeaderName, auth)
			}

			resp, err := s.Do(r)
			if resp != nil {
				log.Printf("[DEBUG] Azure Backend Response: %s for %s\n", resp.Status, r.URL)
			} else {
				log.Printf("[DEBUG] Request to %s completed with no response", r.URL)
			}
			return resp, err
		})
	}
}
