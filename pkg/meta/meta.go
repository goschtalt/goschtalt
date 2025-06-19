// SPDX-FileCopyrightText: 2022-2023 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package meta

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const redactedText = "REDACTED"

const (
	cmdSecret  = "secret"
	cmdReplace = "replace"
	cmdKeep    = "keep"
	cmdFail    = "fail"
	cmdAppend  = "append"
	cmdPrepend = "prepend"
	cmdSplice  = "splice"
	cmdClear   = "clear"
)

var (
	ErrConflict         = errors.New("a conflict has been detected")
	ErrInvalidCommand   = errors.New("invalid command")
	ErrNotFound         = errors.New("not found")
	ErrArrayOutOfBounds = errors.New("array index is out of bounds")
	ErrInvalidIndex     = errors.New("invalid index")
	ErrRecursionTooDeep = errors.New("recursion too deep")
	ErrNonSerializable  = errors.New("non-serializeable objects encountered")
)

// Origin provides details about an origin of a parameter.
type Origin struct {
	File string // Filename where the value originated.
	Line int    // Line number where the value originated.
	Col  int    // Column where the value originated.
}

// String returns a useful representation for the origin.
func (o Origin) String() string {
	file := "unknown"
	line := ""
	col := ""

	if len(o.File) > 0 {
		file = o.File
	}
	if o.Line > 0 || o.Col > 0 {
		line = fmt.Sprintf(":%d", o.Line)
	}
	if o.Col > 0 {
		col = fmt.Sprintf("[%d]", o.Col)
	}

	return fmt.Sprintf("%s%s%s", file, line, col)
}

const (
	Array = iota + 1
	Map
	Value
)

// Object represents either a map, of objects, and array of objects or a specific
// configuration value and the origins of how this Object came into existence.
//
// To build an Object tree, either add to the Array, Map or Value fields.  Don't
// add to them all, as only one will be used.  The order is Array > Map > Value.
// So for example, if you add to the Array field and the Value field, the Value
// field will always be ignored.
type Object struct {
	Origins []Origin          // The list of origins that influenced this Object.
	Array   []Object          // The array of Objects (if a map).
	Map     map[string]Object // The map of Objects (if a map).
	Value   any               // The value of the configuration parameter (if a value).
	secret  bool              // If the value is secret.
}

// Kind provides the specific kind of Object this is.  Array, Map or Value.  If
// it unclear exactly which, Value will be returned.
func (o Object) Kind() int {
	if 0 < len(o.Array) {
		return Array
	}
	if 0 < len(o.Map) {
		return Map
	}
	return Value
}

// OriginString provides the string for all origins for this Object.
func (obj Object) OriginString() string {
	list := make([]string, len(obj.Origins))
	for i, v := range obj.Origins {
		list[i] = v.String()
	}

	return strings.Join(list, ", ")
}

// Fetch looks up the specific asks in the tree (map keys or array indexes) and
// returns the found object or provides a contextual error.  The separater is
// used to provide error context.
func (obj Object) Fetch(asks []string, separater string) (Object, error) {
	return obj.fetch(asks, asks, separater)
}

// getPath is an internal helper that determines the path in use.  Mainly used
// for determining the portion of the original string where the error was
// encountered at.
func getPath(asks, path []string, separater string) string {
	// Trim off the same number of elements that are left in the asks from the
	// path before joining them and returning them.
	path = path[:len(path)-len(asks)]
	return strings.Join(path, separater)
}

// fetch is the internal helper function that actually finds and returns the
// Object of interest.
func (obj Object) fetch(asks, path []string, separater string) (Object, error) {
	if len(asks) == 0 {
		return obj, nil
	}

	switch obj.Kind() {
	case Map:
		key := asks[0]
		next, found := obj.Map[key]
		if found {
			return next.fetch(asks[1:], path, separater)
		}
	case Array:
		idx, err := strconv.Atoi(asks[0])
		if err != nil {
			return Object{}, err
		}
		if 0 <= idx && idx < len(obj.Array) {
			return obj.Array[idx].fetch(asks[1:], path, separater)
		} else {
			return Object{},
				fmt.Errorf("with array len of %d and '%s' %w",
					len(obj.Array),
					getPath(asks[1:], path, separater),
					ErrArrayOutOfBounds)
		}
	}

	return Object{}, fmt.Errorf("with '%s' %w",
		getPath(asks[1:], path, separater), ErrNotFound)
}

