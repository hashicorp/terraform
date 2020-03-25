// Copyright 2019 Huawei Technologies Co.,Ltd.
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use
// this file except in compliance with the License.  You may obtain a copy of the
// License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software distributed
// under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
// CONDITIONS OF ANY KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations under the License.

package obs

import (
	"encoding/xml"
	"fmt"
)

type ObsError struct {
	BaseModel
	Status   string
	XMLName  xml.Name `xml:"Error"`
	Code     string   `xml:"Code"`
	Message  string   `xml:"Message"`
	Resource string   `xml:"Resource"`
	HostId   string   `xml:"HostId"`
}

func (err ObsError) Error() string {
	return fmt.Sprintf("obs: service returned error: Status=%s, Code=%s, Message=%s, RequestId=%s",
		err.Status, err.Code, err.Message, err.RequestId)
}
