// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"github.com/goschtalt/goschtalt/pkg/decoder"
	"github.com/goschtalt/goschtalt/pkg/meta"
)

// record is the basic unit needed to define a configuration and it's name.
// With this information all the records can be decoded.
type record struct {
	name    string // The original, unmodified filename shown to users
	sortKey string // Internal key used for sorting (may contain prefixes)
	val     *value
	buf     *buffer
	tree    meta.Object
}

// fetch normalizes the calls to the val or encoded types of records.
func (rec *record) fetch(delimiter string, u Unmarshaler, decoders *codecRegistry[decoder.Decoder], defaultOpts []ValueOption) error {
	if rec.val != nil {
		tree, err := rec.val.toTree(delimiter, u, defaultOpts...)
		if err != nil {
			return err
		}
		rec.tree = tree
	}

	if rec.buf != nil {
		tree, err := rec.buf.toTree(delimiter, u, decoders)
		if err != nil {
			return err
		}
		rec.tree = tree
	}

	return nil
}
