package api

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"strings"

	"code.cloudfoundry.org/cli/cf/configuration/coreconfig"
	"code.cloudfoundry.org/cli/cf/errors"
	. "code.cloudfoundry.org/cli/cf/i18n"
	"code.cloudfoundry.org/cli/cf/net"
)

//go:generate counterfeiter . CurlRepository

type CurlRepository interface {
	Request(method, path, header, body string) (resHeaders string, resBody string, apiErr error)
}

type CloudControllerCurlRepository struct {
	config  coreconfig.Reader
	gateway net.Gateway
}

func NewCloudControllerCurlRepository(config coreconfig.Reader, gateway net.Gateway) (repo CloudControllerCurlRepository) {
	repo.config = config
	repo.gateway = gateway
	return
}

func (repo CloudControllerCurlRepository) Request(method, path, headerString, body string) (resHeaders, resBody string, err error) {
	url := fmt.Sprintf("%s/%s", repo.config.APIEndpoint(), strings.TrimLeft(path, "/"))

	if method == "" && body != "" {
		method = "POST"
	}

	req, err := repo.gateway.NewRequest(method, url, repo.config.AccessToken(), strings.NewReader(body))
	if err != nil {
		return
	}

	err = mergeHeaders(req.HTTPReq.Header, headerString)
	if err != nil {
		err = fmt.Errorf("%s: %s", T("Error parsing headers"), err.Error())
		return
	}

	res, err := repo.gateway.PerformRequest(req)

	if _, ok := err.(errors.HTTPError); ok {
		err = nil
	}

	if err != nil {
		return
	}
	defer res.Body.Close()

	headerBytes, _ := httputil.DumpResponse(res, false)
	resHeaders = string(headerBytes)

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("%s: %s", T("Error reading response"), err.Error())
	}
	resBody = string(bytes)

	return
}

func mergeHeaders(destination http.Header, headerString string) (err error) {
	headerString = strings.TrimSpace(headerString)
	headerString += "\n\n"
	headerReader := bufio.NewReader(strings.NewReader(headerString))
	headers, err := textproto.NewReader(headerReader).ReadMIMEHeader()
	if err != nil {
		return
	}

	for key, values := range headers {
		destination.Del(key)
		for _, value := range values {
			destination.Add(key, value)
		}
	}

	return
}
