// SPDX-FileCopyrightText: 2022-2023 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"fmt"

	"github.com/goschtalt/goschtalt/internal/print"
	"github.com/mitchellh/mapstructure"
)

// UnmarshalValueOption options are options shared between UnmarshalOption and
// ValueOption interfaces.
type UnmarshalValueOption interface {
	fmt.Stringer

	UnmarshalOption
	ValueOption
}

// DecodeHook ([mapstructure.DecoderConfig.DecodeHook]) is a [mapstructure.DecoderConfig]
// field that defines how goschtalt unmarshals to/from structures.
//
// # Default
//
// No hooks are defined.
func DecodeHook(hook mapstructure.DecodeHookFunc) UnmarshalValueOption {
	return &decodeHookOption{fn: hook}
}

type decodeHookOption struct {
	fn mapstructure.DecodeHookFunc
}

func (d decodeHookOption) unmarshalApply(opts *unmarshalOptions) error {
	opts.decoder.DecodeHook = d.fn
	return nil
}

func (d decodeHookOption) valueApply(opts *valueOptions) error {
	opts.decoder.DecodeHook = d.fn
	return nil
}

func (d decodeHookOption) String() string {
	return print.P("DecodeHook", print.Fn(d.fn), print.SubOpt())
}

// ErrorUnused ([mapstructure.DecoderConfig.ErrorUnused]) is a [mapstructure.DecoderConfig]
// field that defines how goschtalt unmarshals to/from structures.
//
// The unused bool value is optional & assumed to be `true` if omitted.  The
// first specified value is used if provided.  A value of `false` disables the
// option.
//
// # Default
//
// ErrorUnused is set to false.
func ErrorUnused(unused ...bool) UnmarshalValueOption {
	unused = append(unused, true)
	return errorUnusedOption(unused[0])
}

type errorUnusedOption bool

func (val errorUnusedOption) unmarshalApply(opts *unmarshalOptions) error {
	opts.decoder.ErrorUnused = bool(val)
	return nil
}

func (val errorUnusedOption) valueApply(opts *valueOptions) error {
	opts.decoder.ErrorUnused = bool(val)
	return nil
}

func (val errorUnusedOption) String() string {
	return print.P("ErrorUnused", print.BoolSilentTrue(bool(val)), print.SubOpt())
}

// ErrorUnset ([mapstructure.DecoderConfig.ErrorUnset]) is a [mapstructure.DecoderConfig]
// field that defines how goschtalt unmarshals to/from structures.
//
// The unset bool value is optional & assumed to be `true` if omitted.  The
// first specified value is used if provided.  A value of `false` disables the
// option.
//
// # Default
//
// ErrorUnset is set to false.
func ErrorUnset(unset ...bool) UnmarshalValueOption {
	unset = append(unset, true)
	return errorUnsetOption(unset[0])
}

type errorUnsetOption bool

func (val errorUnsetOption) unmarshalApply(opts *unmarshalOptions) error {
	opts.decoder.ErrorUnset = bool(val)
	return nil
}

func (val errorUnsetOption) valueApply(opts *valueOptions) error {
	opts.decoder.ErrorUnset = bool(val)
	return nil
}

func (val errorUnsetOption) String() string {
	return print.P("ErrorUnset", print.BoolSilentTrue(bool(val)), print.SubOpt())
}

// WeaklyTypedInput ([mapstructure.DecoderConfig.WeaklyTypedInput]) is a [mapstructure.DecoderConfig]
// field that defines how goschtalt unmarshals to/from structures.
//
// The weak bool value is optional & assumed to be `true` if omitted.  The
// first specified value is used if provided.  A value of `false` disables the
// option.
//
// # Default
//
// WeaklyTypedInput is set to false.
func WeaklyTypedInput(weak ...bool) UnmarshalValueOption {
	weak = append(weak, true)
	return weaklyTypedInputOption(weak[0])
}

type weaklyTypedInputOption bool

func (val weaklyTypedInputOption) unmarshalApply(opts *unmarshalOptions) error {
	opts.decoder.WeaklyTypedInput = bool(val)
	return nil
}

func (val weaklyTypedInputOption) valueApply(opts *valueOptions) error {
	opts.decoder.WeaklyTypedInput = bool(val)
	return nil
}

func (val weaklyTypedInputOption) String() string {
	return print.P("WeaklyTypedInput", print.BoolSilentTrue(bool(val)), print.SubOpt())
}

// TagName ([mapstructure.DecoderConfig.TagName]) is a [mapstructure.DecoderConfig]
// field that defines how goschtalt unmarshals to/from structures.  The name
// string defines the new tag name to read.
//
// # Default
//
// "mapstructure"
func TagName(name string) UnmarshalValueOption {
	return tagNameOption(name)
}

type tagNameOption string

func (val tagNameOption) unmarshalApply(opts *unmarshalOptions) error {
	opts.decoder.TagName = string(val)
	return nil
}

func (val tagNameOption) valueApply(opts *valueOptions) error {
	opts.decoder.TagName = string(val)
	return nil
}

func (val tagNameOption) String() string {
	return print.P("TagName", print.String(string(val)), print.SubOpt())
}

// IgnoreUntaggedFields ([mapstructure.DecoderConfig.IgnoreUntaggedFields]) is a [mapstructure.DecoderConfig]
// field that defines how goschtalt unmarshals to/from structures.
//
// The ignore bool value is optional & assumed to be `true` if omitted.  The
// first specified value is used if provided.  A value of `false` disables the
// option.
//
// # Default
//
// IgnoreUntaggedFields is set to false.
func IgnoreUntaggedFields(ignore ...bool) UnmarshalValueOption {
	ignore = append(ignore, true)
	return ignoreUntaggedFieldsOption(ignore[0])
}

