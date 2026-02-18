package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"hermes-logos/internal/prompts"
)

type cliArgs struct {
	agentPath string
	tools     []string
	collapse  []string
	watch     bool
}

func main() {
	args := parseArgs(os.Args[1:])

	output(args)

	if args.watch {
		pollAndReoutput(args)
	}
}

func parseArgs(raw []string) cliArgs {
	var result cliArgs
	var positional []string

	for i := 0; i < len(raw); i++ {
		if val, n := matchFlag(raw, i, "tools"); n > 0 {
			result.tools = parseCSV(val)
			i += n - 1
			continue
		}
		if val, n := matchFlag(raw, i, "collapse"); n > 0 {
			result.collapse = parseCSV(val)
			i += n - 1
			continue
		}
		if raw[i] == "--watch" || raw[i] == "-watch" {
			result.watch = true
			continue
		}
		positional = append(positional, raw[i])
	}

	if len(positional) < 1 {
		result.agentPath = pickAgent()
	} else {
		result.agentPath = positional[0]
	}
	return result
}

func pickAgent() string {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "no agent specified and no terminal available")
		os.Exit(1)
	}
	defer tty.Close()

	registry := prompts.CompileRegistry(prompts.PromptsDir)
	keys := sortedKeys(registry.Agents)

	fmt.Fprintln(tty, "pick an agent:")
	for i, k := range keys {
		fmt.Fprintf(tty, "  %d) %s\n", i+1, k)
	}
	fmt.Fprint(tty, "> ")

	scanner := bufio.NewScanner(tty)
	if !scanner.Scan() {
		os.Exit(1)
	}
	input := strings.TrimSpace(scanner.Text())

	var idx int
	if _, err := fmt.Sscanf(input, "%d", &idx); err == nil && idx >= 1 && idx <= len(keys) {
		return keys[idx-1]
	}

	for _, k := range keys {
		if strings.Contains(k, input) {
			return k
		}
	}

	fmt.Fprintf(tty, "no match for %q\n", input)
	os.Exit(1)
	return ""
}

func matchFlag(args []string, i int, name string) (string, int) {
	for _, prefix := range []string{"--" + name, "-" + name} {
		if args[i] == prefix && i+1 < len(args) {
			return args[i+1], 2
		}
		if strings.HasPrefix(args[i], prefix+"=") {
			return args[i][len(prefix)+1:], 1
		}
	}
	return "", 0
}

func output(args cliArgs) {
	var skip func(string) bool
	if len(args.collapse) > 0 {
		skip = buildSkip(args.collapse)
	}

	agent, err := prompts.CompileAgent(prompts.PromptsDir, args.agentPath, skip)
	if err != nil {
		registry := prompts.CompileRegistry(prompts.PromptsDir)
		fmt.Fprintf(os.Stderr, "unknown agent: %s\n\navailable:\n", args.agentPath)
		for _, k := range sortedKeys(registry.Agents) {
			fmt.Fprintf(os.Stderr, "  %s\n", k)
		}
		os.Exit(1)
	}

	segments := agent.Segments

	if len(args.tools) > 0 {
		toolSegments, err := prompts.LoadToolSegments(filepath.Join(prompts.PromptsDir, "tools"), args.tools)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tool prompt error: %v\n", err)
			os.Exit(1)
		}
		segments = append(segments, toolSegments...)
	}

	fmt.Print(formatAnnotated(segments))
}

func formatAnnotated(segments []prompts.Segment) string {
	var b strings.Builder
	for _, s := range segments {
		if s.Source != "" {
			b.WriteString("──── ")
			b.WriteString(s.Source)
			b.WriteString(" ────\n")
		}
		b.WriteString(strings.TrimRight(s.Content, "\n"))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func buildSkip(patterns []string) func(string) bool {
	return func(includePath string) bool {
		for _, p := range patterns {
			if strings.Contains(includePath, p) {
				return true
			}
		}
		return false
	}
}

func pollAndReoutput(args cliArgs) {
	lastMtime := latestMtime(prompts.PromptsDir)
	for {
		time.Sleep(500 * time.Millisecond)
		current := latestMtime(prompts.PromptsDir)
		if current.After(lastMtime) {
			lastMtime = current
			fmt.Println("\n--- recompiled ---")
			output(args)
		}
	}
}

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func sortedKeys(m map[string]prompts.CompiledAgent) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func latestMtime(dir string) time.Time {
	var latest time.Time
	filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return latest
}
