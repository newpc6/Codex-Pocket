package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

func (a *Agent) BrowseDirectories(path string) (DirectoryBrowseResult, error) {
	current, err := resolveBrowsePath(path)
	if err != nil {
		return DirectoryBrowseResult{}, err
	}

	info, err := os.Stat(current)
	if err != nil {
		return DirectoryBrowseResult{}, fmt.Errorf("stat directory: %w", err)
	}
	if !info.IsDir() {
		return DirectoryBrowseResult{}, fmt.Errorf("path is not a directory")
	}

	entries, err := os.ReadDir(current)
	if err != nil {
		return DirectoryBrowseResult{}, fmt.Errorf("read directory: %w", err)
	}

	result := DirectoryBrowseResult{
		CurrentPath: current,
		ParentPath:  parentPath(current),
		HomePath:    homePath(),
		Roots:       rootEntries(),
		Entries:     make([]DirectoryEntry, 0, len(entries)),
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if name == "" {
			continue
		}
		fullPath := filepath.Join(current, name)
		readable := true
		if _, err := os.ReadDir(fullPath); err != nil {
			readable = false
		}
		result.Entries = append(result.Entries, DirectoryEntry{
			Name:       name,
			Path:       fullPath,
			IsDir:      true,
			IsReadable: readable,
		})
	}

	slices.SortFunc(result.Entries, func(a, b DirectoryEntry) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	return result, nil
}

func resolveBrowsePath(path string) (string, error) {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		normalized = homePath()
	}
	if normalized == "" {
		var err error
		normalized, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("resolve working directory: %w", err)
		}
	}
	if normalized == "~" {
		normalized = homePath()
	} else if strings.HasPrefix(normalized, "~/") {
		normalized = filepath.Join(homePath(), strings.TrimPrefix(normalized, "~/"))
	}

	clean := filepath.Clean(normalized)
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil && strings.TrimSpace(resolved) != "" {
		return resolved, nil
	}
	return abs, nil
}

func parentPath(path string) string {
	clean := filepath.Clean(path)
	parent := filepath.Dir(clean)
	if parent == "." || parent == clean {
		return ""
	}
	if runtime.GOOS == "windows" && strings.EqualFold(parent, clean) {
		return ""
	}
	return parent
}

func homePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(home)
}

func rootEntries() []DirectoryEntry {
	if runtime.GOOS == "windows" {
		roots := make([]DirectoryEntry, 0, 8)
		for letter := 'A'; letter <= 'Z'; letter++ {
			root := fmt.Sprintf("%c:\\", letter)
			if info, err := os.Stat(root); err == nil && info.IsDir() {
				roots = append(roots, DirectoryEntry{
					Name:       root,
					Path:       root,
					IsDir:      true,
					IsRoot:     true,
					IsReadable: true,
				})
			}
		}
		return roots
	}

	return []DirectoryEntry{{
		Name:       "/",
		Path:       "/",
		IsDir:      true,
		IsRoot:     true,
		IsReadable: true,
	}}
}
