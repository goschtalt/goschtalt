// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/goschtalt/goschtalt/internal/casbab"
	"github.com/goschtalt/goschtalt/internal/fspath"
	"github.com/goschtalt/goschtalt/internal/natsort"
	"github.com/goschtalt/goschtalt/internal/print"
	"github.com/goschtalt/goschtalt/internal/strs"
	"github.com/goschtalt/goschtalt/pkg/decoder"
	"github.com/goschtalt/goschtalt/pkg/encoder"
)

// Option configures specific behavior of Config as well as the locations used
// for the configurations compiled.  There are 3 basic groups of options:
//
//   - Configuration of locations where to collect configuration
//   - Addition of encoders/decoders
//   - Define default behaviors
type Option interface {
	fmt.Stringer

	// apply applies the changes to the options structure and returns an error
	// if one occurs.
	//
	// Generally options should try to always succeed unless they are is no other
	// way, or they are a validator option.  Validator options are a handy way
	// to easily ensure accuracy or exit without alot of code.
	apply(*options) error

	// ignoreDefaults returns false for all options that do not cause the default
	// options to be ignored.  This function lets us alter the options used
	// without the need to run the options twice to see if the defaults are
	// ignored.
	ignoreDefaults() bool
}

type options struct {
	// Settings where there are one.
	disableAutoCompile bool
	keyDelimiter       string
	sorter             RecordSorter
	hasher             Hasher

	// Codecs where there can be many.
	decoders *codecRegistry[decoder.Decoder]
	encoders *codecRegistry[encoder.Encoder]

	// Behaviors where there can be many.
	marshalOptions   []MarshalOption
	unmarshalOptions []UnmarshalOption
	valueOptions     []ValueOption

	// Defaults where there can be many.
	defaults []record

	// General configurations; there can be many.
	filegroups []filegroup
	values     []record

	// Expansions; there can be many.
	expansions    []expand
	exapansionMax int

	// Hints are special options that check that the configuration makes sense;
	// there can be many.
	hints []func(*options) error
}

// ---- Options follow ---------------------------------------------------------

// AddFile adds exactly one file to the list of files to be compiled into a
// configuration.  The filename must be relative to the fs.  If the file
// specified cannot be processed it is considered an error.
func AddFile(fs fs.FS, filename string) Option {
	return &groupOption{
		name: "AddFile",
		grp: filegroup{
			fs:        fs,
			paths:     []string{filename},
			exactFile: true,
		},
	}
}

// AddFileAs is the same as [AddFile]() except the file is decoded as the
// specified type.
func AddFileAs(fs fs.FS, asType, filename string) Option {
	return &groupOption{
		name: "AddFileAs",
		grp: filegroup{
			fs:        fs,
			paths:     []string{filename},
			exactFile: true,
			as:        asType,
		},
	}
}

// AddFiles adds any number of files to the list of files to be compiled into a
// configuration.  The filenames must be relative to the fs.  Any files that
// cannot be processed will be ignored.  It is not an error if any files are
// missing or if all the files cannot be processed.
//
// Use [AddFile]() if you need to require a file to be present.
//
// All the files that can be processed with a decoder will be compiled into the
// configuration.
func AddFiles(fs fs.FS, filenames ...string) Option {
	return &groupOption{
		name: "AddFiles",
		grp: filegroup{
			fs:    fs,
			paths: filenames,
		},
	}
}

// AddFilesAs is the same as [AddFiles]() except the files are decoded as the
// specified type.
func AddFilesAs(fs fs.FS, asType string, filenames ...string) Option {
	return &groupOption{
		name: "AddFilesAs",
		grp: filegroup{
			fs:    fs,
			paths: filenames,
			as:    asType,
		},
	}
}

// AddFilesHalt adds any number of files to the list of files to be compiled
// into a configuration and if any are added halts the process of adding more
// file records.
//
// This option works the same as [AddFiles]() except no further files are
// processed if this group added any files.
//
// This is generally going to be useful for configuring a set of paths to search
// for configuration and stopping when it is found.
func AddFilesHalt(fs fs.FS, filenames ...string) Option {
	return &groupOption{
		name: "AddFilesHalt",
		grp: filegroup{
			fs:    fs,
			paths: filenames,
			halt:  true,
		},
	}
}

