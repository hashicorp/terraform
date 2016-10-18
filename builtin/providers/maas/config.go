package maas

import (
	"launchpad.net/gomaasapi"
	"log"
)

type NodeInfo struct {
	system_id     string
	hostname      string
	url           string
	power_state   string
	cpu_count     uint16
	architecture  string
	distro_series string
	memory        uint64
	osystem       string
	status        uint16
	substatus     uint16
	tag_names     []string
	data          map[string]interface{}
}

type Config struct {
	APIKey     string
	APIURL     string
	APIver     string
	MAASObject *gomaasapi.MAASObject
}

func (c *Config) Client() (interface{}, error) {
	log.Println("[DEBUG] [Config.Client] Configuring the MAAS API client")
	authClient, err := gomaasapi.NewAuthenticatedClient(c.APIURL, c.APIKey, c.APIver)
	if err != nil {
		log.Printf("[ERROR] [Config.Client] Unable to authenticate against the MAAS Server (%s)", c.APIURL)
		return nil, err
	}
	c.MAASObject = gomaasapi.NewMAAS(*authClient)
	return c, nil
}
