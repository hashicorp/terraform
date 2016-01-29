package atlas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type atlasServer struct {
	URL *url.URL

	t      *testing.T
	ln     net.Listener
	server http.Server
}

func newTestAtlasServer(t *testing.T) *atlasServer {
	hs := &atlasServer{t: t}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	hs.ln = ln

	hs.URL = &url.URL{
		Scheme: "http",
		Host:   ln.Addr().String(),
	}

	mux := http.NewServeMux()
	hs.setupRoutes(mux)

	var server http.Server
	server.Handler = mux
	hs.server = server
	go server.Serve(ln)

	return hs
}

func (hs *atlasServer) Stop() {
	hs.ln.Close()
}

func (hs *atlasServer) setupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/_json", hs.jsonHandler)
	mux.HandleFunc("/_rails-error", hs.railsHandler)
	mux.HandleFunc("/_status/", hs.statusHandler)

	mux.HandleFunc("/_binstore/", hs.binstoreHandler)

	mux.HandleFunc("/api/v1/authenticate", hs.authenticationHandler)
	mux.HandleFunc("/api/v1/token", hs.tokenHandler)

	mux.HandleFunc("/api/v1/artifacts/hashicorp/existing", hs.vagrantArtifactExistingHandler)
	mux.HandleFunc("/api/v1/artifacts/hashicorp/existing/amazon-ami", hs.vagrantArtifactUploadHandler)
	mux.HandleFunc("/api/v1/artifacts/hashicorp/existing1/amazon-ami/search", hs.vagrantArtifactSearchHandler1)
	mux.HandleFunc("/api/v1/artifacts/hashicorp/existing2/amazon-ami/search", hs.vagrantArtifactSearchHandler2)

	mux.HandleFunc("/api/v1/vagrant/applications", hs.vagrantCreateAppHandler)
	mux.HandleFunc("/api/v1/vagrant/applications/", hs.vagrantCreateAppsHandler)
	mux.HandleFunc("/api/v1/vagrant/applications/hashicorp/existing", hs.vagrantAppExistingHandler)
	mux.HandleFunc("/api/v1/vagrant/applications/hashicorp/existing/versions", hs.vagrantUploadAppHandler)

	mux.HandleFunc("/api/v1/packer/build-configurations", hs.vagrantBCCreateHandler)
	mux.HandleFunc("/api/v1/packer/build-configurations/hashicorp/existing", hs.vagrantBCExistingHandler)
	mux.HandleFunc("/api/v1/packer/build-configurations/hashicorp/existing/versions", hs.vagrantBCCreateVersionHandler)

	mux.HandleFunc("/api/v1/terraform/configurations/hashicorp/existing/versions/latest", hs.tfConfigLatest)
	mux.HandleFunc("/api/v1/terraform/configurations/hashicorp/existing/versions", hs.tfConfigUpload)
}

func (hs *atlasServer) statusHandler(w http.ResponseWriter, r *http.Request) {
	slice := strings.Split(r.URL.Path, "/")
	codeStr := slice[len(slice)-1]

	code, err := strconv.ParseInt(codeStr, 10, 32)
	if err != nil {
		hs.t.Fatal(err)
	}

	w.WriteHeader(int(code))
}

func (hs *atlasServer) railsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(422)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"errors": ["this is an error", "this is another error"]}`)
}

func (hs *atlasServer) jsonHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ok": true}`)
}

func (hs *atlasServer) authenticationHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		hs.t.Fatal(err)
	}

	login, password := r.Form["user[login]"][0], r.Form["user[password]"][0]

	if login == "sethloves" && password == "bacon" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `
      {
        "token": "pX4AQ5vO7T-xJrxsnvlB0cfeF-tGUX-A-280LPxoryhDAbwmox7PKinMgA1F6R3BKaT"
      }
    `)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
	}
}

func (hs *atlasServer) tokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	token := r.Header.Get(atlasTokenHeader)
	if token == "a.atlasv1.b" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
	}
}

func (hs *atlasServer) tfConfigLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	fmt.Fprintf(w, `
	{
		"version": {
			"version": 5,
			"metadata": { "foo": "bar" },
			"variables": { "foo": "bar" }
		}
	}
	`)
}

