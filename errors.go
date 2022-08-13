// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import "errors"

var (
	ErrDecoding       = errors.New("decoding error")
	ErrDuplicateFound = errors.New("duplicate found")
	ErrEncoding       = errors.New("encoding error")
	ErrNotCompiled    = errors.New("the Compile() function must be called first")
	ErrNotFound       = errors.New("not found")
	ErrTypeMismatch   = errors.New("type mismatch")
)
