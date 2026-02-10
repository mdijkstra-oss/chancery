package prompts

import "strings"

type Frontmatter struct {
	Requires []string
}

func ParseFrontmatter(content string) (Frontmatter, string) {
	if !strings.HasPrefix(content, "---\n") {
		return Frontmatter{}, content
	}

	rest := content[4:]
	end := strings.Index(rest, "---\n")
	if end == -1 {
		return Frontmatter{}, content
	}

	block := rest[:end]
	body := rest[end+4:]

	var requires []string
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			requires = append(requires, strings.TrimSpace(trimmed[2:]))
		}
	}

	return Frontmatter{Requires: requires}, body
}

func HasRequired(requires, available []string) bool {
	if len(requires) == 0 {
		return true
	}
	set := toSet(available)
	for _, r := range requires {
		if set[r] {
			return true
		}
	}
	return false
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}