// ToRaw converts an Object tree into a native go tree (with no secret or origin history.
func (obj Object) ToRaw() any {
	switch obj.Kind() {
	case Array:
		rv := make([]any, len(obj.Array))
		for i, val := range obj.Array {
			rv[i] = val.ToRaw()
		}
		return rv
	case Map:
		rv := make(map[string]any)

		for key, val := range obj.Map {
			rv[key] = val.ToRaw()
		}
		return rv
	}
	return obj.Value
}

// ObjectFromRaw converts a native go tree into the equivalent Object tree structure.
func ObjectFromRaw(in any, at ...string) (obj Object) {
	return ObjectFromRawWithOrigin(in, nil, at...)
}

// ObjectFromRawWithOrigin converts a native go tree into the equivalent Object
// tree structure with the specified origin applied across the tree.
func ObjectFromRawWithOrigin(in any, where []Origin, at ...string) (obj Object) {
	obj.Origins = []Origin{}
	if len(where) != 0 {
		obj.Origins = where
	}

	if len(at) > 0 && len(at[0]) > 0 {
		obj.Map = make(map[string]Object)
		obj.Map[at[0]] = ObjectFromRawWithOrigin(in, where, at[1:]...)
		return obj
	}

	switch in := in.(type) {
	case []any:
		obj.Array = make([]Object, len(in))
		for i, val := range in {
			obj.Array[i] = ObjectFromRawWithOrigin(val, where)
		}
	case map[string]any:
		obj.Map = make(map[string]Object)
		for key, val := range in {
			obj.Map[key] = ObjectFromRawWithOrigin(val, where)
		}
	default:
		obj.Value = in
	}

	return obj
}

// Add adds an object to the tree assuming the key needs to be split and the tree
// may need to be created or added to depending on what is existing.  The returned
// object is the new tree.
//
// Note that Add will not create arrays, but only maps and values.  It will update
// arrays (either replacing an item, or extending the array by exactly 1 if the
// index is 1 larger than the array).  It is suggested to use Add() to build
// an object tree and then use the ConvertMapsToArrays() method to do the
// conversion.  This helps eliminate problems due to sparcely populated arrays
// not being allowed.
func (obj Object) Add(keyDelimiter, key string, val any, origin ...Origin) (Object, error) {
	splitKey := strings.Split(key, keyDelimiter)
	return obj.add(splitKey, val, origin...)
}

// add is the internal helper that is recursively called to add to the tree.
func (obj Object) add(keys []string, val any, origin ...Origin) (Object, error) {
	kind := obj.Kind()

	if len(origin) == 0 {
		origin = []Origin{{}}
	}

	if len(keys) == 0 {
		return Object{
			Origins: origin,
			Value:   val,
		}, nil
	}

	key := keys[0]
	if kind == Array {
		idx, err := strconv.Atoi(key)
		if err != nil {
			return Object{}, fmt.Errorf("%w: index: '%s' %v", ErrInvalidIndex, key, err) //nolint:errorlint
		}
		if idx < 0 || len(obj.Array) < idx {
			return Object{}, fmt.Errorf("%w: index: '%s' must be %d", ErrArrayOutOfBounds, key, len(obj.Array))
		}

		sub := Object{
			Origins: origin,
		}
		if idx < len(obj.Array) {
			sub = obj.Array[idx]
		}

		next, err := sub.add(keys[1:], val, origin...)
		if err != nil {
			return Object{}, err
		}
		if idx == len(obj.Array) {
			obj.Array = append(obj.Array, next)
		} else {
			obj.Array[idx] = next
		}
		return obj, nil
	}

	// Map
	if obj.Map == nil {
		obj.Map = make(map[string]Object)
	}

	sub, found := obj.Map[key]
	if !found {
		sub = Object{
			Origins: origin,
		}
	}
	next, err := sub.add(keys[1:], val, origin...)
	if err != nil {
		return Object{}, err
	}
	obj.Map[key] = next
	return obj, nil
}

