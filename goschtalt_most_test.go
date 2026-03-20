// SPDX-FileCopyrightText: 2022-2023 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

//go:build !windows && !android

package goschtalt

import (
	"errors"
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/goschtalt/goschtalt/pkg/debug"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdCfgLayoutNotWin(t *testing.T) {
	assert := assert.New(t)
	got := StdCfgLayout("name")

	// Only make sure the other things are called.  Other tests ensure the
	// functionality works.
	assert.NotNil(got)
}

func TestPopulate(t *testing.T) {
	tests := []struct {
		description string
		home        string
		homeSet     bool
	}{
		{
			description: "no HOME set",
		}, {
			home:    "example",
			homeSet: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)

			var s stdLocations

			t.Setenv("HOME", tc.home)

			s.Populate("foo")

			assert.NotNil(s.local)
			assert.NotNil(s.root)
			if tc.homeSet {
				assert.NotNil(s.home)
			} else {
				assert.Nil(s.home)
			}
			assert.NotNil(s.etc)
		})
	}
}

func TestCompileNotWin(t *testing.T) {
	unknownErr := fmt.Errorf("unknown err")
	remappings := debug.Collect{}

	none := fstest.MapFS{}

	local := fstest.MapFS{
		"1.json": &fstest.MapFile{
			Data: []byte(`{"Status": "local - wanted"}`),
			Mode: 0755,
		},
		"dir/2.json": &fstest.MapFile{
			Data: []byte(`{"Other": "local - wanted"}`),
			Mode: 0755,
		},
		"example.json": &fstest.MapFile{
			Data: []byte(`{"Status": "local - default name wanted"}`),
			Mode: 0755,
		},
	}

	localTree := fstest.MapFS{
		"1.json": &fstest.MapFile{
			Data: []byte(`{"Status": "local - wanted"}`),
			Mode: 0755,
		},
		"dir/2.json": &fstest.MapFile{
			Data: []byte(`{"Other": "local - wanted"}`),
			Mode: 0755,
		},
		"conf.d/2.json": &fstest.MapFile{
			Data: []byte(`{"Other": "local - default tree wanted"}`),
			Mode: 0755,
		},
	}

	home := fstest.MapFS{
		"example.json": &fstest.MapFile{
			Data: []byte(`{"Status": "home - wanted"}`),
			Mode: 0755,
		},
	}

	homeTree := fstest.MapFS{
		"conf.d/2.json": &fstest.MapFile{
			Data: []byte(`{"Other": "home - tree wanted"}`),
			Mode: 0755,
		},
	}

	etc := fstest.MapFS{
		"example.json": &fstest.MapFile{
			Data: []byte(`{"Status": "etc - wanted"}`),
			Mode: 0755,
		},
	}

	etcTree := fstest.MapFS{
		"conf.d/2.json": &fstest.MapFile{
			Data: []byte(`{"Other": "etc - tree wanted"}`),
			Mode: 0755,
		},
	}

	never := fstest.MapFS{
		"example.json": &fstest.MapFile{
			Data: []byte(`{"Status": "never wanted"}`),
			Mode: 0755,
		},
	}

	type st struct {
		Status string
		Other  string
	}

	tests := []struct {
		description    string
		compileOption  bool
		opts           []Option
		want           any
		key            string
		expect         any
		files          []string
		expectedRemaps map[string]string
		expectedErr    error
		compare        func(assert *assert.Assertions, a, b any) bool
	}{
		{
			description: "local - one file in the list",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("example", []string{"./1.json"}, stdLocations{
					local: local,
					root:  none,
					home:  home,
					etc:   etc,
				}),
				AddFile(never, "example.json"),
			},
			want: st{},
			expect: st{
				Status: "local - wanted",
			},
			files: []string{"1.json"},
		}, {
			description: "local - default file",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("example", nil, stdLocations{
					local: local,
					root:  none,
					home:  home,
					etc:   etc,
				}),
				AddFile(never, "example.json"),
			},
			want: st{},
			expect: st{
				Status: "local - default name wanted",
			},
			files: []string{"example.json"},
		}, {
			description: "local - default file, empty list",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("example", []string{}, stdLocations{
					local: local,
					root:  none,
					home:  home,
					etc:   etc,
				}),
				AddFile(never, "example.json"),
			},
			want: st{},
			expect: st{
				Status: "local - default name wanted",
			},
			files: []string{"example.json"},
		}, {
			description: "local - default file, list of empty strings",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("example", []string{"", ""}, stdLocations{
					local: local,
					root:  none,
					home:  home,
					etc:   etc,
				}),
				AddFile(never, "example.json"),
			},
			want: st{},
			expect: st{
				Status: "local - default name wanted",
			},
			files: []string{"example.json"},
		}, {
			description: "local - one dir in the list",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("example", []string{"./dir"}, stdLocations{
					local: localTree,
					root:  none,
					home:  home,
					etc:   etc,
				}),
				AddFile(never, "example.json"),
			},
			want: st{},
			expect: st{
				Other: "local - wanted",
			},
			files: []string{"2.json"},
		}, {
			description: "local - a file and dir in the list",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("example", []string{"./dir", "./1.json"}, stdLocations{
					local: local,
					root:  none,
					home:  home,
					etc:   etc,
				}),
				AddFile(never, "example.json"),
			},
			want: st{},
			expect: st{
				Status: "local - wanted",
				Other:  "local - wanted",
			},
			files: []string{"1.json", "2.json"},
		}, {
			description: "home - a file in the home",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("example", nil, stdLocations{
					local: none,
					root:  none,
					home:  home,
					etc:   etc,
				}),
				AddFile(never, "example.json"),
			},
			want: st{},
			expect: st{
				Status: "home - wanted",
			},
			files: []string{"example.json"},
		}, {
			description: "home - one file in the dir",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("example", nil, stdLocations{
					local: none,
					root:  none,
					home:  homeTree,
					etc:   etc,
				}),
				AddFile(never, "example.json"),
			},
			want: st{},
			expect: st{
				Other: "home - tree wanted",
			},
			files: []string{"2.json"},
		}, {
			description: "etc - a file in the home",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("example", nil, stdLocations{
					local: none,
					root:  none,
					home:  none,
					etc:   etc,
				}),
				AddFile(never, "example.json"),
			},
			want: st{},
			expect: st{
				Status: "etc - wanted",
			},
			files: []string{"example.json"},
		}, {
			description: "etc - one file in the dir",
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("example", nil, stdLocations{
					local: none,
					root:  none,
					home:  none,
					etc:   etcTree,
				}),
				AddFile(never, "example.json"),
			},
			want: st{},
			expect: st{
				Other: "etc - tree wanted",
			},
			files: []string{"2.json"},
		}, {
			description:   "invalid appname",
			compileOption: true,
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("", nil, stdLocations{}),
				AddFile(never, "example.json"),
			},
			want:        st{},
			expect:      st{},
			expectedErr: ErrInvalidInput,
		}, {
			description:   "invalid appname",
			compileOption: true,
			opts: []Option{
				WithDecoder(&testDecoder{extensions: []string{"json"}}),
				nonWinStdCfgLayout("foo/bar", nil, stdLocations{}),
				AddFile(never, "example.json"),
			},
			want:        st{},
			expect:      st{},
			expectedErr: ErrInvalidInput,
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			t.Setenv("thing", "ocean")

			remappings.Mapping = make(map[string]string)

			cfg, err := New(tc.opts...)

			if !tc.compileOption {
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
				want := tc.want
				err = cfg.Unmarshal(tc.key, &want)
				require.NoError(err)

				if tc.compare != nil {
					assert.True(tc.compare(assert, tc.expect, want))
				} else {
					assert.Equal(tc.expect, want)
				}

				// check the file order
				assert.Equal(tc.files, cfg.records)

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

			if !tc.compileOption {
				// check the file order is correct
				assert.Empty(cfg.explain.Records)
				assert.NotEmpty(tell)
			}
		})
	}
}

