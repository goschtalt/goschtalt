// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package encoding

import (
	"sort"
	"testing"
)

type testEncodeable struct {
	key   *string
	value *string
}

func (t *testEncodeable) Indent() int           { return 0 }
func (t *testEncodeable) Headers() []string     { return nil }
func (t *testEncodeable) Inline() []string      { return nil }
func (t *testEncodeable) Key() *string          { return t.key }
func (t *testEncodeable) Value() *string        { return t.value }
func (t *testEncodeable) Children() Encodeables { return nil }

func pstr(s string) *string { return &s }

func TestEncodeablesSort(t *testing.T) {
	tests := []struct {
		name     string
		input    Encodeables
		expected []string
	}{
		{
			name: "basic string keys",
			input: Encodeables{
				&testEncodeable{key: pstr("b"), value: pstr("1")},
				&testEncodeable{key: pstr("a"), value: pstr("0")},
				&testEncodeable{key: pstr("c"), value: pstr("2")},
			},
			expected: []string{"a:0", "b:1", "c:2"},
		},
		{
			name: "nil keys sorted first",
			input: Encodeables{
				&testEncodeable{key: nil, value: pstr("0")},
				&testEncodeable{key: pstr("a"), value: pstr("2")},
				&testEncodeable{key: nil, value: pstr("1")},
				&testEncodeable{key: pstr("b"), value: pstr("3")},
			},
			expected: []string{"<nil>:0", "<nil>:1", "a:2", "b:3"},
		},
		{
			name: "equal keys stable sort",
			input: Encodeables{
				&testEncodeable{key: pstr("a"), value: pstr("0")},
				&testEncodeable{key: pstr("a"), value: pstr("1")},
				&testEncodeable{key: pstr("a"), value: pstr("2")},
				&testEncodeable{key: pstr("a"), value: pstr("3")},
			},
			expected: []string{"a:0", "a:1", "a:2", "a:3"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sort.Sort(tc.input)
			for i, enc := range tc.input {
				got := "<nil>"
				if enc.Key() != nil {
					got = *enc.Key()
				}
				if enc.Value() != nil {
					got += ":" + *enc.Value()
				}
				if got != tc.expected[i] {
					t.Errorf("at index %d: got %q, want %q", i, got, tc.expected[i])
				}
			}
		})
	}
}
