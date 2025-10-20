# Repository Guidelines

## 项目结构与模块组织
- 根入口：`main.go`（命令行、配置加载、日志、HTTP、SQLite、导出）。
- 业务包：`internal/aggregate`（调度与内存缓冲）、`internal/fetch`（HTTP 客户端/代理/重试）、`internal/feeds`（Feed 发现）、`internal/friends`（友链页解析）、`internal/export`（JSON 导出）、`internal/store`（SQLite 持久化）、`internal/config`、`internal/rules`、`internal/logx`。
- 测试：`tests/` 以集成化单元测试为主（使用 `httptest` 与临时文件）。

目录示例：

```
.
├─ internal/{aggregate,fetch,feeds,friends,export,store,config,rules,logx}
├─ tests/*.go
├─ settings.yaml, rules.yaml, *.example.yaml
└─ data.db, data.json  (本地产物，不提交)
```

## 架构概览
- 配置与规则：`config` + `rules` 定义输入与解析选择器。
- 获取与解析：`fetch` 负责 HTTP；`friends`/`feeds` 提取朋友与订阅。
- 聚合与清理：`aggregate` 并发抓取、去重、按 `OutdateCleanDays` 清理。
- 存储与导出：正常模式写 `store`(SQLite)；极简模式只 `export` JSON。

## 构建、测试与本地运行
- 下载依赖：`go mod download`
- 构建二进制：`go build -o cof .`
- 本地运行（极简导出）：`go run . -config settings.yaml -rules rules.yaml -export data.json`
- 发现规则调试：`go run . -config settings.yaml -rules rules.yaml -discover`
- 运行测试（与 CI 对齐）：`go test -coverpkg=./... ./tests -count=1 -race -coverprofile=coverage.out -v`
  - CI：见 `.github/workflows/ci.yml`（Go 1.22，上传 `coverage.out`）。

## 编码风格与命名规范
- 使用 Go 1.22；提交前执行：`gofmt -s -w . && go vet ./...`。
- 包名小写、简短，无下划线；导出标识使用 `UpperCamelCase`，非导出使用 `lowerCamelCase`。
- 错误包装：`fmt.Errorf("context: %w", err)`；函数首参优先传入 `context.Context`。
- 目录职责单一，避免跨包循环依赖；新增模块优先放入 `internal/`。

## 测试指南
- 测试文件命名：`*_test.go`，测试函数 `TestXxx`；放置于 `tests/`（`package tests`）。
- 避免真实网络依赖：使用 `httptest`、`t.TempDir()` 与临时 SQLite 路径。
- 覆盖率产物：`coverage.out`；确保新增代码具备可测性并不降低总体稳定性。

## 提交与 Pull Request
- 提交信息：采用 Conventional Commits（如：`feat: ...`、`fix: ...`、`docs: ...`、`refactor: ...`、`test: ...`、`chore: ...`）。
- PR 要求：包含动机与变更说明、必要的截图/日志、关联 Issue；CI 通过（Go 1.22，含 `-race` 与覆盖率）。
- 不要提交生成物/本地数据（例如：`data.json`、`data.db`、`coverage.out`）；配置改动仅提交 `*.example.yaml`。
 - 配置示例同步：如变更配置字段/默认值/行为，必须同步更新 `settings.example.yaml`、`rules.example.yaml` 及 `README.md` 示例，并在 PR 中提供迁移说明。

PR 检查清单：
- [ ] 新/改代码已 `gofmt` 与 `go vet`
- [ ] 测试覆盖新增逻辑；`go test` 通过
- [ ] 无多余调试输出/无敏感信息
 - [ ] 如变更配置或默认值，已更新 `*.example.yaml` 与 `README` 示例，并附迁移说明（必要时标注 BREAKING CHANGE）

## 安全与配置提示（可选）
- 勿提交敏感信息到 `settings.yaml`/`rules.yaml`；私用配置请基于示例复制本地修改。
- 需要代理时通过配置文件设置 `Proxy.HTTP/HTTPS`，勿在代码中硬编码。

## 配置与示例同步
- 新增/重命名配置项：同时调整 `*.example.yaml` 与 `README` 的运行示例；确保默认值可用。
- 变更默认行为：在 PR 描述中明确影响面与回滚方案，优先保持向后兼容；若破坏性变更，使用 `BREAKING CHANGE:` 说明。

## Agent 专用说明（仅对自动化代理生效）
- 搜索优先 `rg`；读取文件分段（≤250 行）。
- 使用 `apply_patch` 进行最小化变更；避免不相关重构。
- 避免执行 `git` 相关操作，除非明确要求。
- 变更遵循 KISS/DRY/YAGNI，新增代码放入 `internal/` 并保持单一职责。
