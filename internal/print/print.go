// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package print

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	nilFuncName  = "nil"
	nilBytesName = "nil"
	nilErrorName = "nil"

	notNilFuncName  = "custom"
	notNilBytesName = "[]byte"
)

// Option is the command interface for the print package.
type Option interface {
	isSubOpt() bool
	isYields() bool
	fmt.Stringer
}

// list is a helper that eliminates repeated loops.
type list []Option

// isSubOpt returns if this is a SubOpt type of print out.
func (l list) isSubOpt() bool {
	for _, item := range l {
		if item != nil && item.isSubOpt() {
			return true
		}
	}
	return false
}

// yields returns the Yields() option if one is present.
func (l list) yields() Option {
	for _, item := range l {
		if item != nil && item.isYields() {
			return item
		}
	}

	return nil
}

func (l list) outputCount() int {
	var count int
	for _, item := range l {
		if item != nil && len(item.String()) > 0 {
			count++
		}
	}

	return count
}

// P renders the function and arguments as provided to a string
func P(name string, opts ...Option) string {
	var b strings.Builder

	spacing := " "
	if list(opts).isSubOpt() {
		spacing = ""
	}
	count := list(opts).outputCount()

	_, _ = b.WriteString(name)
	_, _ = b.WriteString("(")
	if count > 0 {
		_, _ = b.WriteString(spacing)
		comma := ""
		for _, item := range opts {
			if item != nil && !item.isYields() {
				s := item.String()
				if len(s) > 0 {
					_, _ = b.WriteString(comma)
					_, _ = b.WriteString(s)
					comma = ", "
				}
			}
		}
		_, _ = b.WriteString(spacing)
	}
	_, _ = b.WriteString(")")

	yields := list(opts).yields()
	if yields != nil {
		_, _ = b.WriteString(yields.String())
	}

	return b.String()
}

// Bool takes a bool with optional label and renders them consistently.
func Bool(b bool, label ...string) Option {
	return labeledSimpleOption(fmt.Sprintf("%t", b), label...)
}

// BoolSilentFalse takes a bool with optional label and renders them consistently.
// If the value of the bool is false, no output is added.
func BoolSilentFalse(b bool, label ...string) Option {
	if b {
		return Bool(b, label...)
	}
	return nil
}

// BoolSilentTrue takes a bool with optional label and renders them consistently.
// If the value of the bool is true, no output is added.
func BoolSilentTrue(b bool, label ...string) Option {
	if b {
		return nil
	}
	return Bool(b, label...)
}

// Bytes takes a []byte with optional label and renders them consistently.
func Bytes(b []byte, label ...string) Option {
	txt := nilBytesName
	if b != nil {
		txt = notNilBytesName
	}
	return labeledSimpleOption(txt, label...)
}

// Error takes a error with optional label and renders them consistently.
func Error(e error, label ...string) Option {
	txt := nilErrorName
	if e != nil {
		txt = fmt.Sprintf("'%v'", e)
	}
	return labeledSimpleOption(txt, label...)
}

// isFuncNil is a helper function for determining if a function is nil.
// Comparing the f via an interface can't be done with just a comparison to
// nil.  The inner value of the func must be checked if the outer interface
// is not nil.
func isFuncNil(f any) bool {
	return f == nil || reflect.ValueOf(f).IsNil()
}

// Func takes a func with optional label and renders them consistently.
func Func(f any, label ...string) Option {
	txt := nilFuncName
	if !isFuncNil(f) {
		txt = notNilFuncName
	}
	return labeledSimpleOption(txt, label...)
}

// FuncAltNil renders the function like Func() except the alt string is used for when
// the function is nil.
func FuncAltNil(f any, alt string, label ...string) Option {
	if !isFuncNil(f) {
		alt = notNilFuncName
	}
	return labeledSimpleOption(alt, label...)
}

