package discovery

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/registry"
	"github.com/hashicorp/terraform/registry/response"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/disco"
	"github.com/mitchellh/cli"
)

const testProviderFile = "test provider binary"

func TestMain(m *testing.M) {
	server := testReleaseServer()
	l, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		log.Fatal(err)
	}

	// NewUnstartedServer creates a listener. Close that listener and replace
	// with the one we created.
	server.Listener.Close()
	server.Listener = l
	server.Start()
	defer server.Close()

	os.Exit(m.Run())
}

// return the directory listing for the "test" provider
func testListingHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 6 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	provider := parts[4]
	if provider == "test" {
		js, err := json.Marshal(versionList)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(js)
	}
	http.Error(w, ErrorNoSuchProvider.Error(), http.StatusNotFound)
	return

}

// return the download URLs for the "test" provider
func testDownloadHandler(w http.ResponseWriter, r *http.Request) {
	js, err := json.Marshal(downloadURLs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

func testChecksumHandler(w http.ResponseWriter, r *http.Request) {
	// this exact plugin has a signature and checksum file
	if r.URL.Path == "/terraform-provider-template/0.1.0/terraform-provider-template_0.1.0_SHA256SUMS" {
		http.ServeFile(w, r, "testdata/terraform-provider-template_0.1.0_SHA256SUMS")
		return
	}
	if r.URL.Path == "/terraform-provider-template/0.1.0/terraform-provider-template_0.1.0_SHA256SUMS.sig" {
		http.ServeFile(w, r, "testdata/terraform-provider-template_0.1.0_SHA256SUMS.sig")
		return
	}

	// this this checksum file is corrupt and doesn't match the sig
	if r.URL.Path == "/terraform-provider-badsig/0.1.0/terraform-provider-badsig_0.1.0_SHA256SUMS" {
		http.ServeFile(w, r, "testdata/terraform-provider-badsig_0.1.0_SHA256SUMS")
		return
	}
	if r.URL.Path == "/terraform-provider-badsig/0.1.0/terraform-provider-badsig_0.1.0_SHA256SUMS.sig" {
		http.ServeFile(w, r, "testdata/terraform-provider-badsig_0.1.0_SHA256SUMS.sig")
		return
	}

	http.Error(w, "signtaure files not found", http.StatusNotFound)
}

// returns a 200 for a valid provider url, using the patch number for the
// plugin protocol version.
func testHandler(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/versions") {
		testListingHandler(w, r)
		return
	}

	if strings.Contains(r.URL.Path, "/download") {
		testDownloadHandler(w, r)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 7 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// write a dummy file
	z := zip.NewWriter(w)
	fn := fmt.Sprintf("%s_v%s", parts[4], parts[5])
	f, err := z.Create(fn)
	if err != nil {
		panic(err)
	}
	io.WriteString(f, testProviderFile)
	z.Close()
}

func testReleaseServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/providers/terraform-providers/", testHandler)
	handler.HandleFunc("/terraform-provider-template/", testChecksumHandler)
	handler.HandleFunc("/terraform-provider-badsig/", testChecksumHandler)
	handler.HandleFunc("/.well-known/terraform.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"modules.v1":"http://localhost/v1/modules/", "providers.v1":"http://localhost/v1/providers/"}`)
	})

	return httptest.NewUnstartedServer(handler)
}

func TestVersionListing(t *testing.T) {
	server := testReleaseServer()
	server.Start()
	defer server.Close()

	i := newProviderInstaller(server)

	allVersions, err := i.listProviderVersions("test")

	if err != nil {
		t.Fatal(err)
	}

	var versions []*response.TerraformProviderVersion

	for _, v := range allVersions.Versions {
		versions = append(versions, v)
	}

	response.ProviderVersionCollection(versions).Sort()

	expected := []*response.TerraformProviderVersion{
		{Version: "1.2.4"},
		{Version: "1.2.3"},
		{Version: "1.2.1"},
	}

	if len(versions) != len(expected) {
		t.Fatalf("Received wrong number of versions. expected: %q, got: %q", expected, versions)
	}

	for i, v := range versions {
		if v.Version != expected[i].Version {
			t.Fatalf("incorrect version: %q, expected %q", v, expected[i])
		}
	}
}