// StringToBestType does a reasonable effort to determine if there is a better
// type being presented.  Either an int64, float64, bool or the original string
// is returned with preference for the type also in that order.
func StringToBestType(s string) any {
	i64, err := strconv.ParseInt(s, 0, 64)
	if err == nil {
		return i64
	}

	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f
	}

	b, err := strconv.ParseBool(s)
	if err == nil {
		return b
	}

	return s
}

// ToRedacted builds a copy of the tree where secrets are redacted.  Secret maps
// or arrays will now show up as values containing the value 'REDACTED'.
func (obj Object) ToRedacted() Object {
	if obj.secret {
		return Object{
			Origins: []Origin{},
			Value:   redactedText,
			secret:  true,
		}
	}

	switch obj.Kind() {
	case Array:
		array := make([]Object, len(obj.Array))
		for i, val := range obj.Array {
			array[i] = val.ToRedacted()
		}
		obj.Array = array
	case Map:
		m := make(map[string]Object)

		for key, val := range obj.Map {
			m[key] = val.ToRedacted()
		}
		obj.Map = m
	}

	return obj
}

// ToExpanded builds a copy of the tree where any matching variables are expanded
// to the final instance.  The max value is used to prevent recursive substitutions
// from never returning.  Instead the process is stopped and an error is returned.
// The resulting tree is returned.
func (obj Object) ToExpanded(max int, origin, start, end string, expander func(string) (string, bool)) (Object, error) {
	var err error

	switch obj.Kind() {
	case Array:
		array := make([]Object, len(obj.Array))
		for i, val := range obj.Array {
			array[i], err = val.ToExpanded(max, origin, start, end, expander)
			if err != nil {
				return Object{}, err
			}
		}
		obj.Array = array
	case Map:
		m := make(map[string]Object)

		for key, val := range obj.Map {
			m[key], err = val.ToExpanded(max, origin, start, end, expander)
			if err != nil {
				return Object{}, err
			}
		}
		obj.Map = m
	case Value:
		switch v := obj.Value.(type) {
		case string:
			// Limit the expansion to the max depth, but not the entire tree,
			// just the value.
			tmp := max
			val, changed, err := expand(&tmp, v, start, end, expander)
			if err != nil {
				return Object{}, err
			}
			origins := obj.Origins
			if changed {
				origins = append(origins, Origin{File: origin})
			}
			return Object{
				Origins: origins,
				Value:   val,
				secret:  obj.secret,
			}, nil
		default:
		}
	}

	return obj, nil
}

// expand performs the expansion of a string based on the starting and ending
// tokens as well as the mapping function & max replacement depth.
func expand(max *int, in, startToken, endToken string, mapper func(string) (string, bool)) (string, bool, error) {
	if *max < 1 {
		return "", false, ErrRecursionTooDeep
	}
	*max--

	start := strings.Index(in, startToken)
	if start == -1 {
		return in, false, nil
	}

	rest := in[start+len(startToken):]
	end := strings.Index(rest, endToken)

	if end == -1 {
		return in, false, nil
	}

	before := in[:start]
	after := rest[end+len(endToken):]

	key := strings.TrimSpace(rest[:end])

	var full string
	var changed bool
	got, found := mapper(key)
	if found {
		changed = true
		full = before + got
	} else {
		full = before + startToken + key + endToken
	}

	last, expanded, err := expand(max, after, startToken, endToken, mapper)
	if err != nil {
		return "", false, err
	}

	if expanded {
		changed = true
	}

	full += last

	if !changed {
		return in, false, nil
	}

	rv, _, err := expand(max, full, startToken, endToken, mapper)
	return rv, true, err
}

