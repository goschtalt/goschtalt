// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package yaml

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
)

// formatter is a function that attempts to format a value.
// Returns the formatted lines, block style, and success flag.
type formatter func(val string, maxLineLength int) ([]string, string, bool)

// tryPlainScalar attempts to format a value as a plain scalar (no quotes).
func tryPlainScalar(val string, maxLineLength int) ([]string, string, bool) {
	if !strings.Contains(val, "\n") &&
		!hasSpecialChars(val) &&
		!hasSpecialSpaces(val) &&
		!isKeyword(val) &&
		!isNumeric(val) &&
		(maxLineLength <= 0 || len(val) <= maxLineLength) {
		return []string{val}, "", true
	}
	return nil, "", false
}

// trySingleQuotes attempts to format a value using single quotes.
func trySingleQuotes(val string, maxLineLength int) ([]string, string, bool) {
	if !strings.Contains(val, "'") &&
		!strings.Contains(val, "\n") &&
		!strings.Contains(val, "\t") &&
		!hasControlChars(val) &&
		(maxLineLength <= 0 || len(val)+2 <= maxLineLength) { // +2 for the quotes
		return []string{"'" + val + "'"}, "", true
	}
	return nil, "", false
}

// tryWhitespaceOnly attempts to format whitespace-only strings.
func tryWhitespaceOnly(val string, maxLineLength int) ([]string, string, bool) {
	if !hasNormalContent(val) {
		return []string{`"` + strings.ReplaceAll(val, "\n", `\n`) + `"`}, "", true
	}
	return nil, "", false
}

// tryWhitespaceOnlyWithNewlines attempts to format strings with only whitespace and newlines.
func tryWhitespaceOnlyWithNewlines(val string, maxLineLength int) ([]string, string, bool) {
	if !hasNormalContent(val) && strings.Contains(val, "\n") {
		escaped := strings.ReplaceAll(val, "\n", "\\n")
		return []string{`"` + escaped + `"`}, "", true
	}
	return nil, "", false
}

// tryLeadingTrailingSpaces attempts to format strings with leading/trailing spaces but no newlines.
func tryLeadingTrailingSpaces(val string, maxLineLength int) ([]string, string, bool) {
	if hasSpecialSpaces(val) && !strings.Contains(val, "\n") {
		quoted := quoteAndEscapeValue(val)
		return chunkString(quoted, maxLineLength), "", true
	}
	return nil, "", false
}

// tryLiteralBlockScalar attempts to format multi-line strings as literal block scalars.
func tryLiteralBlockScalar(val string, maxLineLength int) ([]string, string, bool) {
	if strings.Contains(val, "\n") && hasNormalContent(val) {
		lines := strings.Split(val, "\n")
		style := "|-"

		// If the original string ends with a newline, use clip style to preserve it
		if strings.HasSuffix(val, "\n") {
			style = "|"
			// Remove the empty last element from split
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
		}

		return lines, style, true
	}
	return nil, "", false
}

