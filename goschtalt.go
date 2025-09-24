// SPDX-FileCopyrightText: 2022 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/goschtalt/goschtalt/pkg/decoder"
	"github.com/goschtalt/goschtalt/pkg/encoder"
	"github.com/goschtalt/goschtalt/pkg/meta"
)

// Root provides a more descriptive name to use for the root node of the
// configuration tree than a naked "".
const Root = ""

// DefaultOptions allows a simple place where decoders can automatically register
// themselves, as well as a simple way to find what is configured by default.
// Most extensions will register themselves using init().  It is safe to change
// this value at pretty much any time & compile afterwards; just know this value
// is not mutex protected so if you are changing it after init() the synchronization
// is up to the caller.
//
// To disable the use of this global variable, use the [DisableDefaultPackageOptions]
// option.
var DefaultOptions = []Option{}

// defaultTag is the go structure tag goschtalt will use for all it's work unless
// otherwise specified.
const defaultTag = "goschtalt"

// Config is a configurable, prioritized, merging configuration registry.
type Config struct {
	mutex      sync.Mutex
	records    []string
	tree       meta.Object
	compiledAt time.Time
	hash       []byte
	explain    Explanation

	rawOpts []Option
	opts    options
}

// New creates a new goschtalt configuration instance with any number of options.
func New(opts ...Option) (*Config, error) {
	c := Config{
		tree: meta.Object{},
		opts: options{
			decoders: newRegistry[decoder.Decoder](),
			encoders: newRegistry[encoder.Encoder](),
		},
	}

	if err := c.With(opts...); err != nil {
		return nil, err
	}

	return &c, nil
}

// With takes a list of options and applies them.  Use of With() is optional as
// New() can take all the same options as well.  If AutoCompile() is not specified
// Compile() will need to be called to see changes in the configuration based on
// the new options.
//
// See also: [AutoCompile], [Compile], [New]
func (c *Config) With(opts ...Option) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cfg := options{
		decoders: newRegistry[decoder.Decoder](),
		encoders: newRegistry[encoder.Encoder](),
	}

	c.explain.reset()

	raw := append(c.rawOpts, opts...)

	// These options must always be present to prevent panics, etc.
	full := []Option{
		SortRecordsNaturally(),
		SetKeyDelimiter("."),
		SetHasher(nil),
		SetMaxExpansions(10000),
	}

	if !ignoreDefaultOpts(raw) {
		local := []Option{
			DefaultUnmarshalOptions(KeymapReport(&c.explain.Keyremapping)),
			DefaultValueOptions(KeymapReport(&c.explain.Keyremapping)),
		}

		full = append(full, local...)
		full = append(full, DefaultOptions...)
	}

	full = append(full, c.rawOpts...)

	full = append(full, opts...)

	for _, opt := range full {
		if opt != nil {
			c.explain.optionInEffect(opt.String())
			if err := opt.apply(&cfg); err != nil {
				return err
			}
		}
	}

	for _, hint := range cfg.hints {
		if err := hint(&cfg); err != nil {
			return err
		}
	}

	// The options are valid, record them.
	c.opts = cfg
	c.rawOpts = raw

	c.explain.extsSupported(c.opts.decoders.extensions())

	if !c.opts.disableAutoCompile {
		return c.compile()
	}

	return nil
}

// Compile reads in all the files configured using the options provided,
// and merges the configuration trees into a single map for later use.
func (c *Config) Compile() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.compile()
}

// compile is the internal compile function that ensures the results are also
// recorded.
func (c *Config) compile() error {
	start := time.Now()
	c.explain.compileStartedAt(start)
	results, e := c.compileInternal(false)
	results.explain.CompileFinishedAt = time.Now()
	results.explain.recordError(e)
	c.explain = results.explain

	if e == nil {
		c.records = results.records
		c.tree = results.merged
		c.compiledAt = start
		c.hash = results.hash
	}

	return e
}

// compileResults is a helper struct that holds the results of the compilation
// process.  It contains the merged configuration tree, the records that were
// processed, the hash of the merged configuration, and an explanation of how
// the configuration was compiled.
//
// Originally this was all done in place in the Config object, but once the
// ability to compile only the defaults was added, it became necessary to
// separate the results of the compilation from the Config object itself.
type compileResults struct {
	explain Explanation
	records []string
	merged  meta.Object
	hash    []byte
}

