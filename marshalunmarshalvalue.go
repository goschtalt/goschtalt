// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"fmt"

	"github.com/goschtalt/goschtalt/internal/print"
)

// MarshalUnmarshalValueOption options are options shared between MarshalOption,
// UnmarshalOption and ValueOption interfaces.
type MarshalUnmarshalValueOption interface {
	fmt.Stringer

	MarshalOption
	UnmarshalOption
	ValueOption
}

// Mapper provides a method that can map from a golang structure field name to a
// goschtalt configuration tree name.
type Mapper interface {
	// Map takes a golang structure field string and outputs a goschtalt
	// configuration tree name string that is one of the following:
	//   - "" indicating this mapper was unable to perform the remapping, continue
	//     calling mappers in the chain
	//   - "-"  indicating this value should be dropped entirely
	//   - anything else indicates the new full name
	Map(s string) string
}

type wrapMap struct {
	m map[string]string
}

func (w wrapMap) Map(s string) string {
	if val, found := w.m[s]; found {
		return val
	}
	return s
}

// Keymap takes a map of strings to strings and adds it to the existing
// chain of keymaps. The key of the map is the golang structure field name and
// the value is the goschtalt configuration tree name string. The value of "-"
// means do not convert, and an empty string means call the next in the chain.
//
// For example, the map below converts a structure field "FooBarIP" to "foobar_ip".
//
//	Keymap( map[string]string{
//		"FooBarIP": "foobar_ip",
//	})
func Keymap(m map[string]string) MarshalUnmarshalValueOption {
	return &keymapOption{
		text: print.P("Keymap", print.StringMap(m), print.SubOpt()),
		m: wrapMap{
			m: m,
		},
	}
}

// KeymapMapper takes a Mapper function and adds it to the existing chain of
// mappers, in the front of the list.
//
// This allows for multiple mappers to be specified instead of requiring a
// single mapper with full knowledge of how to map everything. This makes it
// easy to add logic to remap full keys without needing to re-implement the
// underlying converters.
func KeymapMapper(mapper Mapper) MarshalUnmarshalValueOption {
	return &keymapOption{
		text: print.P("KeymapMapper", print.Obj(mapper), print.SubOpt()),
		m:    mapper,
	}
}

type keymapOption struct {
	text string
	m    Mapper
}

func (k keymapOption) marshalApply(opts *marshalOptions) error {
	if k.m != nil {
		opts.mappers = append(opts.mappers, k.m)
	}
	return nil
}

func (k keymapOption) unmarshalApply(opts *unmarshalOptions) error {
	if k.m != nil {
		opts.mappers = append(opts.mappers, k.m)
	}
	return nil
}

func (k keymapOption) valueApply(opts *valueOptions) error {
	if k.m != nil {
		opts.mappers = append(opts.mappers, k.m)
	}
	return nil
}

func (k keymapOption) String() string {
	return k.text
}

// KeymapReporter is the interface that provides a way to report what was mapped
// to what.  This is designed to help make debugging mapping mistakes easier.
//
// The [github.com/goschtalt/goschtalt/pkg/debug/Collect] package implements
// KeymapReporter for easy use.
type KeymapReporter interface {
	// Report is called with the `from` and `to` strings when a key mapping
	// takes place.
	Report(from, to string)
}
