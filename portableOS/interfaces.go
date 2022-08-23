///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package portableOS contains global OS functions that can be overwritten to be
// used with other filesystems not supported by the [os] package, such as wasm.
//
// Note to those implementing these functions: all function errors of certain
// types must match the following errors
//  "permission denied"
//  "file already exists"
//  "file does not exist"
//  "file already closed"
package portableOS

// File represents an open file descriptor. It contains a subset of the methods
// on os.File that are used in this repository.
type File interface {
	// Close closes the File, rendering it unusable for I/O.
	// On files that support SetDeadline, any pending I/O operations will
	// be canceled and return immediately with an ErrClosed error.
	// Close will return an error if it has already been called.
	Close() error

	// Name returns the name of the file as presented to Open.
	Name() string

	// Read reads up to len(b) bytes from the File and stores them in b.
	// It returns the number of bytes read and any error encountered.
	// At end of file, Read returns 0, io.EOF.
	Read(b []byte) (n int, err error)

	// ReadAt reads len(b) bytes from the File starting at byte offset off.
	// It returns the number of bytes read and the error, if any.
	// ReadAt always returns a non-nil error when n < len(b).
	// At end of file, that error is io.EOF.
	ReadAt(b []byte, off int64) (n int, err error)

	// Seek sets the offset for the next Read or Write on file to offset,
	// interpreted according to whence: 0 means relative to the origin of the
	// file, 1 means relative to the current offset, and 2 means relative to the
	// end. It returns the new offset and an error, if any. The behavior of Seek
	// on a file opened with os.O_APPEND is not specified.
	//
	// If f is a directory, the behavior of Seek varies by operating system; you
	// can seek to the beginning of the directory on Unix-like operating
	// systems, but not on Windows.
	Seek(offset int64, whence int) (ret int64, err error)

	// Sync commits the current contents of the file to stable storage.
	// Typically, this means flushing the file system's in-memory copy
	// of recently written data to disk.
	Sync() error

	// Write writes len(b) bytes from b to the File.
	// It returns the number of bytes written and an error, if any.
	// Write returns a non-nil error when n != len(b).
	Write(b []byte) (n int, err error)
}

// A FileInfo describes a file and is returned by Stat. It contains a subset of
// the methods on os.FileInfo that are used in this repository.
type FileInfo interface {
	// Name returns the base name of the file.
	Name() string

	// Size returns the length in bytes for regular files; system-dependent for
	// others.
	Size() int64

	// IsDir reports whether m describes a directory.
	// That is, it tests for the ModeDir bit being set in m.
	IsDir() bool
}

// A FileMode represents a file's mode and permission bits. The bits have the
// same definition on all systems, so that information about files can be moved
// from one system to another portably. Not all bits apply to all systems. The
// only required bit is os.ModeDir for directories. See os.FileMode for all
// possible values.
type FileMode uint32
