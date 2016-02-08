## vmware-govcd

This package was originally forked from [github.com/vmware/govcloudair](https://github.com/vmware/govcloudair) before pulling in [rickard-von-essen's](https://github.com/rickard-von-essen)
great changes to allow using a [vCloud Director API](https://github.com/rickard-von-essen/govcloudair/tree/vcd-5.5). On top of this I have added features as needed for a terraform provider for vCloud Director

### Example ###

```go
package main

import (
	"fmt"
	"net/url"
    "os"

	"github.com/hmrc/vmware-govcd"
)

type Config struct {
	User     string
	Password string
	Org      string
	Href     string
	VDC      string
}

func (c *Config) Client() (*govcd.VCDClient, error) {
	u, err := url.ParseRequestURI(c.Href)
	if err != nil {
		return nil, fmt.Errorf("Unable to pass url: %s", err)
	}

	vcdclient := govcd.NewVCDClient(*u)
	org, vcd, err := vcdclient.Authenticate(c.User, c.Password, c.Org, c.VDC)
	if err != nil {
		return nil, fmt.Errorf("Unable to authenticate: %s", err)
	}
	vcdclient.Org = org
	vcdclient.OrgVdc = vcd
	return vcdclient, nil
}

func main() {
  config := Config{
		User:     "Username",
		Password: "password",
		Org:      "vcd org",
		Href:     "vcd api url",
		VDC:      "vcd virtual datacenter name",
	}

  client, err := config.Client() // We now have a client
  if err != nil {
      fmt.Println(err)
      os.Exit(1)
  }
  fmt.Printf("Org URL: %s\n", client.OrgHREF.String())
}
```