// ConvertMapsToArrays walks the object tree and looks for any maps that contain
// only sequential numbers starting with 0.  If one is found, then it assumed to
// be an array and restructured accordingly.
func (obj Object) ConvertMapsToArrays() Object {
	switch obj.Kind() {
	case Value, Array:
		return obj
	}

	// Map
	for key := range obj.Map {
		obj.Map[key] = obj.Map[key].ConvertMapsToArrays()
	}

	// Now check to see if the map should be an array.
	indexes := make([]bool, len(obj.Map))

	for key := range obj.Map {
		idx, err := strconv.Atoi(key)
		if err != nil {
			// Can't be an array, exit.
			return obj
		}
		if idx < 0 || len(indexes) <= idx || indexes[idx] {
			// Can't be an array because the indexes aren't sequential, exit.
			return obj
		}
		indexes[idx] = true
	}

	rv := Object{
		Origins: obj.Origins,
		Array:   make([]Object, len(obj.Map)),
	}

	for i := 0; i < len(obj.Map); i++ {
		rv.Array[i] = obj.Map[strconv.Itoa(i)]
	}

	return rv
}

// AlterKeyCase builds a copy of the tree where the keys for all Objects have
// been converted using the specified conversion function.  If the to function
// returns the value "-" then the key and it's subtree is dropped entirely.
func (obj Object) AlterKeyCase(to func(string) string) Object {
	switch obj.Kind() {
	case Array:
		array := make([]Object, len(obj.Array))
		for i, val := range obj.Array {
			array[i] = val.AlterKeyCase(to)
		}
		obj.Array = array
	case Map:
		m := make(map[string]Object)

		for key, val := range obj.Map {
			target := to(key)
			if target != "-" {
				m[target] = val.AlterKeyCase(to)
			}
		}
		obj.Map = m
	}

	return obj
}

// ResolveCommands builds a copy of the tree where the commands have been
// resolved from the keys.
func (obj Object) ResolveCommands() (Object, error) {
	return obj.resolveCommands(false)
}

// resolveCommands is the internal helper function that does the actual resolution.
func (obj Object) resolveCommands(secret bool) (Object, error) {
	if secret {
		obj.secret = true
	}

	switch obj.Kind() {
	case Array:
		array := make([]Object, len(obj.Array))
		for i, val := range obj.Array {
			v, err := val.resolveCommands(false)
			if err != nil {
				return Object{}, err
			}
			array[i] = v
		}
		obj.Array = array
	case Map:
		m := make(map[string]Object)

		for key, val := range obj.Map {
			cmd, err := getValidCmd(key, val)
			if err != nil {
				return Object{}, err
			}
			tmp, err := val.resolveCommands(cmd.secret)
			if err != nil {
				return Object{}, err
			}
			m[cmd.final] = tmp
		}
		obj.Map = m
	}

	return obj, nil
}

// Merge performs a merge of the new Object tree onto the existing Object tree
// using the default semantics and merge rules found in the key commands.
func (obj Object) Merge(next Object) (Object, error) {
	// The 'clear' command is special in that if it is found at all, it
	// overwrites everything else in the existing tree and exists the merge.
	for k := range next.Map {
		cmd, err := getCmd(k)
		if err != nil {
			return Object{}, err
		}
		if cmd.cmd == cmdClear {
			return Object{Origins: []Origin{}}, nil
		}
	}

	return obj.merge(command{}, next)
}

// merge does the actual merging of the trees.
func (obj Object) merge(cmd command, next Object) (Object, error) {
	switch obj.Kind() {
	case Value:
		return obj.mergeValue(cmd, next)
	case Array:
		return obj.mergeArray(cmd, next)
	}
	return obj.mergeMap(cmd, next)
}