func TestCheckProtocolVersions(t *testing.T) {
	tests := []struct {
		VersionMeta *response.TerraformProviderVersion
		Err         bool
	}{
		{
			&response.TerraformProviderVersion{
				Protocols: []string{"1", "2"},
			},
			true,
		},
		{
			&response.TerraformProviderVersion{
				Protocols: []string{"4"},
			},
			false,
		},
		{
			&response.TerraformProviderVersion{
				Protocols: []string{"4.2"},
			},
			false,
		},
	}

	server := testReleaseServer()
	server.Start()
	defer server.Close()
	i := newProviderInstaller(server)

	for _, test := range tests {
		err := i.checkPluginProtocol(test.VersionMeta)
		if test.Err {
			if err == nil {
				t.Fatal("succeeded; want error")
			}
			return
		} else if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
	}
}

func TestProviderInstallerGet(t *testing.T) {
	server := testReleaseServer()
	server.Start()
	defer server.Close()

	tmpDir, err := ioutil.TempDir("", "tf-plugin")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tmpDir)

	// attempt to use an incompatible protocol version
	i := &ProviderInstaller{
		Dir:                   tmpDir,
		PluginProtocolVersion: 5,
		SkipVerify:            true,
		Ui:                    cli.NewMockUi(),
		registry:              registry.NewClient(Disco(server), nil),
	}
	_, err = i.Get("test", AllVersions)

	if err != ErrorNoVersionCompatibleWithPlatform {
		t.Fatal("want error for incompatible version")
	}

	i = &ProviderInstaller{
		Dir:                   tmpDir,
		PluginProtocolVersion: 4,
		SkipVerify:            true,
		Ui:                    cli.NewMockUi(),
		registry:              registry.NewClient(Disco(server), nil),
	}

	{
		_, err := i.Get("test", ConstraintStr(">9.0.0").MustParse())
		if err != ErrorNoSuitableVersion {
			t.Fatal("want error for mismatching constraints")
		}
	}

	{
		_, err := i.Get("nonexist", AllVersions)
		if err != ErrorNoSuchProvider {
			t.Fatal("want error for no such provider")
		}
	}

	gotMeta, err := i.Get("test", AllVersions)
	if err != nil {
		t.Fatal(err)
	}

	// we should have version 1.2.4
	dest := filepath.Join(tmpDir, "terraform-provider-test_v1.2.4")

	wantMeta := PluginMeta{
		Name:    "test",
		Version: VersionStr("1.2.4"),
		Path:    dest,
	}
	if !reflect.DeepEqual(gotMeta, wantMeta) {
		t.Errorf("wrong result meta\ngot:  %#v\nwant: %#v", gotMeta, wantMeta)
	}

	f, err := ioutil.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}

	// provider should have been unzipped
	if string(f) != testProviderFile {
		t.Fatalf("test provider contains: %q", f)
	}

}