// AddFilesHaltAs is the same as [AddFilesHalt]() except the files are decoded as
// the specified type.
func AddFilesHaltAs(fs fs.FS, asType string, filenames ...string) Option {
	return &groupOption{
		name: "AddFilesHaltAs",
		grp: filegroup{
			fs:    fs,
			paths: filenames,
			as:    asType,
			halt:  true,
		},
	}
}

// AddTree adds a directory tree (including all subdirectories) for inclusion
// when compiling the configuration.  Any files that cannot be processed will be
// ignored.  It is not an error if any files are missing, or if all the files
// cannot be processed.
//
// Use [AddFile]() if you need to require a file to be present.
//
// All the files that can be processed with a decoder will be compiled into the
// configuration.
func AddTree(fs fs.FS, path string) Option {
	return &groupOption{
		name: "AddTree",
		grp: filegroup{
			fs:      fs,
			paths:   []string{path},
			recurse: true,
		},
	}
}

// AddTreeHalt adds a directory tree (including all subdirectories) for
// inclusion when compiling the configuration, and if any are added halts the
// process of adding more file records.
//
// This option works the same as [AddTree]() except no further files are
// processed if this group added any files.
//
// This is generally going to be useful for configuring a set of paths to search
// for configuration and stopping when it is found.
func AddTreeHalt(fs fs.FS, path string) Option {
	return &groupOption{
		name: "AddTreeHalt",
		grp: filegroup{
			fs:      fs,
			paths:   []string{path},
			recurse: true,
			halt:    true,
		},
	}
}

// AddTrees adds a list of directory trees (including all subdirectories) for
// inclusion when compiling the configuration.  Any files that cannot be
// processed will be ignored.  It is not an error if any files are missing, or
// if all the files cannot be processed.
//
// Use [AddFile]() if you need to require a file to be present.
//
// All the files that can be processed with a decoder will be compiled into the
// configuration.
func AddTrees(fs fs.FS, paths ...string) Option {
	return &groupOption{
		name: "AddTrees",
		grp: filegroup{
			fs:      fs,
			paths:   paths,
			recurse: true,
		},
	}
}

// AddDir adds a directory (excluding all subdirectories) for inclusion
// when compiling the configuration.  Any files that cannot be processed will be
// ignored.  It is not an error if any files are missing, or if all the files
// cannot be processed.
//
// Use [AddFile]() if you need to require a file to be present.
//
// All the files that can be processed with a decoder will be compiled into the
// configuration.
func AddDir(fs fs.FS, path string) Option {
	return &groupOption{
		name: "AddDir",
		grp: filegroup{
			fs:    fs,
			paths: []string{path},
		},
	}
}

// AddDirs adds a list of directories (excluding all subdirectories) for inclusion
// when compiling the configuration.  Any files that cannot be processed will be
// ignored.  It is not an error if any files are missing, or if all the files
// cannot be processed.
//
// Use [AddFile]() if you need to require a file to be present.
//
// All the files that can be processed with a decoder will be compiled into the
// configuration.
func AddDirs(fs fs.FS, paths ...string) Option {
	return &groupOption{
		name: "AddDirs",
		grp: filegroup{
			fs:    fs,
			paths: paths,
		},
	}
}

// AddJumbled adds any number of files or directories (excluding all
// subdirectories) for inclusion when compiling the configuration.  The files
// and directories are sorted into either a relative based filesystem or an
// absolute path based filesystem.  Any files or directories that cannot be
// processed will be ignored.  It is not an error if any files are missing or if
// all the files cannot be processed.
//
// Note: The paths are expected to use the fs.FS path separator ("/") and format.
//
// Use [AddFile]() if you need to require a file to be present.
//
// Generally this option is useful when processing files from the same filesystem
// but some are absolute path based and others are relative path based.  Instead
// of needing to sort the files into two buckets, this option will handle that
// for you.
func AddJumbled(abs, rel fs.FS, paths ...string) Option {
	return addJumbled("AddJumbled", abs, rel, paths, false)
}