// mergeValue merges two values.  Don't directly call this, call merge() instead.
func (obj Object) mergeValue(cmd command, next Object) (Object, error) {
	rv := obj
	switch cmd.cmd {
	case cmdReplace, "":
		var err error
		rv, err = next.resolveCommands(obj.secret)
		if err != nil {
			return Object{}, err
		}
	case cmdFail:
		return Object{}, fmt.Errorf("%w: merging a value with command 'fail'", ErrConflict)
	case cmdKeep:
	}

	rv.secret = cmd.secret
	return rv, nil
}

// mergeArray merges two array.  Don't directly call this, call merge() instead.
func (obj Object) mergeArray(cmd command, next Object) (Object, error) {
	rv := obj
	next, err := next.resolveCommands(obj.secret)
	if err != nil {
		return Object{}, err
	}
	switch cmd.cmd {
	case cmdAppend, "":
		if obj.secret || next.secret || cmd.secret {
			rv.secret = true
		}
		rv.Origins = append(obj.Origins, next.Origins...)
		rv.Array = append(obj.Array, next.Array...)
	case cmdPrepend:
		if obj.secret || next.secret || cmd.secret {
			rv.secret = true
		}
		rv.Origins = append(next.Origins, obj.Origins...)
		rv.Array = append(next.Array, obj.Array...)
	case cmdReplace:
		rv.secret = cmd.secret
		rv = next
	case cmdKeep:
	case cmdFail:
		return Object{}, fmt.Errorf("%w: merging an array with command 'fail'", ErrConflict)
	}
	return rv, nil
}

// mergeMap merges two maps.  Don't directly call this, call merge() instead.
func (obj Object) mergeMap(cmd command, next Object) (Object, error) {
	switch cmd.cmd {
	case cmdFail:
		return Object{}, fmt.Errorf("%w: merging a map with command 'fail'", ErrConflict)
	case cmdKeep:
		return obj, nil
	case cmdReplace:
		rv, err := next.resolveCommands(false)
		if err != nil {
			return Object{}, err
		}
		rv.secret = cmd.secret
		return rv, nil
	default:
	}

	// cmd.cmd == cmdSplice || "":
	for key, val := range next.Map {
		newCmd, err := getValidCmd(key, val)
		if err != nil {
			return Object{}, err
		}

		existing, found := obj.Map[newCmd.final]
		if !found {
			// Merging with no conflicts.
			v, err := val.resolveCommands(newCmd.secret)
			if err != nil {
				return Object{}, err
			}
			obj.Map[newCmd.final] = v
			continue
		}

		if existing.Kind() == val.Kind() {
			v, err := existing.merge(newCmd, val)
			if err != nil {
				return Object{}, err
			}
			obj.Map[newCmd.final] = v
			continue
		}

		switch newCmd.cmd {
		case cmdSplice, cmdReplace, "":
			v, err := val.resolveCommands(newCmd.secret)
			if err != nil {
				return Object{}, err
			}
			obj.Map[newCmd.final] = v
		case cmdKeep:
			obj.Map[newCmd.final] = existing
		case cmdFail:
			return Object{}, fmt.Errorf("%w: merging map", ErrConflict)
		}
	}

	obj.secret = cmd.secret
	return obj, nil
}

// getValidCmd gets the command from the key string and validates it is supported.
func getValidCmd(key string, obj Object) (command, error) {
	cmd, err := getCmd(key)
	if err != nil {
		return command{}, err
	}

	list := map[int][]string{
		Map:   {"", cmdFail, cmdKeep, cmdReplace, cmdSplice},
		Array: {"", cmdFail, cmdKeep, cmdReplace, cmdAppend, cmdPrepend},
		Value: {"", cmdFail, cmdKeep, cmdReplace},
	}

	opts, found := list[obj.Kind()]
	if found {
		for _, opt := range opts {
			if cmd.cmd == opt {
				return cmd, nil
			}
		}
	}

	return command{}, ErrInvalidCommand
}

