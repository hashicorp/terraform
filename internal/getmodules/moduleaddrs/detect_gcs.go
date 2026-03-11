// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleaddrs

import (
	"fmt"
	"net/url"
	"strings"
)

// detectGCS detects strings that seem like schemeless references to
// Google Cloud Storage and translates them into URLs for the "gcs" getter.
func detectGCS(src string) (string, bool, error) {
	if len(src) == 0 {
		return "", false, nil
	}

	if strings.Contains(src, "googleapis.com/") {
		parts := strings.Split(src, "/")
		if len(parts) < 5 {
			return "", false, fmt.Errorf(
				"URL is not a valid GCS URL")
		}
		version := parts[2]
		bucket := parts[3]
		object := strings.Join(parts[4:], "/")

		url, err := url.Parse(fmt.Sprintf("https://www.googleapis.com/storage/%s/%s/%s",
			version, bucket, object))
		if err != nil {
			return "", false, fmt.Errorf("error parsing GCS URL: %s", err)
		}

		return "gcs::" + url.String(), true, nil
	}

	return "", false, nil
}
