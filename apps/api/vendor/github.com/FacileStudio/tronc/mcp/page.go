package mcp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"unicode/utf8"
)

// ResultBudget caps the encoded size of one tool result at about 25,000
// tokens, counting three bytes a token for JSON.
const ResultBudget = 75_000

// The response formats a read tool accepts in its response_format argument.
const (
	FormatConcise  = "concise"
	FormatDetailed = "detailed"
)

// ErrCursor is what a model sees when it sends a cursor this server did not
// mint.
var ErrCursor = errors.New("the cursor is not valid. Omit it to start again from the first page")

// errFormat is the refusal for an unknown response_format.
var errFormat = errors.New(`response_format must be "concise" or "detailed"`)

// Detailed reports whether format asks for the detailed form. Empty means
// concise; any other value is an error the model can fix.
func Detailed(format string) (bool, error) {
	switch format {
	case "", FormatConcise:
		return false, nil
	case FormatDetailed:
		return true, nil
	default:
		return false, errFormat
	}
}

// EncodeCursor makes an opaque cursor out of a position, any value that
// marshals to JSON.
func EncodeCursor(position any) string {
	raw, _ := json.Marshal(position)
	return base64.RawURLEncoding.EncodeToString(raw)
}

// DecodeCursor reads a cursor made by EncodeCursor into position. An empty
// cursor leaves position untouched; a malformed one is ErrCursor.
func DecodeCursor(cursor string, position any) error {
	if cursor == "" {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil || json.Unmarshal(raw, position) != nil {
		return ErrCursor
	}
	return nil
}

// FitCount returns the largest n in [0, total] for which size(n), the
// encoded size of a result holding the first n items, stays within
// ResultBudget. size must grow with n; a binary search finds n in
// log(total) encodings.
func FitCount(total int, size func(n int) int) int {
	return max(sort.Search(total+1, func(n int) bool { return size(n) > ResultBudget })-1, 0)
}

// EncodedSize is the JSON size of v, for FitCount.
func EncodedSize(v any) int {
	raw, _ := json.Marshal(v)
	return len(raw)
}

// Clip cuts text to at most limit bytes on a rune boundary and reports
// whether anything was cut.
func Clip(text string, limit int) (string, bool) {
	if len(text) <= limit {
		return text, false
	}
	cut := max(limit, 0)
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return strings.TrimSpace(text[:cut]), true
}
