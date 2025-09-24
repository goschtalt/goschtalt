// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/goschtalt/goschtalt/internal/encoding"
	"github.com/goschtalt/goschtalt/internal/natsort"
	"github.com/goschtalt/goschtalt/pkg/doc"
	"github.com/goschtalt/goschtalt/pkg/meta"
)

const (
	noticeDeprecated = "!!! DEPRECATED !!!"
)

type unified struct {
	doc    *doc.Object
	key    *string
	value  any
	preset any
	indent int

	array bool

	children map[string]unified
}

var _ encoding.Encodeable = &unified{}

func (u *unified) Indent() int {
	return u.indent
}

// Headers returns the headers for the unified object.
//
// General format:
//
//	!!! DEPRECATED !!!
//	Documentation Line 1
//	Documentation Line 2
//	Documentation Line 3
//	type: <type>
//	default: <default>
//	!!! DEPRECATED !!!
func (u *unified) Headers() []string {
	var rv []string
	var deprecated bool

	if u.doc != nil {
		rv = strings.Split(u.doc.Doc, "\n")

		if u.doc.Deprecated {
			deprecated = true
		}

		typ := u.doc.TypeString()
		typs := strings.Split(typ, "\n")
		typs[0] = "type: " + typs[0]
		if typ == string(doc.TYPE_ROOT) {
			// remove the type line for root types
			typs = typs[1:]
		}
		rv = append(rv, typs...)
	}

	if u.preset != nil {
		line := fmt.Sprintf("default: %s", toString(u.preset))
		rv = append(rv, line)
	}

	if deprecated {
		rv = append([]string{noticeDeprecated}, rv...)
		rv = append(rv, noticeDeprecated)
	}

	return rv
}

// Inline returns the inline comments associated with the unified object.
//
// I'm not sure if this is needed, but the yaml rendering is there & works.
// It could be helpful in the future, so I'm leaving it here for now.
func (u *unified) Inline() []string {
	return nil
}

func (u *unified) Key() *string {
	return u.key
}

// Children returns the children of the unified object in a sorted order.
func (u *unified) Children() encoding.Encodeables {
	if len(u.children) == 0 {
		return nil
	}
	rv := make(encoding.Encodeables, 0, len(u.children))
	for _, key := range u.childKeys() {
		child := u.children[key]
		rv = append(rv, &child)
	}

	return rv
}

func (u *unified) Value() *string {
	if u.value == nil {
		return nil
	}

	s := toString(u.value)
	return &s
}

// toString converts a value of any type into a string representation that
// is suitable for encoding.
func toString(value any) string {
	s := "%v"
	switch v := value.(type) {
	case fmt.Stringer:
		return v.String()
	case string:
		return v
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		s = "%d"
	case float32, float64:
		s = "%f"
	default:
	}
	return fmt.Sprintf(s, value)
}

// childKeys returns the keys of the children in a sorted order.
func (u unified) childKeys() []string {
	keys := make([]string, 0, len(u.children))
	for key := range u.children {
		keys = append(keys, key)
	}

	// Perform a natural sort: numeric if possible, else string
	sort.Slice(keys, func(i, j int) bool {
		return natsort.CompareFloat(keys[i], keys[j])
	})

	return keys
}

func calcUnified(d *doc.Object, compiled *meta.Object, withOrigins bool) (unified, error) {
	return calcUnifiedInternal(nil, -1, d, compiled, withOrigins)
}

// calcUnifiedInternal calculates the unified representation of a document.
// It recursively processes the document and its compiled representation,
// building a unified structure.
func calcUnifiedInternal(name *string, indent int, d *doc.Object, compiled *meta.Object, withOrigins bool) (unified, error) {
	u := unified{
		indent: indent,
		key:    name,
		doc:    d,
	}

	var docArray bool
	arrayLen := 0
	mapLen := 0
	if d != nil {
		if d.Type == doc.TYPE_ARRAY {
			docArray = true
		} else {
			mapLen = len(d.Children)
		}
	}
	if compiled != nil {
		arrayLen = max(arrayLen, len(compiled.Array))
		mapLen = max(mapLen, len(compiled.Map))
	}

	totalLen := arrayLen + mapLen

	// This is a leaf node.
	if totalLen == 0 {
		if compiled != nil {
			u.value = compiled.Value
		}
		return u, nil
	}

	// Check to see if we have a conflicting definition.
	if (docArray && mapLen > 0) || (mapLen > 0 && arrayLen > 0) {
		return u, errors.New("conflicting definitions: array and map cannot coexist in the same object")
	}

	// Handle arrays first.
	if docArray || arrayLen > 0 {
		return calcUnifiedArray(u, indent, arrayLen, compiled, withOrigins)
	}

	// Handle maps/structs next.
	return calcUnifiedMap(u, indent, mapLen, compiled, withOrigins)
}

// calcUnifiedArray calculates the unified representation of an array object and
// all its elements.
func calcUnifiedArray(u unified, indent, arrayLen int, compiled *meta.Object, withOrigins bool) (unified, error) {
	u.array = true
	nextDoc := u.doc
	if nextDoc != nil {
		if tmp, ok := nextDoc.Children[doc.NAME_ARRAY]; ok {
			nextDoc = &tmp
		}
	}

	u.children = make(map[string]unified, arrayLen)
	for i, next := range compiled.Array {
		got, err := calcUnifiedInternal(nil, indent+1, nextDoc, &next, withOrigins)
		// only output the docs for the first entry in the array.
		nextDoc = nil
		if err != nil {
			return unified{}, err
		}
		u.children[strconv.Itoa(i)] = got
	}

	return u, nil
}

// determineNames determines the names of the keys in the unified object, and
// returns a sorted list of names.
func determineNames(u unified, compiled *meta.Object) []string {
	list := make(map[string]struct{})
	if u.doc != nil {
		for key := range u.doc.Children {
			list[key] = struct{}{}
		}
	}
	if compiled != nil {
		for key := range compiled.Map {
			list[key] = struct{}{}
		}
	}

	names := make([]string, 0, len(list))
	for key := range list {
		names = append(names, key)
	}
	sort.Strings(names)
	return names
}

// calcUnifiedMap calculates the unified representation of a map object and
// all its elements.
func calcUnifiedMap(u unified, indent, mapLen int, compiled *meta.Object, withOrigins bool) (unified, error) {
	names := determineNames(u, compiled)
	u.children = make(map[string]unified, mapLen)

	for _, key := range names {
		switch key {
		case doc.NAME_ARRAY:
			return unified{}, errors.New("array key cannot be used in a map object")
		case doc.NAME_EMBEDDED:
			return unified{}, errors.New("embedded key cannot be used in a map object")
		case doc.NAME_MAP_KEY, doc.NAME_MAP_VALUE:
			// Skip these and handle them later since they're a pair.
		default:
			nextDoc := u.doc
			if nextDoc != nil {
				if tmp, ok := nextDoc.Children[key]; ok {
					nextDoc = &tmp
				} else {
					nextDoc = nil // no doc for this key
				}
			}

			nextCompiled := compiled
			if nextCompiled != nil {
				if val, ok := nextCompiled.Map[key]; ok {
					nextCompiled = &val
				} else {
					nextCompiled = nil // no compiled for this key
				}
			}

			got, err := calcUnifiedInternal(&key, indent+1, nextDoc, nextCompiled, withOrigins)
			if err != nil {
				return unified{}, err
			}
			u.children[key] = got
		}
	}

	return u, nil
}
