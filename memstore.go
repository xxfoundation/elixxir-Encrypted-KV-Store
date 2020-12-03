///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ekv

import (
	"encoding/json"
	"github.com/pkg/errors"
)

const (
	objectNotFoundErr = "object not found"
	setInterfaceErr   = "SetInterface error"
)

// Memstore is an unencrypted memory based map that implements the KV interface
type Memstore map[string][]byte

// Set stores the value if there's no serialization error
func (m Memstore) Set(key string, objectToStore Marshaler) error {
	ser := objectToStore.Marshal()
	m[key] = ser
	return nil
}

// Get returns the value
func (m Memstore) Get(key string, loadIntoThisObject Unmarshaler) error {
	data, ok := m[key]
	if !ok {
		return errors.New(objectNotFoundErr)
	}
	return loadIntoThisObject.Unmarshal(data)
}

// Get returns the value
func (m Memstore) Delete(key string) error {
	delete(m, key)
	return nil
}

// SetInterface sets the value using a json encoder
func (m Memstore) SetInterface(key string, objectToStore interface{}) error {
	data, err := json.Marshal(objectToStore)
	if err != nil {
		return errors.Wrap(err, setInterfaceErr)
	}
	m[key] = data
	return nil
}

// GetInterface gets the value using a json encoder
func (m Memstore) GetInterface(key string, objectToLoad interface{}) error {
	data, ok := m[key]
	if !ok {
		return errors.New(objectNotFoundErr)
	}
	err := json.Unmarshal(data, objectToLoad)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