func (hs *atlasServer) tfConfigUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r.Body); err != nil {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if buf.Len() == 0 {
		w.WriteHeader(http.StatusConflict)
		return
	}

	uploadPath := hs.URL.String() + "/_binstore/"

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
	{
		"version": 5,
		"upload_path": "%s"
	}
	`, uploadPath)
}

func (hs *atlasServer) vagrantArtifactExistingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	fmt.Fprintf(w, `
	{
		"artifact": {
			"username": "hashicorp",
			"name": "existing",
			"tag": "hashicorp/existing"
		}
	}
	`)
}

func (hs *atlasServer) vagrantArtifactSearchHandler1(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	fmt.Fprintf(w, `
	{
		"versions": [{
			"username": "hashicorp",
			"name": "existing",
			"tag": "hashicorp/existing"
		}]
	}
	`)
}

func (hs *atlasServer) vagrantArtifactSearchHandler2(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Form.Get("metadata.1.key") == "" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.Form.Get("metadata.2.key") == "" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	fmt.Fprintf(w, `
	{
		"versions": [{
			"username": "hashicorp",
			"name": "existing",
			"tag": "hashicorp/existing"
		}]
	}
	`)
}

func (hs *atlasServer) vagrantArtifactUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r.Body); err != nil {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if buf.Len() == 0 {
		w.WriteHeader(http.StatusConflict)
		return
	}

	uploadPath := hs.URL.String() + "/_binstore/"

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
	{
		"upload_path": "%s"
	}
	`, uploadPath)
}

func (hs *atlasServer) vagrantAppExistingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	fmt.Fprintf(w, `
		{
		  "username": "hashicorp",
		  "name": "existing",
		  "tag": "hashicorp/existing",
		  "private": true
		}
	`)
}

func (hs *atlasServer) vagrantBCCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var wrapper bcWrapper
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&wrapper); err != nil && err != io.EOF {
		hs.t.Fatal(err)
	}
	bc := wrapper.BuildConfig

	if bc.User != "hashicorp" {
		w.WriteHeader(http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
    {
       "username":"hashicorp",
       "name":"new",
       "tag":"hashicorp/new",
       "private":true
    }
	`)
}

func (hs *atlasServer) vagrantBCCreateVersionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var wrapper bcCreateWrapper
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&wrapper); err != nil && err != io.EOF {
		hs.t.Fatal(err)
	}
	builds := wrapper.Version.Builds

	if len(builds) == 0 {
		w.WriteHeader(http.StatusConflict)
		return
	}

	expected := map[string]interface{}{"testing": true}
	if !reflect.DeepEqual(wrapper.Version.Metadata, expected) {
		hs.t.Fatalf("expected %q to be %q", wrapper.Version.Metadata, expected)
	}

	uploadPath := hs.URL.String() + "/_binstore/"

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
	{
		"upload_path": "%s"
	}
	`, uploadPath)
}

func (hs *atlasServer) vagrantBCExistingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	fmt.Fprintf(w, `
	{
		"username": "hashicorp",
		"name": "existing"
	}
	`)
}

func (hs *atlasServer) vagrantCreateAppHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var aw appWrapper
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&aw); err != nil && err != io.EOF {
		hs.t.Fatal(err)
	}
	app := aw.Application

	if app.User == "hashicorp" && app.Name == "existing" {
		w.WriteHeader(http.StatusConflict)
	} else {
		body, err := json.Marshal(app)
		if err != nil {
			hs.t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, string(body))
	}
}

func (hs *atlasServer) vagrantCreateAppsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	split := strings.Split(r.RequestURI, "/")
	parts := split[len(split)-2:]
	user, name := parts[0], parts[1]

	if user == "hashicorp" && name == "existing" {
		body, err := json.Marshal(&App{
			User: "hashicorp",
			Name: "existing",
		})
		if err != nil {
			hs.t.Fatal(err)
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, string(body))
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (hs *atlasServer) vagrantUploadAppHandler(w http.ResponseWriter, r *http.Request) {
	u := *hs.URL
	u.Path = path.Join(u.Path, "_binstore/630e42d9-2364-2412-4121-18266770468e")

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r.Body); err != nil {
		hs.t.Fatal(err)
	}
	expected := `{"application":{"metadata":{"testing":true}}}`
	if buf.String() != expected {
		hs.t.Fatalf("expected metadata to be %q, but was %q", expected, buf.String())
	}

	body, err := json.Marshal(&appVersion{
		UploadPath: u.String(),
		Token:      "630e42d9-2364-2412-4121-18266770468e",
		Version:    125,
	})
	if err != nil {
		hs.t.Fatal(err)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, string(body))
}

func (hs *atlasServer) binstoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
}