type ignoreUntaggedFieldsOption bool

func (val ignoreUntaggedFieldsOption) unmarshalApply(opts *unmarshalOptions) error {
	opts.decoder.IgnoreUntaggedFields = bool(val)
	return nil
}

func (val ignoreUntaggedFieldsOption) valueApply(opts *valueOptions) error {
	opts.decoder.IgnoreUntaggedFields = bool(val)
	return nil
}

func (val ignoreUntaggedFieldsOption) String() string {
	return print.P("IgnoreUntaggedFields", print.BoolSilentTrue(bool(val)), print.SubOpt())
}

// ZeroFields ([mapstructure.DecoderConfig.ZeroFields]) is a [mapstructure.DecoderConfig]
// field that defines how goschtalt unmarshals to/from structures.
//
// The zero bool value is optional & assumed to be `true` if omitted.  The
// first specified value is used if provided.  A value of `false` disables the
// option.
//
// # Default
//
// ZeroFields is set to false.
func ZeroFields(zero ...bool) UnmarshalValueOption {
	zero = append(zero, true)
	return zeroFieldsOption(zero[0])
}

type zeroFieldsOption bool

func (z zeroFieldsOption) unmarshalApply(opts *unmarshalOptions) error {
	opts.decoder.ZeroFields = bool(z)
	return nil
}

func (z zeroFieldsOption) valueApply(opts *valueOptions) error {
	opts.decoder.ZeroFields = bool(z)
	return nil
}

func (z zeroFieldsOption) String() string {
	return print.P("ZeroFields", print.BoolSilentTrue(bool(z)), print.SubOpt())
}

// Exactly allows setting nearly all the [mapstructure.DecoderConfig] values to
// whatever value is desired at once.  A few fields aren't available ([mapstructure.DecoderConfig.Metadata],
// [mapstructure.DecoderConfig.Squash], [mapstructure.DecoderConfig.Result]) but
// the rest are honored.
//
// This option will mainly be useful in a scope where the code has no idea what
// options have been set & needs something very specific.
func Exactly(this mapstructure.DecoderConfig) UnmarshalValueOption {
	return &exactlyOption{dc: this}
}

type exactlyOption struct {
	dc mapstructure.DecoderConfig
}

func (exact exactlyOption) decoderApply(m *mapstructure.DecoderConfig) error {
	m.DecodeHook = exact.dc.DecodeHook
	m.ErrorUnused = exact.dc.ErrorUnused
	m.ErrorUnset = exact.dc.ErrorUnset
	m.ZeroFields = exact.dc.ZeroFields
	m.WeaklyTypedInput = exact.dc.WeaklyTypedInput
	// Squash ... I don't think we can use it
	// Metadata isn't supported
	// Result is needed by goschtalt
	m.TagName = exact.dc.TagName
	m.IgnoreUntaggedFields = exact.dc.IgnoreUntaggedFields
	m.MatchName = exact.dc.MatchName
	return nil
}

func (exact exactlyOption) unmarshalApply(opts *unmarshalOptions) error {
	return exact.decoderApply(&opts.decoder)
}

func (exact exactlyOption) valueApply(opts *valueOptions) error {
	return exact.decoderApply(&opts.decoder)
}

func (exact exactlyOption) String() string {
	return print.P("Exactly",
		print.Fn(exact.dc.DecodeHook, "DecodeHook"),
		print.Bool(exact.dc.ErrorUnused, "ErrorUnused"),
		print.Bool(exact.dc.ErrorUnset, "ErrorUnset"),
		print.Bool(exact.dc.ZeroFields, "ZeroFields"),
		print.Bool(exact.dc.WeaklyTypedInput, "WeaklyTypedInput"),
		print.String(exact.dc.TagName, "TagName"),
		print.Bool(exact.dc.IgnoreUntaggedFields, "IgnoreUntaggedFields"),
		print.Fn(exact.dc.MatchName, "MatchName"),
		print.SubOpt())
}

// Mapper takes a golang structure field string and outputs a goschtalt
// configuration tree name string that is one of the following:
//   - "" indicating this mapper was unable to perform the remapping, continue
//     calling mappers in the chain
//   - "-"  indicating this value should be dropped entirely
//   - anything else indicates the new full name
type Mapper func(s string) string

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
		m: func(s string) string {
			if val, found := m[s]; found {
				return val
			}
			return ""
		},
	}
}

// KeymapFn takes a Mapper function and adds it to the existing chain of
// mappers, in the front of the list.
//
// This allows for multiple mappers to be specified instead of requiring a
// single mapper with full knowledge of how to map everything. This makes it
// easy to add logic to remap full keys without needing to re-implement the
// underlying converters.
func KeymapFn(mapper Mapper) UnmarshalValueOption {
	return &keymapOption{
		text: print.P("KeymapFn", print.Fn(mapper), print.SubOpt()),
		m:    mapper,
	}
}

type keymapOption struct {
	text string
	m    Mapper
}

func (k keymapOption) unmarshalApply(opts *unmarshalOptions) error {
	if k.m != nil {
		opts.mappers = append([]Mapper{k.m}, opts.mappers...)
	}
	return nil
}

func (k keymapOption) valueApply(opts *valueOptions) error {
	if k.m != nil {
		opts.mappers = append([]Mapper{k.m}, opts.mappers...)
	}
	return nil
}

func (k keymapOption) String() string {
	return k.text
}
