// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/goschtalt/goschtalt/pkg/doc"
	"github.com/goschtalt/goschtalt/pkg/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStringer implements fmt.Stringer for testing
type mockStringer struct {
	value string
}

func (m mockStringer) String() string {
	return m.value
}

func TestUnified_Indent(t *testing.T) {
	tests := []struct {
		name   string
		indent int
	}{
		{
			name:   "zero indent",
			indent: 0,
		}, {
			name:   "positive indent",
			indent: 5,
		}, {
			name:   "negative indent",
			indent: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &unified{indent: tt.indent}
			assert.Equal(t, tt.indent, u.Indent())
		})
	}
}

func TestUnified_Headers(t *testing.T) {
	tests := []struct {
		name     string
		unified  *unified
		expected []string
	}{
		{
			name: "nil doc and preset",
			unified: &unified{
				doc:    nil,
				preset: nil,
			},
			expected: nil,
		}, {
			name: "doc with single line, no deprecated, no preset",
			unified: &unified{
				doc: &doc.Object{
					Doc:        "This is a test documentation",
					Deprecated: false,
					Type:       doc.TYPE_STRING,
				},
				preset: nil,
			},
			expected: []string{
				"This is a test documentation",
				"type: <string>",
			},
		}, {
			name: "doc with multiline, deprecated, with preset",
			unified: &unified{
				doc: &doc.Object{
					Doc:        "Line 1\nLine 2\nLine 3",
					Deprecated: true,
					Type:       doc.TYPE_INT,
				},
				preset: 42,
			},
			expected: []string{
				"!!! DEPRECATED !!!",
				"Line 1",
				"Line 2",
				"Line 3",
				"type: <int>",
				"default: 42",
				"!!! DEPRECATED !!!",
			},
		}, {
			name: "only preset, no doc",
			unified: &unified{
				doc:    nil,
				preset: "default_value",
			},
			expected: []string{
				"default: default_value",
			},
		}, {
			name: "doc with empty string",
			unified: &unified{
				doc: &doc.Object{
					Doc:        "",
					Deprecated: false,
					Type:       doc.TYPE_BOOL,
				},
				preset: nil,
			},
			expected: []string{
				"",
				"type: <bool>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.unified.Headers()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnified_Inline(t *testing.T) {
	u := &unified{}
	result := u.Inline()
	assert.Nil(t, result, "Inline should always return nil")
}

func TestUnified_Key(t *testing.T) {
	tests := []struct {
		name     string
		key      *string
		expected *string
	}{
		{
			name:     "nil key",
			key:      nil,
			expected: nil,
		}, {
			name:     "non-nil key",
			key:      stringPtr("test_key"),
			expected: stringPtr("test_key"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &unified{key: tt.key}
			result := u.Key()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestUnified_Value(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected *string
	}{
		{
			name:     "nil value",
			value:    nil,
			expected: nil,
		},
		{
			name:     "string value",
			value:    "test_string",
			expected: stringPtr("test_string"),
		},
		{
			name:     "integer value",
			value:    42,
			expected: stringPtr("42"),
		},
		{
			name:     "float value",
			value:    3.14,
			expected: stringPtr("3.140000"),
		},
		{
			name:     "boolean value",
			value:    true,
			expected: stringPtr("true"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &unified{value: tt.value}
			result := u.Value()
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestUnified_Children(t *testing.T) {
	tests := []struct {
		name     string
		children map[string]unified
		expected int // number of expected children
	}{
		{
			name:     "no children",
			children: nil,
			expected: 0,
		},
		{
			name:     "empty children map",
			children: make(map[string]unified),
			expected: 0,
		},
		{
			name: "single child",
			children: map[string]unified{
				"child1": {key: stringPtr("child1"), value: "value1"},
			},
			expected: 1,
		},
		{
			name: "multiple children",
			children: map[string]unified{
				"child1": {key: stringPtr("child1"), value: "value1"},
				"child2": {key: stringPtr("child2"), value: "value2"},
				"child3": {key: stringPtr("child3"), value: "value3"},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &unified{children: tt.children}
			result := u.Children()

			if tt.expected == 0 {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Len(t, result, tt.expected)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "nil value",
			value:    nil,
			expected: "<nil>",
		},
		{
			name:     "string value",
			value:    "hello world",
			expected: "hello world",
		},
		{
			name:     "fmt.Stringer implementation",
			value:    mockStringer{value: "custom string"},
			expected: "custom string",
		},
		{
			name:     "int value",
			value:    42,
			expected: "42",
		},
		{
			name:     "int8 value",
			value:    int8(8),
			expected: "8",
		},
		{
			name:     "int16 value",
			value:    int16(16),
			expected: "16",
		},
		{
			name:     "int32 value",
			value:    int32(32),
			expected: "32",
		},
		{
			name:     "int64 value",
			value:    int64(64),
			expected: "64",
		},
		{
			name:     "uint value",
			value:    uint(42),
			expected: "42",
		},
		{
			name:     "uint8 value",
			value:    uint8(8),
			expected: "8",
		},
		{
			name:     "uint16 value",
			value:    uint16(16),
			expected: "16",
		},
		{
			name:     "uint32 value",
			value:    uint32(32),
			expected: "32",
		},
		{
			name:     "uint64 value",
			value:    uint64(64),
			expected: "64",
		},
		{
			name:     "float32 value",
			value:    float32(3.14),
			expected: "3.140000",
		},
		{
			name:     "float64 value",
			value:    float64(2.718281828),
			expected: "2.718282",
		},
		{
			name:     "bool value",
			value:    true,
			expected: "true",
		},
		{
			name:     "time.Time value",
			value:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: "2023-01-01 00:00:00 +0000 UTC",
		},
		{
			name:     "slice value",
			value:    []string{"a", "b", "c"},
			expected: "[a b c]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUnified_ChildrenKeys(t *testing.T) {
	tests := []struct {
		name     string
		children map[string]unified
		expected []string
	}{
		{
			name:     "empty children",
			children: make(map[string]unified),
			expected: []string{},
		},
		{
			name: "single child",
			children: map[string]unified{
				"key1": {},
			},
			expected: []string{"key1"},
		},
		{
			name: "multiple string keys sorted alphabetically",
			children: map[string]unified{
				"zebra": {},
				"alpha": {},
				"beta":  {},
			},
			expected: []string{"alpha", "beta", "zebra"},
		},
		{
			name: "numeric keys sorted numerically",
			children: map[string]unified{
				"10": {},
				"2":  {},
				"1":  {},
				"20": {},
			},
			expected: []string{"1", "2", "10", "20"},
		},
		{
			name: "mixed numeric and string keys",
			children: map[string]unified{
				"10":    {},
				"alpha": {},
				"2":     {},
				"beta":  {},
				"1":     {},
			},
			expected: []string{"1", "2", "10", "alpha", "beta"},
		},
		{
			name: "float keys",
			children: map[string]unified{
				"3.14": {},
				"2.71": {},
				"1.41": {},
			},
			expected: []string{"1.41", "2.71", "3.14"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := unified{children: tt.children}
			result := u.childKeys()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalcUnified(t *testing.T) {
	tests := []struct {
		name        string
		doc         *doc.Object
		compiled    *meta.Object
		withOrigins bool
		wantErr     bool
		validate    func(t *testing.T, result unified)
	}{
		{
			name:     "nil inputs",
			doc:      nil,
			compiled: nil,
			wantErr:  false,
			validate: func(t *testing.T, result unified) {
				assert.Equal(t, -1, result.indent)
				assert.Nil(t, result.key)
				assert.Nil(t, result.doc)
				assert.Nil(t, result.value)
				assert.False(t, result.array)
				assert.Len(t, result.children, 0)
			},
		},
		{
			name: "leaf node with value",
			doc:  nil,
			compiled: &meta.Object{
				Value: "test_value",
			},
			wantErr: false,
			validate: func(t *testing.T, result unified) {
				assert.Equal(t, -1, result.indent)
				assert.Equal(t, "test_value", result.value)
				assert.Len(t, result.children, 0)
			},
		},
		{
			name: "array from doc",
			doc: &doc.Object{
				Type: doc.TYPE_ARRAY,
				Children: map[string]doc.Object{
					doc.NAME_ARRAY: {
						Type: doc.TYPE_STRING,
					},
				},
			},
			compiled: &meta.Object{
				Array: []meta.Object{
					{Value: "item1"},
					{Value: "item2"},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result unified) {
				assert.True(t, result.array)
				assert.Len(t, result.children, 2)
				assert.Contains(t, result.children, "0")
				assert.Contains(t, result.children, "1")
			},
		},
		{
			name: "map object",
			doc: &doc.Object{
				Type: doc.TYPE_STRUCT,
				Children: map[string]doc.Object{
					"field1": {Type: doc.TYPE_STRING},
					"field2": {Type: doc.TYPE_INT},
				},
			},
			compiled: &meta.Object{
				Map: map[string]meta.Object{
					"field1": {Value: "value1"},
					"field2": {Value: 42},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result unified) {
				assert.False(t, result.array)
				assert.Len(t, result.children, 2)
				assert.Contains(t, result.children, "field1")
				assert.Contains(t, result.children, "field2")
			},
		},
		{
			name: "conflicting array and map",
			doc: &doc.Object{
				Type: doc.TYPE_ARRAY,
			},
			compiled: &meta.Object{
				Map: map[string]meta.Object{
					"field1": {Value: "value1"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calcUnified(tt.doc, tt.compiled, tt.withOrigins)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestCalcUnifiedInternal(t *testing.T) {
	tests := []struct {
		name        string
		namePtr     *string
		indent      int
		doc         *doc.Object
		compiled    *meta.Object
		withOrigins bool
		wantErr     bool
		validate    func(t *testing.T, result unified)
	}{
		{
			name:     "basic leaf node",
			namePtr:  stringPtr("test_key"),
			indent:   2,
			doc:      nil,
			compiled: &meta.Object{Value: "leaf_value"},
			wantErr:  false,
			validate: func(t *testing.T, result unified) {
				assert.Equal(t, 2, result.indent)
				assert.Equal(t, "test_key", *result.key)
				assert.Equal(t, "leaf_value", result.value)
				assert.Len(t, result.children, 0)
			},
		},
		{
			name:    "map with documentation",
			namePtr: nil,
			indent:  0,
			doc: &doc.Object{
				Type: doc.TYPE_STRUCT,
				Children: map[string]doc.Object{
					doc.NAME_MAP_KEY: {
						Type: doc.TYPE_STRING,
					},
					doc.NAME_MAP_VALUE: {
						Type: doc.TYPE_STRING,
					},
				},
			},
			compiled: &meta.Object{
				Map: map[string]meta.Object{
					"field1": {Value: "value1"},
				},
			},
			wantErr: false,
		},
		{
			name:    "array with NAME_ARRAY key error",
			namePtr: nil,
			indent:  0,
			doc: &doc.Object{
				Type: doc.TYPE_STRUCT,
				Children: map[string]doc.Object{
					doc.NAME_ARRAY: {},
				},
			},
			compiled: &meta.Object{
				Map: map[string]meta.Object{
					"field1": {Value: "value1"},
				},
			},
			wantErr: true,
		},
		{
			name:    "embedded key error",
			namePtr: nil,
			indent:  0,
			doc: &doc.Object{
				Type: doc.TYPE_STRUCT,
				Children: map[string]doc.Object{
					doc.NAME_EMBEDDED: {},
				},
			},
			compiled: &meta.Object{
				Map: map[string]meta.Object{
					"field1": {Value: "value1"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calcUnifiedInternal(tt.namePtr, tt.indent, tt.doc, tt.compiled, tt.withOrigins)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// Benchmark tests for performance-critical functions
func BenchmarkToString(b *testing.B) {
	values := []any{
		"string_value",
		42,
		3.14159,
		true,
		mockStringer{value: "stringer_value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range values {
			toString(v)
		}
	}
}

func BenchmarkChildrenKeys(b *testing.B) {
	children := make(map[string]unified)
	for i := 0; i < 100; i++ {
		children[strconv.Itoa(i)] = unified{}
		children[fmt.Sprintf("key_%d", i)] = unified{}
	}

	u := unified{children: children}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		u.childKeys()
	}
}
