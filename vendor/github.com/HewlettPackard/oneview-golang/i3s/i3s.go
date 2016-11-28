/*
(c) Copyright [2015] Hewlett Packard Enterprise Development LP

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package i3s - Image Streamer 3.0 -
package i3s

import (
	"github.com/HewlettPackard/oneview-golang/rest"
)

// I3SClient - wrapper class for i3s api's
type I3SClient struct {
	rest.Client
}

// new Client
func (c *I3SClient) NewI3SClient(endpoint string, sslverify bool, apiversion int, apiKey string) *I3SClient {
	return &I3SClient{
		rest.Client{
			Endpoint:   endpoint,
			SSLVerify:  sslverify,
			APIVersion: apiversion,
			APIKey:     apiKey,
		},
	}
}
