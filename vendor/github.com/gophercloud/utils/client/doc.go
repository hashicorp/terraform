/*
Package client provides an ability to create a http.RoundTripper OpenStack
client with extended options, including the JSON requests and responses log
capabilities.

Example usage with the default logger:

	package example

	import (
		"net/http"
		"os"

		"github.com/gophercloud/gophercloud"
		"github.com/gophercloud/gophercloud/openstack"
		"github.com/gophercloud/utils/client"
		"github.com/gophercloud/utils/openstack/clientconfig"
	)

	func NewComputeV2Client() (*gophercloud.ServiceClient, error) {
		ao, err := clientconfig.AuthOptions(nil)
		if err != nil {
			return nil, err
		}

		provider, err := openstack.NewClient(ao.IdentityEndpoint)
		if err != nil {
			return nil, err
		}

		if os.Getenv("OS_DEBUG") != "" {
			provider.HTTPClient = http.Client{
				Transport: &client.RoundTripper{
					Rt:     &http.Transport{},
					Logger: &client.DefaultLogger{},
				},
			}
		}

		err = openstack.Authenticate(provider, *ao)
		if err != nil {
			return nil, err
		}

		return openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
			Region: os.Getenv("OS_REGION_NAME"),
		})
	}

Example usage with the custom logger:

	package example

	import (
		"net/http"
		"os"

		"github.com/gophercloud/gophercloud"
		"github.com/gophercloud/gophercloud/openstack"
		"github.com/gophercloud/utils/client"
		"github.com/gophercloud/utils/openstack/clientconfig"
		log "github.com/sirupsen/logrus"
	)

	type myLogger struct {
		Prefix string
	}

	func (l myLogger) Printf(format string, args ...interface{}) {
		log.Debugf("%s [DEBUG] "+format, append([]interface{}{l.Prefix}, args...)...)
	}

	func NewComputeV2Client() (*gophercloud.ServiceClient, error) {
		ao, err := clientconfig.AuthOptions(nil)
		if err != nil {
			return nil, err
		}

		provider, err := openstack.NewClient(ao.IdentityEndpoint)
		if err != nil {
			return nil, err
		}

		if os.Getenv("OS_DEBUG") != "" {
			provider.HTTPClient = http.Client{
				Transport: &client.RoundTripper{
					Rt:     &http.Transport{},
					Logger: &myLogger{Prefix: "myApp"},
				},
			}
		}

		err = openstack.Authenticate(provider, *ao)
		if err != nil {
			return nil, err
		}

		return openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
			Region: os.Getenv("OS_REGION_NAME"),
		})
	}

Example usage with additinal headers:

	package example

	import (
		"net/http"
		"os"

		"github.com/gophercloud/gophercloud"
		"github.com/gophercloud/gophercloud/openstack"
		"github.com/gophercloud/utils/client"
		"github.com/gophercloud/utils/openstack/clientconfig"
	)

	func NewComputeV2Client() (*gophercloud.ServiceClient, error) {
		ao, err := clientconfig.AuthOptions(nil)
		if err != nil {
			return nil, err
		}

		provider, err := openstack.NewClient(ao.IdentityEndpoint)
		if err != nil {
			return nil, err
		}

		provider.HTTPClient = http.Client{
			Transport: &client.RoundTripper{
				Rt:     &http.Transport{},
			},
		}

		provider.HTTPClient.Transport.(*client.RoundTripper).SetHeaders(map[string][]string{"Cache-Control": {"no-cache"}}})

		err = openstack.Authenticate(provider, *ao)
		if err != nil {
			return nil, err
		}

		return openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
			Region: os.Getenv("OS_REGION_NAME"),
		})
	}

*/
package client
