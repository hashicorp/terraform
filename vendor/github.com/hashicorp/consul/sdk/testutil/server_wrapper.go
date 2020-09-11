package testutil

import "testing"

type WrappedServer struct {
	s *TestServer
	t *testing.T
}

// Wrap wraps the test server in a `testing.t` for convenience.
//
// For example, the following code snippets are equivalent.
//
//   server.JoinLAN(t, "1.2.3.4")
//   server.Wrap(t).JoinLAN("1.2.3.4")
//
// This is useful when you are calling multiple functions and save the wrapped
// value as another variable to reduce the inclusion of "t".
func (s *TestServer) Wrap(t *testing.T) *WrappedServer {
	return &WrappedServer{s, t}
}

func (w *WrappedServer) JoinLAN(addr string) {
	w.s.JoinLAN(w.t, addr)
}

func (w *WrappedServer) JoinWAN(addr string) {
	w.s.JoinWAN(w.t, addr)
}

func (w *WrappedServer) SetKV(key string, val []byte) {
	w.s.SetKV(w.t, key, val)
}

func (w *WrappedServer) SetKVString(key string, val string) {
	w.s.SetKVString(w.t, key, val)
}

func (w *WrappedServer) GetKV(key string) []byte {
	return w.s.GetKV(w.t, key)
}

func (w *WrappedServer) GetKVString(key string) string {
	return w.s.GetKVString(w.t, key)
}

func (w *WrappedServer) PopulateKV(data map[string][]byte) {
	w.s.PopulateKV(w.t, data)
}

func (w *WrappedServer) ListKV(prefix string) []string {
	return w.s.ListKV(w.t, prefix)
}

func (w *WrappedServer) AddService(name, status string, tags []string) {
	w.s.AddService(w.t, name, status, tags)
}

func (w *WrappedServer) AddAddressableService(name, status, address string, port int, tags []string) {
	w.s.AddAddressableService(w.t, name, status, address, port, tags)
}

func (w *WrappedServer) AddCheck(name, serviceID, status string) {
	w.s.AddCheck(w.t, name, serviceID, status)
}
