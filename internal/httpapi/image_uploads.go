package httpapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	maxUploadImageBytes = 15 * 1024 * 1024
	imageTTL            = 24 * time.Hour
)

type imageUpload struct {
	ID        string
	Name      string
	Path      string
	Size      int64
	CreatedAt time.Time
}

type imageUploadStore struct {
	mu      sync.Mutex
	baseDir string
	items   map[string]imageUpload
}

func newImageUploadStore() *imageUploadStore {
	return &imageUploadStore{
		baseDir: filepath.Join(os.TempDir(), "codexpocket", "uploads"),
		items:   make(map[string]imageUpload),
	}
}

func (s *imageUploadStore) Save(name string, payload []byte) (imageUpload, error) {
	if len(payload) == 0 {
		return imageUpload{}, errors.New("empty image payload")
	}
	if len(payload) > maxUploadImageBytes {
		return imageUpload{}, errors.New("image exceeds 15MB size limit")
	}
	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return imageUpload{}, err
	}

	id, err := randomUploadID()
	if err != nil {
		return imageUpload{}, err
	}

	ext := normalizeExt(filepath.Ext(strings.TrimSpace(name)))
	if ext == "" {
		ext = ".bin"
	}
	fileName := id + ext
	path := filepath.Join(s.baseDir, fileName)
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return imageUpload{}, err
	}

	item := imageUpload{
		ID:        id,
		Name:      strings.TrimSpace(name),
		Path:      path,
		Size:      int64(len(payload)),
		CreatedAt: time.Now().UTC(),
	}

	s.mu.Lock()
	s.items[id] = item
	s.cleanupLocked(time.Now().UTC())
	s.mu.Unlock()

	return item, nil
}

func (s *imageUploadStore) SaveInline(dataURL string) (imageUpload, bool, error) {
	mediaType, payload, ok := parseInlineImageDataURL(dataURL)
	if !ok {
		return imageUpload{}, false, nil
	}
	if len(payload) == 0 {
		return imageUpload{}, true, errors.New("empty inline image payload")
	}
	if len(payload) > maxUploadImageBytes {
		return imageUpload{}, true, errors.New("inline image exceeds 15MB size limit")
	}
	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return imageUpload{}, true, err
	}

	sum := sha256.Sum256(payload)
	id := "inline-" + hex.EncodeToString(sum[:16])
	ext := extForImageMediaType(mediaType)
	fileName := id + ext
	path := filepath.Join(s.baseDir, fileName)
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.items[id]; ok {
		if _, err := os.Stat(existing.Path); err == nil {
			existing.CreatedAt = now
			s.items[id] = existing
			return existing, true, nil
		}
	}

	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return imageUpload{}, true, err
	}
	item := imageUpload{
		ID:        id,
		Name:      "inline-image" + ext,
		Path:      path,
		Size:      int64(len(payload)),
		CreatedAt: now,
	}
	s.items[id] = item
	s.cleanupLocked(now)
	return item, true, nil
}

func (s *imageUploadStore) Resolve(uploadID string) (string, error) {
	id := strings.TrimSpace(uploadID)
	if id == "" {
		return "", errors.New("upload id is required")
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(now)
	item, ok := s.items[id]
	if !ok {
		return "", errors.New("uploaded image not found or expired")
	}
	if _, err := os.Stat(item.Path); err != nil {
		delete(s.items, id)
		return "", errors.New("uploaded image is unavailable")
	}
	return item.Path, nil
}

func parseInlineImageDataURL(value string) (string, []byte, bool) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(trimmed), "data:image/") {
		return "", nil, false
	}
	header, encoded, ok := strings.Cut(trimmed, ",")
	if !ok || !strings.Contains(strings.ToLower(header), ";base64") {
		return "", nil, false
	}
	mediaType := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(strings.Split(header, ";")[0])), "data:")
	payload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", nil, false
	}
	return mediaType, payload, true
}

func extForImageMediaType(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/bmp":
		return ".bmp"
	default:
		return ".img"
	}
}

func (s *imageUploadStore) cleanupLocked(now time.Time) {
	for id, item := range s.items {
		if now.Sub(item.CreatedAt) <= imageTTL {
			continue
		}
		_ = os.Remove(item.Path)
		delete(s.items, id)
	}
}

func randomUploadID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func normalizeExt(ext string) string {
	trimmed := strings.TrimSpace(strings.ToLower(ext))
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, ".") {
		return trimmed
	}
	return "." + trimmed
}
