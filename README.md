# paperless2anythingllm

一个用 Go 编写的同步工具：
- 从 Paperless 获取文档并按路径分组
- 在 AnythingLLM 中为每个路径创建同名工作区
- 将文档上传并嵌入到对应工作区
- 每次运行执行增量同步（新增、修改、删除）

## 使用

1. 在项目根目录创建 `config.json`：

```
{
  "paperless": {
    "base_url": "http://paperless.local",
    "token": "YOUR_PAPERLESS_TOKEN",
    "page_size": 100
  },
  "anythingllm": {
    "base_url": "http://anythingllm.local:3001",
    "api_key": "YOUR_ANYTHINGLLM_API_KEY",
    "upload_method": "upload"
  },
  "sync": {
    "grouping": "storage_path",
    "default_workspace": "default",
    "state_file": "sync_state.json"
  }
}
```

2. 运行：

```
go run ./cmd/p2a -config config.json
```

可选：使用 `-dry-run` 查看将执行的增量操作而不真正写入。

## 分组策略

- `storage_path`：按 Paperless 文档的存储路径分组（推荐）。
- `tag`：按第一个标签分组（如果无标签则使用 `default_workspace`）。

## 说明

- Paperless 使用 `Authorization: Token <token>` 调用 `/api/documents/` 和下载接口。
- AnythingLLM 通过 `Bearer <api_key>` 创建工作区、上传文档到 `custom-documents`，并调用 `workspace/{slug}/update-embeddings` 完成嵌入。
- 本工具使用 `sync_state.json` 记录上次同步的文档状态，用于计算增量。
