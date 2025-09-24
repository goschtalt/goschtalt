// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package yaml

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/goschtalt/goschtalt/internal/encoding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ymlTest struct {
	desc     string
	key      *string
	val      *string
	indent   int
	headers  []string
	inline   []string
	children []ymlTest
	r        Renderer
	expect   string
}

func pstr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var _ encoding.Encodeable = &ymlTest{}

func (t *ymlTest) Key() *string {
	return t.key
}

func (t *ymlTest) Value() *string {
	return t.val
}

func (t *ymlTest) Indent() int {
	return t.indent
}
func (t *ymlTest) Headers() []string {
	return t.headers
}
func (t *ymlTest) Inline() []string {
	return t.inline
}
func (t *ymlTest) Children() encoding.Encodeables {
	if len(t.children) == 0 {
		return nil
	}

	children := make([]encoding.Encodeable, len(t.children))
	for i, child := range t.children {
		children[i] = &child
	}
	return children
}

var basicYMLTests = []ymlTest{
	{
		desc:   "empty, no comments",
		key:    pstr("a"),
		expect: "a: \n",
	}, {
		desc:   "simple, no comments",
		key:    pstr("a"),
		val:    pstr("b"),
		expect: "a: b\n",
	}, {
		desc:   "simple, single comment",
		key:    pstr("a"),
		val:    pstr("b"),
		inline: []string{"comment"},
		expect: "a: b # comment\n",
	}, {
		desc:   "simple, single comment, spaced over",
		key:    pstr("a"),
		val:    pstr("b"),
		inline: []string{"comment"},
		r: Renderer{
			MaxLineLength:         20,
			TrailingCommentColumn: 20,
			SpacesPerIndent:       2,
		},
		expect: "a: b                # comment\n",
	}, {
		desc:    "simple, single comment with header comment",
		key:     pstr("a"),
		val:     pstr("b"),
		headers: []string{"header comment"},
		inline:  []string{"comment"},
		expect: "" +
			"# header comment\n" +
			"a: b # comment\n",
	}, {
		desc:    "simple, multi-line comment with header comment",
		key:     pstr("a"),
		val:     pstr("b"),
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"# header comment\n" +
			"a:\n" +
			"  # comment 1\n" +
			"  # comment 2\n" +
			"  b\n",
	}, {
		desc:    "simple, indented multi-line comment with header comment",
		key:     pstr("a"),
		val:     pstr("b"),
		indent:  1,
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"  # header comment\n" +
			"  a:\n" +
			"    # comment 1\n" +
			"    # comment 2\n" +
			"    b\n",
	}, {
		desc:    "simple, indented multi-line comment with header comment",
		val:     pstr("b"),
		indent:  1,
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"  # header comment\n" +
			"  -\n" +
			"    # comment 1\n" +
			"    # comment 2\n" +
			"    b\n",
	}, {
		desc:    "simple, indented comment with leading empty line trimmed",
		val:     pstr("b"),
		indent:  1,
		headers: []string{"  ", "header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"  # header comment\n" +
			"  -\n" +
			"    # comment 1\n" +
			"    # comment 2\n" +
			"    b\n",
	},
}

var quotedYMLTests = []ymlTest{

	// & based value

	{
		desc:   "complex val, no comments",
		key:    pstr("a"),
		val:    pstr("&b"),
		expect: "a: '&b'\n",
	}, {
		desc:   "complex val, single comment",
		key:    pstr("a"),
		val:    pstr("&b"),
		inline: []string{"comment"},
		expect: "a: '&b' # comment\n",
	}, {
		desc:    "complex val, single comment with header comment",
		key:     pstr("a"),
		val:     pstr("&b"),
		headers: []string{"header comment"},
		inline:  []string{"comment"},
		expect: "" +
			"# header comment\n" +
			"a: '&b' # comment\n",
	}, {
		desc:    "complex val, multiple comment with header comment",
		key:     pstr("a"),
		val:     pstr("&b"),
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"# header comment\n" +
			"a:\n" +
			"  # comment 1\n" +
			"  # comment 2\n" +
			"  '&b'\n",
	},

	// x01 based value

	{
		desc:   "control char val, no comments",
		key:    pstr("a"),
		val:    pstr("\x01"),
		expect: "a: \"\\u0001\"\n",
	}, {
		desc:   "control char val, single comment",
		key:    pstr("a"),
		val:    pstr("\x01"),
		inline: []string{"comment"},
		expect: "a: \"\\u0001\" # comment\n",
	}, {
		desc:    "control char val, single comment with header comment",
		key:     pstr("a"),
		val:     pstr("\x01"),
		headers: []string{"header comment"},
		inline:  []string{"comment"},
		expect: "" +
			"# header comment\n" +
			"a: \"\\u0001\" # comment\n",
	}, {
		desc:    "control char val, multiple comment with header comment",
		key:     pstr("a"),
		val:     pstr("\x01"),
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"# header comment\n" +
			"a:\n" +
			"  # comment 1\n" +
			"  # comment 2\n" +
			"  \"\\u0001\"\n",
	},

	// quoted string value

	{
		desc:   "quoted string val, no comments",
		key:    pstr("a"),
		val:    pstr("fred said \"hello, world\""),
		expect: "a: 'fred said \"hello, world\"'\n",
	}, {
		desc:   "quote string val, single comment",
		key:    pstr("a"),
		val:    pstr("fred said \"hello, world\""),
		inline: []string{"comment"},
		expect: "a: 'fred said \"hello, world\"' # comment\n",
	}, {
		desc:    "quote string val, single comment with header comment",
		key:     pstr("a"),
		val:     pstr("fred said \"hello, world\""),
		headers: []string{"header comment"},
		inline:  []string{"comment"},
		expect: "" +
			"# header comment\n" +
			"a: 'fred said \"hello, world\"' # comment\n",
	}, {
		desc:    "quote string val, multiple comment with header comment",
		key:     pstr("a"),
		val:     pstr("fred said \"hello, world\""),
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"# header comment\n" +
			"a:\n" +
			"  # comment 1\n" +
			"  # comment 2\n" +
			"  'fred said \"hello, world\"'\n",
	},

	// " based value

	{
		desc:   "quoted string val, no comments",
		key:    pstr("a"),
		val:    pstr("\""),
		expect: "a: '\"'\n",
	}, {
		desc:   "quote val, single comment",
		key:    pstr("a"),
		val:    pstr("\""),
		inline: []string{"comment"},
		expect: "a: '\"' # comment\n",
	}, {
		desc:    "quote val, single comment with header comment",
		key:     pstr("a"),
		val:     pstr("\""),
		headers: []string{"header comment"},
		inline:  []string{"comment"},
		expect: "" +
			"# header comment\n" +
			"a: '\"' # comment\n",
	}, {
		desc:    "quote val, multiple comment with header comment",
		key:     pstr("a"),
		val:     pstr("\""),
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"# header comment\n" +
			"a:\n" +
			"  # comment 1\n" +
			"  # comment 2\n" +
			"  '\"'\n",
	},

	{
		desc:    "special spaces, single comment with header comment",
		key:     pstr("a"),
		val:     pstr("  b  "),
		headers: []string{"header comment"},
		inline:  []string{"comment"},
		expect: "" +
			"# header comment\n" +
			"a: '  b  ' # comment\n",
	}, {
		desc:    "special spaces,  multiple comment with header comment",
		key:     pstr("a"),
		val:     pstr("  b  "),
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"# header comment\n" +
			"a:\n" +
			"  # comment 1\n" +
			"  # comment 2\n" +
			"  '  b  '\n",
	}, {
		desc:    "just a newline",
		key:     pstr("a"),
		val:     pstr("\n"),
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"# header comment\n" +
			"a:\n" +
			"  # comment 1\n" +
			"  # comment 2\n" +
			"  \"\\n\"\n",
	}, {
		desc:    "just a newline with spaces",
		key:     pstr("a"),
		val:     pstr("   \n   "),
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"# header comment\n" +
			"a:\n" +
			"  # comment 1\n" +
			"  # comment 2\n" +
			"  \"   \\n   \"\n",
	},
}

var keywordYMLTests = []ymlTest{
	{
		desc:   "keyword val, no comments",
		key:    pstr("a"),
		val:    pstr("null"),
		expect: "a: 'null'\n",
	},
}

var multiLineYMLTests = []ymlTest{
	{
		desc: "multi-line, no comments",
		key:  pstr("a"),
		val:  pstr("a\nb\nc"),
		expect: "" +
			"a: |-\n" +
			"  a\n" +
			"  b\n" +
			"  c\n",
	}, {
		desc:    "multi-line, single comments",
		key:     pstr("a"),
		val:     pstr("a\nb\nc"),
		headers: []string{"header comment"},
		inline:  []string{"comment"},
		expect: "" +
			"# header comment\n" +
			"a: # comment\n" +
			"  |-\n" +
			"  a\n" +
			"  b\n" +
			"  c\n",
	}, {
		desc:    "multi-line, multiple comments",
		key:     pstr("a"),
		val:     pstr("a\nb\nc"),
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"# header comment\n" +
			"a:\n" +
			"  # comment 1\n" +
			"  # comment 2\n" +
			"  |-\n" +
			"  a\n" +
			"  b\n" +
			"  c\n",
	}, {
		desc:    "multi-line, trailing newline, multiple comments",
		key:     pstr("a"),
		val:     pstr("a\nb\nc\n"),
		headers: []string{"header comment"},
		inline:  []string{"comment 1", "comment 2"},
		expect: "" +
			"# header comment\n" +
			"a:\n" +
			"  # comment 1\n" +
			"  # comment 2\n" +
			"  |\n" +
			"  a\n" +
			"  b\n" +
			"  c",
	},
}

var maxLineYMLTests = []ymlTest{
	{
		desc:   "max line length, no comments",
		indent: 1,
		key:    pstr("a"),
		val: pstr("" +
			"This is a really long string that should " +
			"definitely exceed the limit of 40 characters. " +
			"Indeed, it has enough text to go beyond " +
			"that boundary."),
		expect: "" +
			"  a: >-\n" +
			"    This is a really long string that\n" +
			"    should definitely exceed the limit\n" +
			"    of 40 characters. Indeed, it has\n" +
			"    enough text to go beyond that\n" +
			"    boundary.\n",
		r: Renderer{
			MaxLineLength:         40,
			TrailingCommentColumn: 0,
			SpacesPerIndent:       2,
		},
	}, {
		desc:   "max line length when it needs to be extended",
		indent: 20,
		key:    pstr("a"),
		val: pstr("" +
			"This is a really long string that should " +
			"definitely exceed the limit of 40 characters. " +
			"Indeed, it has enough text to go beyond " +
			"that boundary."),
		expect: "" +
			"                                        a: >-\n" +
			"                                          This is a really long string that\n" +
			"                                          should definitely exceed the limit of\n" +
			"                                          40 characters. Indeed, it has enough\n" +
			"                                          text to go beyond that boundary.\n",
		r: Renderer{
			MaxLineLength:         40,
			TrailingCommentColumn: 0,
			SpacesPerIndent:       2,
		},
	}, {
		desc:   "max line vs a long word",
		indent: 1,
		key:    pstr("a"),
		val:    pstr("012345678901234567890123456789"),
		expect: "" +
			"  a: \n" +
			"    \"0123\\\n" +
			"    45678\\\n" +
			"    90123\\\n" +
			"    45678\\\n" +
			"    90123\\\n" +
			"    45678\\\n" +
			"    9\"\n",
		r: Renderer{
			MaxLineLength:         10,
			TrailingCommentColumn: 0,
			SpacesPerIndent:       2,
		},
	},
}

var childrenYMLTests = []ymlTest{
	{
		desc: "child node, no comments",
		key:  pstr("a"),
		val:  pstr("b"),
		children: []ymlTest{
			{
				indent: 1,
				key:    pstr("c"),
				val:    pstr("d"),
			},
			{
				indent: 1,
				key:    pstr("0"),
				val:    pstr("9"),
			},
		},
		expect: "" +
			"a: b\n" +
			"  0: '9'\n" +
			"  c: d\n",
	},
}

func TestRendererFormat(t *testing.T) {
	tests := basicYMLTests
	tests = append(tests, quotedYMLTests...)
	tests = append(tests, keywordYMLTests...)
	tests = append(tests, multiLineYMLTests...)
	tests = append(tests, maxLineYMLTests...)
	tests = append(tests, childrenYMLTests...)

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			var buf strings.Builder
			err := tc.r.Encode(&buf, &tc)
			assert.NoError(t, err, "failed to encode item")
			expect := "---\n" + tc.expect + "\n"
			assert.Equal(t, expect, buf.String(), "formatted output does not match expected")
		})
	}
}

