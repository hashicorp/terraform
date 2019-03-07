package requests

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_AcsRequest(t *testing.T) {
	r := defaultBaseRequest()
	assert.NotNil(t, r)

	// query params
	query := r.GetQueryParams()
	assert.Equal(t, 0, len(query))
	r.addQueryParam("key", "value")
	assert.Equal(t, 1, len(query))
	assert.Equal(t, "value", query["key"])

	// form params
	form := r.GetFormParams()
	assert.Equal(t, 0, len(form))
	r.addFormParam("key", "value")
	assert.Equal(t, 1, len(form))
	assert.Equal(t, "value", form["key"])

	// getter/setter for stringtosign
	assert.Equal(t, "", r.GetStringToSign())
	r.SetStringToSign("s2s")
	assert.Equal(t, "s2s", r.GetStringToSign())

	// content type
	_, contains := r.GetContentType()
	assert.False(t, contains)
	r.SetContentType("application/json")
	ct, contains := r.GetContentType()
	assert.Equal(t, "application/json", ct)
	assert.True(t, contains)

	// default 3 headers & content-type
	headers := r.GetHeaders()
	assert.Equal(t, 4, len(headers))
	r.addHeaderParam("x-key", "x-key-value")
	assert.Equal(t, 5, len(headers))
	assert.Equal(t, "x-key-value", headers["x-key"])

	// GetVersion
	assert.Equal(t, "", r.GetVersion())
	// GetActionName
	assert.Equal(t, "", r.GetActionName())

	// GetMethod
	assert.Equal(t, "GET", r.GetMethod())
	r.Method = "POST"
	assert.Equal(t, "POST", r.GetMethod())

	// Domain
	assert.Equal(t, "", r.GetDomain())
	r.SetDomain("ecs.aliyuncs.com")
	assert.Equal(t, "ecs.aliyuncs.com", r.GetDomain())

	// Region
	assert.Equal(t, "", r.GetRegionId())
	r.RegionId = "cn-hangzhou"
	assert.Equal(t, "cn-hangzhou", r.GetRegionId())

	// AcceptFormat
	assert.Equal(t, "JSON", r.GetAcceptFormat())
	r.AcceptFormat = "XML"
	assert.Equal(t, "XML", r.GetAcceptFormat())

	// GetLocationServiceCode
	assert.Equal(t, "", r.GetLocationServiceCode())

	// GetLocationEndpointType
	assert.Equal(t, "", r.GetLocationEndpointType())

	// GetProduct
	assert.Equal(t, "", r.GetProduct())

	// GetScheme
	assert.Equal(t, "", r.GetScheme())
	r.SetScheme("HTTPS")
	assert.Equal(t, "HTTPS", r.GetScheme())

	// GetPort
	assert.Equal(t, "", r.GetPort())

	// GetUserAgent
	r.AppendUserAgent("cli", "1.01")
	assert.Equal(t, "1.01", r.GetUserAgent()["cli"])
	// Content
	assert.Equal(t, []byte(nil), r.GetContent())
	r.SetContent([]byte("The Content"))
	assert.True(t, bytes.Equal([]byte("The Content"), r.GetContent()))
}

type AcsRequestTest struct {
	*baseRequest
	Ontology AcsRequest
	Query    string      `position:"Query" name:"Query"`
	Header   string      `position:"Header" name:"Header"`
	Path     string      `position:"Path" name:"Path"`
	Body     string      `position:"Body" name:"Body"`
	TypeAcs  *[]string   `position:"type" name:"type" type:"Repeated"`
}

func (r AcsRequestTest) BuildQueries() string {
	return ""
}

func (r AcsRequestTest) BuildUrl() string {
	return ""
}

func (r AcsRequestTest) GetBodyReader() io.Reader {
	return nil
}

func (r AcsRequestTest) GetStyle() string {
	return ""
}

func (r AcsRequestTest) addPathParam(key, value string) {
	return
}

func Test_AcsRequest_InitParams(t *testing.T) {
	r := &AcsRequestTest{
		baseRequest: defaultBaseRequest(),
		Query:       "query value",
		Header:      "header value",
		Path:        "path value",
		Body:        "body value",
	}
	tmp := []string{r.Query, r.Header}
	r.TypeAcs = &tmp
	r.addQueryParam("qkey", "qvalue")
	InitParams(r)

	queries := r.GetQueryParams()
	assert.Equal(t, "query value", queries["Query"])
	headers := r.GetHeaders()
	assert.Equal(t, "header value", headers["Header"])
	// TODO: check the body & path
}
