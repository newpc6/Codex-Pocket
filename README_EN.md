# CodexPocket

[中文](README.md) | English

CodexPocket is a web-based session console that runs on the same computer as Codex. It organizes local Codex sessions into an interface optimized for both desktop and mobile browsers, enabling remote session viewing, task takeover, instruction sending, interrupt handling, code change review, and detailed process inspection.

It's not a simple terminal forwarding page, but a visual control plane built around Codex app-server's sessions, turns, tool calls, file changes, images, approvals, and status streams.

## Interface Preview

### Login & New Session

<p>
  <img src="assets/screenshots/18-登录页面.jpg" alt="Login Page" width="260">
  <img src="assets/screenshots/19-新建会话.jpg" alt="New Session" width="260">
</p>

### Mobile Session Home & Project Directory

<p>
  <img src="assets/screenshots/11-mobile-会话首页.jpg" alt="Mobile Session Home" width="220">
  <img src="assets/screenshots/12-mobile-项目目录.jpg" alt="Mobile Project Directory" width="220">
</p>

### Session Details & Real-time Processing

<p>
  <img src="assets/screenshots/13-mobile-会话内容.jpg" alt="Mobile Session Content" width="220">
  <img src="assets/screenshots/14-mobile-会话内容-处理中.jpg" alt="Mobile Session Processing" width="220">
  <img src="assets/screenshots/16--mobile-会话内容-过程详情.jpg" alt="Mobile Process Details" width="220">
</p>

### Code Changes & Session End

<p>
  <img src="assets/screenshots/15--mobile-会话内容-代码改动.jpg" alt="Mobile Code Changes" width="220">
  <img src="assets/screenshots/17--mobile-会话内容-会话结束.jpg" alt="Mobile Session End" width="220">
</p>

## Core Features

- **Mobile-first Session Control**: Session list, project grouping, session details, input field, process folding, and file changes are all optimized for compact mobile browser layouts.
- **Session Discovery & Takeover**: Automatically discovers local Codex historical sessions, grouped by working directory; unattached sessions can be taken over with one click, then continue sending instructions.
- **Real-time Turn Status**: Through SSE and local transcript synchronization, displays Codex thinking, running commands, editing files, and final summaries.
- **Process Folding**: By default highlights user messages and Codex final responses; intermediate commands, tool calls, and editing processes are folded into a "processed" area, expandable in chronological order.
- **Code Change Panel**: Each session turn shows modified code files, addition/deletion line statistics; view current turn diff, single file diff, and support reviewing and reverting workspace changes.
- **Review Mode**: Supports reviewing current workspace, specified commit, base branch, or a specific turn's changes, enabling quick risk assessment on mobile without scrolling through complete diffs.
- **Directory Selector**: When creating a new session, browse project directories on the Agent's computer; mobile can also enter directories and select current directory.
- **Image Input & Preview**: Supports uploading images as input attachments; images in sessions are displayed as thumbnails and can be enlarged for viewing.
- **Commands & Approvals**: Running commands, approval requests, and permission prompts are unified into the session flow and approval center, suitable for remotely handling stuck tasks.
- **Multi-Agent Support**: Currently supports Codex app-server, with reserved capability for Claude Code session discovery and runtime integration.

## Architecture

```text
Codex CLI / codex app-server
        |
        | JSON-RPC over stdio
        v
Go Agent
  - Start and hold local codex app-server
  - Discover, takeover, resume, and end sessions
  - Sync turn, tool, diff, approval, and goal status
  - Expose HTTP API, SSE, and static Web resources
        |
        | HTTP / SSE
        v
Web Console
  - Session home
  - Session details
  - Real-time messages & process folding
  - File changes & Review
  - Approval center & Settings page
```

## Quick Start

### Requirements

- `codex` CLI installed and logged in
- Go 1.26+
- Node.js / npm
- Windows, macOS, or Linux

### Backend Agent

Development run:

```bash
go run ./cmd/codexpocket-agent
```

Build single-file backend:

```bash
go build -o codexpocket-agent.exe ./cmd/codexpocket-agent
```

Default backend listening on:

```text
0.0.0.0:7318
```

Common environment variables:

```text
CODEXPOCKET_LISTEN_ADDR       Backend listen address
CODEXPOCKET_CODEX_PATH        codex executable path
CODEXPOCKET_CLAUDE_PATH       claude executable path
CODEXPOCKET_JWT_SECRET        JWT signing secret
CODEXPOCKET_REFRESH_INTERVAL  Background refresh interval, e.g. 12s
CODEXPOCKET_STATE_DB_PATH     Local state database path
CODEXPOCKET_WEB_DIST_PATH     Web Console dist directory
CODEXPOCKET_ALLOWED_ORIGINS   Allowed origins for API access
```

