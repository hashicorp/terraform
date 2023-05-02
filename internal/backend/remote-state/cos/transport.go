// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

const REQUEST_CLIENT = "TENCENTCLOUD_API_REQUEST_CLIENT"

var ReqClient = "Terraform-latest"

func SetReqClient(name string) {
	if name == "" {
		return
	}
	ReqClient = name
}

type LogRoundTripper struct {
}

func (me *LogRoundTripper) RoundTrip(request *http.Request) (response *http.Response, errRet error) {

	var inBytes, outBytes []byte

	var start = time.Now()

	defer func() { me.log(inBytes, outBytes, errRet, start) }()

	bodyReader, errRet := request.GetBody()
	if errRet != nil {
		return
	}
	var headName = "X-TC-Action"

	if envReqClient := os.Getenv(REQUEST_CLIENT); envReqClient != "" {
		ReqClient = envReqClient
	}

	request.Header.Set("X-TC-RequestClient", ReqClient)
	inBytes = []byte(fmt.Sprintf("%s, request: ", request.Header[headName]))
	requestBody, errRet := ioutil.ReadAll(bodyReader)
	if errRet != nil {
		return
	}
	inBytes = append(inBytes, requestBody...)

	headName = "X-TC-Region"
	appendMessage := []byte(fmt.Sprintf(
		", (host %+v, region:%+v)",
		request.Header["Host"],
		request.Header[headName],
	))

	inBytes = append(inBytes, appendMessage...)

	response, errRet = http.DefaultTransport.RoundTrip(request)
	if errRet != nil {
		return
	}
	outBytes, errRet = ioutil.ReadAll(response.Body)
	if errRet != nil {
		return
	}
	response.Body = ioutil.NopCloser(bytes.NewBuffer(outBytes))
	return
}

func (me *LogRoundTripper) log(in []byte, out []byte, err error, start time.Time) {
	var buf bytes.Buffer
	buf.WriteString("######")
	tag := "[DEBUG]"
	if err != nil {
		tag = "[CRITICAL]"
	}
	buf.WriteString(tag)
	if len(in) > 0 {
		buf.WriteString("tencentcloud-sdk-go: ")
		buf.Write(in)
	}
	if len(out) > 0 {
		buf.WriteString("; response:")
		err := json.Compact(&buf, out)
		if err != nil {
			out := bytes.Replace(out,
				[]byte("\n"),
				[]byte(""),
				-1)
			out = bytes.Replace(out,
				[]byte(" "),
				[]byte(""),
				-1)
			buf.Write(out)
		}
	}

	if err != nil {
		buf.WriteString("; error:")
		buf.WriteString(err.Error())
	}

	costFormat := fmt.Sprintf(",cost %s", time.Since(start).String())
	buf.WriteString(costFormat)

	log.Println(buf.String())
}
