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
	cfgPath := flag.String("config", "config.json", "配置文件路径")
	dryRun := flag.Bool("dry-run", false, "仅显示将执行的操作")
	flag.Parse()

	b, err := os.ReadFile(*cfgPath)
	if err != nil {
		fmt.Println("读取配置失败:", err)
		os.Exit(1)
	}

	var cfg config.Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		fmt.Println("解析配置失败:", err)
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
		fmt.Println("同步失败:", err)
		os.Exit(1)
	}
	fmt.Println("同步完成", time.Since(start))
}
