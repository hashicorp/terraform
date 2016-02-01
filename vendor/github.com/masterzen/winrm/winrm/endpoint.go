package winrm

import "fmt"

type Endpoint struct {
	Host     string
	Port     int
	HTTPS    bool
	Insecure bool
	CACert   *[]byte
}

func (ep *Endpoint) url() string {
	var scheme string
	if ep.HTTPS {
		scheme = "https"
	} else {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s:%d/wsman", scheme, ep.Host, ep.Port)
}
