// SPDX-FileCopyrightText: 2022-2023 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/goschtalt/goschtalt/internal/fspath"
	"github.com/goschtalt/goschtalt/pkg/decoder"
	"github.com/goschtalt/goschtalt/pkg/encoder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	unknownErr := errors.New("unknown err")
	testErr := errors.New("test err")
	fs := fstest.MapFS{}
	abs := fstest.MapFS{}
	rel := fstest.MapFS{}
	absFile, err := filepath.Abs("path1")
	require.NoError(t, err)
	list := []string{"zeta", "alpha", "19beta", "19alpha", "4tango",
		"1alpha", "7alpha", "bravo", "7alpha10", "7alpha2", "7alpha0"}

	retBuf := mockBufferGetter{
		f: func(name string, un Unmarshaler) ([]byte, error) {
			return []byte(name), nil
		},
	}

	sortCheck := func(cfg *options, want []string) bool {
		if cfg.sorter == nil {
			return false
		}

		got := list

		sorter := func(a []string) {
			sort.SliceStable(a, func(i, j int) bool {
				return cfg.sorter.Less(a[i], a[j])
			})
		}

		sorter(got)

		return assert.Equal(t, got, want)
	}

	tests := []struct {
		description string
		opt         Option
		opts        []Option
		goal        options
		check       func(*options) bool
		str         string
		ignore      bool
		initCodecs  bool
		expectErr   error
	}{
		{
			description: "WithError( testErr )",
			opt:         WithError(testErr),
			str:         "WithError( 'test err' )",
			expectErr:   testErr,
		}, {
			description: "WithError( nil )",
			opt:         WithError(nil),
			str:         "WithError( nil )",
		}, {
			description: "AddFile( /, filename )",
			opt:         AddFile(fs, "filename"),
			str:         "AddFile( fs, 'filename' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:        fs,
						paths:     []string{"filename"},
						exactFile: true,
					},
				},
			},
		}, {
			description: "AddFile( /, a ), AddFile( /, b )",
			opts:        []Option{AddFile(fs, "a"), AddFile(fs, "b")},
			goal: options{
				filegroups: []filegroup{
					{
						fs:        fs,
						paths:     []string{"a"},
						exactFile: true,
					}, {
						fs:        fs,
						paths:     []string{"b"},
						exactFile: true,
					},
				},
			},
		}, {
			description: "AddFileAs( /, json, filename )",
			opt:         AddFileAs(fs, "json", "filename"),
			str:         "AddFileAs( fs, 'json', 'filename' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:        fs,
						paths:     []string{"filename"},
						exactFile: true,
						as:        "json",
					},
				},
			},
		}, {
			description: "AddFiles( / )",
			opt:         AddFiles(fs),
			str:         "AddFiles( fs, '' )",
			goal: options{
				filegroups: []filegroup{{fs: fs}},
			},
		}, {
			description: "AddFiles( /, filename )",
			opt:         AddFiles(fs, "filename"),
			str:         "AddFiles( fs, 'filename' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:    fs,
						paths: []string{"filename"},
					},
				},
			},
		}, {
			description: "AddFiles( /, a, b )",
			opt:         AddFiles(fs, "a", "b"),
			str:         "AddFiles( fs, 'a', 'b' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:    fs,
						paths: []string{"a", "b"},
					},
				},
			},
		}, {
			description: "AddFilesAs( /, json, filename )",
			opt:         AddFilesAs(fs, "json", "filename"),
			str:         "AddFilesAs( fs, 'json', 'filename' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:    fs,
						paths: []string{"filename"},
						as:    "json",
					},
				},
			},
		}, {
			description: "AddFilesHaltAs( /, json, filename )",
			opt:         AddFilesHaltAs(fs, "json", "filename"),
			str:         "AddFilesHaltAs( fs, 'json', 'filename' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:    fs,
						paths: []string{"filename"},
						halt:  true,
						as:    "json",
					},
				},
			},
		}, {
			description: "AddTree( /, path )",
			opt:         AddTree(fs, "./path"),
			str:         "AddTree( fs, './path' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:      fs,
						paths:   []string{"./path"},
						recurse: true,
					},
				},
			},
		}, {
			description: "AddTrees( /, path1, path2 )",
			opt:         AddTrees(fs, "./path1", "./path2"),
			str:         "AddTrees( fs, './path1', './path2' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:      fs,
						paths:   []string{"./path1", "./path2"},
						recurse: true,
					},
				},
			},
		}, {
			description: "AddJumbled( /, ., /path1, path2 )",
			opt:         AddJumbled(abs, rel, absFile, "./path2"),
			str:         "AddJumbled( abs, rel, '" + absFile + "', './path2' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:    abs,
						paths: []string{fspath.MustToRel(absFile)},
					}, {
						fs:    rel,
						paths: []string{"path2"},
					},
				},
			},
		}, {
			description: "AddDirs( /, path1, path2)",
			opt:         AddDirs(fs, "./path1", "./path2"),
			str:         "AddDirs( fs, './path1', './path2' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:    fs,
						paths: []string{"./path1", "./path2"},
					},
				},
			},
		}, {
			description: "AddDir( /, path )",
			opt:         AddDir(fs, "./path"),
			str:         "AddDir( fs, './path' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:    fs,
						paths: []string{"./path"},
					},
				},
			}}, {
			description: "AddDirs( /, path1, path2)",
			opt:         AddDirs(fs, "./path1", "./path2"),
			str:         "AddDirs( fs, './path1', './path2' )",
			goal: options{
				filegroups: []filegroup{
					{
						fs:    fs,
						paths: []string{"./path1", "./path2"},
					},
				},
			},
		}, {
			description: "AutoCompile()",
			opt:         AutoCompile(),
			str:         "AutoCompile()",
		}, {
			description: "AutoCompile(false)",
			opt:         AutoCompile(false),
			str:         "AutoCompile( false )",
			goal: options{
				disableAutoCompile: true,
			},
		}, {
			description: "SetKeyDelimiter( . )",
			opt:         SetKeyDelimiter("."),
			str:         "SetKeyDelimiter( '.' )",
			goal: options{
				keyDelimiter: ".",
			},
		}, {
			description: "SetKeyDelimiter( '' )",
			opt:         SetKeyDelimiter(""),
			str:         "SetKeyDelimiter( '' )",
			expectErr:   ErrInvalidInput,
		}, {
			description: "SortRecords( nil )",
			opt:         SortRecords(nil),
			str:         "SortRecords( nil )",
			expectErr:   ErrInvalidInput,
		}, {
			description: "SortRecords( '(reverse)' )",
			opt: SortRecords(mockRecordSorter{
				f: func(a, b string) bool {
					return a > b
				},
			}),
			str: "SortRecords( goschtalt.mockRecordSorter )",
			check: func(cfg *options) bool {
				return sortCheck(cfg, []string{
					"zeta",
					"bravo",
					"alpha",
					"7alpha2",
					"7alpha10",
					"7alpha0",
					"7alpha",
					"4tango",
					"1alpha",
					"19beta",
					"19alpha",
				})
			},
		}, {
			description: "SortRecordsNaturally()",
			opt:         SortRecordsNaturally(),
			str:         "SortRecordsNaturally()",
			check: func(cfg *options) bool {
				return sortCheck(cfg, []string{
					"1alpha",
					"4tango",
					"7alpha",
					"7alpha0",
					"7alpha2",
					"7alpha10",
					"19alpha",
					"19beta",
					"alpha",
					"bravo",
					"zeta",
				})
			},
		}, {
			description: "SortRecordsLexically()",
			opt:         SortRecordsLexically(),
			str:         "SortRecordsLexically()",
			check: func(cfg *options) bool {
				return sortCheck(cfg, []string{
					"19alpha",
					"19beta",
					"1alpha",
					"4tango",
					"7alpha",
					"7alpha0",
					"7alpha10",
					"7alpha2",
					"alpha",
					"bravo",
					"zeta",
				})
			},
		}, {
			description: "WithEncoder( json, yml )",
			opt:         WithEncoder(&testEncoder{extensions: []string{"json", "yml"}}),
			str:         "WithEncoder( 'json', 'yml' )",
			initCodecs:  true,
			check: func(cfg *options) bool {
				return assert.Equal(t, []string{"json", "yml"},
					cfg.encoders.extensions())
			},
		}, {
			description: "WithEncoder( foo )",
			opt:         WithEncoder(&testEncoder{extensions: []string{"foo"}}),
			str:         "WithEncoder( 'foo' )",
			initCodecs:  true,
			check: func(cfg *options) bool {
				return assert.Equal(t, []string{"foo"},
					cfg.encoders.extensions())
			},
		}, {
			description: "WithEncoder(nil)",
			opt:         WithEncoder(nil),
			str:         "WithEncoder( '' )",
		}, {
			description: "WithDecoder( json, yml )",
			opt:         WithDecoder(&testDecoder{extensions: []string{"json", "yml"}}),
			str:         "WithDecoder( 'json', 'yml' )",
			initCodecs:  true,
			check: func(cfg *options) bool {
				return assert.Equal(t, []string{"json", "yml"},
					cfg.decoders.extensions())
			},
		}, {
			description: "WithDecoder( foo )",
			opt:         WithDecoder(&testDecoder{extensions: []string{"foo"}}),
			str:         "WithDecoder( 'foo' )",
			initCodecs:  true,
			check: func(cfg *options) bool {
				return assert.Equal(t, []string{"foo"},
					cfg.decoders.extensions())
			},
		}, {
			description: "WithDecoder(nil)",
			opt:         WithDecoder(nil),
			str:         "WithDecoder( '' )",
		}, {
			description: "DisableDefaultPackageOptions()",
			opt:         DisableDefaultPackageOptions(),
			str:         "DisableDefaultPackageOptions()",
			ignore:      true,
		}, {
			description: "DefaultMarshalOptions()",
			opt:         DefaultMarshalOptions(),
			str:         "DefaultMarshalOptions()",
		}, {
			description: "DefaultMarshalOptions( RedactSecrets(true), IncludeOrigins(true), FormatAs(foo), IncludeDocumentation(true), OnlyDefaults(true) )",
			opt:         DefaultMarshalOptions(RedactSecrets(true), IncludeOrigins(true), FormatAs("foo"), IncludeDocumentation(true), OnlyDefaults(true)),
			str:         "DefaultMarshalOptions( RedactSecrets(), IncludeOrigins(), FormatAs('foo'), IncludeDocumentation(), OnlyDefaults() )",
			goal: options{
				marshalOptions: []MarshalOption{redactSecretsOption(true), includeOriginsOption(true), formatAsOption("foo"), includeDocumentationOption(true), onlyDefaultsOption(true)},
			},
		}, {
			description: "DefaultMarshalOptions( RedactSecrets(false), IncludeOrigins(false), IncludeDocumentation(false), OnlyDefaults(false) )",
			opt:         DefaultMarshalOptions(RedactSecrets(false), IncludeOrigins(false), IncludeDocumentation(false), OnlyDefaults(false)),
			str:         "DefaultMarshalOptions( RedactSecrets(false), IncludeOrigins(false), IncludeDocumentation(false), OnlyDefaults(false) )",
			goal: options{
				marshalOptions: []MarshalOption{redactSecretsOption(false), includeOriginsOption(false), includeDocumentationOption(false), onlyDefaultsOption(false)},
			},
		}, {
			description: "DefaultMarshalOptions( RedactSecrets(), IncludeOrigins(), IncludeDocumentation(), OnlyDefaults() )",
			opt:         DefaultMarshalOptions(RedactSecrets(), IncludeOrigins(), IncludeDocumentation(), OnlyDefaults()),
			str:         "DefaultMarshalOptions( RedactSecrets(), IncludeOrigins(), IncludeDocumentation(), OnlyDefaults() )",
			goal: options{
				marshalOptions: []MarshalOption{redactSecretsOption(true), includeOriginsOption(true), includeDocumentationOption(true), onlyDefaultsOption(true)},
			},
		}, {
			description: "DefaultMarshalOptions( RedactSecrets() ), DefaultMarshalOptions( IncludeOrigins() )",
			opts: []Option{
				DefaultMarshalOptions(RedactSecrets(true)),
				DefaultMarshalOptions(IncludeOrigins(true)),
			},
			goal: options{
				marshalOptions: []MarshalOption{redactSecretsOption(true), includeOriginsOption(true)},
			},
		}, {
			description: "DefaultUnmarshalOptions()",
			opt:         DefaultUnmarshalOptions(),
			str:         "DefaultUnmarshalOptions()",
		}, {
			description: "DefaultUnmarshalOptions( Optional(true), Required(true), WithValidator(nil) )",
			opt:         DefaultUnmarshalOptions(Optional(true), Required(true), WithValidator(nil)),
			str:         "DefaultUnmarshalOptions( Optional(), Required(), WithValidator(nil) )",
			goal: options{
				unmarshalOptions: []UnmarshalOption{
					&optionalOption{
						text:     "Optional()",
						optional: true,
					},
					&optionalOption{
						text: "Required()",
					},
					&validatorOption{},
				},
			},
		}, {
			description: "DefaultUnmarshalOptions( Optional(false), Required(false) )",
			opt:         DefaultUnmarshalOptions(Optional(false), Required(false)),
			str:         "DefaultUnmarshalOptions( Optional(false), Required(false) )",
			goal: options{
				unmarshalOptions: []UnmarshalOption{
					&optionalOption{
						text: "Optional(false)",
					},
					&optionalOption{
						text:     "Required(false)",
						optional: true,
					},
				},
			},
		}, {
			description: "DefaultUnmarshalOptions( Optional(), Required() )",
			opt:         DefaultUnmarshalOptions(Optional(), Required()),
			str:         "DefaultUnmarshalOptions( Optional(), Required() )",
			goal: options{
				unmarshalOptions: []UnmarshalOption{
					&optionalOption{
						text:     "Optional()",
						optional: true,
					},
					&optionalOption{
						text: "Required()",
					},
				},
			},
		}, {
			description: "DefaultUnmarshalOptions( Optional() ), DefaultUnmarshalOptions( Required() )",
			opts: []Option{
				DefaultUnmarshalOptions(Optional()),
				DefaultUnmarshalOptions(Required()),
			},
			goal: options{
				unmarshalOptions: []UnmarshalOption{
					&optionalOption{
						text:     "Optional()",
						optional: true,
					},
					&optionalOption{
						text: "Required()",
					},
				},
			},
		}, {
			description: "DefaultUnmarshalOptions( most )",
			opt: DefaultUnmarshalOptions(
				Strictness(NONE),
				Strictness(SUBSET),
				Strictness(COMPLETE),
				Strictness(EXACT),
				Strictness(Level("Invalid")),
				TagName("tag"),
			),
			goal: options{
				unmarshalOptions: []UnmarshalOption{
					&remapOption{
						level: "NONE",
					},
					&remapOption{
						level:       "SUBSET",
						errorUnused: true,
					},
					&remapOption{
						level:      "COMPLETE",
						errorUnset: true,
					},
					&remapOption{
						level:       "EXACT",
						errorUnused: true,
						errorUnset:  true,
					},
					&remapOption{
						level: "Invalid",
						err:   fmt.Errorf("%w: unsupported strictness level: 'Invalid'", ErrInvalidInput),
					},
					tagNameOption("tag"),
				},
			},
			str: "DefaultUnmarshalOptions( Strictness('NONE'), Strictness('SUBSET'), Strictness('COMPLETE'), Strictness('EXACT'), Strictness('Invalid'), TagName('tag') )",
		}, {
			description: "DefaultValueOptions()",
			opt:         DefaultValueOptions(),
			str:         "DefaultValueOptions()",
		}, {
			description: "AddBuffer( filename.ext, bytes )",
			opt:         AddBuffer("filename.ext", []byte("bytes")),
			str:         "AddBuffer( 'filename.ext', []byte )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "filename.ext" {
						if cfg.values[0].buf.getter != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddBuffer( filename.ext, nil )",
			opt:         AddBuffer("filename.ext", nil),
			str:         "AddBuffer( 'filename.ext', nil )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "filename.ext" {
						if cfg.values[0].buf.getter != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddBuffer( '', bytes )",
			opt:         AddBuffer("", []byte("bytes")),
			str:         "AddBuffer( '', []byte )",
			expectErr:   unknownErr,
		}, {
			description: "AddBufferGetter( filename.ext, nil )",
			opt:         AddBufferGetter("filename.ext", nil),
			str:         "AddBufferGetter( 'filename.ext', nil )",
			expectErr:   unknownErr,
		}, {
			description: "AddBufferGetter( filename.ext, bytes )",
			opt:         AddBufferGetter("filename.ext", retBuf),
			str:         "AddBufferGetter( 'filename.ext', goschtalt.mockBufferGetter )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "filename.ext" {
						if cfg.values[0].buf.getter != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValueGetter( record1, '', func )",
			opt: AddValueGetter("record1", Root,
				mockValueGetter{
					f: func(_ string, un Unmarshaler) (any, error) { return nil, nil },
				},
			),
			str: "AddValueGetter( 'record1', '', goschtalt.mockValueGetter )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "record1" {
						if cfg.values[0].val.getter != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValueGetter( record1, 'key', nil )",
			opt:         AddValueGetter("record1", "key", nil),
			str:         "AddValueGetter( 'record1', 'key', nil )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "record1" {
						if cfg.values[0].val.getter == nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValue( record1, 'key', nil )",
			opt:         AddValue("record1", "key", nil),
			str:         "AddValue( 'record1', 'key', nil )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "record1" {
						if cfg.values[0].val.getter != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValue( record1, key, nil, AsDefault )",
			opt:         AddValue("record1", "key", nil, AsDefault()),
			str:         "AddValue( 'record1', 'key', nil, AsDefault() )",
			check: func(cfg *options) bool {
				if len(cfg.defaults) == 1 {
					if cfg.defaults[0].name == "record1" {
						if cfg.defaults[0].val.getter != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValue( record1, key, nil, AsDefault(false) )",
			opt:         AddValue("record1", "key", nil, AsDefault(false)),
			str:         "AddValue( 'record1', 'key', nil, AsDefault(false) )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "record1" {
						if cfg.values[0].val.getter != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValue( record1, key, nil, WithError(testErr) )",
			opt:         AddValue("record1", "key", nil, WithError(testErr)),
			str:         "AddValue( 'record1', 'key', nil, WithError( 'test err' ) )",
			expectErr:   testErr,
		}, {
			description: "AddBuffer( filename.ext, bytes, AsDefault )",
			opt:         AddBuffer("filename.ext", []byte("bytes"), AsDefault()),
			str:         "AddBuffer( 'filename.ext', []byte, AsDefault() )",
			check: func(cfg *options) bool {
				if len(cfg.defaults) == 1 {
					if cfg.defaults[0].name == "filename.ext" {
						if cfg.defaults[0].buf.getter != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddBuffer( filename.ext, bytes, AsDefault(false) )",
			opt:         AddBuffer("filename.ext", []byte("bytes"), AsDefault(false)),
			str:         "AddBuffer( 'filename.ext', []byte, AsDefault(false) )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "filename.ext" {
						if cfg.values[0].buf.getter != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddDocsJSON()",
			opt:         AddDocsJSON([]byte("invalid json")),
			str:         "WithError( 'input is invalid, AddDocsJSON tree is invalid: failed to unmarshal JSON: invalid character 'i' looking for beginning of value' )",
			expectErr:   ErrInvalidInput,
		}, {
			description: "Options returning an error",
			opt: Options(
				AutoCompile(),      // the options don't matter except that
				WithError(testErr), // there is an error that happens
				AutoCompile(),
			),
			expectErr: testErr,
			str:       "Options( AutoCompile(), WithError( 'test err' ), AutoCompile() )",
		}, {
			description: "Options handles the isDefault case",
			opt: Options(
				WithDecoder(nil),               // the options don't matter except that
				DisableDefaultPackageOptions(), // this option is able to return true
				WithDecoder(nil),               // for the isDefault() case.
			),
			ignore: true,
			str:    "Options( WithDecoder( '' ), DisableDefaultPackageOptions(), WithDecoder( '' ) )",
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			var cfg options

			if tc.initCodecs {
				cfg.encoders = newRegistry[encoder.Encoder]()
				cfg.decoders = newRegistry[decoder.Decoder]()
			}

			var err error
			if len(tc.opts) == 0 {
				err = tc.opt.apply(&cfg)

				s := tc.opt.String()
				assert.Equal(tc.str, s)

				assert.Equal(tc.ignore, tc.opt.ignoreDefaults())
				assert.Equal(tc.ignore, ignoreDefaultOpts([]Option{tc.opt}))
			} else {
				assert.Equal(tc.ignore, ignoreDefaultOpts(tc.opts))
				for _, opt := range tc.opts {
					err = opt.apply(&cfg)
				}
			}

			if tc.expectErr == nil {
				if tc.check != nil {
					assert.True(tc.check(&cfg))
				} else {
					assert.Equal(tc.goal, cfg)
				}
				return
			}

			if !errors.Is(unknownErr, tc.expectErr) {
				assert.ErrorIs(err, tc.expectErr)
				return
			}

		})
	}
}

func TestMergeOverrides(t *testing.T) {
	tests := []struct {
		description string
		in          []map[string]string
		sToC        map[string]string
		expectErr   error
	}{
		{
			description: "single item",
			in: []map[string]string{
				{
					"a": "b",
				},
			},
			sToC: map[string]string{
				"a": "b",
			},
		}, {
			description: "multiple in the array",
			in: []map[string]string{
				{
					"A": "a",
				}, {
					"B": "b",
				},
			},
			sToC: map[string]string{
				"A": "a",
				"B": "b",
			},
		}, {
			description: "duplicated config name",
			in: []map[string]string{
				{
					"A": "a",
				}, {
					"B": "b",
				}, {
					"C": "b",
				},
			},
			sToC: map[string]string{
				"A": "a",
				"B": "b",
				"C": "b",
			},
		}, {
			description: "invalid, duplicated field name",
			in: []map[string]string{
				{
					"A": "a",
				}, {
					"B": "b",
				}, {
					"B": "c",
				},
			},
			expectErr: ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			a, e := mergeOverrides(tc.in)

			if tc.expectErr == nil {
				assert.Equal(a, tc.sToC)
				assert.NoError(e)
				return
			}

			assert.Nil(a)
			assert.ErrorIs(e, tc.expectErr)
		})
	}
}
