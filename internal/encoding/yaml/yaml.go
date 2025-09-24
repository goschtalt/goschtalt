// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package yaml

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/goschtalt/goschtalt/internal/encoding"
)

/*
Output in one of a few formats:

If the value is short enough, it is rendered as a single line.
[short_value] is the short value if possible, otherwise it is the value.
[value] is the value if it is too long to fit on a single line.

The general format is:

---
# header comment
key: [short_value] 	# single line comment
  # multi-line comment
  # multi-line comment
  [block characters if needed]
  [value]

Examples:

---
# header comment
key: value         	# single line comment

---
# header comment
key:
  # multi-line comment
  # multi-line comment
  value

---
# header comment
key: "value"       	# single line comment

---
# header comment
key:
  # multi-line comment
  # multi-line comment
  "value"

---
# header comment
key:				# single line comment
  "valuesplit\
  overmultiple\
  lines"

---
# header comment
key:
  # multi-line comment
  # multi-line comment
  "valuesplit\
  overmultiple\
  lines"

These next two forms are not used because they are hard to correctly
determine since multiple spaces cause confusion/challenges.  These are
generally not used for configuration from what I can tell.  If that
changes, we can add them in the future.

---
# header comment
key:				# single line comment
  <
  value split
  over multiple
  lines

---
# header comment
key:
  # multi-line comment
  # multi-line comment
  <
  value split
  over multiple
  lines

*/

// Renderer defines rendering options for YAML output.
type Renderer struct {
	// MaxLineLength is the maximum line length for wrapping.
	// If <=0, no wrapping is done.
	MaxLineLength int

	// TrailingCommentColumn is the column for comments to start at if positive,
	// or number of spaces to indent comments if negative.
	// If zero, comments are not rendered.
	TrailingCommentColumn int

	// SpacesPerIndent is the number of spaces to indent each line.
	// If zero the default is 2.
	SpacesPerIndent int
}

// Encode encodes an Encodeable item to YAML format.
func (r *Renderer) Encode(w io.Writer, item encoding.Encodeable) error {
	var buf strings.Builder
	fmt.Fprintln(&buf, "---")
	r.encode(&buf, item)

	// Ensure document ends with newline for proper YAML parsing
	buf.WriteRune('\n')

	// Don't add extra at the start of the document.  This can happen if there
	// are comments at the start of the document.  The simplest way to handle
	// this is to just trim it out and ensure the document starts with ---
	s := buf.String()
	if strings.HasPrefix(s, "---\n\n") {
		s = strings.TrimPrefix(s, "---\n")
		buf.Reset()
		buf.WriteString("---")
		buf.WriteString(s)
		s = buf.String()
	}

	_, err := w.Write([]byte(s))
	return err
}

// headers renders the header comments for an Encodeable item.
func (r *Renderer) headers(buf *strings.Builder, item encoding.Encodeable) {
	// Render header comments
	var firstLineRendered bool
	headers := item.Headers()

	prependNewline := true
	for _, hc := range headers {
		hc = strings.TrimSuffix(hc, "\n")

		if !firstLineRendered && strings.TrimSpace(hc) == "" {
			continue
		}

		if prependNewline {
			prependNewline = false
			buf.WriteString("\n")
		}
		fmt.Fprintf(buf, "%s# %s\n", r.indent(item.Indent()), hc)
		firstLineRendered = true
	}
}

// encode is the recursively called function that encodes an Encodeable item
// and its children to YAML format.
func (r *Renderer) encode(buf *strings.Builder, item encoding.Encodeable) {
	r.headers(buf, item)
	r.node(buf, item)

	children := item.Children()
	if children != nil {
		sort.Sort(children)
		for _, child := range children {
			r.encode(buf, child)
		}
	}
}

// node renders a single Encodeable item to YAML format.
func (r *Renderer) node(buf *strings.Builder, item encoding.Encodeable) {
	var tmp strings.Builder

	inline := item.Inline()
	line, v, block := r.prepareLine(item)

	if item.Indent() >= 0 {
		r.writeMainLine(&tmp, line, inline, block)
	}

	r.writeAdditionalContent(&tmp, item, inline, v, block)

	s := tmp.String()
	if strings.TrimSpace(s) != "" {
		buf.WriteString(s)
	}
}

// prepareLine prepares the line for rendering, including the key, value,
// and any necessary formatting for comments or blocks.
func (r *Renderer) prepareLine(item encoding.Encodeable) (string, []string, string) {
	// Render the sequence/key part of the line
	line := fmt.Sprintf("%s-", r.indent(item.Indent()))
	if item.Key() != nil {
		line = fmt.Sprintf("%s%s:", r.indent(item.Indent()), *item.Key())
	}

	var v []string
	var block string
	if item.Value() != nil || item.Children() == nil {
		v, block = formatValue(item.Value(), r.maxLineLength(len(line)-1))
	}

	if len(item.Inline()) <= 1 && len(v) == 1 {
		line += " " + v[0]
	} else if len(v) > 1 && block == "" {
		// Add trailing space when value will be on subsequent lines without block scalar
		line += " "
	}

	return line, v, block
}

