package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"codexpocket/internal/config"
	"codexpocket/internal/runtime"
)

type Server struct {
	agent   *runtime.Agent
	logger  *slog.Logger
	mux     *http.ServeMux
	uploads *imageUploadStore
	jwt     *JWTService
	cfg     config.Config
}

func NewServer(agent *runtime.Agent, logger *slog.Logger, cfg config.Config) *Server {
	server := &Server{
		agent:   agent,
		logger:  logger,
		mux:     http.NewServeMux(),
		uploads: newImageUploadStore(),
		jwt:     NewJWTService(cfg),
		cfg:     cfg,
	}
	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	var handler http.Handler = s.mux
	handler = s.withAuth(handler)
	handler = s.withCORS(handler)
	handler = s.withLogging(handler)
	return handler
}

func (s *Server) routes() {
	// Public routes (no auth required)
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/api/v1/auth/login", s.handleLogin)

	// API routes (auth required, applied via middleware)
	s.mux.HandleFunc("/api/v1/dashboard", s.handleDashboard)
	s.mux.HandleFunc("/api/v1/options", s.handleOptions)
	s.mux.HandleFunc("/api/v1/events", s.handleEvents)
	s.mux.HandleFunc("/api/v1/directories", s.handleDirectories)
	s.mux.HandleFunc("/api/v1/sessions", s.handleSessions)
	s.mux.HandleFunc("/api/v1/sessions/", s.handleSessionByID)
	s.mux.HandleFunc("/api/v1/approvals", s.handleApprovals)
	s.mux.HandleFunc("/api/v1/approvals/", s.handleApprovalByID)
	s.mux.HandleFunc("/api/v1/uploads/image", s.handleImageUpload)
	s.mux.HandleFunc("/api/v1/assets/local-image", s.handleLocalImage)

	// Static file serving for web UI
	if s.cfg.WebDistPath != "" {
		s.serveStaticFiles()
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"timestamp": time.Now(),
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	if !s.cfg.Authenticate(request.Username, request.Password) {
		writeErrorMessage(w, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := s.jwt.GenerateToken(request.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":    token,
		"username": request.Username,
	})
}

func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Public paths that don't require authentication
		publicPaths := map[string]bool{
			"/healthz":           true,
			"/api/v1/auth/login": true,
		}
		if publicPaths[path] {
			next.ServeHTTP(w, r)
			return
		}

		// Static files also don't require auth at HTTP level (SPA handles login redirect)
		if !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Check JWT token from Authorization header or query parameter
		authHeader := r.Header.Get("Authorization")
		var token string
		if authHeader != "" {
			token = strings.TrimPrefix(authHeader, "Bearer ")
			if token == authHeader {
				writeErrorMessage(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}
		} else {
			// Fallback to query parameter (needed for EventSource which doesn't support headers)
			token = r.URL.Query().Get("token")
			if token == "" {
				writeErrorMessage(w, http.StatusUnauthorized, "authorization header required")
				return
			}
		}

		claims, valid := s.jwt.ValidateToken(token)
		if !valid {
			writeErrorMessage(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		// Add username to request context
		ctx := context.WithValue(r.Context(), "username", claims.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) serveStaticFiles() {
	distPath := s.cfg.WebDistPath
	fs := http.FileServer(http.Dir(distPath))

	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Don't interfere with API routes
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the file, fall back to index.html for SPA routing
		filePath := filepath.Join(distPath, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html
		r.URL.Path = "/"
		fs.ServeHTTP(w, r)
	})

	s.logger.Info("serving web UI from", "path", distPath)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, s.agent.Dashboard())
}

func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	writeJSON(w, http.StatusOK, s.agent.Options(ctx))
}

func (s *Server) handleDirectories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	result, err := s.agent.BrowseDirectories(r.URL.Query().Get("path"))
	if err != nil {
		writeErrorMessage(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{
			"data": s.agent.ListSessions(),
		})
	case http.MethodPost:
		var request struct {
			Action            string `json:"action"`
			CWD               string `json:"cwd"`
			Prompt            string `json:"prompt"`
			Agent             string `json:"agent"`
			Model             string `json:"model"`
			ReasoningEffort   string `json:"reasoningEffort"`
			CollaborationMode string `json:"collaborationMode"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		switch request.Action {
		case "refresh":
			if err := s.agent.Refresh(ctx); err != nil {
				writeError(w, http.StatusBadGateway, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		case "start":
			cwd := normalizeCWD(request.CWD)
			prompt := strings.TrimSpace(request.Prompt)

			if cwd == "" {
				writeErrorMessage(w, http.StatusBadRequest, "working directory is required")
				return
			}
			if !filepath.IsAbs(cwd) {
				writeErrorMessage(w, http.StatusBadRequest, "working directory must be an absolute path")
				return
			}
			if prompt == "" {
				writeErrorMessage(w, http.StatusBadRequest, "first prompt is required to materialize a managed session")
				return
			}
			session, err := s.agent.StartSession(ctx, cwd, prompt, request.Agent, runtime.StartSessionOptions{
				Model:             request.Model,
				ReasoningEffort:   request.ReasoningEffort,
				CollaborationMode: request.CollaborationMode,
			})
			if err != nil {
				writeError(w, http.StatusBadGateway, err)
				return
			}
			writeJSON(w, http.StatusCreated, session)
		default:
			writeErrorMessage(w, http.StatusBadRequest, "unsupported sessions action")
		}
	default:
		methodNotAllowed(w)
	}
}

func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/sessions/")
	if path == "" {
		writeErrorMessage(w, http.StatusNotFound, "session not found")
		return
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	sessionID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		offset := -1
		if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				writeErrorMessage(w, http.StatusBadRequest, "invalid offset")
				return
			}
			offset = parsed
		}
		limit := 0
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				writeErrorMessage(w, http.StatusBadRequest, "invalid limit")
				return
			}
			limit = parsed
		}

		fast := strings.TrimSpace(r.URL.Query().Get("fast")) == "1" || strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("fast")), "true")
		var detail runtime.SessionDetail
		var err error
		if fast {
			detail, err = s.agent.FastSessionDetail(sessionID, offset, limit)
		} else {
			detail, err = s.agent.SessionDetail(ctx, sessionID, offset, limit)
		}
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, detail)
		return
	}

	action := strings.Join(parts[1:], "/")
	switch action {
	case "resume":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		session, err := s.agent.ResumeSession(ctx, sessionID)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, session)
	case "detach":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		if err := s.agent.DetachSession(ctx, sessionID); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "end":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		if err := s.agent.EndSession(ctx, sessionID); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "archive":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		if err := s.agent.ArchiveSession(ctx, sessionID); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "rename":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		var request struct {
			Name string `json:"name"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		summary, err := s.agent.RenameSession(ctx, sessionID, request.Name)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, summary)
	case "fork":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		summary, err := s.agent.ForkSession(ctx, sessionID)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusCreated, summary)
	case "compact":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		if err := s.agent.CompactSession(ctx, sessionID); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "rollback":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		var request struct {
			NumTurns int `json:"numTurns"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		detail, err := s.agent.RollbackSession(ctx, sessionID, request.NumTurns)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, detail)
	case "goal":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		var request struct {
			Objective   string `json:"objective"`
			Status      string `json:"status"`
			TokenBudget int64  `json:"tokenBudget"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		goal, err := s.agent.SetSessionGoal(ctx, sessionID, request.Objective, request.Status, request.TokenBudget)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, goal)
	case "goal/clear":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		if err := s.agent.ClearSessionGoal(ctx, sessionID); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "changes":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		changes, err := s.agent.SessionChanges(
			ctx,
			sessionID,
			runtime.ChangeScope(strings.TrimSpace(r.URL.Query().Get("scope"))),
			r.URL.Query().Get("ref"),
			r.URL.Query().Get("base"),
			r.URL.Query().Get("file"),
		)
		if err != nil {
			status := http.StatusBadGateway
			if isClientFacingChangeError(err) {
				status = http.StatusBadRequest
			}
			writeErrorMessage(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, changes)
	case "changes/revert":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		var request runtime.RevertChangesRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		changes, err := s.agent.RevertSessionChanges(ctx, sessionID, request.Files)
		if err != nil {
			status := http.StatusBadGateway
			if isClientFacingChangeError(err) {
				status = http.StatusBadRequest
			}
			writeErrorMessage(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, changes)
	case "review":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		var request runtime.ReviewStartRequest
		if !decodeJSON(w, r, &request) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		turn, err := s.agent.StartReview(ctx, sessionID, request)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusCreated, turn)
	case "turns/start":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		var request struct {
			Prompt string `json:"prompt"`
			Inputs []struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				UploadID string `json:"uploadId"`
			} `json:"inputs"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		input, err := s.buildTurnInput(request.Prompt, request.Inputs)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, err.Error())
			return
		}
		turn, err := s.agent.StartTurn(ctx, sessionID, input)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusCreated, turn)
	case "turns/steer":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		var request struct {
			TurnID string `json:"turnId"`
			Prompt string `json:"prompt"`
			Inputs []struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				UploadID string `json:"uploadId"`
			} `json:"inputs"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		input, err := s.buildTurnInput(request.Prompt, request.Inputs)
		if err != nil {
			writeErrorMessage(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.agent.SteerTurn(ctx, sessionID, request.TurnID, input); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	case "turns/interrupt":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		var request struct {
			TurnID string `json:"turnId"`
		}
		if !decodeJSON(w, r, &request) {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		if err := s.agent.InterruptTurn(ctx, sessionID, request.TurnID); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	default:
		writeErrorMessage(w, http.StatusNotFound, fmt.Sprintf("unsupported session action %q", action))
	}
}

func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": s.agent.PendingRequests(),
	})
}

func (s *Server) handleApprovalByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1/approvals/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[1] != "resolve" {
		writeErrorMessage(w, http.StatusNotFound, "approval endpoint not found")
		return
	}

	var request struct {
		Result json.RawMessage `json:"result"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := s.agent.ResolveRequest(ctx, parts[0], request.Result); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErrorMessage(w, http.StatusInternalServerError, "streaming is not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	subscription := s.agent.Subscribe()
	defer s.agent.Unsubscribe(subscription)

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-subscription:
			data, _ := json.Marshal(event)
			_, _ = fmt.Fprintf(w, "event: %s\n", event.Type)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ticker.C:
			_, _ = fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (s *Server) handleImageUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if err := r.ParseMultipartForm(maxUploadImageBytes + (1 * 1024 * 1024)); err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "invalid multipart form payload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "missing image file in multipart field 'file'")
		return
	}
	defer file.Close()

	payload, err := io.ReadAll(io.LimitReader(file, maxUploadImageBytes+1))
	if err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "failed to read uploaded image")
		return
	}
	if len(payload) == 0 {
		writeErrorMessage(w, http.StatusBadRequest, "uploaded image is empty")
		return
	}
	if len(payload) > maxUploadImageBytes {
		writeErrorMessage(w, http.StatusBadRequest, "image exceeds 15MB size limit")
		return
	}
	if !strings.HasPrefix(http.DetectContentType(payload), "image/") {
		writeErrorMessage(w, http.StatusBadRequest, "uploaded file must be an image")
		return
	}

	name := strings.TrimSpace(header.Filename)
	if name == "" {
		name = "upload-image"
	}
	item, err := s.uploads.Save(name, payload)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":   item.ID,
		"name": item.Name,
		"size": item.Size,
	})
}

func (s *Server) handleLocalImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	path := strings.TrimSpace(r.URL.Query().Get("path"))
	if path == "" {
		writeErrorMessage(w, http.StatusBadRequest, "image path is required")
		return
	}
	if uploadID, ok := strings.CutPrefix(path, "upload:"); ok {
		resolved, err := s.uploads.Resolve(uploadID)
		if err != nil {
			writeErrorMessage(w, http.StatusNotFound, "uploaded image not found or expired")
			return
		}
		path = resolved
	}

	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			writeErrorMessage(w, http.StatusNotFound, "image not found")
			return
		}
		writeError(w, http.StatusBadGateway, err)
		return
	}
	defer file.Close()

	head := make([]byte, 512)
	n, err := file.Read(head)
	if err != nil && err != io.EOF {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	contentType := http.DetectContentType(head[:n])
	if !strings.HasPrefix(contentType, "image/") {
		writeErrorMessage(w, http.StatusBadRequest, "requested file is not an image")
		return
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	info, err := file.Stat()
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=60")
	http.ServeContent(w, r, filepath.Base(path), info.ModTime(), file)
}

func (s *Server) buildTurnInput(
	legacyPrompt string,
	inputs []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		UploadID string `json:"uploadId"`
	},
) ([]map[string]any, error) {
	if len(inputs) == 0 {
		prompt := strings.TrimSpace(legacyPrompt)
		if prompt == "" {
			return nil, fmt.Errorf("prompt or inputs is required")
		}
		return []map[string]any{composeTextInput(prompt)}, nil
	}

	result := make([]map[string]any, 0, len(inputs))
	for _, input := range inputs {
		switch strings.TrimSpace(input.Type) {
		case "text":
			text := strings.TrimSpace(input.Text)
			if text == "" {
				return nil, fmt.Errorf("text input cannot be empty")
			}
			result = append(result, composeTextInput(text))
		case "image":
			path, err := s.uploads.Resolve(input.UploadID)
			if err != nil {
				return nil, err
			}
			result = append(result, map[string]any{
				"type": "localImage",
				"path": path,
			})
		default:
			return nil, fmt.Errorf("unsupported input type %q", input.Type)
		}
	}
	return result, nil
}

func composeTextInput(prompt string) map[string]any {
	return map[string]any{
		"type":          "text",
		"text":          prompt,
		"text_elements": []any{},
	}
}

func (s *Server) withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("http request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, Cache-Control")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeErrorMessage(w, http.StatusMethodNotAllowed, "method not allowed")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		writeErrorMessage(w, http.StatusBadRequest, "invalid json body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeErrorMessage(w, status, err.Error())
}

func writeErrorMessage(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": message,
	})
}

func isClientFacingChangeError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "session not found"),
		strings.Contains(msg, "working directory is unknown"),
		strings.Contains(msg, "is not a git repository"),
		strings.Contains(msg, "commit ref is required"):
		return true
	}
	return false
}

func normalizeCWD(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	if trimmed == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return trimmed
	}

	if strings.HasPrefix(trimmed, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(trimmed, "~/"))
		}
	}

	return trimmed
}

func isAllowedOrigin(origin string) bool {
	override := strings.TrimSpace(os.Getenv("CODEXPOCKET_ALLOWED_ORIGINS"))
	if override != "" {
		return matchesAllowedOrigins(origin, override)
	}

	return strings.HasPrefix(origin, "http://localhost:") ||
		strings.HasPrefix(origin, "http://127.0.0.1:") ||
		strings.HasPrefix(origin, "http://[::1]:") ||
		strings.HasPrefix(origin, "https://localhost:") ||
		strings.HasPrefix(origin, "https://127.0.0.1:") ||
		strings.HasPrefix(origin, "https://[::1]:") ||
		strings.HasPrefix(origin, "chrome-extension://")
}

func matchesAllowedOrigins(origin, raw string) bool {
	for _, entry := range strings.Split(raw, ",") {
		pattern := strings.TrimSpace(entry)
		if pattern == "" {
			continue
		}
		if pattern == "*" || pattern == origin {
			return true
		}
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		}
	}
	return false
}
