// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package moduleaddrs

import (
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
)

var forcedRegexp = regexp.MustCompile(`^([A-Za-z0-9]+)::(.+)$`)

// This is the minimal set of detectors we need for backward compatibility.
//
// Do not add any new detectors. All new source types should use the canonical
// source address syntax.
var detectors = []func(src string) (string, bool, error){
	detectGitHub,
	detectGit,
	detectBitBucket,
	detectGCS,
	detectS3,
	detectAbsFilePath,
}

// detectRemoteSourceShorthands recognizes several non-URL strings that
// Terraform historically accepted as shorthands for module source addresses,
// and converts them each into something more reasonable that specifies
// both a source type and a fully-qualified URL.
func detectRemoteSourceShorthands(src string) (string, error) {
	getForce, getSrc := getForcedSourceType(src)

	// Separate out the subdir if there is one, we don't pass that to detect
	getSrc, subDir := SplitPackageSubdir(getSrc)

	u, err := url.Parse(getSrc)
	if err == nil && u.Scheme != "" {
		// Valid URL
		return src, nil
	}

	for _, d := range detectors {
		result, ok, err := d(getSrc)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}

		var detectForce string
		detectForce, result = getForcedSourceType(result)
		result, detectSubdir := SplitPackageSubdir(result)

		// If we have a subdir from the detection, then prepend it to our
		// requested subdir.
		if detectSubdir != "" {
			if subDir != "" {
				subDir = filepath.Join(detectSubdir, subDir)
			} else {
				subDir = detectSubdir
			}
		}

		if subDir != "" {
			u, err := url.Parse(result)
			if err != nil {
				return "", fmt.Errorf("Error parsing URL: %s", err)
			}
			u.Path += "//" + subDir

			// a subdir may contain wildcards, but in order to support them we
			// have to ensure the path isn't escaped.
			u.RawPath = u.Path

			result = u.String()
		}

		// Preserve the forced getter if it exists. We try to use the
		// original set force first, followed by any force set by the
		// detector.
		if getForce != "" {
			result = fmt.Sprintf("%s::%s", getForce, result)
		} else if detectForce != "" {
			result = fmt.Sprintf("%s::%s", detectForce, result)
		}

		return result, nil
	}

	return "", fmt.Errorf("invalid source address: %s", src)
}

func getForcedSourceType(src string) (string, string) {
	var forced string
	if ms := forcedRegexp.FindStringSubmatch(src); ms != nil {
		forced = ms[1]
		src = ms[2]
	}

	return forced, src
}
