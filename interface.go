////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Elixxir                                                    /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package ekv

import (
	"github.com/pkg/errors"
	"os"
	"strings"
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
	Set(key string, objectToStore Marshaler) error
	Get(key string, loadIntoThisObject Unmarshaler) error
	Delete(key string) error
	SetInterface(key string, objectToSTore interface{}) error
	GetInterface(key string, v interface{}) error
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