func TestYAMLFormatRoundTrip(t *testing.T) {
	_, err := exec.LookPath("yq")
	if err != nil {
		t.Skip("yq command not found, skipping YAML format round-trip test")
	}

	tests := basicYMLTests
	tests = append(tests, quotedYMLTests...)
	tests = append(tests, keywordYMLTests...)
	tests = append(tests, multiLineYMLTests...)
	tests = append(tests, maxLineYMLTests...)

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			var buf strings.Builder
			err := tc.r.Encode(&buf, &tc)
			require.NoError(t, err)
			k := tc.Key()
			if k == nil {
				t.Skip("Skipping round-trip test for item without key")
				return
			}

			key := "." + *k

			var stderr bytes.Buffer
			cmd := exec.Command("yq", "-oj", key, "-")
			cmd.Stdin = strings.NewReader(buf.String())
			cmd.Stderr = &stderr
			out, err := cmd.Output()

			if err != nil {
				fmt.Println("YAML input:")
				fmt.Println(dump(buf.String()))
				fmt.Println("stdout: ", string(out))
				fmt.Println("stderr: ", stderr.String())
				require.NoError(t, err, "yq command failed")
			}

			var got string
			require.NoError(t, json.Unmarshal(out, &got))
			want := "null"
			if tc.Value() != nil {
				want = *tc.Value()
			}

			if want == "null" {
				switch got {
				case "", "null":
				default:
					assert.Fail(t, fmt.Sprintf("expected null or empty string, got: %q", got))
				}
			} else {
				if !assert.Equal(t, want, got) {
					fmt.Println("YAML format round-trip test failed:", tc.desc)
					//pp.Println("Original value:", want)
					//pp.Println("Formatted value:", buf.String())
					//pp.Println("Original value from yq:", string(out))
				}
			}
		})
	}
}
