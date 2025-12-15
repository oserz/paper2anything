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
	BaseURL string
	APIKey  string
	hc      *http.Client
}

type workspaceResp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type uploadResp struct {
	DocumentURL string `json:"document_url"`
	Path        string `json:"path"`
}

func New(baseURL, apiKey string) *Client {
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, hc: &http.Client{Timeout: 60 * time.Second}}
}

func (c *Client) EnsureWorkspace(name, slug string) (string, error) {
	u := c.BaseURL + "/api/v1/workspace/" + slug
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	resp, err := c.hc.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		var raw map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err == nil {
			if v, ok := raw["slug"].(string); ok && v != "" {
				return v, nil
			}
		}
	}
	if resp != nil {
		resp.Body.Close()
	}
	body := map[string]string{"name": name, "slug": slug}
	b, _ := json.Marshal(body)
	req2, _ := http.NewRequest("POST", c.BaseURL+"/api/v1/workspace", bytes.NewReader(b))
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
		if v, ok := raw["slug"].(string); ok && v != "" {
			return v, nil
		}
	}
	return slug, nil
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
