package paperless

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	BaseURL  string
	Token    string
	PageSize int
	hc       *http.Client
}

type Document struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Modified    string     `json:"modified"`
	Created     string     `json:"created"`
	DownloadURL string     `json:"download_url"`
	StoragePath *StorePath `json:"storage_path"`
	Tags        []Tag      `json:"tags"`
}

type Tag struct {
	Name string `json:"name"`
}

type StorePath struct {
	ID    int    `json:"id"`
	Path  string `json:"path"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

func New(baseURL, token string, pageSize int) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Token: token, PageSize: pageSize, hc: &http.Client{Timeout: 60 * time.Second}}
}

func (c *Client) ListDocuments() ([]Document, error) {
	var out []Document
	u := fmt.Sprintf("%s/api/documents/?page_size=%d", c.BaseURL, c.PageSize)
	for {
		req, _ := http.NewRequest("GET", u, nil)
		req.Header.Set("Authorization", "Token "+c.Token)
		resp, err := c.hc.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("paperless list failed: %s", string(b))
		}
		var raw struct {
			Count    int               `json:"count"`
			Next     *string           `json:"next"`
			Previous *string           `json:"previous"`
			Results  []json.RawMessage `json:"results"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			return nil, err
		}
		for _, r := range raw.Results {
			doc, err := decodeDocument(r)
			if err != nil {
				continue
			}
			out = append(out, doc)
		}
		if raw.Next == nil || *raw.Next == "" {
			break
		}
		u = *raw.Next
	}
	return out, nil
}

func (c *Client) Download(doc Document) (string, error) {
	dl := doc.DownloadURL
	if dl == "" {
		dl = fmt.Sprintf("%s/api/documents/%d/download/", c.BaseURL, doc.ID)
	}
	if !strings.HasPrefix(dl, "http") {
		dl = strings.TrimRight(c.BaseURL, "/") + "/" + strings.TrimLeft(dl, "/")
	}
	req, _ := http.NewRequest("GET", dl, nil)
	req.Header.Set("Authorization", "Token "+c.Token)
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", errors.New(string(b))
	}
	fn := fmt.Sprintf("paperless-%d", doc.ID)
	tmp := filepath.Join(os.TempDir(), fn)
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return "", err
	}
	return tmp, nil
}

func GroupKey(doc Document, grouping string, defaultName string) string {
	switch grouping {
	case "storage_path":
		if doc.StoragePath != nil {
			if doc.StoragePath.Path != "" {
				return doc.StoragePath.Path
			}
			if doc.StoragePath.Title != "" {
				return doc.StoragePath.Title
			}
		}
	case "tag":
		if len(doc.Tags) > 0 {
			return doc.Tags[0].Name
		}
	}
	return defaultName
}

func ParseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05Z07:00"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func JoinURL(base, p string) string {
	b := strings.TrimRight(base, "/")
	u, err := url.Parse(b)
	if err != nil {
		return b + "/" + strings.TrimLeft(p, "/")
	}
	return u.Scheme + "://" + u.Host + "/" + strings.TrimLeft(p, "/")
}

func decodeDocument(data []byte) (Document, error) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return Document{}, err
	}
	var d Document
	if v, ok := m["id"].(float64); ok {
		d.ID = int(v)
	}
	if v, ok := m["title"].(string); ok {
		d.Title = v
	}
	if v, ok := m["modified"].(string); ok {
		d.Modified = v
	}
	if v, ok := m["created"].(string); ok {
		d.Created = v
	}
	if v, ok := m["download_url"].(string); ok {
		d.DownloadURL = v
	}
	if sp, ok := m["storage_path"].(map[string]any); ok {
		var s StorePath
		if v, ok := sp["id"].(float64); ok {
			s.ID = int(v)
		}
		if v, ok := sp["path"].(string); ok {
			s.Path = v
		}
		if v, ok := sp["title"].(string); ok {
			s.Title = v
		}
		if v, ok := sp["slug"].(string); ok {
			s.Slug = v
		}
		d.StoragePath = &s
	}
	if tags, ok := m["tags"].([]any); ok {
		for _, t := range tags {
			if tm, ok := t.(map[string]any); ok {
				var tag Tag
				if v, ok := tm["name"].(string); ok {
					tag.Name = v
				}
				if tag.Name != "" {
					d.Tags = append(d.Tags, tag)
				}
			}
		}
	}
	return d, nil
}
