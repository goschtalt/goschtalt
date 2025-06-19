// SPDX-FileCopyrightText: 2022-2023 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package meta

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decode(s string) Object {
	var data any
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		panic(err)
	}
	return ObjectFromRaw(data)
}

func TestObjectFromRawWithOrigin(t *testing.T) {
	origin := Origin{
		File: "file",
		Line: 12,
		Col:  36,
	}
	tests := []struct {
		description string
		thing       any
		where       []Origin
		at          []string
		expected    Object
	}{
		{
			description: "Output an empty origin.",
			thing: map[string]any{
				"one": "fish",
			},
			where: []Origin{origin},
			at:    []string{"a", "b"},
			expected: Object{
				Origins: []Origin{origin},
				Map: map[string]Object{
					"a": {
						Origins: []Origin{origin},
						Map: map[string]Object{
							"b": {
								Origins: []Origin{origin},
								Map: map[string]Object{
									"one": {
										Origins: []Origin{origin},
										Value:   "fish",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			description: "Output an empty origin.",
			thing: map[string]any{
				"one": nil,
			},
			where: []Origin{origin},
			at:    []string{"a", "b"},
			expected: Object{
				Origins: []Origin{origin},
				Map: map[string]Object{
					"a": {
						Origins: []Origin{origin},
						Map: map[string]Object{
							"b": {
								Origins: []Origin{origin},
								Map: map[string]Object{
									"one": {
										Origins: []Origin{origin},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			got := ObjectFromRawWithOrigin(tc.thing, tc.where, tc.at...)

			assert.Equal(tc.expected, got)
		})
	}
}

func TestOrigin_String(t *testing.T) {
	tests := []struct {
		description string
		origin      Origin
		expected    string
	}{
		{
			description: "Output an empty origin.",
			expected:    "unknown",
		}, {
			description: "Output a partially filled out origin.",
			origin: Origin{
				File: "magic.file",
				Col:  88,
			},
			expected: "magic.file:0[88]",
		}, {
			description: "Output a partially filled out origin, no col.",
			origin: Origin{
				File: "magic.file",
				Line: 88,
			},
			expected: "magic.file:88",
		}, {
			description: "Output an filled out origin.",
			origin: Origin{
				File: "magic.file",
				Line: 42,
				Col:  88,
			},
			expected: "magic.file:42[88]",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			got := tc.origin.String()

			assert.Equal(tc.expected, got)
		})
	}
}

func TestFetch(t *testing.T) {
	unknownErr := fmt.Errorf("unknown error")
	tests := []struct {
		description string
		in          string
		asks        []string
		expected    Object
		expectedErr error
	}{
		{
			description: "Simple map lookup.",
			in:          `{"foo":"something"}`,
			asks:        []string{"foo"},
			expected: Object{
				Origins: []Origin{},
				Value:   "something",
			},
		}, {
			description: "Simple map then array lookup.",
			in:          `{"foo":["something", "else"]}`,
			asks:        []string{"foo", "1"},
			expected: Object{
				Origins: []Origin{},
				Value:   "else",
			},
		}, {
			description: "A bit more nested.",
			in:          `{"foo":["something", {"else": "entirely"}]}`,
			asks:        []string{"foo", "1", "else"},
			expected: Object{
				Origins: []Origin{},
				Value:   "entirely",
			},
		}, {
			description: "Not found.",
			in:          `{"foo":["something", {"else": "entirely"}]}`,
			asks:        []string{"foo", "1", "oops"},
			expectedErr: ErrNotFound,
		}, {
			description: "Invalid array value.",
			in:          `{"foo":["something", {"else": "entirely"}]}`,
			asks:        []string{"foo", "0-1.2", "oops"},
			expectedErr: unknownErr,
		}, {
			description: "Array index out of bounds.",
			in:          `{"foo":["something", {"else": "entirely"}]}`,
			asks:        []string{"foo", "10", "oops"},
			expectedErr: unknownErr,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			in := decode(tc.in)
			got, err := in.Fetch(tc.asks, ".")

			if tc.expectedErr == nil {
				assert.NoError(err)
				assert.Equal(tc.expected, got)
				return
			}
			assert.Error(err)
			if !errors.Is(unknownErr, tc.expectedErr) {
				assert.ErrorIs(err, tc.expectedErr)
			}
		})
	}
}

func TestToRaw(t *testing.T) {
	tests := []struct {
		description string
		in          Object
		expected    any
	}{
		{
			description: "Output an empty tree.",
			in: Object{
				Origins: []Origin{},
			},
			expected: nil,
		}, {
			description: "Output an small tree.",
			in: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {},
				},
			},
			expected: map[string]any{
				"foo": nil,
			},
		}, {
			description: "Output an larger tree.",
			in: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						Map: map[string]Object{
							"bar": {
								Origins: []Origin{},
								Value:   int(123),
							},
							"car": {
								Origins: []Origin{},
								Array: []Object{
									{
										Origins: []Origin{},
										Map: map[string]Object{
											"sam": {
												Origins: []Origin{},
												Value:   "cart",
											},
										},
									}, {
										Origins: []Origin{},
										Value:   "golf",
									},
								},
							},
						},
					},
				},
			},
			expected: map[string]any{
				"foo": map[string]any{
					"bar": int(123),
					"car": []any{
						map[string]any{
							"sam": "cart",
						},
						"golf",
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			got := tc.in.ToRaw()

			assert.Equal(tc.expected, got)
		})
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		description string
		key         string
		val         string
		start       Object
		origin      *Origin
		expected    Object
		expectedErr error
	}{
		{
			description: "Add to an empty tree.",
			key:         "Foo.Bar",
			val:         "abc",
			expected: Object{
				Map: map[string]Object{
					"Foo": {
						Origins: []Origin{{}},
						Map: map[string]Object{
							"Bar": {
								Origins: []Origin{{}},
								Value:   "abc",
							},
						},
					},
				},
			},
		}, {
			description: "Add to an empty tree, but add an origin.",
			key:         "Foo.Bar",
			val:         "abc",
			origin:      &Origin{File: "file"},
			expected: Object{
				Map: map[string]Object{
					"Foo": {
						Origins: []Origin{{File: "file"}},
						Map: map[string]Object{
							"Bar": {
								Origins: []Origin{{File: "file"}},
								Value:   "abc",
							},
						},
					},
				},
			},
		}, {
			description: "Add to tree with an array.",
			key:         "Foo.Bar.1",
			val:         "xyz",
			start: Object{
				Map: map[string]Object{
					"Foo": {
						Origins: []Origin{{}},
						Map: map[string]Object{
							"Bar": {
								Origins: []Origin{{}},
								Array: []Object{
									{
										Origins: []Origin{{}},
										Value:   "abc",
									},
								},
							},
						},
					},
				},
			},
			expected: Object{
				Map: map[string]Object{
					"Foo": {
						Origins: []Origin{{}},
						Map: map[string]Object{
							"Bar": {
								Origins: []Origin{{}},
								Array: []Object{
									{
										Origins: []Origin{{}},
										Value:   "abc",
									}, {
										Origins: []Origin{{}},
										Value:   "xyz",
									},
								},
							},
						},
					},
				},
			},
		}, {
			description: "Change a value in an array.",
			key:         "Foo.Bar.0",
			val:         "---",
			start: Object{
				Map: map[string]Object{
					"Foo": {
						Origins: []Origin{{}},
						Map: map[string]Object{
							"Bar": {
								Origins: []Origin{{}},
								Array: []Object{
									{
										Origins: []Origin{{}},
										Value:   "abc",
									}, {
										Origins: []Origin{{}},
										Value:   "xyz",
									},
								},
							},
						},
					},
				},
			},
			expected: Object{
				Map: map[string]Object{
					"Foo": {
						Origins: []Origin{{}},
						Map: map[string]Object{
							"Bar": {
								Origins: []Origin{{}},
								Array: []Object{
									{
										Origins: []Origin{{}},
										Value:   "---",
									}, {
										Origins: []Origin{{}},
										Value:   "xyz",
									},
								},
							},
						},
					},
				},
			},
		}, {
			description: "Fail with an array index that is negative.",
			key:         "Foo.Bar.-1",
			val:         "---",
			expectedErr: ErrArrayOutOfBounds,
			start: Object{
				Map: map[string]Object{
					"Foo": {
						Origins: []Origin{{}},
						Map: map[string]Object{
							"Bar": {
								Origins: []Origin{{}},
								Array: []Object{
									{
										Origins: []Origin{{}},
										Value:   "abc",
									}, {
										Origins: []Origin{{}},
										Value:   "xyz",
									},
								},
							},
						},
					},
				},
			},
		}, {
			description: "Fail with an array index that is too large.",
			key:         "Foo.Bar.10",
			val:         "---",
			expectedErr: ErrArrayOutOfBounds,
			start: Object{
				Map: map[string]Object{
					"Foo": {
						Origins: []Origin{{}},
						Map: map[string]Object{
							"Bar": {
								Origins: []Origin{{}},
								Array: []Object{
									{
										Origins: []Origin{{}},
										Value:   "abc",
									}, {
										Origins: []Origin{{}},
										Value:   "xyz",
									},
								},
							},
						},
					},
				},
			},
		}, {
			description: "Fail with an array index that isn't an int.",
			key:         "Foo.0.invalid",
			val:         "---",
			expectedErr: ErrInvalidIndex,
			start: Object{
				Map: map[string]Object{
					"Foo": {
						Origins: []Origin{{}},
						Array: []Object{
							{
								Origins: []Origin{{}},
								Array: []Object{
									{
										Origins: []Origin{{}},
										Value:   "abc",
									}, {
										Origins: []Origin{{}},
										Value:   "xyz",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			var got Object
			var err error
			if tc.origin == nil {
				got, err = tc.start.Add(".", tc.key, tc.val)
			} else {
				got, err = tc.start.Add(".", tc.key, tc.val, *tc.origin)
			}

			if tc.expectedErr == nil {
				assert.NoError(err)
				assert.Equal(tc.expected, got)
				return
			}

			assert.ErrorIs(err, tc.expectedErr)
		})
	}
}

func TestConvertMapsToArrays(t *testing.T) {
	empty := []Origin{{}}
	tests := []struct {
		description string
		inputs      []string
		origin      Origin
		expected    Object
	}{
		{
			description: "An empty object.",
			expected: Object{
				Origins: []Origin{{}},
			},
		}, {
			description: "An normal example.",
			inputs: []string{
				"foo.bar.cat.0=zero",
				"foo.bar.cat.2=two",
				"foo.bar.cat.1=one",
				"foo.bar.dog=Fred",
				"foo.bar.fish.0.0=Wanda",
				"foo.bar.fish.0.1=Ponyo",
			},
			expected: Object{
				Origins: empty,
				Map: map[string]Object{
					"foo": {
						Origins: empty,
						Map: map[string]Object{
							"bar": {
								Origins: empty,
								Map: map[string]Object{
									"cat": {
										Origins: empty,
										Array: []Object{
											{Origins: empty, Value: "zero"},
											{Origins: empty, Value: "one"},
											{Origins: empty, Value: "two"},
										},
									},
									"fish": {
										Origins: empty,
										Array: []Object{
											{
												Origins: empty,
												Array: []Object{
													{Origins: empty, Value: "Wanda"},
													{Origins: empty, Value: "Ponyo"},
												},
											},
										},
									},
									"dog": {Origins: empty, Value: "Fred"},
								},
							},
						},
					},
				},
			},
		}, {
			description: "An array with a gap.",
			inputs: []string{
				"foo.bar.cat.0=zero",
				"foo.bar.cat.2=two",
			},
			expected: Object{
				Origins: empty,
				Map: map[string]Object{
					"foo": {
						Origins: empty,
						Map: map[string]Object{
							"bar": {
								Origins: empty,
								Map: map[string]Object{
									"cat": {
										Origins: empty,
										Map: map[string]Object{
											"0": {Origins: empty, Value: "zero"},
											"2": {Origins: empty, Value: "two"},
										},
									},
								},
							},
						},
					},
				},
			},
		}, {
			description: "An array with a index out of bounds.",
			inputs: []string{
				"foo.bar.cat.0=zero",
				"foo.bar.cat.-2=two",
			},
			expected: Object{
				Origins: empty,
				Map: map[string]Object{
					"foo": {
						Origins: empty,
						Map: map[string]Object{
							"bar": {
								Origins: empty,
								Map: map[string]Object{
									"cat": {
										Origins: empty,
										Map: map[string]Object{
											"0":  {Origins: empty, Value: "zero"},
											"-2": {Origins: empty, Value: "two"},
										},
									},
								},
							},
						},
					},
				},
			},
		}, {
			description: "An array with a index that isn't a number.",
			inputs: []string{
				"foo.bar.cat.0=zero",
				"foo.bar.cat.two=two",
			},
			expected: Object{
				Origins: empty,
				Map: map[string]Object{
					"foo": {
						Origins: empty,
						Map: map[string]Object{
							"bar": {
								Origins: empty,
								Map: map[string]Object{
									"cat": {
										Origins: empty,
										Map: map[string]Object{
											"0":   {Origins: empty, Value: "zero"},
											"two": {Origins: empty, Value: "two"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			start := Object{
				Origins: []Origin{tc.origin},
			}
			for _, input := range tc.inputs {
				var err error

				kvp := strings.Split(input, "=")
				require.True(len(kvp) == 2)
				start, err = start.Add(".", kvp[0], kvp[1])
				require.NotNil(start)
				require.NoError(err)
			}

			got := start.ConvertMapsToArrays()

			assert.Equal(tc.expected, got)
		})
	}
}

func TestStringToBestType(t *testing.T) {
	tests := []struct {
		description string
		in          string
		expected    any
	}{
		{
			description: "An integer.",
			in:          "10",
			expected:    int64(10),
		}, {
			description: "Zero.",
			in:          "0",
			expected:    int64(0),
		}, {
			description: "A really large integer.",
			in:          "9223372036854775807",
			expected:    int64(9223372036854775807),
		}, {
			description: "A really, really big number.",
			in:          "92233720368547758070",
			expected:    float64(92233720368547758070),
		}, {
			description: "true",
			in:          "true",
			expected:    true,
		}, {
			description: "false",
			in:          "false",
			expected:    false,
		}, {
			description: "An actual string",
			in:          "cows",
			expected:    "cows",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			got := StringToBestType(tc.in)

			assert.Equal(tc.expected, got)
		})
	}
}

func TestToRedacted(t *testing.T) {
	tests := []struct {
		description string
		in          Object
		expected    Object
	}{
		{
			description: "Output an small tree.",
			in: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   "very secret.",
					},
				},
			},
			expected: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   redactedText,
					},
				},
			},
		}, {
			description: "Output an larger tree.",
			in: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						Map: map[string]Object{
							"bar": {
								Origins: []Origin{},
								Value:   int(123),
							},
							"car": {
								Origins: []Origin{},
								Array: []Object{
									{
										Origins: []Origin{},
										Map: map[string]Object{
											"sam": {
												Origins: []Origin{},
												secret:  true,
												Value:   "cart",
											},
										},
									}, {
										Origins: []Origin{},
										Value:   "golf",
									},
								},
							},
						},
					},
				},
			},
			expected: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						Map: map[string]Object{
							"bar": {
								Origins: []Origin{},
								Value:   int(123),
							},
							"car": {
								Origins: []Origin{},
								Array: []Object{
									{
										Origins: []Origin{},
										Map: map[string]Object{
											"sam": {
												Origins: []Origin{},
												secret:  true,
												Value:   redactedText,
											},
										},
									}, {
										Origins: []Origin{},
										Value:   "golf",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			got := tc.in.ToRedacted()

			assert.Equal(tc.expected, got)
		})
	}
}

func TestToExpanded(t *testing.T) {
	tests := []struct {
		description string
		in          Object
		expected    Object
		expectedErr error
		origin      string
		start       string
		end         string
		vars        map[string]string
	}{
		{
			description: "Output an unchanged tree.",
			in: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   "very secret.",
					},
					"bark": {
						Origins: []Origin{},
						Value:   12,
					},
					"candy": {
						Origins: []Origin{},
						Array: []Object{
							{
								Origins: []Origin{},
								Value:   "${{bar",
							},
						},
					},
				},
			},
			expected: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   "very secret.",
					},
					"bark": {
						Origins: []Origin{},
						Value:   12,
					},
					"candy": {
						Origins: []Origin{},
						Array: []Object{
							{
								Origins: []Origin{},
								Value:   "${{bar",
							},
						},
					},
				},
			},
			start: "${{",
			end:   "}}",
			vars: map[string]string{
				"unused": "foo",
			},
		}, {
			description: "Output a changed tree.",
			in: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   "very secret.",
					},
					"bark": {
						Origins: []Origin{},
						Value:   12,
					},
					"candy": {
						Origins: []Origin{},
						Array: []Object{
							{
								Origins: []Origin{},
								Value:   "${{bar}} ... ${{bar}}",
							},
						},
					},
				},
			},
			origin: "expanded:test",
			expected: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   "very secret.",
					},
					"bark": {
						Origins: []Origin{},
						Value:   12,
					},
					"candy": {
						Origins: []Origin{},
						Array: []Object{
							{
								Origins: []Origin{{File: "expanded:test"}},
								Value:   "food${{unknown}} ... food${{unknown}}",
							},
						},
					},
				},
			},
			start: "${{",
			end:   "}}",
			vars: map[string]string{
				"bar":  "${{foo}}",
				"foo":  "${{next}}${{unknown}}",
				"next": "food",
			},
		}, {
			description: "Recurse and fail.",
			in: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   "very secret.",
					},
					"bark": {
						Origins: []Origin{},
						Value:   12,
					},
					"candy": {
						Origins: []Origin{},
						Array: []Object{
							{
								Origins: []Origin{},
								Value:   "${{cat}} ${{bar}}",
							},
						},
					},
				},
			},
			start: "${{",
			end:   "}}",
			vars: map[string]string{
				"bar": "${{car}}",
				"car": "${{bar}}",
				"cat": "tom",
			},
			expectedErr: ErrRecursionTooDeep,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			max := int(100)
			got, err := tc.in.ToExpanded(max, tc.origin, tc.start, tc.end, func(in string) (string, bool) {
				out, found := tc.vars[in]
				return out, found
			})

			if tc.expectedErr == nil {
				assert.Equal(tc.expected, got)
				return
			}

			assert.ErrorIs(err, tc.expectedErr)
		})
	}
}

