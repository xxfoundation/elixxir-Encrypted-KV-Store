////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// This file is only compiled for WebAssembly.

package portableOS

import (
	"strings"

	"gitlab.com/elixxir/wasm-utils/storage"
)

// Wrapper for Javascript externalStorage.
var externalStorage = storage.GetExternalStorage()

// Open opens the named file for reading. If successful, methods on the returned
// file can be used for reading.
var Open = func(name string) (File, error) {
	keyValue, err := externalStorage.Get(name)
	if err != nil {
		return nil, err
	}

	return open(name, string(keyValue), externalStorage), nil
}

// Create creates or truncates the named file. If the file already exists, it is
// truncated. If the file does not exist, it is created. If successful, methods
// on the returned File can be used for I/O.
var Create = func(name string) (File, error) {
	err := externalStorage.Set(name, []byte(""))
	if err != nil {
		return nil, err
	}

	return open(name, "", externalStorage), nil
}

// Remove removes the named file or directory.
var Remove = func(name string) error {
	err := externalStorage.Delete(name)
	if err != nil {
		return err
	}
	return nil
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
var RemoveAll = func(path string) error {
	keys, err := externalStorage.Keys()
	if err != nil {
		return err
	}
	for _, keyName := range keys {
		if strings.HasPrefix(keyName, path) {
			err := externalStorage.Delete(keyName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error. The permission bits perm (before
// umask) are used for all directories that MkdirAll creates. If path is already
// a directory, MkdirAll does nothing and returns nil.
var MkdirAll = func(path string, perm FileMode) error {
	err := externalStorage.Set(path, []byte(""))
	if err != nil {
		return err
	}
	open(path, "", externalStorage)
	return nil
}

// Stat returns a FileInfo describing the named file.
var Stat = func(name string) (FileInfo, error) {
	keyValue, err := externalStorage.Get(name)
	if err != nil {
		return nil, err
	}

	return &jsFileInfo{
		keyName: name,
		size:    int64(len(keyValue)),
	}, nil
}