Windows example:

```powershell
$env:CODEXPOCKET_CODEX_PATH = "C:\path\to\codex.exe"
go run ./cmd/codexpocket-agent
```

### Web Console

Development mode:

```bash
cd web
npm install
npm run dev
```

Development server runs on:

```text
http://localhost:7319
```

`vite` proxies `/api` requests to backend `http://127.0.0.1:7318`.

Production build:

```bash
cd web
npm run build
```

Build output goes to repository root `dist/`. Backend automatically serves Web Console if `dist/` is found alongside the executable.

## Basic Usage

1. Start the backend Agent and confirm Codex CLI is logged in.
2. Start Web Console and open `http://localhost:7319`.
3. Login with configured credentials, default dev account is `admin / admin123`.
4. View sessions by project directory on the session home, or select a working directory to create a new session.
5. Enter session detail page to view user messages, Codex responses, process details, commands, and file changes.
6. Unattached sessions can click "Takeover Session", then continue sending instructions, append steer, or interrupt current turn.
7. After session ends, view changed files in this turn, open diff, initiate review, or revert workspace changes.

## Main Pages

- **Session Home**: Display sessions grouped by project directory, support viewing running, historical, unattached statuses.
- **New Session**: Select working directory, fill initial prompt, choose model, reasoning effort, and collaboration mode.
- **Session Details**: Display session header, takeover status, turn timeline, real-time messages, tool calls, images, and file changes.
- **File Changes**: View code changes in workspace, commit, base branch, or specified turn.
- **Approval Center**: Centrally handle command, file, permission, and user input approvals requested by Codex/Claude.
- **Settings Page**: View Agent status, listen address, Codex path, runtime capabilities, and login info.

## API Overview

Common endpoints:

```text
GET    /healthz
POST   /api/v1/auth/login
GET    /api/v1/dashboard
GET    /api/v1/options
GET    /api/v1/directories
GET    /api/v1/sessions
POST   /api/v1/sessions
GET    /api/v1/sessions/:id
POST   /api/v1/sessions/:id/resume
POST   /api/v1/sessions/:id/detach
POST   /api/v1/sessions/:id/end
POST   /api/v1/sessions/:id/archive
POST   /api/v1/sessions/:id/rename
POST   /api/v1/sessions/:id/fork
POST   /api/v1/sessions/:id/compact
POST   /api/v1/sessions/:id/rollback
GET    /api/v1/sessions/:id/changes
POST   /api/v1/sessions/:id/changes/revert
POST   /api/v1/sessions/:id/review
GET    /api/v1/sessions/:id/goal
POST   /api/v1/sessions/:id/goal
DELETE /api/v1/sessions/:id/goal
POST   /api/v1/sessions/:id/turns/start
POST   /api/v1/sessions/:id/turns/steer
POST   /api/v1/sessions/:id/turns/interrupt
GET    /api/v1/approvals
POST   /api/v1/approvals/:id/resolve
POST   /api/v1/uploads/image
GET    /api/v1/assets/local-image
GET    /api/v1/events
```

## Security Recommendations

CodexPocket can control the computer running Agent to execute Codex operations. Treat it as a local automation tool:

- Do not expose default credentials to the public internet.
- When deploying to LAN or for remote access, modify `CODEXPOCKET_JWT_SECRET` and login credentials.
- Recommended to use with reverse proxy, HTTPS, access control, or VPN.
- Exercise caution with high-privilege operations like command execution, file reversion, and approval handling.

## Development Verification

Common check commands:

```bash
go test ./...
cd web && npm run build
git diff --check
```

During development, frontend uses `npm run dev` for hot reload; backend is a single Go process, requiring restart after changes, or use tools like `air` or `watchexec` for auto-restart.

## Repository Structure

```text
cmd/codexpocket-agent  Go Agent entry point
internal/codex         Codex app-server JSON-RPC adapter
internal/runtime       Session, turn, approval, status, and multi-agent orchestration
internal/httpapi       HTTP API, SSE, authentication, and resource access
internal/store         Local state storage
web                    Vue 3 Web Console
assets/screenshots     README screenshot resources
docs                   Architecture, lifecycle, and roadmap documents
scripts                Helper scripts
```