// compileInternal is the internal compile function that does most of the work.
func (c *Config) compileInternal(defaultsOnly bool) (compileResults, error) {
	var rv compileResults

	rv.explain = c.explain
	full, defaultCount, err := c.getOrderedConfigs()
	if err != nil {
		return rv, err
	}

	rv.merged = meta.Object{Map: make(map[string]meta.Object)}
	rv.records = make([]string, 0, len(full))

	for i, cfg := range full {
		// Build an incremental snapshot of the configuration at this step so
		// user provided functions can use the cfg values to acquire more if
		// needed.
		incremental := rv.merged

		incremental, _, err = expandTree(incremental, c.opts.exapansionMax, c.opts.expansions)
		if err != nil {
			return rv, err
		}

		unmarshalFunc := func(key string, result any, opts ...UnmarshalOption) error {
			// Pass in the merged value from this context and stage of processing.
			return c.unmarshal(key, result, incremental, opts...)
		}

		if err = cfg.fetch(c.opts.keyDelimiter, unmarshalFunc, c.opts.decoders, c.opts.valueOptions); err != nil {
			return rv, err
		}
		rv.merged, err = rv.merged.Merge(cfg.tree)
		if err != nil {
			return rv, err
		}
		rv.records = append(rv.records, cfg.name)
		rv.explain.compileRecord(cfg.name, i < defaultCount, time.Now())

		if defaultsOnly && i < defaultCount {
			return rv, nil
		}
	}

	// Expand the final tree to ensure all values are expanded.
	rv.merged, _, err = expandTree(rv.merged, c.opts.exapansionMax, c.opts.expansions)
	if err != nil {
		return rv, err
	}

	// Record the expansions in effect.
	for _, exp := range c.opts.expansions {
		rv.explain.compileExpansions(exp.String())
	}

	rv.hash, err = c.opts.hasher.Hash(rv.merged)
	if err != nil {
		return rv, err
	}

	return rv, nil
}

// getOrderedConfigs is a helper function that combines the different groups of
// configuration files into a single, correctly ordered list and the number of
// default values that are at the start of the list.
func (c *Config) getOrderedConfigs() ([]record, int, error) {
	cfgs, err := filegroupsToRecords(c.opts.keyDelimiter, c.opts.filegroups, c.opts.decoders)
	if err != nil {
		return nil, 0, err
	}

	cfgs = append(cfgs, c.opts.values...)
	sorter := c.getSorter()
	sorter(cfgs)

	defaultCount := len(c.opts.defaults)
	full := append(c.opts.defaults, cfgs...)

	return full, defaultCount, nil
}

// getSorter does the work of making a sorter for the objects we need to sort.
func (c *Config) getSorter() func([]record) {
	return func(a []record) {
		sort.SliceStable(a, func(i, j int) bool {
			return c.opts.sorter.Less(a[i].name, a[j].name)
		})
	}
}

// OrderList is a helper function that sorts a caller provided list of filenames
// exactly the same way the Config object would sort them when reading and
// merging the records when the configuration is being compiled.  It also filters
// the list based on the decoders present.
func (c *Config) OrderList(list []string) []string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cfgs := make([]record, len(list))
	for i, item := range list {
		cfgs[i] = record{name: item}
	}

	sorter := c.getSorter()
	sorter(cfgs)

	var out []string
	for _, cfg := range cfgs {
		file := cfg.name

		// Only include the file if there is a decoder for it.
		ext := strings.TrimPrefix(path.Ext(file), ".")
		_, err := c.opts.decoders.find(ext)
		if err == nil {
			out = append(out, file)
		}
	}

	return out
}

// CompiledAt returns when the configuration was compiled.
func (c *Config) CompiledAt() time.Time {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.compiledAt
}

// Hash returns the hash of the configuration; even if the configuration is
// empty.  SetHasher() needs to be set to get a useful (non-empty) value.
func (c *Config) Hash() []byte {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.hash
}

// Explain returns a human focused explanation of how the configuration was
// arrived at.  Each time the options change or the configuration is compiled
// the explanation will be updated.
func (c *Config) Explain() Explanation {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.explain
}

// GetTree returns a copy of the compiled tree.  This is useful for debugging
// what the configuration tree looks like with a tool like k0kubun/pp.
//
// The value returned is a deep clone & has nothing to do with the original
// that still resides inside the Config object.
func (c *Config) GetTree() meta.Object {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.tree.Clone()
}
