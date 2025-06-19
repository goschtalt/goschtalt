// SPDX-FileCopyrightText: 2022-2023 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/goschtalt/goschtalt/internal/print"
	"github.com/goschtalt/goschtalt/internal/structs"
	"github.com/goschtalt/goschtalt/pkg/meta"
)

// ValueGetter provides the methods needed to get the value.
type ValueGetter interface {
	// Get is called each time the configuration is compiled.  The recordName
	// and an Unmarshaler with the present stage of configuration and expanded
	// variables are provided to assist.  A data structure (object, string, int, etc)
	// or an error is returned.
	Get(recordName string, u Unmarshaler) (any, error)
}

// The ValueGetterFunc type is an adapter to allow the use of ordinary functions
// as ValueGetters. If f is a function with the appropriate signature,
// ValueGetterFunc(f) is a ValueGetter that calls f.
type ValueGetterFunc func(string, Unmarshaler) (any, error)

// Get calls f(rn, u)
func (f ValueGetterFunc) Get(rn string, u Unmarshaler) (any, error) {
	return f(rn, u)
}

var _ ValueGetter = (*ValueGetterFunc)(nil)

// AddValues provides a simple way to set additional configuration values at
// runtime.
//
// To place the configuration at the root use `goschtalt.Root` ([Root]) instead
// of "" for more clarity.
//
// Valid Option Types:
//   - [BufferValueOption]
//   - [GlobalOption]
//   - [ValueOption]
//   - [UnmarshalValueOption]
func AddValue(recordName, key string, val any, opts ...ValueOption) Option {
	return &value{
		text:       print.P("AddValue", print.String(recordName), print.String(key), print.Obj(val), print.LiteralStringers(opts)),
		recordName: recordName,
		key:        key,
		getter: ValueGetterFunc(
			func(_ string, _ Unmarshaler) (any, error) {
				return val, nil
			}),
		opts: opts,
	}
}

// AddValueGetter provides a simple way to set additional configuration values
// at runtime via a function call.  Note that the provided ValueGetter will be
// called each time the configuration is compiled, allowing the value returned
// to change if desired.
//
// To place the configuration at the root use `goschtalt.Root` ([Root]) instead
// of "" for more clarity.
//
// Valid Option Types:
//   - [BufferValueOption]
//   - [GlobalOption]
//   - [ValueOption]
//   - [UnmarshalValueOption]
func AddValueGetter(recordName, key string, getter ValueGetter, opts ...ValueOption) Option {
	return &value{
		text:       print.P("AddValueGetter", print.String(recordName), print.String(key), print.Obj(getter), print.LiteralStringers(opts)),
		recordName: recordName,
		key:        key,
		getter:     getter,
		opts:       opts,
	}
}

// value defines a key and value that is injected into the configuration tree.
type value struct {
	text string

	// The record to use for sorting this configuration.
	recordName string

	// The key to set the value at.
	key string

	// The getter to use to get the value.
	getter ValueGetter

	// Options that configure how to process the Value provided.
	// These options are in addition to any default settings set with
	// AddDefaultValueOptions().
	opts []ValueOption
}

// toTree does the work of converting from a structure of some sort to the
// normalized object tree goschtalt uses.
func (v value) toTree(delimiter string, u Unmarshaler, defaultOpts ...ValueOption) (meta.Object, error) {
	cfg := valueOptions{
		tagName: defaultTag,
	}

	all := append(defaultOpts, v.opts...)
	for _, opt := range all {
		if err := opt.valueApply(&cfg); err != nil {
			return meta.Object{}, err
		}
	}

	data, err := v.getter.Get(v.recordName, u)
	if err != nil {
		return meta.Object{}, err
	}

	if data == nil {
		return meta.Object{}, nil
	}

	// Dereference the pointer if it is one.
	if reflect.TypeOf(data).Kind() == reflect.Ptr {
		data = reflect.ValueOf(data).Elem().Interface()
	}

	if reflect.TypeOf(data).Kind() == reflect.Struct {
		s := structs.New(data)
		s.TagName = cfg.tagName
		data = s.Map()
	}

	tree := meta.ObjectFromRawWithOrigin(data,
		[]meta.Origin{{File: v.recordName}},
		strings.Split(v.key, delimiter)...)

	tree = tree.AlterKeyCase(func(s string) string {
		return cfg.mapper(s)
	})

	tree, err = tree.AdaptToRaw(adapterIterator(cfg.adapters))
	if err != nil {
		return meta.Object{}, err
	}

	if cfg.failOnNonSerializable {
		if err = tree.ErrOnNonSerializable(); err != nil {
			return meta.Object{}, err
		}
	}

	return tree.FilterNonSerializable(), nil
}

