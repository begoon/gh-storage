package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Data struct {
	Content  []byte `json:"-"`
	Metadata map[string]any
}

type Client struct {
	token, contents, raw string
	http                 *http.Client
}

func New(token, account, repo string) *Client {
	repository := url.PathEscape(account) + "/" + url.PathEscape(repo)
	return &Client{token: token, contents: "https://api.github.com/repos/" + repository + "/contents/", raw: "https://raw.githubusercontent.com/" + repository + "/main/", http: &http.Client{Timeout: 30 * time.Second}}
}

func escapePath(path string) string {
	parts := strings.Split(path, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func (c *Client) request(ctx context.Context, method, endpoint string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "gh-storage")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("GitHub returned HTTP %d", res.StatusCode)
	}
	return io.ReadAll(res.Body)
}

func (c *Client) Exist(ctx context.Context, path string) bool {
	_, err := c.request(ctx, "GET", c.contents+escapePath(path), nil)
	return err == nil
}

func (c *Client) Get(ctx context.Context, path string) (*Data, error) {
	body, err := c.request(ctx, "GET", c.contents+escapePath(path), nil)
	if err != nil {
		return nil, err
	}
	var metadata map[string]any
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, err
	}
	content, _ := metadata["content"].(string)
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, err
	}
	delete(metadata, "content")
	return &Data{Content: decoded, Metadata: metadata}, nil
}

func (c *Client) Raw(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, "GET", c.raw+escapePath(path), nil)
}

func (c *Client) Write(ctx context.Context, path, data, action string) error {
	stamp := time.Now().UTC().Format("2006-01-02 15:04:05.000")
	payload := map[string]any{"message": action + " " + stamp + ", " + path}
	method := "PUT"
	if action == "delete" {
		method = "DELETE"
		payload["message"] = "delete " + stamp
	} else {
		payload["content"] = base64.StdEncoding.EncodeToString([]byte(data))
	}
	if action != "create" {
		current, err := c.Get(ctx, path)
		if err != nil {
			return err
		}
		payload["sha"] = current.Metadata["sha"]
	}
	_, err := c.request(ctx, method, c.contents+escapePath(path), payload)
	return err
}
