// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"errors"
	"testing"
	"time"

	"github.com/goschtalt/goschtalt/pkg/encoder"
	"github.com/goschtalt/goschtalt/pkg/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshal(t *testing.T) {
	var zeroOpt MarshalOption
	testErr := errors.New("test error")

	tests := []struct {
		description string
		input       string
		opts        []MarshalOption
		notCompiled bool
		noEncoders  bool
		expected    string
		expectedErr error
	}{
		{
			description: "Import and export a normal tree.",
			input:       `{"foo":"bar"}`,
			opts:        []MarshalOption{FormatAs("json")},
			expected:    `{"foo":"bar"}`,
		}, {
			description: "Import and export a normal tree, with an empty option.",
			input:       `{"foo":"bar"}`,
			opts:        []MarshalOption{FormatAs("json"), zeroOpt},
			expected:    `{"foo":"bar"}`,
		}, {
			description: "Import and export a tree with a secret.",
			input:       `{"foo((secret))":"bar"}`,
			opts:        []MarshalOption{FormatAs("json")},
			expected:    `{"foo":"bar"}`,
		}, {
			description: "Import and export a tree with a redacted secret.",
			input:       `{"foo((secret))":"bar"}`,
			opts:        []MarshalOption{FormatAs("json"), RedactSecrets(true)},
			expected:    `{"foo":"REDACTED"}`,
		}, {
			description: "Import and export a tree with orgins.",
			input:       `{"foo":"bar"}`,
			opts:        []MarshalOption{FormatAs("json"), IncludeOrigins(true)},
			expected:    `{"Origins":[{"File":"file","Line":1,"Col":123}],"Array":null,"Map":{"foo":{"Origins":[{"File":"file","Line":2,"Col":123}],"Array":null,"Map":null,"Value":"bar"}},"Value":null}`,
		}, {
			description: "Import and export an empty tree.",
			opts:        []MarshalOption{FormatAs("json"), IncludeOrigins(true)},
		}, {
			description: "Not compiled.",
			input:       `{"foo":"bar"}`,
			notCompiled: true,
			opts:        []MarshalOption{FormatAs("json")},
			expectedErr: ErrNotCompiled,
		}, {
			description: "No format exporter found.",
			input:       `{"foo":"bar"}`,
			opts:        []MarshalOption{FormatAs("unsupported")},
			expectedErr: ErrCodecNotFound,
		}, {
			description: "No format exporter found.",
			input:       `{"foo":"bar"}`,
			noEncoders:  true,
			opts:        []MarshalOption{FormatAs("json")},
			expectedErr: ErrCodecNotFound,
		}, {
			description: "Handle an error.",
			input:       `{"foo":"bar"}`,
			opts:        []MarshalOption{WithError(testErr)},
			expectedErr: testErr,
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			var tree meta.Object
			var err error
			if tc.input != "" {
				tree, err = decode("file", tc.input).ResolveCommands()
				require.NoError(err)
			}

			now := time.Time{}
			if !tc.notCompiled {
				now = time.Now()
			}

			c := Config{
				tree:       tree,
				compiledAt: now,
				opts: options{
					encoders:     newRegistry[encoder.Encoder](),
					keyDelimiter: ".",
				},
			}

			if !tc.noEncoders {
				c.opts.encoders.register(&testEncoder{extensions: []string{"json"}})
			}

			got, err := c.Marshal(tc.opts...)

			if tc.expectedErr == nil {
				assert.NoError(err)
				assert.Equal(tc.expected, string(got))
				return
			}

			assert.ErrorIs(err, tc.expectedErr)
		})
	}
}

type MapKeys struct {
	fn func(string) string
}

func (m MapKeys) Map(s string) string {
	return m.fn(s)
}

func TestMarshalOptionsMap(t *testing.T) {
	tests := []struct {
		description string
		input       string
		mapper      []Mapper
		expected    string
	}{
		{
			description: "Map a single key.",
			input:       "foo",
			mapper: []Mapper{
				&MapKeys{
					fn: func(s string) string {
						return s + "_mapped"
					},
				},
			},
			expected: "foo_mapped",
		}, {
			description: "Map multiple mappers.",
			input:       "foo",
			mapper: []Mapper{
				&MapKeys{
					fn: func(s string) string { return s + "_mapped" },
				},
				&MapKeys{
					fn: func(s string) string { return s + "_again" },
				},
			},
			expected: "foo_mapped_again",
		}, {
			description: "Map with an empty response.",
			input:       "foo",
			mapper: []Mapper{&MapKeys{
				fn: func(s string) string { return "" },
			}},
			expected: "foo",
		}, {
			description: "No mapping function provided.",
			input:       "foo",
			expected:    "foo",
		}, {
			description: "Map until the - is returned.",
			input:       "foo",
			mapper: []Mapper{
				&MapKeys{
					fn: func(s string) string { return s + "_mapped" },
				},
				&MapKeys{
					fn: func(s string) string { return "-" },
				},
				&MapKeys{
					fn: func(s string) string { return s + "_unreached" },
				},
			},
			expected: "foo_mapped",
		},
	}
	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			opt := marshalOptions{
				mappers: tc.mapper,
			}

			got := opt.Map(tc.input)

			assert.Equal(t, tc.expected, got)
		})
	}
}