// AddJumbledHalt adds any number of files or directories (excluding all
// subdirectories) for inclusion when compiling the configurationm, and if any
// are added halts the process of adding more file records.
//
// Note: The paths are expected to use the fs.FS path separator ("/") and format.
//
// This option works the same as [AddJumbled]() except no further files are processed
// if this group added any files.
//
// This is generally going to be useful for configuring a set of paths to search
// for configuration and stopping when it is found.
func AddJumbledHalt(abs, rel fs.FS, paths ...string) Option {
	return addJumbled("AddJumbledHalt", abs, rel, paths, true)
}

func addJumbled(name string, abs, rel fs.FS, paths []string, halt bool) Option {
	absPaths := make([]string, 0, len(paths))
	relPaths := make([]string, 0, len(paths))

	for _, p := range paths {
		if p == "" {
			continue
		}

		// If the path is local, it is relative.
		if fspath.IsLocal(p) {
			relPaths = append(relPaths, path.Clean(p))
			continue
		}

		// Everything else ends up being absolute.

		rel, err := fspath.ToRel(p)
		if err != nil {
			return WithError(err)
		}

		absPaths = append(absPaths, rel)
	}

	return &optionsOption{
		text: print.P(name,
			print.Literal("abs"),
			print.Literal("rel"),
			print.Strings(paths),
		),
		opts: []Option{
			&groupOption{
				grp: filegroup{
					fs:    abs,
					paths: absPaths,
				},
			},
			&groupOption{
				grp: filegroup{
					fs:    rel,
					paths: relPaths,
					halt:  halt,
				},
			},
		},
	}
}

type groupOption struct {
	name string
	grp  filegroup
}

var _ Option = (*groupOption)(nil)

func (g groupOption) apply(opts *options) error {
	opts.filegroups = append(opts.filegroups, g.grp)
	return nil
}

func (groupOption) ignoreDefaults() bool {
	return false
}

func (o groupOption) String() string {
	opts := []print.Option{
		print.Literal("fs"),
	}

	if strings.Contains(o.name, "As") {
		opts = append(opts, print.String(o.grp.as))
	}
	opts = append(opts, print.Strings(o.grp.paths))

	return print.P(o.name, opts...)
}

// AutoCompile instructs [New]() and [With]() to also compile the configuration
// after all the options are applied if enable is true or omitted.  Passing
// an enable value of false disables the extra behavior.
//
// The enable bool value is optional & assumed to be `true` if omitted.  The
// first specified value is used if provided.  A value of `false` disables the
// option.
//
// # Default
//
// AutoCompile is enabled.
func AutoCompile(enable ...bool) Option {
	enable = append(enable, true)
	return disableAutoCompileOption(enable[0])
}

type disableAutoCompileOption bool

func (a disableAutoCompileOption) apply(opts *options) error {
	opts.disableAutoCompile = !bool(a)
	return nil
}

func (disableAutoCompileOption) ignoreDefaults() bool { return false }
func (a disableAutoCompileOption) String() string {
	return print.P("AutoCompile", print.BoolSilentTrue(bool(a)))
}

