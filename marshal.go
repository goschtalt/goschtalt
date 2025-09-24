// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"fmt"
	"strings"
	"time"

	"github.com/goschtalt/goschtalt/internal/encoding"
	"github.com/goschtalt/goschtalt/internal/encoding/yaml"
	"github.com/goschtalt/goschtalt/internal/print"
	"github.com/goschtalt/goschtalt/pkg/meta"
)

// Marshal renders the into the format specified ('json', 'yaml' or other extensions
// the Codecs provide and if adding comments should be attempted.  If a format
// does not support comments, an error is returned.  The result of the call is
// a slice of bytes with the information rendered into it.
//
// Valid Option Types:
//   - [GlobalOption]
//   - [MarshalOption]
func (c *Config) Marshal(opts ...MarshalOption) ([]byte, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.compiledAt.Equal(time.Time{}) {
		return nil, ErrNotCompiled
	}

	cfg, err := c.getMarshalOptions(opts)
	if err != nil {
		return nil, err
	}

	tree, err := c.getMarshalTree(cfg)
	if err != nil {
		return nil, err
	}

	// Issue 52 - depending on encoders, they may encode a nil or null object
	// instead of returning an expected empty array of bytes.
	if tree == nil {
		return []byte{}, nil
	}

	if cfg.withDocumentation {
		docs := c.opts.doc.Translate(cfg.Map)

		got, err := calcUnified(&docs, tree, cfg.onlyDefaults)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate unified tree: %w", err)
		}

		var out strings.Builder
		if err = cfg.enc.Encode(&out, &got); err != nil {
			return nil, fmt.Errorf("failed to encode document: %w", err)
		}
		return []byte(out.String()), nil
	}

	enc, err := c.opts.encoders.find(cfg.format)
	if err != nil {
		return nil, err
	}

	if cfg.withOrigins {
		return enc.EncodeExtended(*tree)
	}

	return enc.Encode(tree.ToRaw())
}

func (c *Config) getMarshalOptions(opts []MarshalOption) (*marshalOptions, error) {
	cfg := marshalOptions{
		withDocumentation: false, //true,
		enc:               &yaml.Renderer{},
	}
	exts := c.opts.encoders.extensions()
	if len(exts) > 0 {
		cfg.format = exts[0]
	}

	full := append(c.opts.marshalOptions, opts...)
	for _, opt := range full {
		if opt != nil {
			if err := opt.marshalApply(&cfg); err != nil {
				return nil, err
			}
		}
	}

	return &cfg, nil
}

func (c *Config) getMarshalTree(cfg *marshalOptions) (*meta.Object, error) {
	tree := c.tree
	if cfg.onlyDefaults {
		results, err := c.compileInternal(true)
		if err != nil {
			return nil, err
		}
		tree = results.merged
	}

	if cfg.redactSecrets {
		tree = tree.ToRedacted()
	}

	if tree.IsEmpty() {
		return nil, nil
	}

	return &tree, nil
}

// ---- MarshalOption options follow -------------------------------------------

// MarshalOption provides specific configuration for the process of producing
// a document based on the present information in the goschtalt object.
type MarshalOption interface {
	fmt.Stringer

	// marshalApply applies the options to the Marshal function.
	marshalApply(*marshalOptions) error
}

type marshalOptions struct {
	redactSecrets     bool
	onlyDefaults      bool
	withDocumentation bool
	withOrigins       bool
	format            string
	enc               encoding.Encoder
	mappers           []Mapper
}

func (m marshalOptions) Map(s string) string {
	for _, m := range m.mappers {
		rv := m.Map(s)
		switch rv {
		case "":
			continue
		case "-":
			return s
		default:
			s = rv
		}
	}
	return s
}

// RedactSecrets enables the replacement of secret portions of the tree with
// REDACTED.  Passing a redact value of false disables this behavior.
//
// The unused bool value is optional & assumed to be `true` if omitted.  The
// first specified value is used if provided.  A value of `false` disables the
// option.
//
// # Default
//
// Secret values are redacted.
func RedactSecrets(redact ...bool) MarshalOption {
	redact = append(redact, true)
	return redactSecretsOption(redact[0])
}

