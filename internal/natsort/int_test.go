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

type stringSlice []string

func (s stringSlice) Len() int {
	return len(s)
}

func (s stringSlice) Less(a, b int) bool {
	return CompareInteger(s[a], s[b])
}

func (s stringSlice) Swap(a, b int) {
	s[a], s[b] = s[b], s[a]
}

// SortInt sorts a list of strings in a natural order
func SortInt(l []string) {
	sort.Sort(stringSlice(l))
}

var testIntList = []string{
	"1000X Radonius Maximus",
	"10X Radonius",
	"00010X Radonius 2",
	"200X Radonius",
	"20X Radonius",
	"20X Radonius Prime",
	"30X Radonius",
	"40X Radonius",
	"Allegia 50 Clasteron",
	"Allegia 500 Clasteron",
	"Allegia 50B Clasteron",
	"Allegia 51 Clasteron",
	"Allegia 6R Clasteron",
	"Alpha 100",
	"Alpha 2",
	"Alpha 200",
	"Alpha 2A",
	"Alpha 2A-8000",
	"Alpha 2A-900",
	"Callisto Morphamax",
	"Callisto Morphamax 500",
	"Callisto Morphamax 5000",
	"Callisto Morphamax 600",
	"Callisto Morphamax 6000 SE",
	"Callisto Morphamax 6000 SE2",
	"Callisto Morphamax 700",
	"Callisto Morphamax 7000",
	"Xiph Xlater 10000",
	"Xiph Xlater 2000",
	"Xiph Xlater 300",
	"Xiph Xlater 40",
	"Xiph Xlater 5",
	"Xiph Xlater 50",
	"Xiph Xlater 500",
	"Xiph Xlater 5000",
	"Xiph Xlater 58",
}

func TestIntSort(t *testing.T) {
	tests := []struct {
		description string
		want        []string
	}{
		{
			description: "Test the benchmark testList",
			want: []string{
				"10X Radonius",
				"00010X Radonius 2",
				"20X Radonius",
				"20X Radonius Prime",
				"30X Radonius",
				"40X Radonius",
				"200X Radonius",
				"1000X Radonius Maximus",
				"Allegia 6R Clasteron",
				"Allegia 50 Clasteron",
				"Allegia 50B Clasteron",
				"Allegia 51 Clasteron",
				"Allegia 500 Clasteron",
				"Alpha 2",
				"Alpha 2A",
				"Alpha 2A-900",
				"Alpha 2A-8000",
				"Alpha 100",
				"Alpha 200",
				"Callisto Morphamax",
				"Callisto Morphamax 500",
				"Callisto Morphamax 600",
				"Callisto Morphamax 700",
				"Callisto Morphamax 5000",
				"Callisto Morphamax 6000 SE",
				"Callisto Morphamax 6000 SE2",
				"Callisto Morphamax 7000",
				"Xiph Xlater 5",
				"Xiph Xlater 40",
				"Xiph Xlater 50",
				"Xiph Xlater 58",
				"Xiph Xlater 300",
				"Xiph Xlater 500",
				"Xiph Xlater 2000",
				"Xiph Xlater 5000",
				"Xiph Xlater 10000",
			},
		}, {
			description: "Test a different list with numbers in the middle.",
			want: []string{
				"z1.doc",
				"z2.doc",
				"z3.doc",
				"z4.doc",
				"z5.doc",
				"z6.doc",
				"z7.doc",
				"z8.doc",
				"z9.doc",
				"z10.doc",
				"z11.doc",
				"z12.doc",
				"z13.doc",
				"z14.doc",
				"z15.doc",
				"z16.doc",
				"z17.doc",
				"z18.doc",
				"z19.doc",
				"z20.doc",
				"z100.doc",
				"z101.doc",
				"z102.doc",
			},
		}, {
			description: "Test using more representative list.",
			want: []string{
				"01_foo.yml",
				"2_foo.yml",
				"98_foo.yml",
				"99 dogs.yml",
				"99_Abc.yml",
				"99_cli.yml",
				"99_mine.yml",
				"100_alpha.yml",
			},
		}, {
			description: "Floating point numbers... don't use them.",
			want: []string{
				"01_foo.yml",
				"2_foo.yml",
				"98_foo.yml",
				"99.01_Abc.yml",
				"99.99_cli.yml",
				"99_dogs.yml", // Not where you would expect it to be!!!
				"99_mine.yml", // Not where you would expect it to be!!!
				"100_alpha.yml",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			// The combinations are a factorial, so limit it to 5040 runs
			r := rand.New(rand.NewSource(time.Now().UnixNano())) // nolint:gosec

			// We can't fully cover the combinations, so randomly mix them up.
			for i := 0; i < 1000; i++ {
				list := make([]string, len(tc.want))
				copy(list, tc.want)

				/* shuffle the list, randomly */
				r.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })

				runInt(assert, require, list, tc.want)
			}
		})
	}
}

func runInt(assert *assert.Assertions, _ *require.Assertions, list, want []string) {
	start := make([]string, len(list))
	copy(start, list)

	SortInt(list)

	assert.Equal(want, list)
}

func BenchmarkIntSort(b *testing.B) {
	for n := 0; n < b.N; n++ {
		SortInt(testIntList)
	}
}