func TestStdCfgLayoutUserFilesWithDashPrefix(t *testing.T) {
	// This test verifies that user files that happen to start with "N-"
	// are not incorrectly modified. The sortKey mechanism ensures correct
	// ordering while preserving the original filename.

	t.Run("user files with N- pattern via AddFiles", func(t *testing.T) {
		localWithDashPrefixedFiles := fstest.MapFS{
			"1-config.json": &fstest.MapFile{
				Data: []byte(`{"Status": "user file 1-config.json"}`),
				Mode: 0755,
			},
			"2-override.json": &fstest.MapFile{
				Data: []byte(`{"Status": "user file 2-override.json"}`),
				Mode: 0755,
			},
		}

		type st struct {
			Status string
		}

		assert := assert.New(t)
		require := require.New(t)

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			AddFiles(localWithDashPrefixedFiles, "1-config.json", "2-override.json"),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		var got st
		err = cfg.Unmarshal("", &got)
		require.NoError(err)

		// Verify the value is correct
		assert.Equal("user file 2-override.json", got.Status)

		// Verify file names are preserved correctly in the explanation
		exp := cfg.Explain()
		require.Len(exp.Records, 2)
		assert.Equal("1-config.json", exp.Records[0].Name)
		assert.Equal("2-override.json", exp.Records[1].Name)
	})

	t.Run("user files with N- pattern via StdCfgLayout", func(t *testing.T) {
		// User has a file named "1-production.json" in conf.d
		localWithPrefixedConfD := fstest.MapFS{
			"example.json": &fstest.MapFile{
				Data: []byte(`{"Status": "base config", "Count": 1}`),
				Mode: 0755,
			},
			"conf.d/1-production.json": &fstest.MapFile{
				Data: []byte(`{"Status": "production override", "Count": 2}`),
				Mode: 0755,
			},
		}

		type st struct {
			Status string
			Count  int
		}

		assert := assert.New(t)
		require := require.New(t)

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: localWithPrefixedConfD,
				home:  nil,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		var got st
		err = cfg.Unmarshal("", &got)
		require.NoError(err)

		// Verify values are correct
		assert.Equal("production override", got.Status)
		assert.Equal(2, got.Count)

		// Verify file names are preserved correctly (not "1-1-production.json")
		exp := cfg.Explain()
		require.Len(exp.Records, 2)
		assert.Equal("example.json", exp.Records[0].Name)
		assert.Equal("1-production.json", exp.Records[1].Name)
	})
}

