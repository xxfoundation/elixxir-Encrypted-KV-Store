////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"os"
	"strings"

	"github.com/pkg/errors"
)

// Marshaler interface defines objects which can "Marshal" themselves into a
// byte slice. This should produce a byte slice that can be used to fully
// reconstruct the object later.
type Marshaler interface {
	Marshal() []byte
}

// Unmarshaler interface defines objects which can be initialized by a byte
// slice. An error should be returned if the object cannot be decoded or,
// optionally, when Unmarshal is called against a pre-initialized object.
type Unmarshaler interface {
	Unmarshal([]byte) error
}

// KeyValue is the interface that ekv implements. Simple functions are provided
// for objects that can Marshal and Unmarshal themselves, and an interface
// version of these is provided which should use JSON or another generic object
// encoding system.
type KeyValue interface {
	// Set stores using an object that can marshal itself.
	Set(key string, objectToStore Marshaler) error
	// Get loads into an object that can unmarshal itself.
	Get(key string, loadIntoThisObject Unmarshaler) error
	// Delete destroys a key.
	Delete(key string) error
	// SetInterface uses a JSON encoder to store an interface object.
	SetInterface(key string, objectToSTore interface{}) error
	// GetInterface uses a JSON decode to load an interface object.
	GetInterface(key string, v interface{}) error
	// SetBytes stores raw bytes.
	SetBytes(key string, data []byte) error
	// GetBytes loads raw bytes.
	GetBytes(key string) ([]byte, error)
	// Transaction locks a set of keys while they are being mutated and
	// allows the function to operate on them exclusively.
	// More keys can be added to the transaction, but they must only be operated
	// on in conjunction with the previously locked keys otherwise deadlocks can
	// occur
	// If the op returns an error, the operation will be aborted.
	Transaction(op TransactionOperation, keys ...string) error
}

type TransactionOperation func(files map[string]Operable, ext Extender) error

// Operable describes edits to a single key inside a transaction
type Operable interface {
	// Key returns the key this interface is operating on
	Key() string
	// Exists returns if the file currently exists
	// will panic if the current transaction isn't in scope
	Exists() bool
	// Delete deletes the file at the key and destroy it.
	// will panic if the current transaction isn't in scope
	Delete()
	// Set stores raw bytes.
	// will panic if the current transaction isn't in scope
	Set(data []byte)
	// Get loads raw bytes.
	// will panic if the current transaction isn't in scope
	Get() ([]byte, bool)
	// Flush executes the operation and returns an error if the operation
	// failed. It will set the operable to closed as well.
	// if flush is not called, it will be called by the handler
	Flush() error
	// IsClosed returns true if the current transaction is in scope
	// will always be true if inside the execution of the transaction
	IsClosed() bool
}

type Extender interface {
	// Extend can be used to add more keys to the current transaction
	// if an error is returned, abort and return it
	Extend(keys []string) (map[string]Operable, error)
	// IsClosed returns true if the current transaction is in scope
	// will always be true if inside the execution of the transaction
	IsClosed() bool
}

// Exists determines if the error message is known to report the key does not
// exist. Returns true if the error does not specify or it is nil and false
// otherwise.
func Exists(err error) bool {
	if err == nil {
		return true
	}

	return !(errors.Is(err, os.ErrNotExist) ||
		strings.Contains(err.Error(), objectNotFoundErr))
}
