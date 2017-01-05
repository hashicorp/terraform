// Package pat implements a simple URL pattern muxer
package pat

import (
	"net/http"
	"net/url"
	"strings"
)

// PatternServeMux is an HTTP request multiplexer. It matches the URL of each
// incoming request against a list of registered patterns with their associated
// methods and calls the handler for the pattern that most closely matches the
// URL.
//
// Pattern matching attempts each pattern in the order in which they were
// registered.
//
// Patterns may contain literals or captures. Capture names start with a colon
// and consist of letters A-Z, a-z, _, and 0-9. The rest of the pattern
// matches literally. The portion of the URL matching each name ends with an
// occurrence of the character in the pattern immediately following the name,
// or a /, whichever comes first. It is possible for a name to match the empty
// string.
//
// Example pattern with one capture:
//   /hello/:name
// Will match:
//   /hello/blake
//   /hello/keith
// Will not match:
//   /hello/blake/
//   /hello/blake/foo
//   /foo
//   /foo/bar
//
// Example 2:
//    /hello/:name/
// Will match:
//   /hello/blake/
//   /hello/keith/foo
//   /hello/blake
//   /hello/keith
// Will not match:
//   /foo
//   /foo/bar
//
// A pattern ending with a slash will add an implicit redirect for its non-slash
// version. For example: Get("/foo/", handler) also registers
// Get("/foo", handler) as a redirect. You may override it by registering
// Get("/foo", anotherhandler) before the slash version.
//
// Retrieve the capture from the r.URL.Query().Get(":name") in a handler (note
// the colon). If a capture name appears more than once, the additional values
// are appended to the previous values (see
// http://golang.org/pkg/net/url/#Values)
//
// A trivial example server is:
//
//	package main
//
//	import (
//		"io"
//		"net/http"
//		"github.com/bmizerany/pat"
//		"log"
//	)
//
//	// hello world, the web server
//	func HelloServer(w http.ResponseWriter, req *http.Request) {
//		io.WriteString(w, "hello, "+req.URL.Query().Get(":name")+"!\n")
//	}
//
//	func main() {
//		m := pat.New()
//		m.Get("/hello/:name", http.HandlerFunc(HelloServer))
//
//		// Register this pat with the default serve mux so that other packages
//		// may also be exported. (i.e. /debug/pprof/*)
//		http.Handle("/", m)
//		err := http.ListenAndServe(":12345", nil)
//		if err != nil {
//			log.Fatal("ListenAndServe: ", err)
//		}
//	}
//
// When "Method Not Allowed":
//
// Pat knows what methods are allowed given a pattern and a URI. For
// convenience, PatternServeMux will add the Allow header for requests that
// match a pattern for a method other than the method requested and set the
// Status to "405 Method Not Allowed".
//
// If the NotFound handler is set, then it is used whenever the pattern doesn't
// match the request path for the current method (and the Allow header is not
// altered).
type PatternServeMux struct {
	// NotFound, if set, is used whenever the request doesn't match any
	// pattern for its method. NotFound should be set before serving any
	// requests.
	NotFound http.Handler
	handlers map[string][]*patHandler
}

// New returns a new PatternServeMux.
func New() *PatternServeMux {
	return &PatternServeMux{handlers: make(map[string][]*patHandler)}
}

// ServeHTTP matches r.URL.Path against its routing table using the rules
// described above.
func (p *PatternServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, ph := range p.handlers[r.Method] {
		if params, ok := ph.try(r.URL.Path); ok {
			if len(params) > 0 && !ph.redirect {
				r.URL.RawQuery = url.Values(params).Encode() + "&" + r.URL.RawQuery
			}
			ph.ServeHTTP(w, r)
			return
		}
	}

	if p.NotFound != nil {
		p.NotFound.ServeHTTP(w, r)
		return
	}

	allowed := make([]string, 0, len(p.handlers))
	for meth, handlers := range p.handlers {
		if meth == r.Method {
			continue
		}

		for _, ph := range handlers {
			if _, ok := ph.try(r.URL.Path); ok {
				allowed = append(allowed, meth)
			}
		}
	}

	if len(allowed) == 0 {
		http.NotFound(w, r)
		return
	}

	w.Header().Add("Allow", strings.Join(allowed, ", "))
	http.Error(w, "Method Not Allowed", 405)
}