func TestStdCfgLayoutNestedConfD(t *testing.T) {
	// This test verifies that nested config files in conf.d subdirectories
	// are properly detected and loaded. The hasDecodableFiles() check must
	// use recurse: true to match the actual loading behavior.

	t.Run("nested conf.d files are detected and loaded", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		// Local has no appName.* file, only a nested file in conf.d
		localWithNestedConfD := fstest.MapFS{
			"conf.d/subdirectory/app.json": &fstest.MapFile{
				Data: []byte(`{"Status": "nested local config"}`),
				Mode: 0755,
			},
		}

		// Home has a top-level file
		homeWithConfig := fstest.MapFS{
			"example.json": &fstest.MapFile{
				Data: []byte(`{"Status": "home config"}`),
				Mode: 0755,
			},
		}

		type st struct {
			Status string
		}

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: localWithNestedConfD,
				home:  homeWithConfig,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		var got st
		err = cfg.Unmarshal("", &got)
		require.NoError(err)

		// Should use local (higher priority) even though file is nested
		assert.Equal("nested local config", got.Status)
	})

	t.Run("nested conf.d files prevent fallback to lower priority", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		// Home has nested conf.d file
		homeWithNestedConfD := fstest.MapFS{
			"conf.d/deep/nested/app.json": &fstest.MapFile{
				Data: []byte(`{"Status": "nested home config"}`),
				Mode: 0755,
			},
		}

		// Etc has a top-level file
		etcWithConfig := fstest.MapFS{
			"example.json": &fstest.MapFile{
				Data: []byte(`{"Status": "etc config"}`),
				Mode: 0755,
			},
		}

		type st struct {
			Status string
		}

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: fstest.MapFS{},
				home:  homeWithNestedConfD,
				root:  fstest.MapFS{},
				etc:   etcWithConfig,
			}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		var got st
		err = cfg.Unmarshal("", &got)
		require.NoError(err)

		// Should use home, not fall back to etc
		assert.Equal("nested home config", got.Status)
	})
}

