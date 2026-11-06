package spa

import (
	"slices"
	"strings"
)

type scriptElement struct {
	attributes []string
	text       string
}

// scanScripts finds every script element in page, reading attributes the way
// the HTML tokenizer does, so a quoted value holding ">" or "src=" cannot end
// the tag early or pass for a src attribute.
func scanScripts(page string) []scriptElement {
	lower := asciiLower(page)
	var found []scriptElement
	for offset := 0; ; {
		start := strings.Index(lower[offset:], "<script")
		if start < 0 {
			return found
		}
		after := offset + start + len("<script")
		if after < len(page) && !isTagBoundary(page[after]) {
			offset = after
			continue
		}
		attributes, textStart, ok := scanAttributes(page, after)
		end := closingScript(lower, textStart)
		if !ok || end < 0 {
			return found
		}
		found = append(found, scriptElement{attributes, page[textStart:end]})
		offset = end
	}
}

func (s scriptElement) inline() bool {
	return s.text != "" && !slices.Contains(s.attributes, "src")
}

func scanAttributes(page string, i int) ([]string, int, bool) {
	var names []string
	for i < len(page) {
		switch c := page[i]; {
		case c == '>':
			return names, i + 1, true
		case isSpace(c) || c == '/':
			i++
		default:
			end := i + 1
			for end < len(page) && !isTagBoundary(page[end]) && page[end] != '=' {
				end++
			}
			names = append(names, asciiLower(page[i:end]))
			i = skipValue(page, end)
		}
	}
	return nil, 0, false
}

func skipValue(page string, i int) int {
	i = skipSpaces(page, i)
	if i >= len(page) || page[i] != '=' {
		return i
	}
	i = skipSpaces(page, i+1)
	if i < len(page) && (page[i] == '"' || page[i] == '\'') {
		if end := strings.IndexByte(page[i+1:], page[i]); end >= 0 {
			return i + 1 + end + 1
		}
		return len(page)
	}
	for i < len(page) && !isSpace(page[i]) && page[i] != '>' {
		i++
	}
	return i
}

func closingScript(lower string, i int) int {
	for offset := i; offset <= len(lower); {
		end := strings.Index(lower[offset:], "</script")
		if end < 0 {
			return -1
		}
		end += offset
		after := end + len("</script")
		if after >= len(lower) || isTagBoundary(lower[after]) {
			return end
		}
		offset = after
	}
	return -1
}

func skipSpaces(page string, i int) int {
	for i < len(page) && isSpace(page[i]) {
		i++
	}
	return i
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r'
}

func isTagBoundary(c byte) bool {
	return isSpace(c) || c == '/' || c == '>'
}
