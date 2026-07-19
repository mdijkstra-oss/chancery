package sse

import (
	"bufio"
	"io"
	"strings"
)

const MaxLineBytes = 1024 * 1024

func NewScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, MaxLineBytes), MaxLineBytes)
	return scanner
}

func EventField(line string) (string, bool) {
	return field(line, "event: ")
}

func DataField(line string) (string, bool) {
	return field(line, "data: ")
}

func field(line, prefix string) (string, bool) {
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	return strings.TrimPrefix(line, prefix), true
}
