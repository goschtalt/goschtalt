// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"fmt"
	"os"

	"github.com/goschtalt/goschtalt/internal/print"
)

// Expander provides a method that can expand variables in values.
type Expander interface {
	// Expand is expected to work just like os.Getenv.
	Expand(string) string
}

type envExpander struct{}

func (envExpander) Expand(s string) string {
	return os.Getenv(s)
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

func (_ expand) ignoreDefaults() bool {
	return false
}

func (exp expand) String() string {
	return exp.text
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
