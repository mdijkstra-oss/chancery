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
	name, payload := decodeEvent(event)
	// Only output deltas are shown. A reasoning summary is the model's working,
	// not its answer, so it travels to the client and stops here.
	if name == "response.output_text.delta" {
		if _, err := io.WriteString(w.destination, payload.Delta); err != nil {
			return fmt.Errorf("write text: %w", err)
		}
	}
	if name == "response.completed" {
		w.completed = true
	}
	if name == "response.failed" && w.failure == nil {
		w.failure = failureOf(payload)
	}
	return nil
}

// eventPayload is the part of an event the terminal reads. Everything else in the
// frame is content for the caller and is never decoded here.
type eventPayload struct {
	Type     string `json:"type"`
	Delta    string `json:"delta"`
	Response struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"response"`
}

// An event names its type twice: once on the SSE event line, and again in the
// payload. Either alone is enough, because a backend may frame with the line and
// a backend may leave framing bare, and a stream that carries neither names an
// event this renderer has nothing to do with.
func decodeEvent(event []byte) (string, eventPayload) {
	name := ""
	data := ""
	for _, line := range strings.Split(string(event), "\n") {
		switch {
		case strings.HasPrefix(line, "event: "):
			name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			data += strings.TrimPrefix(line, "data: ")
		}
	}
	var payload eventPayload
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return name, eventPayload{}
	}
	if name == "" {
		name = payload.Type
	}
	return name, payload
}

// A failure the backend describes is reported as it described it; one it does not is
// still a failure.
func failureOf(payload eventPayload) error {
	if payload.Response.Error.Message == "" {
		return errStreamFailed
	}
	return fmt.Errorf("%w: %s: %s",
		errStreamFailed, payload.Response.Error.Type, payload.Response.Error.Message)
}
