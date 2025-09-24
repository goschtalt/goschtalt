// SPDX-FileCopyrightText: 2025 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package doc

import (
	"encoding/json"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// These constants are used to represent special names for objects in the model,
// when other names can't be used.
const (
	NAME_ARRAY     = "<array>"
	NAME_MAP_KEY   = "<key>"
	NAME_MAP_VALUE = "<value>"
	NAME_EMBEDDED  = "<embedded>"
	NAME_ROOT      = ""
)

// LeafType represents the type of a leaf node in the object model.
type LeafType string

// These constants represent the various leaf types that can be used in the
// object model.
const (
	TYPE_ROOT      LeafType = "<root>"
	TYPE_ARRAY     LeafType = "<array>"
	TYPE_MAP       LeafType = "<map>"
	TYPE_STRUCT    LeafType = "<struct>"
	TYPE_BOOL      LeafType = "<bool>"
	TYPE_STRING    LeafType = "<string>"
	TYPE_INT       LeafType = "<int>"
	TYPE_INT8      LeafType = "<int8>"
	TYPE_INT16     LeafType = "<int16>"
	TYPE_INT32     LeafType = "<int32>"
	TYPE_INT64     LeafType = "<int64>"
	TYPE_UINT      LeafType = "<uint>"
	TYPE_UINT8     LeafType = "<uint8>"
	TYPE_UINT16    LeafType = "<uint16>"
	TYPE_UINT32    LeafType = "<uint32>"
	TYPE_UINT64    LeafType = "<uint64>"
	TYPE_UINTPTR   LeafType = "<uintptr>"
	TYPE_FLOAT32   LeafType = "<float32>"
	TYPE_FLOAT64   LeafType = "<float64>"
	TYPE_COMPEX64  LeafType = "<complex64>"
	TYPE_COMPEX128 LeafType = "<complex128>"
)

// Object represents a structured object in the model, which can contain
// nested objects, each with its own name, documentation, type, and optional
// properties such as being deprecated or optional. It can also contain child
// objects, allowing for a hierarchical structure.
type Object struct {
	// Name is the name of the object.
	Name string

	// Doc is the comment for the object.
	Doc string

	// Tag is the struct tag for the object.
	Tag string

	// Deprecated indicates if the object is deprecated.
	Deprecated bool

	// Optional indicates if the object is optional.
	Optional bool

	// Type is the type of the object.
	Type LeafType

	// Children is a map of child objects.
	Children map[string]Object
}

func (o *Object) TypeString() string {
	switch o.Type {
	case TYPE_ARRAY:
		if child, ok := o.Children[NAME_ARRAY]; ok {
			return "array of " + child.TypeString()
		}
	case TYPE_MAP:
		key, hasKey := o.Children[NAME_MAP_KEY]
		value, hasValue := o.Children[NAME_MAP_VALUE]

		if hasKey && hasValue {
			var buf strings.Builder
			fmt.Fprintf(&buf, "map with key %s -> value %s", key.TypeString(), value.TypeString())
			if key.Doc != "" {
				buf.WriteString(mapInfoToString("key", key.TypeString(), key.Doc))
			}
			if value.Doc != "" {
				buf.WriteString(mapInfoToString("value", value.TypeString(), value.Doc))
			}
			return buf.String()
		}
	}

	return string(o.Type)
}

func mapInfoToString(typ, typString, docs string) string {
	var rv strings.Builder

	prefix := fmt.Sprintf("  %s(%s) ", typ, typString)
	rv.WriteString("\n")
	rv.WriteString(prefix)

	lines := strings.Split(docs, "\n")
	rv.WriteString(lines[0])
	for _, line := range lines[1:] {
		rv.WriteString("\n")
		rv.WriteString(strings.Repeat(" ", len(prefix)))
		rv.WriteString(line)
	}

	return rv.String()
}

// Merge combines the trees of two Object instances, but not the attributes of
// the objects themselves.  If there are conflicts in specific objects, an error is returned.
func (o *Object) Merge(other Object) (Object, error) {
	rv := o.Clone()

	if o.Name != "" && o.Name != other.Name {
		return Object{}, fmt.Errorf("cannot merge because the names differ: '%s' != '%s'", o.Name, other.Name)
	}
	if o.Type != "" && o.Type != other.Type {
		return Object{}, fmt.Errorf("cannot merge because the types differ: '%s' != '%s'", o.Type, other.Type)
	}
	if o.Tag != "" && o.Tag != other.Tag {
		return Object{}, fmt.Errorf("cannot merge because the tags differ: '%s' != '%s'", o.Tag, other.Tag)
	}
	if o.Deprecated && o.Deprecated != other.Deprecated {
		return Object{}, fmt.Errorf("cannot merge because the deprecated flags differ: '%t' != '%t'", o.Deprecated, other.Deprecated)
	}
	if o.Optional && o.Optional != other.Optional {
		return Object{}, fmt.Errorf("cannot merge because the optional flags differ: '%t' != '%t'", o.Optional, other.Optional)
	}
	if o.Doc != "" && o.Doc != other.Doc {
		return Object{}, fmt.Errorf("cannot merge because the documentation differs: '%s' != '%s'", o.Doc, other.Doc)
	}

	// Merge children
	for k, v := range other.Children {
		if rv.Children == nil {
			rv.Children = make(map[string]Object, len(other.Children))
		}
		if existing, ok := rv.Children[k]; ok {
			// If the child exists, we need to merge it
			mergedChild, err := existing.Merge(v)
			if err != nil {
				return Object{}, fmt.Errorf("error merging child '%s': %w", k, err)
			}
			// Replace the existing child with the merged one
			rv.Children[k] = mergedChild
		} else {
			// If the child doesn't exist, add it
			rv.Children[k] = v.Clone()
		}
	}

	return rv, nil
}

func (o *Object) ShallowCopy() Object {
	// Create a shallow copy of the Object instance, which copies the
	// top-level fields but does not clone the children.
	return Object{
		Name:       o.Name,
		Doc:        o.Doc,
		Tag:        o.Tag,
		Deprecated: o.Deprecated,
		Optional:   o.Optional,
		Type:       o.Type,
	}
}

// Clone creates a deep copy of the Object instance, including all children.
func (o *Object) Clone() Object {
	clone := Object{
		Name:       o.Name,
		Doc:        o.Doc,
		Tag:        o.Tag,
		Deprecated: o.Deprecated,
		Optional:   o.Optional,
		Type:       o.Type,
	}

	if o.Children != nil {
		clone.Children = make(map[string]Object, len(o.Children))
		// Deep copy the children
		for k, v := range o.Children {
			clone.Children[k] = v.Clone()
		}
	}

	return clone
}

// Equal checks if two Object instances are equal.
func (o *Object) Equal(other Object) bool {
	return o.equal(other, true)
}

func (o *Object) equal(other Object, children bool) bool {
	if o.Doc != other.Doc ||
		o.Tag != other.Tag ||
		o.Deprecated != other.Deprecated ||
		o.Optional != other.Optional ||
		o.Type != other.Type {
		return false
	}

	if children {
		if len(o.Children) != len(other.Children) {
			return false
		}

		for k, v := range o.Children {
			if ov, ok := other.Children[k]; !ok {
				if !v.equal(ov, true) {
					return false
				}
			}
		}
	}

	return true
}

// HashVector generates a hash vector for the Object instance, which is a
// string representation that includes the name, type, documentation, tag,
// optional and deprecated flags, and a sorted representation of its children.
// This is used for hashing or comparison purposes.
func (o *Object) HashVector() string {
	var buf strings.Builder

	buf.WriteString(o.Name)
	buf.WriteString(string(o.Type))
	buf.WriteString(o.Doc)
	buf.WriteString(o.Tag)
	buf.WriteString(fmt.Sprintf("%t%t", o.Optional, o.Deprecated))

	keys := slices.Collect(maps.Keys(o.Children))
	sort.Strings(keys)

	for _, k := range keys {
		if v, ok := o.Children[k]; ok {
			buf.WriteString(k)
			buf.WriteString(v.HashVector())
		}
	}

	return buf.String()
}

// Flatten converts the Object and its children into a flat map where the keys
// are the full names of the objects, including their hierarchy, and the values
// are the Object instances themselves. This is useful for easily accessing
// objects by their full path in the hierarchy.
// The keys are in the format "parent.child.name".
func (o *Object) Flatten() map[string]Object {
	m := make(map[string]Object, len(o.Children)+1)
	o.flatten("", m)
	return m
}

func (o *Object) flatten(prefix string, flat map[string]Object) {
	name := o.Name
	if prefix != "" {
		name = prefix + "." + o.Name
	}

	flat[name] = Object{
		Name:       name,
		Doc:        o.Doc,
		Tag:        o.Tag,
		Deprecated: o.Deprecated,
		Optional:   o.Optional,
		Type:       o.Type,
	}

	for _, v := range o.Children {
		v.flatten(name, flat)
	}
}

type validationError struct {
	msg  string
	path []string
}

func (e *validationError) prepend(path string) *validationError {
	e.path = append([]string{path}, e.path...)
	return e
}

func (e *validationError) Error() string {
	path := "<unknown>"
	switch len(e.path) {
	case 0:
	case 1:
		path = e.path[0]
	default:
		path = strings.Join(e.path, ".")
	}

	return fmt.Sprintf("validation error at '%s': %s", path, e.msg)
}

// Validate checks the Object instance for structural integrity and correctness.
func Validate(obj Object) error {
	if err := obj.validate(false); err != nil {
		return err
	}

	if obj.Name != NAME_ROOT {
		return fmt.Errorf("root object name must be \"\", got '%s'", obj.Name)
	}
	if obj.Type != TYPE_ROOT {
		return fmt.Errorf("root object type must be <root>, got '%s'", obj.Type)
	}

	return nil
}

// validate checks the Object instance for structural integrity and correctness.
// If `nameRequired` is false, it allows the object to have an empty name, but
// all children must still have valid names and types.  This allows the root
// object to have an empty name, but all other objects must have a valid name.
func (o *Object) validate(nameRequired bool) *validationError {
	if nameRequired && o.Name == "" {
		return &validationError{
			msg:  "object name cannot be empty",
			path: []string{"<unknown>"},
		}
	}
	if o.Type == "" {
		return &validationError{
			msg:  "object type cannot be empty",
			path: []string{o.Name},
		}
	}

	switch o.Type {
	case TYPE_ARRAY, TYPE_ROOT:
		for _, child := range o.Children {
			if err := child.validate(false); err != nil {
				return err.prepend(o.Name)
			}
		}
	case TYPE_STRUCT, TYPE_MAP:
		for _, child := range o.Children {
			if err := child.validate(true); err != nil {
				return err.prepend(o.Name)
			}
		}
	default:
		if len(o.Children) > 0 {
			return &validationError{
				msg:  fmt.Sprintf("object of type '%s' cannot have children", o.Type),
				path: []string{o.Name},
			}
		}
	}

	return nil
}

// FromJSON parses a JSON string into an Object instance. It validates the
// object after unmarshalling to ensure it adheres to the expected structure.
func FromJSON(jsonStr []byte) (Object, error) {
	var obj Object
	if err := json.Unmarshal(jsonStr, &obj); err != nil {
		return Object{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	if err := Validate(obj); err != nil {
		return Object{}, fmt.Errorf("validation error: %w", err)
	}

	return obj, nil
}

// ToJSON serializes the Object instance into a JSON string. It includes
// indentation for better readability.
func (o *Object) ToJSON() ([]byte, error) {
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object to JSON: %w", err)
	}
	return data, nil
}

// Keys returns a list of all unique keys in the object and its children.
func (o *Object) Keys() []string {
	keys := make(map[string]struct{}, len(o.Children))
	keys[o.Name] = struct{}{}
	o.keys(keys)

	var result []string
	for k := range keys {
		switch k {
		case NAME_ARRAY, NAME_MAP_KEY, NAME_MAP_VALUE, NAME_EMBEDDED, NAME_ROOT:
			// Do nothing
		default:
			result = append(result, k)
		}
	}
	return result
}

// keys is the internal method for collecting unique keys into the provided map.
func (o *Object) keys(m map[string]struct{}) {
	m[o.Name] = struct{}{}
	for k, child := range o.Children {
		m[k] = struct{}{}
		child.keys(m)
	}
}

func createRegexList(keys []string, mapper func(string) string) map[*regexp.Regexp]string {
	replacements := make(map[*regexp.Regexp]string, len(keys))

	for _, item := range keys {
		replacement := mapper(item)
		if replacement == "" || replacement == item {
			// No replacement needed
			continue
		}

		// Match the exact word to prevent partial word replacements.
		r, _ := regexp.Compile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(item)))

		replacements[r] = replacement
	}
	return replacements
}

func (o *Object) Translate(mapper func(string) string) Object {
	replacements := createRegexList(o.Keys(), mapper)

	fn := func(s string) string {
		for re, replacement := range replacements {
			s = re.ReplaceAllString(s, replacement)
		}
		return s
	}

	return o.translate(fn)
}

// translate creates a deep copy of the Object instance, with the documentation
// remapped using the provided Mapper.
func (o *Object) translate(trans func(string) string) Object {
	clone := Object{
		Name:       trans(o.Name),
		Doc:        trans(o.Doc),
		Tag:        o.Tag,
		Deprecated: o.Deprecated,
		Optional:   o.Optional,
		Type:       o.Type,
	}

	if o.Children != nil {
		clone.Children = make(map[string]Object, len(o.Children))
		// Deep copy the children
		for k, v := range o.Children {
			clone.Children[trans(k)] = v.translate(trans)
		}
	}

	return clone
}