type redactSecretsOption bool

func (r redactSecretsOption) marshalApply(opts *marshalOptions) error {
	opts.redactSecrets = bool(r)
	return nil
}

func (r redactSecretsOption) String() string {
	return print.P("RedactSecrets", print.BoolSilentTrue(bool(r)), print.SubOpt())
}

// IncludeOrigins enables or disables providing the origin for each configuration
// value present.
//
// # Default
//
// Origins are not included by default.
func IncludeOrigins(origins ...bool) MarshalOption {
	origins = append(origins, true)
	return includeOriginsOption(origins[0])
}

type includeOriginsOption bool

func (w includeOriginsOption) marshalApply(opts *marshalOptions) error {
	opts.withOrigins = bool(w)
	return nil
}

func (i includeOriginsOption) String() string {
	return print.P("IncludeOrigins", print.BoolSilentTrue(bool(i)), print.SubOpt())
}

// FormatAs specifies the final document format extension to use when performing
// the operation.
func FormatAs(extension string) MarshalOption {
	return formatAsOption(extension)
}

type formatAsOption string

func (f formatAsOption) marshalApply(opts *marshalOptions) error {
	opts.format = string(f)
	return nil
}

func (f formatAsOption) String() string {
	return print.P("FormatAs", print.String(string(f)), print.SubOpt())
}

// IncludeDocumentation enables or disables including documentation comments in
// the output, if the output format supports it.
func IncludeDocumentation(doc ...bool) MarshalOption {
	doc = append(doc, true)
	return includeDocumentationOption(doc[0])
}

type includeDocumentationOption bool

func (i includeDocumentationOption) marshalApply(opts *marshalOptions) error {
	opts.withDocumentation = bool(i)
	return nil
}

func (i includeDocumentationOption) String() string {
	return print.P("IncludeDocumentation", print.BoolSilentTrue(bool(i)), print.SubOpt())
}

// OnlyDefaults enables or disables including only default values in the output.
// This is mainly useful for generating documentation or for debugging purposes.
func OnlyDefaults(only ...bool) MarshalOption {
	only = append(only, true)
	return onlyDefaultsOption(only[0])
}

type onlyDefaultsOption bool

func (o onlyDefaultsOption) marshalApply(opts *marshalOptions) error {
	opts.onlyDefaults = bool(o)
	return nil
}

func (o onlyDefaultsOption) String() string {
	return print.P("OnlyDefaults", print.BoolSilentTrue(bool(o)), print.SubOpt())
}

type FormatAsYAMLOptions struct {
	// MaxLineLength is the maximum length of a line before it will be broken.
	// If less than 1 is provided, the default value will be used.  The default
	// value is 80.
	MaxLineLength int

	// TrailingCommentColumn is the column at which to place trailing comments.
	// If less than 1 is provided, the default value will be used.  The default
	// value is 80.
	TrailingCommentColumn int

	// SpacesPerIndent is the number of spaces to use for each indentation level.
	// If less than 1 is provided, the default value will be used.  The default
	// value is 2.
	SpacesPerIndent int
}

func (f FormatAsYAMLOptions) marshalApply(opts *marshalOptions) error {
	opts.format = "yaml"
	opts.enc = &yaml.Renderer{
		MaxLineLength:         f.MaxLineLength,
		TrailingCommentColumn: f.TrailingCommentColumn,
		SpacesPerIndent:       f.SpacesPerIndent,
	}
	return nil
}

func (f FormatAsYAMLOptions) String() string {
	return print.P("FormatAsYAML", print.SubOpt())
}

func FormatAsYAML(opt ...FormatAsYAMLOptions) MarshalOption {
	opt = append(opt, FormatAsYAMLOptions{})
	return &opt[0]
}
