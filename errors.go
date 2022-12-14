// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import "errors"

var (
	ErrDecoding      = errors.New("decoding error")
	ErrEncoding      = errors.New("encoding error")
	ErrNotCompiled   = errors.New("the Compile() function must be called first")
	ErrCodecNotFound = errors.New("encoder/decoder not found")
	ErrInvalidInput  = errors.New("input is invalid")
	ErrFileMissing   = errors.New("required file is missing")
)
