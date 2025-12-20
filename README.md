# paperless2anythingllm

[English](#english) | [中文](#中文)

## English

Sync Paperless-ngx documents into AnythingLLM workspaces based on Paperless tags.

### What it does

- Fetches documents from Paperless
- Uses each document's tags as workspace names (multi-tag → multi-workspace)
- Ensures the corresponding workspaces exist in AnythingLLM
- Uploads the document into each target workspace and updates embeddings
- Performs incremental sync using `sync_state.json`
- Physically deletes old AnythingLLM documents when they are replaced or removed
- Supports full cleanup with `-clear-anything` (documents + workspaces)

### Configuration

Create `config.json` in the project root:

```json
{
  "paperless": {
    "base_url": "http://paperless.local",
    "token": "YOUR_PAPERLESS_TOKEN",
    "page_size": 100
  },
  "anythingllm": {
    "base_url": "http://anythingllm.local:3001",
    "api_key": "YOUR_ANYTHINGLLM_API_KEY"
  },
  "sync": {
    "default_workspace": "default",
    "state_file": "sync_state.json"
  }
}
```

- `default_workspace`: workspace used when a document has no tags
- `state_file`: sync state file used for incremental sync

### Run

```bash
go run ./cmd/p2a -config config.json
```

Dry-run (no changes applied):

```bash
go run ./cmd/p2a -config config.json -dry-run
```

Delete all AnythingLLM documents and workspaces tracked by this tool, then remove all workspaces on the instance:

```bash
go run ./cmd/p2a -config config.json -clear-anything
```

### Sync state file

`sync_state.json` stores the last synced version and the AnythingLLM document paths per workspace slug:

```json
{
  "docs": {
    "123": {
      "workspaces": {
        "tag-a": "custom-documents/example-123.pdf-xxxx.json",
        "tag-b": "custom-documents/example-123.pdf-yyyy.json"
      },
      "modified": "2025-12-16T10:59:35.587005+08:00"
    }
  }
}
```

### Linux scheduled runner with logs

Use `sync_loop.sh` to run sync repeatedly and write logs into `./logs/` under the program directory.

```bash
chmod +x ./sync_loop.sh
./sync_loop.sh
```

Environment variables:

- `CONFIG_PATH`: config file path (default: `./config.json`)
- `INTERVAL_SECONDS`: sleep between runs (default: `1800`)
- `BIN_PATH`: binary path (default: `./p2a`, auto-built if missing)

Logs:

- `./logs/p2a-YYYY-MM-DD.log` (stdout+stderr appended)

### Notes

- Paperless API uses `Authorization: Token <token>` and `/api/documents/`
- AnythingLLM API uses `Authorization: Bearer <api_key>`
- Workspaces are created via `/api/v1/workspace/new`
- Documents are deleted via `/api/v1/system/remove-documents`

## 中文

把 Paperless-ngx 的文档按标签同步到 AnythingLLM 的工作区（workspace）。

### 功能说明

- 从 Paperless 拉取文档
- 使用每个文档的标签作为工作区名称（多标签 → 多工作区）
- 在 AnythingLLM 中确保对应工作区存在
- 向每个目标工作区上传文档并更新 embeddings
- 使用 `sync_state.json` 做增量同步
- 当文档被替换或从工作区移除时，会在 AnythingLLM 中真实删除旧文档（不仅仅移除 embedding）
- 提供 `-clear-anything` 一键清理（文档 + 工作区）


### 配置

在项目根目录创建 `config.json`：

```json
{
  "paperless": {
    "base_url": "http://paperless.local",
    "token": "YOUR_PAPERLESS_TOKEN",
    "page_size": 100
  },
  "anythingllm": {
    "base_url": "http://anythingllm.local:3001",
    "api_key": "YOUR_ANYTHINGLLM_API_KEY"
  },
  "sync": {
    "default_workspace": "default",
    "state_file": "sync_state.json"
  }
}
```

- `default_workspace`：当文档没有任何标签时使用的默认工作区
- `state_file`：用于增量同步的状态文件路径

### 运行

```bash
go run ./cmd/p2a -config config.json
```

仅查看将要执行的动作（不写入）：

```bash
go run ./cmd/p2a -config config.json -dry-run
```

清空 AnythingLLM（删除本工具记录的文档，并删除实例上的全部工作区）：

```bash
go run ./cmd/p2a -config config.json -clear-anything
```

### 状态文件

`sync_state.json` 会记录每个 Paperless 文档在不同工作区下对应的 AnythingLLM 文档路径：

```json
{
  "docs": {
    "123": {
      "workspaces": {
        "tag-a": "custom-documents/example-123.pdf-xxxx.json",
        "tag-b": "custom-documents/example-123.pdf-yyyy.json"
      },
      "modified": "2025-12-16T10:59:35.587005+08:00"
    }
  }
}
```

### Linux 定时脚本与日志

使用 `sync_loop.sh` 循环定时运行同步，并将输出写到程序目录下的 `./logs/`。

```bash
chmod +x ./sync_loop.sh
./sync_loop.sh
```

可用环境变量：

- `CONFIG_PATH`：配置文件路径（默认 `./config.json`）
- `INTERVAL_SECONDS`：每次同步之间的间隔秒数（默认 `1800`）
- `BIN_PATH`：二进制路径（默认 `./p2a`，不存在会自动构建）

日志文件：

- `./logs/p2a-YYYY-MM-DD.log`（追加写入 stdout+stderr）

### 说明

- Paperless API 使用 `Authorization: Token <token>`，通过 `/api/documents/` 获取列表并下载
- AnythingLLM API 使用 `Authorization: Bearer <api_key>`
- 创建工作区使用 `/api/v1/workspace/new`
- 删除文档使用 `/api/v1/system/remove-documents`
