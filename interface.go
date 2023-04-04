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
	// Set stores using an object that can marshal itself
	Set(key string, objectToStore Marshaler) error
	// Get loads into an object that can unmarshal itself
	Get(key string, loadIntoThisObject Unmarshaler) error
	// Delete destroys a key
	Delete(key string) error
	// SetInterface uses a JSON encoder to store an interface object
	SetInterface(key string, objectToSTore interface{}) error
	// GetInterface uses a JSON decord to load an interface object
	GetInterface(key string, v interface{}) error
	// SetBytes stores raw bytes
	SetBytes(key string, data []byte) error
	// GetBytes loads raw bytes
	GetBytes(key string) ([]byte, error)
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
