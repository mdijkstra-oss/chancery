package parser

import (
	"regexp"
	"strings"
)

const (
	TagLLMCMDPrefix = "<LLMCMD"
	TagLLMCMDClose  = "</LLMCMD>"
	TagPayload      = "<PAYLOAD>"
	TagPayloadClose = "</PAYLOAD>"
)

const (
	TagPayloadOut      = "<payload>"
	TagPayloadCloseOut = "</payload>"
	TagLLMCMDCloseOut  = "</LLMCMD>"
)

var whitespaceRe = regexp.MustCompile(`\s+`)

func NormalizeTag(s string) string {
	s = strings.ToUpper(s)
	s = whitespaceRe.ReplaceAllString(s, " ")
	s = strings.Replace(s, "< ", "<", 1)
	s = strings.Replace(s, "/ ", "/", 1)
	return s
}

type TagMatch int

const (
	NoMatch TagMatch = iota
	PartialMatch
	FullMatch
)

func MatchTag(buffer, tag string) TagMatch {
	norm := NormalizeTag(buffer)
	if norm == tag {
		return FullMatch
	}
	if strings.HasPrefix(tag, norm) {
		return PartialMatch
	}
	return NoMatch
}

func MatchTagWithArgs(buffer, tagPrefix string) TagMatch {
	norm := NormalizeTag(buffer)

	if !isTagStart(norm) {
		return NoMatch
	}

	if isBuildingTagName(norm, tagPrefix) {
		return PartialMatch
	}

	if !hasTagName(norm, tagPrefix) {
		return NoMatch
	}

	if !isTagClosed(norm) {
		return PartialMatch
	}

	return FullMatch
}

func isTagStart(norm string) bool {
	return strings.HasPrefix(norm, "<")
}

func isBuildingTagName(norm, tagPrefix string) bool {
	return strings.HasPrefix(tagPrefix, norm) && len(norm) < len(tagPrefix)
}

func hasTagName(norm, tagPrefix string) bool {
	return strings.HasPrefix(norm, tagPrefix)
}

func isTagClosed(norm string) bool {
	return strings.Contains(norm, ">")
}

func ParseCmdAttrs(openTag string) (isReply bool) {
	norm := NormalizeTag(openTag)
	inner := strings.TrimPrefix(norm, TagLLMCMDPrefix)
	inner = strings.TrimSuffix(inner, ">")
	inner = strings.TrimSpace(inner)
	return strings.Contains(inner, "REPLY")
}

func MatchPayloadOpen(buffer string) TagMatch {
	return MatchTag(buffer, TagPayload)
}

func MatchPayloadClose(buffer string) TagMatch {
	return MatchTag(buffer, TagPayloadClose)
}

func MatchCmdClose(buffer string) TagMatch {
	return MatchTag(buffer, TagLLMCMDClose)
}

func FindTagStart(s string) int {
	return strings.Index(s, "<")
}
