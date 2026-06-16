package runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"codexpocket/internal/codex"
)

const maxChangedFileContentBytes = 240 * 1024

var errEmptyCommitRef = errors.New("commit ref is required")

func (a *Agent) Options(ctx context.Context) SessionOptions {
	options := defaultSessionOptions()
	options.Models = a.modelOptions(ctx)
	options.CollaborationModes = a.collaborationModeOptions(ctx)
	return options
}

func defaultSessionOptions() SessionOptions {
	return SessionOptions{
		Models:             defaultModelOptions(),
		ReasoningEfforts:   defaultReasoningEfforts(),
		CollaborationModes: defaultCollaborationModes(),
		Presets:            defaultSessionPresets(),
	}
}

func defaultModelOptions() []ModelOption {
	return []ModelOption{
		{ID: "", Name: "Codex 默认", Description: "沿用当前 Codex 配置", Default: true},
		{ID: "gpt-5-codex", Name: "GPT-5 Codex", Description: "适合认真改代码和审查"},
		{ID: "gpt-5-mini", Name: "GPT-5 Mini", Description: "适合快速问答和轻量任务"},
	}
}

func (a *Agent) modelOptions(ctx context.Context) []ModelOption {
	var raw struct {
		Data []struct {
			ID          string `json:"id"`
			Model       string `json:"model"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
			Default     bool   `json:"default"`
			IsDefault   bool   `json:"isDefault"`
		} `json:"data"`
	}
	if err := a.client.Call(ctx, "model/list", map[string]any{}, &raw); err == nil && len(raw.Data) > 0 {
		models := make([]ModelOption, 0, len(raw.Data))
		for _, item := range raw.Data {
			id := firstNonEmpty(item.ID, item.Model)
			if id == "" {
				continue
			}
			name := firstNonEmpty(item.DisplayName, item.Name, id)
			models = append(models, ModelOption{
				ID:          id,
				Name:        name,
				Description: item.Description,
				Default:     item.Default || item.IsDefault,
			})
		}
		if len(models) > 0 {
			return ensureOneDefaultModel(models)
		}
	}

	return defaultModelOptions()
}

func ensureOneDefaultModel(models []ModelOption) []ModelOption {
	for _, model := range models {
		if model.Default {
			return models
		}
	}
	models[0].Default = true
	return models
}

func defaultReasoningEfforts() []ReasoningEffortOption {
	return []ReasoningEffortOption{
		{ID: "", Name: "默认", Default: true},
		{ID: "minimal", Name: "快一点"},
		{ID: "medium", Name: "认真改"},
		{ID: "high", Name: "深度审查"},
	}
}

func defaultCollaborationModes() []CollaborationModeOption {
	return []CollaborationModeOption{
		{ID: "default", Name: "默认协作", Description: "可以分析、修改并验证", Default: true},
		{ID: "plan", Name: "只分析不改", Description: "先给计划和建议，不直接改文件"},
		{ID: "review", Name: "代码审查", Description: "优先找风险、回归和测试缺口"},
	}
}

func (a *Agent) collaborationModeOptions(ctx context.Context) []CollaborationModeOption {
	var raw struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
			Default     bool   `json:"default"`
			IsDefault   bool   `json:"isDefault"`
		} `json:"data"`
	}
	if err := a.client.Call(ctx, "collaborationMode/list", map[string]any{}, &raw); err == nil && len(raw.Data) > 0 {
		modes := make([]CollaborationModeOption, 0, len(raw.Data))
		for _, item := range raw.Data {
			id := strings.TrimSpace(item.ID)
			if id == "" {
				continue
			}
			modes = append(modes, CollaborationModeOption{
				ID:          id,
				Name:        firstNonEmpty(item.DisplayName, item.Name, id),
				Description: item.Description,
				Default:     item.Default || item.IsDefault,
			})
		}
		if len(modes) > 0 {
			return ensureOneDefaultCollaborationMode(modes)
		}
	}
	return defaultCollaborationModes()
}

func ensureOneDefaultCollaborationMode(modes []CollaborationModeOption) []CollaborationModeOption {
	for _, mode := range modes {
		if mode.Default {
			return modes
		}
	}
	modes[0].Default = true
	return modes
}

func defaultSessionPresets() []SessionPreset {
	return []SessionPreset{
		{ID: "balanced", Name: "认真改", Description: "默认模型与中等推理，适合日常开发", ReasoningEffort: "medium", CollaborationMode: "default"},
		{ID: "fast", Name: "快一点", Description: "更快响应，适合小问题和轻量修改", ReasoningEffort: "minimal", CollaborationMode: "default"},
		{ID: "review", Name: "代码审查", Description: "按 review 模式检查风险和测试缺口", ReasoningEffort: "high", CollaborationMode: "review"},
		{ID: "analysis", Name: "只分析不改", Description: "先讨论方案，不直接动代码", ReasoningEffort: "medium", CollaborationMode: "plan"},
	}
}

func (a *Agent) SessionChanges(ctx context.Context, threadID string, scope ChangeScope, ref, base, filePath string) (SessionChanges, error) {
	record, ok := a.store.SnapshotSession(threadID)
	if !ok {
		return SessionChanges{}, errors.New("session not found")
	}
	cwd := strings.TrimSpace(record.Thread.CWD)
	if cwd == "" {
		return SessionChanges{}, errors.New("session working directory is unknown")
	}
	scope = normalizeChangeScope(scope)
	if scope == ChangeScopeCommit && strings.TrimSpace(ref) == "" {
		return emptySessionChanges(scope, ref, base, cwd), nil
	}
	return readGitChanges(ctx, cwd, scope, ref, base, filePath)
}

func (a *Agent) RevertSessionChanges(ctx context.Context, threadID string, files []string) (SessionChanges, error) {
	record, ok := a.store.SnapshotSession(threadID)
	if !ok {
		return SessionChanges{}, errors.New("session not found")
	}
	cwd := strings.TrimSpace(record.Thread.CWD)
	if cwd == "" {
		return SessionChanges{}, errors.New("session working directory is unknown")
	}
	if err := ensureGitWorktree(ctx, cwd); err != nil {
		return SessionChanges{}, err
	}
	targets, err := cleanChangePaths(cwd, files)
	if err != nil {
		return SessionChanges{}, err
	}
	if len(targets) == 0 {
		return SessionChanges{}, errors.New("no files selected")
	}

	for _, target := range targets {
		if isGitTracked(ctx, cwd, target) {
			if _, err := runGit(ctx, cwd, "restore", "--worktree", "--", target); err != nil {
				return SessionChanges{}, err
			}
			continue
		}
		if err := removeWorkspaceFile(cwd, target); err != nil {
			return SessionChanges{}, err
		}
	}
	return readGitChanges(ctx, cwd, ChangeScopeWorkspace, "", "", "")
}

func emptySessionChanges(scope ChangeScope, ref, base, cwd string) SessionChanges {
	return SessionChanges{
		Scope:     scope,
		Ref:       strings.TrimSpace(ref),
		Base:      strings.TrimSpace(base),
		CWD:       cwd,
		Files:     []ChangedFile{},
		Diff:      "",
		Generated: time.Now().UnixMilli(),
	}
}

func (a *Agent) StartReview(ctx context.Context, threadID string, req ReviewStartRequest) (TurnDetail, error) {
	if isClaudeThreadID(threadID) {
		return TurnDetail{}, errors.New("review is not supported for claude sessions")
	}
	scope := normalizeChangeScope(req.Scope)
	if turn, err := a.startNativeReview(ctx, threadID, scope, req.Ref, req.Base); err == nil {
		return turn, nil
	} else {
		a.logger.Debug("native review/start failed, falling back to prompt review", "threadId", threadID, "error", err)
	}

	changes, err := a.SessionChanges(ctx, threadID, scope, req.Ref, req.Base, "")
	if err != nil {
		return TurnDetail{}, err
	}
	if changes.Summary.Files == 0 {
		return TurnDetail{}, errors.New("no changes found for review")
	}
	prompt := buildReviewPrompt(changes)
	return a.StartTurnWithPrompt(ctx, threadID, prompt)
}

func (a *Agent) startNativeReview(ctx context.Context, threadID string, scope ChangeScope, ref, base string) (TurnDetail, error) {
	target, err := reviewTarget(scope, ref, base)
	if err != nil {
		return TurnDetail{}, err
	}
	var response codex.ReviewStartResponse
	if err := a.client.Call(ctx, "review/start", map[string]any{
		"threadId": threadID,
		"delivery": "inline",
		"target":   target,
	}, &response); err != nil {
		return TurnDetail{}, err
	}
	a.store.SetSessionEnded(threadID, false)
	if response.Turn.ID != "" {
		a.store.RecordTurn(threadID, response.Turn)
	}
	a.broker.Publish("turn.started", map[string]string{
		"threadId": threadID,
		"turnId":   response.Turn.ID,
	})

	record, _ := a.store.SnapshotSession(threadID)
	for _, turn := range toSessionDetail(record, 0).Turns {
		if turn.ID == response.Turn.ID {
			return turn, nil
		}
	}
	return TurnDetail{}, errors.New("review turn not found after start")
}

func reviewTarget(scope ChangeScope, ref, base string) (map[string]any, error) {
	switch scope {
	case ChangeScopeCommit:
		sha := strings.TrimSpace(ref)
		if sha == "" {
			return nil, errors.New("commit ref is required")
		}
		return map[string]any{"type": "commit", "sha": sha}, nil
	case ChangeScopeBase:
		branch := strings.TrimSpace(base)
		if branch == "" {
			branch = "main"
		}
		return map[string]any{"type": "baseBranch", "branch": branch}, nil
	default:
		return map[string]any{"type": "uncommittedChanges"}, nil
	}
}

func cleanChangePaths(cwd string, files []string) ([]string, error) {
	cleanCWD, err := filepath.Abs(cwd)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	targets := []string{}
	for _, raw := range files {
		normalized := filepath.ToSlash(strings.TrimSpace(raw))
		if normalized == "" || strings.HasPrefix(normalized, "/") || strings.Contains(normalized, "\x00") {
			return nil, errors.New("invalid file path")
		}
		candidate := filepath.Clean(filepath.Join(cleanCWD, filepath.FromSlash(normalized)))
		rel, err := filepath.Rel(cleanCWD, candidate)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			return nil, errors.New("file is outside working directory")
		}
		cleanRel := filepath.ToSlash(rel)
		if _, ok := seen[cleanRel]; ok {
			continue
		}
		seen[cleanRel] = struct{}{}
		targets = append(targets, cleanRel)
	}
	return targets, nil
}

func isGitTracked(ctx context.Context, cwd, filePath string) bool {
	_, err := runGit(ctx, cwd, "ls-files", "--error-unmatch", "--", filePath)
	return err == nil
}

func removeWorkspaceFile(cwd, filePath string) error {
	cleanCWD, err := filepath.Abs(cwd)
	if err != nil {
		return err
	}
	target := filepath.Clean(filepath.Join(cleanCWD, filepath.FromSlash(filePath)))
	rel, err := filepath.Rel(cleanCWD, target)
	if err != nil {
		return err
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return errors.New("file is outside working directory")
	}
	info, err := os.Stat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("refusing to remove directory")
	}
	return os.Remove(target)
}

func buildReviewPrompt(changes SessionChanges) string {
	target := "当前工作区"
	switch changes.Scope {
	case ChangeScopeCommit:
		target = "commit " + strings.TrimSpace(changes.Ref)
	case ChangeScopeBase:
		target = "相对 base branch " + strings.TrimSpace(changes.Base)
	}
	if strings.TrimSpace(target) == "" {
		target = "当前改动"
	}

	var b strings.Builder
	b.WriteString("请进入 Review 模式，审查")
	b.WriteString(target)
	b.WriteString("的代码改动。优先找 bug、行为回归、风险、遗漏测试和移动端体验问题。")
	b.WriteString("请按严重程度列出发现，并引用具体文件路径；如果没有问题，请明确说明剩余风险。\n\n")
	b.WriteString("改动摘要：")
	b.WriteString(fmt.Sprintf("%d 个文件，+%d -%d", changes.Summary.Files, changes.Summary.Additions, changes.Summary.Deletions))
	if changes.Summary.Untracked > 0 {
		b.WriteString(fmt.Sprintf("，%d 个未跟踪文件", changes.Summary.Untracked))
	}
	b.WriteString("\n\n文件列表：\n")
	for _, file := range changes.Files {
		b.WriteString("- ")
		b.WriteString(file.Status)
		b.WriteString(" ")
		b.WriteString(file.Path)
		b.WriteString(fmt.Sprintf(" (+%d -%d)", file.Additions, file.Deletions))
		if file.Binary {
			b.WriteString(" [binary]")
		}
		b.WriteByte('\n')
	}
	b.WriteString("\n下面是 diff：\n\n```diff\n")
	b.WriteString(truncateTextBytes(changes.Diff, 120*1024))
	b.WriteString("\n```")
	return b.String()
}

func readGitChanges(ctx context.Context, cwd string, scope ChangeScope, ref, base, filePath string) (SessionChanges, error) {
	scope = normalizeChangeScope(scope)
	if err := ensureGitWorktree(ctx, cwd); err != nil {
		return SessionChanges{}, err
	}

	diffArgs, statArgs, err := gitChangeArgs(scope, ref, base)
	if err != nil {
		return SessionChanges{}, err
	}
	if strings.TrimSpace(filePath) != "" {
		diffArgs = append(diffArgs, "--", filePath)
		statArgs = append(statArgs, "--", filePath)
	}

	diff, err := runGit(ctx, cwd, diffArgs...)
	if err != nil {
		return SessionChanges{}, err
	}
	stat, err := runGit(ctx, cwd, statArgs...)
	if err != nil {
		return SessionChanges{}, err
	}

	files := filterChangedFiles(ctx, cwd, parseNumstat(stat))
	if scope == ChangeScopeWorkspace {
		untracked, err := listUntrackedFiles(ctx, cwd)
		if err == nil {
			files = mergeUntracked(files, filterChangedFiles(ctx, cwd, untracked))
		}
	}
	if strings.TrimSpace(filePath) != "" && len(files) == 0 {
		files = append(files, ChangedFile{Path: filePath, Status: "M"})
	}

	result := SessionChanges{
		Scope:     scope,
		Ref:       strings.TrimSpace(ref),
		Base:      strings.TrimSpace(base),
		CWD:       cwd,
		Files:     files,
		Diff:      diff,
		Generated: time.Now().UnixMilli(),
	}
	result.Summary = summarizeChanges(files)

	if strings.TrimSpace(filePath) != "" {
		detail := buildChangedFileDetail(ctx, cwd, filePath, diff, files)
		result.File = &detail
	}
	return result, nil
}

func normalizeChangeScope(scope ChangeScope) ChangeScope {
	switch scope {
	case ChangeScopeCommit, ChangeScopeBase, ChangeScopeWorkspace:
		return scope
	default:
		return ChangeScopeWorkspace
	}
}

func gitChangeArgs(scope ChangeScope, ref, base string) ([]string, []string, error) {
	switch scope {
	case ChangeScopeCommit:
		target := strings.TrimSpace(ref)
		if target == "" {
			return nil, nil, errEmptyCommitRef
		}
		return []string{"diff", "--no-ext-diff", "--find-renames", target + "^", target},
			[]string{"diff", "--numstat", "--find-renames", target + "^", target}, nil
	case ChangeScopeBase:
		target := strings.TrimSpace(base)
		if target == "" {
			target = "main"
		}
		return []string{"diff", "--no-ext-diff", "--find-renames", target + "...HEAD"},
			[]string{"diff", "--numstat", "--find-renames", target + "...HEAD"}, nil
	default:
		return []string{"diff", "--no-ext-diff", "--find-renames", "HEAD"},
			[]string{"diff", "--numstat", "--find-renames", "HEAD"}, nil
	}
}

func ensureGitWorktree(ctx context.Context, cwd string) error {
	out, err := runGit(ctx, cwd, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return errors.New("working directory is not a git repository")
	}
	if strings.TrimSpace(out) != "true" {
		return errors.New("working directory is not a git repository")
	}
	return nil
}

func runGit(ctx context.Context, cwd string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cwd
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return string(out), nil
}

func parseNumstat(output string) []ChangedFile {
	files := []ChangedFile{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		additions, binaryAdd := parseNumstatValue(parts[0])
		deletions, binaryDel := parseNumstatValue(parts[1])
		path := strings.Join(parts[2:], "\t")
		oldPath := ""
		if strings.Contains(path, " => ") {
			oldPath, path = parseRenamedPath(path)
		}
		status := "M"
		if oldPath != "" {
			status = "R"
		}
		files = append(files, ChangedFile{
			Path:      path,
			OldPath:   oldPath,
			Status:    status,
			Additions: additions,
			Deletions: deletions,
			Binary:    binaryAdd || binaryDel,
		})
	}
	return files
}

func filterChangedFiles(ctx context.Context, cwd string, files []ChangedFile) []ChangedFile {
	filtered := make([]ChangedFile, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	ignored := gitIgnoredPathSet(ctx, cwd, files)
	for _, file := range files {
		file.Path = normalizeChangePath(file.Path)
		file.OldPath = normalizeChangePath(file.OldPath)
		if _, ok := ignored[file.Path]; ok {
			continue
		}
		if !shouldShowChangedFile(file) {
			continue
		}
		if _, ok := seen[file.Path]; ok {
			continue
		}
		seen[file.Path] = struct{}{}
		filtered = append(filtered, file)
	}
	return filtered
}

func gitIgnoredPathSet(ctx context.Context, cwd string, files []ChangedFile) map[string]struct{} {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		path := normalizeChangePath(file.Path)
		if path != "" {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return map[string]struct{}{}
	}
	cmd := exec.CommandContext(ctx, "git", "check-ignore", "--no-index", "--stdin")
	cmd.Dir = cwd
	cmd.Stdin = strings.NewReader(strings.Join(paths, "\n"))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return map[string]struct{}{}
	}
	ignored := make(map[string]struct{})
	for _, line := range strings.Split(string(out), "\n") {
		path := normalizeChangePath(line)
		if path != "" {
			ignored[path] = struct{}{}
		}
	}
	return ignored
}

func shouldShowChangedFile(file ChangedFile) bool {
	path := normalizeChangePath(file.Path)
	if path == "" || isGeneratedChangePath(path) || !isCodeChangePath(path) {
		return false
	}
	if file.Binary {
		return false
	}
	return file.Additions > 0 || file.Deletions > 0
}

func normalizeChangePath(path string) string {
	value := filepath.ToSlash(strings.TrimSpace(path))
	value = strings.Trim(value, `"`)
	value = strings.TrimPrefix(value, "./")
	value = strings.TrimPrefix(value, "a/")
	value = strings.TrimPrefix(value, "b/")
	return value
}

func isGeneratedChangePath(path string) bool {
	value := normalizeChangePath(path)
	if value == "" {
		return false
	}
	generatedPrefixes := []string{
		"dist/",
		"web/dist/",
		"build/",
		"web/build/",
		"coverage/",
		"web/coverage/",
		"node_modules/",
		"web/node_modules/",
	}
	for _, prefix := range generatedPrefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func isCodeChangePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(normalizeChangePath(path)))
	switch ext {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".vue", ".css", ".scss", ".sass", ".less", ".html", ".rs", ".py", ".java", ".kt", ".swift", ".c", ".cc", ".cpp", ".h", ".hpp", ".cs", ".sql", ".sh", ".ps1", ".bat", ".cmd", ".svelte":
		return true
	default:
		return false
	}
}

func parseNumstatValue(value string) (int, bool) {
	if value == "-" {
		return 0, true
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return n, false
}

func parseRenamedPath(path string) (string, string) {
	if strings.HasPrefix(path, "{") && strings.Contains(path, "} => ") {
		closing := strings.Index(path, "} => ")
		prefix := path[1:closing]
		suffix := path[closing+5:]
		parts := strings.Split(prefix, " => ")
		if len(parts) == 2 {
			return parts[0] + suffix, parts[1] + suffix
		}
	}
	parts := strings.Split(path, " => ")
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "", path
}

func listUntrackedFiles(ctx context.Context, cwd string) ([]ChangedFile, error) {
	out, err := runGit(ctx, cwd, "ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	files := []ChangedFile{}
	for _, line := range strings.Split(out, "\n") {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		additions, binary := countUntrackedFileLines(cwd, path)
		files = append(files, ChangedFile{
			Path:      path,
			Status:    "??",
			Additions: additions,
			Binary:    binary,
			Untracked: true,
		})
	}
	return files, nil
}

func countUntrackedFileLines(cwd, relPath string) (int, bool) {
	cleanCWD, err := filepath.Abs(cwd)
	if err != nil {
		return 0, false
	}
	candidate := filepath.Clean(filepath.Join(cleanCWD, filepath.FromSlash(relPath)))
	rel, err := filepath.Rel(cleanCWD, candidate)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return 0, false
	}
	data, err := os.ReadFile(candidate)
	if err != nil {
		return 0, false
	}
	if !utf8.Valid(data) {
		return 0, true
	}
	if len(data) == 0 {
		return 0, false
	}
	lines := bytes.Count(data, []byte{'\n'})
	if data[len(data)-1] != '\n' {
		lines++
	}
	return lines, false
}

func mergeUntracked(files []ChangedFile, untracked []ChangedFile) []ChangedFile {
	existing := make(map[string]struct{}, len(files))
	for _, file := range files {
		existing[file.Path] = struct{}{}
	}
	for _, file := range untracked {
		if _, ok := existing[file.Path]; ok {
			continue
		}
		files = append(files, file)
	}
	return files
}

func summarizeChanges(files []ChangedFile) ChangeSummary {
	var summary ChangeSummary
	summary.Files = len(files)
	for _, file := range files {
		summary.Additions += file.Additions
		summary.Deletions += file.Deletions
		if file.Untracked {
			summary.Untracked++
		}
	}
	return summary
}

func buildChangedFileDetail(ctx context.Context, cwd, filePath, diff string, files []ChangedFile) ChangedFileDetail {
	base := ChangedFile{Path: filePath, Status: "M"}
	for _, file := range files {
		if filepath.ToSlash(file.Path) == filepath.ToSlash(filePath) {
			base = file
			break
		}
	}
	detail := ChangedFileDetail{
		ChangedFile: base,
		Diff:        diff,
		Readable:    true,
	}

	if base.Binary {
		detail.Readable = false
		detail.Error = "binary file"
		return detail
	}
	content, truncated, err := readWorkspaceFile(cwd, filePath)
	if err != nil {
		detail.Readable = false
		detail.Error = err.Error()
		return detail
	}
	if !utf8.ValidString(content) {
		detail.Readable = false
		detail.Error = "file is not valid UTF-8"
		return detail
	}
	detail.Content = content
	detail.Truncated = truncated
	return detail
}

func readWorkspaceFile(cwd, relPath string) (string, bool, error) {
	cleanCWD, err := filepath.Abs(cwd)
	if err != nil {
		return "", false, err
	}
	candidate := filepath.Clean(filepath.Join(cleanCWD, filepath.FromSlash(relPath)))
	rel, err := filepath.Rel(cleanCWD, candidate)
	if err != nil {
		return "", false, err
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", false, errors.New("file is outside working directory")
	}
	info, err := os.Stat(candidate)
	if err != nil {
		return "", false, err
	}
	if info.IsDir() {
		return "", false, errors.New("path is a directory")
	}
	file, err := os.Open(candidate)
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	limit := maxChangedFileContentBytes + 1
	data := make([]byte, limit)
	n, err := file.Read(data)
	if err != nil && n == 0 {
		return "", false, err
	}
	truncated := n > maxChangedFileContentBytes
	if truncated {
		n = maxChangedFileContentBytes
	}
	return string(data[:n]), truncated, nil
}

func truncateTextBytes(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "\n\n[diff truncated]"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
