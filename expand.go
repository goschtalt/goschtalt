// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"fmt"
	"os"

	"github.com/goschtalt/goschtalt/internal/print"
	"github.com/goschtalt/goschtalt/pkg/meta"
)

// Expander provides a method that can expand variables in values.
type Expander interface {

	// Expand maps the incoming string to a new string.  The string passed in
	// will not contain the start and end delimiters.  If the string is not
	// found, return the bool value of false, otherwise return true.
	Expand(string) (string, bool)
}

// The ExpanderFunc type is an adapter to allow the use of ordinary functions
// as Expanders. If f is a function with the appropriate signature,
// ExpanderFunc(f) is a Expander that calls f.
type ExpanderFunc func(string) (string, bool)

// Get calls f(s)
func (f ExpanderFunc) Expand(s string) (string, bool) {
	return f(s)
}

var _ Expander = (*ExpanderFunc)(nil)

type envExpander struct{}

func (envExpander) Expand(s string) (string, bool) {
	return os.LookupEnv(s)
}

// ExpandEnv is a simple way to add automatic environment variable expansion
// after the configuration has been compiled.
//
// Expand() and ExpandEnv() directives are evaluated in the order specified.
//
// Valid Option Types:
//   - [ExpandOption]
//   - [GlobalOption]
func ExpandEnv(opts ...ExpandOption) Option {
	exp := expand{
		origin:   "environment",
		expander: envExpander{},
		start:    "${",
		end:      "}",
	}

	for _, opt := range opts {
		if err := opt.expandApply(&exp); err != nil {
			return WithError(fmt.Errorf("ExpandEnv() err: %w", err))
		}
	}

	exp.text = print.P("ExpandEnv",
		print.Literal("..."),
		print.Yields(
			print.String(exp.start, "start"),
			print.String(exp.end, "end"),
			print.String(exp.origin, "origin"),
			print.Int(exp.maximum, "maximum"),
		),
	)

	return &exp
}

// Expand provides a way to expand variables in values throughout the
// configuration tree.  Expand() can be called multiple times to expand
// variables based on additional configurations and mappers.
//
// The initial discovery of a variable to expand in the configuration tree
// value is determined by the Start and End delimiters options provided. The
// default delimiters are "${" and "}" respectively.  Further expansions of
// values replaces ${var} or $var in the string based on the mapping function
// provided.
//
// Expand() and ExpandEnv() directives are evaluated in the order specified.
//
// Valid Option Types:
//   - [ExpandOption]
//   - [GlobalOption]
func Expand(expander Expander, opts ...ExpandOption) Option {
	exp := expand{
		expander: expander,
		start:    "${",
		end:      "}",
	}

	for _, opt := range opts {
		if err := opt.expandApply(&exp); err != nil {
			return WithError(fmt.Errorf("Expand() err: %w", err))
		}
	}

	exp.text = print.P("Expand",
		print.Obj(expander),
		print.Literal("..."),
		print.Yields(
			print.String(exp.start, "start"),
			print.String(exp.end, "end"),
			print.String(exp.origin, "origin"),
			print.Int(exp.maximum, "maximum"),
		),
	)

	return &exp
}

// expand controls how variables are identified and processed.
type expand struct {
	// The text of the option that provided this expand command.
	text string

	// Optional name showing where the value came from.
	origin string

	// The string that prefixes a variable.  "${{" or "${" are common examples.
	// Defaults to "${" if equal to "".
	start string

	// The string that trails a variable.  "}}" or "}" are common examples.
	// Defaults to "}" if equal to "".
	end string

	// The string to string mapping function.
	// Mapping request ignored if nil.
	expander Expander

	// The maximum expansions of a value before a recursion error is returned.
	// Defaults to 10000 if set to less than 1.
	maximum int
}

func (exp expand) apply(opts *options) error {
	if exp.maximum < 1 {
		exp.maximum = 10000
	}
	if exp.expander != nil {
		opts.expansions = append(opts.expansions, exp)
	}

	return nil
}

func (expand) ignoreDefaults() bool {
	return false
}

func (exp expand) String() string {
	return exp.text
}

// expandTree is a helper function that expands variables in the configuration
// tree.  The maximum number of expansions is limited to the max value.
func expandTree(in meta.Object, max int, expansions []expand) (meta.Object, bool, error) {
	changed := true
	for i := 0; changed && i < max; i++ {
		changed = false
		for _, exp := range expansions {
			var err error
			in, err = in.ToExpanded(
				exp.maximum,
				exp.origin,
				exp.start,
				exp.end,
				func(s string) (string, bool) {
					got, found := exp.expander.Expand(s)
					if found {
						changed = true
					}
					return got, found
				},
			)

			if err != nil {
				return meta.Object{}, false, err
			}
		}
	}

	return in, changed, nil
}

// ---- ExpandOption follow --------------------------------------------------

// ExpandOption provides the means to configure options around variable
// expansion.
type ExpandOption interface {
	expandApply(*expand) error
}

// WithOrigin provides the origin name to add showing where a value in the
// configuration tree originates from.
func WithOrigin(origin string) ExpandOption {
	return withOriginOption(origin)
}

type withOriginOption string

func (w withOriginOption) expandApply(exp *expand) error {
	exp.origin = string(w)
	return nil
}

// WithDelimiters provides a way to define different delimiters for the start
// and end of a variable for matching purposes.
func WithDelimiters(start, end string) ExpandOption {
	return &withDelimitersOption{start: start, end: end}
}

type withDelimitersOption struct {
	start string
	end   string
}

func (w withDelimitersOption) expandApply(exp *expand) error {
	exp.start = w.start
	exp.end = w.end
	return nil
}

// WithMaximum provides a way to overwrite the maximum number of times variables
// are expanded.  Any value less than 1 will default to 10000 as a precaution
// against getting trapped in an infinite loop.
func WithMaximum(maximum int) ExpandOption {
	return withMaximumOption(maximum)
}

type withMaximumOption int

func (w withMaximumOption) expandApply(exp *expand) error {
	exp.maximum = int(w)
	return nil
}
