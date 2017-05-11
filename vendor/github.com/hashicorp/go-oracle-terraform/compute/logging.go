package compute

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-oracle-terraform/opc"
)

// Log a string if debug logs are on
func (c *Client) debugLogString(str string) {
	if c.loglevel != opc.LogDebug {
		return
	}
	c.logger.Log(str)
}

func (c *Client) debugLogReq(req *http.Request) {
	// Don't need to log this if not debugging
	if c.loglevel != opc.LogDebug {
		return
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(req.Body)
	c.logger.Log(fmt.Sprintf("DEBUG: HTTP %s Req %s: %s",
		req.Method, req.URL.String(), buf.String()))
}
