package anything

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	BaseURL          string
	APIKey           string
	hc               *http.Client
	workspacesLoaded bool
	workspacesByName map[string]workspaceInfo
}

type workspaceResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type workspaceInfo struct {
	ID   int
	Name string
	Slug string
}

type uploadResp struct {
	DocumentURL string `json:"document_url"`
	Path        string `json:"path"`
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:          strings.TrimRight(baseURL, "/"),
		APIKey:           apiKey,
		hc:               &http.Client{Timeout: 60 * time.Second},
		workspacesLoaded: false,
		workspacesByName: map[string]workspaceInfo{},
	}
}

func (c *Client) EnsureWorkspace(name, slug string) (string, error) {
	if err := c.loadWorkspaces(); err != nil {
		return "", err
	}
	if ws, ok := c.workspacesByName[name]; ok && ws.Slug != "" {
		return ws.Slug, nil
	}
	body := map[string]any{
		"name":                 name,
		"similarityThreshold":  0.7,
		"openAiTemp":           0.7,
		"openAiHistory":        20,
		"chatMode":             "chat",
		"topN":                 4,
		"queryRefusalResponse": "",
		"openAiPrompt":         nil,
	}
	b, _ := json.Marshal(body)
	req2, _ := http.NewRequest("POST", c.BaseURL+"/api/v1/workspace/new", bytes.NewReader(b))
	req2.Header.Set("Authorization", "Bearer "+c.APIKey)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := c.hc.Do(req2)
	if err != nil {
		return "", err
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK && resp2.StatusCode != http.StatusCreated {
		x, _ := io.ReadAll(resp2.Body)
		return "", fmt.Errorf("create workspace failed: %s", string(x))
	}
	var raw map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&raw); err == nil {
		if w, ok := raw["workspace"].(map[string]any); ok {
			var info workspaceInfo
			if v, ok := w["id"].(float64); ok {
				info.ID = int(v)
			}
			if v, ok := w["name"].(string); ok {
				info.Name = v
			} else {
				info.Name = name
			}
			if v, ok := w["slug"].(string); ok {
				info.Slug = v
			} else {
				info.Slug = slug
			}
			if info.Name != "" {
				c.workspacesByName[info.Name] = info
			}
			if info.Slug != "" {
				return info.Slug, nil
			}
		}
	}
	if slug != "" {
		return slug, nil
	}
	return name, nil
}

func (c *Client) UploadDocument(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", err
	}
	mw.Close()

	req, _ := http.NewRequest("POST", c.BaseURL+"/api/v1/document/upload", &buf)
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		x, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed: %s", string(x))
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err == nil {
		if v, ok := raw["document_url"].(string); ok && v != "" {
			return v, nil
		}
		if v, ok := raw["path"].(string); ok && v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("no document url returned")
}

func (c *Client) loadWorkspaces() error {
	if c.workspacesLoaded {
		return nil
	}
	u := c.BaseURL + "/api/v1/workspaces"
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		x, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("list workspaces failed: %s", string(x))
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return err
	}
	ws, ok := raw["workspaces"].([]any)
	if !ok {
		c.workspacesLoaded = true
		return nil
	}
	for _, item := range ws {
		if m, ok := item.(map[string]any); ok {
			var info workspaceInfo
			if v, ok := m["id"].(float64); ok {
				info.ID = int(v)
			}
			if v, ok := m["name"].(string); ok {
				info.Name = v
			}
			if v, ok := m["slug"].(string); ok {
				info.Slug = v
			}
			if info.Name != "" {
				c.workspacesByName[info.Name] = info
			}
		}
	}
	c.workspacesLoaded = true
	return nil
}

func (c *Client) UpdateEmbeddings(slug string, adds, removes []string) error {
	body := map[string][]string{"adds": adds, "removes": removes}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.BaseURL+"/api/v1/workspace/"+slug+"/update-embeddings", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		x, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update embeddings failed: %s", string(x))
	}
	return nil
}
