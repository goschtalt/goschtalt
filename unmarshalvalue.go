// SPDX-FileCopyrightText: 2022-2023 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"fmt"

	"github.com/goschtalt/goschtalt/internal/print"
)

// UnmarshalValueOption options are options shared between UnmarshalOption and
// ValueOption interfaces.
type UnmarshalValueOption interface {
	fmt.Stringer

	UnmarshalOption
	ValueOption
}

// TagName defines which tag goschtalt honors when it unmarshals to/from
// structures.  The name string defines the new tag name to read.  If an empty
// string is passed in then the module default will be used.
//
// # Default
//
// "goschtalt"
func TagName(name string) UnmarshalValueOption {
	return tagNameOption(name)
}

type tagNameOption string

func (val tagNameOption) unmarshalApply(opts *unmarshalOptions) error {
	tag := defaultTag
	if len(string(val)) > 0 {
		tag = string(val)
	}
	opts.decoder.TagName = tag
	return nil
}

func (val tagNameOption) valueApply(opts *valueOptions) error {
	tag := defaultTag
	if len(string(val)) > 0 {
		tag = string(val)
	}
	opts.tagName = tag
	return nil
}

func (val tagNameOption) String() string {
	return print.P("TagName", print.String(string(val)), print.SubOpt())
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
func Keymap(m map[string]string) UnmarshalValueOption {
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
func KeymapMapper(mapper Mapper) UnmarshalValueOption {
	return &keymapOption{
		text: print.P("KeymapMapper", print.Obj(mapper), print.SubOpt()),
		m:    mapper,
	}
}

type keymapOption struct {
	text string
	m    Mapper
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

// KeymapReport takes an object that implements a KeymapReporter interface and
// adds it to the existing chain of reporters.  A KeymapReporter provides a way
// to report on how the keys were actually mapped.
func KeymapReport(r KeymapReporter) UnmarshalValueOption {
	return &keymapReportOption{
		r: r,
	}
}

type keymapReportOption struct {
	r KeymapReporter
}

func (k keymapReportOption) unmarshalApply(opts *unmarshalOptions) error {
	if k.r != nil {
		opts.reporters = append(opts.reporters, k.r)
	}
	return nil
}

func (k keymapReportOption) valueApply(opts *valueOptions) error {
	if k.r != nil {
		opts.reporters = append(opts.reporters, k.r)
	}
	return nil
}

func (k keymapReportOption) String() string {
	return print.P("KeymapReporter", print.Obj(k.r), print.SubOpt())
}