func TestExpand(t *testing.T) {
	tests := []struct {
		in          string
		start       string
		end         string
		vars        map[string]string
		expected    string
		expectedErr error
		changed     bool
	}{
		{
			in:       "nothing found",
			start:    "${{",
			end:      "}}",
			expected: "nothing found",
			changed:  false,
		}, {
			in:    "|nothing| found",
			start: "|",
			end:   "|",
			vars: map[string]string{
				"nothing": "something",
			},
			expected: "something found",
			changed:  true,
		}, {
			in:    "|nothing found",
			start: "|",
			end:   "|",
			vars: map[string]string{
				"nothing": "something",
			},
			expected: "|nothing found",
		}, {
			in:    "|oops|",
			start: "|",
			end:   "|",
			vars: map[string]string{
				"oops": "|oops|",
			},
			expectedErr: ErrRecursionTooDeep,
		}, {
			// This appears to be a bit of a special case for the os.Expand() function
			in:    "|nothing|",
			start: "|",
			end:   "|",
			vars: map[string]string{
				"nothing": "${nothing}",
			},
			expected: "${nothing}",
			changed:  true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			assert := assert.New(t)

			max := int(100)
			got, _, err := expand(&max, tc.in, tc.start, tc.end, func(in string) (string, bool) {
				out, found := tc.vars[in]
				return out, found
			})
			if tc.expectedErr == nil {
				assert.Equal(tc.expected, got)
				return
			}

			assert.ErrorIs(err, tc.expectedErr)
			assert.Equal("", got)
		})
	}
}

