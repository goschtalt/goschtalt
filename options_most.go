// SPDX-FileCopyrightText: 2023 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

//go:build !windows && !android

package goschtalt

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/goschtalt/goschtalt/pkg/decoder"
)

const confDirName = "conf.d"

func stdCfgLayout(appName string, files []string) Option {
	var l stdLocations
	l.Populate(appName)

	return nonWinStdCfgLayout(appName, files, l)
}

type stdLocations struct {
	local fs.FS
	home  fs.FS
	etc   fs.FS
	root  fs.FS
}

func (s *stdLocations) Populate(name string) {
	s.local = os.DirFS(".")
	s.root = os.DirFS("/")
	s.etc = os.DirFS("/" + filepath.Join("etc", name))

	if home := os.Getenv("HOME"); home != "" {
		s.home = os.DirFS(filepath.Join(home, "."+name))
	}
}

func nonWinStdCfgLayout(appName string, files []string, paths stdLocations) Option {
	if appName == "" {
		return WithError(fmt.Errorf("%w: StdCfgLayout appName cannot be empty", ErrInvalidInput))
	}
	if strings.Contains(appName, string(filepath.Separator)) {
		return WithError(fmt.Errorf("%w: StdCfgLayout appName cannot contain character '%s'", ErrInvalidInput, string(filepath.Separator)))
	}

	// Prune out any empty files.
	actualFiles := make([]string, 0, len(files))
	for _, file := range files {
		if file != "" {
			actualFiles = append(actualFiles, file)
		}
	}

	if len(actualFiles) > 0 {
		return AddJumbledHalt(paths.root, paths.local, actualFiles...)
	}

	// Defer location selection until compile time when decoders are available.
	// This ensures we only select locations that have decodable files.
	return &stdCfgLayoutOption{
		appName: appName,
		paths:   paths,
	}
}

const (
	// Sort prefixes for StdCfgLayout filegroups to enforce processing order
	stdCfgAppPrefix   = "0-" // appName.* files are processed first
	stdCfgConfDPrefix = "1-" // conf.d/* files are processed second, override appName.*
)

// stdCfgLayoutOption defers location selection until compile time when
// decoders are available. This ensures we only select locations with
// decodable files and properly fall back to lower-priority locations
// if higher-priority locations only contain non-decodable files.
type stdCfgLayoutOption struct {
	appName string
	paths   stdLocations
}

var _ Option = (*stdCfgLayoutOption)(nil)

func (s *stdCfgLayoutOption) apply(opts *options) error {
	// Defer filegroup selection until compile time when decoders are available.
	// This ensures StdCfgLayout works regardless of option order.
	// Store the current position so resolved filegroups maintain option order.
	insertPos := len(opts.filegroups)
	opts.deferredFilegroups = append(opts.deferredFilegroups, deferredFilegroup{
		insertAt: insertPos,
		fn: func() ([]filegroup, error) {
			// Try each location in priority order: local → home → etc
			locations := []struct {
				name string
				fsys fs.FS
			}{
				{"local", s.paths.local},
				{"home", s.paths.home},
				{"etc", s.paths.etc},
			}

			for _, loc := range locations {
				if loc.fsys == nil {
					continue
				}

				found, err := hasDecodableFiles(loc.fsys, s.appName, opts.decoders)
				if err != nil {
					return nil, err
				}
				if found {
					// Found decodable files in this location, use it.
					// Add two filegroups with prefixes to ensure correct sort order:
					// - "0-" prefix for appName.* files (processed first)
					// - "1-" prefix for conf.d/* files (processed second, overrides appName.*)

					// Add appName.* filegroup with "0-" prefix so it sorts before conf.d
					appFilesFG := filegroup{
						fs:         loc.fsys,
						paths:      []string{s.appName + ".*"},
						namePrefix: stdCfgAppPrefix,
						haltAlways: false, // Don't halt yet, we have conf.d to process
					}

					// Add conf.d/* filegroup with "1-" prefix so it sorts after appName.*
					confDFG := filegroup{
						fs:         loc.fsys,
						paths:      []string{confDirName},
						namePrefix: stdCfgConfDPrefix,
						recurse:    true,
						haltAlways: true, // Halt after conf.d
					}

					return []filegroup{appFilesFG, confDFG}, nil
				}
			}

			// No decodable files found in any location - this is not an error,
			// just means no configuration files are present
			return nil, nil
		},
	})

	return nil
}

func (s *stdCfgLayoutOption) ignoreDefaults() bool {
	return false
}

func (s *stdCfgLayoutOption) String() string {
	return "StdCfgLayout(" + s.appName + ")"
}

// hasDecodableFiles checks if a location has any files that can be decoded.
// Returns an error for unexpected filesystem issues (e.g., fs.ErrInvalid,
// fs.ErrClosed). Permission errors (fs.ErrPermission) and missing directories
// (fs.ErrNotExist) are treated as "no files found" and return (false, nil).
func hasDecodableFiles(fsys fs.FS, appName string, decoders *codecRegistry[decoder.Decoder]) (bool, error) {
	// Check for files matching appName.*
	files, err := fs.Glob(fsys, appName+".*")
	if err != nil {
		return false, fmt.Errorf("glob pattern '%s.*' failed: %w", appName, err)
	}

	for _, file := range files {
		if canDecode(file, decoders) {
			return true, nil
		}
	}

	// Check for files in conf.d directory
	// Use recurse: true to match the actual loading behavior in confDFG
	dirFG := filegroup{
		fs:      fsys,
		recurse: true,
	}
	files, err = dirFG.enumeratePath(confDirName)
	if err != nil {
		// enumeratePath uses normalizeDirError internally, which converts
		// fs.ErrPermission to nil but passes all other errors through unchanged.
		// We explicitly handle fs.ErrNotExist here since a missing conf.d is OK.
		if errors.Is(err, fs.ErrNotExist) {
			// Directory doesn't exist - not an error
			return false, nil
		}
		// Unexpected filesystem error (not permission, not missing)
		return false, fmt.Errorf("enumerating '%s': %w", confDirName, err)
	}

	for _, file := range files {
		if canDecode(file, decoders) {
			return true, nil
		}
	}

	return false, nil
}

// canDecode checks if a file can be decoded based on its extension
func canDecode(filename string, decoders *codecRegistry[decoder.Decoder]) bool {
	ext := filepath.Ext(filename)
	if ext == "" {
		return false
	}
	// Remove leading dot
	ext = ext[1:]

	_, err := decoders.find(ext)
	return err == nil
}