// FuncAltNotNil renders the function like Func() except the alt string is used for
// when the function is not nil.
func FuncAltNotNil(f any, alt string, label ...string) Option {
	if isFuncNil(f) {
		alt = nilFuncName
	}
	return labeledSimpleOption(alt, label...)
}

// Int takes an int with optional label and renders them consistently.
func Int(i int, label ...string) Option {
	return labeledSimpleOption(strconv.Itoa(i), label...)
}

// Literal takes a string literal with optional label and renders them consistently.
func Literal(txt string, label ...string) Option {
	return labeledSimpleOption(txt, label...)
}

// LiteralStrings takes an array of string literals with optional label and
// renders them consistently.
func LiteralStrings(s []string, label ...string) Option {
	return labeledSimpleOption(strings.Join(s, ", "), label...)
}

// LiteralStringerss takes an array of fmt.Stringers literals with optional label
// and renders them consistently.
func LiteralStringers[T fmt.Stringer](list []T, label ...string) Option {
	s := make([]string, len(list))
	for i, item := range list {
		s[i] = item.String()
	}
	return LiteralStrings(s, label...)
}

// Obj takes an object with optional label and renders them consistently.
func Obj(o any, label ...string) Option {
	if o == nil {
		return labeledSimpleOption("nil", label...)
	}
	return labeledSimpleOption(reflect.TypeOf(o).String(), label...)
}

// String takes a string parameter with optional label and renders them consistently.
func String(txt string, label ...string) Option {
	return labeledSimpleOption("'"+txt+"'", label...)
}

// Stringers takes an array of fmt.Stringers parameters with optional label and
// renders them consistently.
func Stringers[T fmt.Stringer](list []T, label ...string) Option {
	s := make([]string, len(list))
	for i, item := range list {
		s[i] = item.String()
	}
	return Strings(s, label...)
}

// Strings takes an array of strings parameter with optional label and renders
// them consistently.
func Strings(s []string, label ...string) Option {
	return labeledSimpleOption("'"+strings.Join(s, "', '")+"'", label...)
}

// StringMap takes a map of strings to strings parameter with optional label and
// renders them consistently.
func StringMap(m map[string]string, label ...string) Option {
	return labeledSimpleOption(fmt.Sprintf("map[%d]", len(m)), label...)
}

// labeledSimpleOption is a helper function to deal with the labels consistently.
func labeledSimpleOption(txt string, label ...string) simpleOption {
	if len(label) > 0 {
		return simpleOption(label[0] + ": " + txt)
	}
	return simpleOption(txt)
}

// simpleOption is a simple option that represent most of the rest of the options.
type simpleOption string

func (simpleOption) isYields() bool   { return false }
func (simpleOption) isSubOpt() bool   { return false }
func (s simpleOption) String() string { return string(s) }

// SubOpt denotes when the documented option is not the main goschtalt.Option
// type and should have it's parenthesis grouped tighter.
func SubOpt() Option {
	return subOption("")
}

type subOption string

func (subOption) isYields() bool { return false }
func (subOption) isSubOpt() bool { return true }
func (subOption) String() string { return "" }

// Yields denotes when a group of options is describing the outcome of set of
// objects for representation using more of the `Foo() --> things` pattern.  The
// options inside this option are what show up after the -->.
func Yields(opts ...Option) Option {
	return &yieldsOption{
		opts: opts,
	}
}

type yieldsOption struct {
	opts []Option
}

func (yieldsOption) isYields() bool { return true }
func (yieldsOption) isSubOpt() bool { return false }
func (y yieldsOption) String() string {
	var b strings.Builder

	_, _ = b.WriteString(" --> ")

	comma := ""
	for _, item := range y.opts {
		if item == nil {
			continue
		}

		s := item.String()
		if len(s) > 0 {
			_, _ = b.WriteString(comma)
			_, _ = b.WriteString(s)
			comma = ", "
		}
	}

	return b.String()
}