// tryFoldedBlockScalar attempts to format long text as a folded block scalar.
func tryFoldedBlockScalar(val string, maxLineLength int) ([]string, string, bool) {
	if maxLineLength <= 0 || len(val) <= maxLineLength || hasSpecialChars(val) {
		return nil, "", false
	}

	words := strings.Fields(val)
	if len(words) <= 1 {
		return nil, "", false
	}

	// Check if we can actually break it into reasonable lines
	canBreak := false
	for _, word := range words {
		if len(word) <= maxLineLength {
			canBreak = true
			break
		}
	}

	if !canBreak {
		return nil, "", false
	}

	lines := []string{}
	currentLine := ""

	for _, word := range words {
		if currentLine == "" {
			currentLine = word
		} else if len(currentLine+" "+word) <= maxLineLength {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines, ">-", true
}

// tryQuotedDefault attempts to format using default quoting and chunking.
func tryQuotedDefault(val string, maxLineLength int) ([]string, string, bool) {
	quotedVal := quoteAndEscapeValue(val)

	if hasSpecialChars(val) || (maxLineLength > 0 && len(quotedVal) > maxLineLength) {
		return chunkString(quotedVal, maxLineLength), "", true
	}

	return []string{quotedVal}, "", true
}

// isKeyword checks if a string is a potential YAML keyword.
func isKeyword(s string) bool {
	switch strings.ToLower(s) {
	case "null",
		"true", "false",
		"~",
		".inf", "-.inf",
		".nan",
		"yes", "no",
		"on", "off":
		return true
	}
	return false
}

// isNumeric checks if a string looks like a number and would be parsed as such by YAML.
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Check for integer (including negative and positive signs)
	for i, r := range s {
		if i == 0 && (r == '-' || r == '+') {
			continue
		}
		if !unicode.IsDigit(r) {
			break
		}
		if i == len(s)-1 {
			return true // All digits (possibly with leading sign)
		}
	}

	// Check for float
	if strings.Contains(s, ".") {
		var f float64
		if _, err := fmt.Sscanf(s, "%f", &f); err == nil {
			return true
		}
	}

	return false
}

// hasSpecialChars checks if a string contains special characters that would
// require it to be quoted in YAML. It allows newlines and tabs, to not be
// qualified as special characters but checks for control characters, certain
// punctuation, and specific sequences that are problematic in YAML.
func hasSpecialChars(s string) bool {
	for _, r := range s {
		if r == '\n' || r == '\t' {
			continue // Allow newlines and tabs
		}
		if unicode.IsControl(r) && r < 128 {
			return true
		}
		if strings.ContainsRune("#&*!|'\"%@`", r) {
			return true
		}
		if strings.Contains(s, ": ") {
			return true
		}
		if strings.Contains(s, string('\\')) {
			return true
		}
	}
	return false
}

// hasControlChars checks if a string contains control characters that require double quotes.
func hasControlChars(s string) bool {
	for _, r := range s {
		if r == '\n' || r == '\t' {
			continue // Allow newlines and tabs
		}
		if unicode.IsControl(r) && r < 128 {
			return true
		}
	}
	return false
}

// hasSpecialSpaces checks if a string has leading or trailing spaces that would
// require it to be quoted in YAML.  Newlines are excluded from this check,
func hasSpecialSpaces(s string) bool {
	var first, last bool

	if len(s) > 0 {
		c0 := rune(s[0])
		clast := rune(s[len(s)-1])
		first = (c0 != '\n') && unicode.IsSpace(c0)
		last = (clast != '\n') && unicode.IsSpace(clast)
	}
	return first || last
}

// hasNormalContent checks if a string contains any characters that are not a
// space or newlines. Returns true if non-space/newline characters are present.
func hasNormalContent(s string) bool {
	for _, r := range s {
		if r == '\n' || unicode.IsSpace(r) {
			continue
		}
		return true
	}
	return false
}

// chunkString splits a string into chunks of a specified length, adding a
// backslash at the end of each chunk except the last one. If n is less than or
// equal to zero, it returns the original string in a slice.  The string passed
// in needs to be a double-quoted string. It tries to split at word boundaries
// when possible to improve readability.
func chunkString(s string, n int) []string {
	if !strings.HasPrefix(s, "\"") {
		s = "\"" + s + "\""
	}
	if n <= 0 {
		return []string{s}
	}

	lines := make([]string, 0, len(s)/n+1)
	remaining := s

	for len(remaining) > 0 {
		// Find the best split point (prefer word boundaries)
		splitPos := findBestSplitPoint(remaining, n)
		if splitPos == len(remaining) {
			lines = append(lines, remaining)
			break
		}

		// Extract the chunk and add backslash
		chunk := remaining[:splitPos] + "\\"
		lines = append(lines, chunk)

		// Move to the next chunk
		remaining = remaining[splitPos:]
	}

	return lines
}

// findBestSplitPoint finds the best position to split a string, preferring
// word boundaries but ensuring we don't exceed the maximum length.
func findBestSplitPoint(s string, maxLen int) int {
	if len(s) <= maxLen {
		return len(s)
	}

	// Reserve space for the backslash
	effectiveMaxLen := maxLen - 1

	// Start from the effective max length and work backwards to find a space
	for i := effectiveMaxLen; i > effectiveMaxLen/2; i-- {
		if i < len(s) && s[i] == ' ' {
			// Found a space, split after it but include the space in current chunk
			return i + 1
		}
	}

	// If no good word boundary found, split at character boundary
	return effectiveMaxLen
}

// quoteAndEscapeValue takes a string value and returns it properly quoted and
// escaped for YAML output. It handles control characters, special YAML
// characters, and other characters that need escaping in double-quoted strings.
func quoteAndEscapeValue(val string) string {
	rv, _ := json.Marshal(val)
	if string(rv) == `"`+val+`"` {
		return val
	}
	return string(rv)
}
