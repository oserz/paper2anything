package config

type Config struct {
	Paperless   PaperlessConfig   `json:"paperless"`
	AnythingLLM AnythingLLMConfig `json:"anythingllm"`
	Sync        SyncConfig        `json:"sync"`
}

type PaperlessConfig struct {
	BaseURL  string `json:"base_url"`
	Token    string `json:"token"`
	PageSize int    `json:"page_size"`
}

type AnythingLLMConfig struct {
	BaseURL      string `json:"base_url"`
	APIKey       string `json:"api_key"`
	UploadMethod string `json:"upload_method"`
}

type SyncConfig struct {
	DefaultWorkspace string `json:"default_workspace"`
	StateFile        string `json:"state_file"`
}
