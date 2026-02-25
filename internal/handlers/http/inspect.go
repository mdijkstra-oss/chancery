package http

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func inspectJSON(label string, v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		slog.Error("inspect marshal failed", "label", label, "error", err)
		return
	}
	showInBat(label, data, "json")
}

func inspectRawJSON(label string, data []byte) {
	var indented json.RawMessage
	if json.Unmarshal(data, &indented) == nil {
		if pretty, err := json.MarshalIndent(indented, "", "  "); err == nil {
			data = pretty
		}
	}
	showInBat(label, data, "json")
}

func showInBat(label string, data []byte, lang string) {
	safeLabel := strings.ReplaceAll(label, "/", "-")
	safeLabel = strings.ReplaceAll(safeLabel, " ", "-")
	path := filepath.Join(os.TempDir(), fmt.Sprintf("hermes-%s.json", safeLabel))
	if err := os.WriteFile(path, data, 0644); err != nil {
		slog.Error("inspect write failed", "label", label, "error", err)
		return
	}

	slog.Info("inspect", "label", label, "path", path)

	cmd := exec.Command("bat", "--language", lang, "--paging", "never", "--file-name", label, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		slog.Error("inspect bat failed", "label", label, "error", err)
	}
}
