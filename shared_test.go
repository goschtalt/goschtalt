// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/goschtalt/goschtalt/pkg/decoder"
	"github.com/goschtalt/goschtalt/pkg/encoder"
	"github.com/goschtalt/goschtalt/pkg/meta"
)

// Test Decoder ////////////////////////////////////////////////////////////////

var _ decoder.Decoder = (*testDecoder)(nil)

type testDecoder struct {
	extensions []string
}

func (t *testDecoder) Decode(ctx decoder.Context, b []byte, m *meta.Object) error {
	var data map[string]any

	if len(b) == 0 {
		*m = meta.Object{}
		return nil
	}

	dec := json.NewDecoder(bytes.NewBuffer(b))
	dec.UseNumber()
	if err := dec.Decode(&data); err != nil {
		return err
	}

	tmp := meta.ObjectFromRaw(data)
	tmp = addOrigin(tmp, &meta.Origin{File: ctx.Filename, Line: 1, Col: 123})
	*m = tmp
	return nil
}

func (t *testDecoder) Extensions() []string {
	return t.extensions
}

func decode(file, s string) meta.Object {
	var data any
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		panic(err)
	}
	obj := meta.ObjectFromRaw(data)
	obj = addOrigin(obj, &meta.Origin{File: file, Line: 1, Col: 123})

	return obj
}

// Test Encoder ////////////////////////////////////////////////////////////////

var _ encoder.Encoder = (*testEncoder)(nil)

type testEncoder struct {
	extensions []string
}

func (t *testEncoder) EncodeExtended(m meta.Object) ([]byte, error) {
	if m.Value == "cause error" {
		return nil, fmt.Errorf("random encoding error")
	}
	return json.Marshal(m)
}

func (t *testEncoder) Encode(v any) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("random encoding error")
	}
	return json.Marshal(v)
}

func (t *testEncoder) Extensions() []string {
	return t.extensions
}

// Test Utilities //////////////////////////////////////////////////////////////

func addOrigin(obj meta.Object, origin *meta.Origin) meta.Object {
	obj.Origins = append(obj.Origins, *origin)
	origin.Line++ // Not accurate, but interesting.

	switch obj.Kind() {
	case meta.Array:
		array := make([]meta.Object, len(obj.Array))
		for i, val := range obj.Array {
			array[i] = addOrigin(val, origin)
		}
		obj.Array = array
	case meta.Map:
		m := make(map[string]meta.Object)

		for key, val := range obj.Map {
			m[key] = addOrigin(val, origin)
		}
		obj.Map = m
	}

	return obj
}

// Test UnmarshalValueOption that lets us easily inject errors.

func testSetResult(v any) UnmarshalValueOption {
	return &testSetResultOption{val: v}
}

func testSetError(e []error) UnmarshalValueOption {
	return &testSetResultOption{err: e}
}

type testSetResultOption struct {
	val any
	i   int
	err []error
}

func (t *testSetResultOption) retErr() error {
	if len(t.err) == 0 || t.i > len(t.err) {
		return nil
	}
	err := t.err[t.i]
	t.i++
	return err
}

func (t *testSetResultOption) unmarshalApply(opts *unmarshalOptions) error {
	if t.val != nil {
		opts.decoder.Result = t.val
	}

	return t.retErr()
}

func (t *testSetResultOption) valueApply(opts *valueOptions) error {
	if t.val != nil {
		opts.decoder.Result = t.val
	}

	return t.retErr()
}

func (t testSetResultOption) String() string {
	return fmt.Sprintf("testSetResultOption{ val: %v, err: %v }", t.val, t.err)
}