func TestAlterKeyCase(t *testing.T) {
	tests := []struct {
		description string
		in          string
		mapper      func(string) string
		expected    string
	}{
		{
			description: "Output a small tree.",
			in:          `{"foo":"something"}`,
			mapper:      strings.ToLower,
			expected:    `{"foo":"something"}`,
		}, {
			description: "Output a larger tree.",
			in:          `{"FOO":{ "BAR": "oNe", "Car": [ { "SAM": "CarT"}, "Golf" ] } }`,
			mapper:      strings.ToLower,
			expected:    `{"foo":{ "bar": "oNe", "car": [ { "sam": "CarT"}, "Golf" ] } }`,
		}, {
			description: "Output a larger tree but remove one item.",
			in:          `{"FOO":{ "BAR": "oNe", "Car": [ { "SAM": "CarT"}, "Golf" ] } }`,
			mapper: func(s string) string {
				if s == "Car" {
					return "-"
				}
				return strings.ToLower(s)
			},
			expected: `{"foo":{ "bar": "oNe" } }`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			in := decode(tc.in)
			expected := decode(tc.expected)

			got := in.AlterKeyCase(tc.mapper)

			assert.Equal(expected, got)
		})
	}
}

func TestResolveCommands(t *testing.T) {
	tests := []struct {
		description string
		in          string
		expected    Object
		expectedErr error
	}{
		{
			description: "Output a small tree.",
			in:          `{"foo ((secret))":"very secret."}`,
			expected: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   "very secret.",
					},
				},
			},
		}, {
			description: "Output a larger tree.",
			in:          `{"foo ((secret,splice))": {"bar":"123", "car((secret))":[{"sam((secret))":"cart"},"golf"]}}`,
			expected: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Map: map[string]Object{
							"bar": {
								Origins: []Origin{},
								Value:   "123",
							},
							"car": {
								Origins: []Origin{},
								secret:  true,
								Array: []Object{
									{
										Origins: []Origin{},
										Map: map[string]Object{
											"sam": {
												Origins: []Origin{},
												secret:  true,
												Value:   "cart",
											},
										},
									}, {
										Origins: []Origin{},
										Value:   "golf",
									},
								},
							},
						},
					},
				},
			},
		}, {
			description: "Output an larger tree.",
			expectedErr: ErrInvalidCommand,
			in:          `{"foo ((secret,splice))": {"bar":"123", "car((secret))":[{"sam((secret, invalid, command))":"cart"},"golf"]}}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			got, err := decode(tc.in).ResolveCommands()

			if tc.expectedErr == nil {
				assert.NoError(err)
				assert.Equal(tc.expected, got)
				return
			}

			assert.ErrorIs(tc.expectedErr, err)
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		description string
		in          string
		next        string
		expected    Object
		expectedErr error
	}{
		{
			description: "Cover basically all the valid conditions.",
			in: `{	"foo":"cats",
				   	"bar":"food",
				   	"squid":"ink",
				   	"wolf":{
				   		"man":"jack"
					},
				   	"fox":{
				   		"is":"red"
					},
				   	"cow":{
						"noise":"bark"
					},
				   	"ox":{
						"cart":["2"]
					},
				   	"mad":{
					   "cat":"crazy",
					   "money":[ "usd","euro" ]
				   	},
					"dogs":[
						"fido",
						"pluto"
					],
					"cats":[
						"scaredy",
						"cheezburger"
					]}`,
			next: `{	"foo((replace, secret))": "very secret.",
						"bar((keep))": "bats",
						"squid((keep))": {
							"giant": [ "20000" ]
						},
						"soap": "bar",
						"wolf": "of a different type",
						"fox((keep))": {
							"run": "fast"
						},
						"cow((replace))": {
							"eats": "hay"
						},
						"ox": {
							"cart((secret, append))": ["3"]
						},
				   		"mad":{
					   		"cat":"crazier",
					   		"money((prepend))":[ "cad","gbp" ]
						},
						"dogs((replace))":[
							{ "snoopy ((keep))": "beagle" }
						],
						"cats((keep))":["ignored"]}`,
			expected: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   "very secret.",
					},
					"bar": {
						Origins: []Origin{},
						Value:   "food",
					},
					"squid": {
						Origins: []Origin{},
						Value:   "ink",
					},
					"soap": {
						Origins: []Origin{},
						Value:   "bar",
					},
					"wolf": {
						Origins: []Origin{},
						Value:   "of a different type",
					},
					"fox": {
						Origins: []Origin{},
						Map: map[string]Object{
							"is": {
								Origins: []Origin{},
								Value:   "red",
							},
						},
					},
					"cow": {
						Origins: []Origin{},
						Map: map[string]Object{
							"eats": {
								Origins: []Origin{},
								Value:   "hay",
							},
						},
					},
					"ox": {
						Origins: []Origin{},
						Map: map[string]Object{
							"cart": {
								Origins: []Origin{},
								secret:  true,
								Array: []Object{
									{
										Origins: []Origin{},
										Value:   "2",
									}, {
										Origins: []Origin{},
										Value:   "3",
									},
								},
							},
						},
					},
					"mad": {
						Origins: []Origin{},
						Map: map[string]Object{
							"cat": {
								Origins: []Origin{},
								Value:   "crazier",
							},
							"money": {
								Origins: []Origin{},
								Array: []Object{
									{
										Origins: []Origin{},
										Value:   "cad",
									}, {
										Origins: []Origin{},
										Value:   "gbp",
									}, {
										Origins: []Origin{},
										Value:   "usd",
									}, {
										Origins: []Origin{},
										Value:   "euro",
									},
								},
							},
						},
					},
					"dogs": {
						Origins: []Origin{},
						Array: []Object{
							{
								Origins: []Origin{},
								Map: map[string]Object{
									"snoopy": {
										Origins: []Origin{},
										Value:   "beagle",
									},
								},
							},
						},
					},
					"cats": {
						Origins: []Origin{},
						Array: []Object{
							{
								Origins: []Origin{},
								Value:   "scaredy",
							}, {
								Origins: []Origin{},
								Value:   "cheezburger",
							},
						},
					},
				},
			},
		}, {
			description: "Clear the tree.",
			in:          `{"foo":"bar"}`,
			next:        `{"ignored((clear))":"ignored"}`,
			expected: Object{
				Origins: []Origin{},
			},
		}, {
			description: "Merge into an empty structure.",
			in:          `{}`,
			next:        `{"foo((secret))":"car"}`,
			expected: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   "car",
					},
				},
			},
		}, {
			description: "Merge into an empty structure, but fail due to invalid command.",
			in:          `{}`,
			next:        `{"foo((invalid))":"car"}`,
			expectedErr: ErrInvalidCommand,
		}, {
			description: "Append to a secret node and make sure it stays secret.",
			in:          `{"foo((secret))":["bar"]}`,
			next:        `{"foo((append))":["car"]}`,
			expected: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Array: []Object{
							{
								Origins: []Origin{},
								Value:   "bar",
							}, {
								Origins: []Origin{},
								Value:   "car",
							},
						},
					},
				},
			},
		}, {
			description: "Prepend to a secret node and make sure it stays secret.",
			in:          `{"foo((secret))":["bar"]}`,
			next:        `{"foo((prepend))":["car"]}`,
			expected: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Array: []Object{
							{
								Origins: []Origin{},
								Value:   "car",
							}, {
								Origins: []Origin{},
								Value:   "bar",
							},
						},
					},
				},
			},
		}, {
			description: "Error attempting to clear.",
			in:          `{"foo":"bar"}`,
			next:        `{"ignored((clear, secret, invalid))":"ignored"}`,
			expectedErr: ErrInvalidCommand,
		}, {
			description: "Error fail on conflict with value.",
			in:          `{"foo":"bar"}`,
			next:        `{"foo((fail))":"new"}`,
			expectedErr: ErrConflict,
		}, {
			description: "Error invalid command with value.",
			in:          `{"foo":"bar"}`,
			next:        `{"foo((invalid))":"new"}`,
			expectedErr: ErrInvalidCommand,
		}, {
			description: "Error fail on conflict with array.",
			in:          `{"foo":["bar"]}`,
			next:        `{"foo((fail))":["new"]}`,
			expectedErr: ErrConflict,
		}, {
			description: "Error invalid command with array.",
			in:          `{"foo":["bar"]}`,
			next:        `{"foo((invalid))":["new"]}`,
			expectedErr: ErrInvalidCommand,
		}, {
			description: "Error fail on conflict with map.",
			in:          `{"foo":{"goo":"bar"}}`,
			next:        `{"foo((fail))":{"goo":"new"}}`,
			expectedErr: ErrConflict,
		}, {
			description: "Error invalid command with map.",
			in:          `{"foo":{"goo":"bar"}}`,
			next:        `{"foo((invalid))":{"goo":"new"}}`,
			expectedErr: ErrInvalidCommand,
		}, {
			description: "Error invalid command with nested replace map.",
			in:          `{"foo":{"goo":"bar"}}`,
			next:        `{"foo((replace))":{"goo((invalid))":"new"}}`,
			expectedErr: ErrInvalidCommand,
		}, {
			description: "Error invalid command with nested array.",
			in:          `{"foo":{"goo":"bar"}}`,
			next:        `{"foo((replace))":[{"goo((invalid))":"new"}]}`,
			expectedErr: ErrInvalidCommand,
		}, {
			description: "Error invalid command with nested array.",
			in:          `{"foo":{"goo":["bar", {"cat":"dog"}]}}`,
			next:        `{"foo":{"goo":[{"sam((invalid))":"eagle"}]}}`,
			expectedErr: ErrInvalidCommand,
		}, {
			description: "Error invalid command with new key.",
			in:          `{"foo":{"goo":"bar"}}`,
			next:        `{"cat":{"goo((invalid))":"new"}}`,
			expectedErr: ErrInvalidCommand,
		}, {
			description: "Error conflict due to fail.",
			in:          `{"foo":{"goo":"crab"}}`,
			next:        `{"foo":{"goo((fail))":{"new": "car"}}}`,
			expectedErr: ErrConflict,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			in, err := decode(tc.in).resolveCommands(false)
			require.NoError(err)
			next := decode(tc.next)

			got, err := in.Merge(next)

			if tc.expectedErr == nil {
				assert.NoError(err)
				assert.Equal(tc.expected, got)
				return
			}

			assert.ErrorIs(err, tc.expectedErr)
		})
	}
}

func TestOrigin_OriginString(t *testing.T) {
	tests := []struct {
		description string
		obj         Object
		expected    string
	}{
		{
			description: "Output an empty object.",
			expected:    "",
		}, {
			description: "Output an empty origin.",
			obj: Object{
				Origins: []Origin{{}},
			},
			expected: "unknown",
		}, {
			description: "Output an filled out origin.",
			obj: Object{
				Origins: []Origin{
					{
						File: "magic.file",
						Line: 42,
						Col:  88,
					},
				},
			},
			expected: "magic.file:42[88]",
		}, {
			description: "Output an filled out origin.",
			obj: Object{
				Origins: []Origin{
					{
						File: "magic.file",
						Line: 42,
						Col:  88,
					}, {
						File: "foo.json",
						Line: 96,
						Col:  32,
					},
				},
			},
			expected: "magic.file:42[88], foo.json:96[32]",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			got := tc.obj.OriginString()

			assert.Equal(tc.expected, got)
		})
	}
}

func TestIsSerializable(t *testing.T) {
	testFunc := func() {}
	ch := make(chan string)

	tests := []struct {
		description string
		thing       Object
		expected    bool
	}{
		{
			description: "simple string",
			expected:    true,
			thing: Object{
				Value: "fish",
			},
		}, {
			description: "empty object",
			expected:    true,
		}, {
			description: "functions can't be serialized",
			expected:    false,
			thing: Object{
				Value: testFunc,
			},
		}, {
			description: "pointers to functions can't be serialized",
			expected:    false,
			thing: Object{
				Value: &testFunc,
			},
		}, {
			description: "channels can't be serialized",
			expected:    false,
			thing: Object{
				Value: ch,
			},
		}, {
			description: "pointers to channels can't be serialized",
			expected:    false,
			thing: Object{
				Value: &ch,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			got := tc.thing.isSerializable()

			assert.Equal(tc.expected, got)
		})
	}
}

func TestFilterNonSerializable(t *testing.T) {
	origin := Origin{
		File: "file",
		Line: 12,
		Col:  36,
	}
	tests := []struct {
		description string
		thing       any
		expected    any
	}{
		{
			description: "everything serializes",
			thing: map[string]any{
				"one": []any{
					map[string]any{"fish": "blue"},
					map[string]any{"dog": "red"},
				},
			},
			expected: map[string]any{
				"one": []any{
					map[string]any{"fish": "blue"},
					map[string]any{"dog": "red"},
				},
			},
		}, {
			description: "one thing does not serializes",
			thing: map[string]any{
				"one": []any{
					map[string]any{"fish": "blue"},
					map[string]any{"dog": func() {}},
				},
			},
			expected: map[string]any{
				"one": []any{
					map[string]any{"fish": "blue"},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			obj := ObjectFromRawWithOrigin(tc.thing, []Origin{origin})
			want := ObjectFromRawWithOrigin(tc.expected, []Origin{origin})
			got := obj.FilterNonSerializable()

			assert.Equal(want, got)
		})
	}
}

func TestErrOnNonSerializable(t *testing.T) {
	origin := Origin{
		File: "file",
		Line: 12,
		Col:  36,
	}
	tests := []struct {
		description string
		thing       any
		expected    error
	}{
		{
			description: "everything serializes",
			thing: map[string]any{
				"one": []any{
					map[string]any{"fish": "blue"},
					map[string]any{"dog": "red"},
				},
			},
		}, {
			description: "one thing does not serializes",
			expected:    ErrNonSerializable,
			thing: map[string]any{
				"one": []any{
					map[string]any{"fish": "blue"},
					map[string]any{"dog": func() {}},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			obj := ObjectFromRawWithOrigin(tc.thing, []Origin{origin})
			err := obj.ErrOnNonSerializable()

			if tc.expected == nil {
				return
			}

			assert.ErrorIs(err, tc.expected)
		})
	}
}

func TestAdaptToRaw(t *testing.T) {
	unknownErr := errors.New("unknownErr")
	common := Object{
		Map: map[string]Object{
			"foo": {
				Map: map[string]Object{
					"bar": {
						Value: time.Second,
					},
					"car": {
						Array: []Object{
							{
								Map: map[string]Object{
									"joe": {
										Value: "other",
									},
									"sam": {
										Value: 10 * time.Second,
									},
								},
							}, {
								Value: 15 * time.Second,
							}, {
								Value: time.Date(2022, time.December, 0, 0, 0, 0, 0, time.UTC),
							}, {
								Value: nil,
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		description string
		thing       Object
		adapter     func(from, to reflect.Value) (any, error)
		expected    Object
		expectedErr error
	}{
		{
			description: "nil adapter",
			thing:       common,
			expected:    common,
		}, {
			description: "duration adapter",
			thing:       common,
			adapter: func(from, to reflect.Value) (any, error) {
				if from.Type() == reflect.TypeOf(time.Second) &&
					to.Type() == reflect.TypeOf("string") {
					return from.Interface().(time.Duration).String(), nil
				}

				return from.Interface(), nil
			},
			expected: Object{
				Map: map[string]Object{
					"foo": {
						Map: map[string]Object{
							"bar": {
								Value: "1s",
							},
							"car": {
								Array: []Object{
									{
										Map: map[string]Object{
											"joe": {
												Value: "other",
											},
											"sam": {
												Value: "10s",
											},
										},
									}, {
										Value: "15s",
									}, {
										Value: time.Date(2022, time.December, 0, 0, 0, 0, 0, time.UTC),
									}, {
										Value: nil,
									},
								},
							},
						},
					},
				},
			},
		}, {
			description: "time adapter",
			thing:       common,
			adapter: func(from, to reflect.Value) (any, error) {
				if from.Type() == reflect.TypeOf(time.Time{}) &&
					to.Type() == reflect.TypeOf("string") {
					return from.Interface().(time.Time).Format("2006"), nil
				}

				return from.Interface(), nil
			},
			expected: Object{
				Map: map[string]Object{
					"foo": {
						Map: map[string]Object{
							"bar": {
								Value: time.Second,
							},
							"car": {
								Array: []Object{
									{
										Map: map[string]Object{
											"joe": {
												Value: "other",
											},
											"sam": {
												Value: time.Second * 10,
											},
										},
									}, {
										Value: time.Second * 15,
									}, {
										Value: "2022",
									}, {
										Value: nil,
									},
								},
							},
						},
					},
				},
			},
		}, {
			description: "return an error for the 'other' string field",
			thing:       common,
			adapter: func(from, to reflect.Value) (any, error) {
				if from.Type() == reflect.TypeOf("string") {
					return nil, unknownErr
				}
				return from.Interface(), nil
			},
			expectedErr: unknownErr,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			got, err := tc.thing.AdaptToRaw(tc.adapter)

			assert.Equal(tc.expected, got)
			if tc.expectedErr == nil {
				assert.NoError(err)
				return
			}

			assert.ErrorIs(err, tc.expectedErr)
		})
	}
}

func TestClone(t *testing.T) {
	tests := []struct {
		description string
		in          Object
	}{
		{
			description: "Output an small tree.",
			in: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						secret:  true,
						Value:   "very secret.",
					},
				},
			},
		}, {
			description: "Output an larger tree.",
			in: Object{
				Origins: []Origin{},
				Map: map[string]Object{
					"foo": {
						Origins: []Origin{},
						Map: map[string]Object{
							"bar": {
								Origins: []Origin{},
								Value:   int(123),
							},
							"car": {
								Origins: []Origin{},
								Array: []Object{
									{
										Origins: []Origin{},
										Map: map[string]Object{
											"sam": {
												Origins: []Origin{},
												secret:  true,
												Value:   "cart",
											},
										},
									}, {
										Origins: []Origin{},
										Value:   "golf",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			got := tc.in.Clone()

			assert.Equal(tc.in, got)
		})
	}
}
