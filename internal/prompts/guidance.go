package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Guidance struct {
	Key         string
	Content     string
	Description string
}

type GuidanceRegistry struct {
	Entries      map[string]Guidance
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

func compileGuidance(promptsDir string) GuidanceRegistry {
	guidanceDir := filepath.Join(promptsDir, "modes", "guidance")
	entries := make(map[string]Guidance)
	var keys []string
	descriptions := make(map[string]string)

	if _, statErr := os.Stat(guidanceDir); os.IsNotExist(statErr) {
		return GuidanceRegistry{Entries: entries, Keys: keys, Descriptions: descriptions}
	}

	err := filepath.Walk(guidanceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(guidanceDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		key := KeyFromPath(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read guidance %s: %w", key, err)
		}
		content := strings.TrimRight(string(data), "\n")
		desc := ExtractDescription(content)

		entries[key] = Guidance{Key: key, Content: content, Description: desc}
		keys = append(keys, key)
		if desc != "" {
			descriptions[key] = desc
		}
		return nil
	})
	if err != nil {
		panic(fmt.Sprintf("compile guidance: %v", err))
	}

	slices.Sort(keys)
	return GuidanceRegistry{Entries: entries, Keys: keys, Descriptions: descriptions}
}
