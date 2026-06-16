package runtime

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"codexpocket/internal/codex"
)

type codexSessionLogEnvelope struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexSessionMetaPayload struct {
	ID            string `json:"id"`
	CWD           string `json:"cwd"`
	CLI           string `json:"cli_version"`
	ModelProvider string `json:"model_provider"`
}

type codexTaskStartedPayload struct {
	TurnID    string `json:"turn_id"`
	StartedAt int64  `json:"started_at"`
}

type codexTaskEndedPayload struct {
	TurnID      string `json:"turn_id"`
	CompletedAt int64  `json:"completed_at"`
	DurationMs  int64  `json:"duration_ms"`
	LastMessage string `json:"last_agent_message"`
}

type codexTurnAbortedPayload struct {
	TurnID      string `json:"turn_id"`
	Reason      string `json:"reason"`
	CompletedAt int64  `json:"completed_at"`
	DurationMs  int64  `json:"duration_ms"`
}

type codexEventMessagePayload struct {
	Type         string `json:"type"`
	Message      string `json:"message"`
	Phase        string `json:"phase"`
	Images       []any  `json:"images"`
	LocalImages  []any  `json:"local_images"`
	TextElements []any  `json:"text_elements"`
}

type codexResponseItemPayload struct {
	Type    string           `json:"type"`
	Role    string           `json:"role"`
	Content []map[string]any `json:"content"`
	Phase   string           `json:"phase"`
}

type codexResponseFunctionCallPayload struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	CallID    string `json:"call_id"`
}

type codexResponseFunctionOutputPayload struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

type codexHistoryParseResult struct {
	thread     codex.Thread
	managedNow bool
}

func readCodexTurns(thread codex.Thread) ([]codex.Turn, int64, bool, error) {
	path, err := codexTranscriptPathForThread(thread)
	if err != nil {
		return nil, 0, false, err
	}

	result, err := parseCodexSessionLog(path)
	if err != nil {
		return nil, 0, false, err
	}
	return result.thread.Turns, result.thread.UpdatedAt, result.managedNow, nil
}