// ConfigIs provides a strict field/key mapper that converts the config
// values from the specified nomenclature into the go structure name.
//
// Since the names of the different formatting styles are not standardized, only
// a few of the common ones have consts defined.  The complete list is below:
//
//   - "two words" aliases: "lower case"
//   - "two-words" aliases: "kebab-case"
//   - "two-Words" aliases: "camel-Kebab-Case"
//   - "two_words" aliases: "snake_case"
//   - "two_Words" aliases: "camel_Snake_Case"
//   - "twowords"  aliases: "flatcase"
//   - "twoWords"  aliases: "camelCase"
//   - "Two Words" aliases: "Title Case"
//   - "Two-Words" aliases: "Pascal-Kebab-Case", "Title-Kebab-Case"
//   - "Two_Words" aliases: "Pascal_Snake_Case", "Title_Snake_Case"
//   - "TwoWords"  aliases: "PascalCase"
//   - "TWO WORDS" aliases: "SCREAMING CASE"
//   - "TWO-WORDS" aliases: "SCREAMING-KEBAB-CASE"
//   - "TWO_WORDS" aliases: "SCREAMING_SNAKE_CASE"
//   - "TWOWORDS"  aliases: "UPPERCASE"
//
// This option provides a KeymapMapper based option that will convert
// every input string, ending the chain 100% of the time.
//
// To make adjustments pass in a map (or many) with keys being the golang
// structure field names and values being the configuration name.
func ConfigIs(format string, overrides ...map[string]string) Option {
	sToC, err := mergeOverrides(overrides)
	if err != nil {
		return WithError(err)
	}

	toCase := casbab.Find(format)
	if toCase == nil {
		return WithError(
			fmt.Errorf("%w, '%s' unknown format ConfigIs()", ErrInvalidInput, format),
		)
	}

	opt := KeymapMapper(&casemapper{
		toCase:      toCase,
		adjustments: sToC,
	})
	return Options(
		DefaultUnmarshalOptions(opt),
		DefaultValueOptions(opt),
	)
}

func mergeOverrides(in []map[string]string) (map[string]string, error) {
	sToC := make(map[string]string, len(in))
	for i := range in {
		for k, v := range in[i] {
			if _, a := sToC[k]; a {
				return nil, fmt.Errorf("%w, '%s' is duplicated.", ErrInvalidInput, k) // nolint:staticcheck
			}
			sToC[k] = v
		}
	}
	return sToC, nil
}

type casemapper struct {
	toCase      func(string) string
	adjustments map[string]string
}

func (c casemapper) Map(in string) string {
	out, found := c.adjustments[in]
	if !found {
		out = c.toCase(in)
	}
	return out
}

// SetKeyDelimiter provides the delimiter used for determining key parts.  A
// string with length of at least 1 must be provided.
//
// # Default
//
// The default value is '.'.
func SetKeyDelimiter(delimiter string) Option {
	return setKeyDelimiterOption(delimiter)
}

type setKeyDelimiterOption string

func (s setKeyDelimiterOption) apply(opts *options) error {
	if len(s) == 0 {
		return fmt.Errorf("%w: a KeyDelimiter with length > 0 must be specified.", ErrInvalidInput) // nolint:staticcheck
	}

	opts.keyDelimiter = string(s)
	return nil
}

func (setKeyDelimiterOption) ignoreDefaults() bool {
	return false
}

func (s setKeyDelimiterOption) String() string {
	return print.P("SetKeyDelimiter", print.String(string(s)))
}

// Hasher provides the methods needed to hash the object tree.
type Hasher interface {
	Hash(any) ([]byte, error)
}

// HasherFunc is an adapter to allow the use of ordinary functions as Hashers.
// If f is a function with the appropriate signature, HasherFunc(f) is a Hasher
// that calls f.
type HasherFunc func(any) ([]byte, error)

// Hash calls f(o)
func (f HasherFunc) Hash(o any) ([]byte, error) {
	return f(o)
}

var _ Hasher = (*HasherFunc)(nil)

// The default do nothing hash function provided.

type defaultHasher struct{}

func (defaultHasher) Hash(o any) ([]byte, error) {
	return []byte{}, nil
}

var _ Hasher = (*defaultHasher)(nil)

// SetHasher provides a way to specify which hash algorithm to use on the object
// tree.  If the provided Hasher is nil, then no hashing is done (default).
func SetHasher(h Hasher) Option {
	if h == nil {
		h = &defaultHasher{}
	}
	return &hashOption{
		hasher: h,
	}
}

type hashOption struct {
	hasher Hasher
}

func (h hashOption) apply(opts *options) error {
	opts.hasher = h.hasher
	return nil
}

func (hashOption) ignoreDefaults() bool { return false }
func (hashOption) String() string       { return "SetHasher" }

