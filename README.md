# go-circle-of-friends

一个用于聚合友链站点文章并导出结构化数据的 Go 实现。

## 快速开始

1. 按需编辑 `settings.yaml`（默认已包含示例）。
2. 运行（需联网拉取依赖）：

```
go run . -config settings.yaml -rules rules.yaml -export data.json
```

默认启用极简模式（`SIMPLE_MODE: true`），运行完成后在根目录生成 `data.json`。极简模式下不会打开/写入数据库；正常模式才使用 `data.db`（SQLite）。
为保证输出体积与性能，`data.json` 仅保留按时间倒序的最新 150 篇文章（全局上限保护）。

启动重置（可选）：
- 正常模式：`RESET_ON_START: true` 会在每次运行前清空数据库表（friends/posts），并删除导出文件（`-export` 路径，默认 `data.json`）。
- 极简模式：不会打开数据库，仅删除导出文件；数据库保持不变。

## 友链页规则调试

如果运行后提示 `no friends discovered (static or page)`，用调试模式打印根据 `rules.yaml` 解析到的朋友列表：

```
go run . -config settings.yaml -rules rules.yaml -discover
```

若输出为 0，请核对：
- `settings.yaml` → `LINK[0].url` 是否为真实的友链页地址（例如 `/friends`、`/links`）。
- `rules.yaml` → `default.friends_page` 的 `item/name/link/avatar` 选择器是否匹配你的 DOM 结构。

## 日志与级别/格式/语言

- 通过 `settings.yaml` 控制：
  - `LOG_LEVEL`: `debug|info|warn|error|none`（默认 `info`）
  - `LOG_FORMAT`: `pretty|text|json`（默认 `pretty`）
  - `LOG_LOCALE`: `zh-CN|en`（默认 `zh-CN`）
  - `LOG_COLOR`: `auto|always|never`（默认 `auto`，仅 pretty 生效）
- 示例：

```
LOG_LEVEL: debug
LOG_FORMAT: pretty
LOG_LOCALE: zh-CN
LOG_COLOR: always
go run . -config settings.yaml -rules rules.yaml -export data.json
```

## 构建二进制

```
go build -o cof .
```

- 常见日志（中文 pretty 格式）：
  - `2025-10-20 12:00:00 [信息] 静态朋友=N，页面来源=M`
  - `2025-10-20 12:00:01 [信息] https://... 解析到 K 位朋友`
  - `2025-10-20 12:00:05 [信息] [某某] 文章解析完成：X`
  - `2025-10-20 12:00:06 [警告] [某某] 发现订阅失败：...`
  - `2025-10-20 12:00:10 [信息] 已导出 data.json`

## 运行测试

- 仅运行不依赖第三方包的核心单测（避免网络依赖）：

```
go test ./internal/fetch ./internal/logx -v
```

- 说明：
  - `internal/fetch` 覆盖 UA 头、失败重试与超时。
  - `internal/logx` 覆盖等级解析、标签与颜色策略。
  - 其余包因依赖外部模块/网络，建议在可联网环境中再补充集成测试。
