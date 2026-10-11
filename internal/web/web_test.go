package web

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"gh-storage/internal/github"
)

type store struct {
	calls              int
	fail               bool
	action, path, body string
}

func (s *store) Exist(context.Context, string) bool { s.calls++; return !s.fail }
func (s *store) Get(context.Context, string) (*github.Data, error) {
	s.calls++
	if s.fail {
		return nil, errors.New("missing")
	}
	return &github.Data{Content: []byte{'h', 'i', 0xff}, Metadata: map[string]any{"sha": "abc", "size": 3}}, nil
}
func (s *store) Raw(context.Context, string) ([]byte, error) {
	s.calls++
	if s.fail {
		return nil, errors.New("missing")
	}
	return []byte{'h', 'i', 0xff}, nil
}
func (s *store) Write(_ context.Context, path, body, action string) error {
	s.calls++
	s.path, s.body, s.action = path, body, action
	if s.fail {
		return errors.New("failed")
	}
	return nil
}

func TestRoutes(t *testing.T) {
	for _, tc := range []struct {
		method, path, auth, body string
		code                     int
		want                     string
		fail                     bool
	}{
		{"GET", "/", "", "", 200, "ha?", false},
		{"GET", "/health", "", "", 200, `{"version":"0.0.0"}`, false},
		{"GET", "/unknown", "", "", 404, "404 Not Found", false},
		{"GET", "/data/a/b", "secret", "", 200, "hi�", false},
		{"GET", "/data/a/b?binary=1", "secret", "", 200, string([]byte{'h', 'i', 255}), false},
		{"GET", "/data/a/b?binary=", "secret", "", 200, "hi�", false},
		{"GET", "/raw/a/b", "secret", "", 200, "hi�", false},
		{"GET", "/data/missing", "secret", "", 404, "404 Not Found", true},
		{"OPTIONS", "/data/a", "secret", "", 200, `{"sha":"abc","size":3}`, false},
		{"POST", "/data/a/b", "secret", "hello", 200, "created", false},
		{"PUT", "/data/a/b", "secret", "hello", 200, "updated", false},
		{"DELETE", "/data/a/b", "secret", "", 200, "deleted", false},
		{"POST", "/data/a", "secret", "", 500, "failed to create", true},
		{"PUT", "/data/a", "secret", "", 500, "failed to update", true},
		{"DELETE", "/data/a", "secret", "", 500, "failed to delete", true},
	} {
		t.Run(tc.method+tc.path+tc.want, func(t *testing.T) {
			s := &store{fail: tc.fail}
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			req.Header.Set("ME", tc.auth)
			res := httptest.NewRecorder()
			New(s, "secret").ServeHTTP(res, req)
			if res.Code != tc.code || res.Body.String() != tc.want {
				t.Fatalf("got %d %q, want %d %q", res.Code, res.Body.String(), tc.code, tc.want)
			}
			if s.action != "" && (s.path != strings.TrimPrefix(tc.path, "/data/") || s.body != tc.body) {
				t.Fatal("write payload/path mismatch")
			}
		})
	}
}

func TestAuthentication(t *testing.T) {
	for _, method := range []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "DELETE"} {
		for _, key := range []string{"", "wrong"} {
			s := &store{}
			req := httptest.NewRequest(method, "/data/file", nil)
			req.Header.Set("ME", key)
			res := httptest.NewRecorder()
			New(s, "secret").ServeHTTP(res, req)
			if res.Code != 403 || s.calls != 0 {
				t.Fatalf("%s must reject before accessing GitHub", method)
			}
		}
	}
	s := &store{}
	res := httptest.NewRecorder()
	New(s, "secret").ServeHTTP(res, httptest.NewRequest("GET", "/raw/file", nil))
	if res.Code != 403 || s.calls != 0 {
		t.Fatal("raw requires auth")
	}
}

func TestStatusAndHead(t *testing.T) {
	for _, fail := range []bool{false, true} {
		s := &store{fail: fail}
		handler := New(s, "secret")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, httptest.NewRequest("GET", "/status", nil))
		want := `"ok":true`
		if fail {
			want = `"ok":false`
		}
		if res.Code != 200 || !strings.Contains(res.Body.String(), want) || s.calls != 3 {
			t.Fatalf("status: %s", res.Body)
		}
		req := httptest.NewRequest("HEAD", "/data/file", nil)
		req.Header.Set("ME", "secret")
		res = httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		code := 200
		if fail {
			code = 404
		}
		if res.Code != code {
			t.Fatalf("head: %d", res.Code)
		}
	}
}
