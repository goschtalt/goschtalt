// SPDX-FileCopyrightText: 2022-2024 Weston Schmidt <weston_schmidt@alumni.purdue.edu>
// SPDX-License-Identifier: Apache-2.0

package goschtalt

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/goschtalt/goschtalt/pkg/decoder"
	"github.com/goschtalt/goschtalt/pkg/meta"
)

// filegroup is a filesystem and paths to examine for configuration files.
type filegroup struct {
	// fs is the filesystem to examine.
	fs fs.FS

	// paths are either exact files, or directories to examine for configuration.
	paths []string

	// recurse specifies if directories encoutered in the paths should be examined
	// recursively or not.
	recurse bool

	// exactFile means that there should be exactly the same number of records as
	// files found or it is considered a failure.  This is mainly to support the
	// AddFile() use case where the file must be present or it is an error.
	exactFile bool

	// halt means processing should stop after this filegroup if any files were
	// found.
	halt bool

	// haltAlways means processing should always stop after this filegroup,
	// regardless of whether any files were found.
	haltAlways bool

	// namePrefix is prepended to record sortKeys from this filegroup to control
	// sort order. This is used internally to ensure certain filegroups are
	// processed before others (e.g., "0-" for appName.* and "1-" for conf.d/*).
	namePrefix string

	// as is the decoder to use for the files described by this filegroup.
	as string
}

// toRecords walks the filegroup and finds all the records that are present and
// can be processed using the present configuration.
func (g filegroup) toRecords(delimiter string, decoders *codecRegistry[decoder.Decoder]) ([]record, error) {
	files, err := g.enumerate()
	if err != nil {
		return nil, err
	}

	list := make([]record, 0, len(files))
	for _, file := range files {
		r, err := g.toRecord(file, delimiter, decoders)
		if err != nil {
			return nil, err
		}

		list = append(list, r...)
	}

	return list, nil
}

// toRecord handles examining a single file and returning it as part of an array
// of records.  This allows for returning 0 or 1 record easily.
func (g filegroup) toRecord(file, delimiter string, decoders *codecRegistry[decoder.Decoder]) ([]record, error) {
	f, err := g.fs.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	basename := stat.Name()
	ext := strings.TrimPrefix(path.Ext(basename), ".")

	// If the user specified a decoder to use, use it.
	if g.as != "" {
		ext = strings.TrimPrefix(g.as, ".")
	}

	dec, err := decoders.find(ext)
	if dec == nil {
		if g.exactFile {
			// No failures allowed.
			return nil, err
		}

		// The file isn't supported by a decoder, skip it.
		return nil, nil
	}

	// Only read the file after we're pretty sure it can be decoded.
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	ctx := decoder.Context{
		Filename:  basename,
		Delimiter: delimiter,
	}

	var tree meta.Object
	err = dec.Decode(ctx, data, &tree)
	if err != nil {
		err = fmt.Errorf("decoder error for extension '%s' processing file '%s' %w %v",
			ext, basename, ErrDecoding, err) //nolint:errorlint

		return nil, err
	}

	// Use basename as the record name (shown to users).
	// If namePrefix is set, use it only for the sortKey to control ordering.
	sortKey := basename
	if g.namePrefix != "" {
		sortKey = g.namePrefix + basename
	}

	return []record{{
		name:    basename,
		sortKey: sortKey,
		tree:    tree,
	}}, nil
}

// enumerate walks the specified paths and collects the files it finds that match
// the specified extensions.
func (g filegroup) enumerate() ([]string, error) {
	var files []string

	for _, glob := range g.paths {
		glob = path.Clean(glob)

		// If it isn't a glob or if the glob didn't find anything, then return
		// the original value as the array for uniform handling.
		paths := []string{glob}
		if !g.exactFile {
			tmp, err := fs.Glob(g.fs, glob)
			if err != nil {
				return nil, err
			}

			if len(tmp) > 0 {
				paths = tmp
			}
		}

		for _, p := range paths {
			found, err := g.enumeratePath(p)
			if err != nil {
				return nil, err
			}
			files = append(files, found...)
		}
	}

	sort.Strings(files)

	return files, nil
}