func TestOrderListWithStdCfgLayout(t *testing.T) {
	// This test verifies that OrderList() returns the actual compile order
	// after compilation, even when StdCfgLayout uses internal sort keys.

	t.Run("OrderList returns actual compile order after compilation", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		localWithBoth := fstest.MapFS{
			"example.json": &fstest.MapFile{
				Data: []byte(`{"Status": "from example.json"}`),
				Mode: 0755,
			},
			"conf.d/01.json": &fstest.MapFile{
				Data: []byte(`{"Status": "from conf.d"}`),
				Mode: 0755,
			},
		}

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: localWithBoth,
				home:  nil,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		// During compile, files are sorted with internal prefixes:
		// "0-example.json" comes before "1-01.json"
		// So actual compile order is: example.json, then 01.json
		exp := cfg.Explain()
		require.Len(exp.Records, 2)
		assert.Equal("example.json", exp.Records[0].Name)
		assert.Equal("01.json", exp.Records[1].Name)

		// After compilation, OrderList() returns the actual compile order
		ordered := cfg.OrderList([]string{"01.json", "example.json"})
		// Should match compile order, not alphabetical order
		assert.Equal([]string{"example.json", "01.json"}, ordered)
	})

	t.Run("OrderList before vs after compilation", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		// Use file names where natural sort differs from compile order
		localWithBoth := fstest.MapFS{
			"app.json": &fstest.MapFile{
				Data: []byte(`{"Status": "from app.json"}`),
				Mode: 0755,
			},
			"conf.d/zzz.json": &fstest.MapFile{
				Data: []byte(`{"Status": "from conf.d"}`),
				Mode: 0755,
			},
		}

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("app", nil, stdLocations{
				local: localWithBoth,
				home:  nil,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
		)
		require.NoError(err)
		// Note: NOT compiled yet

		// Before compilation, OrderList() sorts using the configured sorter (natural sort)
		orderedBefore := cfg.OrderList([]string{"zzz.json", "app.json"})
		// Natural sort: "app.json", "zzz.json" (alphabetical)
		assert.Equal([]string{"app.json", "zzz.json"}, orderedBefore)

		// After compilation, OrderList() uses actual compile order
		err = cfg.Compile()
		require.NoError(err)

		// Verify actual compile order from explanation
		exp := cfg.Explain()
		require.Len(exp.Records, 2)
		assert.Equal("app.json", exp.Records[0].Name) // appName.* (prefix "0-")
		assert.Equal("zzz.json", exp.Records[1].Name) // conf.d/* (prefix "1-")

		// OrderList now matches compile order
		orderedAfter := cfg.OrderList([]string{"zzz.json", "app.json"})
		assert.Equal([]string{"app.json", "zzz.json"}, orderedAfter)

		// In this case they're the same, but that's because natural sort
		// happens to agree with compile order for these filenames
	})

	t.Run("OrderList handles relative paths (issue #233)", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		localWithBoth := fstest.MapFS{
			"example.json": &fstest.MapFile{
				Data: []byte(`{"Status": "from example.json"}`),
				Mode: 0755,
			},
			"conf.d/01.json": &fstest.MapFile{
				Data: []byte(`{"Status": "from conf.d"}`),
				Mode: 0755,
			},
		}

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: localWithBoth,
				home:  nil,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		// Verify the explanation records use basenames
		exp := cfg.Explain()
		require.Len(exp.Records, 2)
		assert.Equal("example.json", exp.Records[0].Name)
		assert.Equal("01.json", exp.Records[1].Name)

		// OrderList should handle both basenames and relative paths
		// Callers may reasonably pass relative paths like "conf.d/01.json"
		// These should match against the basename stored in rec.Name
		ordered := cfg.OrderList([]string{"conf.d/01.json", "example.json"})
		assert.Equal([]string{"example.json", "conf.d/01.json"}, ordered)
	})
}

func TestStdCfgLayoutGlobErrorPropagation(t *testing.T) {
	// This test verifies that filesystem/glob errors are properly propagated
	// instead of being silently treated as "no files found".

	t.Run("glob error is propagated", func(t *testing.T) {
		require := require.New(t)

		// Use a pathological appName that creates an invalid glob pattern
		// The `[` character is invalid in glob patterns
		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("[invalid", nil, stdLocations{
				local: fstest.MapFS{},
				home:  nil,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
		)
		if err != nil {
			// If New() fails due to glob validation, that's also acceptable
			require.Contains(err.Error(), "glob pattern")
			return
		}

		// The error should occur during Compile when deferred filegroups are resolved
		err = cfg.Compile()
		require.Error(err)
		require.Contains(err.Error(), "glob pattern")
	})
}

func TestStdCfgLayoutOptionOrdering(t *testing.T) {
	// This test verifies that StdCfgLayout works correctly regardless of
	// where it appears in the option list relative to WithDecoder.
	// Previously, if StdCfgLayout came before WithDecoder, hasDecodableFiles()
	// would find no decoders and fail to select any location.

	localWithConfig := fstest.MapFS{
		"example.json": &fstest.MapFile{
			Data: []byte(`{"Status": "local config loaded"}`),
			Mode: 0755,
		},
	}

	type st struct {
		Status string
	}

	t.Run("StdCfgLayout before WithDecoder", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		// StdCfgLayout comes BEFORE WithDecoder - this used to fail
		cfg, err := New(
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: localWithConfig,
				home:  nil,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		var got st
		err = cfg.Unmarshal("", &got)
		require.NoError(err)

		assert.Equal("local config loaded", got.Status)
	})

	t.Run("StdCfgLayout after WithDecoder", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		// StdCfgLayout comes AFTER WithDecoder - this always worked
		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: localWithConfig,
				home:  nil,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		var got st
		err = cfg.Unmarshal("", &got)
		require.NoError(err)

		assert.Equal("local config loaded", got.Status)
	})
}