// Head will register a pattern with a handler for HEAD requests.
func (p *PatternServeMux) Head(pat string, h http.Handler) {
	p.Add("HEAD", pat, h)
}

// Get will register a pattern with a handler for GET requests.
// It also registers pat for HEAD requests. If this needs to be overridden, use
// Head before Get with pat.
func (p *PatternServeMux) Get(pat string, h http.Handler) {
	p.Add("HEAD", pat, h)
	p.Add("GET", pat, h)
}

// Post will register a pattern with a handler for POST requests.
func (p *PatternServeMux) Post(pat string, h http.Handler) {
	p.Add("POST", pat, h)
}

// Put will register a pattern with a handler for PUT requests.
func (p *PatternServeMux) Put(pat string, h http.Handler) {
	p.Add("PUT", pat, h)
}

// Del will register a pattern with a handler for DELETE requests.
func (p *PatternServeMux) Del(pat string, h http.Handler) {
	p.Add("DELETE", pat, h)
}

// Options will register a pattern with a handler for OPTIONS requests.
func (p *PatternServeMux) Options(pat string, h http.Handler) {
	p.Add("OPTIONS", pat, h)
}

// Patch will register a pattern with a handler for PATCH requests.
func (p *PatternServeMux) Patch(pat string, h http.Handler) {
	p.Add("PATCH", pat, h)
}

// Add will register a pattern with a handler for meth requests.
func (p *PatternServeMux) Add(meth, pat string, h http.Handler) {
	p.add(meth, pat, h, false)
}

func (p *PatternServeMux) add(meth, pat string, h http.Handler, redirect bool) {
	handlers := p.handlers[meth]
	for _, p1 := range handlers {
		if p1.pat == pat {
			return // found existing pattern; do nothing
		}
	}
	handler := &patHandler{
		pat:      pat,
		Handler:  h,
		redirect: redirect,
	}
	p.handlers[meth] = append(handlers, handler)

	n := len(pat)
	if n > 0 && pat[n-1] == '/' {
		p.add(meth, pat[:n-1], http.HandlerFunc(addSlashRedirect), true)
	}
}

func addSlashRedirect(w http.ResponseWriter, r *http.Request) {
	u := *r.URL
	u.Path += "/"
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

// Tail returns the trailing string in path after the final slash for a pat ending with a slash.
//
// Examples:
//
//	Tail("/hello/:title/", "/hello/mr/mizerany") == "mizerany"
//	Tail("/:a/", "/x/y/z")                       == "y/z"
//
func Tail(pat, path string) string {
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(pat):
			if pat[len(pat)-1] == '/' {
				return path[i:]
			}
			return ""
		case pat[j] == ':':
			var nextc byte
			_, nextc, j = match(pat, isAlnum, j+1)
			_, _, i = match(path, matchPart(nextc), i)
		case path[i] == pat[j]:
			i++
			j++
		default:
			return ""
		}
	}
	return ""
}

type patHandler struct {
	pat string
	http.Handler
	redirect bool
}

func (ph *patHandler) try(path string) (url.Values, bool) {
	p := make(url.Values)
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(ph.pat):
			if ph.pat != "/" && len(ph.pat) > 0 && ph.pat[len(ph.pat)-1] == '/' {
				return p, true
			}
			return nil, false
		case ph.pat[j] == ':':
			var name, val string
			var nextc byte
			name, nextc, j = match(ph.pat, isAlnum, j+1)
			val, _, i = match(path, matchPart(nextc), i)
			p.Add(":"+name, val)
		case path[i] == ph.pat[j]:
			i++
			j++
		default:
			return nil, false
		}
	}
	if j != len(ph.pat) {
		return nil, false
	}
	return p, true
}

func matchPart(b byte) func(byte) bool {
	return func(c byte) bool {
		return c != b && c != '/'
	}
}

func match(s string, f func(byte) bool, i int) (matched string, next byte, j int) {
	j = i
	for j < len(s) && f(s[j]) {
		j++
	}
	if j < len(s) {
		next = s[j]
	}
	return s[i:j], next, j
}

func isAlpha(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isAlnum(ch byte) bool {
	return isAlpha(ch) || isDigit(ch)
}
