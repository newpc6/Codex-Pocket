# Architecture

## Components

### 1. Local Agent

Responsibilities:

- spawn and own a local `codex app-server`
- discover sessions with `thread/list`, `thread/read`, `thread/turns/list`, and `thread/loaded/list`
- subscribe to runtime notifications
- queue approval requests from Codex
- expose a browser-friendly HTTP API and SSE stream
- keep local lifecycle state such as managed, ended, and runtime bindings

### 2. Web Console

Responsibilities:

- dashboard for all sessions
- chat-style session detail with Markdown, images, tools, and command output
- approval center
- directory picker for paths on the agent machine
- remote prompting, steering, interrupt, detach, archive, fork, compact, and rollback actions

### 3. Optional Relay Layer

Planned later:

- secure remote access without exposing the local agent directly
- device pairing
- notification delivery for approvals and turn completion

## Data Flow

1. `CodexFlow Agent` starts `codex app-server --listen stdio://`.
2. Agent initializes the JSON-RPC session with experimental API capability.
3. Agent refreshes thread inventory and listens for notifications.
4. Web Console calls the Agent HTTP API and subscribes to SSE.
5. Approvals are sent back through JSON-RPC response messages.

## Why This Shape

- `stdio` keeps the Codex app-server transport local and simple.
- the Go agent is the single place for policy, lifecycle, local files, relay, and audit.
- the Web Console works against a stable app-specific API instead of speaking raw Codex protocol directly.

## Related Design Docs

- [Session Lifecycle](./session-lifecycle.md)
