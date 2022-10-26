////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// This file is only compiled for WebAssembly.

package portableOS

import (
	"strings"
)

// Open opens the named file for reading. If successful, methods on the returned
// file can be used for reading.
var Open = func(name string) (File, error) {
	keyValue, err := jsDb.getItem(name)
	if err != nil {
		return nil, err
	}

	return open(name, string(keyValue), jsDb), nil
}

// Create creates or truncates the named file. If the file already exists, it is
// truncated. If the file does not exist, it is created. If successful, methods
// on the returned File can be used for I/O.
var Create = func(name string) (File, error) {
	jsDb.setItem(name, []byte(""))

	return open(name, "", jsDb), nil
}

// Remove removes the named file or directory.
var Remove = func(name string) error {
	jsDb.removeItem(name)
	return nil
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
var RemoveAll = func(path string) error {
	for i := 0; i < jsDb.length(); i++ {
		keyName, err := jsDb.key(i)
		if err != nil {
			return err
		}

		if strings.HasPrefix(keyName, path) {
			jsDb.removeItem(keyName)
		}
	}

	return nil
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error. The permission bits perm (before
// umask) are used for all directories that MkdirAll creates. If path is already
// a directory, MkdirAll does nothing and returns nil.
var MkdirAll = func(path string, perm FileMode) error {
	if jsDb == nil {
		// NOTE: This is kind of a hack, but MkDirAll is
		// always called first, so we initialize the DB here.
		InitDB()
	}
	jsDb.setItem(path, []byte(""))
	open(path, "", jsDb)
	return nil
}

// Stat returns a FileInfo describing the named file.
var Stat = func(name string) (FileInfo, error) {
	keyValue, err := jsDb.getItem(name)
	if err != nil {
		return nil, err
	}

	return &jsFileInfo{
		keyName: name,
		size:    int64(len(keyValue)),
	}, nil
}
