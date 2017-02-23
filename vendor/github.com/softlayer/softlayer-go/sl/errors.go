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

package sl

import "fmt"

// Error contains detailed information about an API error, which can be useful
// for debugging, or when finer error handling is required than just the mere
// presence or absence of an error.
//
// Error implements the error interface
type Error struct {
	StatusCode int
	Exception  string `json:"code"`
	Message    string `json:"error"`
	Wrapped    error
}

func (r Error) Error() string {
	if r.Wrapped != nil {
		return r.Wrapped.Error()
	}

	var msg string
	if r.Exception != "" {
		msg = r.Exception + ": "
	}
	if r.Message != "" {
		msg = msg + r.Message + " "
	}
	if r.StatusCode != 0 {
		msg = fmt.Sprintf("%s(HTTP %d)", msg, r.StatusCode)
	}
	return msg
}
