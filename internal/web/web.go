package web

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"gh-storage/internal/github"
)

const Version = "0.0.0"

type Storage interface {
	Exist(context.Context, string) bool
	Get(context.Context, string) (*github.Data, error)
	Raw(context.Context, string) ([]byte, error)
	Write(context.Context, string, string, string) error
}

func text(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func jsonResponse(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	body, err := json.Marshal(value)
	if err != nil {
		text(w, 500, "Internal Server Error")
		return
	}
	_, _ = w.Write(body)
}

// Decode invalid UTF-8 the same way as the original text endpoints.
func decode(data []byte) string { return string([]rune(string(data))) }

func New(gh Storage, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		path := r.URL.Path
		read := r.Method == "GET" || r.Method == "HEAD"
		if read {
			switch path {
			case "/":
				text(w, 200, "ha?")
				return
			case "/health":
				jsonResponse(w, map[string]any{"version": Version})
				return
			case "/status":
				status(w, r, gh)
				return
			}
		}
		raw := strings.HasPrefix(path, "/raw/") && len(path) > len("/raw/")
		data := strings.HasPrefix(path, "/data/") && len(path) > len("/data/")
		allowed := read || r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" || r.Method == "OPTIONS"
		if (!data || !allowed) && (!raw || !read) {
			text(w, 404, "404 Not Found")
			return
		}
		if secret == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("ME")), []byte(secret)) != 1 {
			text(w, 403, "forbidden")
			return
		}
		if raw {
			body, err := gh.Raw(r.Context(), strings.TrimPrefix(path, "/raw/"))
			if err != nil || len(body) == 0 {
				text(w, 404, "404 Not Found")
				return
			}
			text(w, 200, decode(body))
			return
		}
		path = strings.TrimPrefix(path, "/data/")
		switch r.Method {
		case "HEAD":
			if !gh.Exist(r.Context(), path) {
				text(w, 404, "404 Not Found")
				return
			}
			text(w, 200, "exist")
		case "GET", "OPTIONS":
			entry, err := gh.Get(r.Context(), path)
			if err != nil {
				text(w, 404, "404 Not Found")
				return
			}
			if r.Method == "OPTIONS" {
				jsonResponse(w, entry.Metadata)
				return
			}
			if r.URL.Query().Get("binary") != "" {
				w.Header().Set("Content-Type", "application/octet-stream")
				_, _ = w.Write(entry.Content)
			} else {
				text(w, 200, decode(entry.Content))
			}
		default:
			action, reply := "delete", "deleted"
			var body []byte
			if r.Method != "DELETE" {
				var err error
				body, err = io.ReadAll(r.Body)
				if err != nil {
					text(w, 400, "failed to read body")
					return
				}
				if r.Method == "POST" {
					action, reply = "create", "created"
				} else {
					action, reply = "commit", "updated"
				}
			}
			if err := gh.Write(r.Context(), path, decode(body), action); err != nil {
				log.Printf("%s failed: %v", r.Method, err)
				verb := map[string]string{"POST": "create", "PUT": "update", "DELETE": "delete"}[r.Method]
				text(w, 500, "failed to "+verb)
				return
			}
			text(w, 200, reply)
		}
	})
}

func status(w http.ResponseWriter, r *http.Request, gh Storage) {
	start := time.Now()
	gh.Exist(r.Context(), "README.md")
	exist := float64(time.Since(start)) / float64(time.Millisecond)
	start = time.Now()
	entry, readErr := gh.Get(r.Context(), "README.md")
	read := float64(time.Since(start)) / float64(time.Millisecond)
	start = time.Now()
	body, rawErr := gh.Raw(r.Context(), "README.md")
	raw := float64(time.Since(start)) / float64(time.Millisecond)
	ok := readErr == nil && rawErr == nil && decode(entry.Content) == decode(body)
	header := "?"
	if rawErr == nil {
		header = strings.SplitN(decode(body), "\n", 2)[0]
	}
	jsonResponse(w, map[string]any{"version": Version, "exist": exist, "read": read, "raw": raw, "ok": ok, "header": header})
}
