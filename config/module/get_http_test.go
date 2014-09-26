package module

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestHttpGetter_impl(t *testing.T) {
	var _ Getter = new(HttpGetter)
}

func TestHttpGetter_header(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()

	g := new(HttpGetter)
	dst := tempDir(t)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/header"

	// Get it!
	if err := g.Get(dst, &u); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestHttpGetter_meta(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()

	g := new(HttpGetter)
	dst := tempDir(t)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/meta"

	// Get it!
	if err := g.Get(dst, &u); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "main.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestHttpGetter_metaSubdir(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()

	g := new(HttpGetter)
	dst := tempDir(t)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/meta-subdir"

	// Get it!
	if err := g.Get(dst, &u); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Verify the main file exists
	mainPath := filepath.Join(dst, "sub.tf")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestHttpGetter_none(t *testing.T) {
	ln := testHttpServer(t)
	defer ln.Close()

	g := new(HttpGetter)
	dst := tempDir(t)

	var u url.URL
	u.Scheme = "http"
	u.Host = ln.Addr().String()
	u.Path = "/none"

	// Get it!
	if err := g.Get(dst, &u); err == nil {
		t.Fatal("should error")
	}
}

func testHttpServer(t *testing.T) net.Listener {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/header", testHttpHandlerHeader)
	mux.HandleFunc("/meta", testHttpHandlerMeta)
	mux.HandleFunc("/meta-subdir", testHttpHandlerMetaSubdir)

	var server http.Server
	server.Handler = mux
	go server.Serve(ln)

	return ln
}

func testHttpHandlerHeader(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Terraform-Get", testModuleURL("basic").String())
	w.WriteHeader(200)
}

func testHttpHandlerMeta(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf(testHttpMetaStr, testModuleURL("basic").String())))
}

func testHttpHandlerMetaSubdir(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf(testHttpMetaStr, testModuleURL("basic//subdir").String())))
}

func testHttpHandlerNone(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(testHttpNoneStr))
}

const testHttpMetaStr = `
<html>
<head>
<meta name="terraform-get" content="%s">
</head>
</html>
`

const testHttpNoneStr = `
<html>
<head>
</head>
</html>
`
