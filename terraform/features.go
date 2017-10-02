package terraform

import (
	"os"
)

// This file holds feature flags for the next release

var featureOutputErrors = os.Getenv("TF_OUTPUT_ERRORS") != ""