func (v value) apply(opts *options) error {
	if len(v.recordName) == 0 {
		return fmt.Errorf("%w: no valid record name provided", ErrInvalidInput)
	}

	r := record{
		name: v.recordName,
		val:  &v,
	}

	for _, opt := range v.opts {
		var info valueOptions

		if err := opt.valueApply(&info); err != nil {
			return err
		}

		if info.isDefault {
			opts.defaults = append(opts.defaults, r)
			return nil
		}
	}

	opts.values = append(opts.values, r)
	return nil
}

func (value) ignoreDefaults() bool {
	return false
}

func (v value) String() string {
	return v.text
}

// -- ValueOption options follow -----------------------------------------------

// ValueOption provides the means to configure options around variable mapping
// as well as if the specific value being added should be a default or a normal
// configuration value.
//
// See also [UnmarshalValueOption] which can be used as ValueOption options.
type ValueOption interface {
	fmt.Stringer

	valueApply(*valueOptions) error
}

type valueOptions struct {
	tagName               string
	mappers               []Mapper
	adapters              []adapter
	reporters             []KeymapReporter
	failOnNonSerializable bool
	isDefault             bool
}

// mapper is a simple helper that does the mapping based on the specified
// options.
func (v valueOptions) mapper(s string) string {
	in := s
	for _, m := range v.mappers {
		if rv := m.Map(s); rv != "" {
			s = rv
		}
	}

	for _, r := range v.reporters {
		r.Report(in, s)
	}
	return s
}

// FailOnNonSerializable specifies that an error should be returned if any
// non-serializable objects (channels, functions, unsafe pointers) are
// encountered in the resulting configuration tree.  Non-serializable objects
// cannot be in the configuration sets that goschtalt works with.
//
// The fail bool value is optional & assumed to be `true` if omitted.  The
// first specified value is used if provided.  A value of `false` disables the
// option.
//
// The default behavior is to ignore and drop any non-serializable objects.
func FailOnNonSerializable(fail ...bool) ValueOption {
	fail = append(fail, true)
	return failOnNonSerializableOption(fail[0])
}

type failOnNonSerializableOption bool

func (e failOnNonSerializableOption) valueApply(opt *valueOptions) error {
	opt.failOnNonSerializable = bool(e)
	return nil
}

func (e failOnNonSerializableOption) String() string {
	return print.P("FailOnNonSerializable", print.BoolSilentTrue(bool(e)), print.SubOpt())
}

// AdapterToCfg provides a method that maps a golang struct object into the
// configuration form.  It assumed that the converter knows best what that is.
//
// If the mapping is not applicable, the ErrNotApplicable error is returned.
// Any other non-nil error fails the operation entirely.
type AdapterToCfg interface {
	To(from reflect.Value) (any, error)
}

// The AdapterToCfgFunc type is an adapter to allow the use of ordinary functions
// as AdapterToCfgs. If f is a function with the appropriate signature,
// AdapterToCfgFunc(f) is a AdapterToCfg that calls f.
type AdapterToCfgFunc func(reflect.Value) (any, error)

// Get calls f(rn, u)
func (f AdapterToCfgFunc) To(from reflect.Value) (any, error) {
	return f(from)
}

var _ AdapterToCfg = (*AdapterToCfgFunc)(nil)

// AdaptToCfg converts a value a golang struct object into the configuration
// form.  It assumed that the converter knows best what the form should be.
//
// If the combination of from and t are unknown or unsupported return
// ErrNotApplicable as the error with a nil value.
//
// If ErrNotApplicable is returned the value returned will be ignored.
//
// The optional label parameter allows you to provide the function name of the
// adapter so it is more clear which adapters are registered.
func AdaptToCfg(adapter AdapterToCfg, label ...string) ValueOption {
	label = append(label, "")
	return &adaptToCfgOption{
		label:   label[0],
		adapter: adapter,
	}
}

type adaptToCfgOption struct {
	label   string
	adapter AdapterToCfg
}

func (a adaptToCfgOption) valueApply(opts *valueOptions) error {
	opts.adapters = append(opts.adapters,
		func(from, to reflect.Value) (any, error) {
			return a.adapter.To(from)
		})
	return nil
}

func (a adaptToCfgOption) String() string {
	labels := make([]string, 0, 1)
	if len(a.label) > 0 {
		labels = append(labels, a.label)
	}
	return print.P("AdaptToCfg", print.Obj(a.adapter, labels...), print.SubOpt())
}
