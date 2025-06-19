// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"fmt"
	"path"
	"strings"

	"github.com/goschtalt/goschtalt/internal/print"
	"github.com/goschtalt/goschtalt/pkg/decoder"
	"github.com/goschtalt/goschtalt/pkg/meta"
)

// BufferGetter provides the methods needed to get the buffer of bytes.
type BufferGetter interface {
	// Get is called each time the configuration is compiled.  The recordName
	// and an Unmarshaler with the present stage of configuration and expanded
	// variables are provided to assist.  A slice of bytes or an error is
	// returned.
	Get(recordName string, u Unmarshaler) ([]byte, error)
}

// The BufferGetterFunc type is an adapter to allow the use of ordinary functions
// as BufferGetters. If f is a function with the appropriate signature,
// BufferGetterFunc(f) is a BufferGetter that calls f.
type BufferGetterFunc func(string, Unmarshaler) ([]byte, error)

// Get calls f(rn, u)
func (f BufferGetterFunc) Get(rn string, u Unmarshaler) ([]byte, error) {
	return f(rn, u)
}

var _ BufferGetter = (*BufferGetterFunc)(nil)

// AddBuffer adds a buffer of bytes for inclusion when compiling the configuration.
// The format of the bytes is determined by the extension of the recordName field.
// The recordName field is also used for sorting this configuration value relative
// to other configuration values.
//
// Valid Option Types:
//   - [BufferOption]
//   - [BufferValueOption]
//   - [GlobalOption]
func AddBuffer(recordName string, in []byte, opts ...BufferOption) Option {
	return &buffer{
		text:       print.P("AddBuffer", print.String(recordName), print.Bytes(in), print.LiteralStringers(opts)),
		recordName: recordName,
		getter: BufferGetterFunc(
			func(_ string, _ Unmarshaler) ([]byte, error) {
				return in, nil
			}),
		opts: opts,
	}
}

// AddBufferGetter adds a function that is called during compile time of the
// configuration.  The recordName of this record is passed into the getter,
// as well as an Unmarshaler that represents the existing state of the merged
// configuration prior to adding the buffer that
// results in the call to getter.
//
// The format of the bytes is determined by the extension of the recordName field.
// The recordName field is also used for sorting this configuration value relative
// to other configuration values.
//
// Valid Option Types:
//   - [BufferOption]
//   - [BufferValueOption]
//   - [GlobalOption]
func AddBufferGetter(recordName string, getter BufferGetter, opts ...BufferOption) Option {
	return &buffer{
		text:       print.P("AddBufferGetter", print.String(recordName), print.Obj(getter), print.LiteralStringers(opts)),
		recordName: recordName,
		opts:       opts,
		getter:     getter,
	}
}

type buffer struct {
	// The text to use when String() is called.
	text string

	// The record name.
	recordName string

	// The getter to use to get the value.
	getter BufferGetter

	// Options that configure how this buffer is treated and processed.
	// These options are in addition to any default settings set with
	// AddDefaultValueOptions().
	opts []BufferOption
}

func (b buffer) apply(opts *options) error {
	if len(b.recordName) == 0 {
		return fmt.Errorf("%w: a recordName with length > 0 must be specified.", ErrInvalidInput) // nolint:staticcheck
	}

	if b.getter == nil {
		return fmt.Errorf("%w: a non-nil BufferGetter must be specified.", ErrInvalidInput) // nolint:staticcheck
	}

	r := record{
		name: b.recordName,
		buf:  &b,
	}

	for _, opt := range b.opts {
		var info bufferOptions
		if err := opt.bufferApply(&info); err != nil {
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

func (buffer) ignoreDefaults() bool {
	return false
}

func (b buffer) String() string {
	return b.text
}

// toTree converts an buffer into a meta.Object tree.  This will happen
// during the compilation stage.
func (b *buffer) toTree(delimiter string, u Unmarshaler, decoders *codecRegistry[decoder.Decoder]) (meta.Object, error) {
	data, err := b.getter.Get(b.recordName, u)
	if err != nil {
		return meta.Object{}, err
	}

	ext := strings.TrimPrefix(path.Ext(b.recordName), ".")

	dec, err := decoders.find(ext)
	if err != nil {
		return meta.Object{}, err
	}

	ctx := decoder.Context{
		Filename:  b.recordName,
		Delimiter: delimiter,
	}

	var tree meta.Object
	err = dec.Decode(ctx, data, &tree)
	if err != nil {
		err = fmt.Errorf("decoder error for extension '%s' processing buffer '%s' %w %v",
			ext, b.recordName, ErrDecoding, err) //nolint:errorlint

		return meta.Object{}, err
	}

	return tree, nil
}

// -- BufferOption options follow ----------------------------------------------

// BufferOption provides the means to configure options for handling of the
// buffer configuration values.
type BufferOption interface {
	fmt.Stringer

	bufferApply(*bufferOptions) error
}

type bufferOptions struct {
	isDefault bool
}
