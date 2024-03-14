// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleaddrs

import (
	"fmt"
	"net/url"
	"strings"
)

// detectS3 detects strings that seem like schemeless references to
// Amazon S3 and translates them into URLs for the "s3" getter.
func detectS3(src string) (string, bool, error) {
	if len(src) == 0 {
		return "", false, nil
	}

	if strings.Contains(src, ".amazonaws.com/") {
		parts := strings.Split(src, "/")
		if len(parts) < 2 {
			return "", false, fmt.Errorf(
				"URL is not a valid S3 URL")
		}

		hostParts := strings.Split(parts[0], ".")
		if len(hostParts) == 3 {
			return detectS3PathStyle(hostParts[0], parts[1:])
		} else if len(hostParts) == 4 {
			return detectS3OldVhostStyle(hostParts[1], hostParts[0], parts[1:])
		} else if len(hostParts) == 5 && hostParts[1] == "s3" {
			return detectS3NewVhostStyle(hostParts[2], hostParts[0], parts[1:])
		} else {
			return "", false, fmt.Errorf(
				"URL is not a valid S3 URL")
		}
	}

	return "", false, nil
}

func detectS3PathStyle(region string, parts []string) (string, bool, error) {
	urlStr := fmt.Sprintf("https://%s.amazonaws.com/%s", region, strings.Join(parts, "/"))
	url, err := url.Parse(urlStr)
	if err != nil {
		return "", false, fmt.Errorf("error parsing S3 URL: %s", err)
	}

	return "s3::" + url.String(), true, nil
}

func detectS3OldVhostStyle(region, bucket string, parts []string) (string, bool, error) {
	urlStr := fmt.Sprintf("https://%s.amazonaws.com/%s/%s", region, bucket, strings.Join(parts, "/"))
	url, err := url.Parse(urlStr)
	if err != nil {
		return "", false, fmt.Errorf("error parsing S3 URL: %s", err)
	}

	return "s3::" + url.String(), true, nil
}

func detectS3NewVhostStyle(region, bucket string, parts []string) (string, bool, error) {
	urlStr := fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", region, bucket, strings.Join(parts, "/"))
	url, err := url.Parse(urlStr)
	if err != nil {
		return "", false, fmt.Errorf("error parsing S3 URL: %s", err)
	}

	return "s3::" + url.String(), true, nil
}