// RecordSorter provides the methods needed to sort records.
type RecordSorter interface {
	// Less reports whether a is before b.
	Less(a, b string) bool
}

// The RecordSorterFunc type is an adapter to allow the use of ordinary functions
// as RecordSorters. If f is a function with the appropriate signature,
// RecordSorterFunc(f) is a RecordSorter that calls f.
type RecordSorterFunc func(string, string) bool

// Less calls f(a, b)
func (f RecordSorterFunc) Less(a, b string) bool {
	return f(a, b)
}

var _ RecordSorter = (*RecordSorterFunc)(nil)

// SortRecords provides a way to specify how you want the files sorted
// prior to their merge.  This function provides a way to provide a completely
// custom sorting algorithm.
//
// See also: [SortRecordsLexically], [SortRecordsNaturally]
//
// # Default
//
// The default is [SortRecordsNaturally].
func SortRecords(sorter RecordSorter) Option {
	return &sortRecordsOption{
		text:   print.P("SortRecords", print.Obj(sorter)),
		sorter: sorter,
	}
}

// SortRecordsLexically provides a built in sorter based on lexical order.
//
// See also: [SortRecords], [SortRecordsNaturally]
//
// # Default
//
// The default is [SortRecordsNaturally].
func SortRecordsLexically() Option {
	return &sortRecordsOption{
		text: print.P("SortRecordsLexically"),
		sorter: RecordSorterFunc(
			func(a, b string) bool {
				return a < b
			}),
	}
}

// SortRecordsNaturally provides a built in sorter based on natural order.
// More information about natural sort order: https://en.wikipedia.org/wiki/Natural_sort_order
//
// Notes:
//
//   - Don't use floating point numbers.  They are treated like 2 integers separated
//     by the '.' rune.
//   - Any leading 0 values are dropped from the number.
//
// Example sort order:
//
//	01_foo.yml
//	2_foo.yml
//	98_foo.yml
//	99 dogs.yml
//	99_Abc.yml
//	99_cli.yml
//	99_mine.yml
//	100_alpha.yml
//
// See also: [SortRecords], [SortRecordsLexically]
//
// # Default
//
// The default is [SortRecordsNaturally].
func SortRecordsNaturally() Option {
	return &sortRecordsOption{
		text:   print.P("SortRecordsNaturally"),
		sorter: RecordSorterFunc(natsort.Compare),
	}
}

type sortRecordsOption struct {
	text   string
	sorter RecordSorter
}

func (s sortRecordsOption) apply(opts *options) error {
	if s.sorter == nil {
		return fmt.Errorf("%w: a SortRecords function/option must be specified", ErrInvalidInput)
	}

	opts.sorter = s.sorter
	return nil
}

func (sortRecordsOption) ignoreDefaults() bool { return false }
func (s sortRecordsOption) String() string     { return s.text }

// HintEncoder provides a way to suggest importing additional encoders without
// needing to include a specific one in goschtalt.  Generally, this option is
// not needed unless you are creating pre-set option lists.
func HintEncoder(label, url string, exts ...string) Option {
	return &hintOption{
		text: print.P("HintEncoder", print.Literal(label)),
		hint: func(opts *options) error {
			return hintCodecCheck(opts.encoders.extensions(), exts, url, "encoders")
		},
	}
}

// HintDecoder provides a way to suggest importing additional decoders without
// needing to include a specific one in goschtalt.  Generally, this option is
// not needed unless you are creating pre-set option lists.
func HintDecoder(label, url string, exts ...string) Option {
	return &hintOption{
		text: print.P("HintDecoder", print.Literal(label)),
		hint: func(opts *options) error {
			return hintCodecCheck(opts.decoders.extensions(), exts, url, "decoders")
		},
	}
}

func hintCodecCheck(have, want []string, url, typ string) error {
	missing := strs.Missing(have, want)

	// If some of the extensions are supported, that's probably what they wanted.
	if len(missing) < len(want) {
		return nil
	}

	return fmt.Errorf("%w: none of the required (%s) %s are present, try importing %s",
		ErrHint, strings.Join(missing, ", "), typ, url)
}

