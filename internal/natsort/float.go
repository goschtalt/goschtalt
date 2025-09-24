// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package natsort

import (
	"strconv"
)

// CompareFloat compares two strings, attempting to parse them as
// numbers (int/float) for comparison; if parsing fails, it falls back
// to standard string comparison.
func CompareFloat(a, b string) bool {
	aNum, aErr := strconv.ParseFloat(a, 64)
	bNum, bErr := strconv.ParseFloat(b, 64)

	// If both are valid numbers, compare numerically.
	if aErr == nil && bErr == nil {
		if aNum == bNum {
			// If numeric values are the same, compare strings to break ties
			return a < b
		}
		return aNum < bNum
	}

	// If only one is numeric, that one goes first.
	if aErr == nil {
		return true
	}
	if bErr == nil {
		return false
	}

	// Otherwise, do a normal string compare.
	return a < b
}
