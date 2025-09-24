// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package encoding

import (
	"io"
)

// Encoder is an interface that defines a method for encoding an Encodeable
// object to a specific format. The method takes an io.Writer to write the
// encoded output and an Encodeable object to encode. It returns an error if
// the encoding fails.
type Encoder interface {
	Encode(io.Writer, Encodeable) error
}

// Encodeable is an interface that defines the methods required for an object
// to be encoded. It includes methods to get the key, value, indentation level,
// header comments, and inline comments of the object. The key is optional and
// can be nil, indicating that the object does not have a key (e.g., an item
// without a key).
type Encodeable interface {
	// Indent returns the indentation level of the object. This is used to
	// determine how much to indent the output when encoding the object.
	// A higher indentation level means more spaces or tabs before the content.
	Indent() int

	// Headers returns the header comments associated with the object, expanded
	// into lines. These comments are typically used to provide context or
	// documentation for the object. The comments are returned as a slice of
	// strings, where each string is a line of the comment.
	Headers() []string

	// Inline returns the inline comments associated with the object, expanded
	// into lines. These comments are typically used to provide additional
	// context or notes about the object. The comments are returned as a slice
	// of strings, where each string is a line of the comment.
	Inline() []string

	// Key returns the key of the object. It can return nil if the object does
	// not have a key (e.g., an array item).
	Key() *string

	// Value returns the value of the object as a string.  It can return nil if
	// the object does not have a value (e.g., an empty item, and instead
	// represents a map/struct with children).
	Value() *string

	// Children returns a slice of Encodeable objects that are children of this
	// object. This is used to represent nested structures, where an object can
	// have other objects as its children. The children can be of any type that
	// implements the Encodeable interface, allowing for flexible and complex
	// data structures to be represented. If the object does not have children,
	// it should return an empty slice.
	Children() Encodeables
}

// Encodeables is a slice of Encodeable objects that implements the sort.Interface
// interface. It is used to sort a collection of Encodeable objects based on
// their keys. The sorting is done in ascending order based on the key values.
// If the keys are equal, the original order is preserved (stable sort).
// If the keys are nil, the objects are considered already sorted by the order
// they were added to the slice.
type Encodeables []Encodeable

func (e Encodeables) Len() int {
	return len(e)
}
func (e Encodeables) Less(i, j int) bool {
	var empty string

	a := e[i].Key()
	b := e[j].Key()
	if a == nil {
		a = &empty
	}
	if b == nil {
		b = &empty
	}

	if *a == *b {
		return i < j
	}

	return *a < *b
}

func (e Encodeables) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}
