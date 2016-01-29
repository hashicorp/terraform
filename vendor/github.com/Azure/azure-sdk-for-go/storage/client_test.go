package storage

import (
	"encoding/base64"
	"net/url"
	"os"
	"testing"

	chk "github.com/Azure/azure-sdk-for-go/Godeps/_workspace/src/gopkg.in/check.v1"
)

// Hook up gocheck to testing
func Test(t *testing.T) { chk.TestingT(t) }

type StorageClientSuite struct{}

var _ = chk.Suite(&StorageClientSuite{})

// getBasicClient returns a test client from storage credentials in the env
func getBasicClient(c *chk.C) Client {
	name := os.Getenv("ACCOUNT_NAME")
	if name == "" {
		c.Fatal("ACCOUNT_NAME not set, need an empty storage account to test")
	}
	key := os.Getenv("ACCOUNT_KEY")
	if key == "" {
		c.Fatal("ACCOUNT_KEY not set")
	}
	cli, err := NewBasicClient(name, key)
	c.Assert(err, chk.IsNil)
	return cli
}

func (s *StorageClientSuite) TestGetBaseURL_Basic_Https(c *chk.C) {
	cli, err := NewBasicClient("foo", "YmFy")
	c.Assert(err, chk.IsNil)
	c.Assert(cli.apiVersion, chk.Equals, DefaultAPIVersion)
	c.Assert(err, chk.IsNil)
	c.Assert(cli.getBaseURL("table"), chk.Equals, "https://foo.table.core.windows.net")
}

func (s *StorageClientSuite) TestGetBaseURL_Custom_NoHttps(c *chk.C) {
	apiVersion := "2015-01-01" // a non existing one
	cli, err := NewClient("foo", "YmFy", "core.chinacloudapi.cn", apiVersion, false)
	c.Assert(err, chk.IsNil)
	c.Assert(cli.apiVersion, chk.Equals, apiVersion)
	c.Assert(cli.getBaseURL("table"), chk.Equals, "http://foo.table.core.chinacloudapi.cn")
}

func (s *StorageClientSuite) TestGetEndpoint_None(c *chk.C) {
	cli, err := NewBasicClient("foo", "YmFy")
	c.Assert(err, chk.IsNil)
	output := cli.getEndpoint(blobServiceName, "", url.Values{})
	c.Assert(output, chk.Equals, "https://foo.blob.core.windows.net/")
}

func (s *StorageClientSuite) TestGetEndpoint_PathOnly(c *chk.C) {
	cli, err := NewBasicClient("foo", "YmFy")
	c.Assert(err, chk.IsNil)
	output := cli.getEndpoint(blobServiceName, "path", url.Values{})
	c.Assert(output, chk.Equals, "https://foo.blob.core.windows.net/path")
}

func (s *StorageClientSuite) TestGetEndpoint_ParamsOnly(c *chk.C) {
	cli, err := NewBasicClient("foo", "YmFy")
	c.Assert(err, chk.IsNil)
	params := url.Values{}
	params.Set("a", "b")
	params.Set("c", "d")
	output := cli.getEndpoint(blobServiceName, "", params)
	c.Assert(output, chk.Equals, "https://foo.blob.core.windows.net/?a=b&c=d")
}

func (s *StorageClientSuite) TestGetEndpoint_Mixed(c *chk.C) {
	cli, err := NewBasicClient("foo", "YmFy")
	c.Assert(err, chk.IsNil)
	params := url.Values{}
	params.Set("a", "b")
	params.Set("c", "d")
	output := cli.getEndpoint(blobServiceName, "path", params)
	c.Assert(output, chk.Equals, "https://foo.blob.core.windows.net/path?a=b&c=d")
}

func (s *StorageClientSuite) Test_getStandardHeaders(c *chk.C) {
	cli, err := NewBasicClient("foo", "YmFy")
	c.Assert(err, chk.IsNil)

	headers := cli.getStandardHeaders()
	c.Assert(len(headers), chk.Equals, 2)
	c.Assert(headers["x-ms-version"], chk.Equals, cli.apiVersion)
	if _, ok := headers["x-ms-date"]; !ok {
		c.Fatal("Missing date header")
	}
}

func (s *StorageClientSuite) Test_buildCanonicalizedResource(c *chk.C) {
	cli, err := NewBasicClient("foo", "YmFy")
	c.Assert(err, chk.IsNil)

	type test struct{ url, expected string }
	tests := []test{
		{"https://foo.blob.core.windows.net/path?a=b&c=d", "/foo/path\na:b\nc:d"},
		{"https://foo.blob.core.windows.net/?comp=list", "/foo/\ncomp:list"},
		{"https://foo.blob.core.windows.net/cnt/blob", "/foo/cnt/blob"},
	}

	for _, i := range tests {
		out, err := cli.buildCanonicalizedResource(i.url)
		c.Assert(err, chk.IsNil)
		c.Assert(out, chk.Equals, i.expected)
	}
}

func (s *StorageClientSuite) Test_buildCanonicalizedHeader(c *chk.C) {
	cli, err := NewBasicClient("foo", "YmFy")
	c.Assert(err, chk.IsNil)

	type test struct {
		headers  map[string]string
		expected string
	}
	tests := []test{
		{map[string]string{}, ""},
		{map[string]string{"x-ms-foo": "bar"}, "x-ms-foo:bar"},
		{map[string]string{"foo:": "bar"}, ""},
		{map[string]string{"foo:": "bar", "x-ms-foo": "bar"}, "x-ms-foo:bar"},
		{map[string]string{
			"x-ms-version":   "9999-99-99",
			"x-ms-blob-type": "BlockBlob"}, "x-ms-blob-type:BlockBlob\nx-ms-version:9999-99-99"}}

	for _, i := range tests {
		c.Assert(cli.buildCanonicalizedHeader(i.headers), chk.Equals, i.expected)
	}
}

func (s *StorageClientSuite) TestReturnsStorageServiceError(c *chk.C) {
	// attempt to delete a nonexisting container
	_, err := getBlobClient(c).deleteContainer(randContainer())
	c.Assert(err, chk.NotNil)

	v, ok := err.(AzureStorageServiceError)
	c.Check(ok, chk.Equals, true)
	c.Assert(v.StatusCode, chk.Equals, 404)
	c.Assert(v.Code, chk.Equals, "ContainerNotFound")
	c.Assert(v.Code, chk.Not(chk.Equals), "")
}

func (s *StorageClientSuite) Test_createAuthorizationHeader(c *chk.C) {
	key := base64.StdEncoding.EncodeToString([]byte("bar"))
	cli, err := NewBasicClient("foo", key)
	c.Assert(err, chk.IsNil)

	canonicalizedString := `foobarzoo`
	expected := `SharedKey foo:h5U0ATVX6SpbFX1H6GNuxIMeXXCILLoIvhflPtuQZ30=`
	c.Assert(cli.createAuthorizationHeader(canonicalizedString), chk.Equals, expected)
}