type hintOption struct {
	text string
	hint func(*options) error
}

func (h hintOption) apply(opts *options) error {
	opts.hints = append(opts.hints, h.hint)
	return nil
}

func (hintOption) ignoreDefaults() bool {
	return false
}

func (h hintOption) String() string {
	return h.text
}

// WithDecoder registers a Decoder for the specific file extensions provided.
// Attempting to register a duplicate extension is not supported.
//
// See also: [WithEncoder]
func WithDecoder(d decoder.Decoder) Option {
	return &withDecoderOption{decoder: d}
}

type withDecoderOption struct {
	decoder decoder.Decoder
}

func (w withDecoderOption) apply(opts *options) error {
	if w.decoder != nil {
		opts.decoders.register(w.decoder)
	}
	return nil
}

func (withDecoderOption) ignoreDefaults() bool {
	return false
}

func (w withDecoderOption) String() string {
	var s []string
	if w.decoder != nil {
		s = w.decoder.Extensions()
	}

	return print.P("WithDecoder", print.Strings(s))
}

// WithEncoder registers a Encoder for the specific file extensions provided.
// Attempting to register a duplicate extension is not supported.
//
// See also: [WithDecoder]
func WithEncoder(enc encoder.Encoder) Option {
	return &withEncoderOption{enc: enc}
}

type withEncoderOption struct {
	enc encoder.Encoder
}

func (w withEncoderOption) apply(opts *options) error {
	if w.enc != nil {
		opts.encoders.register(w.enc)
	}
	return nil
}

func (withEncoderOption) ignoreDefaults() bool {
	return false
}

func (w withEncoderOption) String() string {
	var s []string
	if w.enc != nil {
		s = w.enc.Extensions()
	}

	return print.P("WithEncoder", print.Strings(s))
}

// DisableDefaultPackageOptions provides a way to explicitly not use any preconfigured
// default values by this package and instead use just the options specified.
//
// See: [DefaultOptions]
func DisableDefaultPackageOptions() Option {
	return disableDefaultPackageOption{}
}

type disableDefaultPackageOption struct{}

func (disableDefaultPackageOption) apply(opts *options) error { return nil }
func (disableDefaultPackageOption) ignoreDefaults() bool      { return true }
func (disableDefaultPackageOption) String() string {
	return print.P("DisableDefaultPackageOptions")
}

// DefaultMarshalOptions allows customization of the desired options for all
// invocations of the [Marshal]() function.  This should make consistent use
// use of the Marshal() call easier.
//
// Valid Option Types:
//   - [GlobalOption]
//   - [MarshalOption]
func DefaultMarshalOptions(opts ...MarshalOption) Option {
	return &defaultMarshalOption{opts: opts}
}

type defaultMarshalOption struct {
	opts []MarshalOption
}

func (d defaultMarshalOption) apply(opts *options) error {
	opts.marshalOptions = append(opts.marshalOptions, d.opts...)
	return nil
}

func (defaultMarshalOption) ignoreDefaults() bool {
	return false
}

func (d defaultMarshalOption) String() string {
	return print.P("DefaultMarshalOptions", print.LiteralStringers(d.opts))
}

// DefaultUnmarshalOptions allows customization of the desired options for all
// invocations of the [Unmarshal]() function.  This should make consistent use
// use of the Unmarshal() call easier.
//
// Valid Option Types:
//   - [GlobalOption]
//   - [UnmarshalOption]
//   - [UnmarshalValueOption]
func DefaultUnmarshalOptions(opts ...UnmarshalOption) Option {
	return &defaultUnmarshalOption{opts: opts}
}

type defaultUnmarshalOption struct {
	opts []UnmarshalOption
}

func (d defaultUnmarshalOption) apply(opts *options) error {
	opts.unmarshalOptions = append(opts.unmarshalOptions, d.opts...)
	return nil
}

func (defaultUnmarshalOption) ignoreDefaults() bool {
	return false
}