// FilterNonSerializable builds a copy of the tree where any non-serializeable
// types are excluded from the values.
func (obj Object) FilterNonSerializable() Object {
	switch obj.Kind() {
	case Array:
		//array := make([]Object, 0, len(obj.Array))
		var array []Object

		for _, val := range obj.Array {
			if val.isSerializable() {
				got := val.FilterNonSerializable()
				if !got.IsEmpty() {
					array = append(array, got)
				}
			}
		}
		obj.Array = array
	case Map:
		m := make(map[string]Object)

		for key, val := range obj.Map {
			if val.isSerializable() {
				got := val.FilterNonSerializable()
				if !got.IsEmpty() {
					m[key] = val.FilterNonSerializable()
				}
			}
		}
		obj.Map = m
	}

	return obj
}

// ErrOnNonSerializable returns if the
// types are excluded from the values.
func (obj Object) ErrOnNonSerializable() error {
	switch obj.Kind() {
	case Array:
		for _, val := range obj.Array {
			if err := val.ErrOnNonSerializable(); err != nil {
				return err
			}
		}
	case Map:
		for _, val := range obj.Map {
			if err := val.ErrOnNonSerializable(); err != nil {
				return err
			}
		}
	case Value:
		if !obj.isSerializable() {
			return ErrNonSerializable
		}
	}

	return nil
}

// isSerializable returns if the object is serializeable or not.  Children
// objects are not explored.
func (obj Object) isSerializable() bool {
	if obj.Value == nil {
		return true
	}

	v := reflect.ValueOf(obj.Value)
	kind := v.Kind()
	for kind == reflect.Pointer {
		v = v.Elem()
		kind = v.Kind()
	}
	switch kind {
	case reflect.Invalid,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Struct,
		reflect.UnsafePointer:
		return false
	}

	return true
}

// IsEmpty returns if the object is an empty node.  Metadata about the node
// like the origin or secret are ignored in this determination.
func (o Object) IsEmpty() bool {
	if len(o.Array) == 0 && len(o.Map) == 0 && o.Value == nil {
		return true
	}
	return false
}

// AdaptToRaw uses the provided converter function against each parameter and
// builds a new tree based on the output.  This allows for user provided
// adapters to translate from string to a time.Time{} for example.  If multiple
// adapters are needed provide a single adapter with the logic of applying the
// list.
//
// # Note
// The adapter must return the original object, and an error of nil if no
// transformation took place.
func (obj Object) AdaptToRaw(adapter func(from, to reflect.Value) (any, error)) (Object, error) {
	switch obj.Kind() {
	case Array:
		var err error
		array := make([]Object, len(obj.Array))
		for i, val := range obj.Array {
			array[i], err = val.AdaptToRaw(adapter)
			if err != nil {
				return Object{}, err
			}
		}
		obj.Array = array
		return obj, nil
	case Map:
		m := make(map[string]Object)

		for key, val := range obj.Map {
			target, err := val.AdaptToRaw(adapter)
			if err != nil {
				return Object{}, err
			}
			m[key] = target
		}
		obj.Map = m
		return obj, nil
	}

	if adapter != nil && obj.Value != nil {
		v, err := adapter(reflect.ValueOf(obj.Value), reflect.ValueOf("ignored"))
		if err != nil {
			return Object{}, err
		}
		obj.Value = v
	}

	return obj, nil
}

// Clone builds a copy of the tree where secrets are redacted.  Secret maps
// or arrays will now show up as values containing the value 'REDACTED'.
func (obj Object) Clone() Object {
	switch obj.Kind() {
	case Array:
		array := make([]Object, len(obj.Array))
		for i, val := range obj.Array {
			array[i] = val.Clone()
		}
		obj.Array = array
	case Map:
		m := make(map[string]Object)

		for key, val := range obj.Map {
			m[key] = val.Clone()
		}
		obj.Map = m
	}

	return obj
}
