// SPDX-FileCopyrightText: 2022-2023 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/goschtalt/goschtalt/pkg/decoder"
	"github.com/goschtalt/goschtalt/pkg/encoder"
	"github.com/k0kubun/pp/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptions(t *testing.T) {
	unknown := errors.New("unknown")

	testErr := errors.New("test err")
	fs := fstest.MapFS{}
	abs := fstest.MapFS{}
	rel := fstest.MapFS{}
	absFile, err := filepath.Abs("path1")
	require.NoError(t, err)
	list := []string{"zeta", "alpha", "19beta", "19alpha", "4tango",
		"1alpha", "7alpha", "bravo", "7alpha10", "7alpha2", "7alpha0"}

	retBuf := func(name string, un UnmarshalFunc) ([]byte, error) {
		return []byte(name), nil
	}

	sortCheck := func(cfg *options, want []string) bool {
		if cfg.sorter == nil {
			return false
		}

		got := list

		sorter := func(a []string) {
			sort.SliceStable(a, func(i, j int) bool {
				return cfg.sorter(a[i], a[j])
			})
		}

		sorter(got)

		return reflect.DeepEqual(got, want)
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
						paths: []string{absFile},
					},
					{
						fs:    rel,
						paths: []string{"./path2"},
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
			goal: options{
				autoCompile: true,
			},
		}, {
			description: "AutoCompile(false)",
			opt:         AutoCompile(false),
			str:         "AutoCompile( false )",
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
			description: "SortRecordsCustomFn( nil )",
			opt:         SortRecordsCustomFn(nil),
			str:         "SortRecordsCustomFn( nil )",
			expectErr:   ErrInvalidInput,
		}, {
			description: "SortRecordsCustomFn( '(reverse)' )",
			opt: SortRecordsCustomFn(func(a, b string) bool {
				return a > b
			}),
			str: "SortRecordsCustomFn( custom )",
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
				return reflect.DeepEqual([]string{"json", "yml"},
					cfg.encoders.extensions())
			},
		}, {
			description: "WithEncoder( foo )",
			opt:         WithEncoder(&testEncoder{extensions: []string{"foo"}}),
			str:         "WithEncoder( 'foo' )",
			initCodecs:  true,
			check: func(cfg *options) bool {
				return reflect.DeepEqual([]string{"foo"},
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
				return reflect.DeepEqual([]string{"json", "yml"},
					cfg.decoders.extensions())
			},
		}, {
			description: "WithDecoder( foo )",
			opt:         WithDecoder(&testDecoder{extensions: []string{"foo"}}),
			str:         "WithDecoder( 'foo' )",
			initCodecs:  true,
			check: func(cfg *options) bool {
				return reflect.DeepEqual([]string{"foo"},
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
			description: "DefaultMarshalOptions( RedactSecrets(true), IncludeOrigins(true), FormatAs(foo) )",
			opt:         DefaultMarshalOptions(RedactSecrets(true), IncludeOrigins(true), FormatAs("foo")),
			str:         "DefaultMarshalOptions( RedactSecrets(), IncludeOrigins(), FormatAs('foo') )",
			goal: options{
				marshalOptions: []MarshalOption{redactSecretsOption(true), includeOriginsOption(true), formatAsOption("foo")},
			},
		}, {
			description: "DefaultMarshalOptions( RedactSecrets(false), IncludeOrigins(false) )",
			opt:         DefaultMarshalOptions(RedactSecrets(false), IncludeOrigins(false)),
			str:         "DefaultMarshalOptions( RedactSecrets(false), IncludeOrigins(false) )",
			goal: options{
				marshalOptions: []MarshalOption{redactSecretsOption(false), includeOriginsOption(false)},
			},
		}, {
			description: "DefaultMarshalOptions( RedactSecrets(), IncludeOrigins() )",
			opt:         DefaultMarshalOptions(RedactSecrets(), IncludeOrigins()),
			str:         "DefaultMarshalOptions( RedactSecrets(), IncludeOrigins() )",
			goal: options{
				marshalOptions: []MarshalOption{redactSecretsOption(true), includeOriginsOption(true)},
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
			description: "DefaultUnmarshalOptions( DecodeHook() )",
			opts: []Option{
				DefaultUnmarshalOptions(DecodeHook(nil)),
			},
			goal: options{
				unmarshalOptions: []UnmarshalOption{
					&decodeHookOption{},
				},
			},
		}, {
			description: "DefaultUnmarshalOptions( most )",
			opt: DefaultUnmarshalOptions(
				DecodeHook(nil),
				ErrorUnused(),
				ErrorUnset(),
				WeaklyTypedInput(),
				TagName("tag"),
				IgnoreUntaggedFields(),
			),
			goal: options{
				unmarshalOptions: []UnmarshalOption{
					&decodeHookOption{},
					errorUnusedOption(true),
					errorUnsetOption(true),
					weaklyTypedInputOption(true),
					tagNameOption("tag"),
					ignoreUntaggedFieldsOption(true),
				},
			},
			str: "DefaultUnmarshalOptions( DecodeHook(nil), ErrorUnused(), ErrorUnset(), WeaklyTypedInput(), TagName('tag'), IgnoreUntaggedFields() )",
		}, {
			description: "DefaultUnmarshalOptions( most )",
			opt: DefaultUnmarshalOptions(
				ErrorUnused(false),
				ErrorUnset(false),
				WeaklyTypedInput(false),
				IgnoreUntaggedFields(false),
			),
			goal: options{
				unmarshalOptions: []UnmarshalOption{
					errorUnusedOption(false),
					errorUnsetOption(false),
					weaklyTypedInputOption(false),
					ignoreUntaggedFieldsOption(false),
				},
			},
			str: "DefaultUnmarshalOptions( ErrorUnused(false), ErrorUnset(false), WeaklyTypedInput(false), IgnoreUntaggedFields(false) )",
		}, {
			description: "DefaultUnmarshalOptions( DecodeHook() )",
			opt: DefaultUnmarshalOptions(
				DecodeHook(func() {}),
			),
			check: func(cfg *options) bool {
				return len(cfg.unmarshalOptions) == 1
			},
			str: "DefaultUnmarshalOptions( DecodeHook(custom) )",
		}, {
			description: "DefaultValueOptions()",
			opt:         DefaultValueOptions(),
			str:         "DefaultValueOptions()",
		}, {
			description: "DefaultValueOptions( DecodeHook() )",
			opt:         DefaultValueOptions(DecodeHook(nil)),
			goal: options{
				valueOptions: []ValueOption{
					&decodeHookOption{},
				},
			},
			str: "DefaultValueOptions( DecodeHook(nil) )",
		}, {
			description: "AddBuffer( filename.ext, bytes )",
			opt:         AddBuffer("filename.ext", []byte("bytes")),
			str:         "AddBuffer( 'filename.ext', []byte )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "filename.ext" {
						if cfg.values[0].buf.fn != nil {
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
						if cfg.values[0].buf.fn != nil {
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
			expectErr:   unknown,
		}, {
			description: "AddBufferFn( filename.ext, nil )",
			opt:         AddBufferFn("filename.ext", nil),
			str:         "AddBufferFn( 'filename.ext', nil )",
			expectErr:   unknown,
		}, {
			description: "AddBufferFn( filename.ext, bytes )",
			opt:         AddBufferFn("filename.ext", retBuf),
			str:         "AddBufferFn( 'filename.ext', custom )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "filename.ext" {
						if cfg.values[0].buf.fn != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValueFn( record1, '', func )",
			opt:         AddValueFn("record1", Root, func(_ string, un UnmarshalFunc) (any, error) { return nil, nil }),
			str:         "AddValueFn( 'record1', '', custom, none )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "record1" {
						if cfg.values[0].val.fn != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValueFn( record1, 'key', nil )",
			opt:         AddValueFn("record1", "key", nil),
			str:         "AddValueFn( 'record1', 'key', '', none )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "record1" {
						if cfg.values[0].val.fn == nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValue( record1, 'key', nil )",
			opt:         AddValue("record1", "key", nil),
			str:         "AddValue( 'record1', 'key', none )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "record1" {
						if cfg.values[0].val.fn != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValue( record1, key, nil, AsDefault )",
			opt:         AddValue("record1", "key", nil, AsDefault()),
			str:         "AddValue( 'record1', 'key', AsDefault() )",
			check: func(cfg *options) bool {
				if len(cfg.defaults) == 1 {
					if cfg.defaults[0].name == "record1" {
						if cfg.defaults[0].val.fn != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValue( record1, key, nil, AsDefault(false) )",
			opt:         AddValue("record1", "key", nil, AsDefault(false)),
			str:         "AddValue( 'record1', 'key', AsDefault(false) )",
			check: func(cfg *options) bool {
				if len(cfg.values) == 1 {
					if cfg.values[0].name == "record1" {
						if cfg.values[0].val.fn != nil {
							return true
						}
					}
				}
				return false
			},
		}, {
			description: "AddValue( record1, key, nil, WithError(testErr) )",
			opt:         AddValue("record1", "key", nil, WithError(testErr)),
			str:         "AddValue( 'record1', 'key', WithError( 'test err' ) )",
			expectErr:   testErr,
		}, {
			description: "AddBuffer( filename.ext, bytes, AsDefault )",
			opt:         AddBuffer("filename.ext", []byte("bytes"), AsDefault()),
			str:         "AddBuffer( 'filename.ext', []byte, AsDefault() )",
			check: func(cfg *options) bool {
				if len(cfg.defaults) == 1 {
					if cfg.defaults[0].name == "filename.ext" {
						if cfg.defaults[0].buf.fn != nil {
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
						if cfg.values[0].buf.fn != nil {
							return true
						}
					}
				}
				return false
			},
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
					assert.True(reflect.DeepEqual(tc.goal, cfg))
					if !reflect.DeepEqual(tc.goal, cfg) {
						pp.Printf("Want:\n%s\n", tc.goal)
						pp.Printf("Got:\n%s\n", cfg)
					}
				}
				return
			}

			if !errors.Is(unknown, tc.expectErr) {
				assert.ErrorIs(err, tc.expectErr)
				return
			}

		})
	}
}
