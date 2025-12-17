package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"paperless2anythingllm/internal/config"
	"paperless2anythingllm/internal/syncer"
	"path/filepath"
	"time"
)

func main() {
	cfgPath := flag.String("config", "config.json", "Path to config file")
	dryRun := flag.Bool("dry-run", false, "Show planned changes without applying them")
	flag.Parse()

	b, err := os.ReadFile(*cfgPath)
	if err != nil {
		fmt.Println("Failed to read config:", err)
		os.Exit(1)
	}

	var cfg config.Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		fmt.Println("Failed to parse config:", err)
		os.Exit(1)
	}

	if cfg.Sync.StateFile == "" {
		cfg.Sync.StateFile = filepath.Join(".", "sync_state.json")
	}
	if cfg.Paperless.PageSize <= 0 {
		cfg.Paperless.PageSize = 100
	}

	start := time.Now()
	err = syncer.Run(cfg, *dryRun)
	if err != nil {
		fmt.Println("Sync failed:", err)
		os.Exit(1)
	}
	fmt.Println("Sync finished in", time.Since(start))
}
