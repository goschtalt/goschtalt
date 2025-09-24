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