func (d defaultUnmarshalOption) String() string {
	return print.P("DefaultUnmarshalOptions", print.LiteralStringers(d.opts))
}

// DefaultValueOptions allows customization of the desired options for all
// invocations of the [AddValue]() and [AddValueFunc]() functions.  This should
// make consistent use use of these functions easier.
//
// Valid Option Types:
//   - [BufferValueOption]
//   - [GlobalOption]
//   - [ValueOption]
//   - [UnmarshalValueOption]
func DefaultValueOptions(opts ...ValueOption) Option {
	return &defaultValueOption{opts: opts}
}

type defaultValueOption struct {
	opts []ValueOption
}

func (d defaultValueOption) apply(opts *options) error {
	opts.valueOptions = append(opts.valueOptions, d.opts...)
	return nil
}

func (defaultValueOption) ignoreDefaults() bool {
	return false
}

func (d defaultValueOption) String() string {
	return print.P("DefaultValueOptions", print.LiteralStringers(d.opts))
}

// Options provides a way to group multiple options together into an easy to
// use Option.
func Options(opts ...Option) Option {
	return NamedOptions("Options", opts...)
}

// NamedOptions provides a way to group multiple options together into an
// easy to use Option with a custom name in the output.  This is helps users
// track down the actual option they called.
func NamedOptions(name string, opts ...Option) Option {
	return &optionsOption{
		text: print.P(name, print.LiteralStringers(opts)),
		opts: opts,
	}
}

// optionsOption allows for returning an option that is actually several
// sub options when needed, without needing to return []Option everywhere.
type optionsOption struct {
	text string
	opts []Option
}

func (m optionsOption) apply(in *options) error {
	for _, opt := range m.opts {
		err := opt.apply(in)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m optionsOption) ignoreDefaults() bool {
	for _, opt := range m.opts {
		if opt.ignoreDefaults() {
			return true
		}
	}
	return false
}

func (m optionsOption) String() string {
	return m.text
}

// StdCfgLayout takes a fairly opinionated view of where configuration files are
// typically found based on the operating system.
//
// # For non-windows implementations:
//
// The first individual configuration file or 'conf.d' directory found is used.
// Once a set of configuration file(s) has been found any configuration files
// included afterward are ignored.
//
// The order the files/directories are searched.
//
//  1. The provided files in the argument list.  If any are provided,
//     configuration must be found exclusively here.
//
// 2. Any files matching this path (if it exists) and glob:
//
//   - $HOME/.<appName>/<appName>.*
//
// 3. Any files found in this directory (if it exists):
//
//   - $HOME/.<appName>/conf.d/
//
// 4. Any files matching this path (if it exists) and glob:
//
//   - /etc/<appName>/<appName>.*
//
// 5. Any files found in this directory (if it exists):
//
//   - /etc/<appName>/conf.d/
//
// # For windows implementations:
//
// StdCfgLayout doesn't support a shared windows layout today.  One is welcome.
func StdCfgLayout(appName string, files ...string) Option {
	return stdCfgLayout(appName, files)
}

// SetMaxExpansions provides a way to set the maximum number of expansions
// allowed before a recursion error is returned.  The value must be greater
// than 0.
//
// # Default
//
// The default value is 10000.
func SetMaxExpansions(max int) Option {
	if max < 1 {
		return WithError(
			fmt.Errorf("%w, SetMaxExpansions must be greater than 0", ErrInvalidInput),
		)
	}
	return setMaxExpansionsOption(max)
}

type setMaxExpansionsOption int

func (s setMaxExpansionsOption) apply(opts *options) error {
	opts.exapansionMax = int(s)
	return nil
}

func (setMaxExpansionsOption) ignoreDefaults() bool {
	return false
}

func (s setMaxExpansionsOption) String() string {
	return print.P("SetMaxExpansions", print.Int(int(s)))
}

// ---- Options related helper functions follow --------------------------------

func ignoreDefaultOpts(opts []Option) bool {
	for _, opt := range opts {
		if opt != nil && opt.ignoreDefaults() {
			return true
		}
	}
	return false
}
