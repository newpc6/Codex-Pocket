# CodexFlow

CodexFlow 是一个面向 Codex CLI / Codex app-server 的 Web 控制台。

它的目标不是“远程看终端”，而是把 Codex 的会话、turn、diff、审批、状态流，整理成一套可以在浏览器中管理和继续指挥的控制平面。

当前仓库包含两部分：

- `Go Agent`：运行在安装了 Codex 的电脑上，负责接入 `codex app-server`、维护会话状态并暴露 HTTP/SSE API。
- `Web Console`：浏览器控制台，负责会话管理、实时消息、审批、继续对话和设置。

## 工作原理

CodexFlow 直接接在 `codex app-server` 之上，通过结构化 JSON-RPC 协议拿到真实的会话和执行状态，再转成 Web 端消费的 API。

```text
Codex CLI / codex app-server
        │
        │ JSON-RPC over stdio
        ▼
Go Agent
  - 启动并持有本地 codex app-server
  - 发现已有 session / loaded session
  - 接收通知、diff、plan、审批请求
  - 暴露 HTTP API + SSE
        │
        │ HTTP / SSE
        ▼
Web Console
  - 会话总览 / 会话详情
  - 实时消息 / Markdown / 图片
  - 审批中心
  - 继续下一步 / steer / interrupt
```

## 当前功能

- 自动发现真实的 Codex 历史会话。
- 支持新建、接管、取消接管、结束和归档会话。
- 支持开始新 turn、steer 当前 turn、interrupt 当前 turn。
- 支持 Codex 消息流、工具调用、命令输出、图片和 Markdown 展示。
- 捕获命令审批、文件变更审批、权限审批、结构化用户输入请求。
- 提供工作目录选择器，目录浏览发生在运行 Agent 的电脑上。
- 支持 Claude Code 历史与 runtime 接入。
- 对外提供 HTTP API 和 SSE 事件流。

## 快速开始

### 环境要求

- Windows / Linux / macOS
- 已安装并可用的 `codex` CLI
- 已完成 Codex 登录
- Go 1.26+
- Node.js / npm（开发 Web Console 时需要）

### 启动 Go Agent

在仓库根目录执行：

```bash
go run ./cmd/codexflow-agent
```

默认监听地址：

```text
127.0.0.1:7318
```

可选环境变量：

- `CODEXFLOW_LISTEN_ADDR`
- `CODEXFLOW_CODEX_PATH`
- `CODEXFLOW_REFRESH_INTERVAL`
- `CODEXFLOW_STATE_DB_PATH`
- `CODEXFLOW_WEB_DIST_PATH`

如果你的 `codex` 不在系统 `PATH` 里，可以显式指定：

```bash
CODEXFLOW_CODEX_PATH=/path/to/codex go run ./cmd/codexflow-agent
```

Windows 示例：

```powershell
$env:CODEXFLOW_CODEX_PATH="C:\path\to\codex.exe"
go run ./cmd/codexflow-agent
```

### 启动 Web Console

开发模式：

```bash
cd web
npm install
npm run dev
```

构建：

```bash
cd web
npm run build
```

构建产物默认输出到仓库根目录的 `dist/`，可通过 `CODEXFLOW_WEB_DIST_PATH` 交给 Go Agent 托管。

### 验证 Agent

```bash
curl http://127.0.0.1:7318/healthz
curl http://127.0.0.1:7318/api/v1/dashboard
```

## 基本使用

1. 打开 Web Console 的“会话”页面，查看当前真实会话。
2. 对历史会话点击“接管”，将其转为 CodexFlow 受控会话。
3. 在会话详情页查看消息流、工具调用、文件变更和审批请求。
4. 在底部输入框继续发送指令；如果当前 turn 正在运行，会以 steer 方式追加输入。
5. 对正在执行的 turn，可以直接中断。
6. 打开“审批”页面处理等待中的审批请求。

## API 概览

- `GET /healthz`
- `POST /api/v1/auth/login`
- `GET /api/v1/dashboard`
- `GET /api/v1/directories`
- `GET /api/v1/sessions`
- `POST /api/v1/sessions`
- `GET /api/v1/sessions/:id`
- `POST /api/v1/sessions/:id/resume`
- `POST /api/v1/sessions/:id/detach`
- `POST /api/v1/sessions/:id/end`
- `POST /api/v1/sessions/:id/archive`
- `POST /api/v1/sessions/:id/rename`
- `POST /api/v1/sessions/:id/fork`
- `POST /api/v1/sessions/:id/compact`
- `POST /api/v1/sessions/:id/rollback`
- `POST /api/v1/sessions/:id/turns/start`
- `POST /api/v1/sessions/:id/turns/steer`
- `POST /api/v1/sessions/:id/turns/interrupt`
- `GET /api/v1/approvals`
- `POST /api/v1/approvals/:id/resolve`
- `POST /api/v1/uploads/image`
- `GET /api/v1/assets/local-image`
- `GET /api/v1/events`

## 仓库结构

```text
cmd/codexflow-agent        Go Agent 启动入口
internal/codex             Codex app-server 协议适配
internal/runtime           会话管理、统计、审批编排
internal/httpapi           HTTP API 与 SSE
internal/store             本地状态存储
web                        Web Console
docs                       架构与路线文档
assets                     README 截图资源
```

## 开发建议

Go 后端适合当前项目的本机常驻服务和单二进制分发。开发期如果觉得重启麻烦，建议后续加入 `air`、`watchexec` 或 PowerShell watch 脚本做自动重编译重启，而不是改成 Python 后端。
