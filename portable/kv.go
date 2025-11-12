////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package portable

import (
	"bytes"
	"os"
	"strings"
	"sync"
)

// GenericKeyValue is a simple key-value storage interface that can be used
// to back the ekv Storage interface. This allows ekv to work with any
// key-value store including browser localStorage, IndexedDB, etc.
type GenericKeyValue interface {
	// Get retrieves the value for the given key.
	// Returns an error if the key does not exist.
	Get(key string) ([]byte, error)

	// Set stores the value for the given key.
	Set(key string, value []byte) error

	// Delete removes the key and its value.
	Delete(key string) error

	// Keys returns all keys in the store.
	Keys() ([]string, error)
}

// kv is a Storage implementation that wraps a GenericKeyValue interface.
type kv struct {
	storage GenericKeyValue
}

// UseKeyValue returns a Storage implementation that uses the provided
// GenericKeyValue interface as its backing store.
func UseKeyValue(storage GenericKeyValue) Storage {
	return &kv{storage: storage}
}

// Open opens the named file for reading. If successful, methods on the returned
// file can be used for reading.
func (k *kv) Open(name string) (File, error) {
	keyValue, err := k.storage.Get(name)
	if err != nil {
		// Convert to os.ErrNotExist if appropriate
		if strings.Contains(err.Error(), "not exist") ||
			strings.Contains(err.Error(), "not found") {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	return openKV(name, string(keyValue), k.storage), nil
}

// Create creates or truncates the named file. If the file already exists, it is
// truncated. If the file does not exist, it is created. If successful, methods
// on the returned File can be used for I/O.
func (k *kv) Create(name string) (File, error) {
	err := k.storage.Set(name, []byte(""))
	if err != nil {
		return nil, err
	}

	return openKV(name, "", k.storage), nil
}

// Remove removes the named file or directory.
func (k *kv) Remove(name string) error {
	err := k.storage.Delete(name)
	if err != nil {
		return err
	}
	return nil
}

// RemoveAll removes path and any children it contains.
// It removes everything it can but returns the first error
// it encounters. If the path does not exist, RemoveAll
// returns nil (no error).
func (k *kv) RemoveAll(path string) error {
	keys, err := k.storage.Keys()
	if err != nil {
		return err
	}
	for _, keyName := range keys {
		if strings.HasPrefix(keyName, path) {
			err := k.storage.Delete(keyName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error. For key-value stores, this is
// essentially a no-op that creates an empty key.
func (k *kv) MkdirAll(path string, perm FileMode) error {
	err := k.storage.Set(path, []byte(""))
	if err != nil {
		return err
	}
	openKV(path, "", k.storage)
	return nil
}

// Stat returns a FileInfo describing the named file.
func (k *kv) Stat(name string) (FileInfo, error) {
	keyValue, err := k.storage.Get(name)
	if err != nil {
		// Convert to os.ErrNotExist if appropriate
		if strings.Contains(err.Error(), "not exist") ||
			strings.Contains(err.Error(), "not found") {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	return &kvFileInfo{
		keyName: name,
		size:    int64(len(keyValue)),
	}, nil
}

// kvFile represents a File for a key-value pair in a GenericKeyValue store.
type kvFile struct {
	keyName string
	reader  *bytes.Reader
	storage GenericKeyValue
	dirty   bool // Is true when data on disk is different from in memory
	mux     sync.Mutex
}

// openKV creates a new in-memory file buffer of the key value.
func openKV(keyName, keyValue string, storage GenericKeyValue) *kvFile {
	f := &kvFile{
		keyName: keyName,
		reader:  bytes.NewReader([]byte(keyValue)),
		storage: storage,
		dirty:   false,
	}

	return f
}

// Close closes the File, rendering it unusable for I/O.
func (f *kvFile) Close() error {
	f.mux.Lock()
	defer f.mux.Unlock()

	f.reader.Reset(nil)
	return nil
}

// Name returns the name of the file as presented to Open.
func (f *kvFile) Name() string {
	return f.keyName
}

// Read reads up to len(b) bytes from the File and stores them in b.
// It returns the number of bytes read and any error encountered.
// At end of file, Read returns 0, io.EOF.
func (f *kvFile) Read(b []byte) (n int, err error) {
	f.mux.Lock()
	defer f.mux.Unlock()

	if f.dirty {
		keyValue, err := f.storage.Get(f.keyName)
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
func (f *kvFile) ReadAt(b []byte, off int64) (n int, err error) {
	f.mux.Lock()
	defer f.mux.Unlock()

	if f.dirty {
		keyValue, err := f.storage.Get(f.keyName)
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
// end. It returns the new offset and an error, if any.
func (f *kvFile) Seek(offset int64, whence int) (ret int64, err error) {
	f.mux.Lock()
	defer f.mux.Unlock()

	if f.dirty {
		keyValue, err := f.storage.Get(f.keyName)
		if err != nil {
			return 0, err
		}

		f.reader.Reset(keyValue)
		f.dirty = false
	}

	return f.reader.Seek(offset, whence)
}

// Sync commits the current contents of the file to stable storage.
func (f *kvFile) Sync() error {
	f.mux.Lock()
	defer f.mux.Unlock()

	keyValue, err := f.storage.Get(f.keyName)
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
func (f *kvFile) Write(b []byte) (n int, err error) {
	f.mux.Lock()
	defer f.mux.Unlock()

	f.dirty = true

	keyValue, err := f.storage.Get(f.keyName)
	if err != nil {
		return 0, err
	}

	keyValue = append(keyValue, b...)

	err = f.storage.Set(f.keyName, keyValue)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

// kvFileInfo represents a FileInfo for a key-value pair in a GenericKeyValue store.
type kvFileInfo struct {
	keyName string
	size    int64
}

// Name returns the base name of the file.
func (f *kvFileInfo) Name() string {
	return f.keyName
}

// Size returns the length in bytes.
func (f *kvFileInfo) Size() int64 {
	return f.size
}

// IsDir reports whether m describes a directory.
// For key-value stores, this always returns true to maintain compatibility.
func (f *kvFileInfo) IsDir() bool {
	return true
}