func codexTranscriptPathForThread(thread codex.Thread) (string, error) {
	if thread.Path != nil && strings.TrimSpace(*thread.Path) != "" {
		return strings.TrimSpace(*thread.Path), nil
	}
	if strings.TrimSpace(thread.ID) == "" {
		return "", errors.New("missing codex thread id")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(home, ".codex", "sessions")

	var candidates []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d == nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		if strings.Contains(filepath.Base(path), strings.TrimSpace(thread.ID)) {
			candidates = append(candidates, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("codex transcript not found for thread %s", thread.ID)
	}

	sort.Slice(candidates, func(i, j int) bool {
		left, leftErr := os.Stat(candidates[i])
		right, rightErr := os.Stat(candidates[j])
		switch {
		case leftErr != nil && rightErr != nil:
			return candidates[i] < candidates[j]
		case leftErr != nil:
			return false
		case rightErr != nil:
			return true
		}
		return left.ModTime().After(right.ModTime())
	})
	return candidates[0], nil
}

func parseCodexSessionLog(path string) (codexHistoryParseResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return codexHistoryParseResult{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return codexHistoryParseResult{}, err
	}

	thread := codex.Thread{
		Path:  stringPtr(path),
		Turns: []codex.Turn{},
	}
	turnsByID := make(map[string]*codex.Turn)
	callTurnByID := make(map[string]string)
	var currentTurnID string
	lastUpdated := info.ModTime().Unix()
	managedNow := false

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var envelope codexSessionLogEnvelope
		if err := json.Unmarshal([]byte(line), &envelope); err != nil {
			continue
		}

		if ts := parseRFC3339ToUnix(envelope.Timestamp); ts > lastUpdated {
			lastUpdated = ts
		}

		switch envelope.Type {
		case "session_meta":
			var payload codexSessionMetaPayload
			if json.Unmarshal(envelope.Payload, &payload) == nil {
				thread.ID = strings.TrimSpace(payload.ID)
				thread.CWD = strings.TrimSpace(payload.CWD)
				thread.CLIVersion = strings.TrimSpace(payload.CLI)
				thread.ModelProvider = strings.TrimSpace(payload.ModelProvider)
			}
		case "event_msg":
			var payload codexEventMessagePayload
			if json.Unmarshal(envelope.Payload, &payload) != nil {
				continue
			}
			switch payload.Type {
			case "task_started":
				currentTurnID = strings.TrimSpace(extractTurnIDFromTaskStarted(envelope.Payload))
				if currentTurnID != "" {
					turn := ensureCodexTurn(&thread.Turns, turnsByID, currentTurnID)
					if turn.StartedAt == nil {
						startedAt := extractStartedAtFromTaskStarted(envelope.Payload)
						if startedAt > 0 {
							turn.StartedAt = &startedAt
						}
					}
					turn.Status = "inProgress"
					managedNow = true
				}
			case "user_message":
				text := buildCodexEventMessageText(payload)
				if text == "" {
					continue
				}
				if currentTurnID == "" {
					currentTurnID = fmt.Sprintf("codex-turn-%d", len(thread.Turns)+1)
				}
				turn := ensureCodexTurn(&thread.Turns, turnsByID, currentTurnID)
				if !turnHasUserMessage(turn.Items) {
					turn.Items = append(turn.Items, composeUserMessageItem(text))
				}
				if turn.StartedAt == nil {
					if ts := parseRFC3339ToUnix(envelope.Timestamp); ts > 0 {
						turn.StartedAt = &ts
					}
				}
				if strings.TrimSpace(turn.Status) == "" {
					turn.Status = "inProgress"
				}
			case "agent_message":
				if strings.TrimSpace(payload.Phase) != "commentary" && strings.TrimSpace(payload.Phase) != "final" {
					continue
				}
				appendCodexAgentMessage(&thread.Turns, turnsByID, currentTurnID, payload.Message, payload.Phase, envelope.Timestamp)
			case "task_complete":
				var done codexTaskEndedPayload
				if json.Unmarshal(envelope.Payload, &done) == nil {
					finalizeCodexTurn(&thread.Turns, turnsByID, done.TurnID, "completed", strings.TrimSpace(done.LastMessage), done.CompletedAt, done.DurationMs, "")
					if currentTurnID == strings.TrimSpace(done.TurnID) {
						currentTurnID = ""
					}
				}
				managedNow = false
			case "turn_aborted":
				var aborted codexTurnAbortedPayload
				if json.Unmarshal(envelope.Payload, &aborted) == nil {
					finalizeCodexTurn(&thread.Turns, turnsByID, aborted.TurnID, "failed", "", aborted.CompletedAt, aborted.DurationMs, strings.TrimSpace(aborted.Reason))
					if currentTurnID == strings.TrimSpace(aborted.TurnID) {
						currentTurnID = ""
					}
				}
				managedNow = false
			}
		case "response_item":
			var rawType struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(envelope.Payload, &rawType) != nil {
				continue
			}
			switch rawType.Type {
			case "message":
				var payload codexResponseItemPayload
				if json.Unmarshal(envelope.Payload, &payload) != nil {
					continue
				}
				text := flattenCodexMessageContent(payload.Content)
				switch strings.TrimSpace(payload.Role) {
				case "user":
					if text == "" {
						continue
					}
					if currentTurnID == "" {
						currentTurnID = fmt.Sprintf("codex-turn-%d", len(thread.Turns)+1)
					}
					turn := ensureCodexTurn(&thread.Turns, turnsByID, currentTurnID)
					if !turnHasUserMessage(turn.Items) {
						turn.Items = append(turn.Items, composeUserMessageItem(text))
					}
					if turn.StartedAt == nil {
						if ts := parseRFC3339ToUnix(envelope.Timestamp); ts > 0 {
							turn.StartedAt = &ts
						}
					}
					if strings.TrimSpace(turn.Status) == "" {
						turn.Status = "inProgress"
					}
				case "assistant":
					if strings.TrimSpace(payload.Phase) != "commentary" && strings.TrimSpace(payload.Phase) != "final" {
						continue
					}
					appendCodexAgentMessage(&thread.Turns, turnsByID, currentTurnID, text, payload.Phase, envelope.Timestamp)
				}
			case "function_call":
				var payload codexResponseFunctionCallPayload
				if json.Unmarshal(envelope.Payload, &payload) != nil {
					continue
				}
				if currentTurnID == "" {
					continue
				}
				callTurnByID[strings.TrimSpace(payload.CallID)] = currentTurnID
				turn := ensureCodexTurn(&thread.Turns, turnsByID, currentTurnID)
				turn.Items = append(turn.Items, map[string]any{
					"id":      strings.TrimSpace(payload.CallID),
					"type":    "dynamicToolCall",
					"title":   strings.TrimSpace(payload.Name),
					"tool":    strings.TrimSpace(payload.Name),
					"summary": payload.Arguments,
					"status":  "completed",
					"result":  "",
				})
			case "function_call_output":
				var payload codexResponseFunctionOutputPayload
				if json.Unmarshal(envelope.Payload, &payload) != nil {
					continue
				}
				turnID := callTurnByID[strings.TrimSpace(payload.CallID)]
				if turnID == "" {
					turnID = currentTurnID
				}
				if turnID == "" {
					continue
				}
				turn := ensureCodexTurn(&thread.Turns, turnsByID, turnID)
				for i := range turn.Items {
					if toString(turn.Items[i]["id"]) != strings.TrimSpace(payload.CallID) {
						continue
					}
					turn.Items[i]["result"] = payload.Output
					turn.Items[i]["status"] = "completed"
					break
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return codexHistoryParseResult{}, err
	}

	thread.UpdatedAt = lastUpdated
	thread.Status = codex.ThreadStatus{Type: "idle"}
	if currentTurnID != "" {
		if turn, ok := turnsByID[currentTurnID]; ok && strings.TrimSpace(turn.Status) == "inProgress" {
			thread.Status = codex.ThreadStatus{Type: "active"}
			managedNow = true
		}
	}
	return codexHistoryParseResult{thread: thread, managedNow: managedNow}, nil
}

func ensureCodexTurn(turns *[]codex.Turn, turnsByID map[string]*codex.Turn, turnID string) *codex.Turn {
	if existing, ok := turnsByID[turnID]; ok {
		return existing
	}
	turn := codex.Turn{
		ID:     strings.TrimSpace(turnID),
		Items:  []map[string]any{},
		Status: "inProgress",
	}
	*turns = append(*turns, turn)
	ptr := &(*turns)[len(*turns)-1]
	turnsByID[turnID] = ptr
	return ptr
}

func appendCodexAgentMessage(turns *[]codex.Turn, turnsByID map[string]*codex.Turn, turnID, text, phase, tsRaw string) {
	trimmed := strings.TrimSpace(text)
	if turnID == "" || trimmed == "" {
		return
	}
	turn := ensureCodexTurn(turns, turnsByID, turnID)
	if turnHasAgentText(turn.Items, trimmed) {
		return
	}
	itemID := fmt.Sprintf("%s:%s:%d", turnID, phase, len(turn.Items))
	turn.Items = append(turn.Items, map[string]any{
		"id":   itemID,
		"type": "agentMessage",
		"text": trimmed,
	})
	if strings.TrimSpace(turn.Status) == "" {
		turn.Status = "inProgress"
	}
	if turn.StartedAt == nil {
		if ts := parseRFC3339ToUnix(tsRaw); ts > 0 {
			turn.StartedAt = &ts
		}
	}
}

func finalizeCodexTurn(turns *[]codex.Turn, turnsByID map[string]*codex.Turn, turnID, status, finalMessage string, completedAt, durationMs int64, failureReason string) {
	turnID = strings.TrimSpace(turnID)
	if turnID == "" {
		return
	}
	turn := ensureCodexTurn(turns, turnsByID, turnID)
	if strings.TrimSpace(finalMessage) != "" && !turnHasAgentText(turn.Items, strings.TrimSpace(finalMessage)) {
		turn.Items = append(turn.Items, map[string]any{
			"id":   fmt.Sprintf("%s:final", turnID),
			"type": "agentMessage",
			"text": strings.TrimSpace(finalMessage),
		})
	}
	turn.Status = status
	if completedAt > 0 {
		turn.CompletedAt = &completedAt
	}
	if durationMs > 0 {
		turn.DurationMs = &durationMs
	}
	if status == "failed" && strings.TrimSpace(failureReason) != "" {
		turn.Error = &codex.TurnError{Message: strings.TrimSpace(failureReason)}
	}
}

func turnHasUserMessage(items []map[string]any) bool {
	for _, item := range items {
		if toString(item["type"]) == "userMessage" {
			return true
		}
	}
	return false
}

func turnHasUserMessageWithText(items []map[string]any, text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	for _, item := range items {
		if toString(item["type"]) != "userMessage" {
			continue
		}
		if strings.TrimSpace(codex.FirstUserText([]map[string]any{item})) == text {
			return true
		}
	}
	return false
}

func turnHasAgentText(items []map[string]any, text string) bool {
	for _, item := range items {
		if toString(item["type"]) != "agentMessage" {
			continue
		}
		if strings.TrimSpace(toString(item["text"])) == text {
			return true
		}
	}
	return false
}

func buildCodexEventMessageText(payload codexEventMessagePayload) string {
	parts := make([]string, 0, 1+len(payload.LocalImages)+len(payload.Images))
	if text := strings.TrimSpace(payload.Message); text != "" {
		parts = append(parts, text)
	}
	parts = append(parts, flattenCodexImageList(payload.LocalImages)...)
	parts = append(parts, flattenCodexImageList(payload.Images)...)
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func flattenCodexMessageContent(items []map[string]any) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		itemType := strings.TrimSpace(toString(item["type"]))
		switch itemType {
		case "output_text", "input_text", "text":
			text := strings.TrimSpace(toString(item["text"]))
			if text != "" {
				parts = append(parts, text)
			}
		case "input_image", "output_image", "image", "localImage":
			if imageMarkdown := codexImageMarkdownFromMap(item); imageMarkdown != "" {
				parts = append(parts, imageMarkdown)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func flattenCodexImageList(items []any) []string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		switch typed := item.(type) {
		case string:
			if text := strings.TrimSpace(typed); text != "" {
				parts = append(parts, fmt.Sprintf("[Attached image: %s]", text))
			}
		case map[string]any:
			if imageMarkdown := codexImageMarkdownFromMap(typed); imageMarkdown != "" {
				parts = append(parts, imageMarkdown)
			}
		}
	}
	return parts
}

func codexImageMarkdownFromMap(item map[string]any) string {
	for _, key := range []string{"path", "file_path", "local_path"} {
		if path := strings.TrimSpace(toString(item[key])); path != "" {
			return fmt.Sprintf("[Attached image: %s]", path)
		}
	}
	for _, key := range []string{"image_url", "url"} {
		if url := strings.TrimSpace(toString(item[key])); url != "" {
			return fmt.Sprintf("![Attached image](%s)", url)
		}
	}
	return ""
}

func extractTurnIDFromTaskStarted(raw json.RawMessage) string {
	var payload codexTaskStartedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.TurnID)
}

func extractStartedAtFromTaskStarted(raw json.RawMessage) int64 {
	var payload codexTaskStartedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return 0
	}
	return payload.StartedAt
}

func parseRFC3339ToUnix(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return 0
	}
	return parsed.Unix()
}