// writeMainLine writes the main line for an Encodeable item to YAML format.
func (r *Renderer) writeMainLine(buf *strings.Builder, line string, inline []string, block string) {
	buf.WriteString(line)

	if len(inline) == 1 {
		spaces := max(r.TrailingCommentColumn-len(line), 1)
		fmt.Fprintf(buf, "%s# %s", strings.Repeat(" ", spaces), inline[0])
	}

	if len(inline) == 0 && block != "" {
		fmt.Fprintf(buf, " %s", block)
	}

	buf.WriteString("\n")
}

// writeAdditionalContent writes any additional content for an Encodeable item,
// including multiline comments and values.
func (r *Renderer) writeAdditionalContent(buf *strings.Builder, item encoding.Encodeable, inline, v []string, block string) {
	left := r.indent(item.Indent() + 1)

	// Write multiline comments
	if len(inline) > 1 {
		for _, c := range inline {
			fmt.Fprintf(buf, "%s# %s\n", left, c)
		}
	}

	r.writeMultilineValue(buf, left, item, inline, v, block)
}

// writeMultilineValue writes the value lines for an Encodeable item when they need
// to be rendered on multiple lines (either due to multiple inline comments or
// multiline values). It handles block style indicators and special newline formatting.
func (r *Renderer) writeMultilineValue(buf *strings.Builder, left string, item encoding.Encodeable, inline []string, v []string, block string) {
	if len(inline) > 1 || len(v) > 1 {
		// Write block indicator if not already written on main line
		if len(inline) > 0 && block != "" {
			fmt.Fprintf(buf, "%s%s\n", left, block)
		}

		// Recalculate value formatting for the new indentation
		v, _ = formatValue(item.Value(), r.maxLineLength(len(left)))

		for i, line := range v {
			// For clip style (|), don't add newline to the last line (it's implied)
			isLastLine := i == len(v)-1
			if block != "|" || !isLastLine {
				line += "\n"
			}
			fmt.Fprintf(buf, "%s%s", left, line)
		}
	}
}

// maxLineLength calculates the maximum line length for wrapping based on the
// MaxLineLength and the prefix length (e.g., key and indent).  If MaxLineLength
// is less than or equal to zero, no wrapping is done and the function returns 0.
// If the prefix takes up more than the MaxLineLength, it will be adjusted to
// ensure the maximum line length is generally sane.  This is done by multiplying
// the MaxLineLength by until it is greater than the prefix length.
func (r *Renderer) maxLineLength(prefixLen int) int {
	if r.MaxLineLength <= 0 {
		return math.MaxInt // No wrapping
	}

	maxLen := r.MaxLineLength - prefixLen
	for i := 2; maxLen < 1; i++ {
		maxLen = r.MaxLineLength*i - prefixLen
	}

	return maxLen
}

// indent returns a string of spaces for indentation based on the
// SpacesPerIndent setting. If SpacesPerIndent is zero, it defaults to 2 spaces.
// If the indent level is less than or equal to zero, it returns an empty string.
func (r *Renderer) indent(i int) string {
	if i <= 0 {
		return ""
	}

	spaces := r.SpacesPerIndent
	if spaces <= 0 {
		spaces = 2 // Default to 2 spaces if not set
	}
	return strings.Repeat(" ", i*spaces)
}

// formatValue formats a value for YAML output, handling special cases like
// keywords, newlines, spaces, and special characters. It returns a slice of
// strings representing the formatted value, and a string indicating the style
// of the value (e.g., block style, quoted style, etc.).
func formatValue(v *string, maxLineLength int) ([]string, string) {
	if v == nil {
		return []string{""}, ""
	}

	val := *v

	// Array of formatters to try in order
	formatters := []formatter{
		tryPlainScalar,
		trySingleQuotes,
		tryWhitespaceOnly,
		tryWhitespaceOnlyWithNewlines,
		tryLeadingTrailingSpaces,
		tryLiteralBlockScalar,
		tryFoldedBlockScalar,
		tryQuotedDefault,
	}

	// Try each formatter until one succeeds
	for _, fmt := range formatters {
		if lines, style, ok := fmt(val, maxLineLength); ok {
			return lines, style
		}
	}

	// This should never happen since tryQuotedDefault always succeeds
	return []string{val}, ""
}
