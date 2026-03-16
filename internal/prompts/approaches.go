package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Approach struct {
	Key         string
	Content     string
	Description string
}

type ApproachRegistry struct {
	Entries      map[string]Approach
	Keys         []string
	Descriptions map[string]string
}

func KeyFromPath(relPath string) string {
	return strings.TrimSuffix(relPath, ".md")
}

func ExtractDescription(content string) string {
	if !strings.HasPrefix(content, "---\n") {
		return ""
	}
	end := strings.Index(content[4:], "\n---")
	if end == -1 {
		return ""
	}
	frontmatter := content[4 : 4+end]
	for _, line := range strings.Split(frontmatter, "\n") {
		if strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return ""
}

func IsIndexKey(key string) bool {
	return key == "index" || strings.HasSuffix(key, "/index")
}

func ParentIndexKeys(key string) []string {
	parts := strings.Split(key, "/")
	keys := make([]string, 0, len(parts)-1)
	for i := 1; i < len(parts); i++ {
		keys = append(keys, strings.Join(parts[:i], "/")+"/index")
	}
	return keys
}

func CollectIndexKeys(keys []string) []string {
	set := map[string]bool{"index": true}
	for _, key := range keys {
		for _, idx := range ParentIndexKeys(key) {
			set[idx] = true
		}
	}
	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}
	slices.Sort(result)
	return result
}

func ResolveApproachKeys(requested []string) []string {
	indexes := CollectIndexKeys(requested)
	sorted := make([]string, len(requested))
	copy(sorted, requested)
	slices.Sort(sorted)

	ordered := make([]string, 0, len(indexes)+len(sorted))
	ordered = append(ordered, indexes...)
	ordered = append(ordered, sorted...)

	seen := make(map[string]bool, len(ordered))
	result := make([]string, 0, len(ordered))
	for _, key := range ordered {
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, key)
	}
	return result
}

func compileApproaches(promptsDir string) ApproachRegistry {
	approachesDir := filepath.Join(promptsDir, "modes", "approaches")
	entries := make(map[string]Approach)
	var keys []string
	descriptions := make(map[string]string)

	if _, statErr := os.Stat(approachesDir); os.IsNotExist(statErr) {
		return ApproachRegistry{Entries: entries, Keys: keys, Descriptions: descriptions}
	}

	err := filepath.Walk(approachesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(approachesDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		key := KeyFromPath(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read approach %s: %w", key, err)
		}
		content := strings.TrimRight(string(data), "\n")
		desc := ExtractDescription(content)

		entries[key] = Approach{Key: key, Content: content, Description: desc}

		if !IsIndexKey(key) {
			keys = append(keys, key)
			if desc != "" {
				descriptions[key] = desc
			}
		}
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("compile approaches: %v", err))
	}

	slices.Sort(keys)
	return ApproachRegistry{Entries: entries, Keys: keys, Descriptions: descriptions}
}