func TestStdCfgLayoutFallbackWithNonDecodableFiles(t *testing.T) {
	// This test verifies that locations with only non-decodable files
	// (e.g., .bak files) don't block fallback to lower-priority locations
	// with decodable config files.

	localWithBak := fstest.MapFS{
		"example.bak": &fstest.MapFile{
			Data: []byte(`{"Status": "this is a backup"}`),
			Mode: 0755,
		},
	}

	homeWithConfig := fstest.MapFS{
		"example.json": &fstest.MapFile{
			Data: []byte(`{"Status": "home config wanted"}`),
			Mode: 0755,
		},
	}

	type st struct {
		Status string
	}

	t.Run("non-decodable in local falls back to home", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: localWithBak,
				home:  homeWithConfig,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		var got st
		err = cfg.Unmarshal("", &got)
		require.NoError(err)

		assert.Equal("home config wanted", got.Status)
	})

	t.Run("non-decodable in conf.d falls back to next location", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		localWithBakInConfD := fstest.MapFS{
			"conf.d/example.bak": &fstest.MapFile{
				Data: []byte(`{"Status": "backup in conf.d"}`),
				Mode: 0755,
			},
		}

		etcWithConfig := fstest.MapFS{
			"example.json": &fstest.MapFile{
				Data: []byte(`{"Status": "etc config wanted"}`),
				Mode: 0755,
			},
		}

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: localWithBakInConfD,
				home:  nil,
				root:  fstest.MapFS{},
				etc:   etcWithConfig,
			}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		var got st
		err = cfg.Unmarshal("", &got)
		require.NoError(err)

		assert.Equal("etc config wanted", got.Status)
	})
}

func TestStdCfgLayoutBothPatternsCompose(t *testing.T) {
	// This test verifies that both appName.* files AND conf.d/* files
	// are loaded from the same location and compose together.

	localWithBoth := fstest.MapFS{
		"example.json": &fstest.MapFile{
			Data: []byte(`{"Status": "from example.json", "Count": 1}`),
			Mode: 0755,
		},
		"conf.d/01.json": &fstest.MapFile{
			Data: []byte(`{"Other": "from conf.d", "Count": 2}`),
			Mode: 0755,
		},
	}

	type st struct {
		Status string
		Other  string
		Count  int
	}

	t.Run("both appName.json and conf.d files compose from local", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: localWithBoth,
				home:  nil,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		var got st
		err = cfg.Unmarshal("", &got)
		require.NoError(err)

		// Both files should be loaded
		assert.Equal("from example.json", got.Status)
		assert.Equal("from conf.d", got.Other)
		// appName.json is processed first, then conf.d/01.json
		// So conf.d wins for overlapping keys
		assert.Equal(2, got.Count)
	})

	homeWithBoth := fstest.MapFS{
		"example.json": &fstest.MapFile{
			Data: []byte(`{"Status": "from home example", "Version": "1.0"}`),
			Mode: 0755,
		},
		"conf.d/override.json": &fstest.MapFile{
			Data: []byte(`{"Other": "from home conf.d", "Version": "2.0"}`),
			Mode: 0755,
		},
	}

	type stWithVersion struct {
		Status  string
		Other   string
		Version string
	}

	t.Run("both appName.json and conf.d files compose from home", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		cfg, err := New(
			WithDecoder(&testDecoder{extensions: []string{"json"}}),
			nonWinStdCfgLayout("example", nil, stdLocations{
				local: fstest.MapFS{},
				home:  homeWithBoth,
				root:  fstest.MapFS{},
				etc:   fstest.MapFS{},
			}),
		)
		require.NoError(err)
		err = cfg.Compile()
		require.NoError(err)

		var got stWithVersion
		err = cfg.Unmarshal("", &got)
		require.NoError(err)

		// Both files should be loaded and composed
		assert.Equal("from home example", got.Status)
		assert.Equal("from home conf.d", got.Other)
		// Files are processed: first AddFiles (example.json), then AddTree (conf.d/*)
		// So conf.d files are loaded after and win for Version
		assert.Equal("2.0", got.Version)
	})
}
