package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitHubOperations(t *testing.T) {
	var methods []string
	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Header.Get("Authorization") != "token test-token" || r.Header.Get("User-Agent") != "gh-storage" {
			t.Error("missing GitHub headers")
		}
		if r.URL.EscapedPath() != "/contents/folder/a%20%23%3F.txt" {
			t.Errorf("path not escaped: %s", r.URL.EscapedPath())
		}
		if r.Method == "GET" {
			_, _ = w.Write([]byte(`{"content":"aGVs\nbG8=\n","sha":"abc","size":5,"name":"a #?.txt","_links":{"self":"url"}}`))
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	c := &Client{token: "test-token", contents: server.URL + "/contents/", http: server.Client()}
	ctx := context.Background()
	path := "folder/a #?.txt"
	data, err := c.Get(ctx, path)
	if err != nil || string(data.Content) != "hello" || data.Metadata["sha"] != "abc" {
		t.Fatalf("Get: %v %v", data, err)
	}
	if _, ok := data.Metadata["content"]; ok {
		t.Fatal("content must not be in metadata")
	}
	for _, action := range []string{"create", "commit", "delete"} {
		methods = nil
		payload = nil
		if err := c.Write(ctx, path, "héllo", action); err != nil {
			t.Fatal(err)
		}
		if action == "create" {
			if len(methods) != 1 || payload["sha"] != nil {
				t.Fatal("create must not fetch or send sha")
			}
		} else if len(methods) != 2 || payload["sha"] != "abc" {
			t.Fatal("update/delete must fetch sha")
		}
		if action == "delete" {
			if methods[len(methods)-1] != "DELETE" || payload["content"] != nil {
				t.Fatal("invalid delete")
			}
		} else if payload["content"] != base64.StdEncoding.EncodeToString([]byte("héllo")) {
			t.Fatal("invalid UTF-8 encoding")
		}
	}
}

func TestFailureAndCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	defer server.Close()
	c := &Client{contents: server.URL + "/", raw: server.URL + "/", http: &http.Client{Timeout: time.Second}}
	ctx := context.Background()
	if c.Exist(ctx, "missing") {
		t.Fatal("missing file exists")
	}
	if _, err := c.Get(ctx, "missing"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.Raw(ctx, "missing"); err == nil {
		t.Fatal("expected error")
	}
	if err := c.Write(ctx, "missing", "", "commit"); err == nil {
		t.Fatal("must fail when sha unavailable")
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := c.Get(ctx, "file"); err == nil {
		t.Fatal("cancelled request succeeded")
	}
}
