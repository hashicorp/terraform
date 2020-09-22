package containers

import (
	"fmt"

	"github.com/tombuildsstuff/giovanni/version"
)

// APIVersion is the version of the API used for all Storage API Operations
const APIVersion = "2018-11-09"

func UserAgent() string {
	return fmt.Sprintf("tombuildsstuff/giovanni/%s storage/%s", version.Number, APIVersion)
}
