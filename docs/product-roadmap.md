# Product Roadmap

## Current Foundation

- local Go agent with live Codex app-server integration
- Web Console with dashboard, session detail, approval center, and settings
- session discovery across historical threads
- real-time SSE updates for turns, items, approvals, and agent message deltas
- browser-side control for continuing, steering, interrupting, detaching, and archiving sessions

## Next Build Targets

### 1. Stronger Session Control

- use `thread/turns/list` for stable paged history loading
- expose rename, fork, compact, and rollback as first-class workflows
- add goal management with `thread/goal/*`

### 2. Live Session Streaming

- keep CodexFlow and Codex app-server runtime state aligned with fewer polling fallbacks
- surface live command output, diff updates, and plan changes with minimal delay
- add explicit connection diagnostics when Codex app-server notifications are missing

### 3. Review And Maintenance

- expose `review/start` for code review workflows
- add context compaction status and rollback markers to the chat stream
- add audit trails for approvals and destructive actions

### 4. Approval Policies

- rules for safe auto-approval
- per-repo network permission presets
- per-session approval audit trail

### 5. Remote Access

- secure device pairing
- authenticated remote access outside the LAN
- notification delivery for approvals and turn completion
