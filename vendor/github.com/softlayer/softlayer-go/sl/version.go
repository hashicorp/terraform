/**
 * Copyright 2016 IBM Corp.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * AUTOMATICALLY GENERATED CODE - DO NOT MODIFY
 */

package sl

import "fmt"

type VersionInfo struct {
	Major int
	Minor int
	Patch int
	Pre   string
}

var Version = VersionInfo{
	Major: 0,
	Minor: 1,
	Patch: 0,
	Pre:   "alpha",
}

func (v VersionInfo) String() string {
	result := fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)

	if v.Pre != "" {
		result += fmt.Sprintf("-%s", v.Pre)
	}

	return result
}
