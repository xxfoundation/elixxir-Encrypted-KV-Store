///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// This file is only compiled for all architectures except WebAssembly.
//go:build !js || !wasm
// +build !js !wasm

package portableOS

import (
	"os"
)

// Open opens the named file for reading. If successful, methods on the returned
// file can be used for reading; the associated file descriptor has mode
// os.O_RDONLY.
var Open = func(name string) (File, error) {
	return os.Open(name)
}

// Create creates or truncates the named file. If the file already exists, it is
// truncated. If the file does not exist, it is created with mode 0666 (before
// umask). If successful, methods on the returned File can be used for I/O; the
// associated file descriptor has mode os.O_RDWR.
var Create = func(name string) (File, error) {
	return os.Create(name)
}

// Remove removes the named file or directory.
var Remove = os.Remove

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
// If there is an error, it will be of type *PathError.
var RemoveAll = os.RemoveAll

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error. The permission bits perm (before
// umask) are used for all directories that MkdirAll creates. If path is already
// a directory, MkdirAll does nothing and returns nil.
var MkdirAll = func(path string, perm FileMode) error {
	return os.MkdirAll(path, os.FileMode(5))
}

// Stat returns a FileInfo describing the named file.
var Stat = func(name string) (FileInfo, error) {
	return os.Stat(name)
}