// enumeratePath examines a specific path and collects all the appropriate files.
// If the path ends up being a specific file return exactly that file.
func (g filegroup) enumeratePath(path string) ([]string, error) {
	isDir, err := g.isDir(path)
	if err != nil {
		return nil, normalizeDirError(err)
	}

	if !isDir {
		return []string{path}, nil
	}

	if g.exactFile {
		return nil, fs.ErrInvalid
	}

	fc := filecollector{
		path: path,
		fg:   g,
	}

	walker := fs.WalkDirFunc(fc.nonrecurse)
	if g.recurse {
		walker = fc.recurse
	}
	err = fs.WalkDir(g.fs, path, walker)

	return fc.files, err
}

// isDir examines a structure to see if it is a directory or something else.
func (g filegroup) isDir(path string) (dir bool, err error) {
	// Make sure the paths are consistent across FS implementations with
	// go's documentation.  This prevents errors due to some FS accepting
	// invalid paths while others correctly reject them.
	if !fs.ValidPath(path) {
		return false, fmt.Errorf("path '%s' %w", path, fs.ErrInvalid)
	}

	var file fs.File

	file, err = g.fs.Open(path)
	if err == nil {
		var stat fs.FileInfo
		stat, err = file.Stat()
		if err == nil {
			dir = stat.IsDir()
		}

		_ = file.Close()
	}

	return dir, err
}

// filegroupsToRecords converts a list of filegroups into a list of records.
func filegroupsToRecords(delimiter string, filegroups []filegroup, decoders *codecRegistry[decoder.Decoder]) ([]record, error) {
	rv := make([]record, 0, len(filegroups))
	for _, grp := range filegroups {
		tmp, err := grp.toRecords(delimiter, decoders)
		if err != nil {
			if grp.exactFile && errors.Is(err, fs.ErrNotExist) {
				return nil, ErrFileMissing
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}
		rv = append(rv, tmp...)

		// Stop processing if:
		// - haltAlways is set (regardless of whether files were found), or
		// - halt is set and we found files in this filegroup
		if grp.haltAlways || (len(tmp) > 0 && grp.halt) {
			break
		}
	}

	return rv, nil
}

// filecollector is a helper structure for collecting files from a directory.
type filecollector struct {
	path  string
	files []string
	fg    filegroup
}

// isReadable checks if a file is readable by trying to open it.
func (fc *filecollector) isReadable(file string) error {
	f, err := fc.fg.fs.Open(file)
	if err == nil {
		f.Close()
	}

	return err
}

// recurse is the function that is called for each file in a directory when
// recursion is enabled while walking the directory.
func (fc *filecollector) recurse(file string, d fs.DirEntry, err error) error {
	if err != nil || d.IsDir() {
		return normalizeFileError(err)
	}

	err = fc.isReadable(file)
	if err == nil {
		fc.files = append(fc.files, file)
	}

	return normalizeFileError(err)
}

// nonrecurse is the function that is called for each file in a directory when
// recursion is disabled while walking the directory.
func (fc *filecollector) nonrecurse(file string, d fs.DirEntry, err error) error {
	if err == nil && file != fc.path {
		if d.IsDir() {
			return fs.SkipDir
		}

		err = fc.isReadable(file)
		if err == nil {
			fc.files = append(fc.files, file)
		}
	}

	return normalizeFileError(err)
}

// normalizeFileError ignores some errors that are not fatal and returns others
// that are fatal.
func normalizeFileError(err error) error {
	// Ignore files we can't read.
	if errors.Is(err, fs.ErrPermission) {
		return nil
	}

	// Ignore files that might have disappeared between the time we found them
	// and the time we tried to read them.
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	// The other errors are fatal because they indicate an unknown problem.
	// Known other errors: (there could be others)
	// 	ErrInvalid
	// 	ErrExist
	// 	ErrClosed
	return err
}

// normalizeDirError ignores some errors that are not fatal and returns others
// that are fatal.
func normalizeDirError(err error) error {
	// Ignore directories we can't read.
	if errors.Is(err, fs.ErrPermission) {
		return nil
	}

	// The other errors are fatal because they indicate an unknown problem.
	// Known other errors: (there could be others)
	// 	ErrInvalid
	//  ErrNotExist
	// 	ErrExist
	// 	ErrClosed
	return err
}