func TestProviderInstallerPurgeUnused(t *testing.T) {
	server := testReleaseServer()
	defer server.Close()

	tmpDir, err := ioutil.TempDir("", "tf-plugin")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(tmpDir)

	unwantedPath := filepath.Join(tmpDir, "terraform-provider-test_v0.0.1_x2")
	wantedPath := filepath.Join(tmpDir, "terraform-provider-test_v1.2.3_x3")

	f, err := os.Create(unwantedPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	f, err = os.Create(wantedPath)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	i := &ProviderInstaller{
		Dir:                   tmpDir,
		PluginProtocolVersion: 3,
		SkipVerify:            true,
		Ui:                    cli.NewMockUi(),
		registry:              registry.NewClient(Disco(server), nil),
	}
	purged, err := i.PurgeUnused(map[string]PluginMeta{
		"test": PluginMeta{
			Name:    "test",
			Version: VersionStr("1.2.3"),
			Path:    wantedPath,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := purged.Count(), 1; got != want {
		t.Errorf("wrong purged count %d; want %d", got, want)
	}
	if got, want := purged.Newest().Path, unwantedPath; got != want {
		t.Errorf("wrong purged path %s; want %s", got, want)
	}

	files, err := ioutil.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	gotFilenames := make([]string, len(files))
	for i, info := range files {
		gotFilenames[i] = info.Name()
	}
	wantFilenames := []string{"terraform-provider-test_v1.2.3_x3"}

	if !reflect.DeepEqual(gotFilenames, wantFilenames) {
		t.Errorf("wrong filenames after purge\ngot:  %#v\nwant: %#v", gotFilenames, wantFilenames)
	}
}

// Test fetching a provider's checksum file while verifying its signature.
func TestProviderChecksum(t *testing.T) {
	tests := []struct {
		URLs *response.TerraformProviderPlatformLocation
		Err  bool
	}{
		{
			&response.TerraformProviderPlatformLocation{
				ShasumsURL:          "http://127.0.0.1:8080/terraform-provider-template/0.1.0/terraform-provider-template_0.1.0_SHA256SUMS",
				ShasumsSignatureURL: "http://127.0.0.1:8080/terraform-provider-template/0.1.0/terraform-provider-template_0.1.0_SHA256SUMS.sig",
				Filename:            "terraform-provider-template_0.1.0_darwin_amd64.zip",
			},
			false,
		},
		{
			&response.TerraformProviderPlatformLocation{
				ShasumsURL:          "http://127.0.0.1:8080/terraform-provider-badsig/0.1.0/terraform-provider-badsig_0.1.0_SHA256SUMS",
				ShasumsSignatureURL: "http://127.0.0.1:8080/terraform-provider-badsig/0.1.0/terraform-provider-badsig_0.1.0_SHA256SUMS.sig",
				Filename:            "terraform-provider-template_0.1.0_darwin_amd64.zip",
			},
			true,
		},
	}

	i := ProviderInstaller{}

	for _, test := range tests {
		sha256sum, err := i.getProviderChecksum(test.URLs)
		if test.Err {
			if err == nil {
				t.Fatal("succeeded; want error")
			}
			return
		} else if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		// get the expected checksum for our os/arch
		sumData, err := ioutil.ReadFile("testdata/terraform-provider-template_0.1.0_SHA256SUMS")
		if err != nil {
			t.Fatal(err)
		}

		expected := checksumForFile(sumData, test.URLs.Filename)

		if sha256sum != expected {
			t.Fatalf("expected: %s\ngot %s\n", sha256sum, expected)
		}
	}
}

// newProviderInstaller returns a minimally-initialized ProviderInstaller
func newProviderInstaller(s *httptest.Server) ProviderInstaller {
	return ProviderInstaller{
		registry: registry.NewClient(Disco(s), nil),
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
	}
}

// Disco return a *disco.Disco mapping registry.terraform.io, localhost,
// localhost.localdomain, and example.com to the test server.
func Disco(s *httptest.Server) *disco.Disco {
	services := map[string]interface{}{
		// Note that both with and without trailing slashes are supported behaviours
		"modules.v1":   fmt.Sprintf("%s/v1/modules", s.URL),
		"providers.v1": fmt.Sprintf("%s/v1/providers", s.URL),
	}
	d := disco.New()

	d.ForceHostServices(svchost.Hostname("registry.terraform.io"), services)
	d.ForceHostServices(svchost.Hostname("localhost"), services)
	d.ForceHostServices(svchost.Hostname("localhost.localdomain"), services)
	d.ForceHostServices(svchost.Hostname("example.com"), services)
	return d
}

var versionList = response.TerraformProvider{
	ID: "test",
	Versions: []*response.TerraformProviderVersion{
		{Version: "1.2.1"},
		{Version: "1.2.3"},
		{
			Version:   "1.2.4",
			Protocols: []string{"4"},
			Platforms: []*response.TerraformProviderPlatform{
				{
					OS:   "darwin",
					Arch: "amd64",
				},
			},
		},
	},
}

var downloadURLs = response.TerraformProviderPlatformLocation{
	ShasumsURL:          "https://registry.terraform.io/terraform-provider-template/1.2.4/terraform-provider-test_1.2.4_SHA256SUMS",
	ShasumsSignatureURL: "https://registry.terraform.io/terraform-provider-template/1.2.4/terraform-provider-test_1.2.4_SHA256SUMS.sig",
	Filename:            "terraform-provider-template_1.2.4_darwin_amd64.zip",
	DownloadURL:         "http://127.0.0.1:8080/v1/providers/terraform-providers/terraform-provider-test/1.2.4/terraform-provider-test_1.2.4_darwin_amd64.zip",
}
