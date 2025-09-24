// SPDX-FileCopyrightText: 2015 Vincent Batoufflet and Marc Falzon
// SPDX-FileCopyrightText: 2022 Mark Karpelès
// SPDX-FileCopyrightText: 2022 Weston Schmidt
// SPDX-License-Identifier: BSD-3-Clause
//
// This file originated from https://github.com/facette/natsort/pull/2/files

package natsort

import (
	"strings"
)

func isDigit(c byte) bool {
	switch c {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

func determineNumberBlock(s string, val byte, max int, pos *int) int {
	num_len := 1
	for *pos+num_len < max {
		x := s[*pos+num_len]
		if val == '0' {
			*pos += 1
			val = x
			continue
		}
		if isDigit(x) {
			num_len += 1
		} else {
			break
		}
	}

	return num_len
}

// CompareInteger compares two strings and provides the order in natural sort order.
//
// Note that only integers are compared.  Floating point numbers are treated
// like 2 integers separated by the '.' rune.  See the test file for a list
// examples.
//
// Note that any leading 0 values are dropped from the number.
func CompareInteger(a, b string) bool {
	len_a := len(a)
	len_b := len(b)
	pos_a := 0 // Position of the character under review.
	pos_b := 0 // Position of the character under review.

	for pos_a < len_a && pos_b < len_b {
		val_a, val_b := a[pos_a], b[pos_b]

		// Go into numeric mode if the next chunks are numbers.
		if isDigit(val_a) && isDigit(val_b) {
			num_len_a := determineNumberBlock(a, val_a, len_a, &pos_a)
			num_len_b := determineNumberBlock(b, val_b, len_b, &pos_b)

			// both have same length, let's compare as string
			if num_len_a != num_len_b {
				return num_len_a < num_len_b
			}

			// both have same length, let's compare as string
			v := strings.Compare(a[pos_a:pos_a+num_len_a], b[pos_b:pos_b+num_len_b])
			if v != 0 {
				return v < 0
			}

			// equal
			pos_a += num_len_a
			pos_b += num_len_b
			continue
		}

		if val_a == val_b {
			pos_a += 1
			pos_b += 1
			continue
		}

		return val_a < val_b
	}

	if len_a <= pos_a {
		// eof on both at the same time (equal)
		return pos_b < len_b
	}
	// eof on b
	return false
}
