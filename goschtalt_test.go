// SPDX-FileCopyrightText: 2022-2023 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/goschtalt/goschtalt/pkg/debug"
	"github.com/goschtalt/goschtalt/pkg/doc"
	"github.com/goschtalt/goschtalt/pkg/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errOpt = errors.New("option error")

func TestNew(t *testing.T) {
	var zeroOpt Option
	tests := []struct {
		description string
		opts        []Option
		expectedErr error
	}{
		{
			description: "A normal case with no options.",
		}, {
			description: "A normal case with options.",
		}, {
			description: "A case with an empty option.",
			opts:        []Option{zeroOpt},
		}, {
			description: "An error case.",
			opts:        []Option{WithError(errOpt)},
			expectedErr: errOpt,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			cfg, err := New(tc.opts...)

			if tc.expectedErr == nil {
				assert.NoError(err)
				assert.NotNil(cfg)
				return
			}

			assert.Error(err)
			assert.ErrorIs(err, tc.expectedErr)
		})
	}
}

func TestCompile(t *testing.T) {
	unknownErr := fmt.Errorf("unknown err")
	testErr := fmt.Errorf("test err")
	remappings := debug.Collect{}

	fs1 := fstest.MapFS{
		"a/1.json": &fstest.MapFile{
			Data: []byte(`{"Hello":"World"}`),
			Mode: 0755,
		},
		"2.json": &fstest.MapFile{
			Data: []byte(`{"Blue":"sky"}`),
			Mode: 0755,
		},
		"3.json": &fstest.MapFile{
			Data: []byte(`{"Hello":"Mr. Blue Sky"}`),
			Mode: 0755,
		},
	}

	fs2 := fstest.MapFS{
		"b/90.json": &fstest.MapFile{
			Data: []byte(`{"Madd": "cat", "Blue": "${thing}"}`),
			Mode: 0755,
		},
	}

	fs3 := fstest.MapFS{
		"b/90.json": &fstest.MapFile{
			Data: []byte(`{"Hello((fail))": "cat", "Blue": "bird"}`),
			Mode: 0755,
		},
	}

	fs4 := fstest.MapFS{
		"b/90.json": &fstest.MapFile{
			Data: []byte(`I'm not valid json!`),
			Mode: 0755,
		},
	}

	fs5 := fstest.MapFS{
		"b/90.txt": &fstest.MapFile{
			Data: []byte(`{"Hello":"Mr. Blue Sky"}`),
			Mode: 0755,
		},
	}

	fs6 := fstest.MapFS{
		"b/90.json": &fstest.MapFile{
			Data: []byte(`{"Hello": "${a}", "Madd": "${b}", "Blue": "${thing}"}`),
			Mode: 0755,
		},
	}

	fs7 := fstest.MapFS{
		"b/90.json": &fstest.MapFile{
			Data: []byte(`{"Hello": "dogs\ncats\nliving together\n"}`),
			Mode: 0755,
		},
	}

	mapper1 := mockExpander{
		f: func(m string) (string, bool) {
			switch m {
			case "thing":
				return "|bird|", true
			}

			return "", false
		},
	}

	mapper2 := mockExpander{
		f: func(m string) (string, bool) {
			switch m {
			case "bird":
				return "jay", true
			}

			return "", false
		},
	}

	// Causes infinite loop
	mapper3 := mockExpander{
		f: func(m string) (string, bool) {
			switch m {
			case "thing":
				return ".${bird}", true
			case "bird":
				return ".${jay}", true
			case "jay":
				return ".${bird}", true
			}

			return "", false
		},
	}

	mapper4 := mockExpander{
		f: func(m string) (string, bool) {
			switch m {
			case "a":
				return "Tom", true
			}
			return "", false
		},
	}

	mapper5 := mockExpander{
		f: func(m string) (string, bool) {
			switch m {
			case "b":
				return "${thing}", true
			}
			return "", false
		},
	}

	type st1 struct {
		Hello string
		Blue  string
		Madd  string
	}

	type st2 struct {
		Hello string
		Blue  string
		Madd  string
		Func  func()
	}

	type st3 struct {
		Dog  string `goschtalt:"Hello"`
		Blue string
		Madd string
	}

	type st4 struct {
		Hello string
	}

	type withAll struct {
		Foo      string
		Duration time.Duration
		T        time.Time
		Func     func(string) string
		Array    []string
	}

	tests := []struct {
		description    string
		skipCompile    bool
		opts           []Option
		key            string
		expectInternal *meta.Object
		expect         any
		files          []string
		expectedRemaps map[string]string
		expectedErr    error
		compare        func(assert *assert.Assertions, a, b any) bool
	}{
		{
			description: "A normal case with options.",
			opts: []Option{
				AddTree(fs1, "."),
				AddTree(fs2, "."),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "${thing}",
				Madd:  "cat",
			},
			files: []string{"1.json", "2.json", "3.json", "90.json"},
		}, {
			description: "A normal case with an encoded buffer.",
			opts: []Option{
				AddBuffer("1.json", []byte(`{"Hello": "Mr. Blue Sky"}`)),
				AddBuffer("2.json", []byte(`{"Blue": "${thing}"}`)),
				AddBuffer("3.json", []byte(`{"Madd": "cat"}`)),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "${thing}",
				Madd:  "cat",
			},
			files: []string{"1.json", "2.json", "3.json"},
		}, {
			description: "A normal case with an encoded buffer function.",
			opts: []Option{
				AddBuffer("3.json", []byte(`{"Madd": "cat"}`)),
				AddBuffer("2.json", []byte(`{"Blue": "${thing}"}`)),
				AddBufferGetter("1.json",
					mockBufferGetter{
						f: func(_ string, _ Unmarshaler) ([]byte, error) {
							return []byte(`{"Hello": "Mr. Blue Sky"}`), nil
						},
					}),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "${thing}",
				Madd:  "cat",
			},
			files: []string{"1.json", "2.json", "3.json"},
		}, {
			description: "A case with an encoded buffer function that looks up something from the tree.",
			opts: []Option{
				AddBuffer("1.json", []byte(`{"Madd": "cat"}`)),
				AddBufferGetter("2.json",
					mockBufferGetter{
						f: func(_ string, un Unmarshaler) ([]byte, error) {
							var s string
							_ = un("Madd", &s)
							return []byte(fmt.Sprintf(`{"Blue": "%s"}`, s)), nil
						},
					}),
				AddBufferGetter("3.json",
					mockBufferGetter{
						f: func(_ string, un Unmarshaler) ([]byte, error) {
							var s string
							_ = un("Blue", &s)
							return []byte(fmt.Sprintf(`{"Hello": "%s"}`, s)), nil
						},
					}),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Hello: "cat",
				Blue:  "cat",
				Madd:  "cat",
			},
			files: []string{"1.json", "2.json", "3.json"},
		}, {
			description: "A case with an encoded buffer that is invalid",
			skipCompile: true,
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				AddBuffer("1.json", []byte(`invalid`)),
			},
			expectedErr: unknownErr,
		}, {
			description: "An encoded buffer can't be decoded.",
			skipCompile: true,
			opts: []Option{
				AddBuffer("1.json", []byte(`invalid`)),
			},
			expectedErr: unknownErr,
		}, {
			description: "An encoded buffer with an option that causes a failure.",
			skipCompile: true,
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				AddBuffer("1.json", []byte(`{}`), WithError(testErr)),
			},
			expectedErr: testErr,
		}, {
			description: "A case with an encoded buffer function that returns an error",
			skipCompile: true,
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				AddBufferGetter("3.json",
					mockBufferGetter{
						f: func(_ string, _ Unmarshaler) ([]byte, error) {
							return nil, unknownErr
						},
					}),
			},
			expectedErr: unknownErr,
		}, {
			description: "A case with an encoded buffer function that returns an invalidly formatted buffer",
			skipCompile: true,
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				AddBufferGetter("3.json",
					mockBufferGetter{
						f: func(_ string, _ Unmarshaler) ([]byte, error) {
							return []byte(`invalid`), nil
						},
					}),
			},
			expectedErr: unknownErr,
		}, {
			description: "A normal case ConfigIs() used.",
			opts: []Option{
				AddBuffer("lower.json", []byte(`{"madd": "cat", "hello": "Mr. Blue Sky", "blue": "${thing}"}`)),
				ConfigIs("flatcase"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "${thing}",
				Madd:  "cat",
			},
			files: []string{"lower.json"},
		}, {
			description: "A normal case ConfigIs() used with overrides.",
			opts: []Option{
				AddBuffer("lower.json", []byte(`{"crazy": "cat", "hello": "Mr. Blue Sky", "blue": "${thing}"}`)),
				ConfigIs("flatcase", map[string]string{"Madd": "crazy"}),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "${thing}",
				Madd:  "cat",
			},
			files: []string{"lower.json"},
		}, {
			description: "An error case where an invalid format is used.",
			opts: []Option{
				ConfigIs("invalid"),
			},
			skipCompile: true,
			expect:      st1{},
			expectedErr: ErrInvalidInput,
		}, {
			description: "An error case where an a duplicate map key is present.",
			opts: []Option{
				ConfigIs("flatcase",
					map[string]string{
						"key": "first",
					},
					map[string]string{
						"key": "second",
					},
				),
			},
			skipCompile: true,
			expect:      st1{},
			expectedErr: ErrInvalidInput,
		}, {
			description: "A normal case with options including expansion.",
			opts: []Option{
				AddTree(fs1, "."),
				AddTree(fs2, "."),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				Expand(mapper1),
				Expand(mapper2, WithDelimiters("|", "|")),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "jay",
				Madd:  "cat",
			},
			files: []string{"1.json", "2.json", "3.json", "90.json"},
		}, {
			description: "A normal case with options including expansion and getters.",
			opts: []Option{
				AddTree(fs1, "."),
				AddTree(fs2, "."),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				AddValueGetter("99", Root,
					mockValueGetter{
						f: func(s string, u Unmarshaler) (any, error) {
							str := struct {
								Blue string
							}{}
							// At this point Blue = "jay"
							err := u("", &str)
							if err != nil {
								return nil, err
							}

							return struct {
								Madd string
							}{
								Madd: str.Blue + " as a ${thing}",
							}, nil
						},
					},
				),
				Expand(mapper1),
				Expand(mapper2, WithDelimiters("|", "|")),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "jay",
				Madd:  "jay as a jay",
			},
			files: []string{"1.json", "2.json", "3.json", "90.json", "99"},
		}, {
			description: "A recursion case where a failure results with a getter.",
			opts: []Option{
				AddTree(fs1, "."),
				AddTree(fs2, "."),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				Expand(mapper3),
				AddValueGetter("99", Root,
					mockValueGetter{
						f: func(s string, u Unmarshaler) (any, error) {
							str := struct {
								Blue string
							}{
								Blue: "${thing}",
							}

							return str, nil
						},
					},
				),
			},
			skipCompile: true,
			expect:      st1{},
			expectedErr: unknownErr,
		}, {
			description: "A normal case with options including env expansion.",
			opts: []Option{
				AddTree(fs1, "."),
				AddTree(fs2, "."),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				ExpandEnv(),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "ocean",
				Madd:  "cat",
			},
			files: []string{"1.json", "2.json", "3.json", "90.json"},
		}, {
			description: "A normal case with multiple expansions.",
			opts: []Option{
				AddTree(fs6, "."),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				// The order is important here, because it shows the exapnsion
				// order works as expected.
				ExpandEnv(),
				Expand(mapper5),
				Expand(mapper4),
			},
			expect: st1{
				Hello: "Tom",
				Blue:  "ocean",
				Madd:  "ocean",
			},
			files: []string{"90.json"},
		}, {
			description: "A normal case with values.",
			opts: []Option{
				AddValue("record1", Root, st1{
					Hello: "Mr. Blue Sky",
					Blue:  "jay",
					Madd:  "cat",
				}),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "jay",
				Madd:  "cat",
			},
			files: []string{"record1"},
		}, {
			description: "A tag remapping case with a pointer to the value.",
			opts: []Option{
				AddValue("record1", Root, &st3{
					Dog:  "Mr. Blue Sky",
					Blue: "jay",
					Madd: "cat",
				}),
			},
			key:    "Hello",
			expect: "Mr. Blue Sky",
			files:  []string{"record1"},
		}, {
			description: "A normal case with values and remapping.",
			opts: []Option{
				AddValue("record1", Root, st1{
					Hello: "Mr. Blue Sky",
					Blue:  "jay",
					Madd:  "cat",
				},
					KeymapMapper(mockMapper{
						f: func(s string) string {
							return strings.ToLower(s)
						},
					}),
					KeymapReport(&remappings),
				),
				DefaultUnmarshalOptions(
					KeymapMapper(mockMapper{
						f: func(s string) string {
							return strings.ToLower(s)
						},
					}),
					KeymapReport(&remappings),
				),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "jay",
				Madd:  "cat",
			},
			expectedRemaps: map[string]string{
				"Blue":  "blue",
				"Hello": "hello",
				"Madd":  "madd",
			},
			files: []string{"record1"},
		}, {
			description: "A normal case with non-serializable values being errors.",
			opts: []Option{
				AddValue("record1", Root,
					st1{
						Hello: "Mr. Blue Sky",
						Blue:  "jay",
						Madd:  "cat",
					},
					FailOnNonSerializable()),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "jay",
				Madd:  "cat",
			},
			files: []string{"record1"},
		}, {
			description: "A normal case with non-serializable values being dropped via Keymap.",
			opts: []Option{
				AddValue("record1", Root,
					st2{
						Hello: "Mr. Blue Sky",
						Blue:  "jay",
						Madd:  "cat",
					},
					FailOnNonSerializable(),
					Keymap(map[string]string{
						"Func": "-",
					}),
				),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "jay",
				Madd:  "cat",
			},
			files: []string{"record1"},
		}, {
			description: "A normal case with non-serializable values being dropped.",
			opts: []Option{
				AddValue("record1", Root,
					st2{
						Hello: "Mr. Blue Sky",
						Blue:  "jay",
						Madd:  "cat",
					}),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "jay",
				Madd:  "cat",
			},
			files: []string{"record1"},
		}, {
			description: "A case with adapters.",
			opts: []Option{
				AddValue("record1", Root,
					withAll{
						Foo:      "string",
						Duration: time.Second,
						T:        time.Date(2022, time.December, 30, 0, 0, 0, 0, time.UTC),
						Func:     strings.ToUpper,
						Array:    []string{"a", "b", "c"},
					},
					adaptTimeToCfg("2006-01-02"),
					adaptDurationToCfg(),
					adaptFuncToCfg(),
				),
				DefaultUnmarshalOptions(
					adaptStringToTime("2006-01-02"),
					adaptStringToDuration(),
					adaptStringToFunc(),
				),
			},
			expectInternal: &meta.Object{
				Origins: []meta.Origin{
					{
						File: "record1",
						Line: 0,
						Col:  0,
					},
				},
				Map: map[string]meta.Object{
					"Array": {
						Origins: []meta.Origin{
							{
								File: "record1",
								Line: 0,
								Col:  0,
							},
						},
						Array: []meta.Object{
							{
								Origins: []meta.Origin{
									{
										File: "record1",
										Line: 0,
										Col:  0,
									},
								},
								Value: "a",
							},
							{
								Origins: []meta.Origin{
									{
										File: "record1",
										Line: 0,
										Col:  0,
									},
								},
								Value: "b",
							},
							{
								Origins: []meta.Origin{
									{
										File: "record1",
										Line: 0,
										Col:  0,
									},
								},
								Value: "c",
							},
						},
					},
					"Duration": {
						Origins: []meta.Origin{
							{
								File: "record1",
								Line: 0,
								Col:  0,
							},
						},
						Value: "1s",
					},
					"Foo": {
						Origins: []meta.Origin{
							{
								File: "record1",
								Line: 0,
								Col:  0,
							},
						},
						Value: "string",
					},
					"Func": {
						Origins: []meta.Origin{
							{
								File: "record1",
								Line: 0,
								Col:  0,
							},
						},
						Value: "upper",
					},
					"T": {
						Origins: []meta.Origin{
							{
								File: "record1",
								Line: 0,
								Col:  0,
							},
						},
						Value: "2022-12-30",
					},
				},
			},
			expect: withAll{
				Foo:      "string",
				Duration: time.Second,
				T:        time.Date(2022, time.December, 30, 0, 0, 0, 0, time.UTC),
				Func:     strings.ToUpper,
				Array:    []string{"a", "b", "c"},
			},
			files: []string{"record1"},
			compare: func(assert *assert.Assertions, z, y any) bool {
				a := z.(withAll)
				b := y.(withAll)

				if a.Func != nil {
					if !assert.NotNil(b.Func) {
						return false
					}
					str := "Random334 String"
					if !assert.Equal(a.Func(str), b.Func(str)) {
						return false
					}
				}

				return assert.Equal(a.Foo, b.Foo) &&
					assert.Equal(a.Duration, b.Duration) &&
					assert.Equal(a.T, b.T) &&
					assert.Equal(a.Array, b.Array)
			},
		}, {

			description: "An empty case.",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{},
		}, {
			description: "A file interpreted.",
			opts: []Option{
				AddFileAs(fs5, "json", "b/90.txt"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st4{
				Hello: "Mr. Blue Sky",
			},
			files: []string{"90.txt"},
		}, {
			description: "An empty set of files.",
			opts: []Option{
				AddFiles(fs1),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{},
		}, {
			description: "A glob of everything.",
			opts: []Option{
				AddFiles(fs1, "*"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Hello: "Mr. Blue Sky",
				Blue:  "sky",
			},
			files: []string{"1.json", "2.json", "3.json"},
		}, {
			description: "A multiline string.",
			opts: []Option{
				AddTree(fs7, "."),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Hello: "dogs\ncats\nliving together\n",
			},
			files: []string{"90.json"},
		}, {
			description: "An invalid file when one must be present.",
			opts: []Option{
				AutoCompile(false),
				AddFile(fs1, "invalid.json"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect:      st1{},
			expectedErr: ErrFileMissing,
		}, {
			description: "AddFile doesn't accept globs since that's multiple files.",
			opts: []Option{
				AutoCompile(false),
				AddFile(fs1, "*"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect:      st1{},
			expectedErr: ErrFileMissing,
		}, {
			description: "A merge failure case.",
			opts: []Option{
				AutoCompile(false),
				AddTree(fs1, "."),
				AddTree(fs3, "."),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect:      st1{},
			expectedErr: meta.ErrConflict,
		}, {
			description: "A decode failure case.",
			opts: []Option{
				AutoCompile(false),
				AddTree(fs1, "."),
				AddTree(fs4, "."),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect:      st1{},
			expectedErr: unknownErr,
		}, {
			description: "A recursion case where a failure results",
			opts: []Option{
				AddTree(fs1, "."),
				AddTree(fs2, "."),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				Expand(mapper3),
			},
			skipCompile: true,
			expect:      st1{},
			expectedErr: unknownErr,
		}, {
			description: "A case where the value adapter returns an error.",
			opts: []Option{
				AutoCompile(false),
				AddValue("record1", Root,
					withAll{
						Foo:      "string",
						Duration: time.Second,
						T:        time.Date(2022, time.December, 30, 0, 0, 0, 0, time.UTC),
					},
					AdaptToCfg(mockAdapterToCfg{
						f: func(reflect.Value) (any, error) { return nil, unknownErr },
					}),
					adaptTimeToCfg("2006-01-02"),
					adaptDurationToCfg(),
				),
			},
			expectedErr: unknownErr,
			files:       []string{"record1"},
		}, {
			description: "A case where the value doesn't have a record name.",
			opts: []Option{
				AddValue("", Root, st1{
					Hello: "Mr. Blue Sky",
					Blue:  "jay",
					Madd:  "cat",
				}),
			},
			skipCompile: true,
			expect:      st1{},
			expectedErr: ErrInvalidInput,
		}, {
			description: "A case where the value function returns an error.",
			opts: []Option{
				AddValueGetter("record", Root,
					mockValueGetter{
						f: func(string, Unmarshaler) (any, error) {
							return nil, testErr
						},
					},
				),
			},
			skipCompile: true,
			expect:      st1{},
			expectedErr: testErr,
		}, {
			description: "A getter returns nil.",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				AddValueGetter("dynamic", Root,
					mockValueGetter{
						f: func(s string, u Unmarshaler) (any, error) {
							return nil, nil
						},
					},
				),
			},
			expect: st1{},
			files:  []string{"dynamic"},
		}, {
			description: "A case where the an option is/becomes an error.",
			opts: []Option{
				AutoCompile(false),
				AddValue("record", Root, st1{
					Hello: "Mr. Blue Sky",
					Blue:  "jay",
					Madd:  "cat",
				},
					// Act like everything is fine the first time through, but then
					// fail the 2nd time to trigger a failure during the compile
					// code.
					testSetError([]error{nil, testErr}),
				),
			},
			expect:      st1{},
			expectedErr: testErr,
		}, {
			description: "A case with non-serializable values producing an error.",
			opts: []Option{
				AutoCompile(false),
				AddValue("record1", Root,
					st2{
						Hello: "Mr. Blue Sky",
						Blue:  "jay",
						Madd:  "cat",
						Func:  func() {},
					},
					FailOnNonSerializable(),
				),
			},
			expect:      st1{},
			expectedErr: meta.ErrNonSerializable,
		}, {
			description: "Make sure the AddFilesHalt doesn't stop if no files are found.",
			opts: []Option{
				AddFilesHalt(fs1, "none.json"),
				AddFiles(fs1, "2.json"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Blue: "sky",
			},
			files: []string{"2.json"},
		}, {
			description: "Make sure the AddFilesHalt stops if files are found.",
			opts: []Option{
				AddFilesHalt(fs1, "2.json"),
				AddFiles(fs1, "3.json"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Blue: "sky",
			},
			files: []string{"2.json"},
		}, {
			description: "Make sure the AddTreeHalt doesn't stop if no files are found.",
			opts: []Option{
				AddTreeHalt(fs1, "none"),
				AddFiles(fs1, "2.json"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Blue: "sky",
			},
			files: []string{"2.json"},
		}, {
			description: "Make sure the AddTreeHalt stops if files are found.",
			opts: []Option{
				AddTreeHalt(fs1, "a"),
				AddTree(fs1, "3.json"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			expect: st1{
				Hello: "World",
			},
			files: []string{"1.json"},
		}, {
			description: "Ensure the HintDecoder() can find one.",
			opts: []Option{
				HintDecoder("json", "http://github.com/goschtalt/json-decoder", "json"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
		}, {
			description: "Ensure the HintDecoder() can find when they are missing.",
			opts: []Option{
				HintDecoder("dogs", "http://github.com/goschtalt/dogs-decoder", "dogs"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
			skipCompile: true,
			expectedErr: ErrHint,
		}, {
			description: "Ensure the HintDecoder() can handle a partial success.",
			opts: []Option{
				HintDecoder("json_dogs", "http://github.com/goschtalt/dogs-decoder", "dogs", "json"),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			},
		}, {
			description: "Ensure the HintEncoder() can find one.",
			opts: []Option{
				HintEncoder("json", "http://github.com/goschtalt/json-decoder", "json"),
				WithEncoder(&testEncoder{extensions: []string{"json"}}),
			},
		}, {
			description: "Ensure the HintEncoder() can find when they are missing.",
			opts: []Option{
				HintEncoder("dogs", "http://github.com/goschtalt/dogs-decoder", "dogs"),
				WithEncoder(&testEncoder{extensions: []string{"json"}}),
			},
			skipCompile: true,
			expectedErr: ErrHint,
		}, {
			description: "Ensure the HintEncoder() can handle a partial success.",
			opts: []Option{
				HintEncoder("json_dogs", "http://github.com/goschtalt/dogs-decoder", "dogs", "json"),
				WithEncoder(&testEncoder{extensions: []string{"json"}}),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			t.Setenv("thing", "ocean")

			remappings.Mapping = make(map[string]string)

			cfg, err := New(tc.opts...)

			if !tc.skipCompile {
				require.NoError(err)
				err = cfg.Compile()
			}

			var tell string
			if cfg != nil {
				tell = cfg.Explain().String()
			}

			if tc.expectedErr == nil {
				assert.NoError(err)
				require.NotNil(cfg)

				if tc.expectInternal != nil {
					if !assert.Equal(*tc.expectInternal, cfg.tree) { // nolint:unused,staticcheck
						//pp.SetDefaultMaxDepth(10)
						//pp.Print(cfg.tree)
					}
				}
				if tc.expect != nil {
					want := reflect.Zero(reflect.TypeOf(tc.expect)).Interface()
					err = cfg.Unmarshal(tc.key, &want)
					require.NoError(err)

					if tc.compare != nil {
						assert.True(tc.compare(assert, tc.expect, want))
					} else {
						assert.Equal(tc.expect, want)
					}
				}

				// check the file order
				if tc.files == nil {
					assert.Empty(cfg.records)
				} else {
					assert.Equal(tc.files, cfg.records)
				}

				assert.NotEmpty(tell)

				if tc.expectedRemaps == nil {
					assert.Empty(remappings.Mapping)
				} else {
					assert.Equal(tc.expectedRemaps, remappings.Mapping)
				}
				return
			}

			assert.Error(err)
			if !errors.Is(unknownErr, tc.expectedErr) {
				assert.ErrorIs(err, tc.expectedErr)
			}

			if !tc.skipCompile {
				// check the file order is correct
				assert.Empty(cfg.records)
				assert.NotEmpty(tell)
			}
		})
	}
}

func TestOrderList(t *testing.T) {
	tests := []struct {
		description string
		in          []string
		expect      []string
	}{
		{
			description: "An empty list",
		}, {
			description: "A simple list",
			in: []string{
				"9.json",
				"3.txt",
				"1.json",
				"2.json",
			},
			expect: []string{
				"1.json",
				"2.json",
				"9.json",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			cfg, err := New(
				AutoCompile(false),
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
			)
			require.NotNil(cfg)
			require.NoError(err)

			got := cfg.OrderList(tc.in)

			assert.Equal(tc.expect, got)
		})
	}
}

func TestHash(t *testing.T) {
	testErr := fmt.Errorf("test err")
	tests := []struct {
		description string
		opts        []Option
		expect      []byte
		expectedErr error
	}{
		{
			description: "The default hasher returns []bytes{}.",
			opts: []Option{
				AutoCompile(false),
			},
		}, {
			description: "A simple hasher",
			opts: []Option{
				SetHasher(HasherFunc(
					func(o any) ([]byte, error) {
						return []byte{0x01, 0x02}, nil
					},
				)),
			},
			expect: []byte{0x01, 0x02},
		}, {
			description: "A simple hasher that always errors",
			opts: []Option{
				SetHasher(HasherFunc(
					func(o any) ([]byte, error) {
						return nil, testErr
					},
				)),
			},
			expectedErr: testErr,
		},
		/*
			{
				description: "Example using hashstructure.Hash().",
				opts: []Option{
					AddValue("rec", Root, map[string]string{"hello": "world"}),
					SetHasher(HasherFunc(
						func(o any) ([]byte, error) {
							h, err := hashstructure.Hash(o, nil)
							if err != nil {
								return nil, err
							}
							b := make([]byte, 8)
							binary.LittleEndian.PutUint64(b, h)
							return b, nil
						},
					)),
				},
				expect: []byte{0xae, 0x06, 0x83, 0xe2, 0x50, 0x82, 0x99, 0xd4},
			},
		*/
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			cfg, err := New(tc.opts...)

			if tc.expectedErr != nil {
				require.Nil(cfg)
				require.ErrorIs(err, tc.expectedErr)
				return
			}

			require.NotNil(cfg)
			require.NoError(err)

			got := cfg.Hash()

			assert.Equal(tc.expect, got)
		})
	}
}

func TestCompiledAt(t *testing.T) {
	tests := []struct {
		description string
		opts        []Option
		timeZero    bool
	}{
		{
			description: "An empty list",
			timeZero:    true,
			opts: []Option{
				AutoCompile(false),
			},
		}, {
			description: "A simple list",
			timeZero:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			cfg, err := New(tc.opts...)
			require.NotNil(cfg)
			require.NoError(err)

			got := cfg.CompiledAt()

			if tc.timeZero {
				assert.Equal(time.Time{}, got)
			} else {
				assert.NotEqual(time.Time{}, got)
			}
		})
	}
}

func TestGetTree(t *testing.T) {
	tests := []struct {
		description string
		opts        []Option
		mapsize     int
	}{
		{
			description: "An empty list",
		}, {
			description: "A simple list",
			opts: []Option{
				AddValue("record1", Root,
					struct {
						Entry string
					}{
						Entry: "side door",
					},
				),
			},
			mapsize: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			cfg, err := New(tc.opts...)
			require.NotNil(cfg)
			require.NoError(err)

			got := cfg.GetTree()
			require.NotNil(got)
			assert.Equal(tc.mapsize, len(got.Map))
		})
	}
}

func TestSetMaxExpansions(t *testing.T) {
	tests := []struct {
		description string
		opts        []Option
		expansions  int
		expectedErr error
	}{
		{
			description: "Set 100",
			opts: []Option{
				SetMaxExpansions(100),
			},
			expansions: 100,
		}, {
			description: "Set -100",
			opts: []Option{
				SetMaxExpansions(-100),
			},
			expectedErr: ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			cfg, err := New(tc.opts...)

			if tc.expectedErr != nil {
				assert.ErrorIs(err, tc.expectedErr)
				return
			}

			require.NotNil(cfg)
			require.NoError(err)

			assert.Equal(tc.expansions, cfg.opts.exapansionMax)
		})
	}
}

func TestDocument(t *testing.T) {
	type st1 struct {
		A []string
	}
	type st2 struct {
		B []string
	}
	type st struct {
		Hello string
		Blue  string
		Madd  string
		Inner st2
		Other []st1
	}

	commonOpts := []Option{
		AddValue("record1", Root, st{
			Hello: "Mr. Blue Sky",
			Blue:  "jay",
			Madd:  "cat",
			Inner: st2{
				B: []string{"one", "two", "three"},
			},
			Other: []st1{
				{A: []string{"four", "five"}},
				{A: []string{"six", "seven"}},
			},
		}, AsDefault()),
		AddValue("record2", Root, st{
			Hello: "Mr. Man",
			Blue:  "jay",
			Madd:  "cat",
		}),
		AddDocs(doc.Object{
			Type: doc.TYPE_ROOT,
			Children: map[string]doc.Object{
				"Hello": {
					Name: "Hello",
					Doc:  "Hello documentation.",
					Type: doc.TYPE_STRING,
				},
				"Inner": {
					Name: "Inner",
					Doc:  "Inner documentation.",
					Type: doc.TYPE_STRUCT,
					Children: map[string]doc.Object{
						"B": {
							Name: "B",
							Doc:  "B documentation.",
							Type: doc.TYPE_ARRAY,
							Children: map[string]doc.Object{
								doc.NAME_ARRAY: {
									Name: doc.NAME_ARRAY,
									Doc:  "Array documentation.",
									Type: doc.TYPE_STRING,
								},
							},
						},
					},
				},
			},
		}),
	}

	tests := []struct {
		description string
		opts        []Option
		expect      string
	}{
		{
			description: "A simple case with a document.",
			opts:        commonOpts,
			expect: `---
Blue: jay

# Hello documentation.
# type: <string>
Hello: Mr. Man

# Inner documentation.
# type: <struct>
Inner:

  # B documentation.
  # type: array of <string>
  B:

    # Array documentation.
    # type: <string>
    - one
    - two
    - three
Madd: cat
Other:
  -
    A:
      - four
      - five
  -
    A:
      - six
      - seven

`,
		}, {
			description: "Only the default values",
			opts:        append(commonOpts, DefaultMarshalOptions(OnlyDefaults())),
			expect: `---
Blue: jay

# Hello documentation.
# type: <string>
Hello: Mr. Blue Sky

# Inner documentation.
# type: <struct>
Inner:

  # B documentation.
  # type: array of <string>
  B:

    # Array documentation.
    # type: <string>
    - one
    - two
    - three
Madd: cat
Other:
  -
    A:
      - four
      - five
  -
    A:
      - six
      - seven

`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			cfg, err := New(tc.opts...)
			require.NoError(err)
			require.NotNil(cfg)

			doc, err := cfg.Marshal(IncludeDocumentation())
			require.NoError(err)

			if !assert.Equal(tc.expect, string(doc)) {
				fmt.Println("Got:")
				fmt.Println(string(doc))
			}
		})
	}
}

func TestWithDocsJSON(t *testing.T) {
	type st1 struct {
		A []string
	}
	type st2 struct {
		B []string
	}
	type st struct {
		Hello string
		Blue  string
		Madd  string
		Inner st2
		Other []st1
	}

	commonOptsJSON := []Option{
		DefaultMarshalOptions(FormatAsYAML()),
		AddValue("record1", Root, st{
			Hello: "Mr. Blue Sky",
			Blue:  "jay",
			Madd:  "cat",
			Inner: st2{
				B: []string{"one", "two", "three"},
			},
			Other: []st1{
				{A: []string{"four", "five"}},
				{A: []string{"six", "seven"}},
			},
		}, AsDefault()),
		AddValue("record2", Root, st{
			Hello: "Mr. Man",
			Blue:  "jay",
			Madd:  "cat",
		}),
		AddDocsJSON([]byte(`{
			"Type": "<root>",
			"Children": {
				"Hello": {
					"Name": "Hello",
					"Doc": "Hello documentation.",
					"Type": "<string>"
				},
				"Inner": {
					"Name": "Inner",
					"Doc": "Inner documentation.",
					"Type": "<struct>",
					"Children": {
						"B": {
							"Name": "B",
							"Doc": "B documentation.",
							"Type": "<array>",
							"Children": {
								"<array>": {
									"Name": "<array>",
									"Doc": "Array documentation.",
									"Type": "<string>"
								}
							}
						}
					}
				}
			}
		}`)),
	}

	tests := []struct {
		description string
		opts        []Option
		expect      string
	}{
		{
			description: "A simple case with a document from JSON.",
			opts:        commonOptsJSON,
			expect: `---
Blue: jay

# Hello documentation.
# type: <string>
Hello: Mr. Man

# Inner documentation.
# type: <struct>
Inner:

  # B documentation.
  # type: array of <string>
  B:

    # Array documentation.
    # type: <string>
    - one
    - two
    - three
Madd: cat
Other:
  -
    A:
      - four
      - five
  -
    A:
      - six
      - seven

`,
		}, {
			description: "Only the default values from JSON docs",
			opts:        append(commonOptsJSON, DefaultMarshalOptions(OnlyDefaults())),
			expect: `---
Blue: jay

# Hello documentation.
# type: <string>
Hello: Mr. Blue Sky

# Inner documentation.
# type: <struct>
Inner:

  # B documentation.
  # type: array of <string>
  B:

    # Array documentation.
    # type: <string>
    - one
    - two
    - three
Madd: cat
Other:
  -
    A:
      - four
      - five
  -
    A:
      - six
      - seven

`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			cfg, err := New(tc.opts...)
			require.NoError(err)
			require.NotNil(cfg)

			doc, err := cfg.Marshal(IncludeDocumentation())
			require.NoError(err)

			if !assert.Equal(tc.expect, string(doc)) {
				fmt.Println("Got:")
				fmt.Println(string(doc))
			}
		})
	}
}
