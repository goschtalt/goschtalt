// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package yaml

import (
	"fmt"
	"strings"
)

// dump creates xxd-style output for a string
func dump(s string) string {
	data := []byte(s)
	var buf strings.Builder

	for i := 0; i < len(data); i += 16 {
		// Offset
		fmt.Fprintf(&buf, "%08x: ", i)

		// Hex bytes (2 groups of 8)
		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				fmt.Fprintf(&buf, "%02x", data[i+j])
			} else {
				buf.WriteString("  ")
			}
			if j == 7 {
				buf.WriteString(" ")
			}
		}

		buf.WriteString("  ")

		// ASCII representation
		for j := 0; j < 16 && i+j < len(data); j++ {
			b := data[i+j]
			if b >= 32 && b <= 126 {
				buf.WriteByte(b)
			} else {
				buf.WriteByte('.')
			}
		}

		buf.WriteString("\n")
	}

	return buf.String()
}
