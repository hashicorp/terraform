package opsgenie

import (
	"fmt"
	"net/http"
)

func checkOpsGenieResponse(code int, status string) error {
	if code == http.StatusOK {
		return nil
	}

	return fmt.Errorf("Unexpected Status Code '%d', Response '%s'", code, status)
}
