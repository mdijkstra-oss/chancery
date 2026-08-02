package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mdijkstra-oss/chancery/internal/prompts"
)

func loadRegistry(configPath string) (prompts.Registry, error) {
	registry, report := prompts.Load(configPath)
	if err := configReportError(report); err != nil {
		return prompts.Registry{}, err
	}
	return registry, nil
}

func configReportError(report prompts.Report) error {
	messages := make([]string, 0, report.ErrorCount())
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Severity == prompts.SeverityError {
			messages = append(messages, fmt.Sprintf("✗ %s: %s", diagnostic.Path, diagnostic.Message))
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return errors.New(strings.Join(messages, "\n"))
}
