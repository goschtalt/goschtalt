// SPDX-FileCopyrightText: 2015 Vincent Batoufflet and Marc Falzon
// SPDX-FileCopyrightText: 2022 Mark Karpelès
// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: BSD-3-Clause
//
// This file originated from https://github.com/facette/natsort/pull/2/files

package natsort

import (
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stringSliceFloat []string

func (s stringSliceFloat) Len() int {
	return len(s)
}

func (s stringSliceFloat) Less(a, b int) bool {
	return CompareFloat(s[a], s[b])
}

func (s stringSliceFloat) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

// SortFloat sorts a list of strings in a natural order
func SortFloat(l []string) {
	sort.Sort(stringSliceFloat(l))
}

var testFloatList = []string{
	"0",
	"0.0",
	"1",
	"1.1",
	"12",
	"12.01",
	"A",
	"a",
	"b",
	"c",
}

func TestFloatSort(t *testing.T) {
	tests := []struct {
		description string
		want        []string
	}{
		{
			description: "Test the benchmark testList",
			want:        testFloatList,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// The combinations are a factorial, so limit it to 5040 runs
			r := rand.New(rand.NewSource(time.Now().UnixNano())) // nolint:gosec

			// We can't fully cover the combinations, so randomly mix them up.
			for i := 0; i < 1; i++ {
				list := make([]string, len(tc.want))
				copy(list, tc.want)

				/* shuffle the list, randomly */
				r.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })

				runFloat(assert, require, list, tc.want)
			}
		})
	}
}

func runFloat(assert *assert.Assertions, _ *require.Assertions, list, want []string) {
	start := make([]string, len(list))
	copy(start, list)

	SortFloat(list)

	assert.Equal(want, list)
}

func BenchmarkFloatSort(b *testing.B) {
	for n := 0; n < b.N; n++ {
		SortFloat(testFloatList)
	}
}

func TestFloat(t *testing.T) {
	tests := []struct {
		name     string
		i        string
		j        string
		expected bool
	}{
		// Both are numbers
		{
			name:     "both integers, i < j",
			i:        "5",
			j:        "10",
			expected: true,
		},
		{
			name:     "both integers, i > j",
			i:        "10",
			j:        "5",
			expected: false,
		},
		{
			name:     "both integers, i == j",
			i:        "5",
			j:        "5",
			expected: false, // "5" == "5" so returns i < j which is false
		},
		{
			name:     "both floats, i < j",
			i:        "3.14",
			j:        "3.15",
			expected: true,
		},
		{
			name:     "both floats, i > j",
			i:        "3.15",
			j:        "3.14",
			expected: false,
		},
		{
			name:     "same numeric value, different string representation",
			i:        "5.0",
			j:        "5",
			expected: false, // 5.0 == 5.0, falls back to string comparison: "5.0" < "5" is false
		},
		// One is number, one is not
		{
			name:     "i is number, j is string",
			i:        "10",
			j:        "abc",
			expected: true,
		},
		{
			name:     "i is string, j is number",
			i:        "abc",
			j:        "10",
			expected: false,
		},
		// Both are strings
		{
			name:     "both strings, i < j",
			i:        "apple",
			j:        "banana",
			expected: true,
		},
		{
			name:     "both strings, i > j",
			i:        "banana",
			j:        "apple",
			expected: false,
		},
		{
			name:     "both strings, i == j",
			i:        "apple",
			j:        "apple",
			expected: false,
		},
		// Edge cases
		{
			name:     "empty strings",
			i:        "",
			j:        "",
			expected: false,
		},
		{
			name:     "empty vs non-empty",
			i:        "",
			j:        "a",
			expected: true,
		},
		{
			name:     "negative numbers",
			i:        "-5",
			j:        "-10",
			expected: false, // -5 > -10
		},
		{
			name:     "zero and positive",
			i:        "0",
			j:        "1",
			expected: true,
		},
		{
			name:     "zero and zero",
			i:        "0.0",
			j:        "0",
			expected: false,
		},
		{
			name:     "zero and zero",
			i:        "1.1",
			j:        "0.0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareFloat(tt.i, tt.j)
			assert.Equal(t, tt.expected, result)
		})
	}
}
