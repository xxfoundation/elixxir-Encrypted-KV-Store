///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package portableOS

// This file is only compiled for WebAssembly.

import (
	"bytes"
	"sync"
)

// jsFile represents a File for a Javascript value saved in local storage.
type jsFile struct {
	keyName string
	reader  *bytes.Reader
	storage *jsStore
	dirty   bool // Is true when data on disk is different from in memory
	mux     sync.Mutex
}

// open creates a new in-memory file buffer of the key value.
func open(keyName, keyValue string, storage *jsStore) *jsFile {
	f := &jsFile{
		keyName: keyName,
		reader:  bytes.NewReader([]byte(keyValue)),
		storage: storage,
		dirty:   false,
	}

	return f
}

// Close closes the File, rendering it unusable for I/O.
// On files that support SetDeadline, any pending I/O operations will
// be canceled and return immediately with an ErrClosed error.
// Close will return an error if it has already been called.
func (f *jsFile) Close() error {
	f.mux.Lock()
	defer f.mux.Unlock()

	f.reader.Reset(nil)
	return nil
}

// Name returns the name of the file as presented to Open.
func (f *jsFile) Name() string {
	return f.keyName
}

// Read reads up to len(b) bytes from the File and stores them in b.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (f *jsFile) Read(b []byte) (n int, err error) {
	f.mux.Lock()
	defer f.mux.Unlock()

	if f.dirty {
		keyValue, err := f.storage.getItem(f.keyName)
		if err != nil {
			return 0, err
		}

		f.reader.Reset(keyValue)
		f.dirty = false
	}

	return f.reader.Read(b)
}

// ReadAt reads len(b) bytes from the File starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (f *jsFile) ReadAt(b []byte, off int64) (n int, err error) {
	f.mux.Lock()
	defer f.mux.Unlock()

	if f.dirty {
		keyValue, err := f.storage.getItem(f.keyName)
		if err != nil {
			return 0, err
		}

		f.reader.Reset(keyValue)
		f.dirty = false
	}

	return f.reader.ReadAt(b, off)
}

// Seek sets the offset for the next Read or Write on file to offset,
// interpreted according to whence: 0 means relative to the origin of the
// file, 1 means relative to the current offset, and 2 means relative to the
// end. It returns the new offset and an error, if any. The behavior of Seek
// on a file opened with os.O_APPEND is not specified.
//
// If f is a directory, the behavior of Seek varies by operating system; you
// can seek to the beginning of the directory on Unix-like operating
// systems, but not on Windows.
func (f *jsFile) Seek(offset int64, whence int) (ret int64, err error) {
	f.mux.Lock()
	defer f.mux.Unlock()

	if f.dirty {
		keyValue, err := f.storage.getItem(f.keyName)
		if err != nil {
			return 0, err
		}

		f.reader.Reset(keyValue)
		f.dirty = false
	}

	return f.reader.Seek(offset, whence)
}

// Sync commits the current contents of the file to stable storage.
// Typically, this means flushing the file system's in-memory copy
// of recently written data to disk.
func (f *jsFile) Sync() error {
	f.mux.Lock()
	defer f.mux.Unlock()

	keyValue, err := f.storage.getItem(f.keyName)
	if err != nil {
		return err
	}

	f.reader.Reset(keyValue)
	f.dirty = false

	return nil
}

// Write writes len(b) bytes from b to the File.
// It returns the number of bytes written and an error, if any.
// Write returns a non-nil error when n != len(b).
func (f *jsFile) Write(b []byte) (n int, err error) {
	f.mux.Lock()
	defer f.mux.Unlock()

	f.dirty = true

	keyValue, err := f.storage.getItem(f.keyName)
	if err != nil {
		return 0, err
	}

	keyValue = append(keyValue, b...)

	f.storage.setItem(f.keyName, keyValue)

	return len(b), nil
}

// jsFileInfo represents a FileInfo for a Javascript value saved in local
// storage.
type jsFileInfo struct {
	keyName string
	size    int64
}

// Name returns the base name of the file.
func (f *jsFileInfo) Name() string {
	return f.keyName
}

// Size returns the length in bytes.
func (f *jsFileInfo) Size() int64 {
	return f.size
}

// IsDir reports whether m describes a directory.
// That is, it tests for the ModeDir bit being set in m.
func (f *jsFileInfo) IsDir() bool {
	return true
}
