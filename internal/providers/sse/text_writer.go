package sse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type TextWriter struct {
	destination io.Writer
	buffer      []byte
	failure     error
	completed   bool
}

func NewTextWriter(destination io.Writer) *TextWriter {
	return &TextWriter{destination: destination}
}

func (w *TextWriter) Write(data []byte) (int, error) {
	w.buffer = append(w.buffer, data...)
	if err := w.writeCompleteEvents(); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (w *TextWriter) Close() error {
	if len(w.buffer) > 0 {
		if err := w.writeEvent(w.buffer); err != nil {
			return err
		}
		w.buffer = nil
	}
	if w.failure != nil {
		return w.failure
	}
	if !w.completed {
		return fmt.Errorf("provider stream ended before response.completed")
	}
	return nil
}

func (w *TextWriter) writeCompleteEvents() error {
	for {
		index := bytes.Index(w.buffer, []byte("\n\n"))
		if index == -1 {
			return nil
		}
		end := index + 2
		if err := w.writeEvent(w.buffer[:end]); err != nil {
			return err
		}
		w.buffer = w.buffer[end:]
	}
}

func (w *TextWriter) writeEvent(event []byte) error {
	response := ParseSSEOutput(event)
	if _, err := io.WriteString(w.destination, response.Text); err != nil {
		return err
	}
	eventType := parseEventType(event)
	if eventType == "response.completed" {
		w.completed = true
	}
	if w.failure == nil {
		w.failure = parseFailure(event)
	}
	return nil
}

func parseEventType(event []byte) string {
	for _, line := range strings.Split(string(event), "\n") {
		if strings.HasPrefix(line, "event: ") {
			return strings.TrimPrefix(line, "event: ")
		}
	}
	return ""
}

func parseFailure(event []byte) error {
	lines := strings.Split(string(event), "\n")
	isFailure := false
	for _, line := range lines {
		if line == "event: response.failed" {
			isFailure = true
			continue
		}
		if !isFailure || !strings.HasPrefix(line, "data: ") {
			continue
		}
		var payload struct {
			Response struct {
				Error struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				} `json:"error"`
			} `json:"response"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
			return fmt.Errorf("provider stream failed")
		}
		if payload.Response.Error.Message == "" {
			return fmt.Errorf("provider stream failed")
		}
		return fmt.Errorf("provider stream failed: %s: %s", payload.Response.Error.Type, payload.Response.Error.Message)
	}
	return nil
}
