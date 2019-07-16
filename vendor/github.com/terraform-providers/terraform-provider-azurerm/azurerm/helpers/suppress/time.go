package suppress

import (
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

func RFC3339Time(_, old, new string, _ *schema.ResourceData) bool {
	ot, oerr := time.Parse(time.RFC3339, old)
	nt, nerr := time.Parse(time.RFC3339, new)

	if oerr != nil || nerr != nil {
		return false
	}

	return nt.Equal(ot)
}
