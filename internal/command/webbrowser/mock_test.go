package webbrowser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMockLauncher(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Length", "0")
		switch req.URL.Path {
		case "/standard-redirect-source":
			resp.Header().Set("Location", "/standard-redirect-target")
			resp.WriteHeader(302)
		case "/custom-redirect-source":
			resp.Header().Set("X-Redirect-To", "/custom-redirect-target")
			resp.WriteHeader(200)
		case "/error":
			resp.WriteHeader(500)
		default:
			resp.WriteHeader(200)
		}
	}))
	defer s.Close()

	t.Run("no redirects", func(t *testing.T) {
		l := NewMockLauncher(context.Background())
		err := l.OpenURL(s.URL)
		if err != nil {
			t.Fatal(err)
		}
		l.Wait() // Let the async work complete
		if got, want := len(l.Responses), 1; got != want {
			t.Fatalf("wrong number of responses %d; want %d", got, want)
		}
		if got, want := l.Responses[0].Request.URL.Path, ""; got != want {
			t.Fatalf("wrong request URL %q; want %q", got, want)
		}
	})
	t.Run("error", func(t *testing.T) {
		l := NewMockLauncher(context.Background())
		err := l.OpenURL(s.URL + "/error")
		if err != nil {
			// Th is kind of error is supposed to happen asynchronously, so we
			// should not see it here.
			t.Fatal(err)
		}
		l.Wait() // Let the async work complete
		if got, want := len(l.Responses), 1; got != want {
			t.Fatalf("wrong number of responses %d; want %d", got, want)
		}
		if got, want := l.Responses[0].Request.URL.Path, "/error"; got != want {
			t.Fatalf("wrong request URL %q; want %q", got, want)
		}
		if got, want := l.Responses[0].StatusCode, 500; got != want {
			t.Fatalf("wrong response status %d; want %d", got, want)
		}
	})
	t.Run("standard redirect", func(t *testing.T) {
		l := NewMockLauncher(context.Background())
		err := l.OpenURL(s.URL + "/standard-redirect-source")
		if err != nil {
			t.Fatal(err)
		}
		l.Wait() // Let the async work complete
		if got, want := len(l.Responses), 2; got != want {
			t.Fatalf("wrong number of responses %d; want %d", got, want)
		}
		if got, want := l.Responses[0].Request.URL.Path, "/standard-redirect-source"; got != want {
			t.Fatalf("wrong request 0 URL %q; want %q", got, want)
		}
		if got, want := l.Responses[1].Request.URL.Path, "/standard-redirect-target"; got != want {
			t.Fatalf("wrong request 1 URL %q; want %q", got, want)
		}
	})
	t.Run("custom redirect", func(t *testing.T) {
		l := NewMockLauncher(context.Background())
		err := l.OpenURL(s.URL + "/custom-redirect-source")
		if err != nil {
			t.Fatal(err)
		}
		l.Wait() // Let the async work complete
		if got, want := len(l.Responses), 2; got != want {
			t.Fatalf("wrong number of responses %d; want %d", got, want)
		}
		if got, want := l.Responses[0].Request.URL.Path, "/custom-redirect-source"; got != want {
			t.Fatalf("wrong request 0 URL %q; want %q", got, want)
		}
		if got, want := l.Responses[1].Request.URL.Path, "/custom-redirect-target"; got != want {
			t.Fatalf("wrong request 1 URL %q; want %q", got, want)
		}
	})
}
