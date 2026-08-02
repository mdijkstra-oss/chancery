package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

var errStreamFailed = errors.New("backend stream failed")

// textWriter renders an event stream as the text a terminal shows. It is the only
// place chancery decodes an event, and it decodes only the three the terminal needs.
type textWriter struct {
	destination io.Writer
	buffer      []byte
	failure     error
	completed   bool
}

func newTextWriter(destination io.Writer) *textWriter {
	return &textWriter{destination: destination}
}

// A partial event is held until its terminator arrives, so text reaches the terminal
// as the backend emits it rather than once the stream is over.
func (w *textWriter) Write(data []byte) (int, error) {
	w.buffer = append(w.buffer, data...)
	if err := w.writeCompleteEvents(); err != nil {
		return 0, err
	}
	return len(data), nil
}

// Close is where a stream that stopped early becomes a non-zero exit: text already
// written is indistinguishable from an answer unless the terminal condition is read.
func (w *textWriter) Close() error {
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
		return errors.New("backend stream ended before response.completed")
	}
	return nil
}

func (w *textWriter) writeCompleteEvents() error {
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

func (w *textWriter) writeEvent(event []byte) error {
	lines := strings.Split(string(event), "\n")
	if _, err := io.WriteString(w.destination, outputText(lines)); err != nil {
		return fmt.Errorf("write text: %w", err)
	}
	if eventType(lines) == "response.completed" {
		w.completed = true
	}
	if w.failure == nil {
		w.failure = failureFrom(lines)
	}
	return nil
}

// Only output deltas are shown. A reasoning summary is the model's working, not its
// answer, so it travels to the client and stops here.
func outputText(lines []string) string {
	text := ""
	current := ""
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "event: "):
			current = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: ") && current == "response.output_text.delta":
			text += extractDelta(strings.TrimPrefix(line, "data: "))
		case line == "":
			current = ""
		}
	}
	return text
}

func eventType(lines []string) string {
	for _, line := range lines {
		if strings.HasPrefix(line, "event: ") {
			return strings.TrimPrefix(line, "event: ")
		}
	}
	return ""
}

// A failure the backend describes is reported as it described it; one it does not is
// still a failure.
func failureFrom(lines []string) error {
	failed := false
	for _, line := range lines {
		if line == "event: response.failed" {
			failed = true
			continue
		}
		if !failed || !strings.HasPrefix(line, "data: ") {
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
			return errStreamFailed
		}
		if payload.Response.Error.Message == "" {
			return errStreamFailed
		}
		return fmt.Errorf("%w: %s: %s",
			errStreamFailed, payload.Response.Error.Type, payload.Response.Error.Message)
	}
	return nil
}

func extractDelta(data string) string {
	var payload struct {
		Delta string `json:"delta"`
	}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return ""
	}
	return payload.Delta
}
